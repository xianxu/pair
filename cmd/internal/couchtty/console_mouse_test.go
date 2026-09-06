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
	// SetSink is what carries a child's output into the console. Without it
	// Feed updates the child's Screen and the console never hears about it --
	// so any test of a TRIGGER passes or fails for the wrong reason. This
	// fixture lacked it, which is why every mode test had to call repaint() by
	// hand and why the missing re-evaluation stayed invisible.
	first, second := ptychild.NewFakeChild(nil), ptychild.NewFakeChild(nil)
	first.SetSink(func(batch ptychild.OutputBatch) { con.Deliver("c1", batch) })
	second.SetSink(func(batch ptychild.OutputBatch) { con.Deliver("c2", batch) })
	con.attachThreadActor("c1", "one", one, "/w/one", "one", first)
	con.attachThreadActor("c2", "two", two, "/w/two", "two", second)
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
	// The child asks for tracking AND for the SGR encoding -- fed through the
	// same path the real pump uses. Both matter: tracking is what makes
	// forwarding right, and the encoding is what makes an SGR report parseable
	// by this child. A test asking only for tracking would expect bytes the
	// child could not read.
	child.Feed([]byte("\x1b[?1000;1006h"))
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

// couch must not demote a child's tracking mode. 1000/1002/1003 are ONE
// mutually-exclusive state, not additive flags, so re-asserting ?1000h under a
// child holding ?1002h replaces button-event tracking with press/release --
// and the child then never receives the motion that closes its drag, which is
// nvim stuck in visual selection.
//
// This is the case a keyboard smoke test cannot reach: it needs a child using
// motion tracking, and the damage is to that child's drag rather than to couch.
func TestCouchDoesNotDemoteAChildsTrackingMode(t *testing.T) {
	con, _, host, _ := newMouseFixture(t)
	child := con.activeChild()
	child.Feed([]byte("\x1b[?1002h"))
	waitFor(t, "the child's motion tracking to register", func() bool { return child.Mouse() })

	host.Reset()
	con.repaint()
	con.repaint()
	waitFor(t, "a repaint", func() bool { return host.Written() != "" })

	if strings.Contains(host.Written(), hostty.EnableMouseClicks) {
		t.Fatal("couch re-asserted ?1000h under a child holding ?1002h, demoting it to press/release")
	}
}

// The mode-transition table, one case per row. couch's own tracking was
// implemented for one of these and pinned for none -- deleting the per-paint
// re-assert left the whole suite green, because the only existing test checked
// the STARTUP write.
//
// The rule under every row: the child's mode wins whenever it has one, and couch
// takes the terminal back the moment it does not.
func TestMouseModeTransitions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		childMode string
		gone      bool
		wantCouch bool
	}{
		{"child enables click tracking: couch stands back", "\x1b[?1000h", false, false},
		{"child enables button-event tracking: couch stands back", "\x1b[?1002h", false, false},
		{"child enables any-event tracking: couch stands back", "\x1b[?1003h", false, false},
		{"child disables: couch takes the terminal back", "\x1b[?1002h\x1b[?1002l", false, true},
		// UNKNOWN is not "no". A fresh Screen for a still-running child -- what
		// a reattach mints -- has observed nothing, and writing on the strength
		// of that overwrites a mode couch never witnessed (pair#196).
		{"nothing observed yet: couch stands back", "", false, false},
		// The surviving actor decides, not the departed one: onExit falls the
		// active slot back to c2, which has observed nothing, so couch stands
		// back for the same reason as the row above. Asserting "couch takes it
		// back" here would have been asserting the fixture's shape rather than
		// the rule -- what the exit itself changes is that c1's mode stops
		// being the answer.
		{"child exits with mouse on: the SURVIVOR decides", "\x1b[?1002h", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			con, _, host, _ := newMouseFixture(t)
			child := con.activeChild()
			if tc.childMode != "" {
				child.Feed([]byte(tc.childMode))
				waitFor(t, "the child's mode to settle", func() bool {
					return child.Mouse() == !strings.HasSuffix(tc.childMode, "l")
				})
			}
			if tc.gone {
				// Through the real exit path, not by deleting the map entry:
				// onExit is what production runs, and it is where the active
				// slot moves to a surviving actor. Reaching into panes would
				// test a state the console never reaches.
				child.Close()
				waitFor(t, "the child to finish", func() bool { return child.Done() })
				con.onExit(childExit{id: "c1", code: 0})
			}

			host.Reset()
			con.repaint()
			waitFor(t, "a repaint", func() bool { return host.Written() != "" })

			got := strings.Contains(host.Written(), hostty.EnableMouseClicks)
			if got != tc.wantCouch {
				if tc.wantCouch {
					t.Fatalf("couch did not re-assert its tracking, so a child's DECRST silently ends the feature")
				}
				t.Fatalf("couch asserted ?1000h over the child's mode, demoting it and wedging its drag")
			}
		})
	}
}

// Switching between children with different modes is the sixth cell, and the one
// most likely to be got wrong: the mode is terminal-global while the state is
// per-pane, so the ARRIVING actor's mode has to win.
func TestSwitchingToAChildWithoutTrackingReturnsTheTerminalToCouch(t *testing.T) {
	con, _, host, _ := newMouseFixture(t)
	// c1 holds motion tracking; c2 holds nothing.
	con.activeChild().Feed([]byte("\x1b[?1002h"))
	waitFor(t, "c1's mode to register", func() bool { return con.activeChild().Mouse() })
	host.Reset()
	con.repaint()
	waitFor(t, "a paint under c1", func() bool { return host.Written() != "" })
	if strings.Contains(host.Written(), hostty.EnableMouseClicks) {
		t.Fatal("couch asserted over c1's motion tracking")
	}

	// c2 must have SAID it holds none. A child that has announced nothing is
	// unknown, not "no", and couch stands back for it -- so switching to a
	// silent c2 would leave the terminal on c1's mode and this test would be
	// asserting the fixture rather than the rule.
	con.mu.Lock()
	other := con.panes["c2"]
	con.mu.Unlock()
	other.child.Feed([]byte("\x1b[?1000h\x1b[?1000l"))
	waitFor(t, "c2 to declare it holds no tracking", func() bool {
		return other.child.MouseObserved() && !other.child.Mouse()
	})

	con.forceSwitch("c2")
	waitFor(t, "couch to take the terminal back under c2", func() bool {
		return strings.Contains(host.Written(), hostty.EnableMouseClicks)
	})
}

