package broadcast

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// Sticky notes (#447): a pointer-link viewer pins a plain-text note to a
// screen cell, and Couch draws NoteGlyph there. A note stays at its cell
// whatever scrolls underneath; it lives in memory and goes when pointing
// turns off or the broadcast ends.
const (
	NoteGlyph        = "💭"
	MaxNotesPerColor = 8
	MaxNoteBytes     = 1024
	// MaxNoteBody bounds a note request: a full note escaped as \uXXXX is
	// about 6 KB.
	MaxNoteBody = 8 << 10
)

var (
	ErrNoteShape     = errors.New("broadcast: malformed note")
	ErrNoteRange     = errors.New("broadcast: note outside the grid")
	ErrNoteText      = errors.New("broadcast: note text not allowed")
	ErrNoteColorFull = errors.New("broadcast: too many notes in one color")
	ErrNoteOverlap   = errors.New("broadcast: a note is already there")
)

// Note is one sticky note. Col, Row is the left of the two cells NoteGlyph
// covers. Color is the author's pointer palette index: the per-color limit
// counts it, and the 💭's chip and glow are drawn in it (M3). At is when it
// was saved, which sets its flash phase; it isn't sent to pages.
type Note struct {
	ID    int       `json:"id"`
	Col   int       `json:"col"`
	Row   int       `json:"row"`
	Color int       `json:"color"`
	Text  string    `json:"text"`
	At    time.Time `json:"-"`
}

// NoteBoard is a broadcast's open notes. Pure: the caller owns locking.
type NoteBoard struct {
	notes  []Note
	nextID int
}

// Add opens n, assigning its ID. At most MaxNotesPerColor notes per color
// are open, and no two notes share a cell.
func (b *NoteBoard) Add(n Note) (Note, error) {
	same := 0
	for _, o := range b.notes {
		if o.Row == n.Row && o.Col-n.Col < 2 && n.Col-o.Col < 2 {
			return Note{}, ErrNoteOverlap
		}
		if o.Color == n.Color {
			same++
		}
	}
	if same >= MaxNotesPerColor {
		return Note{}, ErrNoteColorFull
	}
	b.nextID++
	n.ID = b.nextID
	b.notes = append(b.notes, n)
	return n, nil
}

