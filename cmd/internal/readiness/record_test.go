package readiness

import (
	"strings"
	"testing"
)

func TestReadyRecordRoundTrip(t *testing.T) {
	raw, err := Encode(ReadyRecord{
		Tag:          "work",
		Agent:        "codex",
		Session:      "pair-work",
		Nonce:        "nonce-1",
		PID:          123,
		AgentVersion: "0.164.0",
	})
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	got, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if got.Tag != "work" || got.Agent != "codex" || got.Session != "pair-work" || got.Nonce != "nonce-1" || got.PID != 123 || got.AgentVersion != "0.164.0" {
		t.Fatalf("Decode = %+v, want original record", got)
	}
}

func TestReadyRecordBoundsAgentVersion(t *testing.T) {
	base := ReadyRecord{Tag: "t", Agent: "codex", Session: "s", Nonce: "n", PID: 1}
	for _, bad := range []string{strings.Repeat("9", 65), "0.164\x00.0", "0.164\n.0"} {
		record := base
		record.AgentVersion = bad
		if _, err := Encode(record); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	ok := base
	ok.AgentVersion = "0.164.0"
	if _, err := Encode(ok); err != nil {
		t.Fatal(err)
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

// A Couch-launched wrapper publishes the agent's tty (pair#695 M2); a reader
// from before the field, and every reader for whatever a later build adds,
// must accept members it does not own instead of losing the agent's
// readiness. Owned members stay strict.
func TestReadyRecordToleratesUnknownMembersAndCarriesTTY(t *testing.T) {
	got, err := Decode(`{"tag":"work","agent":"codex","session":"pair-work","nonce":"n","pid":7,"tty":"ttys183","future":{"x":1}}`)
	if err != nil {
		t.Fatalf("Decode with an unknown member: %v", err)
	}
	if got.TTY != "ttys183" || got.PID != 7 {
		t.Fatalf("Decode = %+v", got)
	}
	raw, err := Encode(ReadyRecord{Tag: "work", Agent: "codex", Session: "pair-work", Nonce: "n", PID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, `"tty"`) {
		t.Fatalf("a record without a tty wrote one: %s", raw)
	}
	for name, bad := range map[string]string{
		"duplicate owned": `{"tag":"work","tag":"x","agent":"codex","session":"s","nonce":"n","pid":7}`,
		"wrong type":      `{"tag":"work","agent":"codex","session":"s","nonce":"n","pid":"7"}`,
		"tty with a path": `{"tag":"work","agent":"codex","session":"s","nonce":"n","pid":7,"tty":"../ttys1"}`,
		"tty control":     `{"tag":"work","agent":"codex","session":"s","nonce":"n","pid":7,"tty":"tty\u0007"}`,
	} {
		if _, err := Decode(bad); err == nil {
			t.Errorf("%s: accepted %s", name, bad)
		}
	}
}
