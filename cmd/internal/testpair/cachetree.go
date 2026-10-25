package testpair

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// RemoveCacheTree removes a fixture-owned tree after its children have joined.
// Go extracts modules into read-only directories. Make owned directories
// traversable/writable before removal; WalkDir never follows symbolic links.
// Callers pass their private allocation, never a shared Go cache.
func RemoveCacheTree(root string) error {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || filepath.Dir(root) == root {
		return fmt.Errorf("cache cleanup requires an absolute private root: %q", root)
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Chmod(path, 0700)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("make owned cache removable: %w", err)
	}
	return os.RemoveAll(root)
}

// CacheTempDir handles Go's module permissions at the owning test's teardown.
// Register first so later child cleanups run before the tree disappears.
func CacheTempDir(t testing.TB) string {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() {
		if err := RemoveCacheTree(root); err != nil {
			t.Errorf("remove owned cache tree: %v", err)
		}
	})
	return root
}
