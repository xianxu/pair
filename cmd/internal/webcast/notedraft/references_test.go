package notedraft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureReferencesPrivateAndOutsideSourceQuota(t *testing.T) {
	s, err := newDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.putRef("capture", []byte(`{"bundle":"exact"}`)); err != nil {
		t.Fatal(err)
	}
	if err = s.putRef("capture", []byte(`{"bundle":"new"}`)); err != nil {
		t.Fatal(err)
	}
	n, b, err := s.usage()
	if err != nil || n != 0 || b != 0 {
		t.Fatalf("reference consumes source quota: %d %d %v", n, b, err)
	}
	for path, mode := range map[string]os.FileMode{filepath.Join(s.root, ".refs"): 0700, filepath.Join(s.root, ".refs", "capture.json"): 0600} {
		st, e := os.Stat(path)
		if e != nil || st.Mode().Perm() != mode {
			t.Fatalf("private reference mode: %v %v", st, e)
		}
	}
	reopened, err := newDiskStore(s.root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := reopened.readRef("capture")
	if err != nil || string(data) != `{"bundle":"new"}` {
		t.Fatalf("reopen %s %v", data, err)
	}
	ids, err := reopened.listRefs()
	if err != nil || len(ids) != 1 || ids[0] != "capture" {
		t.Fatal(ids, err)
	}
	if err = s.removeRef("capture"); err != nil {
		t.Fatal(err)
	}
	if err = s.removeRef("capture"); err != nil {
		t.Fatal(err)
	}
}
func TestCaptureReferenceRejectsSymlinksAndInvalidIDs(t *testing.T) {
	s, err := newDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.putRef("../escape", []byte(`{}`)); err == nil {
		t.Fatal("invalid ID accepted")
	}
	if err = s.putRef("safe", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("safe.json", filepath.Join(s.root, ".refs", "link.json")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.readRef("link"); err == nil {
		t.Fatal("symlink read")
	}
	if err = s.putRef("link", []byte(`{}`)); err == nil {
		t.Fatal("symlink replaced")
	}
	if err = s.removeRef("link"); err == nil {
		t.Fatal("symlink removed")
	}
}
func TestCaptureStoresSharePublicationLock(t *testing.T) {
	root := t.TempDir()
	one, err := newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	two, err := newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if one.mu != two.mu {
		t.Fatal("same-root stores do not share lock")
	}
}

func TestCaptureReferenceReopenCleansInterruptedMetadata(t *testing.T) {
	s, err := newDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.putRef("capture", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(s.root, ".refs", ".stage-0123456789abcdef0123456789abcdef")
	if err = os.WriteFile(stage, []byte("interrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = newDiskStore(s.root); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(stage); !os.IsNotExist(err) {
		t.Fatal("reference temporary file survived reopen", err)
	}
}
