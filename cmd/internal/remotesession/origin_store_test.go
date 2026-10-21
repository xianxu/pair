package remotesession

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const originDocA = "0123456789abcdef0123456789abcdef"
const originDocB = "fedcba9876543210fedcba9876543210"

func TestRemoteOriginSurvivesRestartAndRetiresOnlyExplicitIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "origins")
	s, err := OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRemote(originDocA); err != nil {
		t.Fatal(err)
	}
	s, err = OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if marked, err := s.Marked(originDocA); !marked || err != nil {
		t.Fatal("marker lost on restart", err)
	}
	if err := s.MarkRemote(originDocB); err != nil {
		t.Fatal(err)
	}
	if err := s.retire([]string{originDocB}); err != nil {
		t.Fatal(err)
	}
	if marked, err := s.Marked(originDocA); !marked || err != nil {
		t.Fatal("other ID retired")
	}
	if marked, err := s.Marked(originDocB); marked || err != nil {
		t.Fatal("retired ID retained")
	}
}
func TestRemoteOriginRetainsFutureFieldsThroughActualRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "origins")
	raw := []byte(`{"version":1,"future":{"precise":1e999},"documents":{"` + originDocA + `":{"remote_edited":true,"extension":{"nested":[-0,12345678901234567890123456789]}}}}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRemote(originDocB); err != nil {
		t.Fatal(err)
	}
	if err := s.retire([]string{originDocB}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]json.RawMessage
	if err := json.Unmarshal(raw, &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &after); err != nil {
		t.Fatal(err)
	}
	var beforeDocs, afterDocs map[string]json.RawMessage
	json.Unmarshal(before["documents"], &beforeDocs)
	json.Unmarshal(after["documents"], &afterDocs)
	var beforeMarker, afterMarker map[string]json.RawMessage
	json.Unmarshal(beforeDocs[originDocA], &beforeMarker)
	json.Unmarshal(afterDocs[originDocA], &afterMarker)
	if !bytes.Equal(before["future"], after["future"]) || !bytes.Equal(beforeMarker["extension"], afterMarker["extension"]) {
		t.Fatalf("future fields lost: %s", got)
	}
}
func TestRemoteOriginPublicationFailureBlocksFurtherMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "origins")
	s, err := OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s.publish = func(string, []byte) error { return errors.New("disk full") }
	if err := s.MarkRemote(originDocA); err == nil {
		t.Fatal("write failure hidden")
	}
	if _, err := s.Marked(originDocA); err == nil {
		t.Fatal("uncertain owner reported clean")
	}
	if err := s.MarkRemote(originDocB); err == nil {
		t.Fatal("uncertain owner continued")
	}
}
func TestRemoteOriginCodecStrict(t *testing.T) {
	for _, raw := range []string{`null`, `{"version":2,"documents":{}}`, `{"version":1,"documents":null}`, `{"version":1,"Documents":{},"documents":{}}`, `{"version":1,"documents":{"bad":{"remote_edited":true}}}`, `{"version":1,"documents":{"` + originDocA + `":{"remote_edited":false}}}`, `{"version":1,"documents":{"` + originDocA + `":{"remote_edited":true,"remote_edited":true}}}`} {
		if _, err := decodeOrigins([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func FuzzRemoteOriginCodec(f *testing.F) {
	for _, raw := range []string{`{"version":1,"documents":{}}`, `null`, `{"version":1,"documents":{},"future":{"n":1e999}}`} {
		f.Add([]byte(raw))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		record, err := decodeOrigins(raw)
		if err != nil {
			return
		}
		encoded, err := encodeOrigins(record)
		if err != nil {
			t.Fatal(err)
		}
		again, err := decodeOrigins(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if len(record.documents) != len(again.documents) {
			t.Fatal("projection changed")
		}
		for key, value := range record.root {
			if key != "version" && key != "documents" && !bytes.Equal(value, again.root[key]) {
				t.Fatal("opaque root changed")
			}
		}
		for id, fields := range record.documents {
			for key, value := range fields {
				if key != "remote_edited" && !bytes.Equal(value, again.documents[id][key]) {
					t.Fatal("opaque marker changed")
				}
			}
		}
	})
}
