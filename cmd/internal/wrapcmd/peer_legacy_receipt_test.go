package wrapcmd

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// Keep the pre-AcceptedAt message shape literal: using today's Message here
// would let a newly added field silently change the compatibility assertion.
type legacyPeerMessage struct {
	ID       string
	From, To couchmessage.Binding
	Body     string
	Deadline time.Time
}

type legacyPeerReceipt struct {
	Message     legacyPeerMessage
	Status      couchmessage.Status
	Detail      string
	RetainUntil time.Time
}

type legacyPeerEndpointResponse struct {
	Error       string
	Observation couchmessage.Observation
	Receipt     *legacyPeerReceipt
	Tail        *couchmessage.Tail `json:",omitempty"`
}

func TestPeerLegacyReceiptOmitsUnsetAcceptedAt(t *testing.T) {
	for _, op := range []string{"commit", "status"} {
		t.Run(op, func(t *testing.T) {
			raw, message := peerReceiptWire(t, op, time.Time{})
			var legacy legacyPeerEndpointResponse
			if err := strictjson.Decode(raw, &legacy); err != nil {
				t.Fatalf("old supervisor cannot decode %s reply: %v; wire: %s", op, err, raw)
			}
			if legacy.Error != "" || legacy.Receipt == nil || legacy.Receipt.Message.ID != message.ID || legacy.Receipt.Message.Body != message.Body || legacy.Receipt.Status != couchmessage.Queued {
				t.Fatalf("old supervisor lost the queued receipt: %+v", legacy)
			}
		})
	}
}

func TestPeerReceiptPreservesNonzeroAcceptedAt(t *testing.T) {
	accepted := time.Date(2026, 10, 15, 20, 0, 0, 123456789, time.UTC)
	for _, op := range []string{"commit", "status"} {
		t.Run(op, func(t *testing.T) {
			raw, message := peerReceiptWire(t, op, accepted)
			var current couchmessage.EndpointResponse
			if err := strictjson.Decode(raw, &current); err != nil {
				t.Fatal(err)
			}
			if current.Error != "" || current.Receipt == nil || current.Receipt.Message.ID != message.ID || !current.Receipt.Message.AcceptedAt.Equal(accepted) {
				t.Fatalf("wrapper changed admission time: %+v; wire: %s", current, raw)
			}
		})
	}
}

func peerReceiptWire(t *testing.T, op string, accepted time.Time) ([]byte, couchmessage.Message) {
	t.Helper()
	now := time.Date(2026, 10, 15, 20, 1, 0, 0, time.UTC)
	to := peerTestBinding()
	to.Repository, to.Scope, to.Tag = "/fixture/.git", "fixture-scope", "fixture-tag"
	to.Session, to.Nonce = "fixture-session", "fixture-nonce"
	to.Agent, to.Version, to.PID, to.Start = "claude", "fixture-version", 1234, "fixture-start"
	from := to
	from.Slot, from.Tag, from.PID = "pair:0", "fixture-hub", 1235
	message := couchmessage.Message{ID: "legacy-receipt", From: from, To: to, Body: "hello from the older supervisor", Deadline: now.Add(time.Minute), AcceptedAt: accepted}
	d := newPeerDelivery(to, func() time.Time { return now })
	call := func(request couchmessage.EndpointRequest) []byte {
		t.Helper()
		request.Binding = to
		raw, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		reply, err := d.handleEndpoint(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		return reply
	}
	var reserved couchmessage.EndpointResponse
	if err := strictjson.Decode(call(couchmessage.EndpointRequest{Op: "reserve", ID: message.ID}), &reserved); err != nil || reserved.Error != "" {
		t.Fatalf("reserve: response=%+v error=%v", reserved, err)
	}
	committed := call(couchmessage.EndpointRequest{Op: "commit", ID: message.ID, Message: &message})
	if op == "commit" {
		return committed, message
	}
	return call(couchmessage.EndpointRequest{Op: "status", ID: message.ID}), message
}
