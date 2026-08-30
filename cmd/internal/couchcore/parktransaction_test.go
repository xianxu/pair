package couchcore

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

func testParkIdentity() ParkIdentity {
	return ParkIdentity{
		Nonce:   "park-0123456789abcdef",
		Address: ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"},
		PID:     42, ProcessIdentity: "pid-start:123456",
	}
}

func advancePark(t *testing.T, transaction *ParkTransaction, event ParkEvent) (ParkTransaction, ParkDecision) {
	t.Helper()
	next, decision, err := AdvanceParkTransaction(transaction, event)
	if err != nil {
		t.Fatalf("AdvanceParkTransaction(%s): %v", event.Kind, err)
	}
	return next, decision
}

func TestAdvanceParkTransactionMatrix(t *testing.T) {
	identity := testParkIdentity()
	begin := ParkEvent{Kind: ParkBegin, Identity: identity, BaseRevision: 7, RecordRevision: 8}

	t.Run("begin creates requested attempt with stable identity and distinct revisions", func(t *testing.T) {
		next, decision := advancePark(t, nil, begin)
		if next.Identity != identity || next.BaseRevision != 7 || next.RecordRevision != 8 || next.Phase != ParkRequested {
			t.Fatalf("begin = %+v", next)
		}
		if len(next.Attempts) != 1 || next.Attempts[0].Number != 1 || next.Attempts[0].Closed {
			t.Fatalf("attempts = %+v", next.Attempts)
		}
		if decision.Finalize || decision.HistoricalNoOp {
			t.Fatalf("decision = %+v", decision)
		}
	})

	t.Run("request committed advances to awaiting", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		next, _ := advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 9})
		if next.Phase != ParkAwaitingCompletion || next.RecordRevision != 9 {
			t.Fatalf("next = %+v", next)
		}
	})

	t.Run("request publish failure stays requested and retryable", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		next, _ := advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 9,
			Failure: &ParkFailure{Code: pairlifecycle.FailureRequestPublishFailed, Diagnostic: "disk full"},
		})
		if next.Phase != ParkRequested || next.Attempts[0].Closed || next.Attempts[0].Failure == nil || next.Attempts[0].Failure.Code != pairlifecycle.FailureRequestPublishFailed {
			t.Fatalf("next = %+v", next)
		}
	})

	t.Run("timeout retains awaiting and accepts late success", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 9})
		timedOut, _ := advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 10,
			Failure: &ParkFailure{Code: pairlifecycle.FailureTimeout, Diagnostic: "cleanup deadline"},
		})
		if timedOut.Phase != ParkAwaitingCompletion || !timedOut.Attempts[0].TimedOut || timedOut.Attempts[0].Closed {
			t.Fatalf("timed out = %+v", timedOut)
		}
		closed, decision := advancePark(t, &timedOut, ParkEvent{Kind: ParkCompletionSucceeded, Attempt: 1, RecordRevision: 11})
		if !closed.Closed || closed.SuccessfulAttempt != 1 || !decision.Finalize || decision.SuccessfulAttempt != 1 {
			t.Fatalf("closed = %+v, decision = %+v", closed, decision)
		}
	})

	t.Run("missing completion becomes unknown", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 9})
		next, _ := advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 10,
			Failure: &ParkFailure{Code: pairlifecycle.FailureCompletionMissing, Diagnostic: "session absent"},
		})
		if next.Phase != ParkUnknown || next.Attempts[0].Closed {
			t.Fatalf("next = %+v", next)
		}
	})

	t.Run("stale completion is retained as typed occupied failure", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		next, _ := advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 9,
			Failure: &ParkFailure{Code: pairlifecycle.FailureStaleCompletion, Diagnostic: "binding mismatch"},
		})
		if next.Phase != ParkRequested || next.Attempts[0].Failure.Code != pairlifecycle.FailureStaleCompletion {
			t.Fatalf("next = %+v", next)
		}
	})

	t.Run("cleanup failure closes only its attempt and append increases", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 9})
		failed, _ := advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 10,
			Failure: &ParkFailure{Code: pairlifecycle.FailureCleanupFailed, Diagnostic: "editor remains"},
		})
		if !failed.Attempts[0].Closed || failed.Closed {
			t.Fatalf("failed = %+v", failed)
		}
		next, _ := advancePark(t, &failed, ParkEvent{Kind: ParkAttemptAppended, RecordRevision: 11})
		if next.Phase != ParkRequested || len(next.Attempts) != 2 || next.Attempts[1].Number != 2 {
			t.Fatalf("next = %+v", next)
		}
	})

	t.Run("older attempt success closes whole transaction", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 9})
		transaction, _ = advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 10,
			Failure: &ParkFailure{Code: pairlifecycle.FailureTimeout, Diagnostic: "first deadline"},
		})
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkAttemptAppended, RecordRevision: 11})
		closed, decision := advancePark(t, &transaction, ParkEvent{Kind: ParkCompletionSucceeded, Attempt: 1, RecordRevision: 12})
		if !closed.Closed || closed.SuccessfulAttempt != 1 || !decision.Finalize {
			t.Fatalf("closed = %+v, decision = %+v", closed, decision)
		}
	})

	t.Run("immutable failed attempt cannot later become success", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 9})
		transaction, _ = advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 10,
			Failure: &ParkFailure{Code: pairlifecycle.FailureCleanupFailed, Diagnostic: "immutable result"},
		})
		if _, _, err := AdvanceParkTransaction(&transaction, ParkEvent{Kind: ParkCompletionSucceeded, Attempt: 1, RecordRevision: 11}); err == nil {
			t.Fatal("closed failed attempt accepted a later success")
		}
	})

	t.Run("late historical failure cannot clobber newer active attempt", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 9})
		transaction, _ = advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 10,
			Failure: &ParkFailure{Code: pairlifecycle.FailureTimeout, Diagnostic: "first deadline"},
		})
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkAttemptAppended, RecordRevision: 11})
		transaction, _ = advancePark(t, &transaction, ParkEvent{Kind: ParkRequestCommitted, Attempt: 2, RecordRevision: 12})

		lateFailure, _ := advancePark(t, &transaction, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 13,
			Failure: &ParkFailure{Code: pairlifecycle.FailureCleanupFailed, Diagnostic: "late immutable result"},
		})
		if lateFailure.Phase != ParkAwaitingCompletion || !lateFailure.Attempts[0].Closed ||
			lateFailure.Attempts[0].Failure == nil || lateFailure.Attempts[0].Failure.Code != pairlifecycle.FailureCleanupFailed {
			t.Fatalf("late failure clobbered active attempt: %+v", lateFailure)
		}

		before := cloneParkTransaction(lateFailure)
		if _, _, err := AdvanceParkTransaction(&lateFailure, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 14,
			Failure: &ParkFailure{Code: pairlifecycle.FailureStaleCompletion, Diagnostic: "must not replace"},
		}); err == nil {
			t.Fatal("later non-success event overwrote a closed attempt")
		}
		if !reflect.DeepEqual(lateFailure, before) {
			t.Fatal("rejected historical failure mutated transaction")
		}
	})

	t.Run("revision conflict and replacement incarnation become unknown", func(t *testing.T) {
		for _, code := range []pairlifecycle.FailureCode{
			pairlifecycle.FailureRevisionConflict,
			pairlifecycle.FailureReplacementIncarnation,
		} {
			transaction, _ := advancePark(t, nil, begin)
			next, _ := advancePark(t, &transaction, ParkEvent{
				Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 9,
				Failure: &ParkFailure{Code: code, Diagnostic: string(code)},
			})
			if next.Phase != ParkUnknown || next.Attempts[0].Failure.Code != code {
				t.Fatalf("%s next = %+v", code, next)
			}
		}
	})

	t.Run("abandon tombstones and all later results are historical no-ops", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		abandoned, _ := advancePark(t, &transaction, ParkEvent{Kind: ParkAbandoned, RecordRevision: 9})
		if !abandoned.Closed || !abandoned.Tombstoned || abandoned.SuccessfulAttempt != 0 {
			t.Fatalf("abandoned = %+v", abandoned)
		}
		before := abandoned
		after, decision := advancePark(t, &abandoned, ParkEvent{Kind: ParkCompletionSucceeded, Attempt: 1, RecordRevision: 10})
		if !decision.HistoricalNoOp || !reflect.DeepEqual(after, before) {
			t.Fatalf("after = %+v, decision = %+v", after, decision)
		}
	})

	t.Run("success closure makes newer results historical no-ops", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		closed, _ := advancePark(t, &transaction, ParkEvent{Kind: ParkCompletionSucceeded, Attempt: 1, RecordRevision: 9})
		after, decision := advancePark(t, &closed, ParkEvent{
			Kind: ParkFailureObserved, Attempt: 1, RecordRevision: 10,
			Failure: &ParkFailure{Code: pairlifecycle.FailureCleanupFailed, Diagnostic: "late"},
		})
		if !decision.HistoricalNoOp || !reflect.DeepEqual(after, closed) {
			t.Fatalf("after = %+v, decision = %+v", after, decision)
		}
	})

	t.Run("rejects identity mutation stale revision and skipped attempts", func(t *testing.T) {
		transaction, _ := advancePark(t, nil, begin)
		for _, event := range []ParkEvent{
			{Kind: ParkRequestCommitted, Attempt: 1, Identity: ParkIdentity{Nonce: "other"}, RecordRevision: 9},
			{Kind: ParkRequestCommitted, Attempt: 1, RecordRevision: 8},
			{Kind: ParkCompletionSucceeded, Attempt: 2, RecordRevision: 9},
		} {
			before := transaction
			if _, _, err := AdvanceParkTransaction(&transaction, event); err == nil {
				t.Fatalf("accepted event %+v", event)
			}
			if !reflect.DeepEqual(transaction, before) {
				t.Fatalf("failed event mutated input")
			}
		}
	})
}

func TestMonotonicLastActiveAt(t *testing.T) {
	old := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	forward := old.Add(time.Second)
	if got := MonotonicLastActiveAt(old, forward); got != forward {
		t.Fatalf("forward = %v", got)
	}
	if got := MonotonicLastActiveAt(old, old); got != old {
		t.Fatalf("equal = %v", got)
	}
	if got := MonotonicLastActiveAt(old, old.Add(-time.Second)); got != old {
		t.Fatalf("backward = %v", got)
	}
}
