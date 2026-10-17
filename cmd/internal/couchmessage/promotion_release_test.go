package couchmessage

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

type promotionReleaseGate struct {
	*busyEndpoint
	calls   atomic.Int32
	gateAt  int32
	entered chan time.Time
	open    chan struct{}
	once    sync.Once
}

func (e *promotionReleaseGate) unblock() { e.once.Do(func() { close(e.open) }) }

func (e *promotionReleaseGate) Release(ctx context.Context, id string) error {
	// Gate the chosen cleanup independently of scheduling. End-to-end tests
	// skip initial admission cleanup; direct promotion tests gate the first.
	// Every test opens the gate before broker cleanup.
	if e.calls.Add(1) == e.gateAt {
		e.entered <- time.Now()
		<-e.open
	}
	return e.FakeDeliveryEndpoint.Release(ctx, id)
}

func TestPromotionReleaseDoesNotBlockOtherReceiver(t *testing.T) {
	now := time.Now()
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	t.Cleanup(func() { _ = b.Close() })
	from, blocked, other := brokerBinding("pair:1"), brokerBinding("ops:0"), brokerBinding("ops:1")
	gate := &promotionReleaseGate{busyEndpoint: &busyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), busy: true}, gateAt: 2, entered: make(chan time.Time, 1), open: make(chan struct{})}
	t.Cleanup(gate.unblock)
	independent := newFakeEndpoint(now)
	for binding, endpoint := range map[Binding]DeliveryEndpoint{from: newFakeEndpoint(now), blocked: gate, other: independent} {
		if err := b.Register(binding, endpoint); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Send(context.Background(), from, "known", Route{Target: other.Slot}, "known receipt"); err != nil {
		t.Fatal(err)
	}
	delivered(t, independent)
	independent.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, b, from, "known", Submitted)
	if _, err := b.Send(context.Background(), from, "blocked", Route{Target: blocked.Slot}, "wait"); err != nil {
		t.Fatal(err)
	}
	var entered time.Time
	select {
	case entered = <-gate.entered:
	case <-testwait.After(t):
		t.Fatal("promotion never entered Release")
	}
	type result struct {
		name          string
		elapsed       time.Duration
		beforeRelease bool
		err           error
	}
	results := make(chan result, 2)
	for _, operation := range []string{"Send", "Status"} {
		go func(name string) {
			start := time.Now()
			var err error
			if name == "Send" {
				_, err = b.Send(context.Background(), from, "independent", Route{Target: other.Slot}, "another report")
			} else {
				_, err = b.Status(from, "known")
			}
			before := true
			select {
			case <-gate.open:
				before = false
			default:
			}
			results <- result{name, time.Since(start), before, err}
		}(operation)
	}
	var got []result
	budget := testwait.After(t)
	for len(got) < 2 {
		select {
		case r := <-results:
			got = append(got, r)
		case <-budget:
			t.Errorf("only %d/2 independent operations completed while promotion Release was withheld for %v", len(got), time.Since(entered))
			gate.unblock()
			budget = testwait.After(t)
		}
	}
	for _, r := range got {
		t.Logf("%s elapsed=%v completed_before_Release=%v error=%v", r.name, r.elapsed, r.beforeRelease, r.err)
		if r.err != nil || !r.beforeRelease {
			t.Errorf("independent %s waited for unrelated Release: %+v", r.name, r)
		}
	}
	gate.unblock()
}

