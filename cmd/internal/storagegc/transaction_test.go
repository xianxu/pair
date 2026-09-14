package storagegc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func transactionFixture(t *testing.T) (*Collector, CollectionItem) {
	t.Helper()
	c, o := coordinatorFixture(t)
	if err := os.MkdirAll(o.Directory(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := c.Initialize(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(o.Directory(), "draft-test.md")
	if err := os.WriteFile(path, []byte("old draft"), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := artifactpath.MatchArtifact(path, []artifactpath.StorageOwner{o}, []string{"codex"})
	if err != nil {
		t.Fatal(err)
	}
	return &Collector{Coordinator: c, Agents: []string{"codex"}}, CollectionItem{Owner: o, Bucket: artifactpath.SessionRetention, Members: []artifactpath.ArtifactMember{m}}
}
func TestCollectionTransactionRecoversEveryDurableBoundary(t *testing.T) {
	steps := []string{"journal", "rename:0", "detached", "retired", "finalized", "remove:0", "forgotten"}
	for _, step := range steps {
		t.Run(step, func(t *testing.T) {
			c, item := transactionFixture(t)
			hit := false
			c.Fault = func(s string) error {
				if s == step && !hit {
					hit = true
					return errors.New("crash")
				}
				return nil
			}
			err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) })
			if err == nil || !hit {
				t.Fatalf("did not crash at %s: %v", step, err)
			}
			if err := c.Coordinator.Initialize(context.Background(), item.Owner); err == nil {
				t.Fatal("pending transaction allowed owner access")
			}
			c.Fault = nil
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 20) }); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(item.Members[0].Path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("source remains: %v", err)
			}
			if err := c.Coordinator.Initialize(context.Background(), item.Owner); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestCollectionTransactionRefusesEXDEVAndSymlinkSubstitution(t *testing.T) {
	for _, kind := range []string{"exdev", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			c, item := transactionFixture(t)
			if kind == "exdev" {
				c.Rename = func(string, string) error { return syscall.EXDEV }
			} else {
				c.Fault = func(step string) error {
					if step == "journal" {
						if err := os.Remove(item.Members[0].Path); err != nil {
							return err
						}
						return os.Symlink("/outside", item.Members[0].Path)
					}
					return nil
				}
			}
			err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) })
			if err == nil {
				t.Fatal("unsafe rename accepted")
			}
			if _, err := os.Lstat(item.Members[0].Path); err != nil {
				t.Fatal("source lost", err)
			}
		})
	}
}
func TestDetachedRecoveryPreservesNewSourceAndIncarnation(t *testing.T) {
	c, item := transactionFixture(t)
	c.CleanupOwner = func(*Locked, artifactpath.StorageOwner) error {
		return errors.New("must preserve replacement bindings")
	}
	c.Fault = func(step string) error {
		if step == "detached" {
			return errors.New("crash")
		}
		return nil
	}
	if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) }); err == nil {
		t.Fatal("missing crash")
	}
	if err := os.WriteFile(item.Members[0].Path, []byte("new draft"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error {
		s, e := c.Coordinator.ReadOwner(item.Owner)
		if e != nil {
			return e
		}
		s.Activity.Incarnation = "replacement"
		return l.save(s)
	}); err != nil {
		t.Fatal(err)
	}
	c.Fault = nil
	if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 20) }); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(item.Members[0].Path)
	if err != nil || string(raw) != "new draft" {
		t.Fatalf("new source lost: %s %v", raw, err)
	}
	s, err := c.Coordinator.ReadOwner(item.Owner)
	if err != nil || s.Activity.Incarnation != "replacement" {
		t.Fatalf("new metadata lost: %v", err)
	}
}

type transactionReferences struct {
	detached, forgotten int
	finalized           bool
}

func (r *transactionReferences) Snapshot(context.Context, *Locked, []string) (References, error) {
	return References{}, nil
}
func (r *transactionReferences) Detach(_ *Locked, _ string, _ ArchiveReference) error {
	if r.finalized {
		return errors.New("detach replayed after finalization")
	}
	r.detached++
	return nil
}
func (r *transactionReferences) Forget(_ *Locked, _ string, _ ArchiveReference) error {
	r.finalized = true
	r.forgotten++
	return nil
}
func TestCollectionReceiptsFinalizeBeforeForgetAndCleanup(t *testing.T) {
	for _, step := range []string{"archive:0", "cleanup", "forget:0", "finalized"} {
		t.Run(step, func(t *testing.T) {
			c, item := transactionFixture(t)
			refs := &transactionReferences{}
			c.References = refs
			item.Archives = []ArchiveReference{{Owner: item.Owner, Store: "external"}}
			cleanup := 0
			c.CleanupOwner = func(*Locked, artifactpath.StorageOwner) error { cleanup++; return nil }
			c.Fault = func(s string) error {
				if s == step {
					return errors.New("crash")
				}
				return nil
			}
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) }); err == nil {
				t.Fatal("missing crash")
			}
			c.Fault = nil
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 10) }); err != nil {
				t.Fatal(err)
			}
			if refs.forgotten == 0 || cleanup == 0 {
				t.Fatal("cleanup/receipt missing")
			}
		})
	}
}
func TestCollectionDirectoryRejectsMissingAndUnexpectedDescendants(t *testing.T) {
	for _, kind := range []string{"missing-source", "extra-source", "extra-quarantine"} {
		t.Run(kind, func(t *testing.T) {
			c, item := transactionFixture(t)
			dir := filepath.Join(item.Owner.Directory(), "queue-test")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			child := filepath.Join(dir, "1.md")
			if err := os.WriteFile(child, []byte("queued"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{dir, child} {
				m, e := artifactpath.MatchArtifact(path, []artifactpath.StorageOwner{item.Owner}, c.Agents)
				if e != nil {
					t.Fatal(e)
				}
				item.Members = append(item.Members, m)
			}
			c.Fault = func(step string) error {
				if step == "journal" {
					if kind == "missing-source" {
						return os.Remove(child)
					}
					if kind == "extra-source" {
						return os.WriteFile(filepath.Join(dir, "unexpected"), []byte("keep"), 0600)
					}
				}
				if step == "detached" && kind == "extra-quarantine" {
					entries, e := os.ReadDir(filepath.Join(c.Coordinator.Root, ".retention", "quarantine"))
					if e != nil {
						return e
					}
					return os.WriteFile(filepath.Join(c.Coordinator.Root, ".retention", "quarantine", entries[0].Name(), "unexpected"), []byte("keep"), 0600)
				}
				return nil
			}
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) }); err == nil {
				t.Fatal("mutation accepted")
			}
		})
	}
}

