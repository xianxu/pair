package terminal

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderDiffAndIsolation(t *testing.T) {
	a := testFrame(5, 2)
	a.Cells[0] = Cell{Content: "A", Width: 1}
	first, err := Render(Frame{}, a)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "A") || !strings.Contains(string(first), "\x1b[r") {
		t.Fatalf("missing full reset %q", first)
	}
	same, err := Render(a, a.Clone())
	if err != nil || len(same) != 0 {
		t.Fatalf("unchanged repaint %q %v", same, err)
	}
	b := a.Clone()
	b.Cells[3].Content = "B"
	diff, err := Render(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(diff), "A") || !strings.Contains(string(diff), "B") || strings.Contains(string(diff), "\x1b[2J") {
		t.Fatalf("not differential %q", diff)
	}
	b.Cells[3].Content = "abcdef"
	if _, err := Render(a, b); err == nil {
		t.Fatal("multi-cell content escaped rectangle")
	}
}

func TestStyledRowsAllowOnlyTextStylesAndLinks(t *testing.T) {
	rows, err := StyledRows("\x1b[1;31mA\x1b[0m界\r\n\x1b]8;id=one;https://x/a;b\x1b\\Z\x1b]8;;\x1b\\", 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Content != "A" || rows[1].Width != 2 || rows[2].Width != 0 || rows[4].Link.URL != "https://x/a;b" {
		t.Fatalf("bad cells %+v", rows)
	}
	for _, bad := range []string{"\x1b[2J", "x\b", "x\rY", "\x1b]52;c;eA==\a", "\x1b[31", "\x1b]8;;url", "\x1b[?1m", "\x1b[31mA\x1b[0;0H"} {
		if _, err := StyledRows(bad, 4, 2); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestComposePreservesChildAndOwnsBottomRow(t *testing.T) {
	child := testFrame(4, 1)
	child.Cells[0].Content = "A"
	row, err := StyledRows("tab", 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	composed, err := Compose(child, Geometry{Cols: 4, Rows: 2}, row)
	if err != nil {
		t.Fatal(err)
	}
	if composed.Cells[0].Content != "A" || composed.Cells[4].Content != "t" || child.Geometry.Rows != 1 {
		t.Fatal("incorrect composition")
	}
	composed.Cells[0].Content = "X"
	if child.Cells[0].Content != "A" {
		t.Fatal("composition aliases child")
	}
	if _, err := Compose(child, Geometry{Cols: 3, Rows: 2}, row); err == nil {
		t.Fatal("accepted incompatible geometry")
	}
	if _, err := PanelFrame(Geometry{Cols: 4, Rows: 2}, composed.Cells, Cursor{Visible: false}); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkRender(b *testing.B) {
	for _, g := range []Geometry{{Cols: 80, Rows: 24}, {Cols: 240, Rows: 80}} {
		b.Run(fmt.Sprintf("%dx%d", g.Cols, g.Rows), func(b *testing.B) {
			f := testFrame(g.Cols, g.Rows)
			b.Run("unchanged", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := Render(f, f); err != nil {
						b.Fatal(err)
					}
				}
			})
			next := f.Clone()
			next.Cells[0].Content = "X"
			b.Run("single_cell", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := Render(f, next); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
