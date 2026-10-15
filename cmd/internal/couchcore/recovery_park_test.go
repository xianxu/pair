package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestPreparedRecoveryParksSettledOwnerBeforeTerminalRetirement(t *testing.T) {
	for _, failPark := range []bool{false, true} {
		t.Run(map[bool]string{false: "parked", true: "park-failed"}[failPark], func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			f.Proc.Set(42, "old-helper")
			f.Artifacts.SetPairSession(f.record.Address, "old-session", true)
			model := pairlifecycletest.New(f.Now)
			model.SetSession("old-session", true)
			lifecycle := &fakeControllerLifecycle{model: model, store: f.Couch.Threads}
			parkErr := errors.New("park publication refused")
			if failPark {
				lifecycle.publishErr = parkErr
			}
			lifecycle.onPublish = func(pairlifecycle.QuitRequest) {
				current, err := f.Couch.Threads.GetThread(f.record.Address)
				if err != nil || current.RecoveryIntent == nil || f.preserved != 1 {
					t.Fatalf("park before durable evidence: record=%+v preserved=%d err=%v", current, f.preserved, err)
				}
			}
			f.Artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
				if err := model.DeliverTrigger(lifecycle.lastRequest); err != nil {
					return err
				}
				completion := successCompletion(lifecycle.lastRequest, f.Now)
				lifecycle.completion = &completion
				f.Proc.Kill(42)
				f.Artifacts.SetPairSession(f.record.Address, "old-session", false)
				f.Artifacts.SetSessionPresence(f.record.Address, SessionObservation{State: SessionAbsent})
				return model.CommitCompletion(lifecycle.lastRequest, pairlifecycle.CleanupResult{Outcome: pairlifecycle.CompletionSuccess, CompletedAt: f.Now})
			}
			f.Couch.PairLifecycle = &PairLifecycleController{
				Threads: f.Couch.Threads, DataDir: t.TempDir(), Lifecycle: lifecycle,
				Sessions: f.Artifacts, Proc: f.Proc, Clock: f.Couch.Clock,
				Nonce: func() (string, error) { return "park-0123456789abcdef", nil },
			}
			retired := 0
			f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
				return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 70, Identity: "agent", Role: "native-agent"}}, Digest: "exact-terminal", Retire: func(context.Context) error {
					retired++
					current, err := f.Couch.Threads.GetThread(f.record.Address)
					if err != nil || current.VerifiedPark == nil || len(f.Artifacts.TriggeredQuits()) != 1 {
						t.Fatalf("terminal retirement preceded normal park: record=%+v err=%v", current, err)
					}
					return nil
				}}, nil
			}
			_, err := f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh})
			var offer *RecoveryConfirmationRequired
			if !errors.As(err, &offer) {
				t.Fatalf("missing confirmation: %v", err)
			}
			namedHelper := false
			for _, stop := range offer.Offer.Stops {
				if stop.PID == 42 && stop.Identity == "old-helper" && stop.Role == "client-helper" {
					namedHelper = true
				}
			}
			if !namedHelper {
				t.Fatal("normal park's client helper was not named in the stop confirmation")
			}
			_, err = f.Couch.ConfirmRecovery(t.Context(), offer.Offer.Token)
			if failPark {
				if !errors.Is(err, parkErr) || retired != 0 || len(f.Runner.Ops) != 0 {
					t.Fatalf("failed park fell through: retired=%d runner=%v err=%v", retired, f.Runner.Ops, err)
				}
				return
			}
			if err != nil || retired != 1 || len(lifecycle.trace) == 0 || !strings.Contains(strings.Join(lifecycle.trace, ","), "publish-request") {
				t.Fatalf("park route: retired=%d trace=%v err=%v", retired, lifecycle.trace, err)
			}
		})
	}
}

func TestPreparedRecoveryCannotRetireUnobservableConfirmedHelper(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Proc.SetUnknown(42)
	helper := f.record.Incarnations[0]
	_, err := f.Couch.parkPreparedRecovery(t.Context(), f.record, &helper)
	if err == nil || len(f.Proc.Signals[42]) != 0 || len(f.Runner.Ops) != 0 {
		t.Fatalf("unknown confirmed helper treated as absent: err=%v signals=%v runner=%v", err, f.Proc.Signals, f.Runner.Ops)
	}
}
