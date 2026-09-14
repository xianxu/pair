package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func retentionStore(t *testing.T) (*ThreadStore, *storagegc.Coordinator) {
	t.Helper()
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c.Now = func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC) }
	s, err := NewCoordinatedThreadStore(testCouchNamespace(t), c)
	if err != nil {
		t.Fatal(err)
	}
	return s, c
}

func TestRetentionArchiveGraceIncludesUndecodableRecords(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprint(corrupt), func(t *testing.T) {
			s, c := retentionStore(t)
			r := archivableThread(t, s, "couch-0000000000000001")
			if corrupt {
				if err := os.WriteFile(s.recordPath(r.Address), []byte("future record"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.ArchiveThread(r.Address); err != nil {
				t.Fatal(err)
			}
			a, err := readTestArchiveGrace(s, r.Address)
			if err != nil || !a.ArchivedAt.Equal(c.Now()) || len(a.RecordHash) != 64 {
				t.Fatalf("archive %+v %v", a, err)
			}
			c.Now = func() time.Time { return a.ArchivedAt.Add(time.Hour) }
			if err := s.ArchiveThread(r.Address); !errors.Is(err, ErrThreadNotFound) {
				t.Fatalf("archive retry %v", err)
			}
			again, err := readTestArchiveGrace(s, r.Address)
			if err != nil || again.ArchivedAt != a.ArchivedAt {
				t.Fatal("retry changed archive clock")
			}
			if err := s.RestoreThread(r.Address); err != nil {
				t.Fatal(err)
			}
			snapshot, err := s.Snapshot()
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Records)+len(snapshot.Unreadable) != 1 {
				t.Fatalf("restore lost visible reference %+v", snapshot)
			}
			if _, err := os.Stat(s.archiveGracePath(r.Address)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("restore left grace sidecar")
			}
		})
	}
}

