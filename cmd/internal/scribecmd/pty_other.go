//go:build !darwin

package scribecmd

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

func openPTY() (*os.File, *os.File, error) { return pty.Open() }

func startPTY(cmd *exec.Cmd) (*startedPTY, error) {
	master, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &startedPTY{master: master}, nil
}
