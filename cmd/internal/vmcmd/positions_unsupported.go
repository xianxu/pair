//go:build !darwin && !linux

package vmcmd

import (
	"context"
	"fmt"
	"io"
)

type Position struct{}

func AcquirePosition(context.Context, string, Lane, io.Writer) (*Position, error) {
	return nil, fmt.Errorf("VM position locking is unsupported on this platform")
}
func (*Position) Dir() string { return "" }
func (*Position) ReadClaim() (LaunchClaim, bool, error) {
	return LaunchClaim{}, false, fmt.Errorf("VM positions unsupported")
}
func (*Position) WriteClaim(LaunchClaim) error { return fmt.Errorf("VM positions unsupported") }
func (*Position) ClearClaim(string) error      { return fmt.Errorf("VM positions unsupported") }
func (*Position) Close() error                 { return nil }
