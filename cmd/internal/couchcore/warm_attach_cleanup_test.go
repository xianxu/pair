package couchcore

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestPublicWarmAttachCleanupPreservesIndependentLifecycle(t *testing.T) {
	for _, shape := range []string{"start this couch is driving", "park in flight whose session is still up", "live and hosted"} {
		for _, action := range []string{"detach", "leave-detach", "abort"} {
			t.Run(shape+"/"+action, func(t *testing.T) {
				var record ThreadRecord
				for _, tc := range everyThreadShape(t) {
					if tc.name == shape {
						record = tc.record
						break
					}
				}
				env, before, session := readonlyWarmFixture(t, record, Live)
				actor, h := assertReadonlyWarmAttach(t, env, before, session)
				env.Proc.Set(h.PID(), h.Identity())
				env.Proc.DiesOn[h.PID()] = TermSignal
				env.Runner.SetDiesOn(h.ID(), TermSignal, 0)
				env.Proc.OnSignal = func(pid int, _ os.Signal) {
					if pid == h.PID() {
						env.Runner.SetExited(h.ID(), 0)
					}
				}
				switch action {
				case "detach":
					if _, err := env.Couch.Detach(context.Background(), before.Address); err != nil {
						t.Fatal(err)
					}
				case "leave-detach":
					if _, err := env.Couch.Leave(context.Background(), LeaveDetach); err != nil {
						t.Fatal(err)
					}
				case "abort":
					cause := errors.New("console rejected attach")
					err := env.Couch.AbortStarted(context.Background(), StartResult{Record: actor, Handle: h}, cause)
					if !errors.Is(err, cause) {
						t.Fatalf("lost console refusal: %v", err)
					}
				}
				if h.Alive() {
					t.Fatal("attachment helper survived cleanup")
				}
				for _, inc := range before.Incarnations {
					if len(env.Proc.Signals[inc.PID]) != 0 {
						t.Fatalf("signalled independent incarnation %d", inc.PID)
					}
					if inc.Start != nil && len(env.Proc.Signals[inc.Start.OwnerPID]) != 0 {
						t.Fatal("signalled independent claim owner")
					}
				}
				for _, remaining := range env.Couch.actorRegistry().Records() {
					if remaining.ID == actor.ID {
						t.Fatal("helper registry entry survived cleanup")
					}
				}
				assertWarmRecordUnchanged(t, env, before)
				observed, err := env.Artifacts.DetachedSessions(context.Background(), []DetachedCandidate{{Address: before.Address, SessionName: session}})
				if err != nil || len(observed) != 1 || observed[0].SessionName != session {
					t.Fatalf("cleanup lost borrowed session: %v %v", observed, err)
				}
			})
		}
	}
}

func TestPublicWarmAttachRestoredRegistryIsNotHelperOwnership(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	actor, h := assertReadonlyWarmAttach(t, env, before, session)
	env.Proc.Set(h.PID(), h.Identity())
	restored, err := New(env.Couch.Namespace, env.Runner, env.Couch.Path, env.Git, env.Proc, NewStore(env.Dir), env.Couch.Clock, NewFixedIDGen("restored"), newIncrementingEntropy(), env.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range restored.actorRegistry().Records() {
		if r.ID == actor.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("fixture failed to restore persisted actor")
	}
	for _, tc := range []struct {
		c    *Couch
		want ActionableThreadState
	}{{env.Couch, ThreadLive}, {restored, ThreadDetached}} {
		rows, err := tc.c.ActionableThreadInventory(nil)
		if err != nil || len(rows) != 1 || rows[0].State != tc.want {
			t.Fatalf("local handle inventory: %+v %v, want %s", rows, err, tc.want)
		}
	}
	if _, err := restored.Detach(context.Background(), before.Address); err == nil {
		t.Fatal("restored registry authorized helper detach")
	}
	if _, err := restored.Leave(context.Background(), LeaveDetach); err != nil {
		t.Fatal(err)
	}
	if len(env.Proc.Signals[h.PID()]) != 0 || !h.Alive() {
		t.Fatal("restored registry authorized signal")
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestPublicWarmAttachFailuresPreserveIndependentLifecycle(t *testing.T) {
	for _, phase := range []string{"before-ack", "after-ack"} {
		t.Run(phase, func(t *testing.T) {
			var record ThreadRecord
			for _, tc := range everyThreadShape(t) {
				if tc.name == "start this couch is driving" {
					record = tc.record
					break
				}
			}
			env, before, _ := readonlyWarmFixture(t, record, Live)
			cause := errors.New("injected attach failure")
			if phase == "before-ack" {
				env.Runner.BeforeAcknowledge = func(string) error { return cause }
			} else {
				env.Runner.AfterAcknowledge = func(string) error { return cause }
			}
			_, h, err := env.Couch.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
			if !errors.Is(err, cause) || h == nil {
				t.Fatalf("expected failed helper: %v %v", h, err)
			}
			if h.Alive() {
				t.Fatal("failed helper survived")
			}
			if len(env.Couch.actorRegistry().Records()) != 0 {
				t.Fatal("failed helper registered actor")
			}
			if len(env.Proc.Signals[4242]) != 0 {
				t.Fatal("failure signalled independent start owner")
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}
