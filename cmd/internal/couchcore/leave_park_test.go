package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Every combination of evidence, at every drive count, against the invariants
// the retry policy promises (#827).
func TestDecideLeaveParkRetryInvariants(t *testing.T) {
	for bits := 0; bits < 1<<6; bits++ {
		ev := leaveParkEvidence{
			Verified: bits&1 != 0, Failed: bits&2 != 0, OpenTransaction: bits&4 != 0,
			RolledBack: bits&8 != 0, SameSource: bits&16 != 0, BudgetLeft: bits&32 != 0,
		}
		for drives := 1; drives <= leaveParkMaxDrives+1; drives++ {
			got := decideLeaveParkRetry(ev, drives)
			retry := got == leaveParkRetryFresh || got == leaveParkRetryRecover
			switch {
			case ev.Verified && got != leaveParkSettled:
				t.Errorf("%+v drives=%d: verified success not absorbing: %d", ev, drives, got)
			case !ev.Verified && got == leaveParkSettled:
				t.Errorf("%+v drives=%d: settled without verified history", ev, drives)
			case retry && (drives >= leaveParkMaxDrives || !ev.BudgetLeft):
				t.Errorf("%+v drives=%d: retried past the cap or budget", ev, drives)
			case retry && !ev.Failed:
				t.Errorf("%+v drives=%d: retried a drive that returned without error", ev, drives)
			case got == leaveParkRetryFresh && (ev.OpenTransaction || !ev.RolledBack || !ev.SameSource):
				t.Errorf("%+v drives=%d: fresh park without proven rollback of the same source", ev, drives)
			case got == leaveParkRetryRecover && !ev.OpenTransaction:
				t.Errorf("%+v drives=%d: recover without an open transaction", ev, drives)
			case got == leaveParkExhausted && !(drives >= leaveParkMaxDrives || !ev.BudgetLeft):
				t.Errorf("%+v drives=%d: exhausted with drives and budget left", ev, drives)
			}
		}
	}
}

