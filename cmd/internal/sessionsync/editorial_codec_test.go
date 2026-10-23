package sessionsync

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const editorialRoundJSON = `{"base_hash":"a","result_hash":"b","at":"2026-10-23T00:00:00Z","presentation":"highlight","spans":[{"start":[0,0],"end":[0,1],"old":"a","new":"b","presentation":"direct","future":{ "huge": 1e999 }}],"future":{ "nested": [1, 2] }}`

func TestEditorialRoundCodecPreservesExtensions(t *testing.T) {
	r, err := decodeRound([]byte(editorialRoundJSON))
	if err != nil {
		t.Fatal(err)
	}
	raw, ok := encodeRound(*r)
	if !ok {
		t.Fatal("encode refused")
	}
	for _, exact := range []string{`{ "huge": 1e999 }`, `{ "nested": [1, 2] }`, `"presentation":"direct"`} {
		if !bytes.Contains(raw, []byte(exact)) {
			t.Fatalf("lost raw extension %s: %s", exact, raw)
		}
	}
}

func TestEditorialPresentationFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/editorial-rounds/presentation.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Before, Result string
		Round                json.RawMessage
		Records              []roundRecord
		Expected             []string `json:"expected_effective"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			r, err := decodeRound(c.Round)
			if err != nil {
				t.Fatal(err)
			}
			if r.BaseHash != Hash(c.Before) || r.ResultHash != Hash(c.Result) {
				t.Fatal("fixture hash mismatch")
			}
			starts := lineStarts(c.Result)
			for i, s := range r.Spans {
				from, to := starts[s.Start[0]]+s.Start[1], starts[s.End[0]]+s.End[1]
				if c.Result[from:to] != s.New {
					t.Fatalf("span %d coordinate mismatch", i)
				}
				if effective := spanPresentation(s, r.Presentation); effective != c.Expected[i] {
					t.Fatalf("span %d: %s", i, effective)
				}
			}
			body, err := RoundBody(*r, c.Result)
			if err != nil {
				t.Fatal(err)
			}
			block := strings.Split(strings.Split(body, "```review-records\n")[1], "\n```")[0]
			var records []roundRecord
			if err := json.Unmarshal([]byte(block), &records); err != nil {
				t.Fatal(err)
			}
			for i, rec := range records {
				if rec.Presentation != c.Expected[i] {
					t.Errorf("resume lost effective mode at %d: %s", i, rec.Presentation)
				}
			}
		})
	}
}

func TestEditorialRoundCodecRejectsAmbiguity(t *testing.T) {
	mutations := []string{
		strings.Replace(editorialRoundJSON, `"base_hash":"a"`, `"base_hash":null`, 1),
		strings.Replace(editorialRoundJSON, `"start":[0,0]`, `"start":[0,null]`, 1),
		strings.Replace(editorialRoundJSON, `"end":[0,1]`, `"end":[0,1,2]`, 1),
		strings.Replace(editorialRoundJSON, `"end":[0,1]`, `"end":[1,-1]`, 1),
		strings.Replace(editorialRoundJSON, `"old":"a",`, ``, 1),
		strings.Replace(editorialRoundJSON, `"presentation":"direct"`, `"presentation":null`, 1),
		strings.Replace(editorialRoundJSON, `"presentation":"direct"`, `"presentation":"mixed"`, 1),
		strings.Replace(editorialRoundJSON, `"huge": 1e999`, `"x":1,"x":2`, 1),
		strings.Replace(editorialRoundJSON, `"spans":[`, `"spans":null,"unused":[`, 1),
	}
	for i, raw := range mutations {
		if _, err := decodeRound([]byte(raw)); err == nil {
			t.Errorf("mutation %d accepted: %s", i, raw)
		}
	}
}

func TestEditorialRoundRequiredMembersAndLegacySpan(t *testing.T) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(editorialRoundJSON), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"base_hash", "result_hash", "at", "spans"} {
		copy := map[string]json.RawMessage{}
		for k, v := range raw {
			if k != key {
				copy[k] = v
			}
		}
		b, _ := json.Marshal(copy)
		if _, err := decodeRound(b); err == nil {
			t.Errorf("missing %s accepted", key)
		}
	}
	legacy := strings.Replace(editorialRoundJSON, `,"presentation":"direct"`, ``, 1)
	if _, err := decodeRound([]byte(legacy)); err != nil {
		t.Fatalf("legacy span: %v", err)
	}
	legacy = strings.Replace(legacy, `"presentation":"highlight",`, ``, 1)
	if r, err := decodeRound([]byte(legacy)); err != nil || r.Presentation != "highlight" {
		t.Fatalf("legacy round fallback: %+v %v", r, err)
	}
}

func TestEditorialArtifactAndRecordPreserveExtensions(t *testing.T) {
	raw := []byte(`{"file":"/note.md","body":"commit","round":` + editorialRoundJSON + `,"future":{ "n": 1e999 }}`)
	var a RoundArtifact
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	b, err := a.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, exact := range []string{`{ "n": 1e999 }`, `{ "huge": 1e999 }`, `{ "nested": [1, 2] }`} {
		if !bytes.Contains(b, []byte(exact)) {
			t.Errorf("artifact lost %s: %s", exact, b)
		}
	}
	body, err := RoundBody(a.Round, "b")
	if err != nil {
		t.Fatal(err)
	}
	for _, exact := range []string{`"presentation":"direct"`, `{ "huge": 1e999 }`} {
		if !strings.Contains(body, exact) {
			t.Errorf("record lost %s: %s", exact, body)
		}
	}
	for _, input := range []string{`{"old":null,"new":"a"}`, `{"new":"a"}`, `{"old":"a","new":"b","presentation":"invalid"}`, `{"old":"a","new":"b","future":{"x":1,"x":2}}`} {
		var r roundRecord
		if err := json.Unmarshal([]byte(input), &r); err == nil {
			t.Errorf("record accepted %s", input)
		}
	}
}

func FuzzEditorialRoundCodec(f *testing.F) {
	f.Add(editorialRoundJSON)
	f.Add(strings.Replace(editorialRoundJSON, `"huge": 1e999`, `"same":1,"same":2`, 1))
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 65536 {
			t.Skip()
		}
		r, err := decodeRound([]byte(source))
		if err != nil {
			return
		}
		raw, ok := encodeRound(*r)
		if !ok {
			t.Fatal("accepted bounded round cannot encode")
		}
		after, err := decodeRound(raw)
		if err != nil {
			t.Fatal(err)
		}
		check := func(before, next map[string]json.RawMessage) {
			if len(before) != len(next) {
				t.Fatal("extension cardinality changed")
			}
			for key, value := range before {
				if !bytes.Equal(value, next[key]) {
					t.Fatalf("extension %s changed", key)
				}
			}
		}
		check(r.unknown, after.unknown)
		if len(r.Spans) != len(after.Spans) {
			t.Fatal("span count changed")
		}
		for i, span := range r.Spans {
			check(span.unknown, after.Spans[i].unknown)
		}
	})
}
