package couchcore

import (
	"fmt"
	"testing"
	"time"
)

func TestOperationAttemptOrderSelectsNewestFailureWithIdenticalClock(t *testing.T) {
	var history LifecycleFailureHistory
	for order := 1; order <= 65; order++ {
		failure := failureFixture(1)
		failure.StartingPath, failure.AttemptEpoch, failure.AttemptOrder = "/project", "epoch", uint64(order)
		failure.Nonce = fmt.Sprintf("attempt-%02d", order)
		if order == 65 {
			failure.Nonce = "zz-newest"
		}
		var err error
		history, err = history.Upsert(failure)
		if err != nil {
			t.Fatal(err)
		}
	}
	rows := ProjectLifecycleFailures([]ActionableThreadSummary{{StartingPath: "/project"}}, history, "")
	if len(history) != 64 || history[0].Nonce != "zz-newest" || rows[0].Failure == nil || rows[0].Failure.Nonce != "zz-newest" {
		t.Fatalf("clock ties lost the latest admitted failure: count=%d head=%s row=%+v", len(history), history[0].Nonce, rows[0].Failure)
	}
}

func TestLifecycleFailureRecencyIsTransitiveAcrossOwners(t *testing.T) {
	rows := []LifecycleFailure{
		{FailedAt: time.Unix(3, 0), AttemptEpoch: "owner-a", AttemptOrder: 1},
		{FailedAt: time.Unix(1, 0), AttemptEpoch: "owner-a", AttemptOrder: 2},
		{FailedAt: time.Unix(2, 0), AttemptEpoch: "owner-b", AttemptOrder: 1},
		{FailedAt: time.Unix(2, 0), AttemptEpoch: "owner-a", AttemptOrder: 3},
	}
	for _, a := range rows {
		for _, b := range rows {
			for _, c := range rows {
				if compareLifecycleFailureRecency(a, b) > 0 && compareLifecycleFailureRecency(b, c) > 0 && compareLifecycleFailureRecency(a, c) <= 0 {
					t.Fatal("recency comparison cannot form a consistent retained order")
				}
			}
		}
	}
}

func TestOperationAttemptAdmissionOrderSurvivesReversedCompletion(t *testing.T) {
	at := time.Unix(100, 0)
	for _, successes := range []bool{true, false} {
		var state OperationAttemptState
		var err error
		for _, order := range []uint64{1, 2} {
			state, _, err = AdvanceOperationAttempt(state, OperationAttemptEvent{Kind: AttemptAdmitted, Order: order})
			if err != nil {
				t.Fatal(err)
			}
		}
		kind := AttemptFailed
		if successes {
			kind = AttemptSucceeded
		}
		state, _, err = AdvanceOperationAttempt(state, OperationAttemptEvent{Kind: kind, Order: 2, At: at})
		if err != nil {
			t.Fatal(err)
		}
		_, decision, err := AdvanceOperationAttempt(state, OperationAttemptEvent{Kind: AttemptFailed, Order: 1, At: at})
		if err != nil {
			t.Fatal(err)
		}
		if successes && (decision.SupersededAt == nil || !decision.SupersededAt.Equal(at)) {
			t.Fatal("older late failure was not superseded by the newer success")
		}
		if !successes && decision.SupersededAt != nil {
			t.Fatal("newer failure invented a successful supersession")
		}
	}
}

func TestOperationAttemptOlderSuccessCannotSupersedeNewerFailure(t *testing.T) {
	var state OperationAttemptState
	for _, order := range []uint64{1, 2} {
		var err error
		state, _, err = AdvanceOperationAttempt(state, OperationAttemptEvent{Kind: AttemptAdmitted, Order: order})
		if err != nil {
			t.Fatal(err)
		}
	}
	state, _, err := AdvanceOperationAttempt(state, OperationAttemptEvent{Kind: AttemptSucceeded, Order: 1, At: time.Unix(200, 0)})
	if err != nil {
		t.Fatal(err)
	}
	_, decision, err := AdvanceOperationAttempt(state, OperationAttemptEvent{Kind: AttemptFailed, Order: 2, At: time.Unix(100, 0)})
	if err != nil || decision.SupersededAt != nil {
		t.Fatalf("older success hid newer failure: %+v, %v", decision, err)
	}
}

func TestOperationAttemptTransitionRejectsReplayAndUnknownOwner(t *testing.T) {
	initial := OperationAttemptState{}
	admitted, _, err := AdvanceOperationAttempt(initial, OperationAttemptEvent{Kind: AttemptAdmitted, Order: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []OperationAttemptEvent{
		{Kind: AttemptAdmitted, Order: 0},
		{Kind: AttemptAdmitted, Order: 1},
		{Kind: AttemptSucceeded, Order: 2, At: time.Unix(1, 0)},
		{Kind: AttemptFailed, Order: 1},
	} {
		if _, _, err := AdvanceOperationAttempt(admitted, event); err == nil {
			t.Fatalf("accepted invalid transition %+v", event)
		}
	}
	finished, _, err := AdvanceOperationAttempt(admitted, OperationAttemptEvent{Kind: AttemptSucceeded, Order: 1, At: time.Unix(1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := AdvanceOperationAttempt(finished, OperationAttemptEvent{Kind: AttemptFailed, Order: 1, At: time.Unix(2, 0)}); err == nil {
		t.Fatal("accepted duplicate terminal completion")
	}
	if _, _, err := AdvanceOperationAttempt(admitted, OperationAttemptEvent{Kind: AttemptFailed, Order: 1, At: time.Unix(2, 0)}); err != nil {
		t.Fatalf("transition mutated its source state: %v", err)
	}
}
