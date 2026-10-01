package couchmessage

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A broker that has forgotten an ID (a Couch restart) and is asked to send it
// again adopts the recipient's retained outcome; it never delivers twice.
func TestBrokerAdoptsRetainedReceiptInsteadOfDeliveringAgain(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	ep := newFakeEndpoint(now.Add(-time.Minute))
	ep.retained.Add(Receipt{Message: Message{ID: "lost", From: from, To: to, Body: "work"}, Status: Submitted})
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	r, err := b.Send(context.Background(), from, "lost", Route{Target: to.Slot}, "work")
	if err != nil || r.Status != Submitted {
		t.Fatalf("adopted %+v %v", r, err)
	}
	select {
	case m := <-ep.delivered:
		t.Fatalf("delivered a retained message again: %+v", m)
	case <-time.After(100 * time.Millisecond):
	}
	if again, err := b.Status(from, "lost"); err != nil || again.Status != Submitted {
		t.Fatalf("adopted receipt not recorded: %+v %v", again, err)
	}
	ep.retained.Add(Receipt{Message: Message{ID: "other", From: from, To: to, Body: "original"}, Status: Submitted})
	if _, err := b.Send(context.Background(), from, "other", Route{Target: to.Slot}, "changed"); err == nil {
		t.Fatal("a different body adopted a retained ID")
	}
}

// After a restart, --message-status for a forgotten ID asks the connected
// recipients; one that does not answer makes the outcome uncertain.
func TestBrokerStatusAsksRecipientsForForgottenIDs(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to, other := brokerBinding("brain:0"), brokerBinding("pair:1"), brokerBinding("pair:2")
	holder, quiet := newFakeEndpoint(now), newFakeEndpoint(now)
	holder.retained.Add(Receipt{Message: Message{ID: "in-flight", From: from, To: to, Body: "work"}, Status: Delivering})
	holder.retained.Add(Receipt{Message: Message{ID: "done", From: from, To: to, Body: "work"}, Status: Submitted})
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, holder)
	_ = b.Register(other, quiet)
	if r, err := b.StatusContext(context.Background(), from, "done"); err != nil || r.Status != Submitted {
		t.Fatalf("recovered %+v %v", r, err)
	}
	// Still in flight at the recipient: no broker job watches it now.
	if r, err := b.StatusContext(context.Background(), from, "in-flight"); err != nil || r.Status != Indeterminate {
		t.Fatalf("in-flight %+v %v", r, err)
	}
	if _, err := b.StatusContext(context.Background(), other, "done"); err == nil {
		t.Fatal("a third party read another pair's receipt")
	}
	if _, err := b.StatusContext(context.Background(), from, "never"); err == nil || errors.Is(err, ErrUncertain) {
		t.Fatalf("unknown everywhere: %v", err)
	}
	quiet.mu.Lock()
	quiet.silent = true
	quiet.mu.Unlock()
	if _, err := b.StatusContext(context.Background(), from, "never"); !errors.Is(err, ErrUncertain) {
		t.Fatalf("a silent recipient reported absence: %v", err)
	}
	if code := protocolError(ErrUncertain, false).Code; code != "uncertain" {
		t.Fatalf("protocol code %q", code)
	}
}
