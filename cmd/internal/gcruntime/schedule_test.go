package gcruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func scheduleFixture(t *testing.T) (*Service, artifactpath.StorageOwner) {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	o, err := artifactpath.NewStorageOwner(s.Collector.Coordinator.Root, "", "tag")
	if err != nil {
		t.Fatal(err)
	}
	s.Collector.LegacyRoot = nil
	s.Collector.Legacy = func(context.Context, artifactpath.StorageOwner) (storagegc.Liveness, error) {
		return storagegc.ProcessDead, nil
	}
	if err := os.WriteFile(filepath.Join(o.Directory(), "draft-tag.md"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	return s, o
}
func TestScheduledBatchBeforeMigrationOnlyInitializesGrace(t *testing.T) {
	s, o := scheduleFixture(t)
	scheduler := storagegc.Scheduler{Coordinator: s.Collector.Coordinator, Batch: s.Batch, Limit: 1}
	ran, err := scheduler.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("run=%v %v", ran, err)
	}
	if _, err := s.Collector.Coordinator.ReadOwner(o); err != nil {
		t.Fatal("missing grace", err)
	}
	if _, err := os.Stat(filepath.Join(o.Directory(), "draft-tag.md")); err != nil {
		t.Fatal("deleted before migration", err)
	}
	if ran, err := scheduler.RunOnce(context.Background()); err != nil || ran {
		t.Fatalf("completed run did not rate limit: %v %v", ran, err)
	}
}
func TestScheduledBatchContinuesBoundedSessionsThenDiagnostics(t *testing.T) {
	s, o := scheduleFixture(t)
	ctx := context.Background()
	c := s.Collector.Coordinator
	if err := c.Initialize(ctx, o); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteMigration(ctx, nil); err != nil {
		t.Fatal(err)
	}
	old := c.Now()
	c.Now = func() time.Time { return old.Add(61 * 24 * time.Hour) }
	cursor, complete, err := s.Batch(ctx, "", 1)
	if err != nil || complete || cursor == "" {
		t.Fatalf("first batch %q %v %v", cursor, complete, err)
	}
	for n := 0; !complete && n < 4; n++ {
		before := cursor
		cursor, complete, err = s.Batch(ctx, cursor, 1)
		if err != nil {
			t.Fatal(err)
		}
		if !complete && cursor == before {
			t.Fatal("cursor stalled")
		}
	}
	if !complete {
		t.Fatal("pass did not complete")
	}
	if _, err := os.Stat(filepath.Join(o.Directory(), "draft-tag.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("session remains", err)
	}
}
func TestStartAfterReadinessFailureDoesNotStartWorker(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	failed := errors.New("not ready")
	worker, err := StartAfterReady(context.Background(), root, func() error { return failed })
	if worker != nil || !errors.Is(err, failed) {
		t.Fatalf("worker %v err %v", worker, err)
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("readiness failure created root", err)
	}
}
func TestStartAfterReadyWorkerIsContextBoundAndJoined(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	worker, err := StartAfterReady(ctx, root, func() error { called = true; return nil })
	if err != nil || !called || worker == nil {
		t.Fatalf("ready=%v worker=%v err=%v", called, worker, err)
	}
	worker.Stop()
	if err := worker.Wait(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".retention")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled worker performed IO", err)
	}
}

func TestScheduledDiagnosticsAdvanceAcrossDeletedPaths(t *testing.T) {
	s, _ := scheduleFixture(t)
	ctx := context.Background()
	c := s.Collector.Coordinator
	if err := c.CompleteMigration(ctx, nil); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	s.DiagnosticOptions.Now = func() time.Time { return now }
	s.DiagnosticOptions.Proof = func(string, []diagnosticlog.Registration) error { return nil }
	paths := []string{filepath.Join(c.Root, "wrap-events-a.jsonl"), filepath.Join(c.Root, "wrap-events-b.jsonl")}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("old diagnostic"), 0600); err != nil {
			t.Fatal(err)
		}
		old := now.Add(-8 * 24 * time.Hour)
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	cursor, complete, err := s.Batch(ctx, "", 1)
	if err != nil || complete {
		t.Fatalf("session phase %q %v %v", cursor, complete, err)
	}
	for attempts := 0; !complete && attempts < 12; attempts++ {
		before := cursor
		cursor, complete, err = s.Batch(ctx, cursor, 1)
		if err != nil {
			t.Fatal(err)
		}
		if !complete && cursor == before {
			t.Fatal("diagnostic cursor stalled")
		}
	}
	if !complete {
		t.Fatal("diagnostic pass did not complete")
	}
	for _, path := range paths {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missed diagnostic path %s %v", path, err)
		}
	}
}

func TestReadyRuntimeWorkerRunsAndJoinsItsFirstScheduledPass(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	worker, err := StartAfterReady(ctx, root, func() error {
		if _, err := os.Stat(filepath.Join(root, ".retention")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("maintenance started before readiness", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { worker.Stop(); _ = worker.Wait() }()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if _, err := os.Stat(filepath.Join(root, ".retention", "schedule.json")); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("worker did not finish first pass")
		case <-tick.C:
		}
	}
	worker.Stop()
	if err := worker.Wait(); err != nil {
		t.Fatal(err)
	}
}
