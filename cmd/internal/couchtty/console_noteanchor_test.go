package couchtty

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// Exercise real broadcast acceptance and Console composition. A private native
// fixture is an oracle for unexpected queries; terminal notes need no native IO.
func TestNoteAnchorThroughTheBroadcast(t *testing.T) {
	native := zellijtest.New(t)
	for _, entry := range native.Env() {
		key, value, _ := strings.Cut(entry, "=")
		t.Setenv(key, value)
	}
	f, s, link := pointerOnFixture(t, nil)
	noteUntilSaved(t, link, 0, 0, "this line")
	waitFor(t, "note applied", func() bool { notes, _ := f.heldNotes(); return len(notes) == 1 })
	notes, _ := s.Notes()
	n := notes[0]
	if n.Box != cb(0, 0, 2, 1) || n.On != (terminal.Geometry{Cols: 80, Rows: 24}) || n.Text != "this line" || n.Anchor == nil || n.Anchor.Kind != broadcast.AnchorScreen || len(n.Anchor.Path) != 1 || !strings.HasPrefix(n.Anchor.Path[0], "pane ") || n.Anchor.Text != "ch" {
		t.Fatalf("lost draw-time note data: %+v anchor=%+v", n, n.Anchor)
	}
	// Changing output later must not replace the text captured when it opened.
	f.child.Feed([]byte("\rXXchanged"))
	waitFor(t, "later output", func() bool {
		f.con.anchors.mu.Lock()
		defer f.con.anchors.mu.Unlock()
		return TextUnder(f.con.anchors.frame, cb(0, 0, 2, 1)) == "XX"
	})
	notes, _ = s.Notes()
	if notes[0].Anchor.Text != "ch" {
		t.Fatalf("opening text changed: %+v", notes[0].Anchor)
	}
	noteUntilSaved(t, link, 40, 23, "the tab bar")
	waitFor(t, "two notes applied", func() bool { n, _ := f.heldNotes(); return len(n) == 2 })
	notes, _ = s.Notes()
	if a := notes[1].Anchor; a == nil || (a.Kind != broadcast.AnchorTabBar && a.Kind != broadcast.AnchorStatusLine) {
		t.Fatalf("lost own last-row anchor: %+v", a)
	}
	if err := f.con.runTerminalCommand(context.Background(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(native.Queries()) != 0 || len(native.Effects()) != 0 {
		t.Fatalf("terminal notes reached native Zellij: queries=%v effects=%v", native.Queries(), native.Effects())
	}
}

func TestNoteAnchorRejectsDifferentGrid(t *testing.T) {
	c := &Console{}
	f, _ := anchorScreen()
	c.keepNoteFrame(f)
	if a := c.anchorNote(broadcast.Note{Box: cb(0, 0, 2, 1)}, 80, 24); a.Kind != broadcast.AnchorScreen || len(a.Path) != 0 || a.Text != "" {
		t.Fatalf("wrong-grid attribution: %+v", a)
	}
}

func TestNoteAnchorDoesNotGuessReplacementSlot(t *testing.T) {
	f, _, _ := pointerOnFixture(t, nil)
	replacement := ptychild.NewFakeChild(nil)
	f.con.Attach("c2", "replacement", replacement)
	f.con.mu.Lock()
	delete(f.con.panes, "c1")
	f.con.order = []string{"c2"}
	f.con.active = "c2"
	f.con.mu.Unlock()
	a := f.con.anchorNote(broadcast.Note{Box: cb(0, 0, 2, 1)}, 80, 24)
	if len(a.Path) != 1 || a.Path[0] != "pane the slot in view" {
		t.Fatalf("removed endpoint attributed to replacement: %+v", a)
	}
}

func TestNoteAnchorAfterSelectionPublication(t *testing.T) {
	f, _, _ := pointerOnFixture(t, nil)
	replacement := ptychild.NewFakeChild(nil)
	f.con.Attach("c2", "replacement", replacement)
	held, release := holdFirstPublish(t, f.con, "select")
	selected := make(chan error, 1)
	go func() { _, err := f.con.selectActor("c2", true, arrivalOrdinary); selected <- err }()
	waitClosed(t, "selection before commit", held)
	// The existing hook parks selection before presenter publication. Release
	// and join the complete selection, then verify its retained frame's anchor.
	// This is a positive publication control, not a proof that anchoring blocks.
	release()
	select {
	case err := <-selected:
		if err != nil {
			t.Fatal(err)
		}
	case <-testwait.After(t):
		t.Fatal("selection did not finish")
	}
	a := f.con.anchorNote(broadcast.Note{Box: cb(0, 0, 2, 1)}, 80, 24)
	if len(a.Path) != 1 || a.Path[0] != "pane replacement" {
		t.Fatalf("selected frame's anchor: %+v", a)
	}
}
