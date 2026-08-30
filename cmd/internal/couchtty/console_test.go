package couchtty

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// waitFor polls until cond holds. ONE helper for the package, with the
// deadline as a parameter -- three near-identical pollers is how they drift
// apart on the thing that matters (how long is long enough).
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	waitUpTo(t, 3*time.Second, what, cond)
}

// waitLong is for REAL applications: nvim takes over a second to draw its first
// screen, where a fake child answers in microseconds.
func waitLong(t *testing.T, what string, cond func() bool) {
	t.Helper()
	waitUpTo(t, 15*time.Second, what, cond)
}

func waitUpTo(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

type consoleFixture struct {
	host  *hostty.FakeHost
	child *ptychild.Child
	con   *Console
	stdin *io.PipeWriter
	done  chan int
}

func newFixture(t *testing.T, rows, cols uint16) *consoleFixture {
	t.Helper()
	host := hostty.NewFakeHost(ptychild.Size{Rows: rows, Cols: cols})
	pr, pw := io.Pipe()
	con := New(host, pr)

	child := ptychild.NewFakeChild(nil)
	child.SetSink(func(chunk []byte) { con.Deliver("c1", chunk) })
	con.Attach("c1", "brain", child)
	setTestOps(con, func(string, map[string]string) (any, error) { return nil, nil })

	f := &consoleFixture{host: host, child: child, con: con, stdin: pw, done: make(chan int, 1)}
	go func() { f.done <- con.Run() }()
	t.Cleanup(func() {
		con.Stop()
		_ = pw.Close()
	})
	return f
}

func setTestOps(con *Console, effect func(string, map[string]string) (any, error)) {
	con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		delegate := func(call couchcore.OperationCall) (any, error) {
			if call.Operation.Effect == couchcore.EffectConsole {
				return con.ExecuteConsoleOperation(call)
			}
			return effect(call.Name, call.Args)
		}
		return couchcore.DispatchOperation(couchcore.OperationExecutors{
			DirectStore: delegate,
			LiveOwner:   delegate,
		}, call)
	})
}

func TestConsoleSwitchOperationUsesExactThreadAndRefusesStaleTarget(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.attachThreadActor("c2", "c2", panelAddress("c2"), "c1", "brain", other)

	dispatch := f.con.Ops()
	_, err := dispatch(couchcore.OperationCall{
		Name: "switch", Implicit: true,
		Args: map[string]string{"repo-scope": "legacy", "tag": "missing"},
	})
	if err == nil || !strings.Contains(err.Error(), "not attached") {
		t.Fatalf("stale switch error = %v", err)
	}
	f.con.mu.Lock()
	active := f.con.active
	f.con.mu.Unlock()
	if active != "c1" {
		t.Fatalf("stale switch changed active pane to %q", active)
	}

	_, err = dispatch(couchcore.OperationCall{
		Name: "switch", Implicit: true,
		Args: map[string]string{"repo-scope": panelAddress("c2").RepoScope, "tag": string(panelAddress("c2").Tag)},
	})
	if err != nil {
		t.Fatal(err)
	}
	f.con.mu.Lock()
	active = f.con.active
	f.con.mu.Unlock()
	if active != "c2" {
		t.Fatalf("exact switch left active pane %q", active)
	}
}

func TestActiveChildExitFocusesPanelRecordsCauseAndForgetsActor(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.AttachTree("c2", "/w/pair", "pair", other)

	var forgotTree couchcore.Worktree
	var forgotID couchcore.ActorID
	f.con.SetForget(func(tree couchcore.Worktree, id couchcore.ActorID) error {
		forgotTree, forgotID = tree, id
		return nil
	})
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	f.child.Exit(7)
	waitFor(t, "the active exit to land on the panel", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		_, deadPanePresent := f.con.panes["c1"]
		return f.con.focus.IsPanel() && !deadPanePresent
	})

	if forgotTree != "c1" || forgotID != "c1" {
		t.Fatalf("forgot (%q, %q), want (c1, c1)", forgotTree, forgotID)
	}
	if got := f.host.Written(); !strings.Contains(got, "brain") || !strings.Contains(got, "7") {
		t.Fatalf("exit landing does not name actor and code: %q", got)
	}
	f.con.mu.Lock()
	active := f.con.active
	f.con.mu.Unlock()
	if active != "c2" {
		t.Fatalf("active target after root exit = %q, want remaining actor c2", active)
	}
	select {
	case code := <-f.done:
		t.Fatalf("console exited with %d while another child remained", code)
	default:
	}
}

