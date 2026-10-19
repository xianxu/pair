package couchcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type lifecycleQueuedFixture struct {
	prepare func(context.Context) (couchcore.OperationCall, error)
	finish  func(any, error)
}

func newLifecycleQueueFixture() (*lifecycleRemoteQueue, *couchcore.FakeLaunchFailureStore, *[]lifecycleQueuedFixture) {
	store := couchcore.NewFakeLaunchFailureStore()
	c := &couchcore.Couch{LaunchFailures: store}
	jobs := &[]lifecycleQueuedFixture{}
	q := &lifecycleRemoteQueue{couch: c,
		rows: func() []couchcore.ActionableThreadSummary {
			return []couchcore.ActionableThreadSummary{{StartingPath: "/slots/project", Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: "original"}, Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "project", PrimaryRoot: "/project", WorktreeRoot: "/slots/project", Number: 1}}}}
		},
		enqueue: func(_ string, _ string, prepare func(context.Context) (couchcore.OperationCall, error), _ func(), finish func(any, error)) error {
			*jobs = append(*jobs, lifecycleQueuedFixture{prepare, finish})
			return nil
		},
	}
	return q, store, jobs
}

func TestLifecycleFailureQueuePreparationIsRetainedBeforeDispatch(t *testing.T) {
	q, store, jobs := newLifecycleQueueFixture()
	cause := errors.New("park owner is still finishing")
	finished := make(chan error, 1)
	err := q.Enqueue("remote\x00/project:1", "recover-park", func(ctx context.Context) (couchcore.OperationCall, error) {
		if couchcore.OperationAttemptFromContext(ctx) == nil {
			t.Fatal("preparation has no terminal owner")
		}
		return couchcore.OperationCall{}, cause
	}, func() {}, func(_ any, err error) { finished <- err })
	if err != nil {
		t.Fatal(err)
	}
	job := (*jobs)[0]
	_, err = job.prepare(t.Context())
	job.finish(nil, err)
	if result := <-finished; !errors.Is(result, cause) || couchcore.FailureNonce(result) == "" {
		t.Fatal(result)
	}
	if len(store.Writes) != 1 || store.Writes[0].StartingPath != "/slots/project" || store.Writes[0].Address.Tag != "original" {
		t.Fatalf("failed preparation lost its frozen target: %+v", store.Writes)
	}
}

func TestLifecycleFailureQueueReadinessOwnsFinalCompletion(t *testing.T) {
	q, store, jobs := newLifecycleQueueFixture()
	entered, release := make(chan struct{}), make(chan struct{})
	wait := func(context.Context) error {
		close(entered)
		<-release
		return errors.New("readiness was not confirmed")
	}
	finished := make(chan error, 1)
	prepare := func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "relaunch", Context: ctx}, nil
	}
	if err := q.Enqueue("remote\x00/project:1", "relaunch", prepare, func() {}, func(_ any, err error) { finished <- err }, &wait); err != nil {
		t.Fatal(err)
	}
	first := (*jobs)[0]
	if _, err := first.prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	go first.finish(nil, nil)
	<-entered
	if len(store.Writes) != 0 {
		t.Fatal("effect success finished before readiness")
	}
	if err := q.Enqueue("remote\x00/project:1", "resume", prepare, func() {}, func(any, error) {}); err != nil {
		t.Fatal(err)
	}
	newer := (*jobs)[1]
	newer.prepare(t.Context())
	newer.finish(nil, nil)
	close(release)
	if err := <-finished; err == nil {
		t.Fatal("lost readiness failure")
	}
	if len(store.Writes) != 1 || store.Writes[0].FinalError != "readiness was not confirmed" {
		t.Fatalf("late failure was not retained as historical evidence: %+v", store.Writes)
	}
}

func TestLifecycleFailureQueueCompletedCleanupWarningHasReceiptNonce(t *testing.T) {
	q, store, jobs := newLifecycleQueueFixture()
	finished := make(chan couchmessage.ReceiptOutcome, 1)
	if err := q.Enqueue("remote\x00/project:1", "park", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "park", Context: ctx}, nil
	}, func() {}, func(value any, err error) { finished <- slotOperationOutcome(value, err) }); err != nil {
		t.Fatal(err)
	}
	job := (*jobs)[0]
	job.prepare(t.Context())
	job.finish(couchcore.ParkResult{CleanupError: errors.New("sidecar removal unavailable")}, nil)
	outcome := <-finished
	if outcome.Status != couchmessage.ReceiptSucceeded || outcome.FailureNonce == "" || outcome.Warning == "" || len(store.Writes) != 1 || store.Writes[0].Nonce != outcome.FailureNonce {
		t.Fatalf("completed effect lost cleanup warning: %+v, %+v", outcome, store.Writes)
	}
}

