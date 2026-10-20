package termcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestInitialZeroGeometryWaitsBeforeStartingChild(t *testing.T) {
	host := &presentationSessionHost{FakeHost: hostty.NewFakeHost(ptychild.Size{}), readDone: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- runShellOnHost(host, &presentationSessionRuntime{}) }()
	select {
	case err := <-done:
		t.Fatalf("zero geometry ended session before resize: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if got := host.Written(); got != "" {
		t.Fatalf("child painted before valid geometry: %q", got)
	}
	host.SetSize(ptychild.Size{Rows: 5, Cols: 80})
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("valid geometry did not release launch")
	}
}

func TestRecoveryTransitionEveryPair(t *testing.T) {
	type pair struct {
		state  RecoveryState
		effect RecoveryEffect
	}
	// Columns: child exit, wrapper failure, termination, cleanup complete,
	// cleanup failed, handoff. Every rejected event preserves its state.
	want := [6][6]pair{
		{{RecoveryCleanupNormal, RecoveryCleanup}, {RecoveryCleanupFault, RecoveryCleanup}, {RecoveryCleanupTerminated, RecoveryCleanup}, {RecoveryRunning, 0}, {RecoveryRunning, 0}, {RecoveryRunning, 0}},
		{{RecoveryCleanupNormal, 0}, {RecoveryCleanupNormal, 0}, {RecoveryCleanupTerminated, 0}, {RecoveryFinished, RecoveryFinish}, {RecoveryFinished, RecoveryFinish}, {RecoveryCleanupNormal, 0}},
		{{RecoveryCleanupFault, 0}, {RecoveryCleanupFault, 0}, {RecoveryCleanupTerminated, 0}, {RecoveryShellReady, 0}, {RecoveryFinished, RecoveryFinish}, {RecoveryCleanupFault, 0}},
		{{RecoveryCleanupTerminated, 0}, {RecoveryCleanupTerminated, 0}, {RecoveryCleanupTerminated, 0}, {RecoveryFinished, RecoveryFinish}, {RecoveryFinished, RecoveryFinish}, {RecoveryCleanupTerminated, 0}},
		{{RecoveryShellReady, 0}, {RecoveryShellReady, 0}, {RecoveryFinished, RecoveryFinish}, {RecoveryShellReady, 0}, {RecoveryShellReady, 0}, {RecoveryFinished, RecoveryExecShell}},
		{{RecoveryFinished, 0}, {RecoveryFinished, 0}, {RecoveryFinished, 0}, {RecoveryFinished, 0}, {RecoveryFinished, 0}, {RecoveryFinished, 0}},
	}
	for state := RecoveryRunning; state <= RecoveryFinished; state++ {
		for event := RecoveryChildExit; event <= RecoveryHandoff; event++ {
			got, effect := RecoveryTransition(state, event)
			if got != want[state][event].state || effect != want[state][event].effect {
				t.Fatalf("state %d event %d: (%d,%d), want %+v", state, event, got, effect, want[state][event])
			}
		}
	}
	// Termination dominates even a later fault/completion, and cleanup stays owed.
	for _, first := range []RecoveryEvent{RecoveryChildExit, RecoveryWrapperFailure, RecoveryHostTermination} {
		state, _ := RecoveryTransition(RecoveryRunning, first)
		state, _ = RecoveryTransition(state, RecoveryHostTermination)
		state, effect := RecoveryTransition(state, RecoveryHandoff)
		if state != RecoveryCleanupTerminated || effect != RecoveryNoEffect {
			t.Fatal("termination abandoned cleanup")
		}
		state, _ = RecoveryTransition(state, RecoveryCleanupComplete)
		_, effect = RecoveryTransition(state, RecoveryHandoff)
		if effect == RecoveryExecShell {
			t.Fatal("termination launched shell")
		}
	}
}

