//go:build linux

package historyarchive

import (
	"golang.org/x/sys/unix"
	"os"
)

func publishRestoreExclusive(stage *os.File, source string, parent *os.File, target string) error {
	return unix.Renameat2(int(stage.Fd()), source, int(parent.Fd()), target, unix.RENAME_NOREPLACE)
}
