package broadcast

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestNoteBoardLimits(t *testing.T) {
	var b NoteBoard
	for i := 0; i < MaxNotesPerColor; i++ {
		if _, err := b.Add(Note{Box: CellBox{i * 2, 0, 2, 1}, Color: 3, Text: "x"}, 80, 24, 1); err != nil {
			t.Fatalf("note %d: %v", i, err)
		}
	}
	if _, err := b.Add(Note{Box: CellBox{40, 0, 2, 1}, Color: 3, Text: "x"}, 80, 24, 1); err != ErrNoteColorFull {
		t.Fatalf("9th note in one color: %v, want ErrNoteColorFull", err)
	}
	if _, err := b.Add(Note{Box: CellBox{40, 0, 2, 1}, Color: 4, Text: "x"}, 80, 24, 1); err != nil {
		t.Fatalf("another color still has room: %v", err)
	}
}

func TestNoteBoardOverlap(t *testing.T) {
	var b NoteBoard
	if _, err := b.Add(Note{Box: CellBox{10, 5, 2, 1}, Text: "a"}, 80, 24, 1); err != nil {
		t.Fatal(err)
	}
	for _, col := range []int{9, 10, 11} {
		if _, err := b.Add(Note{Box: CellBox{col, 5, 2, 1}, Text: "b"}, 80, 24, 1); err != ErrNoteOverlap {
			t.Errorf("col %d: %v, want ErrNoteOverlap", col, err)
		}
	}
	if _, err := b.Add(Note{Box: CellBox{12, 5, 2, 1}, Text: "b"}, 80, 24, 1); err != nil {
		t.Errorf("adjacent note: %v", err)
	}
	if _, err := b.Add(Note{Box: CellBox{10, 6, 2, 1}, Text: "b"}, 80, 24, 1); err != nil {
		t.Errorf("the row below: %v", err)
	}
}

// A note's pin (#488) is inside its box's top-right corner; when that is
// taken or on the controls, the next spot leftward, then below.
func TestNoteBoardPlacesThePin(t *testing.T) {
	var b NoteBoard
	pin := func(n Note, err error) [2]int {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return [2]int{n.Col, n.Row}
	}
	box := CellBox{20, 3, 10, 4}
	if got := pin(b.Add(Note{Box: box, Text: "a"}, 80, 24, 1)); got != [2]int{28, 3} {
		t.Errorf("first pin %v, want the top-right [28 3]", got)
	}
	if got := pin(b.Add(Note{Box: box, Text: "b"}, 80, 24, 1)); got != [2]int{26, 3} {
		t.Errorf("same box again %v, want the next spot left [26 3]", got)
	}
	narrow := CellBox{40, 3, 2, 3}
	pin(b.Add(Note{Box: narrow, Text: "c"}, 80, 24, 1))
	if got := pin(b.Add(Note{Box: narrow, Text: "d"}, 80, 24, 1)); got != [2]int{40, 4} {
		t.Errorf("a narrow box's second pin %v, want the row below [40 4]", got)
	}
	// A box on the last row whose corner is on the controls: the pin moves
	// left only while it stays off them; with no room it is refused.
	var g NoteBoard
	last := CellBox{0, 23, StatusGuardCols + 3, 1}
	if got := pin(g.Add(Note{Box: last, Text: "e"}, 80, 24, 1)); got != [2]int{StatusGuardCols + 1, 23} {
		t.Errorf("last-row pin %v, want [%d 23]", got, StatusGuardCols+1)
	}
	if _, err := g.Add(Note{Box: last, Text: "f"}, 80, 24, 1); err != ErrNoteOverlap {
		t.Errorf("the only spot off the controls is taken: %v, want ErrNoteOverlap", err)
	}
	if _, err := g.Add(Note{Box: CellBox{0, 23, StatusGuardCols, 1}, Text: "g"}, 80, 24, 1); err != ErrNoteOverlap {
		t.Errorf("a box with only the controls for its pin: %v, want ErrNoteOverlap", err)
	}
}

