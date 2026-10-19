package couchcore

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// recoverPark exercises the shared core without admission-time identity capture.
// Public operation tests separately require the exact captured nonce.
func (c *Couch) recoverPark(ctx context.Context, address ThreadAddress) (ParkRecoveryResult, error) {
	return c.recoverParkExpected(ctx, address, "", false)
}

func interruptedParkEnv(t *testing.T, failedCompletion bool) (*relaunchEnv, ThreadRecord) {
	t.Helper()
	env, live := envWithLiveThread(t)
	env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error {
		if err := env.model.DeliverTrigger(env.Lifecycle.lastRequest); err != nil {
			return err
		}
		if failedCompletion {
			c := cleanupFailureCompletion(env.Lifecycle.lastRequest, env.Now)
			env.Lifecycle.completion = &c
			_ = env.model.CommitCompletion(env.Lifecycle.lastRequest, pairlifecycle.CleanupResult{Outcome: pairlifecycle.CompletionFailure, CompletedAt: env.Now})
		}
		env.Proc.Kill(42)
		env.Artifacts.SetPairSession(live.Address, "pair-exact", false)
		env.Artifacts.SetSessionPresence(live.Address, SessionObservation{State: SessionAbsent})
		if failedCompletion {
			return nil
		}
		return errors.New("interrupted after child quit")
	}
	_, err := env.Couch.PairLifecycle.Park(context.Background(), live.Address)
	if err == nil {
		t.Fatal("interruption should fail")
	}
	current, err := env.Couch.Threads.GetThread(live.Address)
	if err != nil {
		t.Fatal(err)
	}
	return env, current
}

func TestRecoverParkRetiresInterruptedDeadOwner(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "failed"}[failed], func(t *testing.T) {
			env, before := interruptedParkEnv(t, failed)
			result, err := env.Couch.recoverPark(context.Background(), before.Address)
			if err != nil {
				t.Fatal(err)
			}
			after, err := env.Couch.Threads.GetThread(before.Address)
			if err != nil {
				t.Fatal(err)
			}
			if after.Park != nil || len(after.Incarnations) != 0 || after.VerifiedPark != nil {
				t.Fatalf("not safely retired: %+v", after)
			}
			if len(after.ParkHistory) == 0 || !after.ParkHistory[len(after.ParkHistory)-1].Tombstoned {
				t.Fatal("interrupted history lost")
			}
			if !reflect.DeepEqual(before.LatestLaunchProfile, after.LatestLaunchProfile) || !reflect.DeepEqual(before.SessionBinding, after.SessionBinding) {
				t.Fatal("conversation configuration changed")
			}
			if result.OperatorOutcome != "resume-from-disk" || result.NextCommand == "" || result.Diagnostic == "" {
				t.Fatalf("missing operator outcome/warning: %+v", result)
			}
			revision := after.Revision
			if _, err := env.Couch.recoverPark(context.Background(), before.Address); err != nil {
				t.Fatal(err)
			}
			repeated, _ := env.Couch.Threads.GetThread(before.Address)
			if repeated.Revision != revision || len(env.Runner.Ops) != 0 || len(env.Artifacts.TriggeredQuits()) != 1 {
				t.Fatal("duplicate recovery had effects")
			}
		})
	}
}

