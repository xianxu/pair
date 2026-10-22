package broadcast

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// Sticky notes (#447): a pointer-link viewer leaves a plain-text note on a
// box of the screen (#488: dragged, or the default box of a tap), and Couch
// outlines the box and draws NoteGlyph at its top-right corner. A note
// stays where it was placed whatever scrolls underneath; it lives in memory
// and goes when pointing turns off or the broadcast ends.
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
	ErrNoteOwner     = errors.New("broadcast: only the note's author can edit it")
	ErrNoteGone      = errors.New("broadcast: that note is gone")
)

// Note is one sticky note. Box is the area it is about, in cells (#488):
// what Couch outlines, and what a submission crops (NoteCrop). Col, Row is
// its pin, the left of the two cells NoteGlyph covers, which the board
// places inside the box (firstPin); a page never sends it. Color is the
// author's pointer palette index: the per-color limit counts it, and the
// box, the 💭's chip and its glow are drawn in it. At is when it was saved,
// which sets its flash phase; it isn't sent to pages.
type Note struct {
	ID     int     `json:"id"`
	Col    int     `json:"col"`
	Row    int     `json:"row"`
	Box    CellBox `json:"box"`
	Color  int     `json:"color"`
	Text   string  `json:"text"`
	Owner  string  `json:"owner,omitempty"`
	Edited bool    `json:"edited,omitempty"`
	// Anchor is what the box is on (#488), from the operator's program
	// (Config.AnchorNote); nil when it gives none. It is operator-side
	// context, never on the pages' wire: a page neither sends nor receives it.
	Anchor *NoteAnchor `json:"-"`
	// Author is the name the viewer gave the lock page (pair#495); "" when
	// none was sent. It is attribution, not authentication.
	Author string `json:"author,omitempty"`
	// On is the grid the note was placed on (NoteBoard.Add): a box means
	// the screen of that size.
	On terminal.Geometry `json:"-"`
	At time.Time         `json:"-"`
}

// NoteAnchor is what a terminal note is about, in the shape the web cast's
// anchors share (#488, recording.NoteAt's kind and path): Kind is where on
// Couch's screen the box's centre is (AnchorTabBar, AnchorStatusLine,
// AnchorPane, or AnchorScreen when the pane isn't known), Path the steps
// down to it (e.g. "pane pair:3", "draft nvim", "notes.md, lines 12–18"),
// and Text the screen's text under the box.
type NoteAnchor struct {
	Kind string   `json:"kind"`
	Path []string `json:"path,omitempty"`
	Text string   `json:"text,omitempty"`
}

// Anchor kinds.
const (
	AnchorTabBar     = "tabbar"
	AnchorStatusLine = "status"
	AnchorPane       = "pane"
	AnchorScreen     = "screen"
)

// CellBox is a box of the screen in cells: its top-left cell and its size.
// On the wire it is [col, row, w, h], like the web cast's region.
type CellBox struct{ Col, Row, W, H int }

func (b CellBox) MarshalJSON() ([]byte, error) {
	return json.Marshal([4]int{b.Col, b.Row, b.W, b.H})
}

// UnmarshalJSON reads MarshalJSON's array back (Couch's own notes event;
// a page's note request is ParseNote's, which checks it).
func (b *CellBox) UnmarshalJSON(raw []byte) error {
	var a [4]int
	if err := json.Unmarshal(raw, &a); err != nil {
		return err
	}
	*b = CellBox{a[0], a[1], a[2], a[3]}
	return nil
}

// within reports whether b is a box of at least the pin's two cells on a
// cols×rows grid. It compares by subtraction, never adding b's own
// (untrusted) sizes, so no box can overflow its way in (#488 BR-10).
func (b CellBox) within(cols, rows int) bool {
	return b.Col >= 0 && b.Row >= 0 && b.Col < cols && b.Row < rows &&
		b.W >= 2 && b.H >= 1 && b.W <= cols-b.Col && b.H <= rows-b.Row
}

