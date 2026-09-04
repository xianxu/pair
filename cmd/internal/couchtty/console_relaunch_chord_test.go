package couchtty

import (
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// newChordFixture drives REAL operator bytes: a pipe into Run's own input loop,
// an actor focused, one live thread in the inventory. Calling a handler method
// directly -- which is what every other console regression does -- cannot see a
// chord that is intercepted and then dropped for want of a dispatch arm.
func newChordFixture(t *testing.T) (*Console, *io.PipeWriter, couchcore.ThreadAddress) {
	t.Helper()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	reader, writer := io.Pipe()
	con := New(host, reader)
	address := menuAddress("brain")
	con.attachThreadActor("c1", "brain", address, "/w/brain", "brain", ptychild.NewFakeChild(nil))
	setTestOps(con, func(string, map[string]string) (any, error) { return nil, nil })
	con.mu.Lock()
	con.active = "c1"
	con.focus = FocusActor("c1")
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/brain", Name: "brain", State: couchcore.ThreadLive,
	}}, address)
	con.menuReady = true
	con.mu.Unlock()
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	t.Cleanup(func() {
		con.Stop()
		_ = writer.Close()
	})
	return con, writer, address
}

// The chord shipped once with its bytes intercepted and NO arm in
// processInput's switch to receive them, so alt+n was swallowed and silently
// dropped while every interceptor test stayed green. This is the test that
// fails when that happens.
func TestRelaunchChordBytesFromAnActorReachTheConfirmation(t *testing.T) {
	for _, chord := range []struct {
		name  string
		chord workbenchshortcut.Chord
	}{
		{"alt+n", workbenchshortcut.ChordAltN},
		{"ctrl+alt+n", workbenchshortcut.ChordCtrlAltN},
	} {
		for _, encoding := range workbenchshortcut.ChordEncodings(chord.chord) {
			t.Run(chord.name+"/"+renderInputBytes(encoding), func(t *testing.T) {
				con, stdin, address := newChordFixture(t)
				waitFor(t, "the console to start", func() bool { return con.menuSnapshot().Inventory != nil })

				if _, err := stdin.Write(encoding); err != nil {
					t.Fatalf("write chord: %v", err)
				}
				waitFor(t, "the relaunch confirmation", func() bool {
					frame := con.menuSnapshot().CurrentFrame()
					return frame.Kind == MenuFrameConfirmation && frame.Action == "relaunch" && frame.Thread == address
				})
				con.mu.Lock()
				focus := con.focus
				con.mu.Unlock()
				if !focus.IsPanel() {
					t.Fatalf("relaunch left focus on the actor: %+v", focus)
				}
			})
		}
	}
}

// The other half of the contract, and the half an operator notices first:
// alt+shift+n is deliberately NOT couch's, so its bytes must reach the child
// untouched. It is what keeps "same code, new conversation" working inside a
// hosted Pair after couch claimed the heavier chord.
func TestAltShiftNBytesReachTheChildUntouched(t *testing.T) {
	con, stdin, _ := newChordFixture(t)
	waitFor(t, "the console to start", func() bool { return con.menuSnapshot().Inventory != nil })

	encodings := workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltShiftN)
	if len(encodings) == 0 {
		t.Fatal("alt+shift+n has no encoding to forward")
	}
	for _, encoding := range encodings {
		if _, err := stdin.Write(encoding); err != nil {
			t.Fatalf("write chord: %v", err)
		}
	}
	waitFor(t, "the child to receive alt+shift+n", func() bool {
		child := con.activeChild()
		if child == nil {
			return false
		}
		for _, write := range child.Writes() {
			if string(write) == string(encodings[0]) {
				return true
			}
		}
		return false
	})
	con.mu.Lock()
	focus := con.focus
	con.mu.Unlock()
	if focus.IsPanel() {
		t.Fatal("alt+shift+n opened couch's panel; it belongs to the hosted Pair")
	}
}

