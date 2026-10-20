package couchcore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func recoveryTrackedFixture(t *testing.T) (*testEnv, trackedThreadLaunch) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	record := admittedStartRecord(t)
	profile := LaunchProfile{Agent: "codex", Argv: []string{}}
	intent := taggedRecoveryIntent(record, threadrecord.RecoveryFresh)
	claimed, err := AdvanceStartTransaction(record, StartEvent{Kind: StartClaimed, Nonce: "recovery-attempt", Owner: SupervisorOwner{PID: 42, Identity: "owner"}, Profile: &profile, RecoveryIntent: intent})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err = env.Couch.Threads.CreateThread(claimed)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := launcher.BuildCouchFreshLaunchProfile(string(record.Address.Tag), "codex", []string{}, "explicit", "explicit")
	if err != nil {
		t.Fatal(err)
	}
	in := trackedThreadLaunch{Context: context.Background(), Thread: claimed, Nonce: "recovery-attempt", Args: StartArgs{Stack: "codex", Worktree: "/repo"}, StartedAt: record.CreatedAt, ProfileRaw: raw, Fresh: true}
	in.Recovery = true
	return env, in
}

func TestRecoveryLaunchUncertaintyPreservesExactClaim(t *testing.T) {
	for _, scenario := range []string{"registration lost", "registration timeout", "ack uncertainty", "cancel after ack", "promotion drift", "late promotion drift", "helper exited"} {
		t.Run(scenario, func(t *testing.T) {
			env, in := recoveryTrackedFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			in.Context = ctx
			if scenario == "registration timeout" {
				env.Couch.registrationDeadline = func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
					return context.WithDeadline(parent, time.Unix(1, 0))
				}
			}
			cause := errors.New("registration observation lost")
			probes := 0
			env.Couch.FreshRegistration = func(ctx context.Context, address ThreadAddress, agent, nonce string) (bool, error) {
				probes++
				if ctx.Err() != nil && !((scenario == "helper exited" || scenario == "registration timeout") && probes == 1) {
					t.Fatalf("reobservation inherited cancelled parent: %v", ctx.Err())
				}
				if address != in.Thread.Address || agent != "codex" || nonce != in.Nonce {
					t.Fatal("reobservation changed attempt")
				}
				if scenario == "late promotion drift" && probes == 1 {
					return false, cause
				}
				if scenario == "promotion drift" || scenario == "late promotion drift" {
					if probes == 1 || scenario == "late promotion drift" && probes == 2 {
						current, err := env.Couch.Threads.GetThread(address)
						if err != nil {
							t.Fatal(err)
						}
						_, err = env.Couch.Threads.updateExistingThread(address, current.Revision, func(next *ThreadRecord) error { next.Name = "concurrent metadata"; return nil })
						if err != nil {
							t.Fatal(err)
						}
					}
					return true, nil
				}
				if scenario == "registration timeout" && probes == 1 {
					return false, nil
				}
				return false, cause
			}
			env.Runner.AfterAcknowledge = func(id string) error {
				if scenario == "promotion drift" || scenario == "late promotion drift" {
					env.Artifacts.SetPairSession(in.Thread.Address, continuationChildSession(t, env.Runner, id), true)
				}
				if scenario == "cancel after ack" {
					cancel()
				}
				if scenario == "helper exited" {
					env.Runner.SetExited(id, 23)
					// Failed terminal observation, unlike proven absence, cannot
					// release the exited helper's recovery claim.
					env.Artifacts.BeforePairSession = func(ThreadAddress) error { return cause }
				}
				if scenario == "ack uncertainty" {
					return errors.New("ack transport uncertain")
				}
				return nil
			}
			_, h, err := env.Couch.launchTrackedThread(in)
			var unconfirmed *RecoveryLaunchUnconfirmed
			if !errors.As(err, &unconfirmed) || unconfirmed.Address != in.Thread.Address || unconfirmed.Nonce != in.Nonce {
				t.Fatalf("missing typed unconfirmed attempt: %v", err)
			}
			if err == nil || h == nil {
				t.Fatalf("expected uncertain attempt: handle %v err %v", h, err)
			}
			got, readErr := env.Couch.Threads.GetThread(in.Thread.Address)
			if readErr != nil {
				t.Fatalf("claim removed: %v", readErr)
			}
			if len(got.Incarnations) != 1 || got.Incarnations[0].Start == nil || got.Incarnations[0].Start.Nonce != in.Nonce || !reflect.DeepEqual(got.Incarnations[0].Start.RecoveryIntent, in.Thread.Incarnations[0].Start.RecoveryIntent) {
				t.Fatalf("uncertain launch lost exact claim: %+v", got)
			}
			if len(env.Runner.Signals(h.ID())) != 0 || len(env.Artifacts.TriggeredQuits()) != 0 {
				t.Fatal("uncertain recovery was destructively cleaned up")
			}
			starts := 0
			for _, op := range env.Runner.Ops {
				if strings.HasPrefix(op, "start ") {
					starts++
				}
			}
			if starts != 1 || env.Runner.Child(h.ID()).ExecCount != 1 {
				t.Fatalf("retried recovery: starts %d child %+v", starts, env.Runner.Child(h.ID()))
			}
			expected := 2
			if scenario == "cancel after ack" || scenario == "ack uncertainty" {
				expected = 1
			}
			if probes != expected {
				t.Fatalf("registration probes %d want %d", probes, expected)
			}
		})
	}
}

