package historyarchive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreExclusivePublicationNeverReplacesTarget(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	parent, err := openSafeDir(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	stage, err := childDir(parent, "stage", true)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	if err := os.Mkdir(filepath.Join(root, "stage", "output"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stage", "output", "file"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "target"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target", "keep"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishRestoreExclusive(stage, "output", parent, "target"); err == nil {
		t.Fatal("replaced existing target")
	}
	got, err := os.ReadFile(filepath.Join(root, "target", "keep"))
	if err != nil || string(got) != "original" {
		t.Fatal("existing bytes changed", string(got), err)
	}
	if err := publishRestoreExclusive(stage, "output", parent, "fresh"); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(filepath.Join(root, "fresh", "file"))
	if err != nil || string(got) != "new" {
		t.Fatal("whole output missing", string(got), err)
	}
}

func TestRestoreExclusivePublicationRefusesSymlinkTarget(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	parent, err := openSafeDir(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	stage, err := childDir(parent, "stage", true)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	if err := os.Mkdir(filepath.Join(root, "stage", "output"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "foreign"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("foreign", filepath.Join(root, "target")); err != nil {
		t.Fatal(err)
	}
	if err := publishRestoreExclusive(stage, "output", parent, "target"); err == nil {
		t.Fatal("replaced symlink target")
	}
	if got, err := os.Readlink(filepath.Join(root, "target")); err != nil || got != "foreign" {
		t.Fatal(got, err)
	}
}
