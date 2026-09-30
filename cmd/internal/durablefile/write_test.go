package durablefile

import (
	"os"
	"path/filepath"
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
