package couchscratch

import (
	"context"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProbeDoesNotTreatOrdinaryDirectoryAsMount(t *testing.T) {
	root := t.TempDir()
	p := ReadProbe(root)
	if p.Mounted || p.Err != nil {
		t.Fatalf("%+v", p)
	}
	p = ReadProbe(filepath.Join(root, "missing"))
	if p.Mounted || p.Err == nil {
		t.Fatalf("%+v", p)
	}
}
func TestSpecialTopLevelNodesRetained(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "cs-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	scratchRoot := root
	now := time.Now()
	m := New(Options{Probe: func() Probe { return Probe{Root: scratchRoot, Mounted: true, FreeBytes: Floor} }, Now: func() time.Time { return now }})
	env := m.PrepareSlot(Slot{Repo: "p", Identity: "x", Number: 0})
	root = strings.TrimPrefix(env[0], "TMPDIR=")
	fifo := filepath.Join(root, "fifo")
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(root, "socket")
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if err = unix.Bind(fd, &unix.SockaddrUnix{Name: sock}); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-25 * time.Hour)
	for _, path := range []string{fifo, sock} {
		os.Chtimes(path, old, old)
	}
	r, err := m.Sweep(context.Background())
	if err != nil || r.Removed != 0 || r.Retained != 2 {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestPrepareRejectsSymlinkInteriors(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	os.Symlink(outside, filepath.Join(root, "tmp"))
	m := New(Options{Probe: func() Probe { return Probe{Root: root, Mounted: true, FreeBytes: Floor} }})
	if env := m.PrepareSlot(Slot{Repo: "pair", Identity: "x", Number: 0}); env != nil {
		t.Fatal(env)
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("escaped root")
	}
}
