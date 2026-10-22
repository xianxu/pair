package remotesession

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const originDocA = "0123456789abcdef0123456789abcdef"
const originDocB = "fedcba9876543210fedcba9876543210"

func TestRemoteOriginForgetIntentCodec(t *testing.T) {
	for _, fields := range []string{`{"forget_pending":true}`, `{"remote_edited":true,"forget_pending":true}`} {
		raw := []byte(`{"version":1,"documents":{"` + originDocA + `":` + fields + `}}`)
		if _, err := decodeOrigins(raw); err != nil {
			t.Fatalf("valid forget intent refused: %v", err)
		}
	}
	for _, fields := range []string{`{}`, `{"forget_pending":false}`, `{"forget_pending":null}`, `{"forget_pending":"true"}`, `{"forget_pending":true,"remote_edited":false}`, `{"forget_pending":true,"Forget_Pending":true}`, `{"forget_pending":true,"forget_pending":true}`} {
		raw := []byte(`{"version":1,"documents":{"` + originDocA + `":` + fields + `}}`)
		if _, err := decodeOrigins(raw); err == nil {
			t.Fatalf("invalid forget intent accepted: %s", fields)
		}
	}
}

func TestRemoteOriginForgetLifecycleSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "origins")
	s, err := OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRemote(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.RetireForgotten(originDocA); err == nil {
		t.Fatal("retired without persisted forget intent")
	}
	for _, id := range []string{originDocB, originDocA} {
		if err := s.BeginForget(id); err != nil {
			t.Fatal(err)
		}
		if err := s.BeginForget(id); err != nil {
			t.Fatal("repeat intent", err)
		}
		if err := s.MarkRemote(id); err == nil {
			t.Fatal("edited pending forget", id)
		}
	}
	s, err = OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := s.PendingForgets()
	if err != nil || !reflect.DeepEqual(ids, []string{originDocA, originDocB}) {
		t.Fatalf("pending %v: %v", ids, err)
	}
	if marked, err := s.Marked(originDocA); !marked || err != nil {
		t.Fatal("marker lost", err)
	}
	if marked, err := s.Marked(originDocB); marked || err != nil {
		t.Fatal("intent became remote edit", err)
	}
	if err := s.RetireForgotten(originDocB); err != nil {
		t.Fatal(err)
	}
	if marked, err := s.Marked(originDocA); !marked || err != nil {
		t.Fatal("other ID retired", err)
	}
	s, err = OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ids, err = s.PendingForgets()
	if err != nil || !reflect.DeepEqual(ids, []string{originDocA}) {
		t.Fatalf("pending after retirement %v: %v", ids, err)
	}
}

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
	if err := s.BeginForget(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginForget(originDocB); err != nil {
		t.Fatal(err)
	}
	if err := s.RetireForgotten(originDocB); err != nil {
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
	if err := s.BeginForget(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRemote(originDocB); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginForget(originDocB); err != nil {
		t.Fatal(err)
	}
	if err := s.RetireForgotten(originDocB); err != nil {
		t.Fatal(err)
	}
	s, err = OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if ids, err := s.PendingForgets(); err != nil || !reflect.DeepEqual(ids, []string{originDocA}) {
		t.Fatalf("retained intent: %v %v", ids, err)
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

func TestRemoteOriginForgetPublicationUncertainty(t *testing.T) {
	for _, retire := range []bool{false, true} {
		t.Run(fmt.Sprint("retire=", retire), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "origins")
			s, err := OpenOriginStore(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.MarkRemote(originDocA); err != nil {
				t.Fatal(err)
			}
			if retire {
				if err := s.BeginForget(originDocA); err != nil {
					t.Fatal(err)
				}
			}
			publish := s.publish
			s.publish = func(path string, raw []byte) error {
				if err := publish(path, raw); err != nil {
					return err
				}
				return errors.New("publication outcome unavailable")
			}
			mutate := s.BeginForget
			if retire {
				mutate = s.RetireForgotten
			}
			if err := mutate(originDocA); err == nil {
				t.Fatal("uncertainty hidden")
			}
			if _, err := s.Marked(originDocA); err == nil {
				t.Fatal("uncertain owner served evidence")
			}
			if _, err := s.PendingForgets(); err == nil {
				t.Fatal("uncertain owner served recovery")
			}
			if err := s.BeginForget(originDocB); err == nil {
				t.Fatal("uncertain owner continued forget")
			}
			if err := s.RetireForgotten(originDocA); err == nil {
				t.Fatal("uncertain owner continued retirement")
			}
			s, err = OpenOriginStore(path)
			if err != nil {
				t.Fatal(err)
			}
			ids, err := s.PendingForgets()
			if err != nil {
				t.Fatal(err)
			}
			if retire && len(ids) != 0 || !retire && !reflect.DeepEqual(ids, []string{originDocA}) {
				t.Fatalf("recovered intents: %v", ids)
			}
		})
	}
}

func TestRemoteOriginForgetBounds(t *testing.T) {
	t.Run("record capacity", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "origins")
		docs := make(map[string]map[string]json.RawMessage)
		for i := 0; i < maxOriginMarkers; i++ {
			docs[fmt.Sprintf("%032x", i)] = map[string]json.RawMessage{"remote_edited": json.RawMessage("true")}
		}
		raw, err := encodeOrigins(originRecord{documents: docs})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		s, err := OpenOriginStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.BeginForget(originDocA); err == nil {
			t.Fatal("intent exceeded record capacity")
		}
		if err := s.BeginForget(fmt.Sprintf("%032x", 0)); err != nil {
			t.Fatal("could not forget existing at capacity", err)
		}
	})
	t.Run("byte capacity", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "origins")
		prefix := `{"version":1,"documents":{"` + originDocA + `":{"remote_edited":true}},"future":"`
		raw := []byte(prefix + strings.Repeat("x", maxOriginsBytes-len(prefix)-2) + `"}`)
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		s, err := OpenOriginStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.BeginForget(originDocA); err == nil {
			t.Fatal("intent exceeded byte capacity")
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, raw) {
			t.Fatal("rejected update rewrote evidence", err)
		}
		if marked, err := s.Marked(originDocA); !marked || err != nil {
			t.Fatal("prepublication refusal poisoned evidence", err)
		}
	})
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