func TestRetentionSnapshotProtectsUnreadableAndBlocksPendingRecovery(t *testing.T) {
	s, c := retentionStore(t)
	r := archivableThread(t, s, "couch-0000000000000001")
	if err := os.WriteFile(s.recordPath(r.Address), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	var stale *storagegc.Locked
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		stale = l
		snap, err := s.RetentionSnapshot(l)
		if err != nil {
			return err
		}
		if len(snap.Visible) != 1 || snap.Visible[0] != r.Address {
			t.Fatalf("lost unreadable reference %+v", snap)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RetentionSnapshot(stale); err == nil {
		t.Fatal("expired lock accepted")
	}
	s.hooks.AfterJournal = func() error { return errors.New("crash") }
	if err := s.ArchiveThread(r.Address); err == nil {
		t.Fatal("expected fault")
	}
	before, err := os.ReadFile(s.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		_, err := s.RetentionSnapshot(l)
		if err == nil {
			t.Fatal("pending recovery accepted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.journalPath())
	if string(before) != string(after) {
		t.Fatal("preview mutated journal")
	}
}

func TestRetentionArchiveAndRestoreRecoverEachJournalStep(t *testing.T) {
	for _, operation := range []string{"archive", "restore"} {
		for fault := -1; fault < 4; fault++ {
			t.Run(fmt.Sprintf("%s/%d", operation, fault), func(t *testing.T) {
				s, _ := retentionStore(t)
				r := archivableThread(t, s, "couch-0000000000000001")
				if operation == "restore" {
					if err := s.ArchiveThread(r.Address); err != nil {
						t.Fatal(err)
					}
				}
				crash := errors.New("crash")
				if fault < 0 {
					s.hooks.AfterJournal = func() error { return crash }
				} else {
					s.hooks.AfterTarget = func(n int) error {
						if n == fault {
							return crash
						}
						return nil
					}
				}
				var err error
				if operation == "archive" {
					err = s.ArchiveThread(r.Address)
				} else {
					err = s.RestoreThread(r.Address)
				}
				if !errors.Is(err, crash) {
					t.Fatalf("fault not reached: %v", err)
				}
				s.hooks = threadStoreHooks{}
				if err := s.RecoverStoreJournal(); err != nil {
					t.Fatal(err)
				}
				snap, err := s.Snapshot()
				if err != nil {
					t.Fatal(err)
				}
				if operation == "archive" {
					if len(snap.Records) != 0 {
						t.Fatal("archive still visible")
					}
					if _, err := readTestArchiveGrace(s, r.Address); err != nil {
						t.Fatal(err)
					}
				} else {
					if len(snap.Records) != 1 {
						t.Fatal("restore missing")
					}
					if _, err := os.Stat(s.archiveGracePath(r.Address)); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("grace survived restore")
					}
				}
			})
		}
	}
}

func TestRetentionStrictArchiveClocks(t *testing.T) {
	for _, kind := range []string{"missing", "malformed", "unknown", "future", "replacement", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			s, c := retentionStore(t)
			r := archivableThread(t, s, "couch-0000000000000001")
			if err := s.ArchiveThread(r.Address); err != nil {
				t.Fatal(err)
			}
			path := s.archiveGracePath(r.Address)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing":
				err = os.Remove(path)
			case "malformed":
				err = os.WriteFile(path, []byte("bad"), 0600)
			case "unknown":
				err = os.WriteFile(path, append([]byte(`{"unknown":true,`), raw[1:]...), 0600)
			case "future":
				var clock archiveGrace
				if err = json.Unmarshal(raw, &clock); err != nil {
					t.Fatal(err)
				}
				clock.ArchivedAt = c.Now().Add(time.Hour)
				raw, _ = json.Marshal(clock)
				err = os.WriteFile(path, raw, 0600)
			case "replacement":
				err = os.WriteFile(s.archivePath(r.Address), []byte("replaced"), 0600)
			case "symlink":
				target := filepath.Join(t.TempDir(), "clock")
				if err = os.WriteFile(target, raw, 0600); err == nil {
					err = os.Remove(path)
				}
				if err == nil {
					err = os.Symlink(target, path)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := readTestArchiveGrace(s, r.Address); err == nil {
				t.Fatal("unsafe clock accepted")
			}
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
				snap, err := s.RetentionSnapshot(l)
				if err != nil {
					return err
				}
				if len(snap.Archives) != 1 || snap.Archives[0].ClockError == "" {
					t.Fatalf("clock error not carried %+v", snap)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRetentionPreviewDoesNotCreateStoreLock(t *testing.T) {
	s, c := retentionStore(t)
	if err := os.MkdirAll(s.root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		_, err := s.RetentionSnapshot(l)
		if err == nil {
			t.Fatal("partial uninitialized store needs blocker")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.root, "store.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("preview created store lock")
	}
}

func TestRetentionCouchConstructorAttachesCoordinator(t *testing.T) {
	ns := testCouchNamespace(t)
	root := filepath.Join(t.TempDir(), "pair-data")
	artifacts := constructorLifecycleArtifacts{FakeThreadArtifactCollisionChecker: NewFakeThreadArtifactCollisionChecker(), dataDir: root}
	couch, err := New(ns, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(), NewStore(ns.Dir()), FixedClock{T: time.Now()}, NewFixedIDGen("id"), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if couch.Threads.coordinator == nil {
		t.Fatal("production constructor bypassed retention")
	}
}

func TestRetentionReadAdapterDoesNotRegisterNamespace(t *testing.T) {
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	namespace := testCouchNamespace(t)
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		snapshot, err := ReadStoreRetention(namespace, c, l)
		if err != nil {
			return err
		}
		if len(snapshot.Visible) != 0 || len(snapshot.Archives) != 0 {
			t.Fatal("empty namespace not empty")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(namespace.Dir())
	if err != nil || len(entries) != 0 {
		t.Fatalf("read adapter initialized namespace: %v %v", entries, err)
	}
}

func TestRetentionMembershipWaitsForRootCoordination(t *testing.T) {
	s, c := retentionStore(t)
	done := make(chan error, 1)
	started := make(chan struct{})
	err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		go func() {
			close(started)
			record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
			_, err := s.CreateThread(record)
			done <- err
		}()
		<-started
		select {
		case err := <-done:
			t.Fatalf("mutation escaped root lock: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
		_, err := s.RetentionSnapshot(l)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("mutation did not resume")
	}
}

func readTestArchiveGrace(s *ThreadStore, address ThreadAddress) (ArchiveRetention, error) {
	var evidence ArchiveRetention
	err := s.withLock(func() error { var err error; evidence, err = s.readArchiveGraceLocked(address); return err })
	return evidence, err
}

func TestRetentionExplicitResumeTouchesButBackgroundReattachDoesNot(t *testing.T) {
	for _, background := range []bool{false, true} {
		t.Run(fmt.Sprint(background), func(t *testing.T) {
			env, address := warmDetachedThread(t)
			coordinator, err := storagegc.NewCoordinator(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			before := env.Now
			coordinator.Now = func() time.Time { return before }
			env.Couch.Threads.coordinator = coordinator
			owner, err := artifactpath.NewStorageOwner(coordinator.Root, address.RepoScope, string(address.Tag))
			if err != nil {
				t.Fatal(err)
			}
			if err := coordinator.Initialize(context.Background(), owner); err != nil {
				t.Fatal(err)
			}
			after := before.Add(time.Hour)
			coordinator.Now = func() time.Time { return after }
			env.Runner.AfterAcknowledge = func(string) error {
				env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), true)
				return nil
			}
			_, handle, err := env.Couch.ResumeContextWith(context.Background(), address, ResumeOptions{WarmOnly: background})
			if err != nil || handle == nil {
				t.Fatalf("resume %v", err)
			}
			state, err := coordinator.ReadOwner(owner)
			if err != nil {
				t.Fatal(err)
			}
			want := after
			if background {
				want = before
			}
			if !state.Activity.LastUse.Equal(want) || len(state.Processes) != 0 {
				t.Fatalf("resume use/lifetime %+v", state)
			}
			expectedEnv := ""
			if background {
				expectedEnv = "1"
			}
			if got := childEnvValue(env.Runner.Child(handle.ID()).Env, "PAIR_RETENTION_BACKGROUND"); got != expectedEnv {
				t.Fatalf("background context %q", got)
			}
		})
	}
}
