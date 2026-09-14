package storagegc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerBudgetResumeAndDailyCompletion(t *testing.T) {
	c, err := NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	calls := 0
	s := Scheduler{Coordinator: c, Now: func() time.Time { return now }, Limit: 3}
	s.Batch = func(ctx context.Context, cursor string, limit int) (string, bool, error) {
		calls++
		if limit != 3 {
			t.Fatalf("limit=%d", limit)
		}
		// Batch can acquire the root lock itself.
		if err := c.WithLock(ctx, func(*Locked) error { return nil }); err != nil {
			t.Fatal(err)
		}
		if calls == 1 {
			if cursor != "" {
				t.Fatal(cursor)
			}
			return "next", false, nil
		}
		if cursor != "next" {
			t.Fatalf("lost cursor %q", cursor)
		}
		if calls == 2 {
			return "bad", true, errors.New("failed")
		}
		return "", true, nil
	}
	for i := 0; i < 3; i++ {
		ran, err := s.RunOnce(context.Background())
		if !ran || (err != nil) != (i == 1) {
			t.Fatalf("%d: %v %v", i, ran, err)
		}
	}
	if ran, err := s.RunOnce(context.Background()); ran || err != nil {
		t.Fatalf("daily %v %v", ran, err)
	}
	now = now.Add(24 * time.Hour)
	s.Batch = func(context.Context, string, int) (string, bool, error) { calls++; return "", true, nil }
	if ran, err := s.RunOnce(context.Background()); !ran || err != nil {
		t.Fatalf("due %v %v", ran, err)
	}
	if calls != 4 {
		t.Fatal(calls)
	}
}

func TestSchedulerConcurrentWorkerAndCancellation(t *testing.T) {
	c, _ := NewCoordinator(t.TempDir())
	entered := make(chan struct{})
	finished := make(chan struct{})
	s := Scheduler{Coordinator: c, Batch: func(ctx context.Context, _ string, _ int) (string, bool, error) {
		close(entered)
		<-ctx.Done()
		close(finished)
		return "", false, ctx.Err()
	}}
	worker := s.Start(context.Background())
	<-entered
	other := Scheduler{Coordinator: c, Batch: func(context.Context, string, int) (string, bool, error) {
		t.Fatal("overlapping batch")
		return "", true, nil
	}}
	if ran, err := other.RunOnce(context.Background()); ran || err != nil {
		t.Fatalf("busy %v %v", ran, err)
	}
	worker.Stop()
	if err := worker.Wait(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	default:
		t.Fatal("worker was not joined")
	}
	if ran, err := other.RunOnce(canceledContext()); ran || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel %v %v", ran, err)
	}
}
func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestSchedulerRestartAndPersistenceFailure(t *testing.T) {
	c, _ := NewCoordinator(t.TempDir())
	s := Scheduler{Coordinator: c, Batch: func(context.Context, string, int) (string, bool, error) { return "saved", false, nil }}
	if _, err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	fresh, _ := NewCoordinator(c.Root)
	resumed := Scheduler{Coordinator: fresh, Batch: func(_ context.Context, cursor string, _ int) (string, bool, error) {
		if cursor != "saved" {
			t.Fatalf("restart cursor %q", cursor)
		}
		return "", true, nil
	}}
	fresh.BeforePersist = func() error { return errors.New("disk unavailable") }
	if ran, err := resumed.RunOnce(context.Background()); !ran || err == nil {
		t.Fatalf("persist %v %v", ran, err)
	}
	fresh.BeforePersist = nil
	if ran, err := resumed.RunOnce(context.Background()); !ran || err != nil {
		t.Fatalf("retry %v %v", ran, err)
	}
}

func TestSchedulerRejectsMalformedStateAndNoProgress(t *testing.T) {
	for _, raw := range []string{`{"version":2}`, `{"version":1} {}`, `{"version":1,"unexpected":true}`} {
		t.Run(raw, func(t *testing.T) {
			c, _ := NewCoordinator(t.TempDir())
			if err := c.WithLock(context.Background(), func(*Locked) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(c.Root, ".retention", "schedule.json"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			s := Scheduler{Coordinator: c, Batch: func(context.Context, string, int) (string, bool, error) {
				t.Fatal("malformed state ran batch")
				return "", true, nil
			}}
			if ran, err := s.RunOnce(context.Background()); ran || err == nil {
				t.Fatalf("%v %v", ran, err)
			}
		})
	}
	c, _ := NewCoordinator(t.TempDir())
	s := Scheduler{Coordinator: c, Batch: func(context.Context, string, int) (string, bool, error) { return "", false, nil }}
	if ran, err := s.RunOnce(context.Background()); !ran || err == nil {
		t.Fatalf("no progress %v %v", ran, err)
	}
}

func TestSchedulerWorkerBoundsConsecutiveBatches(t *testing.T) {
	c, _ := NewCoordinator(t.TempDir())
	var calls atomic.Int32
	reached := make(chan struct{})
	s := Scheduler{Coordinator: c, Batch: func(context.Context, string, int) (string, bool, error) {
		n := calls.Add(1)
		if n == schedulerBurstLimit {
			close(reached)
		}
		return strconv.Itoa(int(n)), false, nil
	}}
	w := s.Start(context.Background())
	defer w.Stop()
	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not resume batches")
	}
	time.Sleep(30 * time.Millisecond)
	w.Stop()
	if err := w.Wait(); err != nil {
		t.Fatal(err)
	}
	if n := calls.Load(); n != schedulerBurstLimit {
		t.Fatalf("unbounded burst: %d", n)
	}
}