func TestInitialGeometryDeadlineAndTermination(t *testing.T) {
	t.Run("deadline", func(t *testing.T) {
		host := hostty.NewFakeHost(ptychild.Size{})
		start := time.Now()
		_, err := waitInitialGeometry(host, 25*time.Millisecond)
		if err == nil || time.Since(start) < 20*time.Millisecond {
			t.Fatalf("missing bounded wait: %v", err)
		}
	})
	t.Run("termination", func(t *testing.T) {
		host := hostty.NewFakeHost(ptychild.Size{})
		host.Terminate(syscall.SIGHUP)
		_, err := waitInitialGeometry(host, time.Second)
		if !errors.Is(err, errHostTermination) {
			t.Fatalf("lost termination: %v", err)
		}
	})
	t.Run("closed resize", func(t *testing.T) {
		host := hostty.NewFakeHost(ptychild.Size{})
		host.Close()
		_, err := waitInitialGeometry(host, time.Second)
		if err == nil {
			t.Fatal("closed resize source accepted")
		}
	})
}

type recoveryHost struct {
	*presentationSessionHost
	sizeErr                        error
	restoreStarted, restoreRelease chan struct{}
	restoreErr                     error
	postCloseSignal                os.Signal
}

func (h *recoveryHost) Size() (ptychild.Size, error) {
	if h.sizeErr != nil {
		return ptychild.Size{}, h.sizeErr
	}
	return h.FakeHost.Size()
}
func (h *recoveryHost) MakeRaw() (func() error, error) {
	restore, err := h.FakeHost.MakeRaw()
	return func() error {
		if h.restoreStarted != nil {
			close(h.restoreStarted)
			<-h.restoreRelease
		}
		return errors.Join(restore(), h.restoreErr)
	}, err
}
func (h *recoveryHost) Close() error {
	if h.postCloseSignal != nil {
		h.Terminate(h.postCloseSignal)
	}
	return h.FakeHost.Close()
}
func newRecoveryHost() *recoveryHost {
	return &recoveryHost{presentationSessionHost: &presentationSessionHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 5, Cols: 80}), readDone: make(chan struct{})}}
}
func TestRecoveryWaitsForRestoreAndSuppressesTermination(t *testing.T) {
	for _, termination := range []string{"none", "during restore", "at close"} {
		t.Run(termination, func(t *testing.T) {
			host := newRecoveryHost()
			host.sizeErr = errors.New("controlled wrapper failure")
			host.restoreStarted, host.restoreRelease = make(chan struct{}), make(chan struct{})
			if termination == "at close" {
				host.postCloseSignal = syscall.SIGHUP
			}
			calls := 0
			done := make(chan error, 1)
			go func() {
				done <- runShellRecovery(host, &presentationSessionRuntime{}, nil, true, func(_ string, _ []string, _ []string, reason error) error {
					calls++
					if reason == nil || !host.Closed() || host.RawDepth() != 0 {
						t.Error("handoff before restored host and failure")
					}
					return errors.New("exec unavailable")
				})
			}()
			<-host.restoreStarted
			select {
			case <-done:
				t.Fatal("cleanup did not wait for raw restore")
			default:
			}
			if termination == "during restore" {
				host.Terminate(syscall.SIGTERM)
			}
			close(host.restoreRelease)
			err := <-done
			if err == nil {
				t.Fatal("lost original failure")
			}
			want := 0
			if termination == "none" {
				want = 1
			}
			if calls != want {
				t.Fatalf("shell calls %d want %d", calls, want)
			}
		})
	}
}
func TestRecoveryNoShellAfterNormalExitOrUnusableTerminal(t *testing.T) {
	for _, mode := range []string{"normal", "normal cleanup error", "dead io", "missing stdio", "restore failure", "termination"} {
		t.Run(mode, func(t *testing.T) {
			host := newRecoveryHost()
			usable := true
			switch mode {
			case "normal cleanup error":
				host.restoreErr = errors.New("restore failed")
			case "dead io":
				host.sizeErr = syscall.EIO
			case "missing stdio":
				host.sizeErr = errors.New("setup failed")
				usable = false
			case "restore failure":
				host.sizeErr = errors.New("setup failed")
				host.restoreErr = errors.New("restore failed")
			case "termination":
				host.Terminate(syscall.SIGHUP)
			}
			calls := 0
			runShellRecovery(host, &presentationSessionRuntime{}, nil, usable, func(string, []string, []string, error) error { calls++; return nil })
			if calls != 0 {
				t.Fatal("unexpected new shell")
			}
		})
	}
}

