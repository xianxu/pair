package wrapcmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
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
}

func captureHarnessTTY(req harnessTTYCaptureRequest) (out []byte, err error) {
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
		return nil, fmt.Errorf("start %q: %w", req.Executable, err)
	}
	if req.OnStart != nil {
		req.OnStart(cmd)
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
					_, _ = ptmx.Write([]byte("\x1b[1;1R"))
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
					return fmt.Errorf("%q exited before startup; output %q", req.Executable, out)
				}
				remaining := harnessTTYRetentionLimit - len(out)
				if remaining > len(chunk) {
					remaining = len(chunk)
				}
				if remaining > 0 {
					out = append(out, chunk[:remaining]...)
				}
				if req.Startup != nil && req.Startup(out) {
					return nil
				}
			case <-timer.C:
				return fmt.Errorf("%q startup timed out after %s; output %q", req.Executable, timeout, out)
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
	return out, joinHarnessTTYErrors(primaryErr, cleanupErr)
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
		Startup:        func([]byte) bool { return false },
		StartupTimeout: 100 * time.Millisecond,
		ShutdownGrace:  50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("captureHarnessTTY unexpectedly succeeded")
	}
	if len(out) != harnessTTYRetentionLimit {
		t.Fatalf("retained %d bytes, want %d", len(out), harnessTTYRetentionLimit)
	}
}

func TestHarnessTTYLiveMuseCapture(t *testing.T) {
	if os.Getenv("PAIR_LIVE_HARNESS") != "muse" {
		t.Skip("set PAIR_LIVE_HARNESS=muse to capture installed Muse")
	}
	executable, err := exec.LookPath("muse")
	if err != nil {
		t.Fatalf("find installed Muse: %v", err)
	}
	env := append(os.Environ(), "MUSE_NO_AUTO_UPDATE=1")
	versionCmd := exec.Command(executable, "--version")
	versionCmd.Env = env
	versionOut, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s --version: %v: %s", executable, err, strings.TrimSpace(string(versionOut)))
	}
	version := strings.TrimSpace(string(versionOut))
	if version == "" {
		t.Fatal("Muse --version returned empty output")
	}

	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	out, err := captureHarnessTTY(harnessTTYCaptureRequest{
		Executable: executable,
		Env:        env,
		Dir:        repoRoot,
		Startup: func(out []byte) bool {
			return museComposerPrefixEnd(out) > 0
		},
	})
	if err != nil {
		t.Fatalf("capture Muse startup: %v", err)
	}
	out = out[:museComposerPrefixEnd(out)]
	t.Logf("muse executable=%s version=%q bytes=%d", executable, version, len(out))
	if destination := os.Getenv("PAIR_LIVE_CAPTURE_OUT"); destination != "" {
		if !filepath.IsAbs(destination) {
			destination = filepath.Join(repoRoot, destination)
		}
		if err := writeLiteralCapture(destination, out); err != nil {
			t.Fatalf("write literal Muse capture %s: %v", destination, err)
		}
		t.Logf("wrote literal Muse capture to %s", destination)
	}
}

func TestMuseFixtureEvidence(t *testing.T) {
	fixtureDir := "testdata/tty/muse/0.1.0-R708.1"
	raw, err := os.ReadFile(filepath.Join(fixtureDir, "composer.raw"))
	if err != nil {
		t.Fatalf("read literal Muse fixture: %v", err)
	}
	if end := museComposerPrefixEnd(raw); end != len(raw) {
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

	metadataBytes, err := os.ReadFile(filepath.Join(fixtureDir, "metadata.json"))
	if err != nil {
		t.Fatalf("read Muse fixture metadata: %v", err)
	}
	var metadata struct {
		Agent      string            `json:"agent"`
		Version    string            `json:"version"`
		CapturedAt string            `json:"captured_at"`
		Command    []string          `json:"command"`
		Files      map[string]string `json:"files"`
	}
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatalf("decode Muse fixture metadata: %v", err)
	}
	if metadata.Agent != "muse" || metadata.Version != "Muse Code 0.1.0 (0.1.0-R708.1)" {
		t.Fatalf("fixture identity = %q %q, want captured Muse version", metadata.Agent, metadata.Version)
	}
	if _, err := time.Parse(time.RFC3339, metadata.CapturedAt); err != nil {
		t.Fatalf("captured_at %q is not RFC3339: %v", metadata.CapturedAt, err)
	}
	if len(metadata.Command) != 1 || metadata.Command[0] != "muse" {
		t.Fatalf("capture command = %q, want [muse]", metadata.Command)
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
	tmp, err := os.CreateTemp(dir, ".composer.raw.*")
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

func museComposerPrefixEnd(out []byte) int {
	if !bytes.Contains(out, []byte("\x1b[7;1H\x1b[2m── ")) ||
		!bytes.Contains(out, []byte("\x1b[8;1H\x1b[22m⟩")) ||
		!bytes.Contains(out, []byte("\x1b[9;1H\x1b[2m────")) {
		return 0
	}
	marker := []byte("\x1b[?25h\x1b[8;3H")
	if index := bytes.Index(out, marker); index >= 0 {
		return index + len(marker)
	}
	return 0
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
			}
		}
	}
	os.Exit(2)
}
