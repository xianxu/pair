package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

func TestInternalZellijCommandSubprocess(t *testing.T) {
	if os.Getenv("PAIR_TEST_NATIVE_HELPER") != "1" {
		return
	}
	os.Exit(runWithLegacyRuntime([]string{"internal", "zellij-command", "--", "action", "rename-pane", "test"}, os.Stdout, os.Stderr, nil))
}

func TestInternalZellijCommandQualifiesEveryNativeRead(t *testing.T) {
	root := testpair.ShortRoot(t)
	t.Setenv("HOME", root)
	t.Setenv("TMPDIR", root)
	t.Setenv("ZELLIJ_SOCKET_DIR", root)
	t.Setenv("PATH", root)
	called := filepath.Join(root, "called")
	t.Setenv("PAIR_TEST_NATIVE_CALLED", called)
	if err := os.Mkdir(filepath.Join(root, "contract_version_1"), 0700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", filepath.Join(root, "contract_version_1", "unmarked"))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	executable := filepath.Join(root, "zellij")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf native >> \"$PAIR_TEST_NATIVE_CALLED\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	// Independent enumeration of read-only CliAction variants in supported
	// Zellij0.45.1, plus the top-level session listing. Exercise the real early
	// CLI route, not only its classifier, including named and absolute adapters.
	reads := [][]string{
		{"list-sessions", "--no-formatting"},
		{"action", "list-clients"},
		{"action", "list-panes", "--json"},
		{"action", "list-tabs", "--json"},
		{"action", "current-tab-info", "--json"},
		{"action", "are-floating-panes-visible", "--tab-id", "1"},
		{"action", "dump-screen", "--pane-id", "terminal_1"},
		{"action", "dump-layout"},
		{"action", "query-tab-names"},
	}
	for _, read := range reads {
		for _, prefix := range [][]string{nil, {"--session", "unmarked"}, {"-s", "unmarked"}, {executable, "--session", "unmarked"}} {
			t.Run(strings.Join(append(append([]string(nil), prefix...), read...), " "), func(t *testing.T) {
				args := append([]string{"internal", "zellij-command", "--"}, prefix...)
				args = append(args, read...)
				var stdout, stderr bytes.Buffer
				if code := runWithLegacyRuntime(args, &stdout, &stderr, nil); code != 1 || !strings.Contains(stderr.String(), zellijquery.ErrReadinessUnknown.Error()) {
					t.Errorf("unqualified read reached native: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
				}
			})
		}
	}
	if _, err := os.Stat(called); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unqualified native read effects: %v", err)
	}
	// Explicit controls still retain their supported legacy-endpoint behavior.
	for _, mutation := range [][]string{{"action", "rename-pane", "title"}, {"kill-session", "unmarked"}} {
		var stdout, stderr bytes.Buffer
		if code := runWithLegacyRuntime(append([]string{"internal", "zellij-command", "--"}, mutation...), &stdout, &stderr, nil); code != 0 {
			t.Fatalf("explicit control refused: %q code=%d stderr=%q", mutation, code, stderr.String())
		}
	}
	identity := zellijquery.ReadyIdentity{Namespace: root, Scope: "scope", Tag: "work", Agent: "claude", Session: "unmarked", Nonce: "fixture-launch"}
	creator, err := zellijquery.AcquireCreator(t.Context(), nil, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	if err := creator.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := creator.PublishReady(t.Context(), identity); err != nil {
		t.Fatal(err)
	}
	for _, read := range reads {
		var stdout, stderr bytes.Buffer
		if code := runWithLegacyRuntime(append([]string{"internal", "zellij-command", "--"}, read...), &stdout, &stderr, nil); code != 0 {
			t.Fatalf("qualified native read refused: %q code=%d stderr=%q", read, code, stderr.String())
		}
	}
	raw, err := os.ReadFile(called)
	if err != nil || string(raw) != strings.Repeat("native", 2+len(reads)) {
		t.Fatalf("native invocation census: %q %v", raw, err)
	}
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
