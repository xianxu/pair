package couchcore

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// A committed request whose acknowledgement is lost must never be rolled back
// as though publication were known not to have happened.
type interruptedRequestPublication struct{ *fakeControllerLifecycle }

func (p interruptedRequestPublication) PublishRequest(ctx context.Context, paths artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) error {
	if err := p.fakeControllerLifecycle.PublishRequest(ctx, paths, request); err != nil {
		return err
	}
	return &pairlifecycle.PublicationError{Outcome: pairlifecycle.Indeterminate, Err: errors.New("lost request publication acknowledgement")}
}

func TestParkInterruptionBoundaryMatrix(t *testing.T) {
	for _, boundary := range []string{"before-begin", "after-begin-cancel", "before-request", "after-request", "before-request-ack", "before-trigger", "after-trigger", "completion-before-child-death", "dead-without-completion", "before-finalize", "after-finalize-journal", "after-cleanup"} {
		t.Run(boundary, func(t *testing.T) {
			env, original := envWithLiveThread(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stop := errors.New("injected park boundary interruption")
			goodTrigger := env.Artifacts.TriggerQuitHook
			var pending *parkFuture
			completionDeadlineSpent := false
			captureAndCancel := func() {
				w := env.Couch.PairLifecycle.worker
				w.mu.Lock()
				pending = w.active[original.Address].future
				w.mu.Unlock()
				cancel()
			}
			switch boundary {
			case "before-begin":
				env.Couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
					if old.Park == nil && next.Park != nil {
						return stop
					}
					return nil
				}
			case "after-begin-cancel":
				env.Couch.PairLifecycle.Clock = &parkCancelClock{now: env.Now, cancel: captureAndCancel}
			case "before-request":
				env.Lifecycle.publishErr = stop
			case "after-request":
				env.Couch.PairLifecycle.Lifecycle = interruptedRequestPublication{env.Lifecycle}
			case "before-request-ack":
				env.Couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
					if old.Park != nil && next.Park != nil && old.Park.Phase == ParkRequested && next.Park.Phase == ParkAwaitingCompletion {
						return stop
					}
					return nil
				}
			case "before-trigger":
				env.Lifecycle.observeErr = stop
			case "after-trigger":
				env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error {
					if err := env.model.DeliverTrigger(env.Lifecycle.lastRequest); err != nil {
						return err
					}
					return stop
				}
			case "completion-before-child-death":
				// Spend the existing budget only after the matching completion
				// is observed while the exact child remains live. Fixture startup
				// and machine scheduling cannot consume this ordering proof.
				var budget *spendableBudget
				env.Couch.PairLifecycle.completionDeadline = func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
					budget = newSpendableBudget()
					return budget.bind(parent)
				}
				env.Couch.PairLifecycle.Wait = func(ctx context.Context, _ time.Duration) error {
					if budget == nil || env.Lifecycle.completion == nil || env.Proc.Exists(42) != Live {
						return errors.New("deadline preceded matching completion and live-child proof")
					}
					completionDeadlineSpent = true
					budget.spend()
					return ctx.Err()
				}
				env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error {
					if err := env.model.DeliverTrigger(env.Lifecycle.lastRequest); err != nil {
						return err
					}
					completion := successCompletion(env.Lifecycle.lastRequest, env.Now)
					env.Lifecycle.completion = &completion
					return env.model.CommitCompletion(env.Lifecycle.lastRequest, pairlifecycle.CleanupResult{Outcome: pairlifecycle.CompletionSuccess, CompletedAt: env.Now})
				}
			case "dead-without-completion":
				env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error {
					if err := env.model.DeliverTrigger(env.Lifecycle.lastRequest); err != nil {
						return err
					}
					env.Proc.Kill(42)
					env.Artifacts.SetPairSession(original.Address, "pair-exact", false)
					env.Artifacts.SetSessionPresence(original.Address, SessionObservation{State: SessionAbsent})
					return stop
				}
			case "before-finalize", "after-finalize-journal":
				armed := false
				env.Couch.Threads.hooks.AfterJournal = func() error {
					if armed {
						return stop
					}
					return nil
				}
				env.Couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
					if old.Park != nil && next.Park == nil && next.VerifiedPark != nil {
						if boundary == "before-finalize" {
							return stop
						}
						armed = true
					}
					return nil
				}
			case "after-cleanup":
				env.Lifecycle.onCleanup = func(pairlifecycle.QuitRequest) { captureAndCancel() }
			}
			_, err := env.Couch.Relaunch(ctx, original.Address)
			if err == nil {
				t.Fatal("injected interruption was reported as successful relaunch")
			}
			if boundary == "completion-before-child-death" && (!completionDeadlineSpent || !errors.Is(err, context.DeadlineExceeded)) {
				t.Fatalf("completion/live-child boundary did not spend its deadline: spent=%t err=%v", completionDeadlineSpent, err)
			}
			// Caller cancellation can return ahead of the worker's bounded settlement.
			if pending != nil {
				wait, done := context.WithTimeout(context.Background(), time.Second)
				defer done()
				_, waitErr := pending.Await(wait)
				if errors.Is(waitErr, context.DeadlineExceeded) {
					t.Fatal("owned worker did not settle")
				}
			}
			env.Couch.Threads.hooks = threadStoreHooks{}
			current, getErr := env.Couch.Threads.recoveryReadThread(context.Background(), original.Address)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if !reflect.DeepEqual(current.LatestLaunchProfile, original.LatestLaunchProfile) || !reflect.DeepEqual(current.SessionBinding, original.SessionBinding) {
				t.Fatal("interruption changed retained conversation authority")
			}
			if len(env.Runner.Ops) != 0 {
				t.Fatalf("interrupted relaunch spawned a replacement: %v", env.Runner.Ops)
			}
			unpublished := boundary == "before-begin" || boundary == "after-begin-cancel" || boundary == "before-request"
			settled := boundary == "after-finalize-journal" || boundary == "after-cleanup"
			if unpublished {
				if current.Park != nil || current.VerifiedPark != nil || !reflect.DeepEqual(current.Incarnations, original.Incarnations) || len(env.Artifacts.TriggeredQuits()) != 0 {
					t.Fatalf("unpublished interruption affected live child: %+v", current)
				}
				return
			}
			if settled {
				if current.Park != nil || current.VerifiedPark == nil || len(current.Incarnations) != 0 {
					t.Fatalf("durable successful completion not authoritative: %+v", current)
				}
			} else if current.Park == nil || current.VerifiedPark != nil {
				t.Fatalf("uncertain publication was lost or fabricated success: %+v", current)
			}
			env.Lifecycle.publishErr = nil
			env.Lifecycle.observeErr = nil
			env.Lifecycle.onCleanup = nil
			env.Couch.PairLifecycle.Lifecycle = env.Lifecycle
			env.Couch.PairLifecycle.Clock = FixedClock{T: env.Now}
			if boundary != "completion-before-child-death" {
				env.Artifacts.TriggerQuitHook = goodTrigger
			}
			if boundary == "completion-before-child-death" {
				_, recoverErr := env.Couch.recoverPark(context.Background(), original.Address)
				still, _ := env.Couch.Threads.GetThread(original.Address)
				if recoverErr == nil || still.Park == nil || still.VerifiedPark != nil || len(still.Incarnations) != 1 {
					t.Fatalf("completion retired a live child: %+v %v", still, recoverErr)
				}
				env.Proc.Kill(42)
				env.Artifacts.SetPairSession(original.Address, "pair-exact", false)
				env.Artifacts.SetSessionPresence(original.Address, SessionObservation{State: SessionAbsent})
			}
			recovered, recoverErr := env.Couch.recoverPark(context.Background(), original.Address)
			if recoverErr != nil {
				t.Fatalf("explicit recovery after %s: %v", boundary, recoverErr)
			}
			after := recovered.Thread
			if after.Park != nil || len(after.Incarnations) != 0 || len(after.ParkHistory) == 0 || !reflect.DeepEqual(after.LatestLaunchProfile, original.LatestLaunchProfile) {
				t.Fatalf("recovery lost state or remained blocked: %+v", after)
			}
			if boundary == "dead-without-completion" {
				if after.VerifiedPark != nil || !after.ParkHistory[len(after.ParkHistory)-1].Tombstoned || recovered.Diagnostic == "" {
					t.Fatal("missing completion became a verified park or lost interrupted-cleanup warning")
				}
			} else if after.VerifiedPark == nil || after.ParkHistory[len(after.ParkHistory)-1].Tombstoned {
				t.Fatal("successful exact completion was not authoritative")
			}
			if recovered.OperatorOutcome != "resume-from-disk" || len(env.Runner.Ops) != 0 {
				t.Fatalf("recovery launched instead of advising: %+v %v", recovered, env.Runner.Ops)
			}
		})
	}
}