func TestFinalQueuedOutputIsWrittenBeforeLastChildExit(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.con.Deliver("c1", []byte("final output"))
	f.child.Exit(0)
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("console did not exit")
	}
	if got := f.host.Written(); !strings.Contains(got, "final output") {
		t.Fatalf("final queued output was dropped: %q", got)
	}
}

func TestInactiveChildExitKeepsFocusAndRecordsNotice(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild([]byte("pair screen"))
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.AttachTree("c2", "/w/pair", "pair", other)
	f.con.SetForget(func(couchcore.Worktree, couchcore.ActorID) error { return nil })
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.con.Switch("c2")
	waitFor(t, "pair to become active", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.focus == FocusActor("c2")
	})
	f.host.Reset()

	f.child.Exit(19)
	waitFor(t, "the inactive exit notice", func() bool {
		return strings.Contains(f.host.Written(), "brain") && strings.Contains(f.host.Written(), "19")
	})
	f.con.mu.Lock()
	focus := f.con.focus
	_, deadPanePresent := f.con.panes["c1"]
	f.con.mu.Unlock()
	if focus != FocusActor("c2") {
		t.Fatalf("inactive exit stole focus: got %+v, want c2", focus)
	}
	if deadPanePresent {
		t.Fatal("inactive dead pane remained attached")
	}
}

// The whole reserved-row design in one assertion: the child is sized one row
// short of the host, never the full height.
func TestConsoleSizesTheChildOneRowShort(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the initial resize", func() bool { return len(f.child.Resizes()) > 0 })

	got := f.child.Resizes()[0]
	if got.Rows != 23 || got.Cols != 80 {
		t.Fatalf("child sized %+v, want 23x80 on a 24-row host", got)
	}
}

// The SIGWINCH path, covered by a test rather than only by the operator smoke.
func TestConsolePropagatesAHostResizeToTheChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the initial resize", func() bool { return len(f.child.Resizes()) > 0 })

	f.host.SetSize(ptychild.Size{Rows: 40, Cols: 100})
	waitFor(t, "the child to be resized", func() bool {
		rs := f.child.Resizes()
		return len(rs) > 1 && rs[len(rs)-1].Rows == 39 && rs[len(rs)-1].Cols == 100
	})
}

func TestConsoleForwardsTypingToTheActiveChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte("hello"))

	waitFor(t, "the child to receive typing", func() bool {
		for _, w := range f.child.Writes() {
			if strings.Contains(string(w), "hello") {
				return true
			}
		}
		return false
	})
}

// The hotkey is couch's, and the child must never see it -- otherwise every
// trip home also types a NUL into the agent.
func TestConsoleDoesNotForwardTheHotkeyToTheChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte("a\x00b"))

	waitFor(t, "the prefix to reach the child and suffix to reach the panel", func() bool {
		var all string
		for _, w := range f.child.Writes() {
			all += string(w)
		}
		return strings.Contains(all, "a") && strings.Contains(f.host.Written(), "filter: b")
	})
	for _, w := range f.child.Writes() {
		if strings.ContainsRune(string(w), 0x00) {
			t.Fatalf("the hotkey reached the child: %q", w)
		}
		if strings.Contains(string(w), "b") {
			t.Fatalf("the post-hotkey suffix reached the old child: %q", w)
		}
	}
}

// A read boundary is not an event boundary. The suffix after a hotkey belongs
// to the focus reached by that hotkey even when stdin returns both in one read.
func TestConsoleAppliesHotkeyBeforeRoutingSameReadSuffix(t *testing.T) {
	for _, hotkey := range []string{"\x00", "\x1b[32;5u"} {
		t.Run(fmt.Sprintf("%q", hotkey), func(t *testing.T) {
			f := newFixture(t, 24, 80)
			waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
			before := len(f.child.Writes())

			_, _ = f.stdin.Write([]byte(hotkey + "pair"))
			waitFor(t, "the suffix to reach the panel", func() bool {
				return strings.Contains(f.host.Written(), "filter: pair")
			})
			for _, w := range f.child.Writes()[before:] {
				if strings.Contains(string(w), "pair") {
					t.Fatalf("post-hotkey suffix reached the old child: %q", w)
				}
			}
		})
	}
}

func TestConsoleRecognisesKittyHotkeySplitImmediatelyAfterEscape(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	before := len(f.child.Writes())

	_, _ = f.stdin.Write([]byte("\x1b"))
	_, _ = f.stdin.Write([]byte("[32;5upair"))
	waitFor(t, "the split hotkey to open the panel and route its suffix", func() bool {
		return strings.Contains(f.host.Written(), "filter: pair")
	})
	for _, w := range f.child.Writes()[before:] {
		if strings.Contains(string(w), "\x1b") || strings.Contains(string(w), "pair") {
			t.Fatalf("split hotkey or suffix reached the old child: %q", w)
		}
	}
}

