package couchcore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
)

func TestStartupObservesWithoutReplayingOrLaunching(t *testing.T) {
	for _, mode := range []string{"empty", "parked", "pending-journal", "broken-record", "pre-upgrade-live", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			if mode != "empty" {
				r := seedStartupParked(t, env, "couch-0000000000000001", "/repo")
				if mode == "pre-upgrade-live" || mode == "mixed" {
					env.Artifacts.SetDetachedSession(r.Address, "legacy-no-policy")
				}
				if mode == "mixed" {
					broken := seedStartupParked(t, env, "couch-0000000000000002", "/other")
					if err := os.WriteFile(env.Couch.Threads.recordPath(broken.Address), []byte("broken"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "broken-record" {
					if err := os.WriteFile(env.Couch.Threads.recordPath(r.Address), []byte("broken"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if mode == "pending-journal" {
				if err := os.WriteFile(env.Couch.Threads.journalPath(), []byte("pending incomplete journal"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(env.Couch.Threads.manifestPath())
			result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
			if err != nil || result.Handle != nil || len(env.Runner.Ops) != 0 {
				t.Fatalf("startup acted/refused: %+v err=%v ops=%v", result, err, env.Runner.Ops)
			}
			after, _ := os.ReadFile(env.Couch.Threads.manifestPath())
			if !bytes.Equal(before, after) {
				t.Fatal("startup wrote manifest")
			}
			if mode == "pending-journal" {
				b, _ := os.ReadFile(env.Couch.Threads.journalPath())
				if string(b) != "pending incomplete journal" {
					t.Fatal("startup changed journal")
				}
			}
		})
	}
}

func TestConstructorLeavesPendingJournalUntouched(t *testing.T) {
	ns := testCouchNamespace(t)
	s := NewThreadStore(ns)
	if err := os.MkdirAll(s.root, 0700); err != nil {
		t.Fatal(err)
	}
	raw := []byte("interrupted journal is not constructor authority")
	if err := os.WriteFile(s.journalPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := New(ns, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(), NewStore(ns.Dir()), FixedClock{}, NewFixedIDGen("id"), newIncrementingEntropy(), NewFakeThreadArtifactCollisionChecker())
	if err != nil || c == nil {
		t.Fatalf("constructor blocked on slot journal: %v", err)
	}
	after, _ := os.ReadFile(s.journalPath())
	if !bytes.Equal(raw, after) {
		t.Fatal("constructor changed journal")
	}
}

func TestCoordinatedStoreDefersEnrollmentUntilExplicitMutation(t *testing.T) {
	s, coordinator := retentionStore(t)
	if _, err := coordinator.ReadRegistry(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("constructor registered store: %v", err)
	}
	if _, err := s.PreviewSnapshot(); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.ReadRegistry(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview registered store: %v", err)
	}
	r := validThreadRecord(t)
	r.StartingPath, r.WorkingPath = s.namespace.Dir(), s.namespace.Dir()
	if _, err := s.CreateThread(r); err != nil {
		t.Fatal(err)
	}
	registry, err := coordinator.ReadRegistry()
	if err != nil || len(registry.Stores) != 1 {
		t.Fatalf("explicit mutation did not enroll store: %+v %v", registry, err)
	}
}
