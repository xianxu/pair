package wrapcmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/creack/pty"
	"golang.org/x/term"
)

const (
	harnessTTYStartupTimeout = 15 * time.Second
	harnessTTYShutdownGrace  = 2 * time.Second
	harnessTTYRetentionLimit = 1 << 20
)

type harnessTTYCaptureRequest struct {
	Executable     string
	Args           []string
	Env            []string
	Dir            string
	Startup        func([]byte) bool
	StartupTimeout time.Duration
	ShutdownGrace  time.Duration
	OnStart        func(*exec.Cmd)
	Classify       func(chunk, retained []byte) harnessTTYConformanceState
	Input          func(retained []byte) []byte
}

func captureHarnessTTY(req harnessTTYCaptureRequest) (out []byte, err error) {
	out, _, err = captureHarnessTTYClassified(req)
	return out, err
}

func captureHarnessTTYConformance(req harnessTTYCaptureRequest, classify func(chunk, retained []byte) harnessTTYConformanceState) ([]byte, harnessTTYConformanceState, error) {
	req.Classify = classify
	return captureHarnessTTYClassified(req)
}

func captureHarnessTTYClassified(req harnessTTYCaptureRequest) (out []byte, state harnessTTYConformanceState, err error) {
	timeout := req.StartupTimeout
	if timeout == 0 {
		timeout = harnessTTYStartupTimeout
	}
	grace := req.ShutdownGrace
	if grace == 0 {
		grace = harnessTTYShutdownGrace
	}

	cmd := exec.Command(req.Executable, req.Args...)
	cmd.Dir = req.Dir
	if req.Env != nil {
		cmd.Env = req.Env
	}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 38, Cols: 120})
	if err != nil {
		return nil, state, fmt.Errorf("start %q: %w", req.Executable, err)
	}
	if req.OnStart != nil {
		req.OnStart(cmd)
	}

	var writeMu sync.Mutex
	writePTY := func(data []byte) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_, _ = ptmx.Write(data)
	}

	chunks := make(chan []byte)
	readerCancel := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(chunks)
		defer close(readerDone)
		buf := make([]byte, 32*1024)
		var queryTail []byte
		cursorQuery := []byte("\x1b[6n")
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				queryWindow := append(queryTail, buf[:n]...)
				for range bytes.Count(queryWindow, cursorQuery) {
					writePTY([]byte("\x1b[1;1R"))
				}
				keep := len(cursorQuery) - 1
				if keep > len(queryWindow) {
					keep = len(queryWindow)
				}
				queryTail = append(queryTail[:0], queryWindow[len(queryWindow)-keep:]...)
				chunk := append([]byte(nil), buf[:n]...)
				select {
				case chunks <- chunk:
				case <-readerCancel:
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	waitResult := make(chan error, 1)
	go func() { waitResult <- cmd.Wait() }()
	waitBounded := func(timeout time.Duration) (error, bool) {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case waitErr := <-waitResult:
			return waitErr, true
		case <-timer.C:
			return nil, false
		}
	}
	joinReader := func(timeout time.Duration) bool {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-readerDone:
			return true
		case <-timer.C:
			return false
		}
	}

	primaryErr := func() error {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case chunk, ok := <-chunks:
				if !ok {
					return fmt.Errorf("%q exited before startup (%d output bytes)", req.Executable, len(out))
				}
				remaining := harnessTTYRetentionLimit - len(out)
				if remaining > len(chunk) {
					remaining = len(chunk)
				}
				if remaining > 0 {
					out = append(out, chunk[:remaining]...)
				}
				if req.Classify != nil {
					state = req.Classify(chunk, out)
					switch state {
					case harnessTTYRecognized:
						return nil
					case harnessTTYUnauthenticated, harnessTTYWorkspaceTrust, harnessTTYRecognizerDrift:
						return fmt.Errorf("%q live conformance classified %s (%d output bytes)", req.Executable, state, len(out))
					}
				}
				if req.Startup != nil && req.Startup(out) {
					return nil
				}
				if req.Input != nil {
					if input := req.Input(out); len(input) > 0 {
						writePTY(input)
					}
				}
			case <-timer.C:
				if req.Classify != nil {
					state = harnessTTYRecognizerDrift
				}
				return fmt.Errorf("%q startup timed out after %s (%d output bytes)", req.Executable, timeout, len(out))
			}
		}
	}()
	cleanupErr := teardownHarnessTTY(harnessTTYTeardownOps{
		cancel:      func() { close(readerCancel) },
		closePTY:    ptmx.Close,
		signal:      func() error { return cmd.Process.Signal(os.Interrupt) },
		kill:        cmd.Process.Kill,
		waitBounded: waitBounded,
		joinReader:  joinReader,
		grace:       grace,
	})
	return out, state, joinHarnessTTYErrors(primaryErr, cleanupErr)
}

