package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestNativeFixtureProcessHelper(t *testing.T) {
	if os.Getenv("PAIR_AGENTFIXTURE_NATIVE_HELPER") != "1" {
		t.Skip("owned subprocess helper")
	}
	os.Exit(run())
}

// A real slave PTY has different cancellation behavior from io.Pipe: closing a
// blocking os.Stdin in another goroutine does not interrupt its kernel read.
// Keep the master open, send no input, and require a Stop to join and exit.
func TestNativeFixtureStopJoinsIdlePTYReader(t *testing.T) {
	for _, resize := range []bool{false, true} {
		t.Run(map[bool]string{false: "idle", true: "after-resize"}[resize], func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			for _, leaf := range []string{"home", "codex", "repo", "control", "bin"} {
				if err := os.Mkdir(filepath.Join(root, leaf), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(root, agentfixture.OwnershipFile), []byte("native-pty-run"), 0600); err != nil {
				t.Fatal(err)
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			// The executable is a physical owned file named claude, as in native fleets.
			source, err := os.Open(executable)
			if err != nil {
				t.Fatal(err)
			}
			candidate := filepath.Join(root, "bin", "claude")
			destination, err := os.OpenFile(candidate, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
			if err != nil {
				source.Close()
				t.Fatal(err)
			}
			_, copyErr := io.Copy(destination, source)
			source.Close()
			closeErr := destination.Close()
			if copyErr != nil || closeErr != nil {
				t.Fatal(copyErr, closeErr)
			}
			worktree := filepath.Join(root, "repo")
			base := filepath.Join(root, "control")
			control := agentfixture.ActorDirectory(base, worktree)
			cmd := exec.Command(candidate, "-test.run=^TestNativeFixtureProcessHelper$")
			cmd.Dir = worktree
			cmd.Env = testpair.Environment(t, root,
				"PAIR_AGENTFIXTURE_NATIVE_HELPER=1", "COUCH_ISOLATED_ROOT="+root,
				"PAIR_TESTCOUCH_ROOT="+root, "PAIR_TESTCOUCH_CONTROL="+base,
				"PAIR_TESTCOUCH_RUN_ID=native-pty-run", "PAIR_TESTCOUCH_COUCH=/usr/bin/true",
				"HOME="+filepath.Join(root, "home"), "CODEX_HOME="+filepath.Join(root, "codex"),
				"COUCH_TREE="+worktree, "TERM=xterm-256color")
			master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 38, Cols: 120})
			if err != nil {
				t.Fatal(err)
			}
			var output nativeOutput
			drained := make(chan struct{})
			go func() { defer close(drained); _, _ = io.Copy(&output, master) }()
			exited := make(chan error, 1)
			go func() { exited <- cmd.Wait() }()
			joined := false
			defer func() {
				if !joined {
					_ = cmd.Process.Kill()
					<-exited
				}
				_ = master.Close()
				<-drained
			}()
			var ready agentfixture.Ready
			nativeWait(t, func() bool {
				raw, e := os.ReadFile(filepath.Join(control, "ready.json"))
				return e == nil && json.Unmarshal(raw, &ready) == nil
			}, &output)
			if ready.PID != cmd.Process.Pid || ready.Generation == "" {
				t.Fatal("missing exact ready generation", ready)
			}
			// A completed control turn proves Run entered its event loop after
			// starting the native reader; keep the PTY completely input-idle.
			if err := agentfixture.WriteCommand(control, agentfixture.Command{ID: "reader-started", Kind: agentfixture.SetState, State: agentfixture.Idle}); err != nil {
				t.Fatal(err)
			}
			nativeWait(t, func() bool { _, e := agentfixture.ReadResult(control, "reader-started"); return e == nil }, &output)
			if resize {
				if err := pty.Setsize(master, &pty.Winsize{Rows: 30, Cols: 90}); err != nil {
					t.Fatal(err)
				}
				if err := cmd.Process.Signal(syscall.SIGWINCH); err != nil {
					t.Fatal(err)
				}
				nativeWait(t, func() bool {
					return strings.Contains(output.String(), agentfixture.Paint(agentfixture.AgentState{Agent: "claude", Phase: agentfixture.Idle}, 90, 30))
				}, &output)
			}
			if err := agentfixture.WriteCommand(control, agentfixture.Command{ID: "stop-idle-reader", Kind: agentfixture.Stop}); err != nil {
				t.Fatal(err)
			}
			nativeWait(t, func() bool { _, e := agentfixture.ReadResult(control, "stop-idle-reader"); return e == nil }, &output)
			select {
			case err := <-exited:
				joined = true
				if err != nil {
					t.Fatalf("fixture exit: %v; output=%q", err, output.String())
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("Stop handled but idle native PTY reader prevented process exit; output=%q", output.String())
			}
		})
	}
}

type nativeOutput struct {
	mu   sync.Mutex
	data []byte
}

func (o *nativeOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := len(p)
	space := 16384 - len(o.data)
	if len(p) > space {
		p = p[:space]
	}
	o.data = append(o.data, p...)
	return n, nil
}
func (o *nativeOutput) String() string { o.mu.Lock(); defer o.mu.Unlock(); return string(o.data) }
func nativeWait(t *testing.T, check func() bool, output *nativeOutput) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for !check() {
		if time.Now().After(deadline) {
			t.Fatalf("native fixture observation deadline; output=%q", output.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}
