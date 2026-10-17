package couchcore

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

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
			result, err := env.Couch.RecoverPark(context.Background(), before.Address)
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
			if _, err := env.Couch.RecoverPark(context.Background(), before.Address); err != nil {
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
			_, err := env.Couch.RecoverPark(context.Background(), before.Address)
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