func TestLifecycleFailureQueueMissingCacheRetainsCompletion(t *testing.T) {
	for _, phase := range []string{"prepare", "adopt", "ready"} {
		t.Run(phase, func(t *testing.T) {
			q, store, jobs := newLifecycleQueueFixture()
			q.rows = func() []couchcore.ActionableThreadSummary { return nil }
			cause := errors.New(phase + " failed")
			var result error
			var wait func(context.Context) error
			if phase == "ready" {
				wait = func(context.Context) error { return cause }
			}
			done := make(chan struct{})
			err := q.EnqueueForReference("remote\x00/project:1", "relaunch", "project:1", func(ctx context.Context) (couchcore.OperationCall, error) {
				if couchcore.OperationAttemptFromContext(ctx) == nil {
					t.Fatal("missing cache disabled terminal ownership")
				}
				call := couchcore.OperationCall{Name: "park", Context: ctx, Args: map[string]string{"repo-scope": "scope", "tag": "dispatched"}, Implicit: true}
				if phase == "prepare" {
					return call, cause
				}
				_, err := couchcore.DispatchOperation(couchcore.OperationExecutors{Attempt: q.couch.OperationAttemptForCall, LiveOwner: func(couchcore.OperationCall) (any, error) { return nil, nil }}, call)
				return call, err
			}, func() {}, func(_ any, err error) { result = err; close(done) }, &wait)
			if err != nil {
				t.Fatal(err)
			}
			call, err := (*jobs)[0].prepare(t.Context())
			if phase == "adopt" {
				err = cause
			}
			if len(store.Writes) != 0 {
				t.Fatal("dispatcher published before final completion")
			}
			(*jobs)[0].finish(call, err)
			<-done
			if !errors.Is(result, cause) || couchcore.FailureNonce(result) == "" || len(store.Writes) != 1 || store.Writes[0].Target != "project:1" || store.Writes[0].Address.Tag != "dispatched" {
				t.Fatalf("missing completion evidence: %v %+v", result, store.Writes)
			}
		})
	}
}

func TestLifecycleFailureQueueHubMoveRetainsDispatchedTargets(t *testing.T) {
	q, store, jobs := newLifecycleQueueFixture()
	cause := errors.New("hub disconnected")
	if err := q.Queue("hub-move", "move-slot", func(context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "move-slot", FailureTarget: &couchcore.LifecycleFailureTarget{Target: "project:1,project:2"}}, cause
	}, func() {}, func(any, error) {}); err != nil {
		t.Fatal(err)
	}
	_, err := (*jobs)[0].prepare(t.Context())
	(*jobs)[0].finish(nil, err)
	if len(store.Writes) != 1 || store.Writes[0].Target != "project:1,project:2" || store.Writes[0].Operation != "move-slot" {
		t.Fatalf("hub move lost target evidence: %+v", store.Writes)
	}
}

func TestLifecycleFailureQueueRefusalBeforeEnqueueDoesNotPublish(t *testing.T) {
	q, store, _ := newLifecycleQueueFixture()
	cause := errors.New("queue is full")
	q.enqueue = func(string, string, func(context.Context) (couchcore.OperationCall, error), func(), func(any, error)) error {
		return cause
	}
	err := q.Queue("remote\x00/project:1", "relaunch", func(context.Context) (couchcore.OperationCall, error) {
		t.Fatal("refused request ran")
		return couchcore.OperationCall{}, nil
	}, func() { t.Fatal("refused request started") }, func(any, error) { t.Fatal("refused request completed") })
	if !errors.Is(err, cause) || len(store.Writes) != 0 {
		t.Fatalf("refusal changed or wrote diagnostics: %v %+v", err, store.Writes)
	}
}

func TestLifecycleFailureQueueReadinessUsesOwnerLifetime(t *testing.T) {
	q, _, jobs := newLifecycleQueueFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.lifetime = ctx
	entered := make(chan struct{})
	done := make(chan error, 1)
	release := make(chan struct{})
	joined := make(chan struct{})
	wait := func(ctx context.Context) error {
		close(entered)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return context.Canceled
		}
	}
	if err := q.Enqueue("remote\x00/project:1", "relaunch", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "relaunch", Context: ctx}, nil
	}, func() {}, func(_ any, err error) { done <- err }, &wait); err != nil {
		t.Fatal(err)
	}
	(*jobs)[0].prepare(ctx)
	go func() { defer close(joined); (*jobs)[0].finish(nil, nil) }()
	t.Cleanup(func() { close(release); <-joined })
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("readiness worker never entered")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("readiness lost owner cancellation: %v", err)
		}
	case <-testwait.After(t):
		t.Fatal("readiness ignored owner cancellation")
	}
}
