package zellijpane

import (
	"encoding/json"
	"testing"
)

func FuzzFullscreenObservation(f *testing.F) {
	for _, value := range []string{"true", "false", " true \n", "\tfalse ", "null", `"false"`, "0", "{}", "[true]", ""} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		// Decode the value independently and marshal the envelope so fuzz input
		// cannot inject sibling fields/panes and change which observation we test.
		var decoded any
		if json.Unmarshal([]byte(value), &decoded) != nil {
			return
		}
		raw, err := json.Marshal([]map[string]any{{"id": 1, "is_focused": true, "is_fullscreen": decoded}})
		if err != nil {
			t.Fatal(err)
		}
		panes := Parse(raw)
		if len(panes) != 1 {
			t.Fatalf("valid envelope lost pane: %s: %+v", raw, panes)
		}
		want, isBool := decoded.(bool)
		got := panes[0].IsFullscreen
		if (got != nil) != isBool {
			t.Fatalf("value %q: fullscreen=%v, want non-nil iff boolean (%v)", value, got, isBool)
		}
		if isBool && *got != want {
			t.Fatalf("value %q: got %v, want %v", value, *got, want)
		}
	})
}

func TestFullscreenUnknownIsNotFalse(t *testing.T) {
	for _, raw := range []string{`[{"id":1,"is_focused":true}]`, `[{"id":1,"is_focused":true,"is_fullscreen":"false"}]`, `[{"id":1,"is_focused":true,"is_fullscreen":null}]`} {
		panes := Parse([]byte(raw))
		if len(panes) != 1 || panes[0].IsFullscreen != nil {
			t.Fatalf("unknown observation: %+v", panes)
		}
	}
}
