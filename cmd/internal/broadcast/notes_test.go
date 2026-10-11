package broadcast

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestNoteBoardLimits(t *testing.T) {
	var b NoteBoard
	for i := 0; i < MaxNotesPerColor; i++ {
		if _, err := b.Add(Note{Col: i * 2, Row: 0, Color: 3, Text: "x"}); err != nil {
			t.Fatalf("note %d: %v", i, err)
		}
	}
	if _, err := b.Add(Note{Col: 40, Row: 0, Color: 3, Text: "x"}); err != ErrNoteColorFull {
		t.Fatalf("9th note in one color: %v, want ErrNoteColorFull", err)
	}
	if _, err := b.Add(Note{Col: 40, Row: 0, Color: 4, Text: "x"}); err != nil {
		t.Fatalf("another color still has room: %v", err)
	}
}

func TestNoteBoardOverlap(t *testing.T) {
	var b NoteBoard
	if _, err := b.Add(Note{Col: 10, Row: 5, Text: "a"}); err != nil {
		t.Fatal(err)
	}
	for _, col := range []int{9, 10, 11} {
		if _, err := b.Add(Note{Col: col, Row: 5, Text: "b"}); err != ErrNoteOverlap {
			t.Errorf("col %d: %v, want ErrNoteOverlap", col, err)
		}
	}
	if _, err := b.Add(Note{Col: 12, Row: 5, Text: "b"}); err != nil {
		t.Errorf("adjacent note: %v", err)
	}
	if _, err := b.Add(Note{Col: 10, Row: 6, Text: "b"}); err != nil {
		t.Errorf("the row below: %v", err)
	}
}

func TestNoteBoardResolveIdempotent(t *testing.T) {
	var b NoteBoard
	n, _ := b.Add(Note{Col: 0, Row: 0, Text: "a"})
	if !b.Resolve(n.ID) || b.Resolve(n.ID) || b.Resolve(999) {
		t.Fatal("resolve reports true once, then false")
	}
	if len(b.List()) != 0 {
		t.Fatal("resolved note still listed")
	}
}

func TestNoteBoardIDsAndCopies(t *testing.T) {
	var b NoteBoard
	a, _ := b.Add(Note{Col: 0, Row: 0, Text: "a"})
	b.Resolve(a.ID)
	c, _ := b.Add(Note{Col: 0, Row: 0, Text: "c"})
	if c.ID <= a.ID {
		t.Fatalf("ids must keep increasing past a resolve: %d then %d", a.ID, c.ID)
	}
	list := b.List()
	list[0].Text = "changed"
	if b.List()[0].Text != "c" {
		t.Fatal("List must return a copy")
	}
	b.Clear()
	if len(b.List()) != 0 {
		t.Fatal("Clear leaves notes")
	}
	d, _ := b.Add(Note{Col: 0, Row: 0, Text: "d"})
	if d.ID <= c.ID {
		t.Fatal("ids keep increasing past a clear")
	}
}

const okNote = `{"cols":80,"rows":24,"col":3,"row":4,"color":2,"text":"hi\nthere"}`

func TestParseNote(t *testing.T) {
	n, err := ParseNote([]byte(okNote))
	if err != nil {
		t.Fatal(err)
	}
	want := NoteRequest{Cols: 80, Rows: 24, Note: Note{Col: 3, Row: 4, Color: 2, Text: "hi\nthere"}}
	if n != want {
		t.Fatalf("got %+v, want %+v", n, want)
	}
	if _, err := ParseNote([]byte(fmt.Sprintf(`{"cols":80,"rows":24,"col":3,"row":4,"color":0,"text":%q}`, strings.Repeat("é", 512)))); err != nil {
		t.Fatalf("1024 bytes must be accepted: %v", err)
	}
}

// noteRefusals is every malformed or out-of-bounds note body; FuzzParseNote
// seeds from it.
var noteRefusals = []string{
	`{"cols":80,"rows":24,"col":3,"row":4,"color":2,"text":"hi","x":1}`,
	`{"cols":80,"rows":24,"col":"3","row":4,"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":1.5,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":6,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":-1,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":79,"row":4,"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":0,"row":23,"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":3,"row":24,"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":-1,"row":4,"color":0,"text":"hi"}`,
	`{"cols":0,"rows":24,"col":3,"row":4,"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":0,"text":""}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":0}`,
	`{"rows":24,"col":3,"row":4,"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":0,"text":"hi\u001b[2J"}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":0,"text":"hi‮evil"}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":0,"text":"zero​width"}`,
	"{\"cols\":80,\"rows\":24,\"col\":3,\"row\":4,\"color\":0,\"text\":\"bad\xff\"}",
	`{"cols":80,"rows":24,"col":3,"row":4,"color":0,"text":"hi"} {}`,
	`[]`,
	``,
}

