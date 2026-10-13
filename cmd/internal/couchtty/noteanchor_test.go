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

// draftLines is the draft on disk: the screen shows lines 12–13.
var draftLines = append(append(make([]string, 0, 14), "# Plan", "", "- [ ] one", "", "## Review", "", "the header", "", "", "", ""),
	"Please also check the 漢字 footer", "and the nav.")

// A 40×10 Couch screen: the slot's zellij session above (an agent pane on
// rows 0–5, the draft nvim on rows 6–8 with its statusline on 8), and the
// last row: the controls, then the tab bar's chips.
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
	l := ScreenLayout{
		Chips: []ColumnSpan{{10, 16}, {18, 26}, {28, 36}},
		Slot:  "pair:3",
		Agent: "claude",
		Panes: []PaneRect{
			{Role: PaneAgent, X: 0, Y: 0, W: 40, H: 6},
			{Role: PaneDraft, X: 0, Y: 6, W: 40, H: 3, File: "/data/repos/x/draft-pair-3.md", Lines: draftLines},
		},
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
		{"agent pane", cb(0, 0, 24, 3), broadcast.AnchorPane, "pane pair:3 › agent (claude)", "> fix the header color\n⏺ Edited header.css\n  12 lines changed"},
		{"draft lines", cb(0, 6, 20, 2), broadcast.AnchorPane, "pane pair:3 › draft nvim › draft-pair-3.md, lines 12–13 › section: Review", "Please also check th\nand the nav."},
		{"one draft line", cb(10, 6, 8, 1), broadcast.AnchorPane, "pane pair:3 › draft nvim › draft-pair-3.md, line 12 › section: Review", "o check"},
		{"draft statusline", cb(0, 8, 16, 1), broadcast.AnchorPane, "pane pair:3 › draft nvim › statusline", "draft-pair-3.md"},
		{"tab bar", cb(10, 9, 16, 1), broadcast.AnchorTabBar, "tab bar › tabs: pair:1, pair:3 ●", "pair:1  pair:3 ●"},
		{"status line", cb(0, 9, 10, 1), broadcast.AnchorStatusLine, "status line", "LIVE ⏸ 👆"},
	} {
		a := AnchorFor(c.box, f, l)
		if a.Kind != c.kind || strings.Join(a.Path, " › ") != c.path || a.Text != c.text {
			t.Errorf("%s: %q %q %q, want %q %q %q", c.name, a.Kind, strings.Join(a.Path, " › "), a.Text, c.kind, c.path, c.text)
		}
	}
}

// Before the pane layout is known (the anchor taken as a note opens), a
// box above the last row names the slot and is AnchorScreen.
func TestAnchorForWithoutPanes(t *testing.T) {
	f, l := anchorScreen()
	l.Panes = nil
	a := AnchorFor(cb(0, 0, 10, 2), f, l)
	if a.Kind != broadcast.AnchorScreen || strings.Join(a.Path, " › ") != "pane pair:3" || a.Text != "> fix the\n⏺ Edited h" {
		t.Errorf("%+v", a)
	}
	l.Slot = ""
	if a := AnchorFor(cb(0, 0, 10, 2), f, l); a.Path[0] != "pane the slot in view" {
		t.Errorf("no active chip: %+v", a)
	}
	// A draft whose file is unknown stops at "draft nvim"; one whose text
	// isn't on disk (unsaved) names the file only.
	f, l = anchorScreen()
	l.Panes[1].File = ""
	if a := AnchorFor(cb(0, 6, 10, 1), f, l); strings.Join(a.Path, " › ") != "pane pair:3 › draft nvim" {
		t.Errorf("no file: %+v", a)
	}
	f, l = anchorScreen()
	l.Panes[1].Lines = []string{"something else"}
	if a := AnchorFor(cb(0, 6, 10, 1), f, l); strings.Join(a.Path, " › ") != "pane pair:3 › draft nvim › draft-pair-3.md" {
		t.Errorf("unsaved: %+v", a)
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

func TestLocateFindsScreenRowsInTheFile(t *testing.T) {
	lines := []string{"alpha", "a long line that the editor wraps onto two rows", "beta", "alpha", "gamma"}
	for _, c := range []struct {
		rows        []string
		first, last int
		ok          bool
	}{
		{[]string{"beta"}, 2, 2, true},
		{[]string{"alpha", "gamma"}, 3, 4, true}, // the first alpha isn't followed by gamma… it is, later: either answer holds the rows
		{[]string{"a long line that", "wraps onto two rows"}, 1, 1, true},
		{[]string{"delta"}, 0, 0, false},
		{nil, 0, 0, false},
	} {
		first, last, ok := locate(c.rows, lines)
		if ok != c.ok || (ok && (first > last || !strings.Contains(lines[first], c.rows[0]) || !strings.Contains(lines[last], c.rows[len(c.rows)-1]))) {
			t.Errorf("locate(%q) = %d %d %v", c.rows, first, last, ok)
		}
	}
	if got := sectionOf([]string{"# Top", "x", "## Sub", "y"}, 3); got != "Sub" {
		t.Errorf("section %q", got)
	}
	if got := sectionOf([]string{"x"}, 0); got != "" {
		t.Errorf("no section %q", got)
	}
}
