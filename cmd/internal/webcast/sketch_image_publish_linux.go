//go:build linux

package webcast

import (
	"golang.org/x/sys/unix"
	"os"
)

func publishSketchImageExclusive(parent *os.File, source, target string) error {
	return unix.Renameat2(int(parent.Fd()), source, int(parent.Fd()), target, unix.RENAME_NOREPLACE)
}