func TestParseNoteRefusals(t *testing.T) {
	long := fmt.Sprintf(`{"cols":80,"rows":24,"col":3,"row":4,"color":0,"text":%q}`, strings.Repeat("é", 513))
	for _, body := range append(noteRefusals, long) {
		_, err := ParseNote([]byte(body))
		if err == nil {
			t.Errorf("accepted %q", body)
			continue
		}
		if strings.Contains(err.Error(), "hi") || strings.Contains(err.Error(), "evil") {
			t.Errorf("error echoes input: %v", err)
		}
	}
}

func TestParseResolve(t *testing.T) {
	id, err := ParseResolve([]byte(`{"id":3}`))
	if err != nil || id != 3 {
		t.Fatalf("got %d %v", id, err)
	}
	for _, body := range []string{`{"id":"3"}`, `{}`, `{"id":3,"x":1}`, `{"id":-1}`, `{"id":1.5}`, `{"id":3} 1`} {
		if _, err := ParseResolve([]byte(body)); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}

// FuzzParseNote: never panics; anything accepted satisfies every rule,
// checked here independently of the parser; no error echoes input.
func FuzzParseNote(f *testing.F) {
	f.Add(okNote)
	for _, s := range noteRefusals {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, body string) {
		req, err := ParseNote([]byte(body))
		n := req.Note
		if err != nil {
			for _, known := range []error{ErrNoteShape, ErrNoteRange, ErrNoteText} {
				if errors.Is(err, known) {
					return
				}
			}
			t.Fatalf("unexpected error %v", err)
		}
		if n.Col < 0 || n.Row < 0 || n.Col+1 >= req.Cols || n.Row >= req.Rows || n.Color < 0 || n.Color >= PointerColors {
			t.Fatalf("accepted out-of-range %+v", n)
		}
		if n.Text == "" || len(n.Text) > MaxNoteBytes || !utf8.ValidString(n.Text) {
			t.Fatalf("accepted bad text %q", n.Text)
		}
		for _, r := range n.Text {
			if (unicode.IsControl(r) && r != '\n' && r != '\t') || unicode.Is(unicode.Cf, r) {
				t.Fatalf("accepted control or format rune %U", r)
			}
		}
	})
}

func TestNoteVisible(t *testing.T) {
	n := Note{Col: 10, Row: 5}
	cases := []struct {
		cols, rows int
		want       bool
	}{
		{80, 24, true},
		{12, 24, true},
		{11, 24, false}, // second cell off the grid
		{80, 5, false},  // row off the grid
		{80, 6, false},  // now the last row, inside the guard
	}
	for _, c := range cases {
		if got := NoteVisible(n, c.cols, c.rows); got != c.want {
			t.Errorf("%dx%d: %v, want %v", c.cols, c.rows, got, c.want)
		}
	}
	if !NoteVisible(Note{Col: StatusGuardCols, Row: 5}, 80, 6) {
		t.Error("the last row past the guard is visible")
	}
}

func cellsText(f terminal.Frame, row int) string {
	var b strings.Builder
	for _, c := range f.Cells[row*f.Geometry.Cols : (row+1)*f.Geometry.Cols] {
		b.WriteString(c.Content)
	}
	return b.String()
}

func TestDrawNotes(t *testing.T) {
	f := textFrame(t, 10, 3, "abcdefghij\n漢字漢字漢\nxyz", "")
	out := DrawNotes(f, []Note{{Col: 2, Row: 0}}, NoteStyle{})
	if got := cellsText(out, 0); got != "ab💭efghij" {
		t.Fatalf("row 0: %q", got)
	}
	c := out.Cells[2]
	if c.Content != NoteGlyph || c.Width != 2 || out.Cells[3].Content != "" || out.Cells[3].Width != 0 {
		t.Fatalf("glyph cells: %+v %+v", c, out.Cells[3])
	}
	if cellsText(f, 0) != "abcdefghij" {
		t.Fatal("input frame mutated")
	}
	if same := DrawNotes(f, nil, NoteStyle{}); &same.Cells[0] != &f.Cells[0] {
		t.Fatal("no notes must return the frame uncopied")
	}
	if same := DrawNotes(f, []Note{{Col: 9, Row: 0}}, NoteStyle{}); &same.Cells[0] != &f.Cells[0] {
		t.Fatal("only hidden notes must return the frame uncopied")
	}
}

