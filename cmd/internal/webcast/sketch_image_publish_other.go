//go:build !darwin && !linux

package webcast

import (
	"errors"
	"os"
)

func publishSketchImageExclusive(parent *os.File, source, target string) error {
	return errors.New("exclusive sketch-image export unsupported on this platform")
}
