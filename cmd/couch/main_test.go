package main

import (
	"bytes"
	"context"
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

	// Use an isolated repository: opening the console must not enroll or launch it.
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

	t.Run("pty observes current directory without launching", func(t *testing.T) {
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
		consoleReady := make(chan struct{}, 1)
		go func() {
			var out bytes.Buffer
			buf := make([]byte, 4096)
			for {
				n, readErr := master.Read(buf)
				if n > 0 {
					_, _ = out.Write(buf[:n])
					if bytes.Contains(out.Bytes(), []byte("threads")) {
						select {
						case consoleReady <- struct{}{}:
						default:
						}
					}
				}
				if readErr != nil {
					readResult <- out.Bytes()
					return
				}
			}
		}()

		timedOut := false
		select {
		case <-consoleReady:
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
			t.Fatalf("timed out waiting for installed couch thread switcher; output=%q calls=%q", out, calls)
		}
		calls, readErr := os.ReadFile(callLog)
		if readErr != nil && !os.IsNotExist(readErr) {
			t.Fatal(readErr)
		}
		if len(calls) != 0 {
			t.Fatalf("observation-only startup invoked launcher contracts: %q", calls)
		}
		if bytes.Contains(out, []byte("COUCH_INSTALLED_PAIR_MARKER")) {
			t.Fatal("startup launched Pair")
		}
		if _, err := os.Stat(filepath.Join(launchRoot, ".couch", "thread.json")); !os.IsNotExist(err) {
			t.Fatalf("startup created slot record: %v", err)
		}

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
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(key, "PAIR_") && !strings.HasPrefix(key, "COUCH_") && !strings.HasPrefix(key, "XDG_") {
			env = append(env, entry)
		}
	}
	return append(env,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"COUCH_ISOLATED_ROOT="+root,
		"HOME="+filepath.Join(root, "home"),
		"XDG_DATA_HOME="+filepath.Join(root, "data"),
		"PAIR_DATA_DIR="+filepath.Join(root, "data", "pair"),
		"COUCH_STORE_DIR="+filepath.Join(root, "store"),
		"COUCH_IDENTITY_DIR="+filepath.Join(root, "identity"),
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
