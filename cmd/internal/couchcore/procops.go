package couchcore

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// ProcOps is the seam for out-of-process liveness.
//
// It exists because `couch start` blocks (the child inherits our stdio), so
// every read command runs in a SECOND process with no Handle. Liveness is
// therefore recomputed from the persisted {PID, Identity} pair rather than
// held in memory.
//
// Identity is a kernel start token: comparing PIDs alone would report a
// recycled PID as the original actor. sessionwatch/run.go:143 re-authorizes
// for exactly this reason.
type ProcOps interface {
	Identity(pid int) string
	Alive(pid int) bool
	// Signal delivers sig to pid. couch stop has no Handle -- the process
	// that spawned the child is a different one, blocked in Wait -- so
	// stopping goes through the pid, guarded by the identity token.
	Signal(pid int, sig os.Signal) error
}

type OSProcOps struct{}

var _ ProcOps = OSProcOps{}

func (OSProcOps) Identity(pid int) string { return procutil.Identity(strconv.Itoa(pid)) }
func (OSProcOps) Alive(pid int) bool      { return procutil.Alive(strconv.Itoa(pid)) }

func (OSProcOps) Signal(pid int, sig os.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find pid %d: %w", pid, err)
	}
	if err := proc.Signal(sig); err != nil {
		return fmt.Errorf("signal pid %d: %w", pid, err)
	}
	return nil
}

// TermSignal is what `couch stop` sends: SIGTERM, so a child that wants to
// clean up gets the chance. Nothing escalates to SIGKILL automatically -- an
// agent mid-write is worse to truncate than to leave running.
var TermSignal os.Signal = syscall.SIGTERM

// FakeProcOps lets a test place a PID at a chosen identity, including the
// PID-reuse case where the pid is alive but is a different process.
type FakeProcOps struct {
	ids     map[int]string
	Signals map[int][]os.Signal
	// DiesOn scripts which signal ends a pid, so a test can model both a
	// child that exits on SIGTERM and one that ignores it.
	DiesOn map[int]os.Signal
}

var _ ProcOps = (*FakeProcOps)(nil)

func NewFakeProcOps() *FakeProcOps {
	return &FakeProcOps{
		ids:     map[int]string{},
		Signals: map[int][]os.Signal{},
		DiesOn:  map[int]os.Signal{},
	}
}

func (f *FakeProcOps) Signal(pid int, sig os.Signal) error {
	if _, ok := f.ids[pid]; !ok {
		return fmt.Errorf("no such pid %d", pid)
	}
	f.Signals[pid] = append(f.Signals[pid], sig)
	if fatal, ok := f.DiesOn[pid]; ok && fatal == sig {
		delete(f.ids, pid)
	}
	return nil
}

func (f *FakeProcOps) Set(pid int, identity string) { f.ids[pid] = identity }
func (f *FakeProcOps) Kill(pid int)                 { delete(f.ids, pid) }

func (f *FakeProcOps) Identity(pid int) string { return f.ids[pid] }
func (f *FakeProcOps) Alive(pid int) bool      { _, ok := f.ids[pid]; return ok }
