package couchcore

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// PtyRunner is the Runner whose children get their own pty, so a console can
// route the operator's terminal between them.
//
// It sits behind the SAME seam as ExecRunner rather than replacing it: the two
// genuinely differ in what they can offer, and `--no-console` needs the
// stdio-inheriting one to stay a live production path (Decision 2). Which one a
// Couch gets is the composition root's decision; nothing in the domain learns
// that a terminal exists.
type PtyRunner struct {
	LaunchHelper string

	// Size supplies a new child's dimensions, called at Start. A func rather
	// than a value because the console's size changes: the reserved row means
	// a child is one row shorter than the host, and the host is resizable.
	Size func() ptychild.Size

	// Sink receives every chunk a child writes, tagged with its handle id.
	// Installed INSIDE Start so a child that writes immediately cannot lose
	// chunks from the live path to a race with the caller wiring it up.
	Sink func(id string, chunk []byte)
}

var _ Runner = (*PtyRunner)(nil)

// Terminal is the pty capability a Handle may expose.
//
// It is deliberately the CONCRETE *ptychild.Child rather than an interface:
// FakeRunner's terminal double is a ptychild.NewFakeChild, which is the same
// type, so production flow and test flow share this boundary exactly. An
// interface here would let the fake drift into a different shape, which is the
// ARCH-MOCK failure the seam exists to prevent.
type TerminalHandle interface {
	Handle
	Terminal() *ptychild.Child
}

func (r *PtyRunner) Start(dir string, argv, env []string) (Handle, error) {
	return r.start(dir, argv, env, nil)
}

func (r *PtyRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	return startBlockedChild(ctx, r.start, r.LaunchHelper, dir, argv, env, timeout)
}

func (r *PtyRunner) start(dir string, argv, env []string, extraFiles []*os.File) (Handle, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("start: empty argv")
	}
	size := ptychild.Size{Rows: 24, Cols: 80}
	if r.Size != nil {
		size = r.Size()
	}

	// The id is minted BEFORE the child exists and is never derived from the
	// pid.
	//
	// The pump can call the sink before ptychild.Start has even returned, so a
	// handle whose ID() reads a field Start assigns afterwards is a genuine
	// data race -- caught by -race, and in production it would have tagged the
	// first chunks of every session with a zero id. ExecRunner can use the pid
	// because nothing reads ITS id from another goroutine; this one is read
	// from the pump.
	h := &ptyHandle{id: fmt.Sprintf("couch-pty-%d", ptySeq.Add(1))}
	child, err := ptychild.Start(ptychild.Options{
		Dir:        dir,
		Argv:       argv,
		Env:        env,
		Size:       size,
		ExtraFiles: extraFiles,
		Sink: func(chunk []byte) {
			if r.Sink != nil {
				r.Sink(h.ID(), chunk)
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("start %s in %s: %w", argv[0], dir, err)
	}
	h.child = child
	h.pid = child.PID()
	h.identity = procutil.Identity(strconv.Itoa(h.pid))
	return h, nil
}

// ptySeq numbers pty handles. Package-level so ids stay unique across runners,
// which matters once #147 puts more than one console in a process.
var ptySeq atomic.Uint64

type ptyHandle struct {
	id       string
	child    *ptychild.Child
	pid      int
	identity string
}

var _ TerminalHandle = (*ptyHandle)(nil)

func (h *ptyHandle) ID() string                 { return h.id }
func (h *ptyHandle) PID() int                   { return h.pid }
func (h *ptyHandle) Identity() string           { return h.identity }
func (h *ptyHandle) Terminal() *ptychild.Child  { return h.child }
func (h *ptyHandle) Alive() bool                { return ownedProcessGroupAlive(h.pid) }
func (h *ptyHandle) Signal(sig os.Signal) error { return signalOwnedProcessGroup(h.pid, sig) }
func (h *ptyHandle) Wait() int                  { return h.child.Wait() }