type harnessTTYConformanceState string

const (
	harnessTTYWaiting         harnessTTYConformanceState = "waiting"
	harnessTTYRecognized      harnessTTYConformanceState = "recognized"
	harnessTTYUnauthenticated harnessTTYConformanceState = "unauthenticated"
	harnessTTYWorkspaceTrust  harnessTTYConformanceState = "workspace-trust"
	harnessTTYRecognizerDrift harnessTTYConformanceState = "recognizer-drift"
)

type harnessTTYLiveClassifier struct {
	proxy   *proxy
	rolling []byte
}

func newHarnessTTYLiveClassifier(t *testing.T, harness string) *harnessTTYLiveClassifier {
	t.Helper()
	p := &proxy{agentBasename: harness}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatalf("configure %s live classifier: %v", harness, err)
	}
	if p.ttyProfile == nil || p.terminal == nil || p.ttyProfile.recognize == nil {
		t.Fatalf("%s has no positive-gated live profile", harness)
	}
	return &harnessTTYLiveClassifier{proxy: p}
}

func (c *harnessTTYLiveClassifier) Observe(chunk, retained []byte) harnessTTYConformanceState {
	c.proxy.handleChunk(chunk, &c.rolling)
	visible := strings.ToLower(string(retained))
	for _, marker := range []string{"log in", "login required", "sign in", "authentication required"} {
		if strings.Contains(visible, marker) {
			return harnessTTYUnauthenticated
		}
	}
	for _, marker := range []string{"trust this folder", "trust this workspace", "do you trust"} {
		if strings.Contains(visible, marker) {
			return harnessTTYWorkspaceTrust
		}
	}
	if strings.Contains(visible, "harness-recognizer-drift") {
		return harnessTTYRecognizerDrift
	}
	snapshot := c.proxy.terminal.Snapshot()
	if c.proxy.ttyProfile.recognize(snapshot) {
		return harnessTTYRecognized
	}
	return harnessTTYWaiting
}

func (c *harnessTTYLiveClassifier) Close() error {
	return c.proxy.closeTerminal()
}

type harnessTTYTeardownOps struct {
	cancel      func()
	closePTY    func() error
	signal      func() error
	kill        func() error
	waitBounded func(time.Duration) (error, bool)
	joinReader  func(time.Duration) bool
	grace       time.Duration
}

func teardownHarnessTTY(ops harnessTTYTeardownOps) error {
	var cleanupErrors []error
	ops.cancel()
	if err := ops.closePTY(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("close PTY: %w", err))
	}
	if err := ops.signal(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("interrupt child: %w", err))
	}

	waitErr, reaped := ops.waitBounded(ops.grace)
	if reaped {
		if err := unexpectedHarnessTTYWaitError(waitErr); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	} else {
		if err := ops.kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("kill child: %w", err))
		}
		waitErr, reaped = ops.waitBounded(ops.grace)
		if !reaped {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("child reap exceeded %s", ops.grace))
		} else if err := unexpectedHarnessTTYWaitError(waitErr); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	if !ops.joinReader(ops.grace) {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("PTY reader join exceeded %s", ops.grace))
	}
	return errors.Join(cleanupErrors...)
}

func unexpectedHarnessTTYWaitError(err error) error {
	if err == nil || errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return fmt.Errorf("wait for child: %w", err)
}

func joinHarnessTTYErrors(primaryErr, cleanupErr error) error {
	return errors.Join(primaryErr, cleanupErr)
}

