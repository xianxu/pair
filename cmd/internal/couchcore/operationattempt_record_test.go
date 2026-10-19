package couchcore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func attemptRecorderFixture() (*LifecycleFailureRecorder, *FakeLaunchFailureStore) {
	store := NewFakeLaunchFailureStore()
	var n int
	recorder := NewLifecycleFailureRecorder(store, func() time.Time { return time.Unix(100, 0) }, func() (string, error) {
		n++
		return fmt.Sprintf("operation-%d", n), nil
	})
	return recorder, store
}

func TestOperationAttemptFinishLateFailureAfterNoWriteSuccess(t *testing.T) {
	r, store := attemptRecorderFixture()
	target := LifecycleFailureTarget{StartingPath: "/project", Slot: "project:0"}
	older, err := r.Begin("relaunch", target)
	if err != nil {
		t.Fatal(err)
	}
	newer, err := r.Begin("resume", target)
	if err != nil {
		t.Fatal(err)
	}
	if value, err := newer.Finish("running", nil); err != nil || value != "running" {
		t.Fatalf("successful effect changed: %v %v", value, err)
	}
	if len(store.Writes) != 0 {
		t.Fatal("ordinary success wrote diagnostic history")
	}
	cause := errors.New("older readiness unconfirmed")
	_, err = older.Finish(nil, cause)
	if !errors.Is(err, cause) || FailureNonce(err) != older.Nonce() {
		t.Fatalf("lost original failure identity: %v", err)
	}
	history, err := store.Read()
	if err != nil || len(history) != 1 || history[0].SupersededAt == nil {
		t.Fatalf("late failure resurfaced: %+v %v", history, err)
	}
	if r.ActiveSlots() != 0 {
		t.Fatal("coordinator retained a settled slot")
	}
}

func TestOperationAttemptFinishPublishesOnceAndPreservesErrorChain(t *testing.T) {
	r, store := attemptRecorderFixture()
	attempt, err := r.Begin("recover-park", LifecycleFailureTarget{StartingPath: "/project", ParkNonce: "park-exact"})
	if err != nil {
		t.Fatal(err)
	}
	cause := &SlotOperationError{Code: "exact-refusal", Detail: "park is still finishing; retry recover-park"}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, result := attempt.Finish(nil, cause)
			var got *SlotOperationError
			if !errors.As(result, &got) || got != cause || FailureNonce(result) != attempt.Nonce() {
				t.Errorf("lost typed error: %v", result)
			}
		}()
	}
	wg.Wait()
	if len(store.Writes) != 1 || store.Writes[0].ParkNonce != "park-exact" {
		t.Fatalf("duplicated or rebound publication: %+v", store.Writes)
	}
}

func TestOperationAttemptFinishSaveFailureDoesNotInventSuccess(t *testing.T) {
	r, store := attemptRecorderFixture()
	store.SaveError = errors.New("store unavailable")
	attempt, err := r.Begin("park", LifecycleFailureTarget{StartingPath: "/project"})
	if err != nil {
		t.Fatal(err)
	}
	cause := context.Canceled
	_, result := attempt.Finish(nil, cause)
	if !errors.Is(result, cause) || FailureNonce(result) == "" || attempt.DiagnosticWarning() == "" {
		t.Fatalf("save error obscured original outcome: %v", result)
	}
}

func TestOperationAttemptContextBorrowsSameTerminalOwner(t *testing.T) {
	r, _ := attemptRecorderFixture()
	attempt, err := r.Begin("relaunch", LifecycleFailureTarget{StartingPath: "/project"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithOperationAttempt(context.Background(), attempt)
	if OperationAttemptFromContext(ctx) != attempt || OperationAttemptFromContext(context.Background()) != nil {
		t.Fatal("nested operation did not borrow its parent attempt")
	}
	attempt.Finish(nil, nil)
}
