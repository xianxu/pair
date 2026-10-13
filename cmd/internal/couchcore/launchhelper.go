package couchcore

import (
	"context"
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
	"golang.org/x/sys/unix"
)

const launchAckByte byte = 0x1

type launchTargetExec func(argv, env []string) error
type blockedChildStarter func(dir string, argv, env []string, extraFiles []*os.File) (Handle, error)

// startBlockedChild is the single parent-side authority for the acknowledgement
// pipe and helper wrapper. ExecRunner and PtyRunner vary only in how the helper
// process itself is started.
//
// A nonzero nice lowers the helper's process group while the helper is still
// parked before exec (#498): nothing in the group has forked yet, so the target
// and everything it starts -- builds, suites, agents -- inherit it with no race
// and no change to argv or identity, and yield to Couch and the operator's
// terminal. Couch keeps its own priority; raising it would need root.
//
// The value is absolute, and only root may lower a niceness, so a Couch that
// already runs nicer than the setting (started from a niced shell or slot) has
// its request refused; its children then keep the niceness they inherited,
// which already yields at least as much. Slots run at max(nice, Couch's own).
func startBlockedChild(ctx context.Context, start blockedChildStarter, helper string, nice int, dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
		return nil, errors.Join(closeErr, newAcknowledgedHandle(h, writer).Cancel())
	}
	blocked := newAcknowledgedHandle(h, writer)
	if nice != 0 {
		// Every Couch child leads its own process group, so the pgid is its pid.
		err := unix.Setpriority(unix.PRIO_PGRP, h.PID(), nice)
		if err != nil && !errors.Is(err, unix.EACCES) && !errors.Is(err, unix.EPERM) {
			cancelErr := blocked.Cancel()
			if cancelErr == nil {
				_ = blocked.Wait()
			}
			return nil, errors.Join(fmt.Errorf("start blocked: lower %s to nice %d (COUCH_SLOT_NICE): %w", argv[0], nice, err), cancelErr)
		}
	}
	return blocked, nil
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

// A canceled pre-ack terminal cannot be adopted by a presenter; Cancel owns
// publication and endpoint disposal as well as the acknowledgement pipe.
func (h *acknowledgedTerminalHandle) Cancel() error {
	if err := h.acknowledgedHandle.Cancel(); err != nil {
		return err
	}
	return h.terminal.Close()
}

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

func (h *acknowledgedHandle) LaunchOutput() (string, []byte) {
	if output, ok := h.Handle.(LaunchOutputHandle); ok {
		return output.LaunchOutput()
	}
	return "", nil
}