func TestHarnessTTYCaptureNormal(t *testing.T) {
	out, err := captureHarnessTTY(harnessTTYCaptureRequest{
		Executable:     os.Args[0],
		Args:           []string{"-test.run=^TestHarnessTTYControlledChild$", "--", "normal"},
		Env:            append(os.Environ(), "PAIR_HARNESS_TTY_CHILD=1"),
		Startup:        func(out []byte) bool { return bytes.Contains(out, []byte("READY")) },
		StartupTimeout: 500 * time.Millisecond,
		ShutdownGrace:  100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("captureHarnessTTY: %v", err)
	}
	if !bytes.Contains(out, []byte("READY")) {
		t.Fatalf("output = %q, want READY", out)
	}
}

func TestHarnessTTYCaptureAnswersCursorPositionQuery(t *testing.T) {
	out, err := captureHarnessTTY(harnessTTYCaptureRequest{
		Executable:     os.Args[0],
		Args:           []string{"-test.run=^TestHarnessTTYControlledChild$", "--", "cursor-query"},
		Env:            append(os.Environ(), "PAIR_HARNESS_TTY_CHILD=1"),
		Startup:        func(out []byte) bool { return bytes.Contains(out, []byte("READY")) },
		StartupTimeout: 500 * time.Millisecond,
		ShutdownGrace:  100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("captureHarnessTTY: %v (output %q)", err, out)
	}
}

func TestHarnessTTYCaptureMissingExecutable(t *testing.T) {
	_, err := captureHarnessTTY(harnessTTYCaptureRequest{
		Executable: "/pair-test/definitely-missing-muse",
		Startup:    func([]byte) bool { return true },
	})
	if err == nil || !strings.Contains(err.Error(), "start") {
		t.Fatalf("error = %v, want bounded start error", err)
	}
}

func TestHarnessTTYCaptureStartupTimeoutKillsAndReapsChild(t *testing.T) {
	var pid int
	started := time.Now()
	out, err := captureHarnessTTY(harnessTTYCaptureRequest{
		Executable:     os.Args[0],
		Args:           []string{"-test.run=^TestHarnessTTYControlledChild$", "--", "ignore-signals"},
		Env:            append(os.Environ(), "PAIR_HARNESS_TTY_CHILD=1"),
		Startup:        func(out []byte) bool { return bytes.Contains(out, []byte("NEVER")) },
		StartupTimeout: 50 * time.Millisecond,
		ShutdownGrace:  50 * time.Millisecond,
		OnStart:        func(cmd *exec.Cmd) { pid = cmd.Process.Pid },
	})
	if err == nil || !strings.Contains(err.Error(), "startup timed out") {
		t.Fatalf("error = %v, want startup timeout (output %q)", err, out)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("timeout cleanup took %s, want bounded completion", elapsed)
	}
	if processExists(pid) {
		t.Fatalf("child pid %d still exists after timeout", pid)
	}
}

func TestHarnessTTYCaptureCleanupInterruptsAndReapsChild(t *testing.T) {
	marker := t.TempDir() + "/interrupted"
	var pid int
	_, err := captureHarnessTTY(harnessTTYCaptureRequest{
		Executable: os.Args[0],
		Args:       []string{"-test.run=^TestHarnessTTYControlledChild$", "--", "interruptible"},
		Env: append(os.Environ(),
			"PAIR_HARNESS_TTY_CHILD=1",
			"PAIR_HARNESS_TTY_MARKER="+marker,
		),
		Startup:        func(out []byte) bool { return bytes.Contains(out, []byte("READY")) },
		StartupTimeout: 500 * time.Millisecond,
		ShutdownGrace:  500 * time.Millisecond,
		OnStart:        func(cmd *exec.Cmd) { pid = cmd.Process.Pid },
	})
	if err != nil {
		t.Fatalf("captureHarnessTTY: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("cleanup signal marker: %v", err)
	}
	if processExists(pid) {
		t.Fatalf("child pid %d still exists after successful capture", pid)
	}
}

func TestHarnessTTYTeardownSignalFailureStillWaitsAndJoinsReader(t *testing.T) {
	signalErr := errors.New("injected signal failure")
	waitErr := errors.New("injected wait result")
	var order []string
	err := teardownHarnessTTY(harnessTTYTeardownOps{
		cancel:   func() { order = append(order, "cancel") },
		closePTY: func() error { order = append(order, "close"); return nil },
		signal:   func() error { order = append(order, "signal"); return signalErr },
		kill: func() error {
			t.Fatal("kill called after bounded wait observed reap")
			return nil
		},
		waitBounded: func(time.Duration) (error, bool) {
			order = append(order, "wait")
			return waitErr, true
		},
		joinReader: func(time.Duration) bool {
			order = append(order, "join-reader")
			return true
		},
		grace: time.Millisecond,
	})
	if !errors.Is(err, signalErr) || !errors.Is(err, waitErr) {
		t.Fatalf("teardown error = %v, want joined signal and wait failures", err)
	}
	if got, want := strings.Join(order, ","), "cancel,close,signal,wait,join-reader"; got != want {
		t.Fatalf("teardown order = %q, want %q", got, want)
	}
}

func TestHarnessTTYTeardownKillFailureStillObservesReapAndJoinsReader(t *testing.T) {
	killErr := errors.New("injected kill failure")
	var order []string
	waits := 0
	err := teardownHarnessTTY(harnessTTYTeardownOps{
		cancel:   func() { order = append(order, "cancel") },
		closePTY: func() error { order = append(order, "close"); return nil },
		signal:   func() error { order = append(order, "signal"); return nil },
		kill:     func() error { order = append(order, "kill"); return killErr },
		waitBounded: func(time.Duration) (error, bool) {
			waits++
			order = append(order, fmt.Sprintf("wait-%d", waits))
			return nil, waits == 2
		},
		joinReader: func(time.Duration) bool {
			order = append(order, "join-reader")
			return true
		},
		grace: time.Millisecond,
	})
	if !errors.Is(err, killErr) {
		t.Fatalf("teardown error = %v, want kill failure", err)
	}
	if got, want := strings.Join(order, ","), "cancel,close,signal,wait-1,kill,wait-2,join-reader"; got != want {
		t.Fatalf("teardown order = %q, want %q", got, want)
	}
}

func TestHarnessTTYTeardownPTYCloseFailureStillJoinsReader(t *testing.T) {
	closeErr := errors.New("injected PTY close failure")
	var order []string
	err := teardownHarnessTTY(harnessTTYTeardownOps{
		cancel:   func() { order = append(order, "cancel") },
		closePTY: func() error { order = append(order, "close"); return closeErr },
		signal:   func() error { order = append(order, "signal"); return nil },
		kill:     func() error { t.Fatal("unexpected kill"); return nil },
		waitBounded: func(time.Duration) (error, bool) {
			order = append(order, "wait")
			return nil, true
		},
		joinReader: func(time.Duration) bool {
			order = append(order, "join-reader")
			return true
		},
		grace: time.Millisecond,
	})
	if !errors.Is(err, closeErr) {
		t.Fatalf("teardown error = %v, want PTY close failure", err)
	}
	if got, want := strings.Join(order, ","), "cancel,close,signal,wait,join-reader"; got != want {
		t.Fatalf("teardown order = %q, want %q", got, want)
	}
}

func TestHarnessTTYCaptureJoinsPrimaryAndCleanupErrors(t *testing.T) {
	primaryErr := errors.New("capture failed")
	cleanupErr := errors.New("cleanup failed")
	err := joinHarnessTTYErrors(primaryErr, cleanupErr)
	if !errors.Is(err, primaryErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("joined error = %v, want primary and cleanup failures", err)
	}
}

func TestHarnessTTYCaptureBoundsRetainedOutput(t *testing.T) {
	out, err := captureHarnessTTY(harnessTTYCaptureRequest{
		Executable:     os.Args[0],
		Args:           []string{"-test.run=^TestHarnessTTYControlledChild$", "--", "flood"},
		Env:            append(os.Environ(), "PAIR_HARNESS_TTY_CHILD=1"),
		Startup:        func(out []byte) bool { return len(out) == harnessTTYRetentionLimit },
		StartupTimeout: 2 * time.Second,
		ShutdownGrace:  50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("captureHarnessTTY: %v", err)
	}
	if len(out) != harnessTTYRetentionLimit {
		t.Fatalf("retained %d bytes, want %d", len(out), harnessTTYRetentionLimit)
	}
}

func TestHarnessTTYCaptureClassifiesControlledChildStates(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantState harnessTTYConformanceState
		wantErr   bool
	}{
		{name: "recognized composer", mode: "agy-composer", wantState: harnessTTYRecognized},
		{name: "unauthenticated login", mode: "unauthenticated", wantState: harnessTTYUnauthenticated, wantErr: true},
		{name: "workspace trust", mode: "workspace-trust", wantState: harnessTTYWorkspaceTrust, wantErr: true},
		{name: "recognizer drift", mode: "recognizer-drift", wantState: harnessTTYRecognizerDrift, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			classifier := newHarnessTTYLiveClassifier(t, "agy")
			defer classifier.Close()
			out, state, err := captureHarnessTTYConformance(harnessTTYCaptureRequest{
				Executable:     os.Args[0],
				Args:           []string{"-test.run=^TestHarnessTTYControlledChild$", "--", tc.mode},
				Env:            append(os.Environ(), "PAIR_HARNESS_TTY_CHILD=1"),
				StartupTimeout: 500 * time.Millisecond,
				ShutdownGrace:  100 * time.Millisecond,
			}, classifier.Observe)
			if state != tc.wantState {
				t.Fatalf("state = %q, want %q (output bytes=%d, error=%v)", state, tc.wantState, len(out), err)
			}
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, want error=%t", err, tc.wantErr)
			}
			if err != nil && (strings.Contains(err.Error(), "SECRET-CONTROLLED-CHILD") || strings.Contains(err.Error(), "\\x1b")) {
				t.Fatalf("classification error leaked captured bytes: %v", err)
			}
		})
	}
}

func TestHarnessTTYLiveConformance(t *testing.T) {
	harness := os.Getenv("PAIR_LIVE_HARNESS")
	if harness == "" {
		t.Skip("set PAIR_LIVE_HARNESS=agy, codex, or muse to check an installed harness")
	}
	commands := map[string][]string{
		"agy":   {"agy", "--dangerously-skip-permissions"},
		"codex": {"codex", "--no-alt-screen", "-c", "check_for_update_on_startup=false"},
		"muse":  {"muse"},
	}
	command, ok := commands[harness]
	if !ok {
		t.Fatalf("PAIR_LIVE_HARNESS=%q, want agy, codex, or muse", harness)
	}
	executable, err := exec.LookPath(command[0])
	if err != nil {
		t.Fatalf("find installed %s: %v", harness, err)
	}
	env := os.Environ()
	if harness == "muse" {
		env = append(env, "MUSE_NO_AUTO_UPDATE=1")
	}
	versionCmd := exec.Command(executable, "--version")
	versionCmd.Env = env
	versionOut, err := versionCmd.Output()
	if err != nil {
		t.Fatalf("%s --version: %v", executable, err)
	}
	version := strings.TrimSpace(string(versionOut))
	if version == "" {
		t.Fatalf("%s --version returned empty output", harness)
	}

	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	classifier := newHarnessTTYLiveClassifier(t, harness)
	t.Cleanup(func() {
		if err := classifier.Close(); err != nil {
			t.Errorf("close %s live classifier: %v", harness, err)
		}
	})
	out, state, err := captureHarnessTTYConformance(harnessTTYCaptureRequest{
		Executable: executable,
		Args:       command[1:],
		Env:        env,
		Dir:        repoRoot,
	}, classifier.Observe)
	if err != nil {
		t.Fatalf("capture %s startup: state=%s: %v", harness, state, err)
	}
	prefixEnd := firstRecognizedHarnessTTYPrefix(t, harness, out)
	if prefixEnd == 0 {
		t.Fatalf("%s reported recognition but no recognized prefix; recapture destination: %s", harness, harnessTTYRecaptureDestination(harness, version, "composer.raw"))
	}
	out = out[:prefixEnd]
	digest := sha256.Sum256(out)
	recaptureDestination := harnessTTYRecaptureDestination(harness, version, "composer.raw")
	t.Logf("%s executable=%s version=%q argv=%q bytes=%d sha256=%s", harness, executable, version, command, len(out), hex.EncodeToString(digest[:]))
	writeHarnessTTYCaptureIfRequested(t, harness, harness, repoRoot, out)

	// The live check defends behavior, not bytes. Real harness output embeds
	// per-account, per-machine, per-moment content (signed-in address, model
	// name, rate-limit banners, rotating tips) and harnesses self-update, so
	// byte identity against a checked-in fixture is unachievable and its
	// failures say nothing about Return correctness. Byte-level exactness is
	// the fixture replay's job, over frozen bytes.
	assertHarnessTTYLiveDecision(t, harness, out, true)

	metadataPath := filepath.Join("testdata", "tty", harness, ttyFixtureVersionDir(version), "metadata.json")
	if _, err := os.Stat(metadataPath); err != nil {
		t.Fatalf("%s has no fixture for installed version %q; capture one to %s", harness, version, recaptureDestination)
	}
	metadata, _ := readHarnessTTYFixture(t, metadataPath)
	if metadata.Version != version || !reflect.DeepEqual(metadata.Command, command) {
		t.Logf("%s fixture identity drift (not a failure): live version=%q argv=%q, fixture version=%q argv=%q; recapture destination: %s",
			harness, version, command, metadata.Version, metadata.Command, recaptureDestination)
	}
}

// assertHarnessTTYLiveDecision replays a live capture through the production
// profile and asserts the Return decision the captured state must produce.
func assertHarnessTTYLiveDecision(t *testing.T, harness string, raw []byte, wantComposer bool) {
	t.Helper()
	p := &proxy{agentBasename: harness}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatalf("configure %s decision proxy: %v", harness, err)
	}
	defer func() {
		if err := p.closeTerminal(); err != nil {
			t.Errorf("close %s decision proxy: %v", harness, err)
		}
	}()
	var rolling []byte
	p.handleChunk(raw, &rolling)
	snapshot := p.terminal.Snapshot()
	if got := p.ttyProfile.recognize(snapshot); got != wantComposer {
		t.Fatalf("%s composer recognized = %t, want %t (cursor=(%d,%d) visible=%t)",
			harness, got, wantComposer, snapshot.Cursor.X, snapshot.Cursor.Y, snapshot.CursorVisible)
	}
	// emitPlainCR is the production Return path: it consumes overlay state
	// under the detector's own lock and then decides, so asserting on it
	// covers both layers rather than the recognizer alone.
	overlayArmed := p.pickerActive.Load()
	got := p.emitPlainCR(nil)
	want := []byte{'\r'}
	if wantComposer {
		want = p.ttyProfile.keymap.plainCR
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s plain Return = %q, want %q (overlay armed=%t, composer=%t)",
			harness, got, want, overlayArmed, wantComposer)
	}
	t.Logf("%s plain Return = %q (composer=%t, overlay armed=%t)", harness, got, wantComposer, overlayArmed)
}

