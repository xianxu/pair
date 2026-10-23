//go:build darwin

package vmcmd

import (
	"os"

	"golang.org/x/sys/unix"
)

const snapshotNonblock = unix.O_NONBLOCK

func nativeCloneFile(source, parent *os.File, name string) error {
	return unix.Fclonefileat(int(source.Fd()), int(parent.Fd()), name, 0)
}
