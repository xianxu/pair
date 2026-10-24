package changelogcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

const distillerObservationBudget = time.Second

// Each stream has one exec copy producer. Only read these buffers after done;
// Wait joins the copy producers, including on cancellation or pipe failure.
type distillerDiagnostic struct{ buffer bytes.Buffer }

func (w *distillerDiagnostic) String() string { return w.buffer.String() }

func (w *distillerDiagnostic) Write(p []byte) (int, error) {
	n := len(p)
	if left := 32*1024 - w.buffer.Len(); left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		_, _ = w.buffer.Write(p)
	}
	return n, nil
}

type distillerChild struct {
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	done        chan struct{}
	err         error // published by closing done
	out, errOut distillerDiagnostic
}

func startDistillerChild(t *testing.T, parent context.Context, root, mode string) *distillerChild {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	child := &distillerChild{cancel: cancel, done: make(chan struct{})}
	child.cmd = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestManagedDistillerProcess$")
	child.cmd.WaitDelay = 2 * time.Second
	child.cmd.Env = append(os.Environ(), "PAIR_TEST_CHANGELOG_CHILD="+mode, "PAIR_TEST_CHANGELOG_ROOT="+root)
	child.cmd.Stdout, child.cmd.Stderr = &child.out, &child.errOut
	if err := child.cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	go func() {
		child.err = child.cmd.Wait() // sole Wait owner on every path
		close(child.done)
	}()
	t.Cleanup(child.stop) // before the earlier fixture t.TempDir cleanup
	return child
}

func (child *distillerChild) stop() {
	child.cancel()
	<-child.done
}

func (child *distillerChild) wait() error {
	<-child.done
	if child.err != nil {
		return fmt.Errorf("distiller exited: %w; stdout=%q stderr=%q", child.err, child.out.String(), child.errOut.String())
	}
	return nil
}

func (child *distillerChild) awaitLease(c *storagegc.Coordinator, owner artifactpath.StorageOwner) error {
	ctx, cancel := context.WithTimeout(context.Background(), distillerObservationBudget)
	defer cancel()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-child.done:
			return fmt.Errorf("distiller exited before lease: %v", child.wait())
		default:
		}
		state, err := c.ReadOwner(owner)
		if err != nil {
			return fmt.Errorf("read lease: %w", err)
		}
		if len(state.Processes) == 1 {
			process, err := storagegc.CurrentProcessIdentity(child.cmd.Process.Pid)
			if err != nil {
				return fmt.Errorf("child identity: %w", err)
			}
			registration := state.Processes[0]
			if registration.Process != process || registration.Role != "changelog-render" || registration.Target != "" {
				return fmt.Errorf("unexpected distiller lease: %+v", registration)
			}
			return nil
		}
		select {
		case <-child.done:
			return fmt.Errorf("distiller exited before lease: %v", child.wait())
		case <-ctx.Done():
			return fmt.Errorf("distiller lease not observed: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (child *distillerChild) openWriter(path string) (*os.File, error) {
	ctx, cancel := context.WithTimeout(context.Background(), distillerObservationBudget)
	defer cancel()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-child.done:
			return nil, fmt.Errorf("distiller exited before writer admission: %v", child.wait())
		default:
		}
		fd, err := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
		if err == nil {
			return os.NewFile(uintptr(fd), path), nil
		}
		if !errors.Is(err, unix.ENXIO) {
			return nil, fmt.Errorf("open FIFO writer: %w", err)
		}
		select {
		case <-child.done:
			return nil, fmt.Errorf("distiller exited before writer admission: %v", child.wait())
		case <-ctx.Done():
			return nil, fmt.Errorf("FIFO reader not admitted: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func assertDistillerJoined(t *testing.T, child *distillerChild) {
	t.Helper()
	select {
	case <-child.done:
	default:
		t.Fatal("child Wait still running")
	}
	if child.cmd.ProcessState == nil || !errors.Is(child.cmd.Process.Signal(syscall.Signal(0)), os.ErrProcessDone) {
		t.Fatal("child process not reaped")
	}
}