// AwaitJoined holds its caller until the exact admitted work has finished and
// freed its address, even after cancellation; ordinary Await still returns on
// cancellation (#827 PQ-1).
func TestParkFutureAwaitJoinedOutlivesCancellation(t *testing.T) {
	worker := newParkWorker(1)
	address := ThreadAddress{Tag: "couch-0000000000000001"}
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	future, err := worker.Submit(ctx, address, "park-joined", func(context.Context) (ParkResult, error) {
		<-release
		return ParkResult{RequestedCommitDuration: time.Second}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := future.Await(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ordinary Await did not return on cancellation: %v", err)
	}
	joined := make(chan ParkResult, 1)
	go func() {
		result, _ := future.AwaitJoined(ctx)
		joined <- result
	}()
	select {
	case <-joined:
		t.Fatal("AwaitJoined returned while the admitted work was still running")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case result := <-joined:
		if result.RequestedCommitDuration != time.Second {
			t.Fatalf("AwaitJoined lost the work's own result: %+v", result)
		}
	case <-testwait.After(t):
		t.Fatal("AwaitJoined never returned after the work finished")
	}
	worker.mu.Lock()
	_, active := worker.active[address]
	worker.mu.Unlock()
	if active {
		t.Fatal("AwaitJoined returned before the worker freed the address")
	}
}

// fleetLifecycle is the concurrency-safe Pair lifecycle for a whole-fleet
// park: each request completes when its own session's quit is triggered.
type fleetLifecycle struct {
	model *pairlifecycletest.Fake
	now   time.Time

	mu          sync.Mutex
	bySession   map[string]pairlifecycle.QuitRequest
	completions map[string]pairlifecycle.QuitCompletion
	// failPublish returns, per session, how many more publishes fail
	// definitely-unpublished.
	failPublish map[string]int
	published   int
	triggers    map[string]int
	inFlight    int
	peak        int
}

func (f *fleetLifecycle) PublishRequest(_ context.Context, _ artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) error {
	f.mu.Lock()
	f.inFlight++
	f.peak = max(f.peak, f.inFlight)
	defer func() { f.mu.Lock(); f.inFlight--; f.mu.Unlock() }()
	if f.failPublish[request.Session] > 0 {
		f.failPublish[request.Session]--
		f.mu.Unlock()
		return errors.New("definite unpublished request failure")
	}
	f.bySession[request.Session] = request
	f.published++
	f.mu.Unlock()
	return f.model.CommitRequest(request)
}

func (f *fleetLifecycle) ObserveCompletion(_ context.Context, _ artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) (pairlifecycle.QuitCompletion, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	completion, ok := f.completions[completionSlot(request)]
	return completion, ok, nil
}

// completionSlot keys a completion by its exact transaction attempt.
func completionSlot(request pairlifecycle.QuitRequest) string {
	return fmt.Sprintf("%+v/%d", request.Identity, request.Attempt)
}

func (f *fleetLifecycle) CleanupAttempt(context.Context, artifactpath.LifecyclePaths, pairlifecycle.QuitRequest) error {
	return nil
}

func (f *fleetLifecycle) trigger(session string) error {
	f.mu.Lock()
	request, ok := f.bySession[session]
	f.triggers[session]++
	f.mu.Unlock()
	if !ok {
		return fmt.Errorf("no request for session %s", session)
	}
	if err := f.model.DeliverTrigger(request); err != nil {
		return err
	}
	if err := f.model.CommitCompletion(request, pairlifecycle.CleanupResult{Outcome: pairlifecycle.CompletionSuccess, CompletedAt: f.now}); err != nil {
		return err
	}
	f.mu.Lock()
	f.completions[completionSlot(request)] = successCompletion(request, f.now)
	f.mu.Unlock()
	return nil
}

// alternatingClock reads alternate between two instants a modeled commit
// apart, so no interleaving of concurrent BeginParks measures more than one
// step: every commit is slow (2-3s) yet under the threshold.
type alternatingClock struct {
	mu    sync.Mutex
	now   time.Time
	step  time.Duration
	reads int
}

func (c *alternatingClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	if c.reads%2 == 0 {
		return c.now.Add(c.step)
	}
	return c.now
}

type leaveParkFleet struct {
	couch     *Couch
	lifecycle *fleetLifecycle
	artifacts *FakeThreadArtifactCollisionChecker
	threads   []ThreadRecord
	sessions  []string
	nonces    atomic.Int64
	drives    sync.Map // ThreadAddress -> *[]leaveParkAction
	driveN    atomic.Int64
}

type fleetCompletionCancelKey struct{}

// newLeaveParkFleet is n live threads whose parks commit in a modeled 2.5s
// and complete when their own quit is triggered.
func newLeaveParkFleet(t *testing.T, n int) *leaveParkFleet {
	t.Helper()
	// Coordinated like production (couch.go): every in-process store user
	// serializes on the coordinator, so concurrent parks never race the
	// nested nonblocking store lock.
	store, _ := retentionStore(t)
	ns := store.namespace
	now := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	model := pairlifecycletest.New(now)
	f := &leaveParkFleet{
		artifacts: NewFakeThreadArtifactCollisionChecker(),
		lifecycle: &fleetLifecycle{
			model: model, now: now, bySession: map[string]pairlifecycle.QuitRequest{},
			completions: map[string]pairlifecycle.QuitCompletion{}, failPublish: map[string]int{}, triggers: map[string]int{},
		},
	}
	for i := 0; i < n; i++ {
		record := validThreadRecord(t)
		record.Address.Tag = ThreadTag(fmt.Sprintf("couch-%016x", 0xb000+i))
		record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
		record.Reservation = false
		profile := LaunchProfile{Agent: []string{"claude", "codex"}[i%2]}
		record.Incarnations = []ThreadIncarnation{{PID: 200 + i, Identity: fmt.Sprintf("pair-%d", i), State: IncarnationLive, LaunchProfile: &profile}}
		record.LatestLaunchProfile = &profile
		created, err := store.CreateThread(record)
		if err != nil {
			t.Fatal(err)
		}
		session := fmt.Sprintf("pair-session-%d", i)
		model.SetSession(session, true)
		f.artifacts.SetPairSession(created.Address, session, true)
		f.threads = append(f.threads, created)
		f.sessions = append(f.sessions, session)
	}
	f.artifacts.TriggerQuitHook = func(session string, _ launcher.QuitIntent) error { return f.lifecycle.trigger(session) }
	controller := &PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: f.lifecycle, Sessions: f.artifacts,
		Proc: NewFakeProcOps(), Clock: &alternatingClock{now: now, step: 2500 * time.Millisecond},
		Nonce: func() (string, error) {
			return fmt.Sprintf("park-leave-%016x", f.nonces.Add(1)), nil
		},
		CompletionTimeout: time.Second, PollInterval: time.Millisecond,
	}
	// #852: this fixture measures retry work, not host IO latency. Completion
	// is synchronous with trigger; only an actual missing-proof poll spends
	// its budget. WithCancel retains caller cancellation through both seams.
	controller.completionDeadline = func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		return context.WithValue(ctx, fleetCompletionCancelKey{}, cancel), cancel
	}
	controller.Wait = func(ctx context.Context, _ time.Duration) error {
		ctx.Value(fleetCompletionCancelKey{}).(context.CancelFunc)()
		return ctx.Err()
	}
	f.couch = &Couch{
		Threads: store, Proc: NewFakeProcOps(), Artifacts: f.artifacts,
		PairLifecycle: controller, sleep: func(time.Duration) {},
	}
	f.couch.leaveParkDeadline = func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
		return context.WithCancel(parent)
	}
	f.couch.onLeaveParkDrive = func(address ThreadAddress, action leaveParkAction) {
		f.driveN.Add(1)
		actions, _ := f.drives.LoadOrStore(address, &[]leaveParkAction{})
		f.lifecycle.mu.Lock()
		*actions.(*[]leaveParkAction) = append(*actions.(*[]leaveParkAction), action)
		f.lifecycle.mu.Unlock()
	}
	return f
}

