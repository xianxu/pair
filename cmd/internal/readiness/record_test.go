package readiness

import "testing"

func TestReadyRecordRoundTrip(t *testing.T) {
	raw, err := Encode(ReadyRecord{
		Tag:     "work",
		Agent:   "codex",
		Session: "pair-work",
		Nonce:   "nonce-1",
		PID:     123,
	})
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	got, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if got.Tag != "work" || got.Agent != "codex" || got.Session != "pair-work" || got.Nonce != "nonce-1" || got.PID != 123 {
		t.Fatalf("Decode = %+v, want original record", got)
	}
}

func TestReadyRecordRejectsIncompleteOrMalformedInput(t *testing.T) {
	for _, raw := range []string{
		`{"tag":"","agent":"codex","session":"pair-work","nonce":"n","pid":1}`,
		`{"tag":"work","agent":"","session":"pair-work","nonce":"n","pid":1}`,
		`{"tag":"work","agent":"codex","session":"","nonce":"n","pid":1}`,
		`{"tag":"work","agent":"codex","session":"pair-work","nonce":"","pid":1}`,
		`{"tag":"work","agent":"codex","session":"pair-work","nonce":"n","pid":0}`,
		`{"tag":`,
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := Decode(raw); err == nil {
				t.Fatalf("Decode(%q) returned nil error", raw)
			}
		})
	}
}
