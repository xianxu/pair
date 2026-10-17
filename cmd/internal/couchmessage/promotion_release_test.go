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
	entered chan time.Time
	open    chan struct{}
	once    sync.Once
}

func (e *promotionReleaseGate) unblock() { e.once.Do(func() { close(e.open) }) }

func (e *promotionReleaseGate) Release(ctx context.Context, id string) error {
	// Initial admission cleans up first; the second call is the worker's
	// failed promotion. The gate controls its completion independently of
	// scheduling and is always opened before broker cleanup.
	if e.calls.Add(1) == 2 {
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
	gate := &promotionReleaseGate{busyEndpoint: &busyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), busy: true}, entered: make(chan time.Time, 1), open: make(chan struct{})}
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
		t.Logf("%s elapsed=%v completed_before_Release=%v", r.name, r.elapsed, r.beforeRelease)
		if r.err != nil || !r.beforeRelease {
			t.Errorf("independent %s waited for unrelated Release: %+v", r.name, r)
		}
	}
	gate.unblock()
}
