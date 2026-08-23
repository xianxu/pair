package couchtty

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

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

	f := &consoleFixture{host: host, child: child, con: con, stdin: pw, done: make(chan int, 1)}
	go func() { f.done <- con.Run() }()
	t.Cleanup(func() {
		con.Stop()
		_ = pw.Close()
	})
	return f
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

	waitFor(t, "both segments to arrive", func() bool {
		var all string
		for _, w := range f.child.Writes() {
			all += string(w)
		}
		return strings.Contains(all, "a") && strings.Contains(all, "b")
	})
	for _, w := range f.child.Writes() {
		if strings.ContainsRune(string(w), 0x00) {
			t.Fatalf("the hotkey reached the child: %q", w)
		}
	}
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

	f.child.Exit(3)
	select {
	case code := <-f.done:
		if code != 3 {
			t.Fatalf("Run() = %d, want the child's code 3", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not return after the child exited")
	}
	if !strings.Contains(f.host.Written(), hostty.ResetRegion) {
		t.Fatalf("the scrolling region was not reset: %q", f.host.Written())
	}
	if f.host.RawDepth() != 0 {
		t.Fatalf("raw mode left on: RawDepth = %d", f.host.RawDepth())
	}
}

// Restoration on the MID-STREAM teardown path. A restore that only runs on the
// happy path leaves the operator's terminal with a pinned scrolling region.
func TestConsoleRestoresTheTerminalOnTeardownMidStream(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })

	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not return after Stop")
	}
	if !strings.Contains(f.host.Written(), hostty.ResetRegion) {
		t.Fatalf("the scrolling region was not reset on teardown: %q", f.host.Written())
	}
	if f.host.RawDepth() != 0 {
		t.Fatalf("raw mode left on after teardown: RawDepth = %d", f.host.RawDepth())
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
	f.con.Attach("c2", "ariadne", other)
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

	// Park the stream mid-sequence, then hammer every other writer.
	f.child.Feed([]byte("\x1b[2J\x1b[38;2;76"))
	for i := 0; i < 20; i++ {
		f.host.SetSize(ptychild.Size{Rows: uint16(24 + i%3), Cols: 80})
		_, _ = f.stdin.Write([]byte("\x00"))
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