// harnessTTYDrivenScenario is a non-startup screen reachable from the composer
// with one keystroke, plus the gate result that screen must produce.
type harnessTTYDrivenScenario struct {
	name         string
	args         []string
	send         string
	until        string
	wantComposer bool
	file         string
}

var harnessTTYDrivenScenarios = map[string][]harnessTTYDrivenScenario{
	// Codex paints its update interstitial whenever a newer release exists and
	// the startup check is left enabled. Its footer is a registered picker
	// marker, so this is overlay-layer evidence as well as gate evidence.
	"codex": {{
		name: "update interstitial", args: []string{"--no-alt-screen"},
		until: "Press enter to continue", wantComposer: false, file: "overlay.raw",
	}},
	"agy": {
		// The shortcut sheet replaces the composer entirely.
		{name: "shortcut sheet", args: []string{"--dangerously-skip-permissions"},
			send: "?", until: "shortcuts", wantComposer: false, file: "overlay.raw"},
		// The slash menu keeps the composer live below its own box, and Agy
		// paints the menu's selection marker in the SAME bright blue as the
		// composer prompt — so prompt color cannot separate the two. Pinned
		// because that is the assumption an Agy recognizer is most likely to
		// make wrongly. Sending LF here inserts a newline rather than
		// selecting, which is why the gate staying open is tolerable.
		{name: "slash menu", args: []string{"--dangerously-skip-permissions"},
			send: "/", until: "Navigate", wantComposer: true, file: "menu.raw"},
	},
}

