package vt

import (
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

// Literal erase oracles include the cursor cell and ignore scrolling margins.
func TestPairEraseDisplay(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mode, x, y int
		rows       [3]string
	}{
		{"below", 0, 1, 1, [3]string{"ABCD", "E   ", "    "}},
		{"above", 1, 1, 1, [3]string{"    ", "  GH", "IJKL"}},
		{"all", 2, 1, 1, [3]string{"    ", "    ", "    "}},
		{"saved-only", 3, 1, 1, [3]string{"ABCD", "EFGH", "IJKL"}},
		{"below-first", 0, 0, 0, [3]string{"    ", "    ", "    "}},
		{"above-first", 1, 0, 0, [3]string{" BCD", "EFGH", "IJKL"}},
		{"below-last", 0, 3, 2, [3]string{"ABCD", "EFGH", "IJK "}},
		{"above-last", 1, 3, 2, [3]string{"    ", "    ", "    "}},
		{"unknown", 9, 1, 1, [3]string{"ABCD", "EFGH", "IJKL"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEmulator(4, 3)
			defer e.Close()
			e.Scrollback().Push(uv.Line{{Content: "history", Width: 1}})
			e.WriteString("\x1b[44;32;1m\x1b]8;id=old;https://example.com\aABCD\x1b[2;1HEFGH\x1b[3;1HIJKL\x1b]8;;\a\x1b[2;3r\x1b[41m")
			e.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[%dJ", tc.y+1, tc.x+1, tc.mode))
			if e.CursorPosition() != uv.Pos(tc.x, tc.y) {
				t.Fatalf("cursor moved: %v", e.CursorPosition())
			}
			for y, row := range tc.rows {
				for x, want := range row {
					c := e.CellAt(x, y)
					wantContent := string(want)
					if want == ' ' {
						wantContent = ""
					}
					if c.Content != wantContent {
						t.Fatalf("cell %d,%d got %q want %q", x, y, c.Content, string(want))
					}
					bg := ansi.IndexedColor(4)
					if want == ' ' {
						bg = ansi.IndexedColor(1)
						if c.Style.Attrs != 0 || c.Style.Fg != nil || c.Link.URL != "" {
							t.Fatalf("erased cell retained attributes: %+v", c)
						}
					}
					gotStyle, wantStyle := uv.Style{Bg: c.Style.Bg}, uv.Style{Bg: bg}
					if !gotStyle.Equal(&wantStyle) {
						t.Fatalf("cell %d,%d bg=%v want=%v", x, y, c.Style.Bg, bg)
					}
				}
			}
			if tc.mode == 3 && e.ScrollbackLen() != 0 {
				t.Fatal("ED3 retained history")
			}
			if tc.mode != 3 && e.ScrollbackLen() == 0 {
				t.Fatal("ED erased history")
			}
		})
	}
}

func TestPairEraseLineAndCharacters(t *testing.T) {
	for _, tc := range []struct{ command, want string }{
		{"\x1b[K", "A   "}, {"\x1b[1K", "  CD"}, {"\x1b[2K", "    "}, {"\x1b[X", "A CD"}, {"\x1b[2X", "A  D"}, {"\x1b[9K", "ABCD"},
	} {
		e := NewEmulator(4, 2)
		e.WriteString("ABCD\x1b[1;2H\x1b[41m" + tc.command)
		var row strings.Builder
		for x := 0; x < 4; x++ {
			c := e.CellAt(x, 0)
			if c.Width == 1 && c.Content == "" {
				row.WriteByte(' ')
			} else {
				row.WriteString(c.Content)
			}
			gotStyle, wantStyle := uv.Style{Bg: c.Style.Bg}, uv.Style{Bg: ansi.IndexedColor(1)}
			if tc.want[x] == ' ' && !gotStyle.Equal(&wantStyle) {
				t.Fatalf("%q erased bg=%v", tc.command, c.Style.Bg)
			}
		}
		if row.String() != tc.want || e.CursorPosition() != uv.Pos(1, 0) {
			t.Fatalf("%q got %q cursor%v", tc.command, row.String(), e.CursorPosition())
		}
		e.Close()
	}
}
func TestPairEraseLargeCount(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	e.WriteString("ABCD\x1b[1;2H\x1b[100000000X")
	if e.CellAt(0, 0).Content != "A" || e.CellAt(3, 0).Content != "" {
		t.Fatal("ECH clipped incorrectly")
	}
}