func TestParkInterruptionEveryCleanupStage(t *testing.T) {
	stages := []pairlifecycle.CleanupStage{pairlifecycle.StageSessionQuiescence, pairlifecycle.StageEditorReap, pairlifecycle.StageScrollbackPreserve, pairlifecycle.StageSidecarCleanup, pairlifecycle.StagePollerCleanup, pairlifecycle.StageCmuxCleanup}
	for _, stage := range stages {
		t.Run(string(stage), func(t *testing.T) {
			env, original := envWithLiveThread(t)
			env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error {
				if err := env.model.DeliverTrigger(env.Lifecycle.lastRequest); err != nil {
					return err
				}
				env.model.FailStage(stage, errors.New("interrupted cleanup stage"))
				cleanup := pairlifecycle.RunCleanup(context.Background(), pairlifecycle.CleanupCouch, env.model)
				if cleanup.Outcome != pairlifecycle.CompletionFailure || len(cleanup.Failures) != 1 || cleanup.Failures[0].Stage != stage {
					t.Fatalf("fault did not exercise requested stage %s: %+v", stage, cleanup)
				}
				attempts := env.model.AttemptTrace()
				if stage == pairlifecycle.StageSessionQuiescence {
					if len(attempts) != 1 {
						t.Fatalf("quiescence failure allowed later destructive cleanup: %v", attempts)
					}
				} else if !reflect.DeepEqual(attempts, stages) {
					t.Fatalf("independent cleanup stages skipped: %v", attempts)
				}
				completion := cleanupFailureCompletion(env.Lifecycle.lastRequest, env.Now)
				env.Lifecycle.completion = &completion
				env.Proc.Kill(42)
				env.Artifacts.SetPairSession(original.Address, "pair-exact", false)
				env.Artifacts.SetSessionPresence(original.Address, SessionObservation{State: SessionAbsent})
				return env.model.CommitCompletion(env.Lifecycle.lastRequest, cleanup)
			}
			_, err := env.Couch.Relaunch(context.Background(), original.Address)
			if err == nil {
				t.Fatal("cleanup failure reported a successful relaunch")
			}
			before, getErr := env.Couch.Threads.GetThread(original.Address)
			if getErr != nil || before.Park == nil || before.VerifiedPark != nil {
				t.Fatalf("cleanup failure evidence not retained: %+v %v", before, getErr)
			}
			recovered, recoverErr := env.Couch.recoverPark(context.Background(), original.Address)
			if recoverErr != nil {
				t.Fatal(recoverErr)
			}
			after := recovered.Thread
			if after.Park != nil || after.VerifiedPark != nil || len(after.Incarnations) != 0 || len(after.ParkHistory) != 1 || !after.ParkHistory[0].Tombstoned {
				t.Fatalf("failed cleanup became success or remained stranded: %+v", after)
			}
			if !reflect.DeepEqual(after.LatestLaunchProfile, original.LatestLaunchProfile) || !reflect.DeepEqual(after.SessionBinding, original.SessionBinding) || recovered.OperatorOutcome != "resume-from-disk" || recovered.Diagnostic == "" {
				t.Fatalf("retirement lost conversation or warning: %+v", recovered)
			}
			if len(env.Runner.Ops) != 0 || len(env.Artifacts.TriggeredQuits()) != 1 {
				t.Fatal("recovery reran quit or launched a replacement")
			}
		})
	}
}
