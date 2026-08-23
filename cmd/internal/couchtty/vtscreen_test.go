package couchtty

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// vtHost is a Host whose screen is a REAL terminal emulator.
//
// Every other console test asserts on the BYTES the console emits, which cannot
// answer the question Decision 4 actually rests on: does reserving a row by
// pinning the scrolling region survive a child that scrolls? A pty proves
// nothing here -- it passes escapes through uninterpreted. charmbracelet/x/vt
// interprets them, and pair already depends on it for wrapcmd's terminal model.
type vtHost struct {
	*hostty.FakeHost

	// vt.Emulator is not safe for concurrent use, and here it genuinely is
	// concurrent: the console's goroutines write while the test reads rows.
	// The mutex is the harness's, not a hint about production -- production
	// has exactly one writer and no reader.
	mu sync.Mutex
	em *vt.Emulator
}

func newVTHost(rows, cols uint16) *vtHost {
	return &vtHost{
		FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: rows, Cols: cols}),
		em:       vt.NewEmulator(int(cols), int(rows)),
	}
}

func (h *vtHost) Write(p []byte) (int, error) {
	_, _ = h.FakeHost.Write(p) // keep the byte-level assertions available
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.em.Write(p)
}

func (h *vtHost) Close() error {
	h.mu.Lock()
	_ = h.em.Close()
	h.mu.Unlock()
	return h.FakeHost.Close()
}

// childArea is everything the child can draw on: rows 1..rows-1. Assertions use
// it rather than naming a row, because a trailing newline leaves the cursor on
// a fresh blank line -- so "the last line of output" is on rows-2, not rows-1,
// and pinning the exact row would be asserting the test's arithmetic rather
// than the reservation.
func (h *vtHost) childArea() string {
	sz, _ := h.Size()
	var b strings.Builder
	for r := 1; r < int(sz.Rows); r++ {
		b.WriteString(h.row(r)) // row takes h.mu itself
		b.WriteString("\n")
	}
	return b.String()
}