// Resolve removes the note with id, reporting whether there was one.
func (b *NoteBoard) Resolve(id int) bool {
	for i, n := range b.notes {
		if n.ID == id {
			b.notes = append(b.notes[:i], b.notes[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes every note; IDs keep increasing.
func (b *NoteBoard) Clear() { b.notes = nil }

// List is a copy of the open notes, oldest first.
func (b *NoteBoard) List() []Note { return append([]Note{}, b.notes...) }

// NoteRequest is a page asking to open Note, placed on a Cols×Rows grid.
type NoteRequest struct {
	Cols, Rows int
	Note       Note
}

// ParseNote reads a note request strictly, as ParsePointBatch does: one
// JSON object with exactly these fields, integers in range, a cell with room
// for the glyph's second column and off the broadcast's controls, and plain
// text. Its errors are fixed values that never echo the input.
func ParseNote(body []byte) (NoteRequest, error) {
	var raw struct {
		Cols  *int    `json:"cols"`
		Rows  *int    `json:"rows"`
		Col   *int    `json:"col"`
		Row   *int    `json:"row"`
		Color *int    `json:"color"`
		Text  *string `json:"text"`
	}
	// The decoder would turn invalid UTF-8 into U+FFFD; refuse it instead.
	if !utf8.Valid(body) || !DecodeStrict(body, &raw) {
		return NoteRequest{}, ErrNoteShape
	}
	if raw.Cols == nil || raw.Rows == nil || raw.Col == nil || raw.Row == nil || raw.Color == nil || raw.Text == nil {
		return NoteRequest{}, ErrNoteShape
	}
	cols, rows := *raw.Cols, *raw.Rows
	n := Note{Col: *raw.Col, Row: *raw.Row, Color: *raw.Color, Text: *raw.Text}
	if cols < 1 || rows < 1 || cols > maxPointGrid || rows > maxPointGrid || n.Col < 0 || n.Row < 0 ||
		n.Color < 0 || n.Color >= PointerColors || !NoteVisible(n, cols, rows) {
		return NoteRequest{}, ErrNoteRange
	}
	if !PlainText(n.Text) {
		return NoteRequest{}, ErrNoteText
	}
	return NoteRequest{Cols: cols, Rows: rows, Note: n}, nil
}

// ParseResolve reads a resolve request: {"id": n}, n a positive integer.
func ParseResolve(body []byte) (int, error) {
	var raw struct {
		ID *int `json:"id"`
	}
	if !DecodeStrict(body, &raw) || raw.ID == nil || *raw.ID < 1 {
		return 0, ErrNoteShape
	}
	return *raw.ID, nil
}

// DecodeStrict decodes exactly one JSON object with no unknown fields.
func DecodeStrict(body []byte, v any) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return false
	}
	_, err := dec.Token()
	return err == io.EOF
}

// PlainText is 1..MaxNoteBytes of valid UTF-8 with no control characters
// but newline and tab, and no format characters (bidi overrides, zero-width
// marks), so a note can't show text other than what it holds.
func PlainText(s string) bool {
	if s == "" || len(s) > MaxNoteBytes || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if (unicode.IsControl(r) && r != '\n' && r != '\t') || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

// NoteVisible reports whether n draws on a cols×rows grid: both of its cells
// on the grid, and off the broadcast's controls.
func NoteVisible(n Note, cols, rows int) bool {
	return n.Col >= 0 && n.Row >= 0 && n.Col+1 < cols && n.Row < rows && !guarded(cell{n.Col, n.Row}, rows)
}

// DrawNotes returns f with each visible note drawn: NoteGlyph on its two
// cells, on a chip of its author's color, inside a round glow that flashes
// (notehalo.go). A wide character the glyph cuts in half leaves its other
// half as a space in its own style. The glow changes only backgrounds, never
// covers another note's chip, and where two glows meet the stronger wins.
// With nothing to draw it returns f itself, uncopied.
func DrawNotes(f terminal.Frame, notes []Note, style NoteStyle) terminal.Frame {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	var out terminal.Frame
	copied := false
	glow := map[cell]float64{}
	glowColor := map[cell]int{}
	chips := map[cell]bool{}
	for _, n := range notes {
		if !NoteVisible(n, cols, rows) || len(f.Cells) != cols*rows {
			continue
		}
		chips[cell{n.Col, n.Row}], chips[cell{n.Col + 1, n.Row}] = true, true
		flash := noteFlash(n.At, style.Now)
		if n.ID == style.Hover {
			flash = 1
		}
		for c, t := range noteGlow(n, cols, rows, flash) {
			if old, seen := glow[c]; !seen || t < old || (t == old && n.Color < glowColor[c]) {
				glow[c], glowColor[c] = t, n.Color
			}
		}
	}
	for _, n := range notes {
		if !NoteVisible(n, cols, rows) || len(f.Cells) != cols*rows {
			continue
		}
		if !copied {
			out, copied = f.Clone(), true
		}
		i := n.Row*cols + n.Col
		// The checks read out, not f, so nearby notes draw the same in any
		// order.
		if out.Cells[i].Width == 0 && n.Col > 0 && out.Cells[i-1].Width == 2 {
			blank(&out.Cells[i-1])
		}
		if out.Cells[i+1].Width == 2 && n.Col+2 < cols {
			blank(&out.Cells[i+2])
		}
		chip := paletteTint(n.Color, 0, style.BG, style.Blend)
		out.Cells[i] = terminal.Cell{Content: NoteGlyph, Width: 2, Style: uv.Style{Bg: chip}}
		out.Cells[i+1] = terminal.Cell{Style: uv.Style{Bg: chip}}
		// The renderer draws a row's text only up to its used columns; past
		// them is blank tail. Widen them to cover the glyph.
		if n.Row < len(out.Rows) && out.Rows[n.Row].UsedColumns < n.Col+2 {
			out.Rows[n.Row].UsedColumns = n.Col + 2
		}
	}
	if !copied {
		return f
	}
	// Both halves of a wide character take one tint, the stronger of the
	// two, so the result never depends on map order.
	type tint struct {
		t     float64
		color int
	}
	leads := map[cell]tint{}
	for c, t := range glow {
		if chips[c] {
			continue
		}
		color := glowColor[c]
		if i := c.row*cols + c.col; out.Cells[i].Width == 0 && c.col > 0 && out.Cells[i-1].Width == 2 {
			c.col--
		}
		if old, seen := leads[c]; !seen || t < old.t || (t == old.t && color < old.color) {
			leads[c] = tint{t, color}
		}
	}
	for c, tn := range leads {
		tintCell(out, c, paletteTint(tn.color, tn.t, style.BG, style.Blend))
	}
	return out
}

// blank turns a cell into a space, keeping its style.
func blank(c *terminal.Cell) {
	c.Content, c.Width = " ", 1
}
