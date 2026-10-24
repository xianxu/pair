package couchcore

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestProvisionProcessExitAndOutput(t *testing.T) {
	t.Parallel()

	var progress bytes.Buffer
	out, err := (OSProvisionIO{}).Run(context.Background(), ProvisionCommand{Program: "sh", Args: []string{"-c", "printf data; printf problem >&2; exit 7"}, Progress: &progress})
	var code interface{ ExitCode() int }
	if string(out) != "data" || !errors.As(err, &code) || code.ExitCode() != 7 || !strings.Contains(err.Error(), "problem") {
		t.Fatalf("out=%q error=%v", out, err)
	}
	if progress.String() != "problem" {
		t.Fatalf("progress=%q", progress.String())
	}
}

func TestProvisionProcessOutputLimits(t *testing.T) {
	t.Parallel()

	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "structured", true: "stream"}[stream], func(t *testing.T) {
			out, err := (OSProvisionIO{}).Run(context.Background(), ProvisionCommand{Program: "sh", Args: []string{"-c", "head -c 1200000 /dev/zero; printf END"}, StreamOutput: stream, Timeout: 5 * time.Second})
			if stream {
				if err != nil || len(out) > 64<<10 || !bytes.HasSuffix(out, []byte("END")) {
					t.Fatalf("len=%d err=%v", len(out), err)
				}
			} else if err == nil || len(out) > 1<<20 {
				t.Fatalf("len=%d err=%v", len(out), err)
			}
		})
	}
}

func TestProvisionProcessCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	progress := &provisionReadyWriter{ready: make(chan struct{})}
	armed := make(chan func(), 1)
	const requested = 45 * time.Second
	runner := OSProvisionIO{deadline: func(parent context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
		if budget != requested {
			t.Errorf("request budget = %v, want %v", budget, requested)
		}
		child, expire, stop := testwait.DeadlineOnCue(parent, budget)
		armed <- expire
		return child, stop
	}}
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, ProvisionCommand{Program: "sh", Args: []string{"-c", "trap '' TERM; printf ready >&2; sleep 3600 & wait"}, Progress: progress, Timeout: requested})
		done <- err
		close(done)
	}()
	t.Cleanup(func() { cancel(); <-done })
	var expire func()
	select {
	case expire = <-armed:
	case <-testwait.After(t):
		t.Fatal("request deadline was not created")
	}
	// Readiness follows the TERM trap, so expiry cannot bypass group escalation.
	select {
	case <-progress.ready:
	case <-testwait.After(t):
		t.Fatal("TERM-deaf child did not become ready")
	}
	expire()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error=%v", err)
		}
	case <-testwait.After(t):
		t.Fatal("expired command did not join")
	}
}

func TestProvisionProcessStreamSerializesProgress(t *testing.T) {
	t.Parallel()

	var progress bytes.Buffer
	out, err := (OSProvisionIO{}).Run(context.Background(), ProvisionCommand{Program: "sh", Args: []string{"-c", "(i=0; while [ $i -lt 500 ]; do printf a; i=$((i+1)); done) & i=0; while [ $i -lt 500 ]; do printf b >&2; i=$((i+1)); done; wait"}, Progress: &progress, StreamOutput: true})
	if err != nil || len(out) != 1000 || !bytes.Equal(out, progress.Bytes()) {
		t.Fatalf("len=%d progress=%d err=%v", len(out), progress.Len(), err)
	}
}

type provisionReadyWriter struct {
	once  sync.Once
	ready chan struct{}
}

func (w *provisionReadyWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.ready) })
	return len(p), nil
}

func TestProvisionProcessInheritedLeaseCancellation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	lease, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	progress := &provisionReadyWriter{ready: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := (OSProvisionIO{}).Run(ctx, ProvisionCommand{Program: "sh", Args: []string{"-c", "trap '' TERM; printf ready; sleep 30 & wait"}, Lease: lease.File(), Progress: progress, StreamOutput: true})
		done <- err
	}()
	select {
	case <-progress.ready:
	case <-testwait.After(t):
		t.Fatal("child failed to start")
	}
	lease.Close()
	if second, err := AcquireHostCreationLease(root); !errors.Is(err, ErrHostCreationBusy) {
		if second != nil {
			second.Close()
		}
		t.Fatalf("child missing lease: %v", err)
	}
	cancel()
	select {
	case err = <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	case <-testwait.After(t):
		t.Fatal("child failed to stop")
	}
	next, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	next.Close()
}

type provisionExitWriter struct {
	mu          sync.Mutex
	text        string
	ready, exit chan struct{}
	readyOnce   sync.Once
	exitOnce    sync.Once
}

func (w *provisionExitWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.text += string(p)
	if strings.Contains(w.text, "ready") {
		w.readyOnce.Do(func() { close(w.ready) })
	}
	if strings.Contains(w.text, "leader-exited") {
		w.exitOnce.Do(func() { close(w.exit) })
	}
	return len(p), nil
}

func TestProvisionProcessExitedLeaderRetainsDescendantLease(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	lease, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ctx, cancel := context.WithCancel(context.Background())
	progress := &provisionExitWriter{ready: make(chan struct{}), exit: make(chan struct{})}
	done := make(chan error, 1)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		_, err := (OSProvisionIO{}).Run(ctx, ProvisionCommand{
			Program: "sh", Args: []string{"-c", "trap 'printf leader-exited >&2; exit 0' TERM; (trap '' TERM; printf ready; sleep 30 & wait) & wait"},
			Lease: lease.File(), Progress: progress, StreamOutput: true,
		})
		done <- err
	}()
	t.Cleanup(func() { cancel(); <-stopped })
	select {
	case <-progress.ready:
	case <-testwait.After(t):
		t.Fatal("descendant did not install its TERM handler")
	}
	lease.Close()
	if second, err := AcquireHostCreationLease(root); !errors.Is(err, ErrHostCreationBusy) {
		if second != nil {
			second.Close()
		}
		t.Fatalf("descendant did not inherit lease: %v", err)
	}
	cancel()
	select {
	case <-progress.exit:
	case <-testwait.After(t):
		t.Fatal("leader did not process TERM")
	}
	select {
	case err = <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	case <-testwait.After(t):
		t.Fatal("descendant did not stop after grace")
	}
	next, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatalf("descendant retained lease after Run joined: %v", err)
	}
	next.Close()
}
