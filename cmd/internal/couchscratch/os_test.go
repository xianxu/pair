package couchscratch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
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
	longRoot := filepath.Join(t.TempDir(), strings.Repeat("long", 30))
	if err := os.MkdirAll(longRoot, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", longRoot)
	//lint:allow tmp-literal — the private scratch fixture adds tmp/slot/socket components that must fit the native Unix socket limit even when inherited TMPDIR is long
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
	if _, err := m.Sweep(context.Background()); err == nil {
		t.Fatal("sweep accepted symlink shared tmp")
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("escaped root")
	}
}

func TestSharedTmpRejectsForeignOwner(t *testing.T) {
	st := unix.Stat_t{Mode: unix.S_IFDIR | 0755, Uid: uint32(os.Getuid() + 1)}
	if sharedTmpAllowed(st) {
		t.Fatal("foreign-owned container accepted")
	}
}
