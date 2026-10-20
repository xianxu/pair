//go:build darwin || linux

package vmcmd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureSnapshotCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}

func cloneFallbackAllowed(err error) bool {
	return errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP)
}
