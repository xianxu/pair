package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestBareCouchInstalledCommand(t *testing.T) {
	root := repoRoot(t)
	binDir := t.TempDir()
	couchBin := filepath.Join(binDir, "couch")
	build := exec.Command("go", "build", "-o", couchBin, "./cmd/couch")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build couch: %v\n%s", err, out)
	}

	// Use an isolated repository: managed startup enrolls known slot locations.
	fleet, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	launchRoot := filepath.Join(fleet, "alpha")
	init := exec.Command("git", "init", "-b", "main", launchRoot)
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("fixture git init: %v %s", err, out)
	}
	identity, err := json.Marshal(map[string]any{"schema_version": 2, "repo": "alpha", "repo_identity": filepath.Join(launchRoot, ".git"), "primary_root": launchRoot, "fleet_root": fleet, "environment_root": fleet, "worktree_root": launchRoot, "kind": "primary", "address": "alpha:0", "slot": 0, "branch": "main", "head": nil, "resting_branch": "main"})
	if err != nil {
		t.Fatal(err)
	}
	callLog := filepath.Join(t.TempDir(), "calls")
	// Only the workspace identity contract is supported; policy-provider calls
	// remain forbidden. The JSON describes this fixture's real Git checkout.
	writeExecutable(t, filepath.Join(binDir, "sdlc"), fmt.Sprintf(`#!/bin/sh
printf 'sdlc %%s\n' "$*" >> %q
if [ "$#" = 2 ] && [ "$1" = workspace ] && [ "$2" = --json ]; then
cat <<'PAIR_FIXTURE_IDENTITY'
%s
PAIR_FIXTURE_IDENTITY
else
exit 90
fi
`, callLog, identity))
	writeExecutable(t, filepath.Join(binDir, "pair"), fmt.Sprintf(`#!/bin/sh
printf 'pair %%s\n' "$*" >> %q
printf 'COUCH_INSTALLED_PAIR_MARKER\n'
sleep 2
`, callLog))

	t.Run("pty launches current directory", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cmd := exec.CommandContext(ctx, couchBin)
		cmd.Dir = launchRoot
		cmd.Env = installedEnv(t, binDir)
		master, err := pty.Start(cmd)
		if err != nil {
			t.Fatal(err)
		}
		defer master.Close()
		if err := pty.Setsize(master, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
			t.Fatal(err)
		}

		waitResult := make(chan error, 1)
		go func() { waitResult <- cmd.Wait() }()
		readResult := make(chan []byte, 1)
		go func() {
			var out bytes.Buffer
			buf := make([]byte, 4096)
			for {
				n, readErr := master.Read(buf)
				if n > 0 {
					_, _ = out.Write(buf[:n])
				}
				if readErr != nil {
					readResult <- out.Bytes()
					return
				}
			}
		}()
		pairCalled := make(chan struct{}, 1)
		go func() {
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					calls, _ := os.ReadFile(callLog)
					if bytes.Contains(calls, []byte("pair resume ")) {
						pairCalled <- struct{}{}
						return
					}
				}
			}
		}()

		timedOut := false
		select {
		case <-pairCalled:
		case <-time.After(15 * time.Second):
			timedOut = true
		}
		cancel()
		_ = master.Close()
		waitInstalled(t, cmd, cancel, waitResult)
		var out []byte
		select {
		case out = <-readResult:
		case <-time.After(2 * time.Second):
			t.Fatal("installed couch PTY reader did not finish")
		}
		if timedOut {
			calls, _ := os.ReadFile(callLog)
			t.Fatalf("timed out waiting for installed couch marker; output=%q calls=%q", out, calls)
		}
		calls, err := os.ReadFile(callLog)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
			if strings.HasPrefix(line, "sdlc ") && line != "sdlc workspace --json" {
				t.Fatalf("unexpected SDLC contract: %q", line)
			}
		}
		if !bytes.Contains(calls, []byte("sdlc workspace --json")) {
			t.Fatal("workspace context was not resolved")
		}
		assertExactPairResumeCall(t, calls)
	})

	t.Run("pipe refuses before effects", func(t *testing.T) {
		before, _ := os.ReadFile(callLog)
		cmd := exec.Command(couchBin)
		cmd.Dir = launchRoot
		cmd.Env = installedEnv(t, binDir)
		if err := cmd.Run(); err == nil {
			t.Fatal("bare couch with pipes succeeded")
		}
		after, _ := os.ReadFile(callLog)
		if !bytes.Equal(before, after) {
			t.Fatalf("non-terminal launch performed effects: before=%q after=%q", before, after)
		}
	})
}

// The exact shape of the one pair call couch makes. The trailing --layout3 is
// couch's DEFAULT layout since #198, not a pinned constant: a `couch --layout3`
// makes the same call with --layout3. What stays exact is the arity and the
// generated tag -- a stray positional here would be the regression.
func assertExactPairResumeCall(t *testing.T, calls []byte) {
	t.Helper()
	var pairCalls [][]string
	for _, line := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "pair" {
			pairCalls = append(pairCalls, fields)
		}
	}
	if len(pairCalls) != 1 {
		t.Fatalf("pair calls = %q, want exactly one", pairCalls)
	}
	call := pairCalls[0]
	if len(call) != 4 {
		t.Fatalf("pair call = %q, want exactly pair resume <generated-couch-tag> <couch default layout>", call)
	}
	tagHex := strings.TrimPrefix(call[2], "couch-")
	_, tagErr := hex.DecodeString(tagHex)
	if call[1] != "resume" || !strings.HasPrefix(call[2], "couch-") || len(tagHex) != 16 || tagErr != nil || call[3] != "--layout3" {
		t.Fatalf("pair call = %q, want exactly pair resume <generated-couch-tag> <couch default layout>", call)
	}
}

func waitInstalled(t *testing.T, cmd *exec.Cmd, cancel context.CancelFunc, waitResult <-chan error) {
	t.Helper()
	select {
	case <-waitResult:
		return
	case <-time.After(2 * time.Second):
		cancel()
	}
	select {
	case <-waitResult:
		return
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
	}
	select {
	case <-waitResult:
	case <-time.After(2 * time.Second):
		t.Fatal("installed couch was not reaped")
	}
}

func installedEnv(t *testing.T, binDir string) []string {
	t.Helper()
	home := t.TempDir()
	return append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+home,
		"XDG_DATA_HOME="+filepath.Join(home, "data"),
		"COUCH_STORE_DIR="+filepath.Join(home, "store"),
	)
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
