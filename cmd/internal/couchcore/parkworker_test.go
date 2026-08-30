package couchcore

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestParkWorkerBoundsAndCoalesces(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	calls := 0
	worker := newParkWorker(1)
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	work := func(context.Context) (ParkResult, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		close(started)
		<-release
		return ParkResult{}, nil
	}
	first, err := worker.Submit(context.Background(), address, "park-nonce", work)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	duplicate, err := worker.Submit(context.Background(), address, "park-nonce", work)
	if err != nil || duplicate != first {
		t.Fatalf("duplicate = %p, %v; first=%p", duplicate, err, first)
	}
	other := ThreadAddress{RepoScope: address.RepoScope, Tag: "couch-fedcba9876543210"}
	if _, err := worker.Submit(context.Background(), other, "other", func(context.Context) (ParkResult, error) {
		t.Fatal("overloaded work executed")
		return ParkResult{}, nil
	}); !errors.Is(err, ErrParkWorkerOverloaded) {
		t.Fatalf("overload err = %v", err)
	}
	close(release)
	if _, err := first.Await(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := duplicate.Await(context.Background()); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("coalesced calls = %d", calls)
	}
}

func TestParkCoordinatorCoalescesStartupRecoveryAndInteractiveRetry(t *testing.T) {
	store, _, thread := createControllerThread(t)
	identity := ParkIdentity{
		Nonce: "park-overlap", Address: thread.Address, PID: 42, ProcessIdentity: "pair-helper",
	}
	thread, err := store.BeginPark(thread.Address, thread.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(300, 0).UTC()
	model := pairlifecycletest.New(now)
	model.SetSession("pair-exact", true)
	triggerEntered := make(chan struct{})
	releaseTrigger := make(chan struct{})
	lifecycle := &fakeControllerLifecycle{model: model}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	artifacts.TriggerQuitHook = func(_ string, intent launcher.QuitIntent) error {
		request := lifecycle.lastRequest
		if intent.Request == nil || intent.Request.Nonce != request.Identity.Nonce {
			return errors.New("trigger did not carry the active request")
		}
		completion := successCompletion(request, now)
		lifecycle.completion = &completion
		close(triggerEntered)
		<-releaseTrigger
		return nil
	}
	controller := PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
		Proc: NewFakeProcOps(), Clock: FixedClock{T: now},
		Nonce: func() (string, error) { return "unused", nil },
	}
	couch := &Couch{Threads: store, PairLifecycle: &controller}

	type outcome struct {
		result ParkResult
		err    error
	}
	recoveryDone := make(chan error, 1)
	go func() {
		recoveryDone <- couch.RecoverActiveParks(context.Background())
	}()
	select {
	case <-triggerEntered:
	case <-time.After(time.Second):
		t.Fatal("startup recovery did not reach the exact trigger")
	}
	retryDone := make(chan outcome, 1)
	go func() {
		value, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(couch)}, OperationCall{
			Name: "park", Implicit: true, Args: map[string]string{
				"repo-scope": thread.Address.RepoScope, "tag": string(thread.Address.Tag), "mode": "retry",
			},
		})
		result, _ := value.(ParkResult)
		retryDone <- outcome{result: result, err: err}
	}()
	select {
	case got := <-retryDone:
		t.Fatalf("interactive retry did not share the in-flight recovery: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseTrigger)
	recoveryErr := <-recoveryDone
	retry := <-retryDone
	if recoveryErr != nil || retry.err != nil {
		t.Fatalf("overlap errors: recovery=%v retry=%v", recoveryErr, retry.err)
	}
	final, err := store.GetThread(thread.Address)
	if err != nil || !reflect.DeepEqual(final, retry.result.Thread) {
		t.Fatalf("overlap did not share the committed result:\nfinal=%+v, %v\nretry=%+v", final, err, retry.result)
	}
	if got := len(artifacts.TriggeredQuits()); got != 1 {
		t.Fatalf("overlap trigger count = %d, want 1", got)
	}
}

type sequenceClock struct {
	mu    sync.Mutex
	times []time.Time
	last  time.Time
}

func (c *sequenceClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.times) == 0 {
		return c.last
	}
	c.last = c.times[0]
	c.times = c.times[1:]
	return c.last
}

func TestParkWorkerBeginDeadlineHasZeroExternalEffectsAtOneSecond(t *testing.T) {
	for _, test := range []struct {
		name          string
		elapsed       time.Duration
		wantPhase     ParkPhase
		wantLifecycle int
		wantEffect    int
	}{
		{name: "999ms commits before deadline", elapsed: 999 * time.Millisecond, wantPhase: ParkAwaitingCompletion, wantLifecycle: 3, wantEffect: 1},
		{name: "exactly 1s refuses", elapsed: time.Second, wantPhase: ParkRequested, wantEffect: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _, thread := createControllerThread(t)
			start := time.Unix(100, 0).UTC()
			clock := &sequenceClock{times: []time.Time{start, start.Add(test.elapsed)}}
			lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(start)}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(thread.Address, "pair-exact", true)
			controller := PairLifecycleController{
				Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
				Proc:  NewFakeProcOps(),
				Clock: clock, Nonce: func() (string, error) { return "park-deadline", nil },
			}
			result, err := controller.Park(context.Background(), thread.Address)
			if err == nil {
				t.Fatal("Park returned success without a completion")
			}
			if got := len(artifacts.TriggeredQuits()); got != test.wantEffect {
				t.Fatalf("trigger effects = %d, want %d", got, test.wantEffect)
			}
			if got := len(lifecycle.trace); got != test.wantLifecycle {
				t.Fatalf("lifecycle effects = %v", lifecycle.trace)
			}
			persisted, getErr := store.GetThread(thread.Address)
			if getErr != nil || persisted.Park == nil || len(persisted.Incarnations) != 1 {
				t.Fatalf("deadline occupancy = %+v, %v", persisted, getErr)
			}
			if !result.SoftTargetMissed || persisted.Park.Phase != test.wantPhase {
				t.Fatalf("deadline result = %+v, thread=%+v", result, persisted)
			}
		})
	}
}
