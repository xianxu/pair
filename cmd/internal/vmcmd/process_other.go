//go:build !darwin && !linux

package vmcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

type ownedProcess struct{}

func configureSnapshotCommand(cmd *exec.Cmd) *ownedProcess {
	cmd.Err = fmt.Errorf("owned subprocess observation requires macOS or Linux")
	return &ownedProcess{}
}
func (o *ownedProcess) Writer(w io.Writer) io.Writer { return w }
func (o *ownedProcess) AwaitTerminal(context.Context) error {
	return fmt.Errorf("owned subprocess observation requires macOS or Linux")
}
func (o *ownedProcess) Finish(context.Context) (error, error) {
	return nil, fmt.Errorf("owned subprocess cleanup requires macOS or Linux")
}
func cloneFallbackAllowed(err error) bool { return errors.Is(err, syscall.ENOTSUP) }
