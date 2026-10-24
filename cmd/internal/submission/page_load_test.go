package submission

import (
	"bytes"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"testing"
)

func pageLoadSourceObjects(root map[string]any) []map[string]any {
	sk := root["sketches"].([]any)[0].(map[string]any)
	return []map[string]any{sk["source"].(map[string]any), sk["finish"].(map[string]any), root["sketch_page"].(map[string]any)["source"].(map[string]any)}
}
func TestSubmissionPageLoadFactsSurviveTypedPreservingRoundTrip(t *testing.T) {
	root := sharedPageWire(t)
	for _, source := range pageLoadSourceObjects(root) {
		source["page_load"] = map[string]any{"target": source["tab"], "loader": "L", "root": "9223372036854775807", "future": json.RawMessage(`{"n":1e10000}`)}
	}
	s, err := decodePageWire(t, root)
	if err != nil {
		t.Fatal(err)
	}
	want := query.PageLoadFacts{Target: s.Sketches[0].Source.Tab, Loader: "L", Root: "9223372036854775807"}
	if s.Sketches[0].Source.PageLoad != want || s.Sketches[0].Finish.PageLoad != want || s.SketchPage.Source.PageLoad != want {
		t.Fatalf("typed provenance lost: %+v %+v %+v", s.Sketches[0].Source.PageLoad, s.Sketches[0].Finish.PageLoad, s.SketchPage.Source.PageLoad)
	}
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	wire, err := EncodePreserving(s)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(wire, []byte(`"page_load"`)) != 3 || bytes.Count(wire, []byte(`1e10000`)) != 3 {
		t.Fatalf("lost nested raw provenance: %s", wire)
	}
	again, err := DecodePreserving(wire)
	if err != nil || again.SketchPage.Source.PageLoad != want {
		t.Fatalf("rewrite lost typed scope: %+v %v", again, err)
	}
}
func TestSubmissionPageLoadRefusesMalformedOwnedIdentity(t *testing.T) {
	for location := 0; location < 3; location++ {
		for _, bad := range []any{nil, 7, map[string]any{}, map[string]any{"target": "tab", "loader": "L", "root": 1}, map[string]any{"target": "tab", "loader": "L", "root": "01"}, map[string]any{"target": "tab", "loader": "L", "root": "9223372036854775808"}} {
			root := sharedPageWire(t)
			pageLoadSourceObjects(root)[location]["page_load"] = bad
			if s, err := decodePageWire(t, root); err == nil && Validate(s) == nil {
				t.Fatalf("accepted malformed load at %d: %+v", location, bad)
			}
		}
	}
}

func TestSubmissionPageLoadProgrammaticIdentityValidation(t *testing.T) {
	for _, facts := range []query.PageLoadFacts{{Target: "wrong", Loader: "L", Root: "1"}, {Target: "tab", Loader: "L", Root: "+1"}, {Target: "tab", Root: "1"}, {Root: "1"}} {
		s := sketchSubmission(t)
		s.Sketches[0].Source.PageLoad = facts
		if err := Validate(s); err == nil {
			t.Fatalf("accepted programmatic partial/foreign load: %+v", facts)
		}
	}
}