// The viewer's boxHasPin runs the same table (notes.test.mjs).
func TestPinSpotsCaseTable(t *testing.T) {
	raw, err := os.ReadFile("../../../tests/broadcast-viewer/pinspots.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Guard int
		Cases []struct {
			Name       string
			Box        [4]int
			Cols, Rows int
			Has        bool
		}
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatal(err)
	}
	if table.Guard != StatusGuardCols {
		t.Fatalf("the table assumes controls %d wide; they are %d", table.Guard, StatusGuardCols)
	}
	for _, c := range table.Cases {
		if got := hasPin(CellBox{c.Box[0], c.Box[1], c.Box[2], c.Box[3]}, c.Cols, c.Rows, 1); got != c.Has {
			t.Errorf("%s: %v, want %v", c.Name, got, c.Has)
		}
	}
}

func TestNoteCrop(t *testing.T) {
	n := Note{Box: CellBox{10, 5, 20, 4}}
	if got := NoteCrop(n, 80, 24, 1); got != (CellBox{9, 4, 22, 6}) {
		t.Errorf("crop %v", got)
	}
	if got := NoteCrop(Note{Box: CellBox{0, 22, 80, 2}}, 80, 24, 1); got != (CellBox{0, 21, 80, 3}) {
		t.Errorf("a crop is kept on the grid: %v", got)
	}
	if got := NoteCrop(Note{Box: CellBox{70, 20, 10, 4}}, 60, 18, 1); got.W != 0 || got.H != 0 {
		t.Errorf("a box off a grid that shrank crops nothing: %v", got)
	}
}

func TestNoteBoxEdgeAndWire(t *testing.T) {
	edge := boxEdge(CellBox{2, 1, 4, 3}, 80, 24, 1)
	if len(edge) != 10 {
		t.Errorf("a 4x3 box has 10 edge cells, got %d: %v", len(edge), edge)
	}
	if got := boxEdge(CellBox{0, 23, StatusGuardCols + 2, 1}, 80, 24, 1); len(got) != 2 {
		t.Errorf("the controls are never outlined: %v", got)
	}
	if got := boxEdge(CellBox{78, 22, 6, 5}, 80, 24, 1); len(got) != 3 {
		t.Errorf("a box past a shrunk grid is cut: %v", got)
	}
	n := Note{ID: 1, Col: 8, Row: 2, Box: CellBox{1, 2, 9, 3}, Text: "x"}
	raw, err := json.Marshal(n)
	if err != nil || string(raw) != `{"id":1,"col":8,"row":2,"box":[1,2,9,3],"color":0,"text":"x"}` {
		t.Errorf("wire %s %v", raw, err)
	}
	var back Note
	if err := json.Unmarshal(raw, &back); err != nil || back != n {
		t.Errorf("round trip %+v %v", back, err)
	}
}

func TestNoteBoardResolveIdempotent(t *testing.T) {
	var b NoteBoard
	n, _ := b.Add(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1)
	if !b.Resolve(n.ID) || b.Resolve(n.ID) || b.Resolve(999) {
		t.Fatal("resolve reports true once, then false")
	}
	if len(b.List()) != 0 {
		t.Fatal("resolved note still listed")
	}
}

func TestNoteBoardIDsAndCopies(t *testing.T) {
	var b NoteBoard
	a, _ := b.Add(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1)
	b.Resolve(a.ID)
	c, _ := b.Add(Note{Box: CellBox{0, 0, 2, 1}, Text: "c"}, 80, 24, 1)
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
	d, _ := b.Add(Note{Box: CellBox{0, 0, 2, 1}, Text: "d"}, 80, 24, 1)
	if d.ID <= c.ID {
		t.Fatal("ids keep increasing past a clear")
	}
}

const okNote = `{"cols":80,"rows":24,"box":[3,4,10,3],"color":2,"text":"hi\nthere"}`

