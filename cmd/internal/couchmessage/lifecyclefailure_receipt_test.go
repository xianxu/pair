package couchmessage

import (
	"encoding/json"
	"testing"
)

func TestReceiptFinishRetainsFailureNonceJSON(t *testing.T) {
	for _, status := range []ReceiptStatus{ReceiptRefused, ReceiptFailed, ReceiptSucceeded} {
		t.Run(string(status), func(t *testing.T) {
			var outcome ReceiptOutcome
			if err := json.Unmarshal([]byte(`{"Status":"`+string(status)+`","FailureNonce":"operation-exact"}`), &outcome); err != nil {
				t.Fatal(err)
			}
			r, err := ApplyReceiptEvent(OperationReceipt{}, ReceiptEvent{Kind: ReceiptAdmit, ID: "r", Op: "resume", Target: "pair:5"})
			if err != nil {
				t.Fatal(err)
			}
			r, err = ApplyReceiptEvent(r, ReceiptEvent{Kind: ReceiptFinish, ID: "r", Outcome: outcome})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &wire); err != nil {
				t.Fatal(err)
			}
			if string(wire["FailureNonce"]) != `"operation-exact"` {
				t.Fatalf("lost nonce: %s", encoded)
			}
		})
	}
}
func TestReceiptOmitsAbsentFailureNonceJSON(t *testing.T) {
	raw, err := json.Marshal(OperationReceipt{Status: ReceiptSucceeded})
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, present := wire["FailureNonce"]; present {
		t.Fatal("empty nonce emitted")
	}
}
