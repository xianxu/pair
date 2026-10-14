package couchmessage

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

func TestDriveReceiptIntent(t *testing.T) {
	e := ReceiptEvent{Kind: ReceiptAdmit, ID: "drive", Op: "keys", Target: "pair:2", IntentDigest: "a"}
	r, err := ApplyReceiptEvent(OperationReceipt{}, e)
	if err != nil {
		t.Fatal(err)
	}
	e.IntentDigest = "b"
	if _, err = ApplyReceiptEvent(r, e); !errors.Is(err, ErrReceiptIDConflict) {
		t.Fatalf("changed intent: %v", err)
	}
}

func TestDriveReceiptBoundsAndOwnsResult(t *testing.T) {
	result := agentdrive.Result{Outcome: agentdrive.OutcomeConfirmed, Detail: strings.Repeat("detail", 10000), Tail: []string{strings.Repeat("tail", 10000), "last observed line"}}
	r, _ := ApplyReceiptEvent(OperationReceipt{}, ReceiptEvent{Kind: ReceiptAdmit, ID: "drive", Op: "keys"})
	r, err := ApplyReceiptEvent(r, ReceiptEvent{Kind: ReceiptFinish, ID: "drive", Outcome: ReceiptOutcome{Status: ReceiptSucceeded, Drive: &result}})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(r.Drive)
	if len(encoded) > agentdrive.MaxResultTailBytes {
		t.Fatalf("retained %d bytes", len(encoded))
	}
	result.Tail[len(result.Tail)-1] = "changed"
	if r.Drive.Tail[len(r.Drive.Tail)-1] != "last observed line" {
		t.Fatal("receipt aliases endpoint result")
	}
}
