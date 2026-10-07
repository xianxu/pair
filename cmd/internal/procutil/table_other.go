//go:build !darwin && !linux

package procutil

import (
	"errors"
)

// Table has no single-read source on this platform: reap refuses rather than
// pair a parent and an identity read at different moments (#399).
func Table() ([]Process, error) {
	return nil, errors.New("process table unavailable on this platform")
}
