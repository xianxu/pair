package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The existing stage-path environment already belongs to the private helper.
// A sibling marker enables a synchronized cleanup interruption without adding
// an operator-facing environment switch or changing normal conformance runs.
func awaitParkRecoveryCleanupGate(ctx context.Context, gate string) error {
	if gate == "" {
		return nil
	}
	if _, err := os.Stat(gate); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.WriteFile(gate+".entered", []byte("before scrollback preservation\n"), 0600); err != nil {
		return err
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(gate + ".release"); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Only native-conversation evidence is synthetic here. Terminal listing,
// publication/completion, journal replay and exact helper liveness are real.
// This fixture has no native agent and therefore cannot prove a native resume.
type realParkRecoveryArtifacts struct {
	ScopedThreadArtifactCollisionChecker
}

func (realParkRecoveryArtifacts) ResolveEstablished(context.Context, string, string, string) (NativeBindingResolution, error) {
	return NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: "conformance-retained-conversation"}, nil
}

func realParkRecoveryCouch(t *testing.T, d *realParkConformanceDriver) *testEnv {
	t.Helper()
	env := newTestEnv(t, d.current.WorkingPath)
	env.Couch.Threads = d.threads
	env.Couch.Proc = OSProcOps{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Artifacts = realParkRecoveryArtifacts{d.checker}
	env.Couch.PairLifecycle = &PairLifecycleController{
		Threads: d.threads, DataDir: d.checker.GlobalDataDir, Lifecycle: PairLifecycleStoreIO{Store: d.store},
		Sessions: d.checker, Proc: OSProcOps{}, Clock: FixedClock{T: d.completed},
		Nonce: func() (string, error) { return "unused", nil }, CompletionTimeout: 10 * time.Second, PollInterval: 10 * time.Millisecond,
	}
	// Cancellation can return before a worker settles. Join it before the driver
	// removes the helper's private storage and controlled terminal.
	t.Cleanup(func() {
		c := env.Couch.PairLifecycle
		c.workerOnce.Do(func() {
			if c.worker == nil {
				c.worker = newParkWorker(LifecycleParallelism)
			}
		})
		c.worker.mu.Lock()
		active, ok := c.worker.active[d.current.Address]
		c.worker.mu.Unlock()
		if ok {
			select {
			case <-active.future.done:
			case <-time.After(2 * time.Second):
				t.Error("park recovery worker did not settle before fixture cleanup")
			}
		}
	})
	return env
}

func prepareRealParkRecovery(t *testing.T, d *realParkConformanceDriver, r pairlifecycle.QuitRequest) {
	t.Helper()
	t.Cleanup(func() {
		if t.Failed() {
			stages, stageErr := os.ReadFile(d.stagePath)
			observed, storeErr := d.threads.PreviewThreadContext(context.Background(), d.current.Address)
			t.Logf("retained interruption diagnostics: stages=%q stageError=%v thread=%+v storeError=%v exactHelper=%s", stages, stageErr, observed, storeErr, observeExactProcess(OSProcOps{}, d.childID))
		}
	})
	if err := d.PrepareRequest(r); err != nil {
		t.Fatal(err)
	}
	if state := observeExactProcess(OSProcOps{}, d.childID); state != Live {
		t.Fatalf("publication affected helper: %s", state)
	}
	if err := d.Restart(); err != nil {
		t.Fatal(err)
	}
}

func assertRealParkRecovered(t *testing.T, env *testEnv, d *realParkConformanceDriver, verified bool) {
	t.Helper()
	final, err := env.Couch.Threads.GetThread(d.current.Address)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained park state: address=%+v revision=%d park=%+v verified=%+v history=%+v", final.Address, final.Revision, final.Park, final.VerifiedPark, final.ParkHistory)
	if final.Park != nil || len(final.Incarnations) != 0 || (final.VerifiedPark != nil) != verified || len(final.ParkHistory) != 1 {
		t.Fatalf("wrong retained outcome: %+v", final)
	}
	if final.ParkHistory[0].Tombstoned == verified {
		t.Fatalf("interrupted cleanup became verified success, or successful proof was discarded: %+v", final.ParkHistory)
	}
	if !reflect.DeepEqual(final.LatestLaunchProfile, d.current.LatestLaunchProfile) || !reflect.DeepEqual(final.SessionBinding, d.current.SessionBinding) {
		t.Fatal("recovery changed conversation configuration")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("recovery launched replacement: %v", env.Runner.Ops)
	}
}

func TestParkRecoveryLivePublicationInterruption(t *testing.T) {
	ctx, cancel := liveParkContext(t, 30*time.Second)
	defer cancel()
	d, r := newRealParkConformance(t, ctx, false)
	prepareRealParkRecovery(t, d, r)
	env := realParkRecoveryCouch(t, d)
	result, err := env.Couch.recoverPark(ctx, d.current.Address)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.ObserveChildDeath(r); err != nil {
		t.Fatal(err)
	}
	if result.OperatorOutcome != "resume-from-disk" || result.Diagnostic != "" {
		t.Fatalf("result=%+v", result)
	}
	assertRealParkRecovered(t, env, d, true)
}

func TestParkRecoveryLiveFinalizationInterruption(t *testing.T) {
	ctx, cancel := liveParkContext(t, 30*time.Second)
	defer cancel()
	d, r := newRealParkConformance(t, ctx, false)
	prepareRealParkRecovery(t, d, r)
	if err := d.CommitRequest(r); err != nil {
		t.Fatal(err)
	}
	if err := d.DeliverTrigger(r); err != nil {
		t.Fatal(err)
	}
	cleanup, _, err := d.CleanupAndPrepareCompletion(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.CommitCompletion(r, cleanup); err != nil {
		t.Fatal(err)
	}
	if err := d.ObserveChildDeath(r); err != nil {
		t.Fatal(err)
	}
	env := realParkRecoveryCouch(t, d)
	interrupted := errors.New("lost finalization journal reply")
	armed := false
	d.threads.hooks.AfterJournal = func() error {
		if armed {
			return interrupted
		}
		return nil
	}
	d.threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
		if old.Park != nil && next.Park == nil && next.VerifiedPark != nil {
			armed = true
		}
		return nil
	}
	if _, err := env.Couch.recoverPark(ctx, d.current.Address); !errors.Is(err, interrupted) {
		t.Fatalf("missing finalization interruption: %v", err)
	}
	// Reopen the actual store: journal after-image, rather than the lost reply,
	// determines whether a repeated recovery is already settled.
	reopened := NewThreadStore(d.threads.namespace)
	env.Couch.Threads = reopened
	env.Couch.PairLifecycle.Threads = reopened
	result, err := env.Couch.recoverPark(ctx, d.current.Address)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "park-already-settled" || result.OperatorOutcome != "resume-from-disk" {
		t.Fatalf("result=%+v", result)
	}
	assertRealParkRecovered(t, env, d, true)
}

func TestParkRecoveryLiveInterruptedScrollbackCleanup(t *testing.T) {
	ctx, cancel := liveParkContext(t, 30*time.Second)
	defer cancel()
	d, r := newRealParkConformance(t, ctx, false)
	gate := d.stagePath + ".pause-scrollback"
	if err := os.WriteFile(gate, []byte("private fixture gate\n"), 0600); err != nil {
		t.Fatal(err)
	}
	prepareRealParkRecovery(t, d, r)
	if err := d.CommitRequest(r); err != nil {
		t.Fatal(err)
	}
	if err := d.DeliverTrigger(r); err != nil {
		t.Fatal(err)
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(gate + ".entered"); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		select {
		case err := <-d.childWait:
			t.Fatalf("helper exited before synchronized scrollback gate: %v", err)
		case <-ctx.Done():
			t.Fatal("helper did not enter synchronized scrollback gate: ", ctx.Err())
		case <-ticker.C:
		}
	}
	env := realParkRecoveryCouch(t, d)
	if _, err := env.Couch.recoverPark(ctx, d.current.Address); err == nil {
		t.Fatal("retired helper still alive at cleanup gate")
	}
	if state := observeExactProcess(OSProcOps{}, d.childID); state != Live {
		t.Fatalf("live-owner refusal waited for helper exit: %s", state)
	}
	binding, err := observeRecordSession(ctx, d.checker, d.current)
	if err != nil || binding.Present {
		t.Fatalf("quiescence did not precede cleanup gate: %+v %v", binding, err)
	}
	t.Logf("interrupt owned helper at scrollback gate: exact=%+v gate=%s", d.childID, gate)
	stopRealParkRecoveryHelper(t, ctx, d)
	completion, found, err := (PairLifecycleStoreIO{Store: d.store}).ObserveCompletion(context.Background(), d.paths, r)
	if err != nil || found {
		t.Fatalf("interrupted cleanup fabricated completion: %+v found=%v err=%v", completion, found, err)
	}
	result, err := env.Couch.recoverPark(ctx, d.current.Address)
	if err != nil {
		t.Fatal(err)
	}
	assertRealParkRecovered(t, env, d, false)
	// Independently run the same semantic class through the stateful fake.
	fake, before := interruptedParkEnv(t, false)
	fakeResult, err := fake.Couch.recoverPark(ctx, before.Address)
	if err != nil {
		t.Fatal(err)
	}
	if result.OperatorOutcome != fakeResult.OperatorOutcome || result.Diagnostic == "" || fakeResult.Diagnostic == "" {
		t.Fatalf("fake/live interruption outcomes differ: fake=%+v real=%+v", fakeResult, result)
	}
	revision := result.Thread.Revision
	repeated, err := env.Couch.recoverPark(ctx, d.current.Address)
	if err != nil || repeated.Thread.Revision != revision || repeated.Outcome != "park-already-settled" {
		t.Fatalf("duplicate recovery changed state: %+v %v", repeated, err)
	}
}

func stopRealParkRecoveryHelper(t *testing.T, ctx context.Context, d *realParkConformanceDriver) {
	t.Helper()
	// pty.Start creates a private session/group. Check both exact incarnation and
	// its expected group before signalling; never select a process by text.
	if state := observeExactProcess(OSProcOps{}, d.childID); state != Live {
		t.Fatalf("cannot prove owned helper identity before kill: %s", state)
	}
	group, err := syscall.Getpgid(d.childID.PID)
	if err != nil || group != d.childID.PID {
		t.Fatalf("helper is not private group leader: group=%d err=%v", group, err)
	}
	if err := syscall.Kill(-group, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-d.childWait:
		if err == nil {
			t.Fatal("interrupted helper unexpectedly exited successfully")
		}
	case <-ctx.Done():
		t.Fatal(fmt.Errorf("reap interrupted exact helper: %w", ctx.Err()))
	}
	if state := observeExactProcess(OSProcOps{}, d.childID); state != Dead {
		t.Fatalf("helper death unproved: %s", state)
	}
}