// pendingPromotionRelease drives promote explicitly so its cleanup completion is a
// joinable event. Enqueue through the real model without waking the idle worker.
func pendingPromotionRelease(t *testing.T) (*Broker, *SlotActor, Binding, *promotionReleaseGate, <-chan bool) {
	t.Helper()
	now := time.Now()
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	from, to := brokerBinding("pair:1"), brokerBinding("ops:0")
	gate := &promotionReleaseGate{busyEndpoint: &busyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), busy: true}, gateAt: 1, entered: make(chan time.Time, 1), open: make(chan struct{})}
	for binding, endpoint := range map[Binding]DeliveryEndpoint{from: newFakeEndpoint(now), to: gate} {
		if err := b.Register(binding, endpoint); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	a := b.actors[to]
	originalCancel := a.cancel
	a.cancel = func() { originalCancel(); cancel() }
	m := Message{ID: "cleanup", From: from, To: to, Body: "old report", Deadline: now.Add(PasteTimeout), AcceptedAt: now}
	err := b.apply(a, Event{Kind: Enqueue, Binding: to, Message: m, At: now})
	b.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan bool, 1)
	go func() { done <- b.promote(ctx, a); close(done) }()
	t.Cleanup(func() { gate.unblock(); cancel(); _ = b.Close(); <-done })
	select {
	case <-gate.entered:
	case <-testwait.After(t):
		t.Fatal("promotion never entered cleanup")
	}
	return b, a, from, gate, done
}

func TestPromotionReleaseKeepsReservationWhenSenderCancels(t *testing.T) {
	b, a, from, gate, done := pendingPromotionRelease(t)
	b.Disconnect(from)
	next := brokerBinding("pair:2")
	if err := b.Register(next, newFakeEndpoint(time.Now())); err != nil {
		t.Fatal(err)
	}
	r, err := b.Send(context.Background(), next, "next", Route{Target: a.binding.Slot}, "next report")
	if err != nil || r.Status != Queued {
		t.Fatalf("send during cleanup: %+v %v", r, err)
	}
	b.mu.Lock()
	reservation := a.reservation
	b.mu.Unlock()
	gate.mu.Lock()
	refusals := gate.refusals
	gate.mu.Unlock()
	if reservation != "cleanup" || refusals != 1 {
		t.Errorf("cleanup lost exclusivity: reservation=%q reserve attempts=%d", reservation, refusals)
	}
	gate.unblock()
	if retry := <-done; !retry {
		t.Fatal("connected receiver lost promotion retry")
	}
}

func TestPromotionReleaseCannotClearReplacementReservation(t *testing.T) {
	b, old, from, gate, done := pendingPromotionRelease(t)
	replacement := old.binding
	replacement.Nonce = "replacement"
	fresh := &gatedEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(time.Now()), entered: make(chan struct{}), open: make(chan struct{})}
	var openOnce sync.Once
	open := func() { openOnce.Do(func() { close(fresh.open) }) }
	t.Cleanup(open)
	if err := b.Register(replacement, fresh); err != nil {
		t.Fatal(err)
	}
	admitted := make(chan error, 1)
	go func() {
		_, err := b.Send(context.Background(), from, "replacement-attempt", Route{Target: replacement.Slot}, "new report")
		admitted <- err
		close(admitted)
	}()
	t.Cleanup(func() { open(); <-admitted })
	select {
	case <-fresh.entered:
	case <-testwait.After(t):
		t.Fatal("replacement admission never reached Observe")
	}
	gate.unblock()
	if retry := <-done; retry {
		t.Fatal("disconnected actor scheduled another promotion")
	}
	b.mu.Lock()
	reservation := b.actors[replacement].reservation
	b.mu.Unlock()
	if reservation != "replacement-attempt" {
		t.Errorf("old cleanup cleared replacement reservation: %q", reservation)
	}
	open()
	if err := <-admitted; err == nil {
		t.Fatal("gated observation unexpectedly succeeded")
	}
	if _, err := b.Send(context.Background(), from, "fresh", Route{Target: replacement.Slot}, "fresh report"); err != nil {
		t.Fatal(err)
	}
	if m := delivered(t, fresh.FakeDeliveryEndpoint); m.ID != "fresh" || m.To != replacement {
		t.Fatalf("replacement received old work: %+v", m)
	}
	fresh.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, b, from, "fresh", Submitted)
	r, err := b.Status(from, "cleanup")
	if err != nil || r.Status != Cancelled {
		t.Fatalf("old queued work survived replacement: %+v %v", r, err)
	}
}