// TestHarnessTTYLiveDrivenConformance drives the installed harness one
// keystroke past startup and checks the gate's answer on the resulting screen.
func TestHarnessTTYLiveDrivenConformance(t *testing.T) {
	harness := os.Getenv("PAIR_LIVE_HARNESS")
	if harness == "" {
		t.Skip("set PAIR_LIVE_HARNESS to drive an installed harness past startup")
	}
	scenarios, ok := harnessTTYDrivenScenarios[harness]
	if !ok {
		t.Skipf("no driven scenario recorded for %s", harness)
	}
	executable, err := exec.LookPath(harness)
	if err != nil {
		t.Fatalf("find installed %s: %v", harness, err)
	}
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	only := os.Getenv("PAIR_LIVE_SCENARIO")
	for _, scenario := range scenarios {
		if only != "" && scenario.name != only {
			continue
		}
		t.Run(scenario.name, func(t *testing.T) {
			out, err := driveHarnessTTYScenario(t, harness, executable, repoRoot, scenario)
			if err != nil {
				t.Skipf("%s did not reach %q; it may not be reproducible right now: %v", harness, scenario.until, err)
			}
			assertHarnessTTYLiveDecision(t, harness, out, scenario.wantComposer)
			writeHarnessTTYCaptureIfRequested(t, harness, harness+" "+scenario.name, repoRoot, out)
		})
	}
}

