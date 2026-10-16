package couchmessage

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Literal pre-#573 request schema: do not derive Message from current code.
type legacyEndpointMessage struct {
	ID       string
	From, To Binding
	Body     string
	Deadline time.Time
}

type legacyEndpointRequest struct {
	Op       string
	Binding  Binding
	ID       string
	Sequence uint64
	Message  *legacyEndpointMessage
	Lines    int
}

func TestBrokerKeepsAcceptanceWhileLegacyWrapperReceivesNoTimestamp(t *testing.T) {
	from, to := brokerBinding("ops:0"), brokerBinding("parley.nvim:4")
	var commits atomic.Int32
	socket := transportSocket(t)
	server, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var old legacyEndpointRequest
		if err := strictjson.Decode(raw, &old); err != nil {
			return json.Marshal(EndpointResponse{Error: "old wrapper cannot decode: " + err.Error()})
		}
		switch old.Op {
		case "observe":
			return json.Marshal(EndpointResponse{Observation: Observation{LastActivity: time.Now().Add(-time.Minute)}})
		case "commit":
			commits.Add(1)
			m := old.Message
			return json.Marshal(EndpointResponse{Receipt: &Receipt{Message: Message{ID: m.ID, From: m.From, To: m.To, Body: m.Body, Deadline: m.Deadline}, Status: Submitted}})
		default:
			return json.Marshal(EndpointResponse{})
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	b := NewBroker(context.Background(), time.Now, nil)
	defer b.Close()
	if err := b.Register(from, newFakeEndpoint(time.Now().Add(-time.Minute))); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(to, RemoteEndpoint{Binding: to, socket: socket}); err != nil {
		t.Fatal(err)
	}
	accepted, err := b.Send(context.Background(), from, "legacy-wire", Route{Target: to.Slot}, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Message.AcceptedAt.IsZero() {
		t.Fatal("broker did not record acceptance")
	}
	testwait.Until(t, "legacy delivery terminal", func() bool {
		r, err := b.Status(from, "legacy-wire")
		return err == nil && r.Status.Terminal()
	})
	r, err := b.Status(from, "legacy-wire")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != Submitted {
		t.Fatalf("legacy wrapper rejected the wire message: %+v", r)
	}
	if !r.Message.AcceptedAt.Equal(accepted.Message.AcceptedAt) {
		t.Fatal("broker lost its original acceptance time")
	}
	if commits.Load() != 1 {
		t.Fatalf("commits = %d", commits.Load())
	}
}
