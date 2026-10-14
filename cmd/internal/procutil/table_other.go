//go:build !darwin && !linux

package procutil

// Table refuses inspection on platforms without a native process table.
func Table() ([]Process, error) { return nil, ErrUnsupported }
