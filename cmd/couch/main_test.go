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
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestBareCouchInstalledCommand(t *testing.T) {
	root := repoRoot(t)
	binDir := t.TempDir()
	couchBin, fixtureBin := installedExecutables(t)

	// Bare Couch launches its selected hub in an isolated repository.
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
	// Use the real Pair launcher with the existing stateful terminal fixture:
	// a server socket, live wrapper/agent processes and a readiness publication.
	// This exercises the public registration handshake before Couch attaches.
	pairBin := filepath.Join(binDir, "pair-go")
	testpair.At(t, pairBin)

	for _, name := range []string{"zellij", "pair-wrap", "claude"} {
		if err := os.Symlink(fixtureBin, filepath.Join(binDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	writeExecutable(t, filepath.Join(binDir, "pair"), fmt.Sprintf(`#!/bin/sh
printf 'pair %%s\n' "$*" >> %q
printf 'COUCH_INSTALLED_PAIR_MARKER\n'
exec %q launch "$@"
`, callLog, pairBin))
	fixtureFiles := t.TempDir()
	release := filepath.Join(fixtureFiles, "release")
	// Darwin Unix sockets must fit their pathname limit.
	//lint:allow tmp-literal — the terminal fixture binds contract_version_1 plus its session name below this root; macOS TMPDIR can exceed AF_UNIX limits
	sockets, err := os.MkdirTemp("/tmp", "couch-installed-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sockets) })

	t.Run("pty launches and attaches current hub", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cmd := exec.CommandContext(ctx, couchBin)
		cmd.Dir = launchRoot
		cmd.Env = append(installedEnv(t, binDir),
			"PAIR_HOME="+root,
			"ZELLIJ_SOCKET_DIR="+sockets,
			"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1",
			"PAIR_TEST_COUCH_LAUNCH_NATIVE_SIDECAR=1",
			"PAIR_TEST_ZELLIJ_READY="+filepath.Join(fixtureFiles, "ready"),
			"PAIR_TEST_ZELLIJ_RELEASE="+release,
			"PAIR_TEST_ZELLIJ_LOG="+filepath.Join(fixtureFiles, "zellij-calls"),
			"PAIR_TEST_ZELLIJ_PANES="+filepath.Join(fixtureFiles, "panes.json"),
		)
		defer func() { _ = os.WriteFile(release, []byte("release"), 0600) }()
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
					if bytes.Contains(out.Bytes(), []byte("COUCH_INSTALLED_PAIR_MARKER")) {
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
		startupDeadline := time.NewTimer(15 * time.Second)
		defer startupDeadline.Stop()
		select {
		case <-consoleReady:
		case <-startupDeadline.C:
			timedOut = true
		}
		if !timedOut {
			// Attachment can precede agent readiness; observe the fixture's
			// live wrapper publication before stopping the real launcher.
			ticker := time.NewTicker(10 * time.Millisecond)
			waiting := true
			for waiting {
				select {
				case <-ticker.C:
					_, err := os.Stat(filepath.Join(fixtureFiles, "ready"))
					waiting = err != nil
				case <-startupDeadline.C:
					timedOut, waiting = true, false
				}
			}
			ticker.Stop()
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
			zellijCalls, _ := os.ReadFile(filepath.Join(fixtureFiles, "zellij-calls"))
			t.Fatalf("timed out waiting for installed couch hub attachment and wrapper; output=%q calls=%q zellij=%q", out, calls, zellijCalls)
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
		var launches []string
		for _, line := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
			if strings.HasPrefix(line, "pair ") {
				launches = append(launches, line)
			}
		}
		if len(launches) != 1 || launches[0] != "pair --couch-session-v1 resume 1-alpha-1 --layout3" {
			t.Fatalf("hub launches = %q", launches)
		}
		if !bytes.Contains(out, []byte("COUCH_INSTALLED_PAIR_MARKER")) {
			t.Fatal("selected hub terminal was not attached")
		}
		if _, err := os.Stat(filepath.Join(fixtureFiles, "ready")); err != nil {
			t.Fatalf("hub wrapper readiness publication missing: %v", err)
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
	root := testpair.ShortRoot(t)
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	return testpair.Environment(t, home,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"COUCH_ISOLATED_ROOT="+root,
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
