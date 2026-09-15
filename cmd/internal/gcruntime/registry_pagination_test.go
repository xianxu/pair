package gcruntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

func TestRuntimeRegistryEnumeratesPastFilteredResidues(t *testing.T) {
	s, _ := scheduleFixture(t)
	root := s.Collector.Coordinator.Root
	options := diagnosticlog.EnvironmentOptions(func(key string) string {
		if key == "PAIR_DATA_DIR" {
			return root
		}
		return ""
	})
	for n := 0; n < 101; n++ {
		path := filepath.Join(root, fmt.Sprintf("trace%d.log", n))
		entry := diagnosticlog.RegistryEntry{Version: 1, Path: path, Directory: path + ".pair-diagnostics", Lock: path + ".pair-diagnostics.lock"}
		if err := options.Registry(context.Background(), entry); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; n < 101; n++ {
		if err := os.WriteFile(filepath.Join(diagnosticlog.RegistryDirectory(root), fmt.Sprintf(".pending-%d", n)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := s.registryContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 101 {
		t.Fatalf("runtime stopped after%d of101 entries", len(entries))
	}
}

func TestBusyRegistryBatchPreservesSchedulerCursorForRetry(t *testing.T) {
	s, _ := scheduleFixture(t)
	root := s.Collector.Coordinator.Root
	ctx := context.Background()
	options := diagnosticlog.EnvironmentOptions(func(key string) string {
		if key == "PAIR_DATA_DIR" {
			return root
		}
		return ""
	})
	path := filepath.Join(root, "trace.log")
	if err := options.Registry(ctx, diagnosticlog.RegistryEntry{Version: 1, Path: path, Directory: path + ".pair-diagnostics", Lock: path + ".pair-diagnostics.lock"}); err != nil {
		t.Fatal(err)
	}
	saved := `{"phase":"sessions"}`
	scheduler := storagegc.Scheduler{Coordinator: s.Collector.Coordinator, Batch: func(context.Context, string, int) (string, bool, error) { return saved, false, nil }}
	if ran, err := scheduler.RunOnce(ctx); !ran || err != nil {
		t.Fatal(ran, err)
	}
	statePath := filepath.Join(root, ".retention", "schedule.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(diagnosticlog.RegistryDirectory(root), "registry.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if _, _, err := s.Batch(ctx, saved, 1); !errors.Is(err, storagegc.ErrCoordinatorBusy) {
		t.Fatalf("registry contention not retryable %v", err)
	}
	calls := 0
	scheduler.Batch = func(ctx context.Context, cursor string, limit int) (string, bool, error) {
		calls++
		if cursor != saved {
			t.Fatalf("lost durable cursor %q", cursor)
		}
		return s.Batch(ctx, cursor, limit)
	}
	if ran, err := scheduler.RunOnce(ctx); !ran || err != nil {
		t.Fatal("busy scheduler failed", ran, err)
	}
	after, err := os.ReadFile(statePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("busy scheduler changed durable cursor", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if ran, err := scheduler.RunOnce(ctx); !ran || err != nil {
		t.Fatal("retry did not resume", ran, err)
	}
	if calls != 2 {
		t.Fatalf("retrycalls%d", calls)
	}
}
