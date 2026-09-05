package couchtty

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// newMouseFixture drives REAL bytes through Run's input loop with a status row
// painted, which is the only way to test that a click reaches a switch: calling
// onMouse directly would skip the Interceptor, the geometry and the focus rule.
// dispatched records what the console asked for AND the in-flight state at the
// moment it asked. The attention capture is only observable there: InFlight is
// cleared on completion, so a test that waits for the dispatch and then reads
// the flag reads a zero value and passes for the wrong reason.
type dispatched struct {
	call    couchcore.OperationCall
	capture AttentionCapture
	manual  bool
}

func newMouseFixture(t *testing.T) (*Console, *io.PipeWriter, *hostty.FakeHost, chan dispatched) {
	t.Helper()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	reader, writer := io.Pipe()
	con := New(host, reader)
	t.Cleanup(func() {
		con.Stop()
		_ = writer.Close()
	})
	calls := make(chan dispatched, 8)
	con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		con.mu.Lock()
		origin := con.menu.InFlight
		con.mu.Unlock()
		calls <- dispatched{call: call, capture: origin.AttentionCapture, manual: origin.Manual}
		return nil, nil
	})
	one, two := menuAddress("one"), menuAddress("two")
	con.attachThreadActor("c1", "one", one, "/w/one", "one", ptychild.NewFakeChild(nil))
	con.attachThreadActor("c2", "two", two, "/w/two", "two", ptychild.NewFakeChild(nil))
	con.mu.Lock()
	con.active = "c1"
	con.focus = FocusActor("c1")
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{
		{Address: one, Name: "one", WorkingPath: "/w/one", State: couchcore.ThreadLive},
		{Address: two, Name: "two", WorkingPath: "/w/two", State: couchcore.ThreadLive},
	}, one)
	con.menuReady = true
	con.mu.Unlock()
	go con.Run()
	waitFor(t, "the console to start", func() bool { return host.Written() != "" })
	return con, writer, host, calls
}

func clickAt(t *testing.T, w *io.PipeWriter, col, row int) {
	t.Helper()
	if _, err := fmt.Fprintf(w, "\x1b[<0;%d;%dM", col, row); err != nil {
		t.Fatalf("write click: %v", err)
	}
}

// The headline: clicking a chip on the reserved row switches to that actor,
// through the SAME declared operation ctrl-space+Return dispatches.
func TestClickOnAChipDispatchesTheSwitchOperation(t *testing.T) {
	con, writer, _, calls := newMouseFixture(t)
	waitFor(t, "the status row to be painted with chips", func() bool {
		con.mu.Lock()
		defer con.mu.Unlock()
		return len(con.statusChips) == 2
	})
	con.mu.Lock()
	chip := con.statusChips[1]
	rows := int(con.size.Rows)
	con.mu.Unlock()

	// 1-based, as a terminal reports it.
	clickAt(t, writer, chip.Start+1, rows)

	select {
	case got := <-calls:
		if got.call.Name != "switch" {
			t.Fatalf("dispatched %q, want the declared switch", got.call.Name)
		}
		if got.call.Args["tag"] != string(menuAddress("two").Tag) {
			t.Fatalf("switched to %q, want the clicked chip's thread", got.call.Args["tag"])
		}
	case <-timeoutAfter():
		t.Fatal("a click on a chip dispatched nothing")
	}
}

// Clicking bare row space is not a target and must do nothing at all.
func TestClickOnBareRowSpaceDoesNothing(t *testing.T) {
	con, writer, _, calls := newMouseFixture(t)
	waitFor(t, "the status row to be painted with chips", func() bool {
		con.mu.Lock()
		defer con.mu.Unlock()
		return len(con.statusChips) == 2
	})
	con.mu.Lock()
	past := con.statusChips[1].End + 5
	rows := int(con.size.Rows)
	con.mu.Unlock()

	clickAt(t, writer, past+1, rows)

	select {
	case got := <-calls:
		t.Fatalf("a click on empty row space dispatched %q", got.call.Name)
	case <-timeoutAfter():
	}
}

// A child that never enabled mouse tracking receives ZERO mouse bytes while
// couch's own tracking is on. Asserted, not assumed -- couch enables reporting
// terminal-globally, so without the swallow the child gets reports it never
// asked for and renders them as typeahead.
func TestChildWithoutTrackingReceivesNoMouseBytes(t *testing.T) {
	con, writer, _, _ := newMouseFixture(t)
	child := con.activeChild()
	before := len(child.Writes())

	// Every corner of the event space, none of it on couch's row.
	for _, row := range []int{1, 5, 12, 23} {
		clickAt(t, writer, 10, row)
	}
	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the ordinary keystroke to arrive", func() bool {
		for _, w := range child.Writes() {
			if string(w) == "x" {
				return true
			}
		}
		return false
	})
	for _, w := range child.Writes()[before:] {
		if len(w) > 0 && w[0] == 0x1b {
			t.Fatalf("a mouse report reached a child that never enabled tracking: %q", w)
		}
	}
}

func timeoutAfter() <-chan time.Time { return time.After(400 * time.Millisecond) }

