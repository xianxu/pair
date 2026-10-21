package durablefile

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestDurableDirectoryAndPublication(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "authority")
	if err := EnsureDirectory(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "state.json")
	stage := path + ".publication"
	if err := os.WriteFile(stage, []byte("interrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomicStaged(path, []byte("complete"), stage); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "complete" {
		t.Fatal(string(raw), err)
	}
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
}
func TestUnsafePublicationStageRefused(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "victim")
	os.WriteFile(target, []byte("keep"), 0600)
	stage := filepath.Join(dir, "stage")
	os.Symlink(target, stage)
	if err := WriteAtomicStaged(filepath.Join(dir, "state"), []byte("replace"), stage); err == nil {
		t.Fatal("followed unsafe stage")
	}
	raw, _ := os.ReadFile(target)
	if string(raw) != "keep" {
		t.Fatal("changed victim")
	}
}

func TestPublishPreservesReplacedStage(t *testing.T) {
	for _, writable := range []bool{false, true} {
		t.Run(map[bool]string{false: "returned-error", true: "before-rename"}[writable], func(t *testing.T) {
			dir := t.TempDir()
			stage, dest := filepath.Join(dir, "stage"), filepath.Join(dir, "state")
			if err := os.WriteFile(stage, []byte("owned"), 0600); err != nil {
				t.Fatal(err)
			}
			flags := os.O_RDONLY
			if writable {
				flags = os.O_RDWR
			}
			f, err := os.OpenFile(stage, flags, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			original := filepath.Join(dir, "original")
			if err = os.Rename(stage, original); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(stage, []byte("foreign"), 0600); err != nil {
				t.Fatal(err)
			}
			err = publish(dest, []byte("updated owned"), f, 0600)
			if !writable && !errors.Is(err, syscall.EBADF) {
				t.Fatalf("original write error lost: %v", err)
			}
			if writable && err == nil {
				t.Fatal("published replacement stage")
			}
			if got, err := os.ReadFile(stage); err != nil || string(got) != "foreign" {
				t.Fatalf("foreign replacement changed: %q %v", got, err)
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("published foreign data: %v", err)
			}
		})
	}
}