func TestConsoleFlushesALoneEscapeToTheActiveChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	before := len(f.child.Writes())

	_, _ = f.stdin.Write([]byte("\x1b"))
	waitFor(t, "the ambiguity window to flush Escape", func() bool {
		for _, w := range f.child.Writes()[before:] {
			if strings.Contains(string(w), "\x1b") {
				return true
			}
		}
		return false
	})
}

// Child output reaches the host only through the console, so the operator sees
// what the active child writes.
func TestConsoleWritesActiveChildOutputToTheHost(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.child.Feed([]byte("child says hi"))

	waitFor(t, "the output to reach the host", func() bool {
		return strings.Contains(f.host.Written(), "child says hi")
	})
}

// A child that resets margins (nvim on exit) drops the reservation; the console
// must put it back, or the row is silently overwritten and never returns.
func TestConsoleReassertsTheRegionWhenAChildDropsIt(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the initial reserve", func() bool {
		return strings.Contains(f.host.Written(), "\x1b[1;23r")
	})
	f.host.Reset()

	f.child.Feed([]byte("\x1b[r")) // DECSTBM reset, as nvim emits on exit
	waitFor(t, "the region to be re-asserted", func() bool {
		return strings.Contains(f.host.Written(), "\x1b[1;23r")
	})
}

// Restoration on the CHILD-EXIT path.
func TestConsoleRestoresTheTerminalWhenTheChildExits(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.child.Exit(3)
	select {
	case code := <-f.done:
		if code != 3 {
			t.Fatalf("Run() = %d, want the child's code 3", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not return after the child exited")
	}
	assertConsoleRestored(t, f)
}

// Restoration on the MID-STREAM teardown path. A restore that only runs on the
// happy path leaves the operator's terminal with a pinned scrolling region.
func TestConsoleRestoresTheTerminalOnTeardownMidStream(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not return after Stop")
	}
	assertConsoleRestored(t, f)
}

func TestConsoleRevokesChildMouseModeBeforeReturningToShell(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.child.Feed([]byte("\x1b[?1003;1006h"))
	waitFor(t, "child mouse mode to reach host", func() bool {
		return strings.Contains(f.host.Written(), "\x1b[?1003;1006h")
	})

	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not return after Stop")
	}
	written := f.host.Written()
	enabled := strings.Index(written, "\x1b[?1003;1006h")
	disabled := strings.LastIndex(written, hostty.ResetInteractiveModes)
	if enabled < 0 || disabled <= enabled {
		t.Fatalf("mouse reset did not follow child enable: %q", written)
	}
	assertConsoleRestored(t, f)
}

func TestConsoleRestoresTheTerminalOnTerminationSignal(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			f := newFixture(t, 24, 80)
			waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
			f.host.Reset()

			f.host.Terminate(sig)
			select {
			case code := <-f.done:
				if code != 0 {
					t.Fatalf("Run() = %d after %v, want 0", code, sig)
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("Run() did not return after %v", sig)
			}
			assertConsoleRestored(t, f)
		})
	}
}

func assertConsoleRestored(t *testing.T, f *consoleFixture) {
	t.Helper()
	written := f.host.Written()
	for name, want := range map[string]string{
		"scroll region reset":   hostty.ResetRegion,
		"saved cursor restore":  hostty.RestoreCursor,
		"alternate-screen exit": hostty.LeaveAltScreen,
		"cursor visibility":     hostty.ShowCursor,
		"interactive modes off": hostty.ResetInteractiveModes,
	} {
		if !strings.Contains(written, want) {
			t.Errorf("missing %s %q in teardown %q", name, want, written)
		}
	}
	if f.host.RawDepth() != 0 {
		t.Errorf("raw mode left on: RawDepth = %d", f.host.RawDepth())
	}
	if !f.host.Closed() {
		t.Error("host signal/resize watchers were not closed")
	}
}