func (f *leaveParkFleet) drivesOf(address ThreadAddress) []leaveParkAction {
	actions, ok := f.drives.Load(address)
	if !ok {
		return nil
	}
	f.lifecycle.mu.Lock()
	defer f.lifecycle.mu.Unlock()
	return append([]leaveParkAction(nil), *actions.(*[]leaveParkAction)...)
}

// One confirmed full quit over N and 2N threads with modeled 2.5s commits
// parks every thread; a proven-unpublished failure is retried once, fresh,
// inside the same quit. Work grows linearly: at most one extra drive per
// failed thread, one quit per thread, fan-out unchanged.
func TestLeaveParkRetriesProvenRollbackWithinOneQuit(t *testing.T) {
	for _, n := range []int{15, 30} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			withParallelism(t, 4)
			f := newLeaveParkFleet(t, n)
			failed := 0
			for i := 0; i < n; i += 3 {
				f.lifecycle.failPublish[f.sessions[i]] = 1
				failed++
			}
			result, err := f.couch.Leave(context.Background(), LeavePark)
			if err != nil {
				t.Fatalf("Leave: %v", err)
			}
			if len(result.Parked) != n || len(result.Unresolved) != 0 {
				t.Fatalf("parked=%d unresolved=%+v, want all %d parked", len(result.Parked), result.Unresolved, n)
			}
			for i, thread := range f.threads {
				if result.Parked[i] != thread.Address {
					t.Fatalf("report order %v broke snapshot order", result.Parked)
				}
				drives := f.drivesOf(thread.Address)
				want := []leaveParkAction{leaveParkFresh}
				if i%3 == 0 {
					want = []leaveParkAction{leaveParkFresh, leaveParkFresh}
				}
				if fmt.Sprint(drives) != fmt.Sprint(want) {
					t.Fatalf("%s drives=%v want %v", thread.Address.Tag, drives, want)
				}
				if got := f.lifecycle.triggers[f.sessions[i]]; got != 1 {
					t.Fatalf("%s quit triggers=%d, want 1", thread.Address.Tag, got)
				}
				after, _ := f.couch.Threads.GetThread(thread.Address)
				if after.VerifiedPark == nil || after.Park != nil {
					t.Fatalf("%s not verified parked: %+v", thread.Address.Tag, after)
				}
			}
			if got := int(f.driveN.Load()); got != n+failed || got > leaveParkMaxDrives*n {
				t.Fatalf("drives=%d want %d (<= %d)", got, n+failed, leaveParkMaxDrives*n)
			}
			if f.lifecycle.published != n || f.lifecycle.peak > LifecycleParallelism {
				t.Fatalf("published=%d peak=%d, want %d and <= %d", f.lifecycle.published, f.lifecycle.peak, n, LifecycleParallelism)
			}
		})
	}
}

// A modeled completion budget must not expire while the real store is busy.
// An immediately elapsed wall-clock duration makes the old fixture fail
// without sleeping or relying on machine load (#852).
func TestLeaveParkFixtureCompletionBudgetIsDrivenByProof(t *testing.T) {
	withParallelism(t, 1)
	f := newLeaveParkFleet(t, 1)
	//lint:allow test-shrunk-deadline — this rejects wall-clock use by the fixture's controlled deadline, not a production timeout
	f.couch.PairLifecycle.CompletionTimeout = time.Nanosecond
	f.lifecycle.failPublish[f.sessions[0]] = 1
	result, err := f.couch.Leave(context.Background(), LeavePark)
	if err != nil || len(result.Parked) != 1 {
		t.Fatalf("immediate matching proof failed: %+v %v", result, err)
	}
	want := []leaveParkAction{leaveParkFresh, leaveParkFresh}
	if got := fmt.Sprint(f.drivesOf(f.threads[0].Address)); got != fmt.Sprint(want) {
		t.Fatalf("drives=%s, want two fresh drives", got)
	}
}

