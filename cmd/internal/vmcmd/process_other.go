//go:build !darwin && !linux

package vmcmd

import (
	"errors"
	"os/exec"
	"syscall"
)

func configureSnapshotCommand(_ *exec.Cmd) {}
func cloneFallbackAllowed(err error) bool  { return errors.Is(err, syscall.ENOTSUP) }