// A pty read boundary falls wherever the kernel puts it -- including inside one
// of the child's escape sequences. The console must never write its own bytes
// into that gap.
//
// Found by running a REAL nvim under the console: the emitted stream contained
//
//	\x1b7\x1b[12;1H\x1b[2K[brain]\x1b8;82;88m
//
// -- a status-row paint spliced into the middle of nvim's `\x1b[38;2;76;82;88m`,
// which corrupts the child's colours AND loses the row.
//
// THE ORDERING IS THE TEST. An earlier version fed chunk 1, waited for the
// console to process it, then fed chunk 2 -- which made the bug unreproducible,
// because the console's view was momentarily in step with the child's. The M2
// boundary review called that "avoiding the window rather than covering it",
// and it was right: production's window is exactly the case where the child has
// ALREADY read more while the console is still writing an earlier chunk. This
// version reproduces that by completing the sequence at the child before the
// console has drained the first chunk.
func TestConsoleNeverInjectsInsideAChildEscapeSequence(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	// Both chunks reach the child (and its Screen) back to back. The console
	// drains them afterwards, so when it writes chunk 1 the child's own state
	// already reflects chunk 2 -- the exact skew that made asking the child
	// unsound.
	f.child.Feed([]byte("\x1b[2J\x1b[38;2;76"))
	f.child.Feed([]byte(";82;88mCOLOURED"))

	waitFor(t, "the child's output to reach the host", func() bool {
		return strings.Contains(f.host.Written(), "COLOURED")
	})
	if got := f.host.Written(); !strings.Contains(got, "\x1b[38;2;76;82;88m") {
		t.Fatalf("the child's escape sequence was split by an injected paint: %q", got)
	}
}

// The same hazard during an OVER-LONG sequence, which the first fix missed for
// a second reason: Pending() reads 0 while such a sequence is being skipped
// rather than held, so a check built on it reported "safe" mid-sequence.
func TestConsoleNeverInjectsInsideAnOverLongSequence(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	huge := strings.Repeat("A", 70*1024)
	f.child.Feed([]byte("\x1b[2J\x1b]52;c;" + huge))
	f.child.Feed([]byte("\x07DONE"))

	waitFor(t, "the child's output to reach the host", func() bool {
		return strings.Contains(f.host.Written(), "DONE")
	})
	got := f.host.Written()
	body := strings.Index(got, "\x1b]52;c;")
	term := strings.Index(got, "\x07")
	if body < 0 || term < 0 {
		t.Fatalf("the OSC did not reach the host intact: %q", trimForLog(got))
	}
	if paint := strings.Index(got[body:term], "\x1b7"); paint >= 0 {
		t.Fatalf("a paint was injected inside an over-long sequence at +%d", paint)
	}
}

func trimForLog(s string) string {
	if len(s) > 300 {
		return s[:150] + "…" + s[len(s)-150:]
	}
	return s
}

// The deferred paint must still HAPPEN once the stream is safe again -- a
// console that avoids corrupting the child by never painting has traded one bug
// for another.
func TestConsoleRepaintsOnceTheChildStreamIsSafeAgain(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.child.Feed([]byte("\x1b[2J\x1b[38;2;76"))
	f.child.Feed([]byte(";82;88mdone"))

	waitFor(t, "the row to be repainted after the sequence completed", func() bool {
		return strings.Contains(f.host.Written(), "\x1b[24;1H")
	})
}