// Controlled time must still end a wait when the fake withholds its proof.
func TestLeaveParkFixtureMissingProofSpendsCompletionBudget(t *testing.T) {
	withParallelism(t, 1)
	f := newLeaveParkFleet(t, 1)
	f.artifacts.TriggerQuitHook = func(session string, _ launcher.QuitIntent) error {
		if err := f.lifecycle.trigger(session); err != nil {
			return err
		}
		f.lifecycle.mu.Lock()
		clear(f.lifecycle.completions)
		f.lifecycle.mu.Unlock()
		return nil
	}
	result, err := f.couch.Leave(context.Background(), LeavePark)
	if !errors.Is(err, context.Canceled) || len(result.Parked) != 0 || len(result.Unresolved) != 1 {
		t.Fatalf("missing proof did not spend controlled budget: %+v %v", result, err)
	}
	if got := len(f.drivesOf(f.threads[0].Address)); got != leaveParkMaxDrives {
		t.Fatalf("missing proof drives=%d, want cap %d", got, leaveParkMaxDrives)
	}
	after, err := f.couch.Threads.GetThread(f.threads[0].Address)
	if err != nil || after.VerifiedPark != nil || after.Park == nil {
		t.Fatalf("missing proof fabricated success or lost transaction: %+v %v", after, err)
	}
}

func TestLeaveParkFixtureHonorsCallerCancellation(t *testing.T) {
	f := newLeaveParkFleet(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := f.couch.Leave(ctx, LeavePark)
	if !errors.Is(err, context.Canceled) || len(result.Parked) != 0 || len(result.Unresolved) != 1 {
		t.Fatalf("caller cancellation lost: %+v %v", result, err)
	}
	if f.driveN.Load() != 0 || f.lifecycle.published != 0 {
		t.Fatal("canceled caller started a park")
	}
}

// A target that never publishes stops at the drive cap and is named; its
// siblings still park, and the quit is reported unfinished, not complete.
func TestLeaveParkExhaustedTargetIsNamedOnce(t *testing.T) {
	withParallelism(t, 2)
	f := newLeaveParkFleet(t, 4)
	stuck := f.threads[1]
	f.lifecycle.failPublish[f.sessions[1]] = 100
	result, err := f.couch.Leave(context.Background(), LeavePark)
	if err == nil {
		t.Fatal("unresolved full quit reported success")
	}
	if len(result.Parked) != 3 || len(result.Unresolved) != 1 || result.Unresolved[0].Address != stuck.Address {
		t.Fatalf("result=%+v", result)
	}
	if got := len(f.drivesOf(stuck.Address)); got != leaveParkMaxDrives {
		t.Fatalf("stuck drives=%d want %d", got, leaveParkMaxDrives)
	}
	if f.lifecycle.triggers[f.sessions[1]] != 0 {
		t.Fatal("an unpublished park sent a quit")
	}
	var incomplete *LeaveParkIncompleteError
	if !errors.As(err, &incomplete) || err.Error() != FormatLeaveParkSummary(result) {
		t.Fatalf("leave error is not the complete summary: %v", err)
	}
	summary := FormatLeaveParkSummary(result)
	if u := result.Unresolved[0]; strings.Contains(u.Reason, u.Next) || strings.Count(summary, u.Next) != 1 {
		t.Fatalf("next action repeated in the reason:\n%s", summary)
	}
	if strings.Count(summary, "\n- "+string(stuck.Address.Tag)+":") != 1 || !strings.Contains(summary, "next:") || !strings.Contains(summary, "incomplete") {
		t.Fatalf("summary does not name the stuck thread once with its next action:\n%s", summary)
	}
}

// A rollback that cannot be written leaves the transaction open: the retry
// drives that same transaction through Recover and never mints another nonce.
func TestLeaveParkOpenTransactionRetriesThroughRecover(t *testing.T) {
	withParallelism(t, 1)
	f := newLeaveParkFleet(t, 1)
	thread := f.threads[0]
	f.lifecycle.failPublish[f.sessions[0]] = 1
	refuse := errors.New("rollback write refused")
	f.couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
		if old.Park != nil && next.Park == nil && next.VerifiedPark == nil {
			return refuse
		}
		return nil
	}
	result, err := f.couch.Leave(context.Background(), LeavePark)
	drives := f.drivesOf(thread.Address)
	if len(drives) < 2 || drives[0] != leaveParkFresh || drives[1] != leaveParkRecover {
		t.Fatalf("drives=%v, want fresh then recover", drives)
	}
	for _, action := range drives[1:] {
		if action != leaveParkRecover {
			t.Fatalf("drives=%v: fresh park over an open transaction", drives)
		}
	}
	if got := f.nonces.Load(); got != 1 {
		t.Fatalf("minted %d nonces, want the one transaction kept", got)
	}
	if err != nil || len(result.Parked) != 1 {
		t.Fatalf("recover did not finish the open transaction: %+v %v", result, err)
	}
}

