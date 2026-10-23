package couchcore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestParkWorkerBoundsAndCoalesces(t *testing.T) {
	t.Parallel()

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
	// At capacity, other work WAITS for a free unit (pair#205): capacity is a
	// throughput bound shared by every park submitter, never a refusal.
	other := ThreadAddress{RepoScope: address.RepoScope, Tag: "couch-fedcba9876543210"}
	otherRan := make(chan struct{})
	otherWaiting := make(chan struct{}, 1)
	worker.onWait = func() { otherWaiting <- struct{}{} }
	otherSubmitted := make(chan error, 1)
	go func() {
		future, err := worker.Submit(context.Background(), other, "other", func(context.Context) (ParkResult, error) {
			close(otherRan)
			return ParkResult{}, nil
		})
		if err == nil {
			_, err = future.Await(context.Background())
		}
		otherSubmitted <- err
	}()
	<-otherWaiting // blocked on capacity: it waits rather than being refused
	select {
	case <-otherRan:
		t.Fatal("work beyond capacity ran while the worker was full")
	default:
	}
	close(release)
	if err := <-otherSubmitted; err != nil {
		t.Fatalf("waiting work = %v, want it admitted once capacity freed", err)
	}
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

func TestParkCoordinatorCoalescesExplicitRecoveryAndRetry(t *testing.T) {
	t.Parallel()

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
	joined := make(chan struct{})
	worker := newParkWorker(LifecycleParallelism)
	worker.onJoin = func() { close(joined) }
	controller := PairLifecycleController{
		worker:  worker,
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
		_, err := couch.Park(context.Background(), thread.Address, "recover")
		recoveryDone <- err
	}()
	select {
	case <-triggerEntered:
	case <-testwait.After(t):
		t.Fatal("explicit recovery did not reach the exact trigger")
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
	case <-joined:
	case <-testwait.After(t):
		t.Fatal("interactive retry did not join the admitted recovery")
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

func TestParkWorkerBeginDeadlineHasZeroExternalEffectsAtThreshold(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name          string
		elapsed       time.Duration
		wantPhase     ParkPhase
		wantLifecycle int
		wantEffect    int
	}{
		{name: "just under threshold commits", elapsed: parkCommitDeadline - time.Millisecond, wantPhase: ParkAwaitingCompletion, wantLifecycle: 3, wantEffect: 1},
		{name: "exactly threshold refuses", elapsed: parkCommitDeadline, wantPhase: ParkRequested, wantEffect: 0},
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
			if getErr != nil || len(persisted.Incarnations) != 1 {
				t.Fatalf("deadline occupancy = %+v, %v", persisted, getErr)
			}
			if test.elapsed >= parkCommitDeadline {
				if persisted.Park != nil || len(persisted.ParkHistory) != 1 || !persisted.ParkHistory[0].Tombstoned || !result.SoftTargetMissed {
					t.Fatalf("expired unpublished begin not rolled back: %+v", persisted)
				}
				return
			}
			if persisted.Park == nil || !result.SoftTargetMissed || persisted.Park.Phase != test.wantPhase {
				t.Fatalf("deadline result = %+v, thread=%+v", result, persisted)
			}
		})
	}
}

// A cancelled park caller returns at once (TestCanceledParkAwaitStill...), and
// its thread stays guarded while the work runs on: the record's open park
// transaction and the worker's active entry refuse a second lifecycle
// operation, whether or not the caller's gate hold has been released
// (pair#205 D4, revised during M1).
func TestACancelledParkStillRefusesOtherLifecycleOperations(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	c := env.Couch
	reached, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	env.Lifecycle.onPublish = func(pairlifecycle.QuitRequest) { close(reached); <-release }
	ctx, cancel := context.WithCancel(context.Background())
	returned := make(chan error, 1)
	go func() { _, err := c.PairLifecycle.Park(ctx, source.Address); returned <- err }()
	<-reached
	worker := c.PairLifecycle.worker
	worker.mu.Lock()
	future := worker.active[source.Address].future
	worker.mu.Unlock()
	defer func() {
		// Let the still-running park settle before the test's temp dirs go.
		unblock()
		waitCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_, _ = future.Await(waitCtx)
	}()
	cancel()
	if err := <-returned; !errors.Is(err, context.Canceled) {
		t.Fatalf("park = %v, want prompt cancellation", err)
	}
	// Detaching the durable lifecycle remains forbidden while its park is open.
	if _, err := c.Detach(context.Background(), source.Address); err == nil || !strings.Contains(err.Error(), "open park transaction") {
		t.Fatalf("detach bypassed active park: %v", err)
	}
	before, err := c.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	actor, helper, err := c.ResumeContext(context.Background(), source.Address)
	if err != nil || helper == nil || actor.Shape != StartWarmReattach {
		t.Fatalf("listed Continue refused during cancelled park: %v", err)
	}
	t.Cleanup(func() { env.Runner.SetExited(helper.ID(), 0) })
	assertWarmRecordUnchanged(t, env.testEnv, before)
	// A second transaction carries its own nonce (production nonces are random;
	// this fixture's is fixed, which would coalesce onto the running park).
	c.PairLifecycle.Nonce = func() (string, error) { return "park-second-transaction", nil }
	if _, err := c.PairLifecycle.ParkExpected(context.Background(), source.Address, 0); err == nil || !strings.Contains(err.Error(), "another park transaction") {
		t.Fatalf("second park transaction = %v, want the worker's another-park-transaction refusal", err)
	}
}

// parkWorker frees its address before it signals done, so a caller that has
// seen its park finish can submit the next park on that address at once.
func TestParkWorkerFreesTheAddressBeforeDone(t *testing.T) {
	t.Parallel()

	worker := newParkWorker(1)
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	done := func(context.Context) (ParkResult, error) { return ParkResult{}, nil }
	for i := 0; i < 1000; i++ {
		future, err := worker.Submit(context.Background(), address, fmt.Sprintf("nonce-%d", i), done)
		if err != nil {
			t.Fatalf("iteration %d: submit after a finished park refused: %v", i, err)
		}
		if _, err := future.Await(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

// A capacity wait honours its context (pair#205).
func TestParkWorkerCapacityWaitHonoursContext(t *testing.T) {
	t.Parallel()

	worker := newParkWorker(1)
	release := make(chan struct{})
	defer close(release)
	first := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	if _, err := worker.Submit(context.Background(), first, "first", func(context.Context) (ParkResult, error) {
		<-release
		return ParkResult{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	waiting := make(chan struct{}, 1)
	worker.onWait = func() { waiting <- struct{}{} }
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := worker.Submit(ctx, ThreadAddress{RepoScope: first.RepoScope, Tag: "couch-fedcba9876543210"}, "second",
			func(context.Context) (ParkResult, error) { return ParkResult{}, nil })
		done <- err
	}()
	<-waiting // the submitter is blocked on capacity, not merely scheduled
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled capacity wait = %v, want context.Canceled", err)
	}
}
