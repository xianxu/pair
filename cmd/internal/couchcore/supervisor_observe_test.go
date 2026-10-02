package couchcore

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

func TestObserveSupervisorReadOnly(t *testing.T) {
	ns := testCouchNamespace(t)
	proc := testOwnerProc(t, "owner")
	o, e := ObserveSupervisor(ns, proc, nil)
	if e != nil || o.Held {
		t.Fatal(o, e)
	}
	entries, _ := os.ReadDir(ns.Dir())
	if len(entries) != 0 {
		t.Fatal("probe created files")
	}
	l, e := AcquireSupervisorLease(ns, proc)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	o, e = ObserveSupervisor(ns, proc, nil)
	if e != nil || !o.Held || o.Owner == nil {
		t.Fatal(o, e)
	}
	o, e = ObserveSupervisor(ns, proc, l)
	if e != nil || o.Held {
		t.Fatal(o, e)
	}
	os.WriteFile(filepath.Join(ns.Dir(), supervisorOwnerName), []byte("{"), 0600)
	o, e = ObserveSupervisor(ns, proc, nil)
	if e == nil || !o.Held {
		t.Fatal(o, e)
	}
}
func TestObserveSupervisorRejectsSymlinksAndNonregularLocks(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			ns := testCouchNamespace(t)
			p := filepath.Join(ns.Dir(), supervisorLockName)
			switch kind {
			case "symlink":
				target := filepath.Join(t.TempDir(), "target")
				os.WriteFile(target, []byte("untouched"), 0600)
				os.Symlink(target, p)
			case "directory":
				os.Mkdir(p, 0700)
			case "fifo":
				unix.Mkfifo(p, 0600)
			}
			if _, e := ObserveSupervisor(ns, nil, nil); e == nil {
				t.Fatal("unsafe lock accepted")
			}
		})
	}
}
