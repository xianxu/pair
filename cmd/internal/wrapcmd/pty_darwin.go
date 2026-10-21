//go:build darwin

package wrapcmd

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

func startPTY(cmd *exec.Cmd) (*startedPTY, error) {
	return startPTYWithOpen(cmd, pty.Open)
}

// Match upstream Start while retaining the slave through output drain. Darwin
// otherwise discards unread bytes when a child closes its last slave. Leader
// exit drains/revokes the terminal; closing stdio alone does not end ownership.
func startPTYWithOpen(cmd *exec.Cmd, open func() (*os.File, *os.File, error)) (*startedPTY, error) {
	master, slave, err := open()
	if err != nil {
		if master != nil {
			_ = master.Close()
		}
		if slave != nil {
			_ = slave.Close()
		}
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