// When the budget ends mid-fan-out, every thread is accounted for exactly once:
// parked, or unresolved with its next action (including never-started ones).
func TestLeaveParkBudgetEndNamesEveryUnfinishedThread(t *testing.T) {
	withParallelism(t, 1)
	f := newLeaveParkFleet(t, 6)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.couch.leaveParkDeadline = func(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		if d != LeaveParkBudget {
			t.Errorf("budget=%s want %s", d, LeaveParkBudget)
		}
		return ctx, cancel
	}
	f.lifecycle.failPublish[f.sessions[1]] = 1
	next := f.couch.onLeaveParkDrive
	f.couch.onLeaveParkDrive = func(address ThreadAddress, action leaveParkAction) {
		next(address, action)
		if address == f.threads[1].Address {
			cancel() // the budget ends during the second thread's first drive
		}
	}
	result, err := f.couch.Leave(context.Background(), LeavePark)
	if err == nil {
		t.Fatal("budget end reported a complete quit")
	}
	seen := map[ThreadAddress]int{}
	for _, address := range result.Parked {
		seen[address]++
	}
	for _, u := range result.Unresolved {
		seen[u.Address]++
		if u.Next == "" || u.Reason == "" {
			t.Fatalf("unresolved without reason/next: %+v", u)
		}
	}
	for _, thread := range f.threads {
		if seen[thread.Address] != 1 {
			t.Fatalf("%s accounted %d times: %+v", thread.Address.Tag, seen[thread.Address], result)
		}
	}
	if len(result.Parked) != 1 || result.Parked[0] != f.threads[0].Address {
		t.Fatalf("parked=%v, want only the first", result.Parked)
	}
	if got := len(f.drivesOf(f.threads[1].Address)); got != 1 {
		t.Fatalf("retried after the budget ended: %d drives", got)
	}
}

// An unreadable record may hide a live agent: the full quit names it rather
// than exiting as if it were done.
func TestLeaveParkNamesUnreadableRecords(t *testing.T) {
	f := newLeaveParkFleet(t, 2)
	bad := f.threads[1]
	if err := os.WriteFile(f.couch.Threads.recordPath(bad.Address), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := f.couch.Leave(context.Background(), LeavePark)
	if err == nil {
		t.Fatal("unreadable record allowed a complete quit")
	}
	if len(result.Parked) != 1 || len(result.Unresolved) != 1 || result.Unresolved[0].Address != bad.Address {
		t.Fatalf("result=%+v", result)
	}
}

// A thread whose incarnation Couch cannot vouch for is never parked, so the
// full quit cannot claim to be complete: it is named and Couch stays open
// (#827 BR-1; the plan's state table).
func TestLeaveParkSkippedThreadKeepsCouchOpen(t *testing.T) {
	f := newLeaveParkFleet(t, 2)
	unknown := f.threads[1]
	if _, err := f.couch.Threads.MarkIncarnationUnknown(unknown.Address, ProcessIdentity{PID: unknown.Incarnations[0].PID, Identity: unknown.Incarnations[0].Identity}); err != nil {
		t.Fatal(err)
	}
	result, err := f.couch.Leave(context.Background(), LeavePark)
	var incomplete *LeaveParkIncompleteError
	if !errors.As(err, &incomplete) {
		t.Fatalf("skipped thread let the full quit succeed: %+v %v", result, err)
	}
	if len(result.Parked) != 1 || len(result.Unresolved) != 1 || result.Unresolved[0].Address != unknown.Address || result.Unresolved[0].Next == "" {
		t.Fatalf("result=%+v", result)
	}
	if got := len(f.drivesOf(unknown.Address)); got != 0 {
		t.Fatalf("drove a park on an unvouched incarnation: %d drives", got)
	}
}
