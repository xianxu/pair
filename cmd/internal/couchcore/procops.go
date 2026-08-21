package couchcore

import (
	"strconv"

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
}

type OSProcOps struct{}

var _ ProcOps = OSProcOps{}

func (OSProcOps) Identity(pid int) string { return procutil.Identity(strconv.Itoa(pid)) }
func (OSProcOps) Alive(pid int) bool      { return procutil.Alive(strconv.Itoa(pid)) }

// FakeProcOps lets a test place a PID at a chosen identity, including the
// PID-reuse case where the pid is alive but is a different process.
type FakeProcOps struct{ ids map[int]string }

var _ ProcOps = (*FakeProcOps)(nil)

func NewFakeProcOps() *FakeProcOps { return &FakeProcOps{ids: map[int]string{}} }

func (f *FakeProcOps) Set(pid int, identity string) { f.ids[pid] = identity }
func (f *FakeProcOps) Kill(pid int)                 { delete(f.ids, pid) }

func (f *FakeProcOps) Identity(pid int) string { return f.ids[pid] }
func (f *FakeProcOps) Alive(pid int) bool      { _, ok := f.ids[pid]; return ok }
