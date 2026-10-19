package couchscratch

import (
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
