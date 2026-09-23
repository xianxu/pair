package couchcore

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProvisionProcessExitAndOutput(t *testing.T) {
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
	start := time.Now()
	_, err := (OSProvisionIO{}).Run(context.Background(), ProvisionCommand{Program: "sh", Args: []string{"-c", "trap '' TERM; sleep 30 & wait"}, Timeout: 100 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancellation exceeded bounded stop")
	}
}

func TestProvisionProcessStreamSerializesProgress(t *testing.T) {
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
	case <-time.After(5 * time.Second):
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
	case <-time.After(5 * time.Second):
		t.Fatal("child failed to stop")
	}
	next, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	next.Close()
}
