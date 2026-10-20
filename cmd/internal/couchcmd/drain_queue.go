package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type drainQueue interface {
	Pause(context.Context, string, string, bool) error
	Resume(context.Context, string) error
	Status(context.Context) (QueueSnapshot, error)
}

type sdlcDrainQueue struct {
	workspace  string
	executable string
}

func (q sdlcDrainQueue) Pause(ctx context.Context, owner, reason string, fast bool) error {
	args := []string{"queue", "pause", "--reason", reason, "--owner", owner}
	if fast {
		args = append(args, "--cancel-running")
	}
	_, err := q.run(ctx, args)
	return err
}
func (q sdlcDrainQueue) Resume(ctx context.Context, owner string) error {
	_, err := q.run(ctx, []string{"queue", "resume", "--owner", owner})
	return err
}
func (q sdlcDrainQueue) Status(ctx context.Context) (QueueSnapshot, error) {
	b, err := q.run(ctx, []string{"queue", "status", "--json"})
	if err != nil {
		return QueueSnapshot{State: "unknown", Detail: drainQueueDiagnostic(err.Error(), 4096)}, err
	}
	return decodeQueueSnapshot(b)
}

type drainQueueBudget struct {
	mu        sync.Mutex
	remaining int
}
type drainQueueOutput struct {
	buffer bytes.Buffer
	budget *drainQueueBudget
}

func (b *drainQueueOutput) Write(p []byte) (int, error) {
	b.budget.mu.Lock()
	defer b.budget.mu.Unlock()
	left := b.budget.remaining
	defer func() { b.budget.remaining -= min(len(p), left) }()
	if len(p) > left {
		n, _ := b.buffer.Write(p[:left])
		return n, io.ErrShortBuffer
	}
	return b.buffer.Write(p)
}

func (q sdlcDrainQueue) run(parent context.Context, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	name := q.executable
	if name == "" {
		name = "sdlc"
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = q.workspace
	budget := &drainQueueBudget{remaining: maxDrainRecordBytes}
	out := drainQueueOutput{budget: budget}
	stderr := drainQueueOutput{budget: budget}
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	// Each call owns its process group, including pipe-holding descendants. This
	// matches runStaleCommand's bounded ownership without borrowing its text cap.
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
	_ = kill()
	if err != nil {
		return nil, fmt.Errorf("sdlc queue %s unavailable/uncertain: %w; stderr %q", args[1], err, drainQueueDiagnostic(stderr.buffer.String(), 1024))
	}
	return out.buffer.Bytes(), nil
}
