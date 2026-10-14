package board

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOperatorCaptureTokenAndStatus(t *testing.T) {
	for _, op := range []string{"save-note", "draft-status", "cancel-draft"} {
		body := map[string]any{"draft": "u." + strings.Repeat("a", 20000) + ".abc"}
		if op == "save-note" {
			body["color"] = 0
			body["text"] = "hello"
		}
		raw, _ := json.Marshal(map[string]any{"op": op, "request": "req", "generation": "1", "body": body})
		if _, err := ParseOperatorRequest(raw); err != nil {
			t.Fatalf("%s: %v", op, err)
		}
	}
}
