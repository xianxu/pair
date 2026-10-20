package couchcmd

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDrainQueueExecutableExactArgv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sdlc")
	log := filepath.Join(dir, "argv")
	t.Setenv("DRAIN_TEST_ARGV", log)
	t.Setenv("DRAIN_TEST_STATUS", drainQueueFixture)
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$DRAIN_TEST_ARGV\"\nif [ \"$2\" = status ]; then printf '%s' \"$DRAIN_TEST_STATUS\"; fi\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	q := sdlcDrainQueue{workspace: dir, executable: path}
	ctx := context.Background()
	for _, tc := range []struct {
		call func() error
		want []string
	}{
		{func() error {
			return q.Pause(ctx, "couch-drain:id", "Couch drain id: reason with $() and spaces", false)
		}, []string{"queue", "pause", "--reason", "Couch drain id: reason with $() and spaces", "--owner", "couch-drain:id"}},
		{func() error { return q.Pause(ctx, "couch-drain:id", "Couch drain id: reason", true) }, []string{"queue", "pause", "--reason", "Couch drain id: reason", "--owner", "couch-drain:id", "--cancel-running"}},
		{func() error { return q.Resume(ctx, "couch-drain:id") }, []string{"queue", "resume", "--owner", "couch-drain:id"}},
		{func() error { _, err := q.Status(ctx); return err }, []string{"queue", "status", "--json"}},
	} {
		if err := tc.call(); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n"); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("argv %q want %q", got, tc.want)
		}
	}
}

func TestDrainQueueUnavailableAndBounded(t *testing.T) {
	dir := t.TempDir()
	q := sdlcDrainQueue{workspace: dir, executable: filepath.Join(dir, "absent")}
	if got, err := q.Status(context.Background()); err == nil || got.Complete || got.State != "unknown" {
		t.Fatalf("unsupported became success: %+v %v", got, err)
	}
	q.executable = filepath.Join(dir, "slow")
	if err := os.WriteFile(q.executable, []byte("#!/bin/sh\nsleep 20\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := q.Status(ctx); err == nil {
		t.Fatal("timeout succeeded")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("cleanup unbounded")
	}
}

func TestDrainQueueFakeOwnerIsolationAndLostReplies(t *testing.T) {
	f := NewFakeDrainQueue()
	ctx := context.Background()
	f.Enqueue(QueueActivity{ID: "a", Workspace: "pair:1", Kind: "suite", State: "running"})
	f.Enqueue(QueueActivity{ID: "b", Workspace: "pair:2", Kind: "merge", State: "running"})
	if !f.AdmitNext() {
		t.Fatal("no admission")
	}
	f.LoseNextReply = true
	if err := f.Pause(ctx, "one", "first", false); err == nil {
		t.Fatal("reply not lost")
	}
	snap, _ := f.Status(ctx)
	if len(snap.PauseOwners) != 1 || len(snap.Active) != 1 || f.AdmitNext() {
		t.Fatalf("pause ineffective %+v", snap)
	}
	if err := f.Pause(ctx, "two", "second", true); err != nil {
		t.Fatal(err)
	}
	snap, _ = f.Status(ctx)
	if snap.Active[0].State != "interrupting" {
		t.Fatal("fast did not record interruption")
	}
	f.Finish("a")
	if err := f.Resume(ctx, "one"); err != nil {
		t.Fatal(err)
	}
	if f.AdmitNext() {
		t.Fatal("released other's hold")
	}
	f.LoseNextReply = true
	if err := f.Resume(ctx, "two"); err == nil {
		t.Fatal("resume reply not lost")
	}
	snap, _ = f.Status(ctx)
	if snap.State != "running" || len(snap.PauseOwners) != 0 || !f.AdmitNext() {
		t.Fatalf("resume not reconciled %+v", snap)
	}
	snap, _ = f.Status(ctx)
	if snap.Active[0].ID != "b" {
		t.Fatal("FIFO lost")
	}
	f.ObservationError = context.DeadlineExceeded
	if snap, err := f.Status(ctx); err == nil || snap.Complete {
		t.Fatal("failed observation complete")
	}
}

func TestDrainQueueExecutableRejectsUnboundedOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sdlc")
	// The fake emits more than the protocol's budget, not machine queue state.
	script := "#!/bin/sh\nhead -c 1048577 /dev/zero\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	q := sdlcDrainQueue{workspace: dir, executable: path}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if s, err := q.Status(ctx); err == nil || s.Complete || s.State != "unknown" {
		t.Fatalf("oversized observation accepted %+v %v", s, err)
	}
}

func TestDrainQueueCombinedOutputBudget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sdlc")
	script := "#!/bin/sh\nhead -c 700000 /dev/zero\nhead -c 700000 /dev/zero >&2\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	q := sdlcDrainQueue{workspace: dir, executable: path}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := q.Pause(ctx, "one", "x", false); err == nil {
		t.Fatal("accepted more than 1MiB combined stdout/stderr")
	}
}

func TestDrainQueueUnknownDiagnosticBounded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sdlc")
	script := "#!/bin/sh\nhead -c 7000 /dev/zero >&2\nexit 1\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	q := sdlcDrainQueue{workspace: dir, executable: path}
	s, err := q.Status(context.Background())
	if err == nil || len(s.Detail) > 4096 {
		t.Fatalf("unknown detail exceeds text bound: %d", len(s.Detail))
	}
}

func TestDrainQueueFakeFastMatchesRealKindsAndReplay(t *testing.T) {
	f := NewFakeDrainQueue()
	ctx := context.Background()
	for _, a := range []QueueActivity{
		{ID: "suite", Kind: "suite", State: "running"},
		{ID: "merge", Kind: "merge", State: "running"},
		{ID: "merge", Kind: "landing", State: "running"},
		{ID: "landing", Kind: "landing", State: "running"},
		{ID: "combined", Kind: "suite", State: "running"},
		{ID: "combined", Kind: "merge", State: "running"},
	} {
		f.Enqueue(a)
		if !f.AdmitNext() {
			t.Fatal("no admission")
		}
	}
	if err := f.Pause(ctx, "owner", "original", true); err != nil {
		t.Fatal(err)
	}
	if err := f.Pause(ctx, "owner", "changed replay", false); err != nil {
		t.Fatal(err)
	}
	s, err := f.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.PauseOwners) != 1 || s.PauseOwners[0].Reason != "original" || !s.PauseOwners[0].CancelRunning {
		t.Fatalf("replay changed owner: %+v", s.PauseOwners)
	}
	for _, a := range s.Active {
		expected := "running"
		if a.ID == "suite" || a.ID == "combined" {
			expected = "interrupting"
		}
		if a.State != expected {
			t.Errorf("%s/%s state=%s, want %s", a.ID, a.Kind, a.State, expected)
		}
	}
	if err := f.Resume(ctx, "owner"); err != nil {
		t.Fatal(err)
	}
	after, _ := f.Status(ctx)
	for i, a := range after.Active {
		if a.State != s.Active[i].State {
			t.Fatal("resume restarted or interrupted active work")
		}
	}
}