// The row must say WHICH actor wants attention -- that is Decision 8's whole
// justification for spending a permanent terminal row before #147's transport
// exists. StatusActor.Bell shipped with no writer at M2's boundary (BR-27), so
// the row could never have said it.
func TestConsoleMarksAnInactiveActorThatRangTheBell(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.AttachTree("c2", "/w/ariadne", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	other.Feed([]byte("\x07"))

	waitFor(t, "the row to mark the actor", func() bool {
		return strings.Contains(f.host.Written(), "ariadne*")
	})
	if strings.Contains(f.host.Written(), "[ariadne]") {
		t.Fatal("the inactive actor was marked active")
	}
}

// A bell from the actor the operator is already looking at is not a page.
func TestConsoleDoesNotMarkTheActiveActorOnItsOwnBell(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.child.Feed([]byte("\x07done"))
	waitFor(t, "the output to reach the host", func() bool {
		return strings.Contains(f.host.Written(), "done")
	})
	if strings.Contains(f.host.Written(), "brain*") {
		t.Fatalf("the active actor was flagged as wanting attention: %q", f.host.Written())
	}
}

// Child output must not be silently dropped when the console is slow. Nothing
// repaints from the ring at this milestone, so a dropped chunk is output the
// operator never sees (BR-29).
func TestConsoleDoesNotDropChildOutputUnderBurst(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	const n = 2000 // well past the channel's buffer
	for i := 0; i < n; i++ {
		f.child.Feed([]byte(fmt.Sprintf("line-%04d\r\n", i)))
	}
	waitFor(t, "the last line to reach the host", func() bool {
		return strings.Contains(f.host.Written(), fmt.Sprintf("line-%04d", n-1))
	})

	got := f.host.Written()
	for _, i := range []int{0, 1, n / 2, n - 2, n - 1} {
		if !strings.Contains(got, fmt.Sprintf("line-%04d", i)) {
			t.Fatalf("line-%04d was dropped from the live path", i)
		}
	}
}

// The CLASS behind BR-21: exactly one goroutine may write to the host, so there
// is no path to the screen that bypasses the mid-sequence check.
//
// The first fix framed the console's own output but left applyLayout (SIGWINCH)
// and the hotkey path writing from other goroutines, so both could still splice
// into the child's stream. This drives all three concurrently against a child
// that is parked mid-sequence.
func TestConsoleNeverSplicesFromAnyPath(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	// Park the stream mid-sequence, then hammer the INTERLEAVING writers.
	//
	// Resizes are the case: they paint the row into a stream that continues.
	// The hotkey is deliberately not hammered here -- since M3 it opens the
	// panel, which is a screen TAKEOVER rather than an interleaved paint, and a
	// takeover legitimately ends the child's stream's claim on the screen.
	// TestPanelIsNotPaintedOverByABackgroundChild covers that path instead.
	f.child.Feed([]byte("\x1b[2J\x1b[38;2;76"))
	for i := 0; i < 20; i++ {
		f.host.SetSize(ptychild.Size{Rows: uint16(24 + i%3), Cols: 80})
	}
	f.child.Feed([]byte(";82;88mCOLOURED"))

	waitFor(t, "the child's output to reach the host", func() bool {
		return strings.Contains(f.host.Written(), "COLOURED")
	})
	if got := f.host.Written(); !strings.Contains(got, "\x1b[38;2;76;82;88m") {
		t.Fatalf("a writer other than the child's chunk path spliced the sequence: %q", got)
	}
}

// A console that cannot take the terminal must SAY so. Returning a bare 1 was
// the other half of BR-23 -- the operator saw an exit code and nothing else.
func TestConsoleReportsWhyItCannotTakeTheTerminal(t *testing.T) {
	host := &refusingHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})}
	var errw bytes.Buffer
	con := New(host, strings.NewReader(""))
	con.SetErrorWriter(&errw)

	if code := con.Run(); code != 1 {
		t.Fatalf("Run() = %d, want 1", code)
	}
	if !strings.Contains(errw.String(), "cannot take the terminal") {
		t.Fatalf("nothing explained the failure: %q", errw.String())
	}
}

type refusingHost struct{ *hostty.FakeHost }

func (h *refusingHost) MakeRaw() (func() error, error) {
	return nil, errors.New("inappropriate ioctl for device")
}

// An INACTIVE pane's row damage is real; it just cannot be acted on yet. The
// first version consumed the child's latch for every pane and acted on it only
// for the active one, so a background child's erase was thrown away and
// attaching to it would land on a screen with no status row.
func TestConsoleKeepsAnInactivePanesRowDamage(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	// The inactive child clears its screen, then goes quiet.
	other.Feed([]byte("\x1b[2Jbackground work"))

	// Poll: Feed is synchronous but the CONSOLE processes asynchronously, so a
	// one-shot check here races the loop -- and would have read the latch
	// before it was ever set.
	waitFor(t, "the inactive pane to record its row damage", func() bool {
		return f.con.PaneRowDirty("c2")
	})
	if f.con.PaneRowDirty("c1") {
		t.Fatal("the active pane kept damage it had already repaired")
	}
}

// A switcher that loses what was said while you were away is not a switcher.
// An inactive child's output must reach its ring even though it does not reach
// the screen.
func TestConsoleKeepsInactiveChildOutputOffScreenButInItsRing(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	other.Feed([]byte("background progress"))
	// Order behind a marker from the ACTIVE child. The chunk channel is FIFO,
	// so seeing this on the host proves the console has already drained past
	// the inactive child's chunk.
	//
	// Polling the inactive child's own Snapshot does not work: Feed is
	// synchronous, so it is true before the console has looked at anything, and
	// the assertion below then runs too early. That produced a test which
	// passed with the isActive guard DELETED -- caught by the deletion check
	// not firing, which is the fourth time this shape has appeared here.
	f.child.Feed([]byte("MARKER-FROM-ACTIVE"))
	waitFor(t, "the console to drain past both chunks", func() bool {
		return strings.Contains(f.host.Written(), "MARKER-FROM-ACTIVE")
	})

	if strings.Contains(f.host.Written(), "background progress") {
		t.Fatal("an inactive child's output reached the screen")
	}
	if !strings.Contains(string(other.Snapshot()), "background progress") {
		t.Fatal("an inactive child's output was lost instead of buffered")
	}
	if other.Done() {
		t.Fatal("switching away stopped the inactive child instead of leaving it warm")
	}
}