func TestParseNote(t *testing.T) {
	n, err := ParseNote([]byte(okNote))
	if err != nil {
		t.Fatal(err)
	}
	want := NoteRequest{Cols: 80, Rows: 24, Note: Note{Box: CellBox{3, 4, 10, 3}, Color: 2, Text: "hi\nthere"}}
	if n != want {
		t.Fatalf("got %+v, want %+v", n, want)
	}
	if _, err := ParseNote([]byte(fmt.Sprintf(`{"cols":80,"rows":24,"box":[3,4,2,1],"color":0,"text":%q}`, strings.Repeat("é", 512)))); err != nil {
		t.Fatalf("1024 bytes must be accepted: %v", err)
	}
	// A box on the last row reaching past the controls has a pin spot.
	if _, err := ParseNote([]byte(fmt.Sprintf(`{"cols":80,"rows":24,"box":[0,23,%d,1],"color":0,"text":"x"}`, StatusGuardCols+2))); err != nil {
		t.Fatalf("a box past the controls: %v", err)
	}
}

// noteRefusals is every malformed or out-of-bounds note body; FuzzParseNote
// seeds from it.
var noteRefusals = []string{
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":2,"text":"hi","x":1}`,
	`{"cols":80,"rows":24,"col":3,"row":4,"color":2,"text":"hi"}`, // a point, from before #488
	`{"cols":80,"rows":24,"box":[3,4,10,3],"col":3,"row":4,"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":["3",4,10,3],"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,10],"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3,1],"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,null,3],"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":null,"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,1.5,3],"color":2,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":1.5,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":6,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":-1,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[79,4,2,1],"color":0,"text":"hi"}`,  // past the right edge
	`{"cols":80,"rows":24,"box":[3,4,1,3],"color":0,"text":"hi"}`,   // narrower than the pin
	`{"cols":80,"rows":24,"box":[3,4,10,0],"color":0,"text":"hi"}`,  // no height
	`{"cols":80,"rows":24,"box":[3,22,10,3],"color":0,"text":"hi"}`, // past the bottom
	`{"cols":80,"rows":24,"box":[-1,4,10,3],"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,-4,10,3],"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,9999999999,3],"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[1,0,9223372036854775807,1],"color":0,"text":"hi"}`, // Col+W overflows (BR-10)
	`{"cols":80,"rows":24,"box":[0,1,2,9223372036854775807],"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[9223372036854775807,0,2,1],"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[-9223372036854775808,0,2,1],"color":0,"text":"hi"}`,
	`{"cols":0,"rows":24,"box":[0,0,2,1],"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":0,"text":""}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":0}`,
	`{"rows":24,"box":[3,4,10,3],"color":0,"text":"hi"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":0,"text":"hi\u001b[2J"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":0,"text":"hi‮evil"}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":0,"text":"zero​width"}`,
	"{\"cols\":80,\"rows\":24,\"box\":[3,4,10,3],\"color\":0,\"text\":\"bad\xff\"}",
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":0,"text":"hi"} {}`,
	`{"cols":80,"rows":24,"box":[3,4,10,3],"color":0,"te`, // truncated
	`[]`,
	``,
}

// Every refusal is quick: no box makes the parser loop over more than the
// grid (BR-10's overflow spun the pin search for ~9e18 iterations).
func TestParseNoteRefusesQuickly(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, body := range noteRefusals {
			ParseNote([]byte(body))
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ParseNote did not return on the refusal bodies")
	}
}

