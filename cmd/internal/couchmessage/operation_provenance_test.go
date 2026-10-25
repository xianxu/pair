package couchmessage

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// pair#847: provenance is additive. Only a succeeded outcome records it, a nil
// value is omitted on the wire, and a receipt from before it decodes unchanged.
func TestReceiptProvenanceIsAdditive(t *testing.T) {
	proof := &couchcore.LaunchProvenance{Root: "/w/pair-slot2/pair", Tag: "t", Generation: "n1", Agent: "claude", Argv: []string{"--x"}, ArgvSource: "explicit"}
	for _, c := range []struct {
		status ReceiptStatus
		kept   bool
	}{{ReceiptSucceeded, true}, {ReceiptFailed, false}, {ReceiptRefused, false}} {
		r, err := ApplyReceiptEvent(OperationReceipt{}, ReceiptEvent{Kind: ReceiptAdmit, ID: "b", Op: "bench-start", Target: "pair:2", At: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		if r, err = ApplyReceiptEvent(r, ReceiptEvent{Kind: ReceiptStart, ID: "b", At: time.Now()}); err != nil {
			t.Fatal(err)
		}
		r, err = ApplyReceiptEvent(r, ReceiptEvent{Kind: ReceiptFinish, ID: "b", At: time.Now(), Outcome: ReceiptOutcome{Status: c.status, Provenance: proof}})
		if err != nil {
			t.Fatal(err)
		}
		if (r.Provenance != nil) != c.kept {
			t.Fatalf("%s: provenance %+v", c.status, r.Provenance)
		}
	}
	raw, err := json.Marshal(OperationReceipt{ID: "old", Op: "reboot", Status: ReceiptSucceeded})
	if err != nil || strings.Contains(string(raw), "Provenance") {
		t.Fatalf("nil provenance on the wire: %s %v", raw, err)
	}
	var decoded OperationReceipt
	if err := json.Unmarshal([]byte(`{"ID":"old","Op":"reboot","Target":"pair:1","Status":"succeeded","Tag":"t"}`), &decoded); err != nil || decoded.Provenance != nil || decoded.Tag != "t" {
		t.Fatalf("older receipt: %+v %v", decoded, err)
	}
}
