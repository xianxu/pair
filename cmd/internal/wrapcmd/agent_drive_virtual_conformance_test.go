package wrapcmd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"golang.org/x/term"
)

// The relay records bytes actually accepted by a native Codex PTY. It adds no
// input translation; its outer PTY is raw and output is copied unchanged.
func TestAgentDriveNativeRelayHelper(t *testing.T) {
	if os.Getenv("PAIR_DRIVE_BOUNDARY_ROLE") != "relay" {
		t.Skip("subprocess helper")
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	native := os.Getenv("PAIR_DRIVE_BOUNDARY_NATIVE")
	if len(args) == 1 && args[0] == "--version" {
		c := exec.Command(native, args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if c.Run() != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	old, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), old)
	size, err := pty.GetsizeFull(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	for until := time.Now().Add(2 * time.Second); (size.Rows == 0 || size.Cols == 0) && time.Now().Before(until); {
		time.Sleep(10 * time.Millisecond)
		size, err = pty.GetsizeFull(os.Stdin)
		if err != nil {
			t.Fatal(err)
		}
	}
	if size.Rows == 0 || size.Cols == 0 {
		t.Fatal("outer wrapper did not transfer terminal size")
	}
	c := exec.Command(native, args...)
	c.Env = os.Environ()
	master, err := pty.StartWithSize(c, size)
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	root := os.Getenv("PAIR_DRIVE_BOUNDARY_ROOT")
	if err := os.WriteFile(filepath.Join(root, "native.pid"), []byte(strconv.Itoa(c.Process.Pid)), 0600); err != nil {
		t.Fatal(err)
	}
	record, err := os.OpenFile(filepath.Join(root, "accepted.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer record.Close()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, e := os.Stdin.Read(buf)
			if n > 0 {
				nativeInput := buf[:n]
				accepted, we := master.Write(nativeInput)
				json.NewEncoder(record).Encode(struct {
					Hex   string
					N     int
					Error string
				}{Hex: hex.EncodeToString(nativeInput[:accepted]), N: accepted, Error: fmt.Sprint(we)})
				if we != nil {
					return
				}
			}
			if e != nil {
				return
			}
		}
	}()
	raw, err := os.OpenFile(filepath.Join(root, "native.raw"), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	go io.Copy(io.MultiWriter(os.Stdout, raw), master)
	err = c.Wait()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestAgentDriveRunBoundaryHelper(t *testing.T) {
	if os.Getenv("PAIR_DRIVE_BOUNDARY_ROLE") != "wrapper" {
		t.Skip("subprocess helper")
	}
	os.Exit(Run([]string{filepath.Join(os.Getenv("PAIR_DRIVE_BOUNDARY_ROOT"), "bin", "codex"), "--no-daemon", "--no-alt-screen", "--sandbox", "read-only", "--ask-for-approval", "never", "-c", "check_for_update_on_startup=false"}, os.Stdin, os.Stdout, os.Stderr))
}

type boundaryScreen struct {
	mu  sync.Mutex
	raw []byte
}

func (s *boundaryScreen) add(b []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.raw)+len(b) < harnessTTYRetentionLimit {
		s.raw = append(s.raw, b...)
	}
}
func (s *boundaryScreen) bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.raw...)
}

// This passes through Run, its actual input writer, the real wrapper socket,
// and Pair's virtual terminal query responses, unlike the direct-proxy harness.
func TestAgentDriveVirtualTerminalLiveConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_AGENT_DRIVE_VIRTUAL") != "codex" {
		t.Skip("set PAIR_LIVE_AGENT_DRIVE_VIRTUAL=codex")
	}
	for _, mode := range []string{"scratch", "pair-virtual"} {
		virtual := mode != "scratch"
		t.Run(mode, func(t *testing.T) {
			runAgentDriveBoundary(t, virtual)
		})
	}
}

