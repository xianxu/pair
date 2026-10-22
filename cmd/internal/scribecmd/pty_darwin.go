//go:build darwin

package scribecmd

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

func startPTY(cmd *exec.Cmd) (*startedPTY, error) {
	return startPTYWithOpen(cmd, pty.Open)
}

// Keep the upstream Start orchestration, but transfer the slave to our drain
// owner instead of closing it on return. XNU otherwise discards unread output
// when a non-terminal-aware child closes its last slave. Session-leader exit
// drains and revokes the tty, delivering EOF even while this lease is held.
func startPTYWithOpen(cmd *exec.Cmd, open func() (*os.File, *os.File, error)) (*startedPTY, error) {
	master, slave, err := open()
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