// driveHarnessTTYScenario waits for the composer, sends the scenario keystroke,
// and captures until the scenario's screen text appears.
func driveHarnessTTYScenario(t *testing.T, harness, executable, repoRoot string, scenario harnessTTYDrivenScenario) ([]byte, error) {
	t.Helper()
	composer := newHarnessTTYLiveClassifier(t, harness)
	t.Cleanup(func() {
		if err := composer.Close(); err != nil {
			t.Errorf("close %s scenario classifier: %v", harness, err)
		}
	})
	ready, sent := false, false
	return captureHarnessTTY(harnessTTYCaptureRequest{
		Executable: executable,
		Args:       scenario.args,
		Env:        os.Environ(),
		Dir:        repoRoot,
		Classify: func(chunk, retained []byte) harnessTTYConformanceState {
			if composer.Observe(chunk, retained) == harnessTTYRecognized {
				ready = true
			}
			return harnessTTYWaiting
		},
		Input: func([]byte) []byte {
			if sent || !ready || scenario.send == "" {
				return nil
			}
			sent = true
			return []byte(scenario.send)
		},
		Startup: func(retained []byte) bool {
			if scenario.send != "" && !sent {
				return false
			}
			return strings.Contains(strings.ToLower(string(stripTerminalControls(retained))), strings.ToLower(scenario.until))
		},
	})
}