func TestDrawNotesCutsWideCharacters(t *testing.T) {
	// Row 1 is 漢字漢字漢: wide characters start at even columns.
	f := textFrame(t, 10, 3, "\n漢字漢字漢\n", "")
	// At col 3 the note covers the right half of 字 (cols 2-3) and the left
	// half of 漢 (cols 4-5): both leftovers become spaces.
	out := DrawNotes(f, []Note{{Col: 3, Row: 1}}, NoteStyle{})
	if got := cellsText(out, 1); got != "漢 💭 字漢" {
		t.Fatalf("row 1: %q", got)
	}
	for _, i := range []int{1*10 + 2, 1*10 + 5} {
		if out.Cells[i].Width != 1 || out.Cells[i].Content != " " {
			t.Fatalf("leftover half at %d: %+v", i, out.Cells[i])
		}
	}
}

func TestDrawNotesBlankKeepsStyle(t *testing.T) {
	// The blanked half sits in the glow, which sets its background (M3); the
	// rest of its style, here the foreground, is its own.
	f := textFrame(t, 10, 3, "\n\x1b[34m漢字漢字漢\x1b[0m\n", "")
	fg := f.Cells[1*10+2].Style.Fg
	if fg == nil {
		t.Fatal("fixture has no foreground")
	}
	out := DrawNotes(f, []Note{{Col: 3, Row: 1}}, NoteStyle{})
	if c := out.Cells[1*10+2]; c.Style.Fg != fg || c.Content != " " {
		t.Fatalf("blanked half lost its style: %+v", c)
	}
}

// Property: over random rows of narrow and wide characters and random note
// sets, the drawing doesn't depend on note order, never leaves half a wide
// character, and touches nothing outside the notes and their neighbours.
func TestDrawNotesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(447))
	const cols, rows = 12, 4
	for iter := 0; iter < 500; iter++ {
		var body strings.Builder
		for r := 0; r < rows; r++ {
			if r > 0 {
				body.WriteString("\n")
			}
			for w := 0; w < cols; {
				if w+2 <= cols && rng.Intn(2) == 0 {
					body.WriteString("字")
					w += 2
				} else {
					body.WriteString("a")
					w++
				}
			}
		}
		f := textFrame(t, cols, rows, body.String(), "")
		var board NoteBoard
		for k := 0; k < 4; k++ {
			board.Add(Note{Col: rng.Intn(cols), Row: rng.Intn(rows - 1), Text: "x"})
		}
		notes := board.List()
		a := DrawNotes(f, notes, NoteStyle{})
		rev := append([]Note(nil), notes...)
		for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
			rev[i], rev[j] = rev[j], rev[i]
		}
		b := DrawNotes(f, rev, NoteStyle{})
		for i := range a.Cells {
			if a.Cells[i].Content != b.Cells[i].Content || a.Cells[i].Width != b.Cells[i].Width {
				t.Fatalf("iter %d: order matters at cell %d", iter, i)
			}
		}
		near := map[int]bool{}
		for _, n := range notes {
			if !NoteVisible(n, cols, rows) {
				continue
			}
			for c := n.Col - 1; c <= n.Col+2; c++ {
				near[n.Row*cols+c] = true
			}
		}
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				i := r*cols + c
				cell := a.Cells[i]
				if cell.Width == 2 && (c+1 >= cols || a.Cells[i+1].Width != 0 || a.Cells[i+1].Content != "") {
					t.Fatalf("iter %d: wide cell at %d,%d lost its continuation", iter, c, r)
				}
				if cell.Width == 0 && (c == 0 || a.Cells[i-1].Width != 2) {
					t.Fatalf("iter %d: orphan continuation at %d,%d", iter, c, r)
				}
				if !near[i] && (cell.Content != f.Cells[i].Content || cell.Width != f.Cells[i].Width) {
					t.Fatalf("iter %d: cell %d,%d changed away from any note", iter, c, r)
				}
			}
		}
	}
}

// The renderer draws text only up to a row's used columns, so a note on an
// empty row or past its text widens them (#447 M1 smoke: the 💭 was missing
// on blank rows while marks, which are background only, showed).
func TestDrawNotesWidensUsedColumns(t *testing.T) {
	f := textFrame(t, 10, 3, "ab", "")
	f.Rows = []terminal.RowMetadata{{UsedColumns: 2}, {UsedColumns: 0}, {UsedColumns: 0}}
	out := DrawNotes(f, []Note{{Col: 5, Row: 1}, {Col: 0, Row: 0}}, NoteStyle{})
	if out.Rows[1].UsedColumns != 7 {
		t.Fatalf("empty row used columns %d, want 7", out.Rows[1].UsedColumns)
	}
	if out.Rows[0].UsedColumns != 2 {
		t.Fatalf("a note inside the text changed used columns to %d", out.Rows[0].UsedColumns)
	}
	if f.Rows[1].UsedColumns != 0 {
		t.Fatal("input row metadata mutated")
	}
}