func TestRecoverParkRefusesUnprovedOrReplacementOwner(t *testing.T) {
	for _, kind := range []string{"live", "unknown", "replacement", "unknown-terminal"} {
		t.Run(kind, func(t *testing.T) {
			env, before := interruptedParkEnv(t, false)
			switch kind {
			case "live":
				env.Proc.Set(42, "pair-helper")
			case "unknown":
				env.Proc.SetUnknown(42)
			case "replacement":
				_, err := env.Couch.Threads.updateExistingThread(before.Address, before.Revision, func(r *ThreadRecord) error {
					r.Park.Phase = ParkUnknown
					r.Park.Attempts[0].Failure = &ParkFailure{Code: pairlifecycle.FailureReplacementIncarnation, Diagnostic: "replacement"}
					r.Incarnations[0].PID = 43
					r.Incarnations[0].Identity = "other"
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			case "unknown-terminal":
				env.Artifacts.BeforePairSession = func(ThreadAddress) error { return errors.New("terminal observation unavailable") }
			}
			expected, _ := env.Couch.Threads.GetThread(before.Address)
			_, err := env.Couch.recoverPark(context.Background(), before.Address)
			if err == nil {
				t.Fatal("unproved retirement accepted")
			}
			after, _ := env.Couch.Threads.GetThread(before.Address)
			if !reflect.DeepEqual(expected, after) {
				t.Fatalf("refusal destroyed state: %v", err)
			}
		})
	}
}

func TestRecoverParkAbsentTerminalRefusesBeforeCompletionLock(t *testing.T) {
	for _, owner := range []Liveness{Live, Unknown} {
		t.Run(owner.String(), func(t *testing.T) {
			env, before := interruptedParkEnv(t, false)
			if owner == Live {
				env.Proc.Set(42, "pair-helper")
			} else {
				env.Proc.SetUnknown(42)
			}
			observed := false
			env.Lifecycle.onObserve = func(pairlifecycle.QuitRequest) {
				observed = true
				// A real cleanup lock can stay held until this owner exits.
				// Recovery must refuse before entering that observation.
				env.Proc.Kill(42)
			}
			_, err := env.Couch.recoverPark(context.Background(), before.Address)
			if err == nil || observed {
				t.Fatalf("refusal entered completion reconciliation: observed=%t err=%v", observed, err)
			}
			if !strings.Contains(err.Error(), "Recover park") || (owner == Live && !strings.Contains(err.Error(), "park is still finishing")) {
				t.Fatalf("refusal lacks finishing/retry advice: %v", err)
			}
			after, getErr := env.Couch.Threads.GetThread(before.Address)
			if getErr != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("refusal changed retained state: %+v %v", after, getErr)
			}
		})
	}
}

func TestDecideParkRecoveryAuthorityCrossProduct(t *testing.T) {
	for _, open := range []bool{false, true} {
		for _, exact := range []bool{false, true} {
			for _, unpublished := range []bool{false, true} {
				for _, owner := range []Liveness{Unknown, Live, Dead} {
					for _, terminal := range []SessionState{SessionUnresolved, SessionAbsent, SessionPresent, SessionOrphaned} {
						in := ParkRecoveryFacts{Open: open, ExactOwner: exact, Unpublished: unpublished, Owner: owner, Terminal: terminal}
						got := DecideParkRecovery(in)
						if !open {
							if got != parkRecoverySettled {
								t.Fatalf("settled: %+v =>%s", in, got)
							}
							continue
						}
						switch got {
						case parkRecoveryRollback:
							if !exact || !unpublished {
								t.Fatalf("rollback lacks authority: %+v", in)
							}
						case parkRecoveryRetire:
							if !exact || unpublished || owner != Dead || terminal != SessionAbsent {
								t.Fatalf("retirement lacks authority: %+v", in)
							}
						case parkRecoveryDrive:
							if !exact || unpublished || owner == Unknown || terminal != SessionPresent {
								t.Fatalf("drive lacks authority: %+v", in)
							}
						case parkRecoveryRefuse:
							if exact && (unpublished || (owner != Unknown && terminal == SessionPresent) || (owner == Dead && terminal == SessionAbsent)) {
								t.Fatalf("safe case inexplicably refused: %+v", in)
							}
						default:
							t.Fatalf("unknown decision: %s", got)
						}
					}
				}
			}
		}
	}
}

func TestRetireInterruptedParkJournalRecovery(t *testing.T) {
	for _, boundary := range []string{"before-update", "after-journal", "after-target"} {
		t.Run(boundary, func(t *testing.T) {
			env, before := interruptedParkEnv(t, false)
			interrupted := errors.New("interrupted journal")
			switch boundary {
			case "before-update":
				env.Couch.Threads.hooks.BeforeUpdate = func(ThreadRecord, ThreadRecord) error { return interrupted }
			case "after-journal":
				env.Couch.Threads.hooks.AfterJournal = func() error { return interrupted }
			case "after-target":
				env.Couch.Threads.hooks.AfterTarget = func(int) error { return interrupted }
			}
			_, err := env.Couch.Threads.RetireInterruptedPark(context.Background(), before.Address, before.Revision, before.Park.Identity, env.Now)
			if !errors.Is(err, interrupted) {
				t.Fatalf("missing injected failure: %v", err)
			}
			env.Couch.Threads.hooks = threadStoreHooks{}
			after, err := env.Couch.Threads.recoveryReadThread(context.Background(), before.Address)
			if err != nil {
				t.Fatal(err)
			}
			if boundary == "before-update" {
				if !reflect.DeepEqual(after, before) {
					t.Fatal("precommit failure mutated record")
				}
				return
			}
			if after.Park != nil || len(after.Incarnations) != 0 || after.VerifiedPark != nil || len(after.ParkHistory) != 1 {
				t.Fatalf("journal failed to converge atomically: %+v", after)
			}
		})
	}
}

func TestRecoverParkExpectedAbsentCannotDriveNewPark(t *testing.T) {
	env, before := interruptedParkEnv(t, false)
	if _, err := env.Couch.RecoverParkExpected(context.Background(), before.Address, ""); err == nil {
		t.Fatal("absent-park selection drove a newly opened park")
	}
	after, _ := env.Couch.Threads.GetThread(before.Address)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("stale empty nonce changed transaction")
	}
}

// Signal the real advisory-lock boundary without replacing its contention.
type recoveryContendedRuntime struct {
	pairlifecycle.OSRuntime
	entered chan struct{}
	once    sync.Once
}

func (r *recoveryContendedRuntime) Lock(path string) (pairlifecycle.Unlocker, error) {
	r.once.Do(func() { close(r.entered) })
	return r.OSRuntime.Lock(path)
}

