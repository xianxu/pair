package wrapcmd

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// commitAndFinish puts m through the wrapper's real reserve/commit path and
// records a terminal outcome, as dispatchPeer would after submitting.
func commitAndFinish(t *testing.T, d *peerDelivery, m couchmessage.Message, status couchmessage.Status) {
	t.Helper()
	d.mu.Lock()
	seq := d.sequence
	d.mu.Unlock()
	if err := d.reserve(m.ID, seq); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
	d.mu.Lock()
	d.current.Status = status
	d.reservation = ""
	d.mu.Unlock()
}

// The wrapper remembers past deliveries after current moves on, refuses to
// take a reused ID again, and still answers its status (#365).
func TestPeerDeliveryRetainsReceiptsAndRefusesReusedIDs(t *testing.T) {
	first := peerTestMessage(time.Now())
	d := newPeerDelivery(first.To, time.Now)
	commitAndFinish(t, d, first, couchmessage.Submitted)
	second := first
	second.ID = "second-id"
	commitAndFinish(t, d, second, couchmessage.Submitted)

	if r, ok := d.retained(first.ID); !ok || r.Status != couchmessage.Submitted {
		t.Fatalf("forgot the earlier delivery: %+v %v", r, ok)
	}
	d.mu.Lock()
	seq := d.sequence
	d.mu.Unlock()
	var committed *couchmessage.AlreadyCommittedError
	if err := d.reserve(first.ID, seq); !errors.As(err, &committed) || committed.Receipt.Message.ID != first.ID {
		t.Fatalf("reused ID reservable: %v", err)
	}
}

// A receipt lost with a broker restart is recovered from the wrapper over the
// real endpoint socket, and a sender re-using the ID gets that outcome rather
// than a second paste.
func TestPeerLostReceiptRecoveredAfterBrokerRestart(t *testing.T) {
	namespace, err := os.MkdirTemp("/tmp", "pair-peer-recovery-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(namespace) })
	binding := func(slot, tag string) couchmessage.Binding {
		return couchmessage.Binding{Slot: slot, Repository: "/fixture/.git", Scope: "fixture-scope", Tag: tag, Session: "fixture-" + tag, Nonce: "launch-" + tag, Agent: "claude", Version: "test-fixture", PID: os.Getpid(), Start: "fixture-start"}
	}
	from, to := binding("brain:0", "sender"), binding("pair:1", "receiver")
	m := couchmessage.Message{ID: "lost-receipt", From: from, To: to, Body: "hello", Deadline: time.Now().Add(couchmessage.DeliveryTimeout)}
	d := newPeerDelivery(to, time.Now)
	commitAndFinish(t, d, m, couchmessage.Submitted)
	socket, err := couchmessage.EndpointSocket(namespace, to)
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(context.Background(), socket, d.handleEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })

	// A broker that never saw m (a Couch restart). The re-send comes first, so
	// nothing is cached: it must travel reserve -> wire already-committed ->
	// adoption, not a broker short-circuit (#365 BR-12).
	newBroker := func() *couchmessage.Broker {
		b := couchmessage.NewBroker(context.Background(), time.Now, func(context.Context, couchmessage.Binding) (bool, error) { return true, nil })
		t.Cleanup(func() { _ = b.Close() })
		for _, v := range []couchmessage.Binding{from, to} {
			if err := b.Register(v, couchmessage.RemoteEndpoint{Namespace: namespace, Binding: v}); err != nil {
				t.Fatal(err)
			}
		}
		return b
	}
	resend := newBroker()
	if r, err := resend.Send(context.Background(), from, m.ID, couchmessage.Route{Target: to.Slot}, m.Body); err != nil || r.Status != couchmessage.Submitted {
		t.Fatalf("re-send of a known ID %+v %v", r, err)
	}
	status := newBroker()
	r, err := status.StatusContext(context.Background(), from, m.ID)
	if err != nil || r.Status != couchmessage.Submitted || r.Message.Body != m.Body {
		t.Fatalf("recovered status %+v %v", r, err)
	}
	d.mu.Lock()
	current := d.current.Message.ID
	d.mu.Unlock()
	if current != m.ID {
		t.Fatal("the wrapper accepted a second delivery")
	}
}