func TestPresentationZeroResizeRetainsInputOutputAndIdentity(t *testing.T) {
	mux, parent := presentationFixture(t)
	child := addPresentationTab(t, mux, 1, "before")
	id := child.Endpoint().ID()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 5, Cols: 8})
	before := len(child.Resizes())
	host.SetSize(ptychild.Size{})
	mux.inheritSize(host)
	if len(child.Resizes()) != before || mux.rows != 5 || mux.cols != 8 {
		t.Fatal("zero geometry changed child")
	}
	mux.writeEvent(terminal.InputEvent{Event: uv.KeyPressEvent{Code: 'x', Text: "x"}})
	child.Feed([]byte("\x1b[1;1Hafter"))
	flushPresentation(t, mux, child)
	if got := string(bytes.Join(child.Writes(), nil)); got != "x" {
		t.Fatalf("input lost: %q", got)
	}
	if !strings.Contains(renderedParent(t, parent, 8, 5), "after") {
		t.Fatal("output stopped while zero sized")
	}
	host.SetSize(ptychild.Size{Rows: 6, Cols: 10})
	mux.inheritSize(host)
	if mux.failure != nil || child.Endpoint().ID() != id || mux.presenter.View().Admitted != id || child.Done() {
		t.Fatalf("resize replaced endpoint or failed: %v", mux.failure)
	}
	frame, err := child.Endpoint().Snapshot(time.Now())
	if err != nil || frame.Geometry.Rows != 5 || frame.Geometry.Cols != 10 {
		t.Fatalf("valid resize not restored: %+v %v", frame.Geometry, err)
	}
}

type failedGeometryOSHost struct {
	*hostty.OSHost
	termination bool
}

func (h failedGeometryOSHost) Size() (ptychild.Size, error) {
	if h.termination {
		select {
		case signal := <-h.Terminated():
			if signal != nil {
				return ptychild.Size{}, errHostTermination
			}
		case <-time.After(time.Second):
		}
	}
	return ptychild.Size{}, errors.New("fixture wrapper geometry failure")
}

type nativeRecoveryRuntime struct{ fakeRuntime }

func (*nativeRecoveryRuntime) ShellCommand() (string, []string) { return "/bin/sh", []string{"-i"} }

func TestRecoveryNativePTYFallback(t *testing.T) {
	if mode := os.Getenv("PAIR_TEST_NATIVE_RECOVERY"); mode != "" {
		host := failedGeometryOSHost{OSHost: hostty.NewOSHost(os.Stdin, os.Stdout), termination: mode == "termination"}
		if host.termination {
			syscall.Kill(os.Getpid(), syscall.SIGTERM)
		}
		err := runShellRecovery(host, &nativeRecoveryRuntime{}, nil, true, execNativeShell, "TERM=virtual-should-not-leak", "TERMINFO=/virtual/profile")
		if host.termination && errors.Is(err, errHostTermination) {
			fmt.Fprintln(os.Stdout, "TERMINATED_WITHOUT_HANDOFF")
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(91)
	}
	for _, mode := range []string{"shell", "termination"} {
		t.Run(mode, func(t *testing.T) {
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestRecoveryNativePTYFallback$")
			cmd.Env = append(os.Environ(), "PAIR_TEST_NATIVE_RECOVERY="+mode, "TERM=original-native", "TERMINFO=/original/profile")
			master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 8, Cols: 80})
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			// Split the success token in the command so echo cannot satisfy it.
			_, writeErr := master.Write([]byte("printf 'NATIVE_%s:%s:%s\\n' OK \"$TERM\" \"$TERMINFO\"; exit\n"))
			if writeErr != nil && mode == "shell" {
				t.Fatal(writeErr)
			}
			output := make(chan string, 1)
			go func() { b, _ := io.ReadAll(master); output <- string(b) }()
			waitErr := cmd.Wait()
			var got string
			select {
			case got = <-output:
			case <-ctx.Done():
				master.Close()
				t.Fatal("native shell PTY did not drain")
			}
			if waitErr != nil {
				t.Fatalf("native shell: %v output=%q", waitErr, got)
			}
			if mode == "shell" {
				if !strings.Contains(got, "NATIVE_OK:original-native:/original/profile") {
					t.Fatalf("fallback did not accept command with original environment: %q", got)
				}
			} else if !strings.Contains(got, "TERMINATED_WITHOUT_HANDOFF") || strings.Contains(got, "NATIVE_OK") {
				t.Fatalf("termination became a shell: %q", got)
			}
		})
	}
}

