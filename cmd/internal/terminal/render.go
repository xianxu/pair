package terminal

import (
	"fmt"
	"strings"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Each presented frame is one synchronized-output update (DECSET 2026): the
// parent parses the whole frame into its grid and draws only the finished
// result, never the erase-and-redraw inside it (#262). A terminal without the
// mode ignores it. parentReleaseControls also closes it, for a write that failed
// mid-frame.
const (
	syncBegin = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
)

// cursorEpilogue places the cursor and restores its shape and visibility after a
// frame painted with the cursor hidden.
func cursorEpilogue(c Cursor) string {
	shape := c.Shape
	if shape == 0 {
		shape = 1
	}
	code := shape * 2
	if c.Blink {
		code--
	}
	s := fmt.Sprintf("\x1b[%d;%dH\x1b[%d q", c.Y+1, c.X+1, code)
	if c.Visible {
		s += "\x1b[?25h"
	}
	return s
}

// Render produces only compositor-owned terminal drawing. A zero previous
// frame invalidates the cache. The caller may retain next only after every byte
// succeeds; a partial/error outcome invalidates that cache permanently.
func Render(prev, next Frame) ([]byte, error) {
	if err := next.Validate(); err != nil {
		return nil, err
	}
	full := prev.Geometry != next.Geometry || len(prev.Cells) != len(next.Cells)
	dirty := full || prev.Cursor != next.Cursor
	if !dirty {
		for i := range next.Cells {
			if !next.Cells[i].Equal(&prev.Cells[i]) {
				dirty = true
				break
			}
		}
	}
	if !dirty {
		return nil, nil
	}
	var out strings.Builder
	out.WriteString(syncBegin)
	// Disable autowrap while painting the lower-right cell, and reset origin and
	// margins independently of whatever was on the parent's screen before us.
	// Re-asserted every frame on purpose, not deltaed: the state has no
	// synchronous confirm path (atlas/terminal.md, #262 M2).
	out.WriteString("\x1b[?25l\x1b[?6l\x1b[r\x1b[?7l\x1b[0m\x1b]8;;\x1b\\")
	if full {
		out.WriteString("\x1b[2J")
	}
	var style uv.Style
	var link uv.Link
	writeX, writeY := -1, -1
	cols := next.Geometry.Cols
	for y := 0; y < next.Geometry.Rows; y++ {
		for x := 0; x < cols; x++ {
			i := y*cols + x
			c := next.Cells[i]
			if x > 0 && next.Cells[i-1].Width == 2 {
				continue
			}
			changed := full || !c.Equal(&prev.Cells[i])
			if !changed && c.Width == 2 {
				changed = !next.Cells[i+1].Equal(&prev.Cells[i+1])
			}
			if !changed {
				continue
			}
			if writeX != x || writeY != y {
				fmt.Fprintf(&out, "\x1b[%d;%dH", y+1, x+1)
			}
			if !style.Equal(&c.Style) {
				// Style.String is a set, not a reset, so reset removed attributes first.
				out.WriteString("\x1b[0m")
				out.WriteString(c.Style.String())
				style = c.Style
			}
			if link != c.Link {
				out.WriteString("\x1b]8;" + c.Link.Params + ";" + c.Link.URL + "\x1b\\")
				link = c.Link
			}
			content := c.Content
			if content == "" {
				content = " "
			}
			out.WriteString(content)
			writeX = x + max(1, c.Width)
			writeY = y
		}
	}
	out.WriteString("\x1b]8;;\x1b\\\x1b[0m\x1b[?7h")
	out.WriteString(cursorEpilogue(next.Cursor))
	out.WriteString(syncEnd)
	return []byte(out.String()), nil
}

// StyledRows clips trusted UI text to its rectangle. Only SGR, OSC 8 and
// newline separators are accepted; no cursor, erase, mode or other effects.
// Validation continues after clipping so hidden escape payloads cannot pass.
func StyledRows(text string, cols, rows int) ([]Cell, error) {
	g := Geometry{Cols: cols, Rows: rows}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	if len(text) > MaxInputBytes {
		return nil, fmt.Errorf("terminal: chrome exceeds text limit")
	}
	cells := make([]Cell, cols*rows)
	for i := range cells {
		cells[i] = uv.EmptyCell
	}
	p := ansi.GetParser()
	defer ansi.PutParser(p)
	var style uv.Style
	var link uv.Link
	x, y := 0, 0
	for len(text) > 0 {
		if strings.HasPrefix(text, "\r\n") {
			x = 0
			y++
			text = text[2:]
			continue
		}
		if text[0] == '\n' {
			x = 0
			y++
			text = text[1:]
			continue
		}
		if strings.HasPrefix(text, "\x1b[") {
			end := strings.IndexByte(text, 'm')
			if end < 0 || end > MaxStringBytes {
				return nil, fmt.Errorf("terminal: incomplete chrome SGR")
			}
			for _, b := range []byte(text[2:end]) {
				if (b < '0' || b > '9') && b != ';' && b != ':' {
					return nil, fmt.Errorf("terminal: forbidden chrome control")
				}
			}
			seq, _, n, _ := ansi.DecodeSequence(text[:end+1], 0, p)
			if n != end+1 || len(seq) != n || p.Command() != 'm' {
				return nil, fmt.Errorf("terminal: invalid chrome SGR")
			}
			uv.ReadStyle(p.Params(), &style)
			text = text[n:]
			continue
		}
		if strings.HasPrefix(text, "\x1b]8;") {
			end, term := -1, 0
			for i := 4; i < len(text); i++ {
				if text[i] == 7 {
					end = i
					term = 1
					break
				}
				if text[i] == 27 && i+1 < len(text) && text[i+1] == '\\' {
					end = i
					term = 2
					break
				}
			}
			if end < 0 {
				return nil, fmt.Errorf("terminal: incomplete chrome hyperlink")
			}
			parts := strings.SplitN(text[4:end], ";", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("terminal: malformed chrome hyperlink")
			}
			link = uv.Link{Params: parts[0], URL: parts[1]}
			if !validLink(Cell{Link: link}) {
				return nil, fmt.Errorf("terminal: unsafe chrome hyperlink")
			}
			text = text[end+term:]
			continue
		}
		// A leading zero-width rune has no printable base to amend. Consume
		// it before grouping: a leading joiner + selector must not fabricate
		// a wide cell. Controls remain rejected, including zero-width controls.
		_, runeBytes := utf8.DecodeRuneInString(text)
		lead, leadWidth := ansi.FirstGraphemeCluster(text[:runeBytes], ansi.GraphemeWidth)
		if !plainText(lead) {
			return nil, fmt.Errorf("terminal: forbidden chrome text or control")
		}
		if leadWidth == 0 {
			text = text[runeBytes:]
			continue
		}
		seq, width := ansi.FirstGraphemeCluster(text, ansi.GraphemeWidth)
		n := len(seq)
		if n < 1 || width < 1 || width > 2 || !plainText(seq) || len(seq) > MaxClusterBytes {
			return nil, fmt.Errorf("terminal: forbidden chrome text or control")
		}
		if y < rows && x+width <= cols {
			cells[y*cols+x] = Cell{Content: seq, Width: width, Style: style, Link: link}
			if width == 2 {
				cells[y*cols+x+1] = Cell{}
			}
		}
		x += width
		text = text[n:]
	}
	return cells, nil
}

// Compose preserves the child geometry with an optional single bottom row.
func Compose(child Frame, host Geometry, bottom []Cell) (Frame, error) {
	if err := child.Validate(); err != nil {
		return Frame{}, err
	}
	if err := host.Validate(); err != nil {
		return Frame{}, err
	}
	chromeRows := 0
	if len(bottom) > 0 {
		chromeRows = 1
	}
	if child.Geometry.Cols != host.Cols || child.Geometry.Rows+chromeRows != host.Rows || (len(bottom) != 0 && len(bottom) != host.Cols) {
		return Frame{}, fmt.Errorf("terminal: incompatible chrome geometry")
	}
	f := child
	f.Geometry = host
	f.Rows = make([]RowMetadata, host.Rows)
	for y := 0; y < child.Geometry.Rows; y++ {
		f.Rows[y] = child.rowMetadata(y)
	}
	if chromeRows == 1 {
		f.Rows[host.Rows-1] = rowMetadata(bottom)
	}
	f.Cells = make([]Cell, len(child.Cells)+len(bottom))
	for i, c := range child.Cells {
		f.Cells[i] = cloneCell(c)
	}
	for i, c := range bottom {
		f.Cells[len(child.Cells)+i] = cloneCell(c)
	}
	if err := f.Validate(); err != nil {
		return Frame{}, err
	}
	return f, nil
}

// PanelFrame makes a compositor-owned surface with no child input destination.
func PanelFrame(host Geometry, cells []Cell, cursor Cursor) (Frame, error) {
	f := Frame{Geometry: host, Cells: cells, Cursor: cursor}
	if err := f.Validate(); err != nil {
		return Frame{}, err
	}
	return f.Clone(), nil
}