// The behaviour this issue's commit message headlines, and which the plan calls
// "not optional" — and it shipped unpinned. A click is ALWAYS a manual switch,
// so ctrl+backspace undoes it even when the clicked actor was paging. Enter on a
// paging actor is a notification hop and non-pinning; that is the one input
// where click and Enter legitimately differ.
func TestClickIsAManualSwitch(t *testing.T) {
	con, writer, _, calls := newMouseFixture(t)
	two := menuAddress("two")
	// `two` is PAGING. Through Enter this would be a notification hop.
	con.mu.Lock()
	con.attention.Mark(two, "paging")
	con.mu.Unlock()
	con.repaint()
	waitFor(t, "the status row to be painted with chips", func() bool {
		con.mu.Lock()
		defer con.mu.Unlock()
		return len(con.statusChips) == 2
	})
	con.mu.Lock()
	chip := con.statusChips[1]
	rows := int(con.size.Rows)
	con.mu.Unlock()

	clickAt(t, writer, chip.Start+1, rows)
	var got dispatched
	select {
	case got = <-calls:
	case <-timeoutAfter():
		t.Fatal("the click dispatched nothing")
	}

	// Observed AT dispatch, which is the only moment it exists. A zero capture
	// is what makes ExecuteConsoleOperation derive arrivalOrdinary, and
	// arrivalOrdinary is what re-pins `previous` so ctrl+backspace undoes the
	// click -- even though this actor is paging, where Enter would not.
	if !got.manual {
		t.Error("the click was not marked manual")
	}
	if got.capture != 0 {
		t.Errorf("the click captured attention (%d), so it is classified as a notification hop and ctrl+backspace will not undo it", got.capture)
	}

	// Asserted on the CAPTURE, not on the Manual flag: InFlight is cleared when
	// the operation completes, so reading the flag after waiting for the
	// dispatch reads a zero value and passes for the wrong reason. The capture
	// is what arrivalNotification is derived from, so its absence IS the
	// property -- a click never becomes a notification hop.
	waitFor(t, "the switch to be recorded", func() bool {
		con.mu.Lock()
		defer con.mu.Unlock()
		return con.menu.InFlight.Operation == "" || con.menu.InFlight.Manual
	})
}

// A child that DID enable tracking receives its events unchanged -- the exact
// wire bytes, never a re-encoding from the parsed fields.
func TestForwardPreservesRawBytes(t *testing.T) {
	con, writer, _, _ := newMouseFixture(t)
	child := con.activeChild()
	// The child ASKS for mouse reporting -- fed through the same path the real
	// pump uses, so its Screen sees the DECSET exactly as it would in
	// production. That is what makes forwarding the right answer for it.
	child.Feed([]byte("\x1b[?1000h"))
	waitFor(t, "the child's mouse mode to register", func() bool { return child.Mouse() })

	const report = "\x1b[<0;7;9M"
	if _, err := writer.Write([]byte(report)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the report to reach the child", func() bool {
		for _, w := range child.Writes() {
			if string(w) == report {
				return true
			}
		}
		return false
	})
}

// Teardown leaves the host terminal with mouse reporting off, or the operator's
// shell starts emitting SGR bytes on pointer movement.
func TestTeardownDisablesMouseTracking(t *testing.T) {
	con, _, host, _ := newMouseFixture(t)
	waitFor(t, "couch to enable its own tracking", func() bool {
		return strings.Contains(host.Written(), hostty.EnableMouseClicks)
	})
	host.Reset()
	con.Stop()
	waitFor(t, "teardown to reset the terminal", func() bool {
		return strings.Contains(host.Written(), hostty.ResetInteractiveModes)
	})
}

// couch must ENABLE its own tracking, or the terminal sends nothing and no click
// is possible at all.
func TestCouchEnablesItsOwnMouseTracking(t *testing.T) {
	_, _, host, _ := newMouseFixture(t)
	waitFor(t, "couch to ask the terminal for clicks", func() bool {
		return strings.Contains(host.Written(), hostty.EnableMouseClicks)
	})
}

// A click in the SWITCHER takes the same declared operation Return takes, chosen
// by the same rule -- live rows switch, resumable rows resume.
func TestClickInTheSwitcherTakesTheReturnPath(t *testing.T) {
	con, writer, _, calls := newMouseFixture(t)
	con.mu.Lock()
	con.focus = FocusPanel()
	con.mu.Unlock()
	con.showMenu()
	waitFor(t, "the menu to record its extents", func() bool {
		con.mu.Lock()
		defer con.mu.Unlock()
		return len(con.menuExtents) == 2
	})
	con.mu.Lock()
	extent := con.menuExtents[1]
	con.mu.Unlock()

	// 1-based row, as a terminal reports it.
	clickAt(t, writer, 3, extent.Start+1)
	select {
	case got := <-calls:
		if got.call.Name != "switch" {
			t.Fatalf("dispatched %q, want the declared switch Return dispatches", got.call.Name)
		}
		if got.call.Args["tag"] != string(extent.Thread.Tag) {
			t.Fatalf("switched to %q, want the clicked row's thread %q", got.call.Args["tag"], extent.Thread.Tag)
		}
	case <-timeoutAfter():
		t.Fatal("a click in the switcher dispatched nothing")
	}
}