type closedSignalHost struct {
	*presentationSessionHost
	signals chan os.Signal
}

func (h *closedSignalHost) Terminated() <-chan os.Signal { return h.signals }

type commandRecoveryRuntime struct {
	fakeRuntime
	command string
}

func (r *commandRecoveryRuntime) ShellCommand() (string, []string) {
	return "/bin/sh", []string{"-c", r.command}
}
func TestRecoveryClosedSignalStreamDoesNotTerminateLiveChild(t *testing.T) {
	signals := make(chan os.Signal)
	close(signals)
	host := &closedSignalHost{presentationSessionHost: &presentationSessionHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 5, Cols: 80}), readDone: make(chan struct{})}, signals: signals}
	err := runShellOnHost(host, &commandRecoveryRuntime{command: "sleep 0.05; printf completed-after-closed-signal"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(host.Written(), "completed-after-closed-signal") {
		t.Fatalf("closed signal channel killed live child: %q", host.Written())
	}
}

type failingPaintRecoveryHost struct{ *recoveryHost }

func (h *failingPaintRecoveryHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if bytes.Contains(p, []byte("PAINTFAIL")) {
		return 0, errors.New("fixture paint failure")
	}
	return h.FakeHost.WriteContext(ctx, p)
}
func TestRecoveryRunningChildAndProducersFinishBeforeHandoff(t *testing.T) {
	for _, terminate := range []bool{false, true} {
		t.Run(fmt.Sprintf("terminate=%v", terminate), func(t *testing.T) {
			base := newRecoveryHost()
			base.restoreStarted, base.restoreRelease = make(chan struct{}), make(chan struct{})
			host := &failingPaintRecoveryHost{base}
			pidFile := filepath.Join(t.TempDir(), "child.pid")
			runtime := &commandRecoveryRuntime{command: "echo $$ > '" + pidFile + "'; printf PAINTFAIL; exec sleep 30"}
			calls := 0
			done := make(chan error, 1)
			go func() {
				done <- runShellRecovery(host, runtime, nil, true, func(string, []string, []string, error) error { calls++; return errors.New("fixture exec failed") })
			}()
			select {
			case <-base.restoreStarted:
			case <-time.After(3 * time.Second):
				t.Fatal("fault did not reach raw restore")
			}
			select {
			case <-base.readDone:
			default:
				t.Fatal("input producer still running at raw restore")
			}
			rawPID, err := os.ReadFile(pidFile)
			if err != nil {
				t.Fatal(err)
			}
			pid, err := strconv.Atoi(strings.TrimSpace(string(rawPID)))
			if err != nil {
				t.Fatal(err)
			}
			if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
				t.Fatalf("child still alive at restore: %d %v", pid, err)
			}
			if terminate {
				base.Terminate(syscall.SIGHUP)
			}
			close(base.restoreRelease)
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("fault not reported")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("session teardown stalled")
			}
			want := 1
			if terminate {
				want = 0
			}
			if calls != want {
				t.Fatalf("exec calls=%d want=%d", calls, want)
			}
			if !strings.Contains(base.Written(), "\x1b[?2004l") {
				t.Fatal("presenter did not restore bracketed paste before handoff")
			}
		})
	}
}