func runAgentDriveBoundary(t *testing.T, virtual bool) {
	root, err := os.MkdirTemp("/tmp", "pair-drive-boundary-")
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	for _, name := range []string{"home", "codex", "claude", "config", "cache", "data", "state", "repo", "bin", "namespace"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	cwd := filepath.Join(root, "repo")
	if err := exec.Command("git", "init", "-q", "-b", "main", cwd).Run(); err != nil {
		t.Fatal(err)
	}
	env := peerLiveEnvironment(os.Environ(), root)
	env = peerLiveLocalAuth(t, "codex", root, cwd, env)
	native, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nPAIR_DRIVE_BOUNDARY_ROLE=relay exec '" + strings.ReplaceAll(exe, "'", "'\\''") + "' -test.run='^TestAgentDriveNativeRelayHelper$' -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(root, "bin", "codex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	namespace := filepath.Join(root, "namespace")
	env = append(env, "PAIR_DRIVE_BOUNDARY_ROLE=wrapper", "PAIR_DRIVE_BOUNDARY_NATIVE="+native, "PAIR_DRIVE_BOUNDARY_ROOT="+root, "COUCH_STORE_DIR="+namespace, "COUCH_THREAD_SCOPE=boundary", "COUCH_THREAD_TAG=boundary", "PAIR_SESSION_NAME=isolated-boundary", "PAIR_LAUNCH_NONCE=boundary-launch")
	if virtual {
		source := filepath.Join(root, "pair.terminfo")
		if err := os.WriteFile(source, []byte(terminal.DefaultProfile().TerminfoSource()), 0600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("tic", "-x", "-o", filepath.Join(root, "terminfo"), source).CombinedOutput(); err != nil {
			t.Fatalf("compile disposable terminfo: %v %s", err, out)
		}
		env = setEnv(env, "TERM", terminal.PairTERM)
		env = append(env, "TERM_PROGRAM=ghostty", "TERMINFO="+filepath.Join(root, "terminfo"))
	}
	life, cancel := context.WithCancel(context.Background())
	defer cancel()
	hello := make(chan couchmessage.Binding, 1)
	registryPath, err := couchmessage.SocketPath(namespace, "registry")
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartSessionServer(life, registryPath, couchmessage.SessionHandler{Open: func(_ couchmessage.SessionToken, h couchmessage.SessionHello) error {
		select {
		case hello <- h.Binding:
		default:
		}
		return nil
	}, Frame: func(couchmessage.SessionToken, couchmessage.SessionFrame) {}, Closed: func(couchmessage.SessionToken) {}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	wrapper := exec.Command(exe, "-test.run=^TestAgentDriveRunBoundaryHelper$")
	wrapper.Env = env
	wrapper.Dir = cwd
	master, err := pty.StartWithSize(wrapper, &pty.Winsize{Rows: 38, Cols: 120})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- wrapper.Wait() }()
	t.Cleanup(func() {
		if b, e := os.ReadFile(filepath.Join(root, "native.pid")); e == nil {
			if pid, e := strconv.Atoi(string(b)); e == nil && pid > 0 {
				syscall.Kill(-pid, syscall.SIGKILL)
			}
		}
		master.Close()
		select {
		case <-done:
			return
		case <-time.After(time.Second):
		}
		wrapper.Process.Kill()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("wrapper child not reaped")
		}
	})
	var host *terminal.Endpoint
	transport, e := ttyio.NewFile(master, master, false)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { transport.Close() })
	if virtual {
		host, e = terminal.NewEndpoint("isolated", terminal.Geometry{Rows: 38, Cols: 120}, transport)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { host.Close() })
	}
	screen := &boundaryScreen{}
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		buf := make([]byte, 32768)
		var tail []byte
		for {
			n, e := transport.Read(buf)
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				screen.add(chunk)
				if host != nil {
					if _, e := host.Feed(chunk, time.Now()); e != nil {
						return
					}
				} else {
					window := append(tail, chunk...)
					for range bytes.Count(window, []byte("\x1b[6n")) {
						transport.Write([]byte("\x1b[1;1R"))
					}
					keep := len(window)
					if keep > 3 {
						keep = 3
					}
					tail = append(tail[:0], window[len(window)-keep:]...)
				}
			}
			if e != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		transport.Close()
		select {
		case <-readerDone:
		case <-time.After(time.Second):
			t.Error("boundary reader not joined")
		}
	})
	var binding couchmessage.Binding
	select {
	case binding = <-hello:
	case <-time.After(25 * time.Second):
		t.Fatalf("wrapper not registered; output=%q", strings.ReplaceAll(string(screen.bytes()[max(0, len(screen.bytes())-4000):]), root, "/scratch"))
	}
	endpoint := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: binding}
	ctx, stop := context.WithTimeout(context.Background(), 25*time.Second)
	defer stop()
	var observed couchmessage.DriveObservation
	for ctx.Err() == nil {
		observed, err = endpoint.ObserveDrive(ctx)
		if err == nil && observed.Idle {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || !observed.Idle {
		t.Fatalf("no settled empty composer: %v observation=%+v output=%q", err, observed, strings.ReplaceAll(string(screen.bytes()[max(0, len(screen.bytes())-4000):]), root, "/scratch"))
	}
	request := agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "gpt-6.1-sol", Effort: "low"}}
	result, err := endpoint.StartDrive(ctx, "boundary-model", couchmessage.DriveStart{Request: request, Instance: observed.Instance, Sequence: observed.Sequence, Deadline: time.Now().Add(12 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	for !result.Terminal() && ctx.Err() == nil {
		time.Sleep(20 * time.Millisecond)
		result, err = endpoint.DriveStatus(ctx, "boundary-model")
		if err != nil {
			t.Fatal(err)
		}
	}
	accepted, _ := os.ReadFile(filepath.Join(root, "accepted.jsonl"))
	t.Logf("virtual=%t negotiated=%v accepted=%s", virtual, bytes.Contains(screen.bytes(), []byte("\x1b[>5u")), accepted)
	nativeRaw, _ := os.ReadFile(filepath.Join(root, "native.raw"))
	if out := os.Getenv("PAIR_DRIVE_BOUNDARY_CAPTURE_OUT"); out != "" {
		sanitized := strings.ReplaceAll(string(nativeRaw), root, "/scratch")
		if e := os.WriteFile(out+".raw", []byte(sanitized), 0600); e != nil {
			t.Error(e)
		}
		if e := os.WriteFile(out+".screen", []byte(strings.ReplaceAll(strings.Join(result.Tail, "\n"), root, "/scratch")+"\n"), 0600); e != nil {
			t.Error(e)
		}
	}
	t.Logf("native reverse7=%t reverse27=%t model-heading=%t", bytes.Contains(nativeRaw, []byte("\x1b[7m")), bytes.Contains(nativeRaw, []byte("\x1b[27m")), bytes.Contains(nativeRaw, []byte("Select Model and Effort")))
	t.Logf("result=%s", strings.ReplaceAll(fmt.Sprintf("%+v", result), root, "/scratch"))
	if result.Outcome != agentdrive.OutcomeConfirmed {
		t.Fatalf("actual wrapper/virtual terminal did not confirm selection: %s", result.Outcome)
	}
	boundaryDirtyRecovery(t, endpoint, transport)
}

// A single-line slash draft is deliberately inserted through Run's stdin and
// then cleared through the structured recovery operation. No conversation or
// model command is submitted by this check.
func boundaryDirtyRecovery(t *testing.T, endpoint couchmessage.RemoteEndpoint, input io.Writer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, err := input.Write([]byte("/model")); err != nil {
		t.Fatal(err)
	}
	var observation couchmessage.DriveObservation
	var err error
	for ctx.Err() == nil {
		observation, err = endpoint.ObserveDrive(ctx)
		if err == nil && observation.Recoverable && !observation.Idle {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || !observation.Recoverable || observation.Idle {
		t.Fatalf("dirty recovery observation %+v: %v", observation, err)
	}
	request := agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}}
	result, err := endpoint.StartDrive(ctx, "boundary-recovery", couchmessage.DriveStart{Request: request, Instance: observation.Instance, Sequence: observation.Sequence, Deadline: time.Now().Add(5 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	for !result.Terminal() && ctx.Err() == nil {
		time.Sleep(20 * time.Millisecond)
		result, err = endpoint.DriveStatus(ctx, "boundary-recovery")
		if err != nil {
			t.Fatal(err)
		}
	}
	if result.Outcome != agentdrive.OutcomeConfirmed {
		t.Fatalf("explicit dirty recovery: %s (%s)", result.Outcome, result.Detail)
	}
	for ctx.Err() == nil {
		observation, err = endpoint.ObserveDrive(ctx)
		if err == nil && observation.Idle {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || !observation.Idle {
		t.Fatalf("recovery did not settle empty: %+v %v", observation, err)
	}
}
