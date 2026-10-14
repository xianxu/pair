package couchmessage

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSupervisionSubmission(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	view, ok := any(b).(interface {
		Submission(Binding) (uint64, bool)
	})
	if !ok {
		t.Fatal("broker does not expose accepted submission evidence")
	}
	worker := brokerBinding("pair:7")
	check := func(want uint64, known bool) {
		t.Helper()
		if got, ok := view.Submission(worker); got != want || ok != known {
			t.Fatalf("Submission = (%d, %v), want (%d, %v)", got, ok, want, known)
		}
	}
	check(0, false)
	if err := b.Register(worker, newFakeEndpoint(now)); err != nil {
		t.Fatal(err)
	}
	check(0, true)
	if err := b.ReconcileObservation(worker, Observation{LastActivity: now, Sequence: 2, Submission: 4}); err != nil {
		t.Fatal(err)
	}
	check(4, true)
	if err := b.ReconcileObservation(worker, Observation{LastActivity: now, Sequence: 1, Submission: 2}); err != nil {
		t.Fatal(err)
	}
	check(4, true)
	// A rejected observation is not accepted operator-submission evidence.
	if err := b.ReconcileObservation(worker, Observation{Sequence: 3, Submission: 5}); err == nil {
		t.Fatal("submission without an activity timestamp was accepted")
	}
	check(4, true)
	b.Disconnect(worker)
	check(0, false)
}

func TestSupervisionReceiptSnapshot(t *testing.T) {
	for _, end := range []string{"queue expires", "sender disconnects"} {
		t.Run(end, func(t *testing.T) {
			var clock atomic.Int64
			clock.Store(time.Unix(1000, 0).UnixNano())
			now := func() time.Time { return time.Unix(0, clock.Load()) }
			b := NewBroker(context.Background(), now, nil)
			defer b.Close()
			view, ok := any(b).(interface {
				ReceiptSnapshot(string) (Receipt, bool, bool)
			})
			if !ok {
				t.Fatal("broker does not expose supervisor receipt snapshots")
			}
			if _, ok, _ := view.ReceiptSnapshot("unknown"); ok {
				t.Fatal("unknown receipt exists")
			}
			hub, worker := brokerBinding("ops:0"), brokerBinding("pair:7")
			if err := b.Register(hub, newFakeEndpoint(now())); err != nil {
				t.Fatal(err)
			}
			endpoint := &busyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now()), busy: true}
			if err := b.Register(worker, endpoint); err != nil {
				t.Fatal(err)
			}
			r, err := b.Send(context.Background(), hub, "prompt", Route{Target: worker.Slot}, "please resend relevant reports")
			if err != nil || r.Status != Queued {
				t.Fatalf("send: %+v, %v", r, err)
			}
			if got, ok, _ := view.ReceiptSnapshot("prompt"); !ok || got.Status != Queued || got.Message != r.Message {
				t.Fatalf("queued snapshot: %+v, %v", got, ok)
			}
			want := Expired
			if end == "sender disconnects" {
				b.Disconnect(hub)
				want = Cancelled
				if _, err := b.Status(hub, "prompt"); err == nil {
					t.Fatal("disconnected caller retained ordinary status access")
				}
			} else {
				clock.Add(int64(QueueWait))
			}
			got, ok, _ := view.ReceiptSnapshot("prompt")
			if !ok || got.Status != want {
				t.Fatalf("terminal snapshot: %+v, %v; want %s", got, ok, want)
			}
			clock.Store(got.RetainUntil.UnixNano())
			if _, ok, _ := view.ReceiptSnapshot("prompt"); ok {
				t.Fatal("snapshot kept receipt beyond retention horizon")
			}
		})
	}
}

// A terminal receipt is not proof that the endpoint has finished accounting.
type supervisionLateEndpoint struct {
	*FakeDeliveryEndpoint
	entered, release chan struct{}
}

func (e *supervisionLateEndpoint) Deliver(ctx context.Context, _ Message, _ func(Receipt)) (Receipt, error) {
	close(e.entered)
	<-e.release
	return Receipt{}, ctx.Err()
}
func TestSupervisionReceiptSnapshotDeliveryLifetime(t *testing.T) {
	now := time.Now()
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	view, ok := any(b).(interface {
		ReceiptSnapshot(string) (Receipt, bool, bool)
	})
	if !ok {
		t.Fatal("receipt snapshot does not distinguish active delivery from terminal outcome")
	}
	hub, worker := brokerBinding("ops:0"), brokerBinding("pair:7")
	endpoint := &supervisionLateEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), entered: make(chan struct{}), release: make(chan struct{})}
	defer func() {
		select {
		case <-endpoint.release:
		default:
			close(endpoint.release)
		}
	}()
	if err := b.Register(hub, newFakeEndpoint(now)); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(worker, endpoint); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Send(context.Background(), hub, "late", Route{Target: worker.Slot}, "generated"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-endpoint.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("endpoint never entered")
	}
	b.Disconnect(hub)
	r, known, active := view.ReceiptSnapshot("late")
	if !known || !r.Status.Terminal() || !active {
		t.Fatalf("cancelled delivery must remain active: %+v known=%v active=%v", r, known, active)
	}
	close(endpoint.release)
	deadline := time.Now().Add(3 * time.Second)
	for {
		r, known, active = view.ReceiptSnapshot("late")
		if !active {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("delivery remained active after endpoint returned")
		}
		time.Sleep(time.Millisecond)
	}
	if !known || !r.Status.Terminal() {
		t.Fatalf("lost final receipt: %+v known=%v", r, known)
	}
}