// Landing on a child must repaint it from its ring -- otherwise switching lands
// on a blank screen and the operator has to press a key to see where they are.
func TestConsoleReplaysOnAttach(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild([]byte("earlier output from ariadne"))
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.con.Switch("c2")
	waitFor(t, "the replay to reach the host", func() bool {
		return strings.Contains(f.host.Written(), "earlier output from ariadne")
	})
	if !strings.Contains(f.host.Written(), hostty.HomeAndClear) {
		t.Fatal("the replay did not clear first; it would land on top of the previous child's screen")
	}
}

// #127 arriving at a new site: a raw replay re-ASKS the host terminal for its
// capabilities, and the answer arrives as the newly active child's input.
func TestConsoleStripsQueriesFromTheReplay(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild([]byte("prompt \x1b[c\x1b[?1006h done"))
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.con.Switch("c2")
	waitFor(t, "the replay", func() bool {
		return strings.Contains(f.host.Written(), "done")
	})
	got := f.host.Written()
	if strings.Contains(got, "\x1b[c") {
		t.Fatalf("the replay re-asked the host terminal: %q", got)
	}
	if !strings.Contains(got, "\x1b[?1006h") {
		t.Fatal("the replay dropped a legitimate DECSET — mouse mode would be lost on every switch")
	}
}

// The status row must be repainted AFTER the child's screen, or the landing
// paints over it.
func TestConsoleRepaintsTheRowAfterTheReplay(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild([]byte("ariadne screen"))
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.con.Switch("c2")
	waitFor(t, "the row", func() bool { return strings.Contains(f.host.Written(), "[ariadne]") })

	got := f.host.Written()
	if strings.LastIndex(got, "ariadne screen") > strings.LastIndex(got, "[ariadne]") {
		t.Fatal("the child's replay landed after the row and painted over it")
	}
}

// Switching to an actor the console does not host must not blank the screen.
func TestConsoleIgnoresASwitchToAnUnknownActor(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	f.con.Switch("nope")
	f.child.Feed([]byte("still here"))
	waitFor(t, "the active child to keep the screen", func() bool {
		return strings.Contains(f.host.Written(), "still here")
	})
}

// With the panel up, nobody is looking at the child -- so a child that keeps
// streaming must not paint over couch's own screen.
func TestPanelIsNotPaintedOverByABackgroundChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	// ctrl-space from the root actor opens the panel.
	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel to open", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
	})
	f.host.Reset()

	f.child.Feed([]byte("still streaming in the background"))
	// Order behind a marker the CONSOLE sets, not one the child sets.
	//
	// With the panel up nothing reaches the host, so a host marker is
	// unavailable -- and polling the child's own Snapshot is satisfied
	// synchronously by Feed, before the console has looked at anything. An
	// erase sets the pane's row-dirty latch when the console drains it, and the
	// chunk channel is FIFO, so seeing it proves both chunks were processed.
	f.child.Feed([]byte("\x1b[2J"))
	waitFor(t, "the console to drain both chunks", func() bool {
		return f.con.PaneRowDirty("c1")
	})

	if strings.Contains(f.host.Written(), "still streaming") {
		t.Fatal("a background child painted over the panel")
	}
}

// ctrl-space from the root actor reaches the panel, and the panel lists the
// actors -- including a parked one.
func TestHotkeyFromTheRootActorOpensThePanel(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
	})
	got := f.host.Written()
	for _, want := range []string{"▸ [live] brain", "  [live] ariadne"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the panel does not list %q: %q", want, got)
		}
	}
}

func TestConsolePanelRefreshUsesInjectedSummaries(t *testing.T) {
	f := newFixture(t, 24, 80)
	name := "parked"
	f.con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
		return []couchcore.ThreadSummary{
			{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
			{Address: panelAddress("pair"), WorkingPath: "/w/pair", Name: name, PublishedSummary: "waiting for review"},
		}, nil
	})
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the parked summary", func() bool {
		return strings.Contains(f.host.Written(), "parked") &&
			strings.Contains(f.host.Written(), "waiting for review")
	})

	name = "renamed"
	f.con.rebuildPanel()
	f.con.mu.Lock()
	rows := append([]PanelRow(nil), f.con.panel.Rows()...)
	f.con.mu.Unlock()
	if len(rows) != 2 || rows[1].Label != "renamed" {
		t.Fatalf("refreshed rows = %+v, want renamed summary", rows)
	}
}

