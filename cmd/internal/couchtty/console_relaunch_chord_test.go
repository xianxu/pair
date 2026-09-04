package couchtty

import (
	"io"
	"testing"

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