type measuredGeometryHost struct {
	*hostty.FakeHost
	measurements chan ptychild.Size
}

func (h *measuredGeometryHost) Size() (ptychild.Size, error) {
	size, err := h.FakeHost.Size()
	h.measurements <- size
	return size, err
}
func TestInitialGeometryMeasuresOnlyOnResizeAndRequiresBothDimensions(t *testing.T) {
	host := &measuredGeometryHost{FakeHost: hostty.NewFakeHost(ptychild.Size{}), measurements: make(chan ptychild.Size, 8)}
	done := make(chan error, 1)
	go func() { _, err := waitInitialGeometry(host, time.Second); done <- err }()
	<-host.measurements
	select {
	case size := <-host.measurements:
		t.Fatalf("geometry busy retry: %+v", size)
	case <-time.After(20 * time.Millisecond):
	}
	for _, size := range []ptychild.Size{{Rows: 5}, {Cols: 80}, {Rows: 5, Cols: 80}} {
		host.SetSize(size)
		select {
		case measured := <-host.measurements:
			if measured != size {
				t.Fatalf("measured %+v want %+v", measured, size)
			}
		case <-time.After(time.Second):
			t.Fatal("resize not measured")
		}
		if size.Rows == 0 || size.Cols == 0 {
			select {
			case err := <-done:
				t.Fatalf("partial geometry accepted: %v", err)
			default:
			}
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryActiveSessionTerminationRemainsTypedThroughCleanupError(t *testing.T) {
	host := newRecoveryHost()
	host.restoreErr = errors.New("fixture restore error")
	calls := 0
	done := make(chan error, 1)
	go func() {
		done <- runShellRecovery(host, &commandRecoveryRuntime{command: "printf READY_TO_TERMINATE; exec sleep 30"}, nil, true, func(string, []string, []string, error) error { calls++; return nil })
	}()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ready := false
	for !ready {
		select {
		case output := <-host.Writes():
			ready = bytes.Contains(output, []byte("READY_TO_TERMINATE"))
		case <-deadline.C:
			t.Fatal("child not ready")
		}
	}
	host.Terminate(syscall.SIGTERM)
	select {
	case err := <-done:
		var ending *sessionEnding
		if !errors.As(err, &ending) || ending.event != RecoveryHostTermination || !errors.Is(err, errHostTermination) || ending.cleanup == nil {
			t.Fatalf("termination marker lost through cleanup: %v", err)
		}
	case <-deadline.C:
		t.Fatal("termination teardown stalled")
	}
	if calls != 0 {
		t.Fatal("termination launched a new shell")
	}
}

func TestRecoveryProductionProfileFailureUsesNativeShellOnTTY(t *testing.T) {
	root := filepath.Join(t.TempDir(), "blocked-data-root")
	if err := os.WriteFile(root, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_DATA_DIR", root)
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	var report bytes.Buffer
	calls := 0
	code := runShellWithExec(slave, slave, &report, &nativeRecoveryRuntime{}, func(name string, args, env []string, reason error) error {
		calls++
		if name != "/bin/sh" || len(args) != 1 || args[0] != "-i" || reason == nil {
			t.Errorf("bad native handoff %s %v %v", name, args, reason)
		}
		return nil
	})
	if code != 0 || calls != 1 {
		t.Fatalf("profile failure code=%d calls=%d report=%q", code, calls, report.String())
	}
	if !strings.Contains(report.String(), "plain shell") {
		t.Fatalf("missing fallback reason: %q", report.String())
	}
}
func TestRecoveryProductionRejectsNonTTYStdio(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer output.Close()
	var report bytes.Buffer
	calls := 0
	code := runShellWithExec(input, output, &report, &nativeRecoveryRuntime{}, func(string, []string, []string, error) error { calls++; return nil })
	if code != 1 || calls != 0 {
		t.Fatalf("nonTTY code=%d calls=%d report=%q", code, calls, report.String())
	}
}
