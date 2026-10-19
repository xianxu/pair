package couchcore

import (
	"context"
	"errors"
	"testing"
)

func TestDispatchLifecycleFailureFinishesOneOwnedAttempt(t *testing.T) {
	r, store := attemptRecorderFixture()
	cause := errors.New("terminal quit was not confirmed")
	var captured *OperationAttempt
	executors := OperationExecutors{
		Attempt: func(call OperationCall) (*OperationAttempt, error) {
			var err error
			captured, err = r.Begin(call.Name, LifecycleFailureTarget{StartingPath: "/project"})
			return captured, err
		},
		LiveOwner: func(call OperationCall) (any, error) {
			if OperationAttemptFromContext(call.Context) != captured {
				t.Fatal("executor did not borrow the dispatch attempt")
			}
			return nil, cause
		},
	}
	_, err := DispatchOperation(executors, OperationCall{Name: "park", Context: context.Background(), Args: map[string]string{"repo-scope": "scope", "tag": "exact"}, Implicit: true})
	if !errors.Is(err, cause) || FailureNonce(err) != captured.Nonce() || len(store.Writes) != 1 {
		t.Fatalf("missing single terminal failure: %v, %+v", err, store.Writes)
	}
}

func TestDispatchLifecycleFailureBorrowsDeferredAttemptAndIgnoresReads(t *testing.T) {
	r, store := attemptRecorderFixture()
	parent, err := r.Begin("relaunch", LifecycleFailureTarget{StartingPath: "/project"})
	if err != nil {
		t.Fatal(err)
	}
	executors := OperationExecutors{
		Attempt: func(OperationCall) (*OperationAttempt, error) {
			t.Fatal("nested or read call allocated an attempt")
			return nil, nil
		},
		LiveOwner:   func(OperationCall) (any, error) { return nil, errors.New("effect failed") },
		DirectStore: func(OperationCall) (any, error) { return "read", nil },
	}
	_, cause := DispatchOperation(executors, OperationCall{Name: "park", Context: WithOperationAttempt(context.Background(), parent), Args: map[string]string{"repo-scope": "scope", "tag": "exact"}, Implicit: true})
	if len(store.Writes) != 0 {
		t.Fatal("execution published before terminal adoption/readiness")
	}
	parent.Finish(nil, cause)
	if len(store.Writes) != 1 {
		t.Fatal("terminal owner did not publish")
	}
	if _, err := DispatchOperation(executors, OperationCall{Name: "list"}); err != nil {
		t.Fatal(err)
	}
}

func TestDispatchLifecycleFailureSuccessIgnoresUnavailableDiagnostics(t *testing.T) {
	r, store := attemptRecorderFixture()
	store.ReadError = errors.New("diagnostics unreadable")
	var warning string
	executors := OperationExecutors{
		Attempt: func(call OperationCall) (*OperationAttempt, error) {
			return r.Begin(call.Name, LifecycleFailureTarget{StartingPath: "/project"})
		},
		DiagnosticWarning: func(text string) { warning = text },
		LiveOwner:         func(OperationCall) (any, error) { return "completed", nil },
	}
	value, err := DispatchOperation(executors, OperationCall{Name: "park", Args: map[string]string{"repo-scope": "scope", "tag": "exact"}, Implicit: true})
	if err != nil || value != "completed" || warning != "" {
		t.Fatalf("success depended on diagnostics: %v %v %q", value, err, warning)
	}
}

func TestLifecycleFailureUnresolvedRelativeTargetPreservesAdmission(t *testing.T) {
	c := &Couch{}
	for _, path := range []string{".", "relative", "/project/../other"} {
		attempt, err := c.OperationAttemptForCall(OperationCall{Name: "prepare-start", Args: map[string]string{"path": path}})
		if err != nil || attempt == nil {
			t.Fatalf("completion log failed to retain unresolved dispatch evidence %q: %v %v", path, attempt, err)
		}
	}
}