// row reads what the emulator actually shows on a 1-based row.
func (h *vtHost) row(n int) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var b strings.Builder
	w, _ := h.Size()
	for x := 0; x < int(w.Cols); x++ {
		if c := h.em.CellAt(x, n-1); c != nil {
			b.WriteString(c.Content)
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func newVTFixture(t *testing.T, rows, cols uint16) (*vtHost, *ptychild.Child, *Console) {
	t.Helper()
	host := newVTHost(rows, cols)
	pr, pw := io.Pipe()
	con := New(host, pr)

	child := ptychild.NewFakeChild(nil)
	child.SetSink(func(chunk []byte) { con.Deliver("c1", chunk) })
	con.Attach("c1", "brain", child)

	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	t.Cleanup(func() {
		con.Stop()
		_ = pw.Close()
		<-done
	})
	waitFor(t, "the console to reserve", func() bool { return len(child.Resizes()) > 0 })
	return host, child, con
}

// The property the whole reserved-row design rests on: a child scrolling at the
// bottom of ITS screen scrolls inside the region and cannot reach the row below.
func TestReservedRowSurvivesAScrollingChild(t *testing.T) {
	host, child, _ := newVTFixture(t, 8, 40)

	waitFor(t, "the status row to be painted", func() bool {
		return strings.Contains(host.row(8), "brain")
	})
	for i := 0; i < 40; i++ {
		child.Feed([]byte("scrolling line\r\n"))
	}
	waitFor(t, "the child output to render", func() bool {
		return strings.Contains(host.childArea(), "scrolling line")
	})

	if got := host.row(8); !strings.Contains(got, "brain") {
		t.Fatalf("40 lines of child output overwrote the reserved row: %q", got)
	}
}

// nvim emits `\x1b[r` on exit, dropping the margins. The console must put the
// reservation back, or the row is silently lost for the rest of the session.
func TestReservedRowComesBackAfterAChildResetsMargins(t *testing.T) {
	host, child, _ := newVTFixture(t, 8, 40)
	waitFor(t, "the status row", func() bool { return strings.Contains(host.row(8), "brain") })

	child.Feed([]byte("\x1b[r"))
	waitFor(t, "the region to be re-asserted", func() bool {
		return strings.Contains(host.Written(), "\x1b[1;7r")
	})

	for i := 0; i < 40; i++ {
		child.Feed([]byte("after the reset\r\n"))
	}
	waitFor(t, "the child output to render", func() bool {
		return strings.Contains(host.childArea(), "after the reset")
	})
	if got := host.row(8); !strings.Contains(got, "brain") {
		t.Fatalf("the row did not survive a margin reset: %q", got)
	}
}

// Teardown must leave a terminal the operator's shell can use: full-height
// region, no stale row.
func TestReleaseLeavesAUsableScreen(t *testing.T) {
	host, child, con := newVTFixture(t, 8, 40)
	waitFor(t, "the status row", func() bool { return strings.Contains(host.row(8), "brain") })

	con.Stop()
	waitFor(t, "the region reset", func() bool {
		return strings.Contains(host.Written(), hostty.ResetRegion)
	})
	waitFor(t, "the row to be cleared", func() bool { return host.row(8) == "" })

	// And the shell that follows can scroll the WHOLE screen again. The last
	// write has no trailing newline, so the cursor -- and the text -- land on
	// the bottom row, which is the row that was fenced off a moment ago.
	for i := 0; i < 40; i++ {
		_, _ = host.Write([]byte("shell line\r\n"))
	}
	_, _ = host.Write([]byte("bottom row is usable"))
	if got := host.row(8); !strings.Contains(got, "bottom row is usable") {
		t.Fatalf("row 8 is still fenced off after release: %q", got)
	}
	_ = child
}

// Reported from the M2 operator smoke: the row appeared, then vanished about a
// second later as pair drew its first full screen.
//
// DECSTBM restricts SCROLLING, not ADDRESSING or ERASING. A child that clears
// the display -- which every full-screen app does on startup -- wipes the whole
// screen including the reserved row, and the reservation does nothing to stop
// it. The emulator tests missed this because a scrolling child never clears.
func TestReservedRowComesBackAfterAChildClearsTheScreen(t *testing.T) {
	for _, clear := range []struct {
		name string
		seq  string
	}{
		{"ED 2 (whole display)", "\x1b[2J"},
		{"ED default (cursor to end)", "\x1b[1;1H\x1b[J"},
		{"ED 1 (start to cursor)", "\x1b[8;40H\x1b[1J"},
		{"ED 3 (display + scrollback)", "\x1b[3J"},
		{"RIS", "\x1bc"},
	} {
		t.Run(clear.name, func(t *testing.T) {
			host, child, _ := newVTFixture(t, 8, 40)
			waitFor(t, "the status row", func() bool {
				return strings.Contains(host.row(8), "brain")
			})

			// Prove the clear was PROCESSED before asserting the row, using
			// a marker written after it. Chunks keep their order, so seeing
			// the marker means the clear has already been applied.
			//
			// Two earlier shapes of this were wrong. Asserting the row
			// directly passed on every case, because the poll ran before the
			// async chunk reached the screen and saw the row still standing
			// from BEFORE -- an assertion satisfied by the pre-state proves
			// nothing. Waiting to OBSERVE the row vanish is flaky the other
			// way: when the repaint is fast (RIS), the damaged state may never
			// be visible at all.
			child.Feed([]byte(clear.seq))
			child.Feed([]byte("\x1b[1;1HMARKER"))
			waitFor(t, "the clear and the marker to reach the screen", func() bool {
				return strings.Contains(host.childArea(), "MARKER")
			})
			waitFor(t, "the row to be repainted", func() bool {
				return strings.Contains(host.row(8), "brain")
			})
		})
	}
}

// The whole startup shape of a full-screen child: clear, then draw. The row has
// to survive the pair.
func TestReservedRowSurvivesAFullScreenChildStartingUp(t *testing.T) {
	host, child, _ := newVTFixture(t, 8, 40)
	waitFor(t, "the status row", func() bool { return strings.Contains(host.row(8), "brain") })

	child.Feed([]byte("\x1b[2J\x1b[1;1H"))
	for i := 1; i <= 7; i++ {
		child.Feed([]byte("pane content\r\n"))
	}
	waitFor(t, "the child's screen", func() bool {
		return strings.Contains(host.childArea(), "pane content")
	})
	waitFor(t, "the row to be repainted", func() bool {
		return strings.Contains(host.row(8), "brain")
	})
}
