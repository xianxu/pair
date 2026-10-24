package couchcore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestStartupLeavesMissingHubAndOtherRetainedRecordsUntouched(t *testing.T) {
	for _, mode := range []string{"empty", "parked", "broken-record", "pre-upgrade-live", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			if mode != "empty" {
				address := detachedThreadAt(t, env, "/other", mode)
				if mode == "parked" || mode == "pre-upgrade-live" {
					current, err := env.Couch.Threads.GetThread(address)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := env.Couch.Threads.updateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
						if mode == "parked" {
							markActionableParked(next, time.Unix(100, 0).UTC())
						} else {
							next.LatestLaunchProfile = nil
							next.Layout = ""
						}
						return nil
					}); err != nil {
						t.Fatal(err)
					}
					if mode == "parked" {
						env.Artifacts.SetDetachedSession(address, "")
						env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), false)
					}
				}
				if mode == "broken-record" || mode == "mixed" {
					if err := os.WriteFile(env.Couch.Threads.recordPath(address), []byte("broken"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, err := env.Couch.Threads.PreviewSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
			if err != nil || result.Handle != nil || len(env.Runner.Ops) != 0 {
				t.Fatalf("startup: %+v err=%v ops=%v", result, err, env.Runner.Ops)
			}
			after, err := env.Couch.Threads.PreviewSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			for _, old := range before.Records {
				found := false
				for _, next := range after.Records {
					if old.Address == next.Address {
						found = true
						if !reflect.DeepEqual(old, next) {
							t.Fatal("startup changed unrelated retained record")
						}
					}
				}
				if !found {
					t.Fatal("startup removed unrelated retained record")
				}
			}
			if !reflect.DeepEqual(before.Unreadable, after.Unreadable) {
				t.Fatal("startup changed unreadable retained records")
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