// The operator's report, as a test: alt+n opened a confirmation whose only
// action read "park brain" under a "relaunch" breadcrumb, and it could not be
// chosen. Both halves are one defect -- the item's first word is its dispatch
// id, so a hand-written "park " prefix both mislabels the screen and fails the
// id == action guard that Enter checks.
func TestRelaunchConfirmationNavigatesAndDispatchesRelaunch(t *testing.T) {
	// Both arrow encodings, because the operator reaches this screen FROM the
	// pair pane: nvim leaves the terminal in application-cursor mode, where
	// down is \x1bOB rather than \x1b[B, and a decoder that knew only one
	// would strand exactly this confirmation.
	for _, arrow := range []string{"\x1b[B", "\x1bOB"} {
		t.Run(renderInputBytes([]byte(arrow)), func(t *testing.T) {
			relaunchConfirmationRoundTrip(t, arrow)
		})
	}
}

func relaunchConfirmationRoundTrip(t *testing.T, arrowDown string) {
	t.Helper()
	con, stdin, address := newChordFixture(t)
	dispatched := make(chan string, 4)
	setTestOps(con, func(name string, _ map[string]string) (any, error) {
		dispatched <- name
		return nil, nil
	})
	waitFor(t, "the console to start", func() bool { return con.menuSnapshot().Inventory != nil })

	if _, err := stdin.Write([]byte("\x1b[110;3u")); err != nil {
		t.Fatalf("write alt+n: %v", err)
	}
	waitFor(t, "the relaunch confirmation", func() bool {
		frame := con.menuSnapshot().CurrentFrame()
		return frame.Kind == MenuFrameConfirmation && frame.Action == "relaunch" && frame.Thread == address
	})

	// The action item must NAME relaunch, and must not name park.
	items := confirmationMenuItems(con.menuSnapshot(), con.menuSnapshot().CurrentFrame())
	if len(items) != 2 || !strings.HasPrefix(items[1], "relaunch ") {
		t.Fatalf("confirmation items = %q, want the second to start with %q", items, "relaunch ")
	}

	// Arrow down off "cancel" must land on it.
	if _, err := stdin.Write([]byte(arrowDown)); err != nil {
		t.Fatalf("write arrow down: %v", err)
	}
	waitFor(t, "the arrow to move the selection off cancel", func() bool {
		return con.menuSnapshot().CurrentFrame().SelectedItem == "relaunch"
	})

	// And Enter must actually dispatch, not bounce to the root with
	// "thread action is no longer applicable".
	if _, err := stdin.Write([]byte("\r")); err != nil {
		t.Fatalf("write enter: %v", err)
	}
	select {
	case name := <-dispatched:
		if name != "relaunch" {
			t.Fatalf("dispatched %q, want %q", name, "relaunch")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Enter dispatched nothing; notice = %+v", con.menuSnapshot().Notice)
	}
}

// The operator's second report: relaunch worked, the record said live, the
// switcher rendered "live" -- and Return could not reach the thread because it
// had no pane. couch had spawned a child and never adopted it. Adoption ran off
// a concrete StartResult assertion that a RelaunchResult could not satisfy.
func TestRelaunchResultIsAdoptedByTheConsole(t *testing.T) {
	address := menuAddress("brain")
	record := couchcore.ActorRecord{ID: "brain", Thread: address}
	for _, tc := range []struct {
		name   string
		value  any
		attach bool
	}{
		{"a completed relaunch is adopted", couchcore.RelaunchResult{
			Outcome: couchcore.Relaunched, Record: record,
		}, true},
		{"a start is still adopted", couchcore.StartResult{Record: record}, true},
		// Every other outcome accompanies an error and has no live child; a
		// console that attached to one would be pointing at nothing.
		{"a refused relaunch is not", couchcore.RelaunchResult{
			Outcome: couchcore.RefusedBeforePark, Record: record,
		}, false},
		{"a park-without-resume is not", couchcore.RelaunchResult{
			Outcome: couchcore.ParkedNotResumed, Record: record,
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(con.Stop)
			var dispatched []string
			con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				dispatched = append(dispatched, call.Name)
				return nil, nil
			})
			con.finishOperation(operationCompletion{
				name:   "relaunch",
				value:  tc.value,
				origin: MenuOperationOrigin{Operation: "relaunch", Address: address, Attempt: 1},
			})
			attached := slices.Contains(dispatched, "attach")
			if attached != tc.attach {
				t.Fatalf("attach dispatched = %v, want %v (dispatched %q)", attached, tc.attach, dispatched)
			}
		})
	}
}
