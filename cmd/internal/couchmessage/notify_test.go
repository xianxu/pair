package couchmessage

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNotifyRequestShape(t *testing.T) {
	base := map[string]any{"Op": "notify", "Scope": "scope", "Tag": "tag", "Session": "session", "Nonce": "nonce", "Title": "Couch", "Body": "ready"}
	check := func(m map[string]any) error {
		raw, _ := json.Marshal(m)
		var r Request
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		return ValidateRequest(r)
	}
	if err := check(base); err != nil {
		t.Fatalf("valid notify: %v", err)
	}
	for key, value := range map[string]any{"ID": "id", "Target": "hub", "Agent": "codex", "Binding": map[string]string{}, "Confirmed": true, "SameBinary": true, "ForceUnknown": true, "TailScope": "s", "TailTag": "t", "Lines": 1, "Title": strings.Repeat("x", 257), "Body": strings.Repeat("x", 4097), "Scope": "", "Tag": "", "Session": "", "Nonce": ""} {
		t.Run(key, func(t *testing.T) {
			m := map[string]any{}
			for k, v := range base {
				m[k] = v
			}
			m[key] = value
			if err := check(m); err == nil {
				t.Fatal("accepted invalid notify")
			}
		})
	}
}

func TestNotifyTitleRejectedOnOtherOperations(t *testing.T) {
	for _, op := range []string{"register", "operator-submit", "tail", "replay-share", "broadcast-status", "actors", "send", "status", "operation-status", "switch-agent", "submission-prepare"} {
		raw, _ := json.Marshal(map[string]any{"Op": op, "Scope": "s", "Tag": "t", "Session": "n", "Nonce": "n", "Title": "hidden"})
		var r Request
		_ = json.Unmarshal(raw, &r)
		if err := ValidateRequest(r); err == nil {
			t.Errorf("title accepted on %s", op)
		}
	}
}
