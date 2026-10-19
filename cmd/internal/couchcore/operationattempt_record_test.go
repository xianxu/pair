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

func TestOperationAttemptSuccessDoesNoDiagnosticIO(t *testing.T) {
	calls := 0
	r := NewLifecycleFailureRecorder(nil, time.Now, func() (string, error) { calls++; return "", errors.New("entropy unavailable") })
	a, err := r.Begin("reboot", LifecycleFailureTarget{Address: ThreadAddress{RepoScope: "scope", Tag: "exact"}})
	if err != nil {
		t.Fatalf("diagnostics refused admission: %v", err)
	}
	value, err := a.Finish("completed", nil)
	if err != nil || value != "completed" || a.DiagnosticWarning() != "" || calls != 0 {
		t.Fatalf("success touched diagnostics or changed result: %v %v %q calls=%d", value, err, a.DiagnosticWarning(), calls)
	}
}

func TestOperationAttemptLateFailureIsHistoricalEvidence(t *testing.T) {
	r, store := attemptRecorderFixture()
	target := LifecycleFailureTarget{StartingPath: "/project"}
	older, _ := r.Begin("relaunch", target)
	newer, _ := r.Begin("resume", target)
	newer.Finish("running", nil)
	cause := errors.New("older readiness unconfirmed")
	older.Finish(nil, cause)
	history, err := store.Read()
	if err != nil || len(history) != 1 || history[0].FinalError != cause.Error() || len(store.Writes) != 1 {
		t.Fatalf("completion log lost historical failure: %+v %v", history, err)
	}
}

func TestOperationAttemptNonceFailurePreservesEffectFailure(t *testing.T) {
	r := NewLifecycleFailureRecorder(NewFakeLaunchFailureStore(), time.Now, func() (string, error) { return "", errors.New("entropy unavailable") })
	a, err := r.Begin("park", LifecycleFailureTarget{Target: "pair:8"})
	if err != nil {
		t.Fatalf("diagnostics refused admission: %v", err)
	}
	cause := context.Canceled
	_, err = a.Finish(nil, cause)
	if !errors.Is(err, cause) || a.DiagnosticWarning() == "" {
		t.Fatalf("original failure or diagnostic warning lost: %v", err)
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

// ownerContextFailureStore observes the lifetime forwarded to optional storage.
type ownerContextFailureStore struct {
	*FakeLaunchFailureStore
	observed error
}

func (s *ownerContextFailureStore) Save(ctx context.Context, _ LaunchFailure) error {
	s.observed = ctx.Err()
	return s.observed
}
func TestOperationAttemptPublicationHonorsOwnerLifetime(t *testing.T) {
	store := &ownerContextFailureStore{FakeLaunchFailureStore: NewFakeLaunchFailureStore()}
	r := NewLifecycleFailureRecorder(store, time.Now, func() (string, error) { return "operation", nil })
	a, _ := r.Begin("park", LifecycleFailureTarget{Target: "pair:8"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cause := errors.New("original lifecycle failure")
	_, err := a.FinishContext(ctx, nil, cause)
	if !errors.Is(store.observed, context.Canceled) || !errors.Is(err, cause) || a.DiagnosticWarning() == "" {
		t.Fatalf("publication ignored owner cancellation or replaced result: observed=%v result=%v warning=%q", store.observed, err, a.DiagnosticWarning())
	}
}
