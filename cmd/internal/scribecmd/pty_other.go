//go:build !darwin

package scribecmd

import (
	"os/exec"

	"github.com/creack/pty"
)

func startPTY(cmd *exec.Cmd) (*startedPTY, error) {
	master, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &startedPTY{master: master}, nil
}
