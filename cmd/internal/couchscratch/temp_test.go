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

func TestOwnerKnownMembersRejectNull(t *testing.T) {
	for _, raw := range []string{`{"Repo":"pair","Identity":"git","Number":null}`, `{"Repo":null,"Identity":"git","Number":0}`, `{"Repo":"pair","Identity":null,"Number":0}`} {
		if owner, err := decodeOwner([]byte(raw)); err == nil {
			t.Fatalf("accepted null known member: %s %+v", raw, owner)
		}
	}
}

func TestPrepareRefusesUnsafeLock(t *testing.T) {
	for _, kind := range []string{"mode", "fifo", "directory"} {
		t.Run(kind, func(t *testing.T) {
			m, root, _ := fixture(t)
			lock := filepath.Join(filepath.Dir(filepath.Dir(root)), "couch-scratch", "owners", ".lock")
			if err := os.Remove(lock); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "mode":
				os.WriteFile(lock, nil, 0644)
			case "fifo":
				if err := unix.Mkfifo(lock, 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				os.Mkdir(lock, 0700)
			}
			done := make(chan []string, 1)
			go func() { done <- m.PrepareSlot(Slot{Repo: "pair", Identity: "git-common", Number: 3}) }()
			select {
			case env := <-done:
				if env != nil {
					t.Fatal("accepted unsafe lock", env)
				}
			case <-time.After(time.Second):
				t.Fatal("blocked on unsafe lock")
			}
		})
	}
}
func TestOversizeOwnerDeclarationRejectedByPrepareAndSweep(t *testing.T) {
	m, root, _ := fixture(t)
	declaration := filepath.Join(filepath.Dir(filepath.Dir(root)), "couch-scratch", "owners", "pair-slot3.json")
	prefix := `{"Repo":"pair","Identity":"git-common","Number":3}`
	data := prefix + strings.Repeat(" ", (1<<20)-len(prefix)) + "trailing"
	if err := os.WriteFile(declaration, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if env := m.PrepareSlot(Slot{Repo: "pair", Identity: "git-common", Number: 3}); env != nil {
		t.Fatal("accepted oversize owner")
	}
	fd, err := openDir(unix.AT_FDCWD, filepath.Dir(declaration))
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if _, err := readOwnerAt(fd, filepath.Base(declaration), &Result{}); err == nil {
		t.Fatal("sweep accepted oversize owner")
	}
}

func TestSharedTmp0755PreservedWithPrivateSlot(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "tmp")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(parent)
	if err != nil {
		t.Fatal(err)
	}
	m := New(Options{Probe: func() Probe { return Probe{Root: root, Mounted: true, FreeBytes: Floor} }})
	env := m.PrepareSlot(Slot{Repo: "pair", Identity: "git", Number: 4})
	if len(env) != 3 {
		t.Fatalf("shared parent rejected: %v", env)
	}
	after, err := os.Stat(parent)
	if err != nil || !os.SameFile(before, after) || after.Mode().Perm() != 0755 {
		t.Fatalf("parent changed: %v %v", after, err)
	}
	slot, err := os.Stat(strings.TrimPrefix(env[0], "TMPDIR="))
	if err != nil || slot.Mode().Perm() != 0700 {
		t.Fatalf("slot not private: %v %v", slot, err)
	}
	if _, err := m.Sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestUnsafeSharedTmpRefusesPrepareAndSweep(t *testing.T) {
	for _, mode := range []os.FileMode{0777, 0775, 0757} {
		t.Run(mode.String(), func(t *testing.T) {
			m, root, _ := fixture(t)
			parent := filepath.Dir(root)
			if err := os.Chmod(parent, mode); err != nil {
				t.Fatal(err)
			}
			if env := m.PrepareSlot(Slot{Repo: "pair", Identity: "git-common", Number: 3}); env != nil {
				t.Fatal("unsafe shared directory accepted")
			}
			if _, err := m.Sweep(context.Background()); err == nil {
				t.Fatal("unsafe shared directory swept")
			}
		})
	}
}

func TestPrepareHeldOwnerLockFallsBackPromptly(t *testing.T) {
	m, root, _ := fixture(t)
	owners := filepath.Join(filepath.Dir(filepath.Dir(root)), "couch-scratch", "owners")
	lock, err := unix.Open(filepath.Join(owners, ".lock"), unix.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(lock)
	if err = unix.Flock(lock, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer unix.Flock(lock, unix.LOCK_UN)
	before, err := os.ReadFile(filepath.Join(owners, "pair-slot3.json"))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan []string, 1)
	go func() { done <- m.PrepareSlot(Slot{Repo: "pair", Identity: "git-common", Number: 3}) }()
	select {
	case env := <-done:
		if env != nil {
			t.Fatal("held lock accepted", env)
		}
	case <-time.After(time.Second):
		unix.Flock(lock, unix.LOCK_UN)
		<-done
		t.Fatal("optional scratch blocked behind held owner lock")
	}
	after, err := os.ReadFile(filepath.Join(owners, "pair-slot3.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("owner changed while lease held", err)
	}
}
