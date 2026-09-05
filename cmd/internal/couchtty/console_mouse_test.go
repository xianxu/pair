package couchtty

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// newMouseFixture drives REAL bytes through Run's input loop with a status row
// painted, which is the only way to test that a click reaches a switch: calling
// onMouse directly would skip the Interceptor, the geometry and the focus rule.
func newMouseFixture(t *testing.T) (*Console, *io.PipeWriter, *hostty.FakeHost, chan couchcore.OperationCall) {
	t.Helper()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	reader, writer := io.Pipe()
	con := New(host, reader)
	t.Cleanup(func() {
		con.Stop()
		_ = writer.Close()
	})
	calls := make(chan couchcore.OperationCall, 8)
	con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		calls <- call
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
	case call := <-calls:
		if call.Name != "switch" {
			t.Fatalf("dispatched %q, want the declared switch", call.Name)
		}
		if call.Args["tag"] != string(menuAddress("two").Tag) {
			t.Fatalf("switched to %q, want the clicked chip's thread", call.Args["tag"])
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
	case call := <-calls:
		t.Fatalf("a click on empty row space dispatched %q", call.Name)
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
