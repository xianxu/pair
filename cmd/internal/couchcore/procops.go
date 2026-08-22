package couchcore

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Liveness is deliberately three-valued.
//
// A two-valued probe collapses "the process is gone" and "I could not check"
// into the same answer, and couch PRUNES on that answer -- so a probe that
// fails for any reason (a sandbox blocking the syscall, an exec limit, a
// transient error) silently deletes a live actor's record and then lets a
// second agent onto its tree. That happened in smoke testing and destroyed a
// running session's registration.
//
// So: prune only on Dead. Unknown must fail CLOSED, keeping the record and the
// guard it implies.
type Liveness int

const (
	Unknown Liveness = iota
	Live
	Dead
)

func (l Liveness) String() string {
	switch l {
	case Live:
		return "live"
	case Dead:
		return "dead"
	default:
		return "unknown"
	}
}

// ProcOps is the seam for out-of-process liveness.
//
// It exists because `couch start` blocks (the child inherits our stdio), so
// every read command runs in a SECOND process with no Handle. Liveness is
// recomputed from the persisted {PID, Identity} pair rather than held in
// memory, and Identity is a kernel start token so a recycled PID does not
// impersonate the original actor (sessionwatch/run.go:143 re-authorizes for
// the same reason).
type ProcOps interface {
	// Exists distinguishes gone from unknowable, which is the whole point.
	Exists(pid int) Liveness
	// Identity returns the kernel start token, or an error if it could not be
	// read -- which is NOT the same as the process being absent.
	Identity(pid int) (string, error)
	// Signal delivers sig to pid. couch stop has no Handle -- the process that
	// spawned the child is a different one, blocked in Wait -- so stopping
	// goes through the pid, guarded by the identity token.
	Signal(pid int, sig os.Signal) error
}

type OSProcOps struct{}

var _ ProcOps = OSProcOps{}

// Exists uses syscall.Kill(pid, 0) directly rather than forking `kill -0`.
// Forking makes the probe fail whenever spawning is restricted, and a failed
// fork is indistinguishable from a dead process -- exactly the conflation this
// type exists to avoid.
func (OSProcOps) Exists(pid int) Liveness {
	if pid <= 0 {
		return Dead
	}
	err := syscall.Kill(pid, 0)
	switch {
	case err == nil:
		return Live
	case errors.Is(err, syscall.ESRCH):
		return Dead
	case errors.Is(err, syscall.EPERM):
		// It exists; it just is not ours to signal.
		return Live
	default:
		return Unknown
	}
}

func (OSProcOps) Identity(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid pid %d", pid)
	}
	id := procutil.Identity(strconv.Itoa(pid))
	if id == "" {
		return "", fmt.Errorf("no identity token for pid %d", pid)
	}
	return id, nil
}

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

// FakeProcOps models a pid table, including the case that matters most: a
// probe that cannot answer.
type FakeProcOps struct {
	ids         map[int]string
	unknown     map[int]bool
	Signals     map[int][]os.Signal
	DiesOn      map[int]os.Signal
	IdentityErr map[int]bool
}

var _ ProcOps = (*FakeProcOps)(nil)

func NewFakeProcOps() *FakeProcOps {
	return &FakeProcOps{
		ids:         map[int]string{},
		unknown:     map[int]bool{},
		Signals:     map[int][]os.Signal{},
		DiesOn:      map[int]os.Signal{},
		IdentityErr: map[int]bool{},
	}
}

func (f *FakeProcOps) Set(pid int, identity string) { f.ids[pid] = identity }
func (f *FakeProcOps) Kill(pid int)                 { delete(f.ids, pid) }

// SetUnknown models a probe that cannot answer for this pid.
func (f *FakeProcOps) SetUnknown(pid int) { f.unknown[pid] = true }

func (f *FakeProcOps) Exists(pid int) Liveness {
	if f.unknown[pid] {
		return Unknown
	}
	if _, ok := f.ids[pid]; ok {
		return Live
	}
	return Dead
}

func (f *FakeProcOps) Identity(pid int) (string, error) {
	if f.unknown[pid] || f.IdentityErr[pid] {
		return "", fmt.Errorf("cannot read identity for pid %d", pid)
	}
	id, ok := f.ids[pid]
	if !ok {
		return "", fmt.Errorf("no such pid %d", pid)
	}
	return id, nil
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