func TestParseNoteRefusals(t *testing.T) {
	long := fmt.Sprintf(`{"cols":80,"rows":24,"box":[3,4,2,1],"color":0,"text":%q}`, strings.Repeat("é", 513))
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
		// The bounds, stated without adding the request's own numbers.
		b := n.Box
		if req.Cols < 1 || req.Rows < 1 || req.Cols > 1000 || req.Rows > 1000 ||
			b.Col < 0 || b.Row < 0 || b.W < 2 || b.H < 1 || b.Col >= req.Cols || b.Row >= req.Rows ||
			b.W > req.Cols-b.Col || b.H > req.Rows-b.Row || n.Color < 0 || n.Color >= PointerColors {
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
	// Inside the guard's width, so it is hidden once row 5 is the last row.
	n := Note{Col: StatusGuardCols - 2, Row: 5}
	cases := []struct {
		cols, rows int
		want       bool
	}{
		{80, 24, true},
		{StatusGuardCols, 24, true},
		{StatusGuardCols - 1, 24, false}, // second cell off the grid
		{80, 5, false},                   // row off the grid
		{80, 6, false},                   // now the last row, inside the guard
	}
	for _, c := range cases {
		if got := NoteVisible(n, c.cols, c.rows, 1); got != c.want {
			t.Errorf("%dx%d: %v, want %v", c.cols, c.rows, got, c.want)
		}
	}
	if !NoteVisible(Note{Col: StatusGuardCols, Row: 5}, 80, 6, 1) {
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

// A note's box is outlined (#488): its edge cells take the author's
// color as their background and keep their text; inside, away from the
// pin's glow, nothing changes.
func TestDrawNotesOutlinesTheBox(t *testing.T) {
	rows := []string{}
	for range 12 {
		rows = append(rows, strings.Repeat("x", 40))
	}
	f := textFrame(t, 40, 12, strings.Join(rows, "\n"), "")
	n := Note{Col: 30, Row: 2, Box: CellBox{2, 2, 30, 8}, Color: 1}
	out := DrawNotes(f, []Note{n}, NoteStyle{BG: color.RGBA{0, 0, 0, 0xff}, Blend: true})
	want := paletteTint(1, noteOutline, color.RGBA{0, 0, 0, 0xff}, true)
	for _, c := range [][2]int{{2, 6}, {31, 6}, {10, 2}, {10, 9}} {
		cl := out.Cells[c[1]*40+c[0]]
		if cl.Style.Bg != want || cl.Content != "x" {
			t.Errorf("edge cell %v: bg %v content %q, want the outline tint and its text", c, cl.Style.Bg, cl.Content)
		}
	}
	for _, c := range [][2]int{{5, 6}, {1, 6}, {10, 10}} {
		if cl := out.Cells[c[1]*40+c[0]]; cl.Style.Bg != f.Cells[c[1]*40+c[0]].Style.Bg {
			t.Errorf("cell %v off the edge changed: %v", c, cl.Style.Bg)
		}
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
			board.Add(Note{Box: CellBox{rng.Intn(cols - 1), rng.Intn(rows - 1), 2 + rng.Intn(6), 1 + rng.Intn(3)}, Text: "x"}, cols, rows, 1)
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
			if !NoteVisible(n, cols, rows, 1) {
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

// Count the search work rather than requiring a loaded host to finish quickly.
func TestHasPinStopsAtTheFirstSpot(t *testing.T) {
	for _, size := range []int{1000, 2000} {
		for range 1000 {
			visits := 0
			_, ok := firstPin(CellBox{0, 0, size, size}, size, size, 1, func(cell) bool {
				visits++
				return false
			})
			if !ok || visits != 1 || !hasPin(CellBox{0, 0, size, size}, size, size, 1) {
				t.Fatalf("grid %d: pin=%v, visits=%d; want first-cell success", size, ok, visits)
			}
		}
	}
}

// pair#495: a note carries the name the viewer gave the lock page, on every
// cast's rule; the field is optional, and a name off the rule is refused.
func TestParseNoteAuthor(t *testing.T) {
	body := func(author string) []byte {
		return []byte(fmt.Sprintf(`{"cols":80,"rows":24,"box":[3,4,2,1],"color":0,"text":"hi","author":%q}`, author))
	}
	n, err := ParseNote(body("  Ada  "))
	if err != nil || n.Note.Author != "Ada" {
		t.Fatalf("author: %+v %v", n.Note, err)
	}
	if n, err := ParseNote([]byte(okNote)); err != nil || n.Note.Author != "" {
		t.Fatalf("no author: %+v %v", n.Note, err)
	}
	for _, bad := range []string{"A\nB", "A​B", strings.Repeat("x", MaxViewerNameRunes+1)} {
		if _, err := ParseNote(body(bad)); err != ErrNoteText {
			t.Errorf("author %q: %v", bad, err)
		}
	}
}