func TestCaptureTransactionPairsRecoveryAndPreservesParentActivity(t *testing.T) {
	for _, step := range []string{"rename:0", "rename:1", "detached", "finalized", "remove:0"} {
		t.Run(step, func(t *testing.T) {
			c, item := transactionFixture(t)
			before, err := c.Coordinator.ReadOwner(item.Owner)
			if err != nil {
				t.Fatal(err)
			}
			p, err := artifactpath.ResolveScoped(item.Owner.Directory(), item.Owner.Tag)
			if err != nil {
				t.Fatal(err)
			}
			set, err := p.ParkedScrollbackArtifacts("20260101T010101")
			if err != nil {
				t.Fatal(err)
			}
			item.Bucket = artifactpath.CaptureRetention
			item.Members = nil
			for _, path := range []string{set.Raw, set.Events} {
				if err := os.WriteFile(path, []byte("old capture"), 0600); err != nil {
					t.Fatal(err)
				}
				m, e := artifactpath.MatchArtifact(path, []artifactpath.StorageOwner{item.Owner}, c.Agents)
				if e != nil {
					t.Fatal(e)
				}
				item.Members = append(item.Members, m)
			}
			c.CleanupOwner = func(*Locked, artifactpath.StorageOwner) error { return errors.New("capture ran session cleanup") }
			c.Fault = func(s string) error {
				if s == step {
					return errors.New("crash")
				}
				return nil
			}
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) }); err == nil {
				t.Fatal("missing crash")
			}
			c.Fault = nil
			if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.recoverTransactions(l, 10) }); err != nil {
				t.Fatal(err)
			}
			after, err := c.Coordinator.ReadOwner(item.Owner)
			if err != nil || after.Activity != before.Activity {
				t.Fatalf("parent activity changed: %v", err)
			}
			for _, path := range []string{set.Raw, set.Events} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("capture remains %s %v", path, err)
				}
			}
		})
	}
}

func TestCollectionPrepareRejectsOmittedDirectoryChild(t *testing.T) {
	c, item := transactionFixture(t)
	dir := filepath.Join(item.Owner.Directory(), "queue-test")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "1.md"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := artifactpath.MatchArtifact(dir, []artifactpath.StorageOwner{item.Owner}, c.Agents)
	if err != nil {
		t.Fatal(err)
	}
	item.Members = append(item.Members, m)
	if err := c.Coordinator.WithLock(context.Background(), func(l *Locked) error { return c.collectItem(l, item) }); err == nil {
		t.Fatal("omitted child accepted")
	}
	if _, err := os.Lstat(item.Members[0].Path); err != nil {
		t.Fatal("effect happened before validation", err)
	}
}

func TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays(t *testing.T) {
	c, o := collectorFixture(t)
	now := c.Coordinator.Now()
	p, _ := artifactpath.ResolveScoped(o.Directory(), o.Tag)
	set, _ := p.ParkedScrollbackArtifacts(now.Add(-24 * time.Hour).Format("20060102T150405"))
	for _, path := range []string{set.Raw, set.Events} {
		if err := os.WriteFile(path, []byte("young capture"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, now.Add(-24*time.Hour), now.Add(-24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	report, err := c.Apply(context.Background(), 100)
	if err != nil || report.Collected != 1 {
		t.Fatalf("session collection %+v %v", report, err)
	}
	if _, err := c.Coordinator.ReadOwner(o); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("session state not retired: %v", err)
	}
	for _, path := range []string{set.Raw, set.Events} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("young capture lost", err)
		}
	}
	c.Coordinator.Now = func() time.Time { return now.Add(8 * 24 * time.Hour) }
	report, err = c.Apply(context.Background(), 100)
	if err != nil || report.Collected != 1 {
		t.Fatalf("capture collection after parent retirement %+v %v", report, err)
	}
	for _, path := range []string{set.Raw, set.Events} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("expired capture retained", err)
		}
	}
}

func TestMalformedPendingTransactionFailsClosedForManagedWrites(t *testing.T) {
	c, item := transactionFixture(t)
	if err := os.Mkdir(c.transactionDir(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.transactionDir(), "broken.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.Coordinator.Initialize(context.Background(), item.Owner); err == nil {
		t.Fatal("malformed transaction allowed owner access")
	}
}
