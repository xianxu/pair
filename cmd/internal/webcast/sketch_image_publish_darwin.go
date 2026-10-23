//go:build darwin

package webcast

import (
	"golang.org/x/sys/unix"
	"os"
)

func publishSketchImageExclusive(parent *os.File, source, target string) error {
	return unix.RenameatxNp(int(parent.Fd()), source, int(parent.Fd()), target, unix.RENAME_EXCL)
}
