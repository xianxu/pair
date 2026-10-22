//go:build darwin

package scribecmd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

// ptyAllocAttempts bounds the retry of a transient ENXIO from XNU's ptmx
// allocator. Measured on darwin/arm64 (#809), failures per allocation attempt at
// N concurrent allocators:
//
//	N=1  0/20,000    N=2  0/40,000    N=4  2/80,000
//	N=8  46/160,000  N=8  52/160,000  N=16 218/320,000
//
// Monotonic in concurrency, so a race in the allocator rather than a resource
// limit — peak occupancy was 230 of kern.tty.ptmx_max 511. At 8 allocators all
// 52 failures cleared on the immediately following attempt; at 16, 217 of 218
// did and one needed a third. Three attempts covers that measurement; one retry
// does not. No backoff: recovery was immediate in every observation.
const ptyAllocAttempts = 3

func startPTY(cmd *exec.Cmd) (*startedPTY, error) {
	return startPTYWithOpen(cmd, openPTY)
}

// openPTY allocates a pty pair, riding out the transient ENXIO. Test fixtures
// allocate through it too: a fixture doing its own pty.Open is exposed to the
// same race, and its failure there reads as a failure of Run.
func openPTY() (*os.File, *os.File, error) { return openPTYRetry(pty.Open) }

func openPTYRetry(open func() (*os.File, *os.File, error)) (*os.File, *os.File, error) {
	var master, slave *os.File
	var err error
	// Retrying leaks nothing: open() closes the ptmx descriptor it already
	// holds on every error path after allocating it.
	for attempt := 0; attempt < ptyAllocAttempts; attempt++ {
		master, slave, err = open()
		if err == nil || !isTransientPTYAlloc(err) {
			break
		}
	}
	return master, slave, err
}

// isTransientPTYAlloc reports whether err is ENXIO from allocating a pty.
//
// The sign has to be normalised. creack/pty reaches /dev/ptmx through
// syscall.Open and its three ioctls through ioctlInner's raw syscall.Syscall,
// and a failing ptmx allocation has been measured arriving from BOTH as an
// Errno holding the negated code — int value -6, rendering "errno -6" — while an
// ordinary failure such as ENOENT arrives positive. errors.Is(err,
// syscall.ENXIO) compares by value, so it matches only the positive spelling and
// would make this guard silently never fire on the one #809 actually observed.
func isTransientPTYAlloc(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	n := int(errno)
	if n < 0 {
		n = -n
	}
	return syscall.Errno(n) == syscall.ENXIO
}

// Keep the upstream Start orchestration, but transfer the slave to our drain
// owner instead of closing it on return. XNU otherwise discards unread output
// when a non-terminal-aware child closes its last slave. Session-leader exit
// drains and revokes the tty, delivering EOF even while this lease is held.
func startPTYWithOpen(cmd *exec.Cmd, open func() (*os.File, *os.File, error)) (*startedPTY, error) {
	master, slave, err := openPTYRetry(open)
	if err != nil {
		return nil, err
	}
	if cmd.Stdin == nil {
		cmd.Stdin = slave
	}
	if cmd.Stdout == nil {
		cmd.Stdout = slave
	}
	if cmd.Stderr == nil {
		cmd.Stderr = slave
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setctty = true
	if err := cmd.Start(); err != nil {
		_ = master.Close()
		_ = slave.Close()
		return nil, err
	}
	return &startedPTY{master: master, slave: slave}, nil
}
