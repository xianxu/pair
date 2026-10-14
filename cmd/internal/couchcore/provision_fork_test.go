//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

func pausedCreationFork(t *testing.T, lease *HostCreationLease) func() {
	t.Helper()
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("native fork conformance requires cc")
	}
	binary := filepath.Join(t.TempDir(), "fork")
	buildCtx, buildCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer buildCancel()
	output, err := exec.CommandContext(buildCtx, cc, "-Wall", "-Wextra", "-Werror", "-o", binary, "testdata/creation_fork.c").CombinedOutput()
	if err != nil {
		t.Fatalf("compile fork fixture: %v: %s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	cmd := exec.CommandContext(ctx, binary)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	cmd.ExtraFiles = []*os.File{lease.File()}
	input, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		cancel()
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		cancel()
		t.Fatal(err)
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			_, writeErr := input.Write([]byte("E"))
			input.Close()
			waitErr := cmd.Wait()
			cancel()
			if writeErr != nil || waitErr != nil {
				t.Errorf("fork fixture release: write=%v wait=%v", writeErr, waitErr)
			}
		})
	}
	t.Cleanup(release)
	var ready [1]byte
	if _, err = io.ReadFull(stdout, ready[:]); err != nil || ready[0] != 'R' {
		t.Fatalf("fork fixture readiness: %q, %v", ready, err)
	}
	return release
}

func TestProvisionLeaseWaitsForPausedForkExec(t *testing.T) {
	root := t.TempDir()
	lease, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	release := pausedCreationFork(t, lease)
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		next, err := AcquireHostCreationLease(root)
		if next != nil {
			next.Close()
		}
		result <- err
	}()
	// Keep the actual child paused beyond the old production budget. Pure
	// retry-policy tests separately pin the exact deadline without a scheduler.
	select {
	case err := <-result:
		t.Fatalf("acquisition finished while fork was paused: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	release()
	if err = <-result; err != nil {
		t.Fatalf("retake after child exec: %v", err)
	}
}
