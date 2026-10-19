package couchcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
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
	first.finish(nil, nil)
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
	if len(store.Writes) != 1 || store.Writes[0].SupersededAt == nil {
		t.Fatalf("late older failure obscured newer success: %+v", store.Writes)
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
