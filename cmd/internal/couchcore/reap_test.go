package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// fakeOrphanReaper records which servers it was asked to reap; reaping makes
// the session read absent, as the real tree kill plus record cleanup does.
type fakeOrphanReaper struct {
	reaped    []launcher.SessionServerIdentity
	artifacts *FakeThreadArtifactCollisionChecker
	address   ThreadAddress
	err       error
	hook      func() // runs as the reap happens, before its effect
}

func (f *fakeOrphanReaper) ReapOrphan(_ context.Context, server launcher.SessionServerIdentity, _, _ string) error {
	f.reaped = append(f.reaped, server)
	if f.hook != nil {
		f.hook()
	}
	if f.err == nil && f.artifacts != nil {
		f.artifacts.SetSessionPresence(f.address, SessionObservation{State: SessionAbsent})
	}
	return f.err
}

// orphanedEnv is a thread whose launcher died with couch and whose server
// survived, orphaned.
func orphanedEnv(t *testing.T) (*testEnv, ThreadAddress, launcher.SessionServerIdentity, *fakeOrphanReaper) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(first.PID)
	if err := env.Couch.Forget("/repo", first.ID); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	server := launcher.SessionServerIdentity{PID: 9090, Identity: "t9090", Session: "📁repo-1"}
	env.Artifacts.SetSessionPresence(first.Thread, SessionObservation{State: SessionOrphaned, Orphan: &server})
	reaper := &fakeOrphanReaper{artifacts: env.Artifacts, address: first.Thread}
	env.Couch.Reaper = reaper
	env.Couch.sleep = func(time.Duration) {}
	return env, first.Thread, server, reaper
}

// Reap ends an orphaned thread's server tree; the thread then reads like any
// other thread whose session ended, and resume is its ordinary next step.
func TestReapEndsAnOrphanedServer(t *testing.T) {
	env, address, server, reaper := orphanedEnv(t)
	result, err := env.Couch.Reap(context.Background(), ReapTarget{Address: address})
	if err != nil {
		t.Fatal(err)
	}
	if len(reaper.reaped) != 1 || reaper.reaped[0] != server || result.Server != server {
		t.Fatalf("reaped %+v, result %+v", reaper.reaped, result)
	}
	rows, err := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Address == address && row.Reason == ReasonOrphanedServer {
			t.Fatalf("still orphaned after reap: %+v", row)
		}
	}
}

// A single snapshot is provisional: a server that is still starting is in ps
// before its socket exists. Reap re-observes after an interval and refuses if
// the orphan verdict did not hold for the same server identity.
func TestReapRequiresTheOrphanToHoldAcrossTwoObservations(t *testing.T) {
	for _, tc := range []struct {
		name  string
		later SessionObservation
	}{
		{"socket appeared: it was starting", SessionObservation{State: SessionPresent}},
		{"a different server now", SessionObservation{State: SessionOrphaned, Orphan: &launcher.SessionServerIdentity{PID: 9090, Identity: "other", Session: "📁repo-1"}}},
		{"could not observe", SessionObservation{State: SessionUnresolved}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, address, _, reaper := orphanedEnv(t)
			env.Couch.sleep = func(time.Duration) { env.Artifacts.SetSessionPresence(address, tc.later) }
			if _, err := env.Couch.Reap(context.Background(), ReapTarget{Address: address}); err == nil {
				t.Fatal("reaped on a verdict that did not hold")
			}
			if len(reaper.reaped) != 0 {
				t.Fatalf("signalled anyway: %+v", reaper.reaped)
			}
		})
	}
}

// Reap acts only on orphans: a parked or live thread is refused untouched.
func TestReapRefusesAThreadThatIsNotOrphaned(t *testing.T) {
	env, address, _, reaper := orphanedEnv(t)
	env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionAbsent})
	_, err := env.Couch.Reap(context.Background(), ReapTarget{Address: address})
	var refusal *ReapRefusal
	if !errors.As(err, &refusal) || len(reaper.reaped) != 0 {
		t.Fatalf("err %v, reaped %+v", err, reaper.reaped)
	}
}

// ActorActions offers reap, and only reap, on an orphaned row.
func TestActorActionsOffersReapOnAnOrphan(t *testing.T) {
	got := ActorActions(ActorRowFacts{State: ThreadUnusable, Reason: ReasonOrphanedServer})
	if len(got) != 1 || got[0] != "reap" {
		t.Fatalf("ActorActions(orphan) = %v", got)
	}
}