func TestRemoteOriginInspectDistinguishesPendingIntent(t *testing.T) {
	s, err := OpenOriginStore(filepath.Join(t.TempDir(), "origins"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRemote(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginForget(originDocB); err != nil {
		t.Fatal(err)
	}
	marked, pending, err := s.Inspect(originDocA)
	if err != nil || !marked || pending {
		t.Fatalf("marked: %v %v %v", marked, pending, err)
	}
	marked, pending, err = s.Inspect(originDocB)
	if err != nil || marked || !pending {
		t.Fatalf("pending: %v %v %v", marked, pending, err)
	}
	if _, _, err := s.Inspect("bad"); err == nil {
		t.Fatal("invalid ID inspected")
	}
}

func TestRemoteOriginSubmitSeenSurvivesRestartAndPreservesMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "origins")
	s, err := OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSubmitted(originDocA); err != nil {
		t.Fatal(err)
	}
	if marked, err := s.Marked(originDocA); err != nil || marked {
		t.Fatal("submit became edit", err)
	}
	if err := s.MarkRemote(originDocA); err != nil {
		t.Fatal(err)
	}
	s, err = OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if seen, err := s.Submitted(originDocA); err != nil || !seen {
		t.Fatal("submit evidence lost", err)
	}
	if marked, err := s.Marked(originDocA); err != nil || !marked {
		t.Fatal("edit evidence lost", err)
	}
	if err := s.BeginForget(originDocB); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSubmitted(originDocB); err == nil {
		t.Fatal("submit admitted during forget")
	}
}

func TestRemoteOriginSubmitCodec(t *testing.T) {
	for _, fields := range []string{`{"submit_seen":true}`, `{"submit_seen":true,"remote_edited":true}`} {
		if _, err := decodeOrigins([]byte(`{"version":1,"documents":{"` + originDocA + `":` + fields + `}}`)); err != nil {
			t.Fatal(err)
		}
	}
	for _, fields := range []string{`{"submit_seen":false}`, `{"submit_seen":null}`, `{"submit_seen":"true"}`, `{"submit_seen":true,"Submit_Seen":true}`, `{"submit_seen":true,"submit_seen":true}`} {
		if _, err := decodeOrigins([]byte(`{"version":1,"documents":{"` + originDocA + `":` + fields + `}}`)); err == nil {
			t.Fatal("invalid submit evidence accepted", fields)
		}
	}
}

func TestRemoteOriginSubmitSeenCannotRetire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "origins")
	raw := []byte(`{"version":1,"documents":{"` + originDocA + `":{"remote_edited":true,"submit_seen":true,"forget_pending":true}}}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RetireForgotten(originDocA); err == nil {
		t.Fatal("submit evidence retired")
	}
}

func TestRemoteOriginSubmitPreservesUnknownFieldsThroughRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "origins")
	raw := []byte(`{"version":1,"future":{"precise":1e999},"documents":{"` + originDocA + `":{"remote_edited":true,"opaque":{"number":-0}}}}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSubmitted(originDocA); err != nil {
		t.Fatal(err)
	}
	s, err = OpenOriginStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if seen, err := s.Submitted(originDocA); err != nil || !seen {
		t.Fatal("submit evidence lost", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(after, []byte(`"precise":1e999`)) || !bytes.Contains(after, []byte(`"number":-0`)) {
		t.Fatalf("opaque fields changed: %s", after)
	}
}
