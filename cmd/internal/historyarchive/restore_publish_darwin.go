//go:build darwin

package historyarchive

import (
	"golang.org/x/sys/unix"
	"os"
)

// The kernel arbitrates destination existence atomically, including symlinks.
func publishRestoreExclusive(stage *os.File, source string, parent *os.File, target string) error {
	return unix.RenameatxNp(int(stage.Fd()), source, int(parent.Fd()), target, unix.RENAME_EXCL)
}
