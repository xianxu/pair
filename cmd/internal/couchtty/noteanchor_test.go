package couchtty

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func cb(col, row, w, h int) broadcast.CellBox {
	return broadcast.CellBox{Col: col, Row: row, W: w, H: h}
}

// screenFrame is a frame of cols×len(lines) cells; a rune in wide is two
// cells wide.
func screenFrame(cols int, lines ...string) terminal.Frame {
	f := terminal.Frame{Geometry: terminal.Geometry{Cols: cols, Rows: len(lines)}, Cells: make([]terminal.Cell, cols*len(lines))}
	for row, line := range lines {
		col := 0
		for _, r := range line {
			if col >= cols {
				break
			}
			w := 1
			if strings.ContainsRune("漢字💭", r) {
				w = 2
			}
			f.Cells[row*cols+col] = terminal.Cell{Content: string(r), Width: w}
			if w == 2 && col+1 < cols {
				f.Cells[row*cols+col+1] = terminal.Cell{}
			}
			col += w
		}
		for ; col < cols; col++ {
			f.Cells[row*cols+col] = terminal.Cell{Content: " ", Width: 1}
		}
	}
	return f
}

// A 40×10 Couch screen: hosted output above and Couch's own controls and
// tab chips on the last row. Hosted text carries no pane/file attribution.
func anchorScreen() (terminal.Frame, ScreenLayout) {
	f := screenFrame(40,
		"> fix the header color",
		"⏺ Edited header.css",
		"  12 lines changed",
		"",
		"",
		"> ",
		"Please also check the 漢字 footer",
		"and the nav.",
		"draft-pair-3.md  1,1  All",
		"LIVE ⏸ 👆  pair:1  pair:3 ●  Chrome ⏸",
	)
	f.ChromeRows = 1
	l := ScreenLayout{
		Chips: []ChipSpan{{Start: 10, End: 16}, {Start: 18, End: 26}, {Start: 28, End: 36}},
		Slot:  "pair:3",
	}
	return f, l
}

func TestAnchorForEachPartOfCouchsScreen(t *testing.T) {
	f, l := anchorScreen()
	for _, c := range []struct {
		name string
		box  broadcast.CellBox
		kind string
		path string
		text string
	}{
		{"agent pane", cb(0, 0, 24, 3), broadcast.AnchorScreen, "pane pair:3", "> fix the header color\n⏺ Edited header.css\n  12 lines changed"},
		{"draft lines", cb(0, 6, 20, 2), broadcast.AnchorScreen, "pane pair:3", "Please also check th\nand the nav."},
		{"one draft line", cb(10, 6, 8, 1), broadcast.AnchorScreen, "pane pair:3", "o check"},
		{"draft statusline", cb(0, 8, 16, 1), broadcast.AnchorScreen, "pane pair:3", "draft-pair-3.md"},
		{"tab bar", cb(10, 9, 16, 1), broadcast.AnchorTabBar, "tab bar › tabs: pair:1, pair:3 ●", "pair:1  pair:3 ●"},
		{"status line", cb(0, 9, 10, 1), broadcast.AnchorStatusLine, "status line", "LIVE ⏸ 👆"},
	} {
		a := AnchorFor(c.box, f, l)
		if a.Kind != c.kind || strings.Join(a.Path, " › ") != c.path || a.Text != c.text {
			t.Errorf("%s: %q %q %q, want %q %q %q", c.name, a.Kind, strings.Join(a.Path, " › "), a.Text, c.kind, c.path, c.text)
		}
	}
}

// A box above Couch's last row names only the slot and is AnchorScreen.
func TestAnchorForScreenAndUnknownSlot(t *testing.T) {
	f, l := anchorScreen()
	a := AnchorFor(cb(0, 0, 10, 2), f, l)
	if a.Kind != broadcast.AnchorScreen || strings.Join(a.Path, " › ") != "pane pair:3" || a.Text != "> fix the\n⏺ Edited h" {
		t.Errorf("%+v", a)
	}
	l.Slot = ""
	if a := AnchorFor(cb(0, 0, 10, 2), f, l); a.Path[0] != "pane the slot in view" {
		t.Errorf("no active chip: %+v", a)
	}
}

func TestTextUnderIsBoundedAndKeepsWideCharacters(t *testing.T) {
	f, _ := anchorScreen()
	if got := TextUnder(f, cb(22, 6, 6, 1)); got != "漢字 f" {
		t.Errorf("wide text %q", got)
	}
	long := screenFrame(300, strings.Repeat("é", 300), strings.Repeat("é", 300), strings.Repeat("é", 300), strings.Repeat("é", 300))
	got := TextUnder(long, cb(0, 0, 300, 4))
	if len(got) > maxAnchorText || !utf8.ValidString(got) {
		t.Errorf("text %d bytes, valid %v", len(got), utf8.ValidString(got))
	}
	if got := TextUnder(f, cb(0, 3, 10, 2)); got != "" {
		t.Errorf("blank rows: %q", got)
	}
}