func firstRecognizedHarnessTTYPrefix(t *testing.T, harness string, raw []byte) int {
	t.Helper()
	classifier := newHarnessTTYLiveClassifier(t, harness)
	defer func() {
		if err := classifier.Close(); err != nil {
			t.Errorf("close %s prefix classifier: %v", harness, err)
		}
	}()
	for i := range raw {
		if classifier.Observe(raw[i:i+1], raw[:i+1]) == harnessTTYRecognized {
			return i + 1
		}
	}
	return 0
}

func harnessTTYRecaptureDestination(harness, version, file string) string {
	return filepath.Join("cmd", "internal", "wrapcmd", "testdata", "tty", harness, ttyFixtureVersionDir(version), file)
}

func TestMuseFixtureEvidence(t *testing.T) {
	fixtureDir := "testdata/tty/muse/0.1.0-R708.1"
	raw, err := os.ReadFile(filepath.Join(fixtureDir, "composer.raw"))
	if err != nil {
		t.Fatalf("read literal Muse fixture: %v", err)
	}
	// The fixture must be exactly the shortest prefix the production
	// recognizer accepts, under the same shared rule every harness capture
	// uses — not a per-harness marker scan.
	if end := firstRecognizedHarnessTTYPrefix(t, "muse", raw); end != len(raw) {
		t.Fatalf("literal fixture prefix end = %d, want exact length %d", end, len(raw))
	}

	model := newTerminalModelForTest(t, 120, 38)
	if err := model.Feed(raw); err != nil {
		t.Fatalf("feed literal Muse fixture: %v", err)
	}
	snapshot := model.Snapshot()
	if !museFixtureHasQualifiedComposer(snapshot) {
		t.Fatalf("literal Muse fixture lacks qualified composer: cursor=%v visible=%v", snapshot.Cursor, snapshot.CursorVisible)
	}

	bare := newTerminalModelForTest(t, 120, 38)
	if err := bare.Feed([]byte("\x1b[8;1H›\x1b[?25h\x1b[8;3H")); err != nil {
		t.Fatalf("feed unrelated prompt glyph: %v", err)
	}
	if got := bare.Snapshot().CellAt(0, 7).Content; got != "›" {
		t.Fatalf("unrelated glyph = %q, want bare ›", got)
	}
	if museFixtureHasQualifiedComposer(bare.Snapshot()) {
		t.Fatal("bare unrelated › unexpectedly qualifies as Muse composer evidence")
	}

	withoutRuleFaint := museFixtureWithoutRuleFaint(snapshot)
	if withoutRuleFaint.Cursor != snapshot.Cursor || !withoutRuleFaint.CursorVisible {
		t.Fatalf("style-only negative changed cursor evidence: got %v visible=%v, want %v visible=true", withoutRuleFaint.Cursor, withoutRuleFaint.CursorVisible, snapshot.Cursor)
	}
	for _, point := range [][2]int{{0, 6}, {0, 7}, {0, 8}} {
		got, want := withoutRuleFaint.CellAt(point[0], point[1]), snapshot.CellAt(point[0], point[1])
		if got == nil || want == nil || got.Content != want.Content {
			t.Fatalf("style-only negative cell (%d,%d) = %#v, want captured content %#v", point[0], point[1], got, want)
		}
	}
	if withoutRuleFaint.CellAt(0, 6).Style.Attrs&uv.AttrFaint != 0 || withoutRuleFaint.CellAt(0, 8).Style.Attrs&uv.AttrFaint != 0 {
		t.Fatal("style-only negative retained faint rule attributes")
	}
	if museFixtureHasQualifiedComposer(withoutRuleFaint) {
		t.Fatal("captured Muse glyph and geometry without faint rules unexpectedly qualify")
	}

	metadata, rawFiles := readHarnessTTYFixture(t, filepath.Join(fixtureDir, "metadata.json"))
	if metadata.Agent != "muse" || metadata.Version != "Muse Code 0.1.0 (0.1.0-R708.1)" {
		t.Fatalf("fixture identity = %q %q, want captured Muse version", metadata.Agent, metadata.Version)
	}
	if len(metadata.Command) != 1 || metadata.Command[0] != "muse" {
		t.Fatalf("capture command = %q, want [muse]", metadata.Command)
	}
	if !bytes.Equal(rawFiles["composer.raw"], raw) {
		t.Fatal("shared fixture reader returned different composer bytes")
	}
	digest := sha256.Sum256(raw)
	if got, want := metadata.Files["composer.raw"], hex.EncodeToString(digest[:]); got != want {
		t.Fatalf("composer.raw digest = %q, want %q", got, want)
	}
}

