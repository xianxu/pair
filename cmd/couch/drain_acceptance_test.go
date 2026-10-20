package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"strconv"
)

// TestDisposableCouchDrainFlags uses actual Pair launches, real pair-wrap and
// public CLI subprocesses run by admitted fixture agents. Two disposable peer
// checkouts are explicit secondary hubs; this covers other-hub flagging,
// rather than claiming coverage of non-hub worker ownership.
func TestDisposableCouchDrainFlags(t *testing.T) {
	if os.Getenv("PAIR_DRAIN_ACCEPTANCE") != "1" {
		t.Skip("opt in with PAIR_DRAIN_ACCEPTANCE=1")
	}
	couch, fixture := installedExecutables(t)
	bin := t.TempDir()
	pair := testpair.At(t, filepath.Join(bin, "pair-go"))
	for name, source := range map[string]string{"couch": couch, "zellij": fixture, "claude": fixture} {
		if err := os.Symlink(source, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	writeExecutable(t, filepath.Join(bin, "pair"), fmt.Sprintf("#!/bin/sh\nexec %q launch \"$@\"\n", pair))
	writeExecutable(t, filepath.Join(bin, "pair-wrap"), fmt.Sprintf("#!/bin/sh\nexec %q wrap \"$@\"\n", pair))
	fleet, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(fleet, "alpha")
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	runGit("init", "-b", "main", primary)
	runGit("-C", primary, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-m", "fixture")
	paths := []string{primary}
	for _, name := range []string{"beta", "gamma"} {
		path := filepath.Join(fleet, name)
		runGit("init", "-b", "main", path)
		runGit("-C", path, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-m", "fixture")
		paths = append(paths, path)
	}
	for _, path := range paths {
		headCmd := exec.Command("git", "-C", path, "rev-parse", "HEAD")
		headRaw, err := headCmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(path)
		identity := map[string]any{"schema_version": 2, "repo": name, "repo_identity": filepath.Join(path, ".git"), "primary_root": path, "fleet_root": fleet, "environment_root": fleet, "worktree_root": path, "kind": "primary", "address": name + ":0", "slot": 0, "branch": "main", "head": strings.TrimSpace(string(headRaw)), "resting_branch": "main"}
		raw, err := json.Marshal(identity)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, ".fixture-workspace.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Only workspace identity is stubbed, using metadata for real Git checkouts.
	// The test never substitutes registration credentials or accepted bindings.
	writeExecutable(t, filepath.Join(bin, "sdlc"), "#!/bin/sh\nif [ \"$#\" = 2 ] && [ \"$1\" = workspace ] && [ \"$2\" = --json ]; then exec cat .fixture-workspace.json; fi\nexit 90\n")
	envBase := installedEnv(t, bin)
	var isolated string
	for _, entry := range envBase {
		if strings.HasPrefix(entry, "COUCH_ISOLATED_ROOT=") {
			isolated = strings.TrimPrefix(entry, "COUCH_ISOLATED_ROOT=")
		}
	}
	controls := filepath.Join(isolated, "acceptance")
	if err := os.MkdirAll(controls, 0700); err != nil {
		t.Fatal(err)
	}
	// Preserve complete private fixture evidence after owned children join.
	t.Cleanup(func() {
		parent := os.Getenv("PAIR_DRAIN_ACCEPTANCE_ARTIFACT_DIR")
		if parent != "" && !filepath.IsAbs(parent) {
			t.Error("artifact directory must be absolute")
			return
		}
		archive, err := os.MkdirTemp(parent, "pair-drain-artifacts-")
		if err != nil {
			t.Errorf("archive directory: %v", err)
			return
		}
		if err := os.CopyFS(archive, os.DirFS(controls)); err != nil {
			t.Errorf("archive evidence: %v", err)
		}
		t.Logf("retained disposable acceptance artifacts: %s", archive)
	})
	release := filepath.Join(controls, "release")
	t.Cleanup(func() { _ = os.WriteFile(release, []byte("release"), 0600) })
	// Socket paths must fit Darwin's AF_UNIX bound even when TMPDIR is long.
	//lint:allow tmp-literal — private short socket root, removed by owned cleanup
	sockets, err := os.MkdirTemp("/tmp", "couch-drain-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sockets) })
	env := append(envBase, "PAIR_HOME="+repoRoot(t), "ZELLIJ_SOCKET_DIR="+sockets, "PAIR_TEST_COUCH_TERMINAL_FIXTURE=1", "PAIR_TEST_COUCH_LAUNCH_NATIVE_SIDECAR=1", "PAIR_TEST_COUCH_REAL_WRAPPER=1", "PAIR_TEST_COUCH_ACCEPTANCE_DIR="+controls, "PAIR_TEST_ZELLIJ_RELEASE="+release, "TERM=xterm-256color")
	var store string
	for _, entry := range envBase {
		if strings.HasPrefix(entry, "COUCH_STORE_DIR=") {
			store = strings.TrimPrefix(entry, "COUCH_STORE_DIR=")
		}
	}
	if err := os.MkdirAll(store, 0700); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(store, "drain.json")
	legacy := []byte("legacy opaque operator evidence: do not read or rewrite\n")
	if err := os.WriteFile(legacyPath, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	verifyDisk := func() {
		t.Helper()
		raw, err := os.ReadFile(legacyPath)
		if err != nil || !bytes.Equal(raw, legacy) {
			t.Fatalf("legacy drain bytes changed: %v %q", err, raw)
		}
		err = filepath.WalkDir(store, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.Contains(strings.ToLower(entry.Name()), "drain") && path != legacyPath {
				return fmt.Errorf("new drain persistence: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	started := time.Now()
	t.Logf("isolated installed acceptance: controls=%s socket_root=%s primary=%s; runtime budget=90s (build setup excluded); secondary peer checkouts beta:0/gamma:0 are explicit hubs", controls, sockets, primary)
	type console struct {
		cmd      *exec.Cmd
		terminal *os.File
		done     chan struct{}
		readDone chan struct{}
		err      error
		output   *drainAcceptanceOutput
	}
	start := func() *console {
		cmd := exec.CommandContext(ctx, couch, paths...)
		cmd.Dir = primary
		cmd.Env = env
		terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 30, Cols: 120})
		if err != nil {
			t.Fatal(err)
		}
		c := &console{cmd: cmd, terminal: terminal, done: make(chan struct{}), readDone: make(chan struct{}), output: &drainAcceptanceOutput{}}
		capture, err := os.Create(filepath.Join(controls, fmt.Sprintf("console-%d.raw", cmd.Process.Pid)))
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			defer close(c.readDone)
			defer capture.Close()
			_, _ = io.Copy(io.MultiWriter(c.output, capture), terminal)
		}()
		go func() { c.err = cmd.Wait(); close(c.done) }()
		t.Cleanup(func() {
			_ = os.WriteFile(release, []byte("release"), 0600)
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = terminal.Close()
			select {
			case <-c.done:
			case <-time.After(5 * time.Second):
				t.Error("Couch was not joined")
			}
			select {
			case <-c.readDone:
			case <-time.After(5 * time.Second):
				t.Error("Couch PTY reader was not joined")
			}
			if err := os.WriteFile(filepath.Join(controls, fmt.Sprintf("console-%d.raw", cmd.Process.Pid)), []byte(c.output.String()), 0600); err != nil {
				t.Errorf("console evidence: %v", err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for {
				entries, _ := os.ReadDir(controls)
				pending := false
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					for _, name := range []string{"ready.server", "owned-wrapper.json", "owned-agent.json"} {
						raw, err := os.ReadFile(filepath.Join(controls, entry.Name(), name))
						if os.IsNotExist(err) {
							continue
						}
						if err != nil {
							t.Error(err)
							continue
						}
						var child struct {
							PID      int
							Identity string
						}
						if err := json.Unmarshal(raw, &child); err != nil {
							t.Error(err)
							continue
						}
						pid := strconv.Itoa(child.PID)
						if child.PID > 0 && child.Identity != "" && procutil.Alive(pid) && procutil.Identity(pid) == child.Identity {
							pending = true
						}
					}
				}
				if !pending {
					break
				}
				if time.Now().After(deadline) {
					t.Error("terminal fixture servers were not joined")
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
		return c
	}
	c := start()
	wait := func(what string, check func() bool) {
		t.Helper()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			if check() {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatalf("%s: %v; console=%s", what, ctx.Err(), c.output.String())
			case <-c.done:
				t.Fatalf("%s: Couch exited: %v; console=%s", what, c.err, c.output.String())
			case <-ticker.C:
			}
		}
	}
	agents := map[string]string{}
	wait("three genuine agent environments", func() bool {
		entries, _ := os.ReadDir(controls)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(controls, entry.Name())
			raw, e := os.ReadFile(filepath.Join(dir, "environment.json"))
			if e != nil {
				continue
			}
			var values []string
			if json.Unmarshal(raw, &values) != nil {
				continue
			}
			raw, _ = os.ReadFile(filepath.Join(dir, "cwd"))
			for _, path := range paths {
				if string(raw) == path {
					agents[path] = dir
				}
			}
		}
		return len(agents) == 3
	})
	callNumber := 0
	lastCLIOutput := ""
	allowUnavailable := true
	callFrom := func(agentPath string, args ...string) couchmessage.Response {
		t.Helper()
		dir := agents[agentPath]
		resultPath := filepath.Join(dir, "result.json")
		_ = os.Remove(resultPath)
		raw, _ := json.Marshal(args)
		if err := os.WriteFile(filepath.Join(dir, "command.json.tmp"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(dir, "command.json.tmp"), filepath.Join(dir, "command.json")); err != nil {
			t.Fatal(err)
		}
		var result struct{ Output, Error string }
		wait("public CLI "+strings.Join(args, " "), func() bool { raw, e := os.ReadFile(resultPath); return e == nil && json.Unmarshal(raw, &result) == nil })
		lastCLIOutput = result.Output
		callNumber++
		t.Logf("public CLI %d %v: error=%q output=%s", callNumber, args, result.Error, result.Output)
		evidence, _ := json.Marshal(struct {
			Args          []string
			Output, Error string
		}{args, result.Output, result.Error})
		if err := os.WriteFile(filepath.Join(controls, fmt.Sprintf("cli-%03d.json", callNumber)), evidence, 0600); err != nil {
			t.Fatal(err)
		}
		if result.Error != "" && allowUnavailable {
			var response couchmessage.Response
			if json.Unmarshal([]byte(result.Output), &response) == nil && response.Code == "unavailable" {
				return response
			}
		}
		if result.Error != "" {
			t.Fatalf("public CLI %v: %s %s", args, result.Error, result.Output)
		}
		var response couchmessage.Response
		if strings.Contains(strings.Join(args, " "), "--json") {
			if err := json.Unmarshal([]byte(result.Output), &response); err != nil {
				t.Fatalf("CLI JSON %v: %v %s", args, err, result.Output)
			}
		}
		return response
	}
	call := func(args ...string) couchmessage.Response { return callFrom(primary, args...) }
	wait("registered actors", func() bool { return len(call("--actors", "--json").Actors) == 3 })
	allowUnavailable = false
	checkFlags := func(response couchmessage.Response, count int) {
		t.Helper()
		if response.Drain == nil || len(response.Drain.Rows) != count {
			t.Fatalf("flags = %+v, want %d", response.Drain, count)
		}
		for _, row := range response.Drain.Rows {
			if row.Workspace == "alpha:0" {
				t.Fatal("issuing steward was flagged")
			}
		}
	}
	submitted := func(response couchmessage.Response) {
		t.Helper()
		if len(response.DrainDeliveries) != 2 {
			t.Fatalf("notice deliveries=%d, want two", len(response.DrainDeliveries))
		}
		seen := map[string]bool{}
		for _, delivery := range response.DrainDeliveries {
			seen[delivery.Workspace] = true
		}
		if !seen["beta:0"] || !seen["gamma:0"] {
			t.Fatalf("unexpected notice recipients: %+v", response.DrainDeliveries)
		}
		for _, delivery := range response.DrainDeliveries {
			if delivery.Error != "" || delivery.Receipt == nil {
				t.Fatalf("notice admission %+v", delivery)
			}
			id := delivery.Receipt.Message.ID
			wait("submitted ordinary notice", func() bool {
				r := call("--message-status", id, "--json")
				if r.Receipt != nil && r.Receipt.Status.Terminal() && r.Receipt.Status != couchmessage.Submitted {
					t.Fatalf("notice failed: %+v", r.Receipt)
				}
				return r.Receipt != nil && r.Receipt.Status == couchmessage.Submitted
			})
		}
	}
	waitSent := func(body string) {
		t.Helper()
		fields := strings.Fields(lastCLIOutput)
		if len(fields) == 0 {
			t.Fatal("public send has no receipt ID")
		}
		id := fields[0]
		wait("ordinary send submitted", func() bool {
			r := call("--message-status", id, "--json")
			if r.Receipt == nil {
				return false
			}
			if r.Receipt.Status.Terminal() && r.Receipt.Status != couchmessage.Submitted {
				t.Fatalf("send failed: %+v", r.Receipt)
			}
			if r.Receipt.Status != couchmessage.Submitted {
				return false
			}
			if r.Receipt.Message.Body != body {
				t.Fatalf("receipt body=%q", r.Receipt.Message.Body)
			}
			return true
		})
	}
	response := call("--drain", "--json")
	checkFlags(response, 2)
	submitted(response)
	// Normal peer delivery remains usable while flags are set.
	call("--send-to", "beta:0", "--message", "ordinary-while-flagged")
	waitSent("ordinary-while-flagged")
	wait("ordinary message reaches real wrapper", func() bool {
		raw, _ := os.ReadFile(filepath.Join(agents[paths[1]], "received"))
		return bytes.Contains(raw, []byte("ordinary-while-flagged"))
	})
	callFrom(paths[1], "--send-to", "alpha:0", "--message", "acknowledge message received, will resume processing after drain is lifted")
	waitSent("acknowledge message received, will resume processing after drain is lifted")
	wait("worker reply reaches steward through real wrapper", func() bool {
		raw, _ := os.ReadFile(filepath.Join(agents[primary], "received"))
		return bytes.Contains(raw, []byte("acknowledge message received, will resume processing after drain is lifted"))
	})
	response = call("--undrain", "--json")
	checkFlags(response, 0)
	submitted(response)
	response = call("--drain", "--fast", "--json")
	checkFlags(response, 2)
	submitted(response)
	response = call("--undrain", "--message", "custom-continuation-marker", "--json")
	checkFlags(response, 0)
	submitted(response)
	response = call("--drain", "--json")
	checkFlags(response, 2)
	submitted(response)
	// Quit/park retains its normal confirmation while flagged. Cancel it;
	// completing park-all belongs to the separate park conformance test.
	if _, err := c.terminal.Write([]byte("\x00")); err != nil {
		t.Fatal(err)
	}
	wait("switcher before quit confirmation", func() bool { return strings.Contains(c.output.String(), "threads") })
	if _, err := c.terminal.Write([]byte("\x1bx")); err != nil {
		t.Fatal(err)
	}
	wait("normal park-all confirmation while flagged", func() bool { return strings.Contains(c.output.String(), "leave couch") })
	if _, err := c.terminal.Write([]byte("\x1b")); err != nil {
		t.Fatal(err)
	}
	checkFlags(call("--drain", "--status", "--json"), 2)
	// Public whole-Couch detach is Ctrl-Space then Alt+d. It must complete even
	// with flags present, independently of the informational drain posture.
	wait("switcher render", func() bool { return strings.Contains(c.output.String(), "threads") })
	if _, err := c.terminal.Write([]byte("\x1bd")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.done:
		if c.err != nil {
			t.Fatalf("public detach: %v; console=%s", c.err, c.output.String())
		}
	case <-ctx.Done():
		t.Fatalf("detach waited on drain flags: %v", ctx.Err())
	}
	_ = c.terminal.Close()
	c = start()
	allowUnavailable = true
	wait("restarted messaging reconnects", func() bool { return len(call("--actors", "--json").Actors) == 3 })
	allowUnavailable = false
	checkFlags(call("--drain", "--status", "--json"), 0)
	call("--send-to", "gamma:0", "--message", "ordinary-after-restart")
	waitSent("ordinary-after-restart")
	wait("post-restart real wrapper delivery", func() bool {
		raw, _ := os.ReadFile(filepath.Join(agents[paths[2]], "received"))
		return bytes.Contains(raw, []byte("ordinary-after-restart"))
	})
	verifyDisk()
	t.Logf("installed drain flags, generic/custom continuation, ordinary delivery, public detach and restart completed in %s", time.Since(started))
}

type drainAcceptanceOutput struct {
	mu sync.Mutex
	bytes.Buffer
}

func (o *drainAcceptanceOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Buffer.Write(p)
}
func (o *drainAcceptanceOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Buffer.String()
}
