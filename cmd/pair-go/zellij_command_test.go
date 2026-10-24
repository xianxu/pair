package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

func TestInternalZellijCommandSubprocess(t *testing.T) {
	if os.Getenv("PAIR_TEST_NATIVE_HELPER") != "1" {
		return
	}
	os.Exit(runWithLegacyRuntime([]string{"internal", "zellij-command", "--", "action", "rename-pane", "test"}, os.Stdout, os.Stderr, nil))
}

func TestInternalZellijCommandTerminationJoinsNativeBeforeUnlock(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("TMPDIR", root)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(root, "native"))
	t.Setenv("PATH", root)
	// Native child publishes its PID then blocks on inherited input. The test
	// independently proves that this exact child has exited before unlock.
	if err := os.WriteFile(filepath.Join(root, "zellij"), []byte("#!/bin/sh\nprintf 'native-started %s\\n' \"$$\"\nread waiting\n"), 0700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestInternalZellijCommandSubprocess$")
	cmd.Env = append(os.Environ(), "PAIR_TEST_NATIVE_HELPER=1")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "native-started ") {
		t.Fatalf("native did not enter: %q %v", line, err)
	}
	var pid int
	if _, err := fmt.Sscanf(line, "native-started %d", &pid); err != nil {
		t.Fatal(err)
	}
	defer syscall.Kill(pid, syscall.SIGKILL)
	blocked, stop := context.WithTimeout(ctx, 25*time.Millisecond)
	lease, err := zellijquery.AcquireExplicitCommand(blocked, os.Environ())
	stop()
	if lease != nil {
		lease.Close()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("helper failed exclusion: %v", err)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("termination reported success")
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("helper returned before native child exited: pid=%d err=%v", pid, err)
	}
	lease, err = zellijquery.AcquireExplicitCommand(ctx, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
}
