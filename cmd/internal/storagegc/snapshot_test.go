package storagegc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadSnapshotDoesNotInitializeRoot(t *testing.T) {
	c, _ := coordinatorFixture(t)
	if err := c.WithReadLock(context.Background(), func(l *Locked) error {
		if !l.Holds(c.Root) {
			t.Fatal("missing snapshot token")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(c.Root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("preview wrote files: %v %v", entries, err)
	}
}
func TestReadSnapshotRejectsSymlinkLock(t *testing.T) {
	c, _ := coordinatorFixture(t)
	if err := os.Mkdir(filepath.Join(c.Root, ".retention"), 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "lock")
	os.WriteFile(target, nil, 0600)
	os.Symlink(target, filepath.Join(c.Root, ".retention", "coordinator.lock"))
	if err := c.WithReadLock(context.Background(), func(*Locked) error { return nil }); err == nil {
		t.Fatal("followed snapshot lock symlink")
	}
}