// couch requests ?1006 for itself, so the terminal emits SGR regardless of what
// the child wanted. A child holding ?1000h WITHOUT ?1006h asked for the legacy
// encoding and cannot parse an SGR report -- forwarding one puts unparseable
// bytes in its input, which is "receives its own events unchanged" read
// backwards.
func TestAChildThatDidNotAskForSGRIsNotSentSGR(t *testing.T) {
	con, writer, _, _ := newMouseFixture(t)
	child := con.activeChild()
	// Tracking, but the LEGACY encoding.
	child.Feed([]byte("\x1b[?1000h"))
	waitFor(t, "the child's tracking to register", func() bool { return child.Mouse() })
	if child.SGRMouse() {
		t.Fatal("the fixture child asked for SGR, so this test cannot distinguish the encodings")
	}
	before := len(child.Writes())

	if _, err := writer.Write([]byte("\x1b[<0;7;9M")); err != nil {
		t.Fatal(err)
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
			t.Fatalf("an SGR report was forwarded to a child that asked for the legacy encoding: %q", w)
		}
	}
}

// The operator's pair#196, as a test: agent-pane drag selection loses its live
// highlight after some couch reattachments.
//
// A detach/reattach mints a NEW Child for a still-running agent, and its Screen
// starts empty. The agent will not re-emit the startup DECSET it sent long ago,
// and replay can only re-derive it while that sequence is still inside the ring.
// So couch's belief is not "the child wants no mouse" — it is "couch has not
// seen the child say anything", and writing ?1000 on the strength of that
// demoted a child holding ?1002 to press/release: selection still works, but the
// during-drag feedback is gone, because motion reports stop.
//
// Silence is not consent. This is the case a keyboard smoke test cannot reach
// and mainstream children hide: nvim and zellij both announce ?1006, so the
// common configuration looks fine.
func TestAReattachedChildKeepsItsTrackingMode(t *testing.T) {
	con, _, host, _ := newMouseFixture(t)
	address := menuAddress("one")

	// The agent announced motion tracking long ago, on a Child that is gone.
	// What couch has NOW is a fresh pane for the same running thread, which is
	// what a detach/reattach mints -- reached through the real exit path rather
	// than by deleting from the map, which no production path does and which
	// leaves the console painting a pane it still believes in.
	original := con.activeChild()
	original.Feed([]byte("\x1b[?1002h"))
	waitFor(t, "the original child's tracking to register", func() bool { return original.Mouse() })
	_ = original.Close()
	waitFor(t, "the original child to finish", func() bool { return original.Done() })
	con.onExit(childExit{id: "c1", code: 0})

	con.attachThreadActor("c1b", "one", address, "/w/one", "one", ptychild.NewFakeChild(nil))
	con.mu.Lock()
	con.active = "c1b"
	fresh := con.panes["c1b"].child
	con.mu.Unlock()

	if fresh.MouseObserved() {
		t.Fatal("a fresh Child claims to have observed a mode, so this test cannot reproduce the reattach")
	}
	if fresh.Mouse() {
		t.Fatal("a fresh Child claims tracking, so the false-negative this test exists for cannot occur")
	}

	host.Reset()
	con.repaint()
	con.repaint()
	waitFor(t, "a repaint", func() bool { return host.Written() != "" })

	if strings.Contains(host.Written(), hostty.EnableMouseClicks) {
		t.Fatal("couch wrote ?1000 over a reattached child whose mode it never observed, demoting a still-tracking agent")
	}
}

// The TRIGGER, which every other mode test misses because they call repaint()
// by hand -- a test that observes one interleaving the author chose.
//
// A child dropping its tracking must cause couch to take the terminal back on
// its own. Before the latch, `?1002l` left couch's clicks off until some
// unrelated paint happened to run, so the feature silently stopped and nothing
// said why.
func TestAChildDroppingItsModeMakesCouchReclaimTheTerminalWithoutAPaintCall(t *testing.T) {
	con, _, host, _ := newMouseFixture(t)
	child := con.activeChild()
	child.Feed([]byte("\x1b[?1002h"))
	waitFor(t, "the child's tracking to register", func() bool { return child.Mouse() })
	// Reset AFTER the mode has registered. The latch means the DECSET itself
	// triggers a paint, and that paint races the registration -- it can run
	// while Screen still reports unobserved, which is correct behaviour (couch
	// stands back when it does not know) but writes EnableMouseClicks into the
	// buffer. Asserting on bytes written before the fact is asserting the race.
	host.Reset()
	con.repaint()
	waitFor(t, "a paint under the tracking child", func() bool { return host.Written() != "" })
	if strings.Contains(host.Written(), hostty.EnableMouseClicks) {
		t.Fatal("couch asserted over a child holding ?1002")
	}

	// No repaint() here. The child's DECRST alone must get couch's mode back.
	host.Reset()
	child.Feed([]byte("\x1b[?1002l"))
	waitFor(t, "couch to reclaim the terminal on its own", func() bool {
		return strings.Contains(host.Written(), hostty.EnableMouseClicks)
	})
}