func TestConsolePanelSurfacesDurableReadFailures(t *testing.T) {
	t.Run("inventory", func(t *testing.T) {
		f := newFixture(t, 24, 80)
		f.con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
			return nil, errors.New("corrupt manifest")
		})
		waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
		_, _ = f.stdin.Write([]byte("\x00"))
		waitFor(t, "the durable inventory error", func() bool {
			return strings.Contains(f.host.Written(), "error: thread inventory unavailable: corrupt manifest")
		})
	})

	t.Run("reference", func(t *testing.T) {
		f := newFixture(t, 24, 80)
		f.con.SetResolver(func(string) ([]couchcore.ThreadAddress, error) {
			return nil, errors.New("missing addressed record")
		})
		waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
		_, _ = f.stdin.Write([]byte("\x00"))
		waitFor(t, "the panel", func() bool { return strings.Contains(f.host.Written(), "couch — actors") })
		_, _ = f.stdin.Write([]byte("x"))
		waitFor(t, "the durable reference error", func() bool {
			return strings.Contains(f.host.Written(), "error: thread reference unavailable: missing addressed record")
		})
	})
}

// The property the whole project rests on: from a NON-root child, one key goes
// home to the root actor -- not to the panel.
func TestHotkeyFromANonRootChildGoesHome(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild([]byte("ariadne screen"))
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	f.con.Switch("c2")
	waitFor(t, "the switch", func() bool { return strings.Contains(f.host.Written(), "[ariadne]") })
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "to land back on the root actor", func() bool {
		return strings.Contains(f.host.Written(), "[brain]")
	})
	if strings.Contains(f.host.Written(), "couch — actors") {
		t.Fatal("ctrl-space from a non-root child opened the panel instead of going home")
	}
}

func TestPanelPrintableCommandRunesAreTypeahead(t *testing.T) {
	for _, query := range []string{"start", "xray", "name", "describe", "2fa", ":", "1", "9"} {
		t.Run(query, func(t *testing.T) {
			f := newFixture(t, 24, 80)
			f.con.SetResolver(func(string) ([]couchcore.ThreadAddress, error) { return nil, nil })
			waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
			_, _ = f.stdin.Write([]byte("\x00"))
			waitFor(t, "the panel", func() bool { return strings.Contains(f.host.Written(), "couch — actors") })
			_, _ = f.stdin.Write([]byte(query))
			waitFor(t, "the complete query", func() bool {
				return strings.Contains(f.host.Written(), "filter: "+query)
			})
		})
	}
}

// Typeahead filters through the INJECTED resolver, so the panel finds a child
// by whatever couchcore matches on -- including an agent-published description.
func TestPanelTypeaheadUsesTheInjectedResolver(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.AttachTree("c2", "/w/ariadne", "ariadne", other)

	asked := ""
	f.con.SetResolver(func(q string) ([]couchcore.ThreadAddress, error) {
		asked = q
		// Production resolves human text to the child's WORKTREE, not to its
		// per-incarnation actor id. The panel must retain both identities:
		// worktree for matching, actor id for switching.
		return []couchcore.ThreadAddress{panelAddress("c2")}, nil
	})
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
	})
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("ari"))
	waitFor(t, "the filter to narrow", func() bool {
		return strings.Contains(f.host.Written(), "filter: ari")
	})
	if asked != "ari" {
		t.Fatalf("the resolver was asked %q, want the typed query", asked)
	}
	// And Enter takes the single filtered row.
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "Enter to switch", func() bool {
		return strings.Contains(f.host.Written(), "[ariadne]")
	})
}

// Keys typed at the panel must not reach the child behind it.
func TestPanelKeysDoNotReachTheChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
	})
	before := len(f.child.Writes())

	_, _ = f.stdin.Write([]byte("typing at the panel"))
	waitFor(t, "the query to render", func() bool {
		return strings.Contains(f.host.Written(), "filter: typing")
	})
	if len(f.child.Writes()) != before {
		t.Fatalf("keys aimed at the panel reached the child: %q", f.child.Writes()[before:])
	}
}

// The bug the operator hit: a mouse move over the panel typed `[<;0;M[<;;M...`
// into the filter, which matched nothing, showed "(no match)", and left no way
// back because Escape did nothing either.
func TestPanelIgnoresMouseReports(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
	})
	f.host.Reset()

	// A burst of SGR mouse reports, as a mouse move produces.
	_, _ = f.stdin.Write([]byte("\x1b[<0;12;4M\x1b[<0;13;4M\x1b[<0;14;5m"))
	// Order behind a real keystroke: FIFO on the same path.
	_, _ = f.stdin.Write([]byte("z"))
	waitFor(t, "the real keystroke to land", func() bool {
		return strings.Contains(f.host.Written(), "filter: z")
	})

	if strings.Contains(f.host.Written(), "filter: [<") {
		t.Fatalf("mouse bytes were typed into the filter: %q", f.host.Written())
	}
}

