package wrapcmd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"io"
	"os"
	"os/exec"
	ossignal "os/signal"
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
	"github.com/xianxu/pair/cmd/internal/procutil"
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
	shutdown := make(chan os.Signal, 1)
	ossignal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer ossignal.Stop(shutdown)
	c := exec.Command(native, args...)
	c.Env = os.Environ()
	master, err := pty.StartWithSize(c, size)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("PAIR_DRIVE_BOUNDARY_ROOT")
	owner := boundaryProcessOwner{PID: c.Process.Pid, Start: procutil.StrictIdentity(strconv.Itoa(c.Process.Pid))}
	nativeIO, err := ttyio.NewFile(master, master, false)
	if err != nil {
		c.Process.Kill()
		c.Wait()
		master.Close()
		t.Fatal(err)
	}
	inputIO, err := ttyio.NewFile(os.Stdin, os.Stdout, false)
	if err != nil {
		nativeIO.Close()
		c.Process.Kill()
		c.Wait()
		master.Close()
		t.Fatal(err)
	}
	var record, raw *os.File
	relayCtx, cancelRelay := context.WithCancel(context.Background())
	waited := make(chan struct{})
	var waitErr error
	go func() { waitErr = c.Wait(); close(waited) }()
	inputDone, outputDone := make(chan struct{}), make(chan struct{})
	// Setup failures have no pumps to join; replace these once pumps start.
	joinPumps := func(time.Duration) bool { return true }
	defer func() {
		cleanupErr := teardownHarnessTTY(harnessTTYTeardownOps{
			cancel:   func() { cancelRelay(); inputIO.Close() },
			closePTY: func() error { nativeIO.Close(); return master.Close() },
			signal: func() error {
				select {
				case <-waited:
					return os.ErrProcessDone
				default:
				}
				return signalBoundaryGroup(owner, procutil.StrictIdentity, boundaryProbe, syscall.Kill, syscall.SIGINT)
			},
			kill: func() error {
				return signalBoundaryGroup(owner, procutil.StrictIdentity, boundaryProbe, syscall.Kill, syscall.SIGKILL)
			},
			waitBounded: func(d time.Duration) (error, bool) {
				select {
				case <-waited:
					return waitErr, true
				case <-time.After(d):
					return nil, false
				}
			},
			joinReader: func(d time.Duration) bool { return joinPumps(d) }, grace: time.Second,
		})
		if cleanupErr != nil {
			t.Error(cleanupErr)
		}
		if record != nil {
			record.Close()
		}
		if raw != nil {
			raw.Close()
		}
		if cleanupErr == nil {
			if e := os.WriteFile(filepath.Join(root, "relay.cleaned"), []byte("native reaped; input and output pumps joined\n"), 0600); e != nil {
				t.Error(e)
			}
		}
	}()
	if owner.Start == "" {
		t.Fatal("native process identity unavailable; group signaling refused")
	}
	ownerBytes, _ := json.Marshal(owner)
	if err := os.WriteFile(filepath.Join(root, "native.owner"), ownerBytes, 0600); err != nil {
		t.Fatal(err)
	}

	record, err = os.OpenFile(filepath.Join(root, "accepted.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}

	raw, err = os.OpenFile(filepath.Join(root, "native.raw"), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}

	joinPumps = func(d time.Duration) bool {
		timer := time.NewTimer(d)
		defer timer.Stop()
		for _, done := range []<-chan struct{}{inputDone, outputDone} {
			select {
			case <-done:
			case <-timer.C:
				return false
			}
		}
		return true
	}
	go func() {
		defer close(inputDone)
		buf := make([]byte, 4096)
		for {
			n, e := inputIO.ReadContext(relayCtx, buf)
			if n > 0 {
				nativeInput := buf[:n]
				accepted, we := nativeIO.WriteContext(relayCtx, nativeInput)
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
	go func() { defer close(outputDone); io.Copy(io.MultiWriter(raw, inputIO), nativeIO) }()
	select {
	case <-waited:
	case <-shutdown:
		return
	}
	// Drain native output to EOF before closing its transport. A failed drain is
	// explicit rather than claiming the final output was retained.
	select {
	case <-outputDone:
	case <-time.After(time.Second):
		t.Error("native output did not drain before teardown")
	}

}

func TestAgentDriveRunBoundaryHelper(t *testing.T) {
	if os.Getenv("PAIR_DRIVE_BOUNDARY_ROLE") != "wrapper" {
		t.Skip("subprocess helper")
	}
	// TestMain removes inherited live Couch identity for ordinary tests (#597).
	// This isolated helper intentionally hosts its own registry and wrapper.
	t.Setenv("COUCH_THREAD_SCOPE", "boundary")
	t.Setenv("COUCH_THREAD_TAG", "boundary")
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
	env := peerLiveEnvironment(testpair.Environment(t, root), root)
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
	env = append(env, "PAIR_DRIVE_BOUNDARY_ROLE=wrapper", "PAIR_DRIVE_BOUNDARY_NATIVE="+native, "PAIR_DRIVE_BOUNDARY_ROOT="+root, "COUCH_STORE_DIR="+namespace, "COUCH_THREAD_SCOPE=boundary", "COUCH_THREAD_TAG=boundary", "PAIR_SESSION_NAME=isolated-boundary", "PAIR_LAUNCH_NONCE=boundary-launch", "PAIR_WRAP_LOG="+filepath.Join(root, "wrapper.log"))
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
	wrapper.Env = testpair.Environment(t, root, env...)
	wrapper.Dir = cwd
	master, err := pty.StartWithSize(wrapper, &pty.Winsize{Rows: 38, Cols: 120})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- wrapper.Wait() }()
	t.Cleanup(func() {
		if b, e := os.ReadFile(filepath.Join(root, "native.owner")); e == nil {
			var owner boundaryProcessOwner
			if e = json.Unmarshal(b, &owner); e != nil {
				t.Error(e)
			} else if e = signalBoundaryGroup(owner, procutil.StrictIdentity, boundaryProbe, syscall.Kill, syscall.SIGKILL); e != nil && !errors.Is(e, os.ErrProcessDone) {
				t.Error(e)
			}
		}

		if _, e := os.Stat(filepath.Join(root, "native.owner")); e == nil {
			joined := false
			for until := time.Now().Add(3 * time.Second); time.Now().Before(until); {
				if _, e := os.Stat(filepath.Join(root, "relay.cleaned")); e == nil {
					joined = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !joined {
				t.Error("relay did not acknowledge native reap and pump joins")
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
		raw := screen.bytes()
		diagnostic, _ := os.ReadFile(filepath.Join(root, "wrapper.log"))
		t.Logf("wrapper diagnostic=%s", strings.ReplaceAll(string(diagnostic), root, "/scratch"))
		t.Fatalf("wrapper not registered; output=%q", strings.ReplaceAll(string(raw[max(0, len(raw)-4000):]), root, "/scratch"))
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
		raw := screen.bytes()
		t.Fatalf("no settled empty composer: %v observation=%+v output=%q", err, observed, strings.ReplaceAll(string(raw[max(0, len(raw)-4000):]), root, "/scratch"))
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
	result, err := boundaryRecoveryDrive(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
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

// An identity is persisted with the PID because teardown can run after an
// asynchronous native exit. Never signal a recycled or unobservable group.
type boundaryProcessOwner struct {
	PID   int
	Start string
}

func signalBoundaryGroup(owner boundaryProcessOwner, identity func(string) string, probe func(int) error, signal func(int, syscall.Signal) error, sig syscall.Signal) error {
	if owner.PID <= 0 || owner.Start == "" {
		return fmt.Errorf("invalid native process ownership")
	}
	current := identity(strconv.Itoa(owner.PID))
	if current == "" {
		if errors.Is(probe(owner.PID), syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return fmt.Errorf("native process identity unavailable; group signal refused")
	}
	if current != owner.Start {
		return os.ErrProcessDone
	}
	return signal(-owner.PID, sig)
}
func boundaryProbe(pid int) error { return syscall.Kill(pid, 0) }

func TestBoundaryGroupSignalRequiresCurrentIdentity(t *testing.T) {
	for _, current := range []string{"", "reused", "owned"} {
		t.Run(current, func(t *testing.T) {
			calls := 0
			err := signalBoundaryGroup(boundaryProcessOwner{PID: 4321, Start: "owned"}, func(string) string { return current }, func(int) error { return nil }, func(pid int, sig syscall.Signal) error {
				calls++
				if pid != -4321 || sig != syscall.SIGKILL {
					t.Fatalf("unexpected signal %d %v", pid, sig)
				}
				return nil
			}, syscall.SIGKILL)
			if current == "owned" {
				if calls != 1 || err != nil {
					t.Fatalf("owned signal calls=%d err=%v", calls, err)
				}
			} else if calls != 0 || err == nil {
				t.Fatalf("unproved signal calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestBoundaryGroupSignalGoneOrUnobservable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		probe error
		gone  bool
	}{{"gone", syscall.ESRCH, true}, {"permission", syscall.EPERM, false}, {"unknown-alive", nil, false}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := signalBoundaryGroup(boundaryProcessOwner{PID: 4321, Start: "owned"}, func(string) string { return "" }, func(int) error { return tc.probe }, func(int, syscall.Signal) error { calls++; return nil }, syscall.SIGKILL)
			if calls != 0 || err == nil || errors.Is(err, os.ErrProcessDone) != tc.gone {
				t.Fatalf("calls=%d err=%v gone=%v", calls, err, tc.gone)
			}
		})
	}
}

type boundaryRecoveryEndpoint interface {
	ObserveDrive(context.Context) (couchmessage.DriveObservation, error)
	StartDrive(context.Context, string, couchmessage.DriveStart) (agentdrive.Result, error)
	DriveStatus(context.Context, string) (agentdrive.Result, error)
}

func boundaryRecoveryDrive(ctx context.Context, endpoint boundaryRecoveryEndpoint) (agentdrive.Result, error) {
	request := agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}}
	var result agentdrive.Result
	for attempt := 0; attempt < 3; attempt++ {
		var observation couchmessage.DriveObservation
		for {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			var err error
			observation, err = endpoint.ObserveDrive(ctx)
			if err != nil {
				return result, err
			}
			if observation.Recoverable && !observation.Idle {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		// Each attempt is a NEW request after a proved zero-input refusal, never
		// replay of an accepted/uncertain effect. The outer context bounds all tries.
		id := fmt.Sprintf("boundary-recovery-%d", attempt)
		deadline := time.Now().Add(5 * time.Second)
		if outer, ok := ctx.Deadline(); ok && outer.Before(deadline) {
			deadline = outer
		}
		var err error
		result, err = endpoint.StartDrive(ctx, id, couchmessage.DriveStart{Request: request, Instance: observation.Instance, Sequence: observation.Sequence, Deadline: deadline})
		if err != nil {
			return result, err
		}
		for !result.Terminal() {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			time.Sleep(20 * time.Millisecond)
			result, err = endpoint.DriveStatus(ctx, id)
			if err != nil {
				return result, err
			}
		}
		if result.Outcome != agentdrive.OutcomeRefused || result.Step != 0 {
			return result, nil
		}
		switch result.Detail {
		case "wrapper observation changed; inspect the slot and send a new request",
			"agent composer must be empty with no turn or picker; wait until it is idle",
			"wait until the agent composer is empty and its turn finishes":
			// These admission/dispatch guards precede every write. Requalify first.
		default:
			return result, nil
		}
	}
	return result, nil
}

type boundaryRecoverySequence struct {
	observations []couchmessage.DriveObservation
	outcomes     []agentdrive.Result
	starts       []couchmessage.DriveStart
	ids          []string
	polls        int
}

func (f *boundaryRecoverySequence) ObserveDrive(context.Context) (couchmessage.DriveObservation, error) {
	if len(f.observations) == 0 {
		return couchmessage.DriveObservation{}, fmt.Errorf("unexpected observation")
	}
	o := f.observations[0]
	f.observations = f.observations[1:]
	return o, nil
}
func (f *boundaryRecoverySequence) StartDrive(_ context.Context, id string, s couchmessage.DriveStart) (agentdrive.Result, error) {
	f.starts = append(f.starts, s)
	f.ids = append(f.ids, id)
	if !s.Request.RecoveryOnly() {
		return agentdrive.Result{}, fmt.Errorf("not recovery")
	}
	return agentdrive.Result{}, nil
}
func (f *boundaryRecoverySequence) DriveStatus(context.Context, string) (agentdrive.Result, error) {
	f.polls++
	if len(f.outcomes) == 0 {
		return agentdrive.Result{}, fmt.Errorf("unexpected status")
	}
	r := f.outcomes[0]
	f.outcomes = f.outcomes[1:]
	return r, nil
}
func TestBoundaryRecoveryReobservesAfterZeroInputDispatchRefusal(t *testing.T) {
	f := &boundaryRecoverySequence{
		observations: []couchmessage.DriveObservation{
			{Instance: "wrapper", Sequence: 1, Recoverable: true},
			{Instance: "wrapper", Sequence: 2},
			{Instance: "wrapper", Sequence: 3, Recoverable: true},
		},
		outcomes: []agentdrive.Result{
			{Outcome: agentdrive.OutcomeRefused, Detail: "wait until the agent composer is empty and its turn finishes"},
			{Outcome: agentdrive.OutcomeConfirmed, Step: 1},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := boundaryRecoveryDrive(ctx, f)
	if err != nil || result.Outcome != agentdrive.OutcomeConfirmed || len(f.starts) != 2 || f.starts[0].Sequence != 1 || f.starts[1].Sequence != 3 || f.ids[0] == f.ids[1] {
		t.Fatalf("result=%+v err=%v starts=%+v ids=%v", result, err, f.starts, f.ids)
	}
}
func TestBoundaryRecoveryDoesNotReplayUncertainInput(t *testing.T) {
	f := &boundaryRecoverySequence{observations: []couchmessage.DriveObservation{{Instance: "wrapper", Sequence: 1, Recoverable: true}}, outcomes: []agentdrive.Result{{Outcome: agentdrive.OutcomeUnconfirmed, Detail: "write outcome unknown"}}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := boundaryRecoveryDrive(ctx, f)
	if err != nil || result.Outcome != agentdrive.OutcomeUnconfirmed || len(f.starts) != 1 {
		t.Fatalf("%+v %v %+v", result, err, f.starts)
	}
}
