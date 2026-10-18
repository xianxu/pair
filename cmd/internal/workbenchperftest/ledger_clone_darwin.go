//go:build darwin

package workbenchperftest

import (
	"errors"

	"golang.org/x/sys/unix"
)

const ledgerCloneAvailable = true

func cloneLedgerFile(source, destination string) error {
	err := unix.Clonefile(source, destination, 0)
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EXDEV) {
		return errors.Join(errLedgerCloneUnsupported, err)
	}
	return err
}
