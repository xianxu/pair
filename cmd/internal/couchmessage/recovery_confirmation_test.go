package couchmessage

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"testing"
)

func TestRecoveryConfirmationClosedRequest(t *testing.T) {
	r := Request{Op: "recovery-confirmation", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", ID: "request", RecoveryConfirmation: "opaque"}
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Request){func(r *Request) { r.Target = "pair:1" }, func(r *Request) { r.CandidateKey = "choice" }, func(r *Request) { r.Confirmed = true }, func(r *Request) { r.Op = "resume" }, func(r *Request) { r.RecoveryConfirmation = "" }} {
		bad := r
		mutate(&bad)
		if ValidateRequest(bad) == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}
func TestRecoveryConfirmationReceiptRetainsOfferAndTokenIdentity(t *testing.T) {
	event := ReceiptEvent{Kind: ReceiptAdmit, ID: "request", Op: "recovery-confirmation", Target: "pair:2", RecoveryConfirmation: "one"}
	receipt, err := ApplyReceiptEvent(OperationReceipt{}, event)
	if err != nil {
		t.Fatal(err)
	}
	event.RecoveryConfirmation = "two"
	if _, err = ApplyReceiptEvent(receipt, event); !errors.Is(err, ErrReceiptIDConflict) {
		t.Fatalf("replay changed token: %v", err)
	}
	offer := &couchcore.RecoveryConfirmationOffer{Token: "token", Command: "couch --recovery-confirmation token", Stops: []couchcore.RecoveryStopProcess{{PID: 123, Identity: "generation", Role: "helper"}}}
	receipt, err = ApplyReceiptEvent(receipt, ReceiptEvent{Kind: ReceiptFinish, ID: "request", Outcome: ReceiptOutcome{Status: ReceiptRefused, Confirmation: offer}})
	if err != nil || receipt.Confirmation == nil || receipt.Confirmation.Stops[0].PID != 123 {
		t.Fatalf("offer lost: %+v %v", receipt, err)
	}
}
