//go:build darwin || linux

package vmcmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func configureSnapshotCommand(cmd *exec.Cmd) func() error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stop := func() error {
		if cmd.Process == nil {
			return nil
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if errors.Is(err, syscall.EPERM) {
			present, lookupErr := procutil.ProcessGroupMembers(cmd.Process.Pid)
			if lookupErr == nil && !present {
				return nil
			}
			return errors.Join(err, lookupErr)
		}
		return err
	}
	cmd.Cancel = func() error {
		if err := stop(); err != nil {
			return err
		}
		return os.ErrProcessDone
	}
	return func() error {
		if err := stop(); err != nil {
			return err
		}
		if cmd.Process == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return awaitSnapshotGroupGone(ctx, func() (bool, error) {
			err := syscall.Kill(-cmd.Process.Pid, 0)
			if errors.Is(err, syscall.ESRCH) {
				return false, nil
			}
			if errors.Is(err, syscall.EPERM) {
				return procutil.ProcessGroupMembers(cmd.Process.Pid)
			}
			return err == nil, err
		})
	}
}

func cloneFallbackAllowed(err error) bool {
	return errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP)
}

// A signal is a request. Confirm disappearance before releasing owned inputs.
func awaitSnapshotGroupGone(ctx context.Context, present func() (bool, error)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		found, err := present()
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		timer := time.NewTimer(5 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
