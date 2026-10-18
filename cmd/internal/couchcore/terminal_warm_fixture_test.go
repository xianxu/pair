package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestListedDetachedTerminalAttachesWithoutNewServer(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"zellij", "pair-wrap", "claude"} {
		if err = os.Symlink(exe, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	//lint:allow tmp-literal — the terminal fixture appends contract_version_1 and a session name containing this basename; macOS TMPDIR would exceed AF_UNIX limits
	sockets, err := os.MkdirTemp("/tmp", "pair-warm-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sockets) })
	files := t.TempDir()
	release := filepath.Join(files, "release")
	clientRelease := filepath.Join(files, "client-release")
	warmRelease := filepath.Join(files, "warm-release")
	request := filepath.Join(files, "server-request")
	attached := filepath.Join(files, "attached")
	scope, tag := "0123456789abcdef", "couch-0000000000000001"
	// Server discovery is host-wide by name, independent of socket directory.
	name := "pair-fixture-" + filepath.Base(sockets)
	paths, err := artifactpath.ResolveScoped(filepath.Join(root, "repos", scope), tag)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(root, "repos", scope), 0700); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"HOME":                             home,
		"PATH":                             bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"XDG_CACHE_HOME":                   "",
		"XDG_DATA_HOME":                    "",
		"COUCH_PAIR_DATA_DIR":              root,
		"PAIR_DATA_DIR":                    filepath.Join(root, "repos", scope),
		"PAIR_SCOPE_KEY":                   scope,
		"PAIR_TAG":                         tag,
		"PAIR_AGENT":                       "claude",
		"PAIR_SESSION_NAME":                name,
		"PAIR_LAUNCH_NONCE":                "fixture-warm-nonce",
		"PAIR_AGENT_READY_PATH":            paths.AgentReady("claude"),
		"PAIR_SCROLLBACK_RAW_PATH":         paths.ScrollbackRaw("claude"),
		"ZELLIJ_SOCKET_DIR":                sockets,
		"PAIR_TEST_COUCH_TERMINAL_FIXTURE": "1",
		"PAIR_TEST_ZELLIJ_READY":           filepath.Join(files, "ready"),
		"PAIR_TEST_ZELLIJ_RELEASE":         release,
		"PAIR_TEST_ZELLIJ_LOG":             filepath.Join(files, "calls"),
		"PAIR_TEST_ZELLIJ_PANES":           filepath.Join(files, "panes"),
		"PAIR_TEST_ZELLIJ_SERVER_REQUEST":  request,
		"PAIR_TEST_ZELLIJ_CLIENT_RELEASE":  clientRelease,
		"PAIR_TEST_ZELLIJ_ATTACH_READY":    attached,
		"PAIR_TEST_ZELLIJ_ATTACH_RELEASE":  warmRelease,
	}
	var explicit []string
	for key, value := range env {
		explicit = append(explicit, key+"="+value)
	}
	fixtureEnv := testpair.Environment(t, home, explicit...)
	testpair.ApplyEnvironment(t, home, explicit...)
	cfg := t.TempDir()
	if err = os.WriteFile(filepath.Join(cfg, "config.kdl"), []byte("session_serialization false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	type result struct {
		code int
		err  error
	}
	cold := make(chan result, 1)
	warm := make(chan result, 1)
	coldJoined, warmStarted, warmJoined := false, false, false
	var server *exec.Cmd
	t.Cleanup(func() {
		for _, path := range []string{clientRelease, warmRelease, release} {
			_ = os.WriteFile(path, []byte("release"), 0600)
		}
		cancel()
		if !coldJoined {
			select {
			case <-cold:
			case <-time.After(5 * time.Second):
				t.Error("cold client did not stop")
			}
		}
		if warmStarted && !warmJoined {
			select {
			case <-warm:
			case <-time.After(5 * time.Second):
				t.Error("warm client did not stop")
			}
		}
		if server != nil {
			_ = server.Wait()
		}
	})
	go func() {
		code, err := (launcher.OSRuntime{GlobalDataDir: root, DataDir: filepath.Join(root, "repos", scope)}).LaunchSession(ctx, name, cfg, "fixture-layout.kdl")
		cold <- result{code, err}
	}()
	waitFile := func(path string) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
			if _, err := os.Stat(path); err == nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		select {
		case got := <-cold:
			coldJoined = true
			t.Fatalf("fixture never published %s; cold=%v", filepath.Base(path), got.err)
		default:
		}
		t.Fatalf("fixture never published %s", filepath.Base(path))
	}
	waitFile(request)
	socket, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	server = exec.Command(filepath.Join(bin, "zellij"), "--server", string(socket))
	server.Env = fixtureEnv
	server.Stderr = os.Stderr
	if err = server.Start(); err != nil {
		t.Fatal(err)
	}
	waitFile(paths.AgentReady("claude"))
	waitFixtureHandoffLease(t, ctx, string(socket))
	before := procutil.Identity(strconv.Itoa(server.Process.Pid))
	if before == "" {
		t.Fatal("fixture server identity unavailable")
	}
	// Detach only the client, leaving its listed server alive.

	if err = os.WriteFile(clientRelease, []byte("detach"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-cold:
		coldJoined = true
		if got.code != 0 || got.err != nil {
			t.Fatalf("cold handoff code=%d err=%v", got.code, got.err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	sessions, err := (launcher.ZellijSource{}).SnapshotSessionsContext(ctx, []string{name})
	if err != nil || len(sessions) != 1 || sessions[0].State != launcher.SessionDetached {
		t.Fatalf("not detached: %+v %v", sessions, err)
	}
	readyBefore, err := os.ReadFile(paths.AgentReady("claude"))
	if err != nil {
		t.Fatal(err)
	}
	warmStarted = true
	go func() {
		code, err := (launcher.OSRuntime{GlobalDataDir: root, DataDir: filepath.Join(root, "repos", scope)}).AttachSession(name, cfg)
		warm <- result{code, err}
	}()
	waitFile(attached)
	cache, err := artifactpath.ResolvePairCache(home)
	if err != nil {
		t.Fatal(err)
	}
	leasePath, err := cache.TerminalLease(string(socket))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := launcher.AcquireTerminalLease(ctx, leasePath)
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
	if err = os.WriteFile(warmRelease, []byte("detach"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-warm:
		warmJoined = true
		if got.code != 0 || got.err != nil {
			t.Fatalf("warm handoff code=%d err=%v", got.code, got.err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	after := procutil.Identity(strconv.Itoa(server.Process.Pid))
	if after != before {
		t.Fatalf("warm replaced server: before=%s after=%s", before, after)
	}

	readyAfter, err := os.ReadFile(paths.AgentReady("claude"))
	if err != nil || string(readyAfter) != string(readyBefore) {
		t.Fatalf("warm attach changed agent readiness: %v", err)
	}
	calls, err := os.ReadFile(env["PAIR_TEST_ZELLIJ_LOG"])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(calls), "--new-session-with-layout") != 1 || strings.Count(string(calls), "--server ") != 1 || strings.Count(string(calls), " attach ") != 1 {
		t.Fatalf("warm created another terminal: %s", calls)
	}
}

func waitFixtureHandoffLease(t *testing.T, ctx context.Context, socket string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(filepath.Dir(socket))
	if err != nil {
		t.Fatal(err)
	}
	cache, err := artifactpath.ResolvePairCache(os.Getenv("HOME"))
	if err != nil {
		t.Fatal(err)
	}
	path, err := cache.TerminalLease(filepath.Join(dir, filepath.Base(socket)))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := launcher.AcquireTerminalLease(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
}
