package couchmessage

import (
	"context"
	"testing"
	"time"
)

func TestSubmissionObserverOnlyAcceptedEvidence(t *testing.T) {
	now := time.Now()
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	view, ok := any(b).(interface{ SetSubmissionObserver(func(Binding, uint64)) })
	if !ok {
		t.Fatal("no accepted-submission observer")
	}
	var values []uint64
	view.SetSubmissionObserver(func(_ Binding, n uint64) { values = append(values, n) })
	binding := brokerBinding("pair:7")
	if err := b.Register(binding, newFakeEndpoint(now)); err != nil {
		t.Fatal(err)
	}
	for _, n := range []uint64{2, 2, 1, 5} {
		if err := b.ReconcileObservation(binding, Observation{Submission: n, LastActivity: now, Sequence: n}); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.ReconcileObservation(binding, Observation{Submission: 6}); err == nil {
		t.Fatal("invalid observation accepted")
	}
	if len(values) != 2 || values[0] != 2 || values[1] != 5 {
		t.Fatalf("accepted counts=%v", values)
	}
}
