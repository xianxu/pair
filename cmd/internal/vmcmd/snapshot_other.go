//go:build !darwin

package vmcmd

import (
	"os"
	"syscall"
)

const snapshotNonblock = 0

func nativeCloneFile(_, _ *os.File, _ string) error { return syscall.ENOTSUP }
