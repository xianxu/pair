// Package zellijexec owns synchronous bounded Zellij command processes.
package zellijexec

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Run joins a caller-configured exec.CommandContext and stops its private
// process group on every completion path. The caller owns its context budget,
// arguments, environment, and output observers. Use only for synchronous
// bounded commands, never server creation or interactive attach. Callers own
// mutation authority and namespace admission separately.
// With successful collection it returns the original subprocess error unchanged.
// Cancellation/collection failures are joined; callers normalizing an exit
// diagnostic must require a direct *exec.ExitError, never search a joined error.
func Run(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		return fmt.Errorf("zellij query command already started")
	}
	attr := syscall.SysProcAttr{}
	if cmd.SysProcAttr != nil {
		attr = *cmd.SysProcAttr
	}
	attr.Setpgid, attr.Pgid = true, 0
	cmd.SysProcAttr = &attr
	if cmd.WaitDelay <= 0 || cmd.WaitDelay > time.Second {
		cmd.WaitDelay = time.Second
	}
	stop := func() error {
		if cmd.Process == nil {
			return nil
		}
		return stopGroup(cmd.Process.Pid, syscall.Kill, procutil.ProcessGroupMembers)
	}
	var cancelErr error
	cmd.Cancel = func() error {
		cancelErr = stop()
		return cancelErr
	}
	err := cmd.Run()
	// Wait joins CommandContext's cancellation callback before returning, so its
	// error is safe to inspect here even when the command's exit takes priority.
	cleanupErr := stop()
	if cancelErr != nil || cleanupErr != nil {
		if cancelErr != nil {
			cancelErr = fmt.Errorf("cancel Zellij query group: %w", cancelErr)
		}
		if cleanupErr != nil {
			cleanupErr = fmt.Errorf("collect Zellij query group: %w", cleanupErr)
		}
		return errors.Join(err, cancelErr, cleanupErr)
	}
	return err
}

func stopGroup(pid int, kill func(int, syscall.Signal) error, members func(int) (bool, error)) error {
	err := kill(-pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	if errors.Is(err, syscall.EPERM) {
		// Darwin can retain an empty group briefly after Wait. Resolve only
		// this ambiguous error, using independent exact-group evidence.
		present, lookupErr := members(pid)
		if lookupErr != nil {
			return errors.Join(err, lookupErr)
		}
		if !present {
			return nil
		}
	}
	return err
}
