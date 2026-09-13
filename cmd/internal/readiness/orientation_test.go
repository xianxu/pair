package readiness

import (
	"github.com/xianxu/pair/cmd/internal/orientation"
	"testing"
)

func TestOrientationStatusValidation(t *testing.T) {
	r := ReadyRecord{Tag: "t", Agent: "codex", Session: "s", Nonce: "n", PID: 1, Orientation: &orientation.DeliveryState{Phase: orientation.DeliverySubmitted, BodyWritten: true}}
	raw, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(raw)
	if err != nil || got.Orientation == nil || got.Orientation.Phase != orientation.DeliverySubmitted {
		t.Fatalf("decode %#v %v", got, err)
	}
	for _, state := range []orientation.DeliveryState{{Phase: "bogus"}, {Phase: orientation.DeliverySubmitted}, {Phase: orientation.DeliveryFailed, BodyWritten: true}, {Phase: orientation.DeliveryIndeterminate}, {Phase: orientation.DeliveryCancelled, Reason: "bad\x1b"}} {
		r.Orientation = &state
		if _, err := Encode(r); err == nil {
			t.Fatalf("accepted %#v", state)
		}
	}
	if _, err := Decode(`{"tag":"t","agent":"codex","session":"s","nonce":"n","pid":1,"orientation":{"phase":"submitted","body_written":true,"unknown":true}}`); err == nil {
		t.Fatal("accepted unknown status field")
	}
}
