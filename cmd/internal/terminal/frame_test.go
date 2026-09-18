package terminal

import (
	"image/color"
	"testing"
)

func testFrame(cols, rows int) Frame {
	f := Frame{EndpointID: "a", Geometry: Geometry{Cols: cols, Rows: rows}, Cursor: Cursor{Visible: true}}
	f.Cells = make([]Cell, cols*rows)
	for i := range f.Cells {
		f.Cells[i] = Cell{Content: " ", Width: 1}
	}
	return f
}

func TestGeometryRejectsInvalidBeforeAllocation(t *testing.T) {
	for _, g := range []Geometry{{}, {Cols: -1, Rows: 2}, {Cols: 262145, Rows: 1}, {Cols: 1 << 62, Rows: 4}} {
		if g.Validate() == nil {
			t.Fatalf("accepted %+v", g)
		}
	}
	if (Geometry{Cols: 512, Rows: 512}).Validate() != nil {
		t.Fatal("ceiling rejected")
	}
}

func TestFrameCloneOwnsMutableColorAndCells(t *testing.T) {
	f := testFrame(2, 1)
	c := &color.RGBA{R: 12, A: 255}
	f.Cells[0].Style.Fg = c
	copy := f.Clone()
	c.R = 99
	f.Cells[0].Content = "X"
	r, _, _, _ := copy.Cells[0].Style.Fg.RGBA()
	if copy.Cells[0].Content != " " || r != 12*257 {
		t.Fatalf("aliased clone: %+v", copy)
	}
}

func TestFrameRejectsDrawingInjectionAndBrokenWideCells(t *testing.T) {
	for _, cell := range []Cell{{Content: "\x1b[2J", Width: 1}, {Content: "x", Width: 0}, {Content: "界", Width: 2}, {Content: "x", Width: 1, Link: struct{ URL, Params string }{URL: "x\x1b\\"}}} {
		f := testFrame(1, 1)
		f.Cells[0] = cell
		if f.Validate() == nil {
			t.Fatalf("accepted %+v", cell)
		}
	}
}

// Shape 0 hands shape and blink to the parent, so a blinking default is a
// second spelling of the same visible state and is refused (#283).
func TestFrameRejectsBlinkingDefaultCursor(t *testing.T) {
	f := testFrame(2, 1)
	f.Cursor = Cursor{Visible: true, Blink: true}
	if f.Validate() == nil {
		t.Fatal("accepted a blinking default cursor")
	}
	f.Cursor.Shape = 1
	if err := f.Validate(); err != nil {
		t.Fatalf("rejected a blinking block: %v", err)
	}
}
