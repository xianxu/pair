package readiness

import (
	"strings"
	"testing"
)

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

func TestReadyRecordCarriesExactLaunchOrdinal(t *testing.T) {
	const raw = `{"tag":"work","agent":"codex","session":"pair-work","nonce":"attempt","pid":123,"launch_ordinal":17}`
	record, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if record.LaunchOrdinal != 17 {
		t.Fatalf("ordinal = %d", record.LaunchOrdinal)
	}
	encoded, err := Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Decode(encoded)
	if err != nil || again.LaunchOrdinal != 17 {
		t.Fatalf("round trip = %+v %v", again, err)
	}
}

func TestReadyAgentIdentityPreservesLegacyAndRejectsInvalid(t *testing.T) {
	legacy := ReadyRecord{Tag: "work", Agent: "codex", Session: "session", Nonce: "nonce", PID: 1}
	raw, e := Encode(legacy)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := Decode(raw)
	if e != nil || decoded.AgentIdentity != "" {
		t.Fatal("legacy gained identity", e)
	}
	legacy.AgentIdentity = "123456.789"
	raw, e = Encode(legacy)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e = Decode(raw)
	if e != nil || decoded.AgentIdentity != legacy.AgentIdentity {
		t.Fatal("identity roundtrip failed", e)
	}
	for _, bad := range []string{"bad\nidentity", strings.Repeat("x", 257), string([]byte{0xff})} {
		legacy.AgentIdentity = bad
		if _, e = Encode(legacy); e == nil {
			t.Fatal("accepted malformed identity")
		}
	}
}
