//go:build !darwin && !linux

package historyarchive

import (
	"errors"
	"os"
)

func publishRestoreExclusive(stage *os.File, source string, parent *os.File, target string) error {
	return errors.New("exclusive restore publication is unsupported on this platform")
}
