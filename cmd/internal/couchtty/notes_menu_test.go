package couchtty

import (
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func menuFrame(t *testing.T, cols, rows int) terminal.Frame {
	t.Helper()
	cells, err := terminal.StyledRows(strings.Repeat("x", cols*rows), cols, rows)
	if err != nil {
		t.Fatal(err)
	}
	f, err := terminal.PanelFrame(terminal.Geometry{Cols: cols, Rows: rows}, cells, terminal.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func rowString(f terminal.Frame, y int) string {
	var b strings.Builder
	for _, c := range f.Cells[y*f.Geometry.Cols : (y+1)*f.Geometry.Cols] {
		b.WriteString(c.Content)
	}
	return b.String()
}

var menuNotes = []broadcast.Note{
	{ID: 1, Color: 0, Text: "look here"},
	{ID: 2, Color: 3, Text: "a longer note that will need to wrap onto more than one line in the box"},
}

func TestNotesMenuPlacement(t *testing.T) {
	m := RenderNotesMenu(menuNotes, 80, 24)
	if m.W != notesMenuMaxWidth || m.X != 80-notesMenuMaxWidth {
		t.Fatalf("box %+v", m)
	}
	if m.Y+m.H != 23 {
		t.Fatalf("box must end right above the status row: %+v", m)
	}
	small := RenderNotesMenu(menuNotes, 40, 10)
	if small.W != 38 || small.X != 2 || small.Y+small.H != 9 || small.Y < 0 {
		t.Fatalf("narrow box %+v", small)
	}
	if RenderNotesMenu(nil, 80, 24).Open() {
		t.Fatal("no notes, no menu")
	}
	if RenderNotesMenu(menuNotes, 8, 24).Open() || RenderNotesMenu(menuNotes, 80, 4).Open() {
		t.Fatal("no room, no menu")
	}
}

func TestNotesMenuWrapsAndCaps(t *testing.T) {
	long := strings.Repeat("word ", 60)
	m := RenderNotesMenu([]broadcast.Note{{ID: 9, Text: long}}, 80, 24)
	if len(m.items) != 1 || m.items[0].Bottom-m.items[0].Top != notesMenuItemRows {
		t.Fatalf("items %+v", m.items)
	}
	last := m.lines[len(m.lines)-1].text
	if !strings.HasSuffix(last, "…") {
		t.Fatalf("a cut note ends in …: %q", last)
	}
	two := RenderNotesMenu(menuNotes, 80, 24)
	if got := two.items[1].Bottom - two.items[1].Top; got < 2 {
		t.Fatalf("the long note should wrap: %d rows", got)
	}
}

func TestNotesMenuSaysHowManyMore(t *testing.T) {
	var many []broadcast.Note
	for i := 1; i <= 20; i++ {
		many = append(many, broadcast.Note{ID: i, Text: "n"})
	}
	m := RenderNotesMenu(many, 80, 10) // room for 7 rows
	if m.H > 7 {
		t.Fatalf("box taller than the room: %d", m.H)
	}
	shown := len(m.items)
	if last := m.lines[len(m.lines)-1].text; last != " +"+strconv.Itoa(20-shown)+" more" {
		t.Fatalf("last line %q with %d shown", last, shown)
	}
}

func TestNotesMenuDrawAndHitTests(t *testing.T) {
	f := menuFrame(t, 80, 24)
	m := RenderNotesMenu(menuNotes, 80, 24)
	before := rowString(f, m.Y)
	out := m.Draw(f, 2, broadcast.NoteStyle{})
	header := rowString(out, m.Y)
	if !strings.Contains(header, "Notes") || !strings.HasSuffix(strings.TrimRight(header, " "), notesMenuClose) {
		t.Fatalf("header %q", header)
	}
	if !strings.Contains(rowString(out, m.items[0].Top), "look here") {
		t.Fatalf("item row %q", rowString(out, m.items[0].Top))
	}
	if rowString(out, 23) != rowString(f, 23) {
		t.Fatal("the menu covered the status row")
	}
	// The chip: two cells of the author's color right after a space.
	chip := out.Cells[m.items[1].Top*80+m.X+1]
	if chip.Style.Bg != broadcast.NoteChipColor(3, broadcast.NoteStyle{}) {
		t.Fatalf("chip %+v", chip)
	}
	// Hover: the hovered item's rows on the hover background, the other's not.
	if out.Cells[m.items[1].Top*80+m.X+6].Style.Bg != notesMenuHover || out.Cells[m.items[0].Top*80+m.X+6].Style.Bg != notesMenuBg {
		t.Fatal("hover highlight")
	}
	if m.ItemAt(m.X+10, m.items[0].Top) != 1 || m.ItemAt(m.X+10, m.items[1].Bottom-1) != 2 || m.ItemAt(m.X+10, m.Y) != 0 || m.ItemAt(m.X-1, m.items[0].Top) != 0 {
		t.Fatal("ItemAt")
	}
	if !m.CloseAt(m.X+m.W-1, m.Y) || m.CloseAt(m.X+2, m.Y) || m.CloseAt(m.X+m.W-1, m.Y+1) {
		t.Fatal("CloseAt")
	}
	if !m.Contains(m.X, m.Y) || m.Contains(m.X-1, m.Y) || m.Contains(m.X, m.Y+m.H) {
		t.Fatal("Contains")
	}
	if rowString(f, m.Y) != before {
		t.Fatal("input frame mutated")
	}
}

// Note text is drawn as text, never as terminal control.
func TestNotesMenuTextIsLiteral(t *testing.T) {
	f := menuFrame(t, 80, 24)
	m := RenderNotesMenu([]broadcast.Note{{ID: 1, Text: "a\x1b[2Jb"}}, 80, 24)
	out := m.Draw(f, 0, broadcast.NoteStyle{})
	for _, c := range out.Cells {
		if strings.ContainsRune(c.Content, 0x1b) {
			t.Fatal("an escape reached a cell")
		}
	}
	if err := out.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNotesMenuCutsWideCharactersAtItsEdge(t *testing.T) {
	cells, _ := terminal.StyledRows(strings.Repeat("字", 40), 80, 24)
	f, _ := terminal.PanelFrame(terminal.Geometry{Cols: 80, Rows: 24}, cells, terminal.Cursor{})
	m := RenderNotesMenu(menuNotes, 80, 24)
	m.X, m.W = 31, 49 // an odd edge, through the middle of a wide character
	out := m.Draw(f, 0, broadcast.NoteStyle{})
	if err := out.Validate(); err != nil {
		t.Fatal(err)
	}
}