func TestRecoveryLaunchLateRegistrationPromotesWithoutRetry(t *testing.T) {
	for _, phase := range []string{"registration", "ack", "cancel"} {
		t.Run(phase, func(t *testing.T) {
			env, in := recoveryTrackedFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			in.Context = ctx
			probes := 0
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
				probes++
				if phase == "registration" && probes == 1 {
					return false, errors.New("initial observation unavailable")
				}
				return true, nil
			}
			env.Runner.AfterAcknowledge = func(id string) error {
				env.Artifacts.SetPairSession(in.Thread.Address, continuationChildSession(t, env.Runner, id), true)
				switch phase {
				case "ack":
					return errors.New("ack transport uncertain")
				case "cancel":
					cancel()
				}
				return nil
			}
			actor, h, err := env.Couch.launchTrackedThread(in)
			if err != nil {
				t.Fatal(err)
			}
			got, err := env.Couch.Threads.GetThread(in.Thread.Address)
			if err != nil {
				t.Fatal(err)
			}
			expected := 1
			if phase == "registration" {
				expected = 2
			}
			if probes != expected || actor.Thread != in.Thread.Address || h == nil || got.Incarnations[0].State != IncarnationLive || got.Incarnations[0].Start != nil || got.RecoveryIntent == nil {
				t.Fatalf("late proof not promoted: probes %d record %+v", probes, got)
			}
			if len(env.Runner.Signals(h.ID())) != 0 || env.Runner.Child(h.ID()).ExecCount != 1 {
				t.Fatal("late proof relaunched or killed attempt")
			}
		})
	}
}

func TestRecoveryLaunchPreAckCancellationStillRollsBack(t *testing.T) {
	env, in := recoveryTrackedFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in.Context = ctx
	env.Runner.AfterBlockedStart = func(string) { cancel() }
	_, h, err := env.Couch.launchTrackedThread(in)
	var unconfirmed *RecoveryLaunchUnconfirmed
	if !errors.Is(err, context.Canceled) || errors.As(err, &unconfirmed) || h == nil {
		t.Fatalf("pre-ack failure incorrectly uncertain: %v", err)
	}
	if h.Alive() || env.Runner.Child(h.ID()).ExecCount != 0 {
		t.Fatal("definite pre-ack cancellation did not reap blocked helper")
	}
}

func TestRecoveryFinalRegistrationLookPreservesPaneBirthGate(t *testing.T) {
	env, in := recoveryTrackedFixture(t)
	in.Fresh, in.Resume = false, true
	probes := 0
	env.Artifacts.BeforePairSession = func(ThreadAddress) error { probes++; return errors.New("observation unavailable") }
	baseline := PaneMarks{}
	if err := env.Couch.reobserveRecoveryRegistration(context.Background(), in.Thread, in, &baseline); err == nil {
		t.Fatal("unborn terminal treated as registered")
	}
	if probes != 0 {
		t.Fatal("final recovery look probed terminal before pane birth")
	}
	env.Artifacts.SetPaneSidecar(in.Thread.Address, "codex")
	_ = env.Couch.reobserveRecoveryRegistration(context.Background(), in.Thread, in, &baseline)
	if probes != 1 {
		t.Fatalf("birth did not permit one exact terminal observation: %d", probes)
	}
}
