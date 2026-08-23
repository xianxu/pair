package couchtty

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
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