// firstPin is where a box's pin goes: the first spot, best first, that
// taken (nil: none is) doesn't refuse. Best first is the top-right corner
// inside the box (the operator's choice, #488), then leftward along the
// top row, then the rows below; only spots where NoteGlyph is visible (on
// the grid, off the broadcast's controls). A box not within the grid has
// none, so the loops only ever run over the grid, and stop at the first
// spot found (a large box costs nothing past it).
func firstPin(b CellBox, cols, rows, chromeRows int, taken func(cell) bool) (cell, bool) {
	if chromeRows <= 0 || chromeRows >= rows || !b.within(cols, rows) {
		return cell{}, false
	}
	for row := b.Row; row < b.Row+b.H; row++ {
		for col := b.Col + b.W - 2; col >= b.Col; col-- {
			c := cell{col, row}
			if NoteVisible(Note{Col: col, Row: row}, cols, rows, chromeRows) && (taken == nil || !taken(c)) {
				return c, true
			}
		}
	}
	return cell{}, false
}

// hasPin reports whether a box has a spot for its pin at all.
func hasPin(b CellBox, cols, rows, chromeRows int) bool {
	_, ok := firstPin(b, cols, rows, chromeRows, nil)
	return ok
}

// NoteCrop is the part of the screen a submission of n shows (#455): its
// box plus margin cells each way, kept on the cols×rows grid.
func NoteCrop(n Note, cols, rows, margin int) CellBox {
	c0, r0 := max(n.Box.Col-margin, 0), max(n.Box.Row-margin, 0)
	c1, r1 := min(n.Box.Col+n.Box.W+margin, cols), min(n.Box.Row+n.Box.H+margin, rows)
	return CellBox{c0, r0, max(c1-c0, 0), max(r1-r0, 0)}
}

// MaxViewerNameRunes bounds a viewer's name on every cast; web/collab.js's
// validName is the same rule, for the pages.
const MaxViewerNameRunes = 40

// ViewerName is a viewer's name as every cast takes it: trimmed, one line of
// plain text (no control or format character), at most MaxViewerNameRunes.
func ViewerName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > MaxViewerNameRunes {
		return "", false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return "", false
		}
	}
	return s, true
}

// NoteBoard is a broadcast's open notes. Pure: the caller owns locking.
type NoteBoard struct {
	notes  []Note
	nextID int
}

// Add opens n, placed on a cols×rows grid, assigning its ID and its pin:
// the first of the box's pin spots no open note's pin covers. At most
// MaxNotesPerColor notes per color are open, and no two pins share a cell;
// a box with no free spot is refused.
func (b *NoteBoard) Add(n Note, cols, rows, chromeRows int) (Note, error) {
	same := 0
	for _, o := range b.notes {
		if o.Color == n.Color {
			same++
		}
	}
	if same >= MaxNotesPerColor {
		return Note{}, ErrNoteColorFull
	}
	pin, ok := firstPin(n.Box, cols, rows, chromeRows, b.pinTaken)
	if !ok {
		return Note{}, ErrNoteOverlap
	}
	n.Col, n.Row = pin.col, pin.row
	n.On = terminal.Geometry{Cols: cols, Rows: rows}
	b.nextID++
	n.ID = b.nextID
	b.notes = append(b.notes, n)
	return n, nil
}