func museFixtureHasQualifiedComposer(snapshot terminalSnapshot) bool {
	top, prompt, bottom := snapshot.CellAt(0, 6), snapshot.CellAt(0, 7), snapshot.CellAt(0, 8)
	return snapshot.CursorVisible && snapshot.Cursor.X == 2 && snapshot.Cursor.Y == 7 &&
		top != nil && top.Content == "─" && top.Style.Attrs&uv.AttrFaint != 0 &&
		prompt != nil && prompt.Content == "⟩" && prompt.Style.Attrs&uv.AttrFaint == 0 &&
		bottom != nil && bottom.Content == "─" && bottom.Style.Attrs&uv.AttrFaint != 0
}

func museFixtureWithoutRuleFaint(snapshot terminalSnapshot) terminalSnapshot {
	mutated := cloneTerminalSnapshot(snapshot)
	for _, y := range []int{6, 8} {
		for x := 0; x < mutated.Width; x++ {
			if cell := mutated.CellAt(x, y); cell != nil && cell.Content == "─" {
				cell.Style.Attrs &^= uv.AttrFaint
			}
		}
	}
	return mutated
}

func writeLiteralCapture(destination string, data []byte) error {
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(destination)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, destination)
}

func processExists(pid int) bool {
	return pid > 0 && syscall.Kill(pid, 0) == nil
}

func TestHarnessTTYControlledChild(t *testing.T) {
	if os.Getenv("PAIR_HARNESS_TTY_CHILD") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			switch os.Args[i+1] {
			case "normal":
				fmt.Fprintln(os.Stdout, "READY")
				for {
					time.Sleep(time.Hour)
				}
			case "cursor-query":
				oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
				if err != nil {
					os.Exit(4)
				}
				defer term.Restore(int(os.Stdin.Fd()), oldState)
				_, _ = os.Stdout.Write([]byte("\x1b[6n"))
				reply := make([]byte, len("\x1b[1;1R"))
				if _, err := io.ReadFull(os.Stdin, reply); err != nil || string(reply) != "\x1b[1;1R" {
					os.Exit(5)
				}
				fmt.Fprintln(os.Stdout, "READY")
				for {
					time.Sleep(time.Hour)
				}
			case "ignore-signals":
				signal.Ignore(os.Interrupt, syscall.SIGHUP)
				for {
					time.Sleep(time.Hour)
				}
			case "interruptible":
				signals := make(chan os.Signal, 1)
				signal.Notify(signals, os.Interrupt, syscall.SIGHUP)
				fmt.Fprintln(os.Stdout, "READY")
				<-signals
				if err := os.WriteFile(os.Getenv("PAIR_HARNESS_TTY_MARKER"), []byte("interrupted\n"), 0o600); err != nil {
					os.Exit(3)
				}
				os.Exit(0)
			case "flood":
				block := bytes.Repeat([]byte("x"), 32*1024)
				for j := 0; j < 64; j++ {
					_, _ = os.Stdout.Write(block)
				}
				for {
					time.Sleep(time.Hour)
				}
			case "agy-composer":
				_, _ = os.Stdout.Write([]byte(agyLiveComposerPaint()))
				for {
					time.Sleep(time.Hour)
				}
			case "unauthenticated":
				_, _ = os.Stdout.Write([]byte("\x1b[31mLogin required: SECRET-CONTROLLED-CHILD\x1b[0m"))
				for {
					time.Sleep(time.Hour)
				}
			case "workspace-trust":
				_, _ = os.Stdout.Write([]byte("\x1b[33mDo you trust this workspace? SECRET-CONTROLLED-CHILD\x1b[0m"))
				for {
					time.Sleep(time.Hour)
				}
			case "recognizer-drift":
				_, _ = os.Stdout.Write([]byte("\x1b[2JHARNESS-RECOGNIZER-DRIFT SECRET-CONTROLLED-CHILD"))
				for {
					time.Sleep(time.Hour)
				}
			}
		}
	}
	os.Exit(2)
}

// writeHarnessTTYCaptureIfRequested writes a live capture to the path named by
// PAIR_LIVE_CAPTURE_OUT, if set. Relative paths resolve against the repo root.
func writeHarnessTTYCaptureIfRequested(t *testing.T, harness, label, repoRoot string, out []byte) {
	t.Helper()
	destination := os.Getenv("PAIR_LIVE_CAPTURE_OUT")
	if destination == "" {
		return
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(repoRoot, destination)
	}
	if err := writeLiteralCapture(destination, out); err != nil {
		t.Fatalf("write literal %s capture %s: %v", label, destination, err)
	}
	t.Logf("wrote literal %s capture to %s", label, destination)
}
