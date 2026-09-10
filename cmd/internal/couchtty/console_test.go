package couchtty

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
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
	child.SetSink(func(batch ptychild.OutputBatch) { con.Deliver("c1", batch) })
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

func TestWriteChildDoesNotRetainFramingOnlyOutput(t *testing.T) {
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	con := New(host, bytes.NewReader(nil))
	chunk := bytes.Repeat([]byte("ordinary child output"), 1024)

	for range 100 {
		con.writeChild(chunk)
	}

	if parts := con.hostScan.TakeOutputParts(); len(parts) != 0 {
		t.Fatalf("framing-only scanner retained %d output parts", len(parts))
	}
}

func TestConsoleSwitchOperationUsesExactThreadAndRefusesStaleTarget(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.attachThreadActor("c2", "c2", menuAddress("c2"), "c1", "brain", other)

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
		Args: map[string]string{"repo-scope": menuAddress("c2").RepoScope, "tag": string(menuAddress("c2").Tag)},
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
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

	f.con.Deliver("c1", ptychild.OutputBatch{Raw: []byte("final output")})
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.AttachTree("c2", "/w/ariadne", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	other.Feed([]byte("\x07"))

	waitFor(t, "the row to mark the actor", func() bool {
		return strings.Contains(f.host.Written(), "\x1b[38;5;220mariadne\x1b[0m")
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
	// TestOpeningThePanelBlanksTheChildsScreenDeliberately covers that path.
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
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
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
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

// ctrl-space means the switcher, from ANY actor -- #170 deleted the child ->
// root-actor -> panel ladder, so there is no longer a first actor that one key
// goes "home" to. The predecessor of this test asserted the opposite; it is
// replaced rather than deleted, because the key still has to do something
// definite from a non-first child and that is the interesting case.
func TestHotkeyFromAnyActorOpensTheSwitcher(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild([]byte("ariadne screen"))
	other.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.Attach("c2", "ariadne", other)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	f.con.Switch("c2")
	waitFor(t, "the switch", func() bool { return strings.Contains(f.host.Written(), "[ariadne]") })
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the switcher to open", func() bool {
		return strings.Contains(f.host.Written(), "threads")
	})
	if strings.Contains(f.host.Written(), "[brain]") {
		t.Fatal("ctrl-space walked to another actor instead of opening the switcher")
	}
}

// LeaveResult exists so the operator learns what leaving did, especially about
// threads Couch deliberately did NOT park because it could not prove their
// state. A field written at zero read sites teaches nobody anything.
func TestConsoleReportsWhatLeaveDid(t *testing.T) {
	f := newFixture(t, 24, 80)
	var out bytes.Buffer
	f.con.stderr = &out

	f.con.reportLeave(couchcore.LeaveResult{
		Detached: []couchcore.ThreadAddress{{RepoScope: "s", Tag: "couch-one"}},
		Parked:   []couchcore.ThreadAddress{{RepoScope: "s", Tag: "couch-two"}},
		Skipped:  []couchcore.ThreadAddress{{RepoScope: "s", Tag: "couch-three"}},
	})

	got := out.String()
	for _, want := range []string{"detached 1", "parked 1", "couch-three", "occupied"} {
		if !strings.Contains(got, want) {
			t.Fatalf("leave report = %q, want it to mention %q", got, want)
		}
	}
}

// BR-78: the drain loop's condition must MATCH the entry guard's.
//
// With the entry guard stricter (SafeToPaint) than the loop (MidSequence), a
// chunk that takes the cursor slot mid-drain leaves onChunk re-deferring the
// notification straight back onto the queue this loop pops from -- and the
// loop, still asking the looser question, spins forever.
//
// This one pins the easy half: with the child ALREADY holding the save, the
// drain returns. It does not reproduce the spin -- the entry guard
// short-circuits, so the loop is never entered -- and for two rounds a comment
// here claimed the spin was unstageable, which was measurably false. The spin
// needs the gate SAFE at entry and closed DURING the drain, and the drain closes
// it itself: see the test below.
func TestTheNotificationDrainTerminatesWhenTheChildHoldsTheCursorSave(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), strings.NewReader(""))

	// A save held, stream at a boundary: SafeToPaint false, MidSequence FALSE.
	// That gap is exactly where the loop and its entry guard disagreed.
	c.mu.Lock()
	c.hostScan.FeedFraming([]byte("\x1b7"))
	safe, mid := c.hostScan.SafeToPaint(), c.hostScan.MidSequence()
	// A chunk carrying a NOTIFICATION part. An empty chunk re-defers nothing,
	// so the loop drains it and returns even with the bug present -- the first
	// version of this test passed against the reverted fix for exactly that
	// reason, which is the same "passes for the wrong reason" trap that has
	// caught several tests in this issue. onChunk only re-defers when it has a
	// notification to defer.
	c.deferredNotifications = append(c.deferredNotifications, chunk{
		id: "probe",
		batch: ptychild.OutputBatch{
			Parts: []ptychild.OutputPart{{Notification: &ptychild.NotificationObservation{}}},
		},
	})
	c.mu.Unlock()
	if safe || mid {
		t.Fatalf("setup does not reproduce the gap: SafeToPaint=%v MidSequence=%v; "+
			"the spin needed unsafe-but-not-mid-sequence", safe, mid)
	}

	done := make(chan struct{})
	go func() { c.flushDeferredNotifications(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("flushDeferredNotifications did not return; the drain loop is asking " +
			"a looser question than its entry guard and spins on a re-deferred chunk")
	}
}

// couch defers its OWN paint while the child holds the cursor save (BR-77).
//
// The Critical was that couch asked a narrower question than `pair term` while
// running the identical primitive. That was fixed and then pinned only by a
// source-text guard -- but the behaviour is one call away, and a source guard
// cannot show that the deferral actually happens.
func TestCouchDefersItsOwnPaintWhileTheChildHoldsTheCursorSave(t *testing.T) {
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	c := New(host, strings.NewReader(""))

	c.writeChild([]byte("\x1b7")) // zsh's right-prompt save, outside the alt screen
	host.Reset()
	c.writeOwn("STRIP")

	if got := host.Written(); strings.Contains(got, "STRIP") {
		t.Fatalf("couch painted inside the child's save/restore pair; wrote %q", got)
	}
	c.mu.Lock()
	pending := c.paintPending
	c.mu.Unlock()
	if !pending {
		t.Fatal("the paint was dropped rather than owed; the debt must be paid at the " +
			"next safe boundary")
	}

	// And the debt is payable once the child restores.
	c.writeChild([]byte("\x1b8"))
	c.writeOwn("STRIP")
	if got := host.Written(); !strings.Contains(got, "STRIP") {
		t.Fatalf("the gate never reopened after the child's restore; wrote %q", got)
	}
}

// A takeover must leave the scanner knowing what the NEW child's screen says
// (BR-82).
//
// The reset is right -- the old child's partial sequence is gone from the screen
// -- but resetting alone means the scanner believes the PRIMARY screen after
// every switch. SafeToPaint's alt-screen carve-out then cannot apply, so a
// full-screen child taking the cursor-save slot freezes the strip for its whole
// session: BR-79's symptom reached through the switch path instead of the
// startup one.
func TestATakeoverRelearnsTheChildsModesFromTheBodyItDraws(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), strings.NewReader(""))

	// nvim's screen: it entered the alt screen, and that is what the replay says.
	c.takeOverScreen(nil, []byte("\x1b[?1049hnvim's screen\x1b[1;1H"))

	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hostScan.AltScreen() {
		t.Fatal("after a takeover the scanner does not know the child is in the alt " +
			"screen, so the paint gate's carve-out cannot apply")
	}
	// The measurement from the finding: a save taken NOW must not close the gate,
	// because a full-screen child repaints over any disturbance.
	c.hostScan.FeedFraming([]byte("\x1b[?1048h"))
	if !c.hostScan.SafeToPaint() {
		t.Fatal("the gate closed for the rest of the child's alt-screen session; " +
			"the takeover dropped the alt-screen fact that keeps it open")
	}
}

