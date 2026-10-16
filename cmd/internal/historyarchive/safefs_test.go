package historyarchive

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

func archiveTemp(t *testing.T) string {
	t.Helper()
	p, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestSafeFSRefusesLinksAndTraversal(t *testing.T) {
	d := archiveTemp(t)
	if e := os.WriteFile(filepath.Join(d, "data"), []byte("x"), 0600); e != nil {
		t.Fatal(e)
	}
	os.Symlink("data", filepath.Join(d, "link"))
	os.Symlink(d, filepath.Join(d, "dirlink"))
	root, e := openSafeDir(d, false)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	for _, p := range []string{"link", "dirlink/data", "../data", "/data", "a/../data"} {
		f, e := openRegularAt(root, p, unix.O_RDONLY)
		if e == nil {
			f.Close()
			t.Errorf("accepted %q", p)
		}
	}
	f, e := openRegularAt(root, "data", unix.O_RDONLY)
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
}

func TestSafeFSRejectsSpecialAndRootSymlink(t *testing.T) {
	d := archiveTemp(t)
	if e := unix.Mkfifo(filepath.Join(d, "fifo"), 0600); e != nil {
		t.Fatal(e)
	}
	root, e := openSafeDir(d, false)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	if f, e := openRegularAt(root, "fifo", unix.O_RDONLY); e == nil {
		f.Close()
		t.Fatal("FIFO accepted")
	}
	os.Symlink(d, filepath.Join(d, "rootlink"))
	if f, e := openSafeDir(filepath.Join(d, "rootlink"), false); e == nil {
		f.Close()
		t.Fatal("root symlink accepted")
	}
}
