package vt

import "testing"

func TestPairPrimaryResizeReflowsVisibleLogicalLines(t *testing.T) {
	for _, tc := range []struct {
		name, wire                           string
		width, height, nextWidth, nextHeight int
		lines                                []string
		wrapped                              []bool
		history                              string
		x, y                                 int
	}{
		{"one-column-growth", "ABC", 1, 2, 8, 2, []string{"BC", ""}, []bool{true, false}, "A", 2, 0},
		{"full-phantom-growth", "ABCDEF", 3, 2, 8, 2, []string{"ABCDEF", ""}, []bool{false, false}, "", 6, 0},
		{"hard-boundaries", "AB\r\nCD", 4, 2, 8, 2, []string{"AB", "CD"}, []bool{false, false}, "", 2, 1},
		{"narrow-scroll", "ABCDE", 5, 2, 2, 2, []string{"CD", "E"}, []bool{true, true}, "AB", 1, 1},
		{"early-wide-gap", "AAA界Z", 4, 2, 8, 2, []string{"AAA界Z", ""}, []bool{false, false}, "", 6, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := historyEmulator(t, tc.width, tc.height)
			e.WriteString(tc.wire)
			e.Resize(tc.nextWidth, tc.nextHeight)
			for y, want := range tc.lines {
				got := ""
				for x := 0; x < e.Width(); x++ {
					got += e.CellAt(x, y).Content
				}
				if got != want {
					t.Errorf("row%d=%q want%q", y, got, want)
				}
				if e.RowMetadata(y).Wrapped != tc.wrapped[y] {
					t.Errorf("row%d wrap=%v", y, e.RowMetadata(y).Wrapped)
				}
			}
			got := ""
			for _, r := range e.HistorySnapshot().Rows {
				for _, c := range r.Cells {
					got += c.Content
				}
			}
			if got != tc.history {
				t.Errorf("history=%q want%q", got, tc.history)
			}
			if p := e.CursorPosition(); p.X != tc.x || p.Y != tc.y {
				t.Errorf("cursor=%v want%d,%d", p, tc.x, tc.y)
			}
			e.WriteString("!")
			if c := e.CellAt(tc.x, tc.y); c.Content != "!" {
				t.Errorf("next output=%#v", c)
			}
		})
	}
}
func TestPairAlternateResizeDoesNotReflow(t *testing.T) {
	e := historyEmulator(t, 3, 2)
	e.WriteString("\x1b[?1049hABCDEF")
	e.Resize(8, 2)
	if e.CellAt(0, 0).Content != "A" || e.CellAt(0, 1).Content != "D" {
		t.Fatal("alternate rows reflowed")
	}
}

func TestPairResizeClippedWideRestoresSource(t *testing.T) {
	e := historyEmulator(t, 4, 2)
	e.WriteString("界Z")
	e.Resize(1, 2)
	for y := 0; y < 2; y++ {
		if c := e.CellAt(0, y); c.Width != 1 {
			t.Fatalf("invalid clipped cell %#v", c)
		}
	}
	e.Resize(4, 2)
	if c := e.CellAt(0, 0); c.Content != "界" || c.Width != 2 {
		t.Fatalf("source glyph lost %#v", c)
	}
	if e.CellAt(2, 0).Content != "Z" {
		t.Fatal("following source lost")
	}
}

func TestPairResizeClippedWideScrollAndOverwrite(t *testing.T) {
	e := historyEmulator(t, 4, 2)
	e.WriteString("界Z")
	e.Resize(1, 2)
	e.WriteString("\r\nQ")
	h := e.HistorySnapshot()
	if len(h.Rows) == 0 || len(h.Rows[0].Cells) == 0 || h.Rows[0].Cells[0].Content != "界" {
		t.Fatalf("clipped history lost %#v", h)
	}
	e2 := historyEmulator(t, 4, 2)
	e2.WriteString("界Z")
	e2.Resize(1, 2)
	e2.WriteString("\x1b[1;1HX")
	e2.Resize(4, 2)
	if e2.CellAt(0, 0).Content != "X" {
		t.Fatal("overwritten clipped glyph resurrected")
	}
}

func TestPairResizePreservesPendingWrap(t *testing.T) {
	e := historyEmulator(t, 3, 2)
	e.WriteString("ABCDEF")
	e.Resize(6, 2)
	e.WriteString("G")
	if e.CellAt(5, 0).Content != "F" || e.CellAt(0, 1).Content != "G" || !e.RowMetadata(1).Wrapped {
		t.Fatalf("pending wrap lost %q", e.String())
	}
}
