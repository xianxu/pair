package storagegc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestInterruptedMetadataPublisherProcess(t *testing.T) {
	if root := os.Getenv("PAIR_TEST_METADATA_ROOT"); root != "" {
		c, err := NewCoordinator(root)
		if err != nil {
			t.Fatal(err)
		}
		c.AfterTempWrite = func(path string) error {
			if os.Getenv("PAIR_TEST_METADATA_PARTIAL") == "1" {
				if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
					return err
				}
			}
			fmt.Println("PENDING-READY")
			for {
				time.Sleep(time.Hour)
			}
		}
		err = c.WithLock(context.Background(), func(l *Locked) error {
			return l.atomicJSON(os.Getenv("PAIR_TEST_METADATA_TARGET"), map[string]string{"new": "unpublished"})
		})
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	for _, directory := range []string{"owners", "transactions", "registry", "capture"} {
		for _, partial := range []string{"0", "1"} {
			t.Run(directory+"/partial="+partial, func(t *testing.T) {
				c, _ := coordinatorFixture(t)
				dir := filepath.Join(c.Root, ".retention", directory)
				if directory == "registry" {
					dir = filepath.Join(c.Root, ".retention")
				}
				if directory == "capture" {
					dir = filepath.Join(c.Root, "repos", "scope")
				}
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(dir, "test.json")
				original := []byte(`{"old":"authoritative"}`)
				if err := os.WriteFile(target, original, 0600); err != nil {
					t.Fatal(err)
				}
				child := exec.Command(os.Args[0], "-test.run=^TestInterruptedMetadataPublisherProcess$")
				child.Env = append(os.Environ(), "PAIR_TEST_METADATA_ROOT="+c.Root, "PAIR_TEST_METADATA_TARGET="+target, "PAIR_TEST_METADATA_PARTIAL="+partial)
				stdout, err := child.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				if err := child.Start(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
				ready := make(chan bool, 1)
				go func() {
					scanner := bufio.NewScanner(stdout)
					for scanner.Scan() {
						if scanner.Text() == "PENDING-READY" {
							ready <- true
							return
						}
					}
					ready <- false
				}()
				select {
				case ok := <-ready:
					if !ok {
						t.Fatal("publisher exited before fault")
					}
				case <-time.After(5 * time.Second):
					t.Fatal("publisher did not reach publication boundary")
				}
				if err := child.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				_ = child.Wait()
				raw, err := os.ReadFile(target)
				if err != nil || string(raw) != string(original) {
					t.Fatalf("unpublished bytes became authoritative: %q %v", raw, err)
				}
				entries, err := os.ReadDir(c.PendingMetadataDir())
				if err != nil || len(entries) != 1 {
					t.Fatalf("missing interrupted temp: %v %v", entries, err)
				}
				if err := c.WithLock(context.Background(), func(l *Locked) error {
					n, e := l.RecoverPendingMetadata(context.Background(), c.PendingMetadataDir(), 4)
					if n != 1 {
						t.Errorf("processed=%d", n)
					}
					return e
				}); err != nil {
					t.Fatal(err)
				}
				entries, err = os.ReadDir(c.PendingMetadataDir())
				if err != nil || len(entries) != 0 {
					t.Fatalf("residue survived: %v %v", entries, err)
				}
			})
		}
	}
}

func TestLegacyPendingMetadataRecoveryBoundAndNonAuthority(t *testing.T) {
	for _, directory := range []string{"owners", "transactions", "registry", "capture", "legacy-root"} {
		t.Run(directory, func(t *testing.T) {
			c, _ := coordinatorFixture(t)
			dir := filepath.Join(c.Root, ".retention", directory)
			if directory == "registry" {
				dir = filepath.Join(c.Root, ".retention")
			}
			if directory == "capture" {
				dir = filepath.Join(c.Root, "repos", "scope")
			}
			if directory == "legacy-root" {
				dir = c.Root
			}
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			pending := filepath.Join(dir, ".pending-123456")
			if err := os.WriteFile(pending, []byte("{"), 0600); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(dir, "published.json")
			if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			symlink := filepath.Join(dir, ".pending-654321")
			if err := os.Symlink(sentinel, symlink); err != nil {
				t.Fatal(err)
			}
			if err := c.WithLock(context.Background(), func(l *Locked) error {
				n, err := l.RecoverPendingMetadata(context.Background(), dir, 20)
				if n > 20 {
					t.Fatalf("unbounded work: %d", n)
				}
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(pending); !os.IsNotExist(err) {
				t.Fatalf("legacy temp remains: %v", err)
			}
			if _, err := os.Lstat(symlink); err != nil {
				t.Fatalf("symlink removed: %v", err)
			}
			raw, err := os.ReadFile(sentinel)
			if err != nil || string(raw) != "keep" {
				t.Fatal("authoritative file changed")
			}
		})
	}
}

func TestPendingMetadataCleanupIsBoundedAndCancelable(t *testing.T) {
	c, _ := coordinatorFixture(t)
	if err := c.WithLock(context.Background(), func(l *Locked) error {
		if err := os.MkdirAll(c.PendingMetadataDir(), 0700); err != nil {
			return err
		}
		for i := 0; i < 8; i++ {
			if err := os.WriteFile(filepath.Join(c.PendingMetadataDir(), fmt.Sprintf(".pending-%d", i)), nil, 0600); err != nil {
				return err
			}
		}
		n, err := l.RecoverPendingMetadata(context.Background(), c.PendingMetadataDir(), 2)
		if err != nil {
			return err
		}
		if n != 2 {
			t.Fatalf("work=%d", n)
		}
		remaining, _ := os.ReadDir(c.PendingMetadataDir())
		if len(remaining) != 6 {
			t.Fatalf("removed beyond bound: %d", len(remaining))
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		n, err = l.RecoverPendingMetadata(ctx, c.PendingMetadataDir(), 2)
		if n != 0 || err != context.Canceled {
			t.Fatalf("canceled work=%d err=%v", n, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestTransactionTempRecoveryChargesEachVisitedEntry(t *testing.T) {
	c, _ := coordinatorFixture(t)
	collector := &Collector{Coordinator: c}
	dir := collector.transactionDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf(".pending-%d", i)), []byte("{"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for remaining := 2; remaining >= 0; remaining-- {
		err := c.WithLock(context.Background(), func(l *Locked) error { return collector.recoverTransactions(l, 1) })
		if remaining > 0 && !errors.Is(err, ErrMaintenanceYield) {
			t.Fatalf("work limit did not yield: %v", err)
		}
		if remaining == 0 && err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != remaining {
			t.Fatalf("remaining=%d entries=%v err=%v", remaining, entries, err)
		}
	}
}

func legacyCollectorPendingFiles(t *testing.T, gc *Collector, count int) []string {
	t.Helper()
	dirs := []string{gc.Coordinator.Root, filepath.Join(gc.Coordinator.Root, "repos", "0123456789abcdef"), filepath.Join(gc.Coordinator.Root, ".retention", "owners")}
	var paths []string
	for i := range count {
		dir := dirs[i%len(dirs)]
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, fmt.Sprintf(".pending-%d", 100000+i))
		if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return paths
}

func TestCollectorLegacyPendingPreviewAndApply(t *testing.T) {
	gc, owner := collectorFixture(t)
	paths := legacyCollectorPendingFiles(t, gc, 3)
	before, err := os.ReadFile(gc.Coordinator.statePath(owner))
	if err != nil {
		t.Fatal(err)
	}
	report, err := gc.Preview(context.Background())
	if err != nil || !report.Complete || len(report.Unknown) != 0 || len(report.Items) != 1 || report.Items[0].Decision.State != Eligible {
		t.Fatalf("uncommitted temp blocked preview: report=%+v err=%v", report, err)
	}
	for _, path := range paths {
		if body, err := os.ReadFile(path); err != nil || string(body) != "{" {
			t.Fatalf("preview changed pending file %s: %q %v", path, body, err)
		}
	}
	if after, err := os.ReadFile(gc.Coordinator.statePath(owner)); err != nil || string(after) != string(before) {
		t.Fatalf("preview changed owner metadata: %v", err)
	}
	report, err = gc.Apply(context.Background(), 100)
	if err != nil || report.Collected != 1 {
		t.Fatalf("apply could not recover residue and collect: %+v %v", report, err)
	}
	for _, path := range append(paths, filepath.Join(owner.Directory(), "draft-tag.md")) {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("recovered/collected file remains %s: %v", path, err)
		}
	}
}

func TestCollectorLegacyPendingCancellationAndSharedBudget(t *testing.T) {
	gc, owner := collectorFixture(t)
	paths := legacyCollectorPendingFiles(t, gc, 103)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gc.Apply(ctx, 100); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled apply: %v", err)
	}
	remaining := func() int {
		t.Helper()
		n := 0
		for _, path := range paths {
			if _, err := os.Lstat(path); err == nil {
				n++
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
		}
		return n
	}
	if n := remaining(); n != len(paths) {
		t.Fatalf("canceled apply removed %d pending files", len(paths)-n)
	}
	if _, err := gc.Apply(context.Background(), 100); !errors.Is(err, ErrMaintenanceYield) {
		t.Fatalf("over-budget cleanup did not yield: %v", err)
	}
	if n := remaining(); n < 3 || n >= len(paths) {
		t.Fatalf("shared cleanup budget failed: %d of %d files remain", n, len(paths))
	}
	if _, err := os.Stat(filepath.Join(owner.Directory(), "draft-tag.md")); err != nil {
		t.Fatalf("yield collected payload before inventory completed: %v", err)
	}
	report, err := gc.Apply(context.Background(), 100)
	if err != nil || report.Collected != 1 || remaining() != 0 {
		t.Fatalf("retry did not finish residue and collection: %+v %v remaining=%d", report, err, remaining())
	}
}
