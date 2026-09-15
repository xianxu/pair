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

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func assertNoUnreachableQuarantine(t *testing.T, c *Collector) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(c.Coordinator.Root, ".retention", "quarantine"))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if _, err := os.Stat(filepath.Join(c.transactionDir(), entry.Name()+".json")); err != nil {
			t.Fatalf("unpublished quarantine %s: %v", entry.Name(), err)
		}
	}
}

func TestCollectionPublicationFailureHasNoUnreachableArtifacts(t *testing.T) {
	for _, stage := range []string{"before-persist", "temporary-written", "cancelled"} {
		t.Run(stage, func(t *testing.T) {
			c, item := transactionFixture(t)
			for attempt := 0; attempt < 3; attempt++ {
				ctx, cancel := context.WithCancel(context.Background())
				c.Coordinator.BeforePersist = nil
				c.Coordinator.AfterTempWrite = nil
				switch stage {
				case "before-persist":
					c.Coordinator.BeforePersist = func() error { return errors.New("publication failed") }
				case "temporary-written":
					c.Coordinator.AfterTempWrite = func(string) error { return errors.New("publication failed") }
				case "cancelled":
					c.Coordinator.AfterTempWrite = func(string) error { cancel(); return nil }
				}
				err := c.Coordinator.WithLock(ctx, func(l *Locked) error { return c.collectItem(l, item) })
				cancel()
				if err == nil {
					t.Fatal("publication failure not exercised")
				}
				assertNoUnreachableQuarantine(t, c)
			}
			c.Coordinator.BeforePersist = nil
			c.Coordinator.AfterTempWrite = nil
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 100) }); err != nil {
				t.Fatal(err)
			}
			assertNoUnreachableQuarantine(t, c)
			if body, err := os.ReadFile(item.Members[0].Path); err != nil || string(body) != "old draft" {
				t.Fatalf("unpublished operation changed source: %q %v", body, err)
			}
		})
	}
}

// Subprocess dies at an actual publication boundary, rather than relying on
// deferred cleanup after an injected Go error.
func TestCollectionPublicationKilledProcess(t *testing.T) {
	if root := os.Getenv("PAIR_TEST_COLLECTION_ROOT"); root != "" {
		coordinator, err := NewCoordinator(root)
		if err != nil {
			t.Fatal(err)
		}
		owner, err := artifactpath.NewStorageOwner(coordinator.Root, "scope", "test")
		if err != nil {
			t.Fatal(err)
		}
		member, err := artifactpath.MatchArtifact(filepath.Join(owner.Directory(), "draft-test.md"), []artifactpath.StorageOwner{owner}, []string{"codex"})
		if err != nil {
			t.Fatal(err)
		}
		c := &Collector{Coordinator: coordinator, Agents: []string{"codex"}}
		pause := func() {
			fmt.Println("COLLECTION-BOUNDARY")
			for {
				time.Sleep(time.Hour)
			}
		}
		if os.Getenv("PAIR_TEST_COLLECTION_STAGE") == "unpublished" {
			coordinator.AfterTempWrite = func(string) error { pause(); return nil }
		} else {
			c.Fault = func(step string) error {
				if step == "journal" {
					pause()
				}
				return nil
			}
		}
		if err := coordinator.WithLock(context.Background(), func(l *Locked) error {
			return c.collectItem(l, CollectionItem{Owner: owner, Bucket: artifactpath.SessionRetention, Members: []artifactpath.ArtifactMember{member}})
		}); err != nil {
			t.Fatal(err)
		}
		return
	}
	for _, stage := range []string{"unpublished", "published"} {
		t.Run(stage, func(t *testing.T) {
			c, item := transactionFixture(t)
			child := exec.Command(os.Args[0], "-test.run=^TestCollectionPublicationKilledProcess$")
			child.Env = append(os.Environ(), "PAIR_TEST_COLLECTION_ROOT="+c.Coordinator.Root, "PAIR_TEST_COLLECTION_STAGE="+stage)
			out, err := child.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
			ready := make(chan bool, 1)
			go func() {
				scanner := bufio.NewScanner(out)
				for scanner.Scan() {
					if scanner.Text() == "COLLECTION-BOUNDARY" {
						ready <- true
						return
					}
				}
				ready <- false
			}()
			select {
			case ok := <-ready:
				if !ok {
					t.Fatal("publisher exited before boundary")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("publisher readiness timeout")
			}
			if err := child.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = child.Wait()
			assertNoUnreachableQuarantine(t, c)
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error {
				if _, err := l.RecoverPendingMetadata(context.Background(), c.Coordinator.PendingMetadataDir(), 100); err != nil {
					return err
				}
				return c.recoverTransactions(l, 100)
			}); err != nil {
				t.Fatal(err)
			}
			assertNoUnreachableQuarantine(t, c)
			_, err = os.Stat(item.Members[0].Path)
			if stage == "published" && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("published authority not recovered: %v", err)
			}
			if stage == "unpublished" && err != nil {
				t.Fatalf("unpublished source lost: %v", err)
			}
		})
	}
}

