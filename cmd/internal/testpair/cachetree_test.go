package testpair

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveCacheTree(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	module := filepath.Join(root, "go", "pkg", "mod", "example@v1", "nested")
	if err := os.MkdirAll(module, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(outside, "keep")
	if err := os.WriteFile(marker, []byte("unchanged"), 0400); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(module, "outside")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example"), 0444); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(module)
	t.Cleanup(func() { // Restore the controlled RED fixture even if its owner fails.
		for _, dir := range []string{parent, module} {
			if err := os.Chmod(dir, 0700); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Error(err)
			}
		}
	})
	for _, dir := range []string{module, parent} {
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(root); err == nil {
		t.Fatal("control unexpectedly removed read-only module directories")
	}
	if err := RemoveCacheTree(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned cache still present: %v", err)
	}
	if err := RemoveCacheTree(root); err != nil {
		t.Fatalf("repeat: %v", err)
	}
	info, err := os.Stat(marker)
	if err != nil || info.Mode().Perm() != 0400 {
		t.Fatalf("external target changed: %v %v", info, err)
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "unchanged" {
		t.Fatalf("external bytes: %q %v", raw, err)
	}
}

func TestCacheTempDirOwnsCleanupAfterChildren(t *testing.T) {
	var root string
	t.Run("owner", func(t *testing.T) {
		root = CacheTempDir(t)
		module := filepath.Join(root, "module@v1")
		if err := os.Mkdir(module, 0555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := os.Stat(module); err != nil {
				t.Errorf("child cleanup lost its root: %v", err)
			}
		})
	})
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache cleanup left root: %v", err)
	}
}

func TestRemoveCacheTreeRejectsUnownedPath(t *testing.T) {
	for _, root := range []string{"", ".", "relative", string(filepath.Separator)} {
		if err := RemoveCacheTree(root); err == nil {
			t.Errorf("accepted %q", root)
		}
	}
}

func TestRemoveCacheTreeDoesNotFollowRootSymlink(t *testing.T) {
	outside := t.TempDir()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := RemoveCacheTree(link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("removed symlink target: %v", err)
	}
}

func TestRemoveCacheTreeReportsRemovalFailure(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "cache")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(parent, 0700); err != nil {
			t.Error(err)
		}
	})
	if err := RemoveCacheTree(root); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("cleanup failure not returned: %v", err)
	}
}
