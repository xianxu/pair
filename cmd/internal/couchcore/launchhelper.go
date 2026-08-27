package couchcore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

const launchAckByte byte = 0x1

type launchTargetExec func(argv, env []string) error
type blockedChildStarter func(dir string, argv, env []string, extraFiles []*os.File) (Handle, error)

// startBlockedChild is the single parent-side authority for the acknowledgement
// pipe and helper wrapper. ExecRunner and PtyRunner vary only in how the helper
// process itself is started.
func startBlockedChild(start blockedChildStarter, helper, dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("start blocked: empty argv")
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("start blocked: acknowledgement pipe: %w", err)
	}
	h, err := start(dir, launchHelperArgv(helper, timeout, argv), env, []*os.File{reader})
	closeErr := reader.Close()
	if err != nil {
		return nil, errors.Join(err, closeErr, writer.Close())
	}
	if closeErr != nil {
		_ = writer.Close()
		return nil, closeErr
	}
	return newAcknowledgedHandle(h, writer), nil
}

// RunLaunchHelper blocks before target exec until its parent supplies one exact
// acknowledgement byte. Closing the channel or reaching the deadline closes
// the descriptor and returns without starting the target.
func RunLaunchHelper(ack io.ReadCloser, timeout time.Duration, argv, env []string, execTarget launchTargetExec) error {
	if ack == nil {
		return errors.New("launch helper: nil acknowledgement channel")
	}
	if timeout <= 0 {
		_ = ack.Close()
		return errors.New("launch helper: positive timeout is required")
	}
	if len(argv) == 0 {
		_ = ack.Close()
		return errors.New("launch helper: empty target argv")
	}
	if execTarget == nil {
		_ = ack.Close()
		return errors.New("launch helper: nil target exec")
	}

	type readResult struct {
		byte byte
		err  error
	}
	result := make(chan readResult, 1)
	go func() {
		var one [1]byte
		_, err := io.ReadFull(ack, one[:])
		result <- readResult{byte: one[0], err: err}
	}()

	select {
	case got := <-result:
		if err := ack.Close(); err != nil {
			return fmt.Errorf("launch helper: close acknowledgement channel: %w", err)
		}
		if got.err != nil {
			return fmt.Errorf("launch helper: acknowledgement unavailable: %w", got.err)
		}
		if got.byte != launchAckByte {
			return fmt.Errorf("launch helper: invalid acknowledgement byte 0x%x", got.byte)
		}
	case <-time.After(timeout):
		_ = ack.Close()
		return fmt.Errorf("launch helper: acknowledgement timed out after %s", timeout)
	}

	if err := execTarget(argv, env); err != nil {
		return fmt.Errorf("launch helper: exec target: %w", err)
	}
	return nil
}

func execLaunchTarget(argv, env []string) error {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	return syscall.Exec(path, argv, env)
}

// LaunchHelperMain is the small command's testable shell. Descriptor 3 is the
// sole inherited acknowledgement channel; it is closed before target exec.
func LaunchHelperMain(args []string, stderr io.Writer) int {
	if len(args) < 3 || args[1] != "--" {
		fmt.Fprintln(stderr, "pair-launch-helper: usage: pair-launch-helper <timeout> -- <target> [args...]")
		return 2
	}
	timeout, err := time.ParseDuration(args[0])
	if err != nil || timeout <= 0 {
		fmt.Fprintf(stderr, "pair-launch-helper: invalid timeout %q\n", args[0])
		return 2
	}
	ack := os.NewFile(3, "couch-launch-ack")
	if ack == nil {
		fmt.Fprintln(stderr, "pair-launch-helper: acknowledgement descriptor unavailable")
		return 1
	}
	if err := RunLaunchHelper(ack, timeout, args[2:], os.Environ(), execLaunchTarget); err != nil {
		fmt.Fprintf(stderr, "pair-launch-helper: %v\n", err)
		return 1
	}
	return 0
}

func launchHelperArgv(configured string, timeout time.Duration, target []string) []string {
	helper := configured
	if helper == "" {
		helper = defaultLaunchHelperPath()
	}
	argv := []string{helper, timeout.String(), "--"}
	return append(argv, target...)
}

func defaultLaunchHelperPath() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "pair-launch-helper")
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate
		}
	}
	return "pair-launch-helper"
}

type acknowledgedHandle struct {
	Handle
	mu     sync.Mutex
	writer *os.File
}

func newAcknowledgedHandle(handle Handle, writer *os.File) BlockedHandle {
	blocked := &acknowledgedHandle{Handle: handle, writer: writer}
	if terminal, ok := handle.(TerminalHandle); ok {
		return &acknowledgedTerminalHandle{acknowledgedHandle: blocked, terminal: terminal.Terminal()}
	}
	return blocked
}

type acknowledgedTerminalHandle struct {
	*acknowledgedHandle
	terminal *ptychild.Child
}

func (h *acknowledgedTerminalHandle) Terminal() *ptychild.Child { return h.terminal }

func (h *acknowledgedHandle) Acknowledge() error { return h.resolve(true) }
func (h *acknowledgedHandle) Cancel() error      { return h.resolve(false) }

func (h *acknowledgedHandle) resolve(acknowledge bool) error {
	h.mu.Lock()
	writer := h.writer
	h.writer = nil
	h.mu.Unlock()
	if writer == nil {
		return errors.New("blocked start acknowledgement already resolved")
	}
	if !acknowledge {
		return writer.Close()
	}
	_, writeErr := writer.Write([]byte{launchAckByte})
	return errors.Join(writeErr, writer.Close())
}