// pinTaken reports whether a pin at c would share a cell with an open
// note's pin.
func (b *NoteBoard) pinTaken(c cell) bool {
	for _, o := range b.notes {
		if o.Row == c.row && o.Col-c.col < 2 && c.col-o.Col < 2 {
			return true
		}
	}
	return false
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
// JSON object with exactly these fields
// ({"cols","rows","box":[col,row,w,h],"color","text"}), integers in range, a
// box on the grid at least the pin's two cells wide, and plain text. Current
// frame admission decides whether its pin can avoid the controls. The pin is the board's
// to place (NoteBoard.Add). Its errors are fixed values that never echo
// the input.
func ParseNote(body []byte) (NoteRequest, error) {
	var raw struct {
		Cols   *int    `json:"cols"`
		Rows   *int    `json:"rows"`
		Box    *[]*int `json:"box"`
		Color  *int    `json:"color"`
		Text   *string `json:"text"`
		Author *string `json:"author"` // optional (pair#495)
	}
	// The decoder would turn invalid UTF-8 into U+FFFD; refuse it instead.
	if !utf8.Valid(body) || !DecodeStrict(body, &raw) {
		return NoteRequest{}, ErrNoteShape
	}
	if raw.Cols == nil || raw.Rows == nil || raw.Box == nil || len(*raw.Box) != 4 || raw.Color == nil || raw.Text == nil {
		return NoteRequest{}, ErrNoteShape
	}
	var box [4]int
	for i, v := range *raw.Box {
		if v == nil {
			return NoteRequest{}, ErrNoteShape
		}
		if *v < 0 || *v > maxPointGrid { // bounded before any arithmetic
			return NoteRequest{}, ErrNoteRange
		}
		box[i] = *v
	}
	cols, rows := *raw.Cols, *raw.Rows
	n := Note{Box: CellBox{box[0], box[1], box[2], box[3]}, Color: *raw.Color, Text: *raw.Text}
	if cols < 1 || rows < 1 || cols > maxPointGrid || rows > maxPointGrid ||
		!n.Box.within(cols, rows) || n.Color < 0 || n.Color >= PointerColors {
		return NoteRequest{}, ErrNoteRange
	}
	if !PlainText(n.Text) {
		return NoteRequest{}, ErrNoteText
	}
	if raw.Author != nil {
		name, ok := ViewerName(*raw.Author)
		if !ok {
			return NoteRequest{}, ErrNoteText
		}
		n.Author = name
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
func NoteVisible(n Note, cols, rows, chromeRows int) bool {
	return n.Col >= 0 && n.Row >= 0 && n.Col+1 < cols && n.Row < rows && !guarded(cell{n.Col, n.Row}, rows, chromeRows)
}

// DrawNotes returns f with each visible note drawn: its box's edge cells
// tinted in its author's color, keeping their text (#488, the operator's
// choice), and NoteGlyph on its pin's two cells, on a chip of that color,
// inside a round glow that flashes (notehalo.go). A wide character the glyph cuts in half leaves its other
// half as a space in its own style. The glow changes only backgrounds, never
// covers another note's chip, and where two glows meet the stronger wins.
// With nothing to draw it returns f itself, uncopied.
func DrawNotes(f terminal.Frame, notes []Note, style NoteStyle) terminal.Frame {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	chromeRows := f.ChromeRows
	var out terminal.Frame
	copied := false
	glow := map[cell]float64{}
	glowColor := map[cell]int{}
	chips := map[cell]bool{}
	for _, n := range notes {
		if !NoteVisible(n, cols, rows, chromeRows) || len(f.Cells) != cols*rows ||
			guardedGlyph(f, cell{n.Col, n.Row}) || guardedGlyph(f, cell{n.Col + 1, n.Row}) {
			continue
		}
		chips[cell{n.Col, n.Row}], chips[cell{n.Col + 1, n.Row}] = true, true
		flash := noteFlash(n.At, style.Now)
		if n.ID == style.Hover {
			flash = 1
		}
		take := func(c cell, t float64) {
			if old, seen := glow[c]; !seen || t < old || (t == old && n.Color < glowColor[c]) {
				glow[c], glowColor[c] = t, n.Color
			}
		}
		for _, c := range boxEdge(n.Box, cols, rows, chromeRows) {
			take(c, noteOutline)
		}
		for c, t := range noteGlow(n, cols, rows, chromeRows, flash) {
			take(c, t)
		}
	}
	for _, n := range notes {
		if !NoteVisible(n, cols, rows, chromeRows) || len(f.Cells) != cols*rows ||
			guardedGlyph(f, cell{n.Col, n.Row}) || guardedGlyph(f, cell{n.Col + 1, n.Row}) {
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

// noteOutline is how far a box's edge is weakened toward the background
// (paletteTint's t): plainly the author's color, light enough that the text
// it covers stays readable.
const noteOutline = 0.55

// boxEdge is the cells of b's edge on a cols×rows grid (a grid that shrank
// cuts it), never the broadcast's controls.
func boxEdge(b CellBox, cols, rows, chromeRows int) []cell {
	var out []cell
	add := func(col, row int) {
		if c := (cell{col, row}); col >= 0 && row >= 0 && col < cols && row < rows && !guarded(c, rows, chromeRows) {
			out = append(out, c)
		}
	}
	last := b.Row + b.H - 1
	for col := b.Col; col < b.Col+b.W; col++ {
		add(col, b.Row)
		if last != b.Row {
			add(col, last)
		}
	}
	for row := b.Row + 1; row < last; row++ {
		add(b.Col, row)
		if b.W > 1 {
			add(b.Col+b.W-1, row)
		}
	}
	return out
}

// blank turns a cell into a space, keeping its style.
func blank(c *terminal.Cell) {
	c.Content, c.Width = " ", 1
}
