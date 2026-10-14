package couchmessage

import (
	"context"
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
