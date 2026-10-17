package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// Each stream has an independent bound. Returning a write error stops exec's
// copier; cancellation also collects descendants that inherited its pipes.
type staleOutput struct{ buffer bytes.Buffer }

func (b *staleOutput) Write(p []byte) (int, error) {
	remaining := couchmessage.MaxBodyBytes - b.buffer.Len()
	if len(p) > remaining {
		n, _ := b.buffer.Write(p[:remaining])
		return n, io.ErrShortBuffer
	}
	return b.buffer.Write(p)
}

func runStaleStatusCheck(ctx context.Context, file string) (string, error) {
	return runStaleCommand(exec.CommandContext(ctx, "tl-lead", "status", "check", file))
}

func runStaleCommand(cmd *exec.Cmd) (string, error) {
	var stdout, stderr staleOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	kill := func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.Cancel = kill
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	// The command owns its group through completion, including descendants
	// that closed their pipes before the parent exited successfully.
	_ = kill()
	if err != nil {
		return stdout.buffer.String(), fmt.Errorf("tl-lead status check: %w; stderr %q", err, stderr.buffer.String())
	}
	return stdout.buffer.String(), nil
}
