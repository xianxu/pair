package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type countedFailureSnapshotStore struct {
	LaunchFailureStorage
	reads int
}

func (s *countedFailureSnapshotStore) Read() ([]LaunchFailure, error) {
	s.reads++
	return s.LaunchFailureStorage.Read()
}
func TestLifecycleFailureProjectionLatestUnresolved(t *testing.T) {
	at := time.Unix(100, 0)
	superseded := at
	rows := []ActionableThreadSummary{{StartingPath: "/a", State: ThreadDetached}, {StartingPath: "/b", State: ThreadLive}}
	h := []LaunchFailure{{StartingPath: "/a", Nonce: "old", FailedAt: at}, {StartingPath: "/a", Nonce: "new", FailedAt: at.Add(time.Second)}, {StartingPath: "/b", Nonce: "hidden", SupersededAt: &superseded}}
	got := ProjectLifecycleFailures(rows, h, "")
	if got[0].Failure == nil || got[0].Failure.Nonce != "new" || got[1].Failure != nil || got[0].State != ThreadDetached {
		t.Fatalf("projection: %+v", got)
	}
	got[0].Failure.Nonce = "changed"
	if h[1].Nonce != "new" {
		t.Fatal("projection aliases retained evidence")
	}
}
func TestLifecycleFailureSnapshotReplacementAndCorruption(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &LaunchFailureStore{path: filepath.Join(dir, "failures.json")}
	count := &countedFailureSnapshotStore{LaunchFailureStorage: s}
	r := NewLifecycleFailureRecorder(count, time.Now, nil)
	f := LaunchFailure{Address: ThreadAddress{RepoScope: "scope", Tag: "tag"}, Nonce: "one", FailedAt: time.Unix(100, 0)}
	if err := s.Save(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	for range 16 {
		if _, err := r.Snapshot(); err != nil {
			t.Fatal(err)
		}
	}
	if count.reads != 1 {
		t.Fatalf("unchanged reads=%d", count.reads)
	}
	f.Nonce = "two"
	if err := s.Save(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if h, err := r.Snapshot(); err != nil || len(h) != 2 {
		t.Fatalf("replacement=%+v %v", h, err)
	}
	if count.reads != 2 {
		t.Fatalf("replacement reads=%d", count.reads)
	}
	if err := os.WriteFile(s.Path(), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Snapshot(); err == nil {
		t.Fatal("corruption treated as absence")
	}
}

func TestLifecycleFailureProjectionScaledRowsAndHistory(t *testing.T) {
	for _, n := range []int{8, 16} {
		rows := make([]ActionableThreadSummary, n)
		history := make([]LaunchFailure, n)
		for i := range n {
			path := filepath.Join("/repo", string(rune('a'+i)))
			rows[i] = ActionableThreadSummary{StartingPath: path, State: ThreadDetached}
			history[i] = LaunchFailure{StartingPath: path, Nonce: path, FailedAt: time.Unix(int64(i), 0)}
		}
		projected := ProjectLifecycleFailures(rows, history, "")
		for i := range n {
			if projected[i].Failure == nil || projected[i].Failure.Nonce != history[i].Nonce || projected[i].State != rows[i].State {
				t.Fatalf("n=%d row=%d projection=%+v", n, i, projected[i])
			}
		}
		unavailable := ProjectLifecycleFailures(rows, nil, "corrupt diagnostics")
		for i := range n {
			if unavailable[i].DiagnosticError == "" || unavailable[i].State != rows[i].State {
				t.Fatalf("unavailable changed authority: %+v", unavailable[i])
			}
		}
	}
}

func TestLifecycleFailureSnapshotRejectsSymlinkedParentOnCacheHit(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "safe")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s := &LaunchFailureStore{path: filepath.Join(dir, "failures.json")}
	if err := s.Save(t.Context(), LaunchFailure{Address: ThreadAddress{RepoScope: "scope", Tag: "tag"}, Nonce: "one", FailedAt: time.Unix(100, 0)}); err != nil {
		t.Fatal(err)
	}
	r := NewLifecycleFailureRecorder(s, time.Now, nil)
	if _, err := r.Snapshot(); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "moved")
	if err := os.Rename(dir, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Snapshot(); err == nil {
		t.Fatal("cached payload bypassed symlink-parent refusal")
	}
}

func TestLifecycleFailureInventoryProjectionReadsOncePerRefresh(t *testing.T) {
	for _, n := range []int{8, 16} {
		env := newTestEnv(t, "/repo")
		fake := NewFakeLaunchFailureStore()
		count := &countedFailureSnapshotStore{LaunchFailureStorage: fake}
		env.Couch.LaunchFailures = count
		rows := make([]ActionableThreadSummary, n)
		for i := range n {
			path := filepath.Join("/repo", string(rune('a'+i)))
			rows[i] = ActionableThreadSummary{StartingPath: path}
			if err := fake.Save(t.Context(), LaunchFailure{StartingPath: path, Nonce: path, FailedAt: time.Unix(100, 0)}); err != nil {
				t.Fatal(err)
			}
		}
		got := env.Couch.applyLifecycleFailures(rows)
		if count.reads != 1 {
			t.Fatalf("n=%d refresh payload reads=%d", n, count.reads)
		}
		for i := range n {
			if got[i].Failure == nil {
				t.Fatalf("n=%d row=%d missing", n, i)
			}
		}
	}
}

func TestLifecycleFailureSnapshotConcurrentReadersSharePayload(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &LaunchFailureStore{path: filepath.Join(dir, "failures.json")}
	if err := s.Save(t.Context(), LaunchFailure{StartingPath: dir, Nonce: "one", FailedAt: time.Unix(100, 0), Output: []byte("owned")}); err != nil {
		t.Fatal(err)
	}
	count := &countedFailureSnapshotStore{LaunchFailureStorage: s}
	r := NewLifecycleFailureRecorder(count, time.Now, nil)
	finished := make(chan error, 16)
	for range 16 {
		go func() {
			h, err := r.Snapshot()
			if err == nil {
				if len(h) != 1 || string(h[0].Output) != "owned" {
					err = errors.New("snapshot not independently owned")
				} else {
					h[0].Output[0] = 'x'
				}
			}
			finished <- err
		}()
	}
	for range 16 {
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	}
	if count.reads != 1 {
		t.Fatalf("concurrent payload reads=%d", count.reads)
	}
}