func TestRecoverParkCancellationSettlesContendedLifecycleWorker(t *testing.T) {
	for _, present := range []bool{false, true} {
		t.Run(map[bool]string{false: "dead-absent", true: "terminal-present"}[present], func(t *testing.T) {
			env, before := interruptedParkEnv(t, false)
			if present {
				env.Artifacts.SetPairSession(before.Address, "pair-exact", true)
			}
			_, paths, err := env.Couch.PairLifecycle.requestFor(before, "pair-exact")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(paths.Dir(), 0700); err != nil {
				t.Fatal(err)
			}
			lock, err := (pairlifecycle.OSRuntime{}).Lock(paths.Lock())
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			runtime := &recoveryContendedRuntime{entered: make(chan struct{})}
			env.Couch.PairLifecycle.Lifecycle = PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: runtime}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := env.Couch.RecoverParkExpected(ctx, before.Address, before.Park.Identity.Nonce)
				result <- err
			}()
			select {
			case <-runtime.entered:
			case <-time.After(time.Second):
				t.Fatal("worker did not reach lifecycle lock")
			}
			worker := env.Couch.PairLifecycle.worker
			worker.mu.Lock()
			future := worker.active[before.Address].future
			worker.mu.Unlock()
			if future == nil {
				t.Fatal("missing admitted future")
			}
			defer func() { _ = lock.Close(); <-future.done }()
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("caller: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("caller stuck")
			}
			select {
			case <-future.done:
			case <-time.After(time.Second):
				t.Fatal("canceled recovery retained worker reservation behind lifecycle lock")
			}
			if env.Couch.PairLifecycle.parkActive(before.Address) {
				t.Fatal("worker reservation retained")
			}
			after, err := env.Couch.Threads.GetThread(before.Address)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("canceled observation mutated state: %v", err)
			}
		})
	}
}

func (r *recoveryContendedRuntime) LockContext(ctx context.Context, path string) (pairlifecycle.Unlocker, error) {
	r.once.Do(func() { close(r.entered) })
	return r.OSRuntime.LockContext(ctx, path)
}

func TestRecoverParkSettlesWhenThreadStoreBecomesBusyAfterAdmission(t *testing.T) {
	for _, completion := range []bool{false, true} {
		t.Run(map[bool]string{false: "drive-read", true: "completion-finalize"}[completion], func(t *testing.T) {
			env, before := interruptedParkEnv(t, false)
			env.Artifacts.SetPairSession(before.Address, "pair-exact", true)
			if completion {
				value := successCompletion(env.Lifecycle.lastRequest, env.Now)
				env.Lifecycle.completion = &value
			}
			entered := make(chan struct{})
			release := make(chan struct{})
			var once sync.Once
			env.Lifecycle.onObserve = func(pairlifecycle.QuitRequest) {
				once.Do(func() {
					lock, err := acquireThreadStoreLockMode(env.Couch.Threads.root, false)
					if err != nil {
						panic(err)
					}
					go func() { <-release; _ = lock.Close() }()
					close(entered)
				})
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			future, err := env.Couch.PairLifecycle.submitFuture(ctx, before.Address, before.Park.Identity.Nonce, func(ctx context.Context) (ParkResult, error) {
				return env.Couch.PairLifecycle.recoverInterruptedPark(ctx, before.Address, before.Park.Identity.Nonce)
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { close(release); <-future.done }()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("observation gate not reached")
			}
			cancel()
			select {
			case <-future.done:
			case <-time.After(time.Second):
				t.Fatal("recovery retained worker behind post-admission threadstore lock")
			}
			if env.Couch.PairLifecycle.parkActive(before.Address) {
				t.Fatal("address reservation retained")
			}
			if future.work.err == nil {
				t.Fatal("contended/canceled recovery reported success")
			}
		})
	}
}

func TestCanceledLifecycleObservationAndCleanupPreserveEvidence(t *testing.T) {
	env, before := interruptedParkEnv(t, false)
	request, paths, err := env.Couch.PairLifecycle.requestFor(before, "pair-exact")
	if err != nil {
		t.Fatal(err)
	}
	store := pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}
	if err := store.PublishRequest(paths, request); err != nil {
		t.Fatal(err)
	}
	if err := store.PublishCompletion(paths, successCompletion(request, env.Now)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	adapter := PairLifecycleStoreIO{Store: store}
	if _, found, err := adapter.ObserveCompletion(ctx, paths, request); !errors.Is(err, context.Canceled) || found {
		t.Fatalf("canceled observation became absence/success: found=%t err=%v", found, err)
	}
	if err := adapter.CleanupAttempt(ctx, paths, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("cleanup ignored cancellation: %v", err)
	}
	requestPath, _ := paths.Request(request.Attempt)
	completionPath, _ := paths.Completion(request.Attempt)
	for _, path := range []string{requestPath, completionPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("evidence removed: %v", err)
		}
	}
}