// The spin itself, deterministic and single-goroutine (BR-83).
//
// The drain closes its own gate. onChunk writes a part's BYTES through
// writeChild, which feeds them to hostScan -- so a chunk carrying `\x1b7`
// followed by a notification part takes the terminal's cursor-save slot
// mid-drain, and onChunk then re-defers the notification onto the very queue
// this loop pops from. With the loop asking the looser question (MidSequence is
// FALSE here: the stream sits on a boundary) it pops the same chunk forever.
//
// Two earlier attempts at this test passed against the reverted fix: one queued
// an empty chunk, which re-defers nothing, and one used an unregistered pane id,
// so onChunk returned at `if !known` before reaching any of this.
func TestTheDrainStopsWhenTheDrainItselfTakesTheCursorSave(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), strings.NewReader(""))

	c.mu.Lock()
	c.panes["probe"] = &pane{}
	c.active = "probe"
	// The zero Focus is the PANEL, and onChunk only writes a child's bytes when
	// the operator is actually looking at it -- without this the gate never
	// closes and the test proves nothing.
	c.focus = FocusActor("probe")
	safe, mid := c.hostScan.SafeToPaint(), c.hostScan.MidSequence()
	c.deferredNotifications = append(c.deferredNotifications, chunk{
		id:                "probe",
		focusedAtDelivery: true, // skip the attention branch; not what is under test
		batch: ptychild.OutputBatch{Parts: []ptychild.OutputPart{
			{Bytes: []byte("\x1b7")}, // DECSC: takes the shared save slot
			{Notification: &ptychild.NotificationObservation{}},
		}},
	})
	c.mu.Unlock()
	if !safe || mid {
		t.Fatalf("setup must START safe and not mid-sequence: safe=%v mid=%v", safe, mid)
	}

	done := make(chan struct{})
	go func() { c.flushDeferredNotifications(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("flushDeferredNotifications never returned: the drain loop asks a " +
			"looser question than its entry guard, so a chunk that closes the gate " +
			"mid-drain is re-deferred and re-popped forever")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hostScan.SafeToPaint() {
		t.Fatal("the drain did not close its own gate; this test no longer " +
			"reproduces the condition the spin needed")
	}
	if len(c.deferredNotifications) != 1 {
		t.Fatalf("expected the notification to be left deferred for the next safe "+
			"chunk, found %d", len(c.deferredNotifications))
	}
}

// The invariant behind the spin, checked at the source because it is not
// reachable behaviourally (see above): the drain loop and its entry guard must
// ask the SAME predicate. The bug was them disagreeing, not either value.
func TestTheNotificationDrainAndItsEntryGuardAskOneQuestion(t *testing.T) {
	raw, err := os.ReadFile("console.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	i := strings.Index(src, "func (c *Console) flushDeferredNotifications()")
	if i < 0 {
		t.Fatal("flushDeferredNotifications not found; this guard is checking nothing")
	}
	body := src[i:]
	if j := strings.Index(body[1:], "\nfunc "); j >= 0 {
		body = body[:j]
	}
	if strings.Contains(body, "hostScan.MidSequence()") {
		t.Error("the drain still gates on MidSequence somewhere; entry guard and loop " +
			"must ask SafeToPaint, or a save taken mid-drain spins the loop forever")
	}
	if n := strings.Count(body, "SafeToPaint()"); n < 2 {
		t.Errorf("expected the entry guard AND the loop to ask SafeToPaint; found %d", n)
	}
}

// The operator's reported bug is couch's, and the boundary review measured that
// couch's half of the fix was pinned by NOTHING: deleting the repaint request
// left this suite green (#209 BR-2). This is its mutation test — dropping
// `child.RequestRepaint()` from `takeOverScreen` gives resizes = [] and fails
// here with the message below. (The request moved there in C2, so it is one act
// with the takeover and no site can compose a screen without asking; a recipe
// naming the old call site sends a reader looking for code that is not there.)
//
// The nudge is what makes a switch CORRECT rather than probable: the replay is
// the immediate paint, but the retained tail is bounded, so a thread whose last
// full frame has aged out can only be repainted by the child that still holds
// it. Rows and not columns, because a column change reflows wrapped lines; and
// restored to the size the console already owns, so no new authority over child
// geometry is introduced.
func TestSwitchAsksTheIncomingChildToRepaint(t *testing.T) {
	f := newFixture(t, 24, 80)
	incoming := ptychild.NewFakeChild(nil)
	incoming.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.AttachTree("c2", "/w/pair", "pair", incoming)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	before := len(incoming.Resizes())
	f.con.switchTo("c2", false, arrivalOrdinary)

	waitFor(t, "the incoming child to be asked to repaint", func() bool {
		return len(incoming.Resizes()) >= before+2
	})
	got := incoming.Resizes()[before:]
	want := f.con.ChildSize()
	if len(got) != 2 {
		t.Fatalf("switch issued %d resizes %v, want exactly the shrink-and-restore pair", len(got), got)
	}
	if got[0].Rows != want.Rows-1 || got[0].Cols != want.Cols {
		t.Errorf("shrank to %v, want one row shorter than %v with the columns untouched", got[0], want)
	}
	if got[1] != want {
		t.Errorf("restored to %v, want the size the console already owns, %v", got[1], want)
	}
}

// A switch must not nudge the pane the operator is ALREADY on. The tracker
// ignores a landing on the current actor, and a resize round-trip on a
// full-screen child is a whole reflow — 19,317 bytes on a single-pane zellij,
// measured by cmd/probes/zellijrepaint — so paying it for a no-op switch is exactly
// the keystroke-path cost ARCH-CONSTRAINTS asks to be deliberate about.
func TestSwitchingToTheActiveThreadAsksForNoRepaint(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	before := len(f.child.Resizes())
	f.con.switchTo("c1", false, arrivalOrdinary)
	time.Sleep(20 * time.Millisecond)

	if got := f.child.Resizes(); len(got) != before {
		t.Fatalf("a switch to the already-active thread resized it: %v", got[before:])
	}
}

// couch's leg of the differential (#209 BR-9): what the console WRITES on a
// takeover is exactly what hostty.RepaintFor composes — no prefix of its own,
// no byte dropped. termcmd's leg asserts the same thing against the same
// function, which is what makes the two consoles byte-identical for the same
// child state; hostty's golden fixes what that function emits.
func TestSwitchWritesExactlyTheComposedRepaint(t *testing.T) {
	f := newFixture(t, 24, 80)
	incoming := ptychild.NewFakeChild(nil)
	incoming.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.AttachTree("c2", "/w/pair", "pair", incoming)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	incoming.Feed([]byte("\x1b[?1049hretained frame"))
	waitFor(t, "the incoming child's output to reach its ring", func() bool {
		return len(incoming.Snapshot()) > 0
	})
	f.host.Reset()

	f.con.mu.Lock()
	body := incoming.ReplayThrough(f.con.panes["c2"].replayCutoff)
	f.con.mu.Unlock()
	want := hostty.RepaintFor(incoming, body)
	if len(want) == 0 {
		t.Fatal("fixture produced nothing to compose; the assertion below would be vacuous")
	}

	f.con.switchTo("c2", false, arrivalOrdinary)
	waitFor(t, "the takeover to reach the host", func() bool {
		return strings.Contains(f.host.Written(), string(want))
	})
}

// BR-8's INTENT is byte-inert today, and saying so is better than a test that
// pretends otherwise (#209 I1).
//
// The close review measured BR-8's fix as silently revertible and asked for a
// test. I wrote one, and it stayed green under the revert — because the two
// intents differed in exactly one case, an EMPTY body, and the panel always
// renders something. That was the same disposition BR-6 got: correct by
// construction, unpinnable until the case it governs is reachable.
//
// C-1 then established that the empty-body case had no correct caller anywhere
// and deleted both the branch and the intent enum, which retires the question:
// every takeover blanks, so there is no longer a distinction here to drift.
//
// What this test DOES pin is worth keeping and is not the same claim: opening
// the panel blanks what was on the screen, so a child's frame cannot show
// through couch's own surface. Deleting the takeover fails it.
func TestOpeningThePanelBlanksTheChildsScreenDeliberately(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.child.Feed([]byte("the child's frame"))
	waitFor(t, "the child's frame to reach the host", func() bool {
		return strings.Contains(f.host.Written(), "the child's frame")
	})
	f.host.Reset()

	f.con.showMenu()

	if got := f.host.Written(); !strings.Contains(got, hostty.HomeAndClear) {
		t.Fatalf("opening the panel wrote %q, want the screen blanked first — the panel "+
			"is couch's own surface and must not have a child's frame showing through it", got)
	}
}

// And the panel asks NOBODY to repaint. The repaint request rides with the
// takeover so no site can forget it, which makes the converse worth pinning:
// there is no child behind couch's own surface, and nudging the outgoing one
// would resize a child whose screen is not on the terminal.
func TestOpeningThePanelAsksNoChildToRepaint(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	before := len(f.child.Resizes())
	f.con.showMenu()
	time.Sleep(2 * ptychild.RepaintSettle)

	if got := f.child.Resizes(); len(got) != before {
		t.Fatalf("opening the panel resized the child: %v", got[before:])
	}
}

// C2's consumer half: couch's switch is reached from TWO goroutines — the Run
// loop, and the operationQueue for the switcher's Enter and the status-chip
// click, which is the operator's primary gesture. The first version of the
// nudge took a size and documented a rule ("callers must stay on the goroutine
// that serializes their other resizes"); this path broke it immediately.
//
// The rule is gone and the mechanism replaced it: RequestRepaint takes no size
// and reads its restore under the child's own geometry lock. So the property to
// pin here is that the OPERATION path nudges exactly like the Run path — no
// caller has an ordering obligation left to get wrong.
func TestASwitchThroughTheOperationQueueNudgesLikeAnyOther(t *testing.T) {
	f := newFixture(t, 24, 80)
	incoming := ptychild.NewFakeChild(nil)
	incoming.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.attachThreadActor("c2", "c2", menuAddress("c2"), "c1", "brain", incoming)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	before := len(incoming.Resizes())
	// The switcher's Enter, not con.Switch: this is the dispatcher path that
	// runs off the Run goroutine.
	if _, err := f.con.Ops()(couchcore.OperationCall{
		Name: "switch", Implicit: true,
		Args: map[string]string{
			"repo-scope": menuAddress("c2").RepoScope,
			"tag":        string(menuAddress("c2").Tag),
		},
	}); err != nil {
		t.Fatal(err)
	}

	waitFor(t, "the incoming child to be asked to repaint", func() bool {
		return len(incoming.Resizes()) >= before+2
	})
	got := incoming.Resizes()[before:]
	want := f.con.ChildSize()
	if got[0].Rows != want.Rows-1 || got[1] != want {
		t.Fatalf("operation-path switch resized %v, want a shrink-and-restore around %v", got, want)
	}
	if size := incoming.Size(); size != want {
		t.Fatalf("child left at %v, want %v", size, want)
	}
}