// Escape must back out: clear the filter if there is one, otherwise return to
// the actor. A panel with no way back is a trap.
func TestPanelEscapeClearsThenReturns(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
	})

	_, _ = f.stdin.Write([]byte("zz"))
	waitFor(t, "the filter", func() bool {
		return strings.Contains(f.host.Written(), "filter: zz")
	})
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\x1b")) // first Escape clears the filter
	waitFor(t, "the filter to clear", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.query == ""
	})
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\x1b")) // second Escape leaves the panel
	waitFor(t, "to return to the actor", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return !f.con.focus.IsPanel()
	})
}

// Panel detach uses the same forced replay path as a child-to-child attach.
// If returning merely calls onSwitch, the active id is unchanged and the
// replay is skipped, stranding the operator on couch's panel screen.
func TestReturningFromPanelReplaysWhatTheWarmChildAccumulated(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
	})
	f.host.Reset()

	f.child.Feed([]byte("progress while detached at the panel"))
	if f.child.Done() {
		t.Fatal("opening the panel stopped the hosted child")
	}
	_, _ = f.stdin.Write([]byte("\x1b"))
	waitFor(t, "the warm child's accumulated output to replay", func() bool {
		return strings.Contains(f.host.Written(), "progress while detached at the panel")
	})
	if !strings.Contains(f.host.Written(), hostty.HomeAndClear) {
		t.Fatal("panel return bypassed the shared clear-and-replay attach path")
	}
}

// Arrows move the highlight, and Enter takes the highlighted row -- the panel
// has to be navigable, not just filterable.
func TestPanelArrowsMoveTheSelection(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild([]byte("ariadne screen"))
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		return strings.Contains(f.host.Written(), "▸ [live] brain")
	})
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\x1b[B")) // down
	waitFor(t, "the highlight to move", func() bool {
		return strings.Contains(f.host.Written(), "▸ [live] ariadne")
	})
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "Enter to switch to the highlighted actor", func() bool {
		return strings.Contains(f.host.Written(), "[ariadne]")
	})
}

// The panel shows WHICH actor wants attention -- the reason it is a place to
// look rather than a list.
func TestPanelShowsTheBellMarker(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.AttachTree("c2", "/w/ariadne", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	other.Feed([]byte("\x07"))
	waitFor(t, "the bell to register", func() bool {
		return strings.Contains(f.host.Written(), "ariadne*")
	})

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel to mark it", func() bool {
		return strings.Contains(f.host.Written(), "[live] ariadne *")
	})
}

// The operator's report: Escape in the panel did nothing. Under the Kitty
// keyboard protocol -- which zellij enables, so it is what a real session
// leaves the terminal in -- Escape arrives as `\x1b[27u`.
func TestPanelEscapeWorksInBothEncodings(t *testing.T) {
	for _, esc := range []string{"\x1b", "\x1b[27u", "\x1b[27;1u"} {
		t.Run(fmt.Sprintf("%q", esc), func(t *testing.T) {
			f := newFixture(t, 24, 80)
			waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
			_, _ = f.stdin.Write([]byte("\x00"))
			waitFor(t, "the panel", func() bool {
				return strings.Contains(f.host.Written(), "couch — actors")
			})
			f.host.Reset()

			_, _ = f.stdin.Write([]byte(esc))
			waitFor(t, "focus to return to the actor", func() bool {
				f.con.mu.Lock()
				defer f.con.mu.Unlock()
				return !f.con.focus.IsPanel()
			})
			waitFor(t, "the actor repaint", func() bool {
				got := f.host.Written()
				return strings.Contains(got, "[brain]") &&
					strings.LastIndex(got, "[brain]") > strings.LastIndex(got, "couch — actors")
			})
		})
	}
}

// Same for the keys that move and commit.
func TestPanelNavigationWorksInBothEncodings(t *testing.T) {
	for _, keys := range []struct{ down, enter string }{
		{"\x1b[B", "\r"},
		{"\x1b[1;1B", "\x1b[13u"},
	} {
		t.Run(fmt.Sprintf("%q", keys.down), func(t *testing.T) {
			f := newFixture(t, 24, 80)
			other := ptychild.NewFakeChild([]byte("ariadne screen"))
			other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
			f.con.Attach("c2", "ariadne", other)
			waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

			_, _ = f.stdin.Write([]byte("\x00"))
			waitFor(t, "the panel", func() bool {
				return strings.Contains(f.host.Written(), "▸ [live] brain")
			})
			_, _ = f.stdin.Write([]byte(keys.down))
			waitFor(t, "the highlight to move", func() bool {
				return strings.Contains(f.host.Written(), "▸ [live] ariadne")
			})
			_, _ = f.stdin.Write([]byte(keys.enter))
			waitFor(t, "Enter to switch", func() bool {
				return strings.Contains(f.host.Written(), "[ariadne]")
			})
		})
	}
}
