package scribecmd

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

// startedPTY owns the optional Darwin slave until master output is drained.
// The caller owns the master and is the sole authority to wait for the child.
type startedPTY struct {
	master     *os.File
	slave      io.Closer
	release    sync.Once
	releaseErr error
}

func (p *startedPTY) releaseSlave() error {
	p.release.Do(func() {
		if p.slave != nil {
			p.releaseErr = p.slave.Close()
		}
	})
	return p.releaseErr
}

// abortPTY is only for failures after Start but before normal output draining.
// Kill the owned direct child, then close both descriptors BEFORE waiting: on
// Darwin a child can itself be blocked in exit waiting for unread PTY output.
func abortPTY(cmd *exec.Cmd, p *startedPTY) error {
	killErr := cmd.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	masterErr := p.master.Close()
	slaveErr := p.releaseSlave()
	waitErr := cmd.Wait()
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		waitErr = nil
	} // expected after abort
	return errors.Join(killErr, masterErr, slaveErr, waitErr)
}
