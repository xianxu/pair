package readiness

import (
	"strings"
	"testing"
)

// A ready record from a wrapper before #522 may carry its orientation delivery
// state; it decodes (the slot is still running) and the state is ignored.
func TestALegacyOrientationStateStillDecodes(t *testing.T) {
	got, err := Decode(`{"tag":"t","agent":"codex","session":"s","nonce":"n","pid":1,"orientation":{"phase":"submitted","body_written":true}}`)
	if err != nil || got.Nonce != "n" || got.PID != 1 {
		t.Fatalf("decode %#v %v", got, err)
	}
	raw, err := Encode(ReadyRecord{Tag: "t", Agent: "codex", Session: "s", Nonce: "n", PID: 1})
	if err != nil || len(raw) == 0 || strings.Contains(raw, "orientation") {
		t.Fatalf("encode %q %v", raw, err)
	}
}