func TestCollectionPublicationSyncFailureRecoversWithoutQuarantine(t *testing.T) {
	c, item := transactionFixture(t)
	c.Coordinator.SyncDirectory = func(path string) error {
		if path == c.transactionDir() {
			return errors.New("journal directory sync failed")
		}
		return syncDirectory(path)
	}
	if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) }); err == nil {
		t.Fatal("expected publication sync failure")
	}
	assertNoUnreachableQuarantine(t, c)
	entries, err := os.ReadDir(c.transactionDir())
	if err != nil || len(entries) != 1 {
		t.Fatalf("published journal missing: %v %v", entries, err)
	}
	c.Coordinator.SyncDirectory = nil
	if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 100) }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(item.Members[0].Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published source not collected: %v", err)
	}
	assertNoUnreachableQuarantine(t, c)
}

func TestCollectionPublicationRejectsUnsafeQuarantine(t *testing.T) {
	for _, kind := range []string{"parent-symlink", "transaction-symlink", "transaction-file"} {
		t.Run(kind, func(t *testing.T) {
			c, item := transactionFixture(t)
			outside := t.TempDir()
			sentinel := filepath.Join(outside, "sentinel")
			if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			c.Fault = func(step string) error {
				if step != "journal" {
					return nil
				}
				entries, err := os.ReadDir(c.transactionDir())
				if err != nil {
					return err
				}
				var txn CollectionTransaction
				if err := readStateJSON(filepath.Join(c.transactionDir(), entries[0].Name()), &txn); err != nil {
					return err
				}
				parent := filepath.Dir(c.quarantine(txn))
				if kind == "parent-symlink" {
					return os.Symlink(outside, parent)
				}
				if err := os.Mkdir(parent, 0700); err != nil {
					return err
				}
				if kind == "transaction-file" {
					return os.WriteFile(c.quarantine(txn), []byte("keep"), 0600)
				}
				return os.Symlink(outside, c.quarantine(txn))
			}
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) }); err == nil {
				t.Fatal("unsafe quarantine accepted")
			}
			c.Fault = nil
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 100) }); err == nil {
				t.Fatal("unsafe quarantine accepted during recovery")
			}
			if body, err := os.ReadFile(sentinel); err != nil || string(body) != "keep" {
				t.Fatalf("outside changed: %q %v", body, err)
			}
			if body, err := os.ReadFile(item.Members[0].Path); err != nil || string(body) != "old draft" {
				t.Fatalf("source changed: %q %v", body, err)
			}
		})
	}
}

func TestCollectionRecoveryPreservesUnjournaledQuarantine(t *testing.T) {
	c, _ := transactionFixture(t)
	unknown := filepath.Join(c.Coordinator.Root, ".retention", "quarantine", "unreferenced")
	if err := os.MkdirAll(unknown, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unknown, "unknown"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 100) }); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(unknown, "unknown")); err != nil || string(body) != "keep" {
		t.Fatalf("unproven artifact changed: %q %v", body, err)
	}
}
