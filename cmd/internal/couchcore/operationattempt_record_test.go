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
	if activeOperationAttemptSlots(r) != 0 {
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

func TestOperationAttemptThreadOnlySuccessSupersedesExactIdentity(t *testing.T) {
	for _, path := range []string{"", "/project"} {
		r, store := attemptRecorderFixture()
		target := LifecycleFailureTarget{Address: ThreadAddress{RepoScope: "scope", Tag: "exact"}}
		failed, err := r.Begin("park", target)
		if err != nil {
			t.Fatal(err)
		}
		failed.Finish(nil, errors.New("park failed"))
		other, err := r.Begin("park", LifecycleFailureTarget{Address: ThreadAddress{RepoScope: "scope", Tag: "other"}})
		if err != nil {
			t.Fatal(err)
		}
		other.Finish(nil, nil)
		before, _ := store.Read()
		if before[0].SupersededAt != nil {
			t.Fatal("another conversation superseded exact-thread evidence")
		}
		target.StartingPath = path
		success, err := r.Begin("park", target)
		if err != nil {
			t.Fatal(err)
		}
		success.Finish(nil, nil)
		after, _ := store.Read()
		if after[0].SupersededAt == nil {
			t.Fatalf("same exact conversation left its failure unresolved (path %q)", path)
		}
	}
}

func TestOperationAttemptMixedTargetLateFailureCannotHideNewerSuccess(t *testing.T) {
	r, store := attemptRecorderFixture()
	threads, _, thread := createControllerThread(t)
	c := &Couch{Threads: threads, failureRecorder: r}
	reads := 0
	threads.hooks.AfterGetThread = func(address ThreadAddress) error {
		if address != thread.Address {
			t.Fatalf("looked up another thread: %+v", address)
		}
		reads++
		return nil
	}
	older, err := c.OperationAttemptForCall(OperationCall{Name: "park", Args: map[string]string{"repo-scope": thread.Address.RepoScope, "tag": string(thread.Address.Tag)}})
	if err != nil {
		t.Fatal(err)
	}
	newer, err := c.OperationAttemptForCall(OperationCall{Name: "park", FailureTarget: &LifecycleFailureTarget{Address: thread.Address, StartingPath: thread.StartingPath}})
	if err != nil {
		t.Fatal(err)
	}
	if reads != 1 {
		t.Fatalf("wanted one exact bind and no cached-path lookup, got %d", reads)
	}
	newer.Finish(nil, nil)
	if len(store.Writes) != 0 {
		t.Fatal("ordinary success wrote diagnostics")
	}
	older.Finish(nil, errors.New("late readiness failure"))
	history, err := store.Read()
	if err != nil || len(history) != 1 || history[0].SupersededAt == nil || history[0].StartingPath != thread.StartingPath {
		t.Fatalf("mixed target keys resurfaced an older failure after success: %+v %v", history, err)
	}
}

func TestOperationAttemptUnknownThreadBindingRefusesBeforeEffects(t *testing.T) {
	threads, _, thread := createControllerThread(t)
	for _, scope := range []string{thread.Address.RepoScope, ""} {
		c := &Couch{Threads: threads}
		_, err := DispatchOperation(OperationExecutors{
			Attempt:   c.OperationAttemptForCall,
			LiveOwner: func(OperationCall) (any, error) { t.Fatal("unknown binding reached effects"); return nil, nil },
		}, OperationCall{Name: "park", Args: map[string]string{"repo-scope": scope, "tag": "couch-fedcba9876543210"}, Implicit: true})
		if err == nil {
			t.Fatal("unknown thread binding was admitted")
		}
	}
}
