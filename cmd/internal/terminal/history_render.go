package terminal

import (
	"fmt"
	"reflect"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

// HistoryState describes only a completely written parent presentation.
// FirstID records the bounded exported suffix; continuous parent history may
// retain an older prefix under the parent terminal's own scrollback limit.
type HistoryState struct {
	EndpointID string
	Cursor     HistoryCursor
	FirstID    uint64
	Columns    int
	AltScreen  bool
}

type historyPaintRow struct {
	cells   []Cell
	used    int
	wrapped bool
}

// HistoryRender borrows immutable publication/frame data until Emit returns.
// The caller commits NextState only after every emitted chunk succeeds.
type HistoryRender struct {
	previous, next                   Frame
	history                          HistoryWindow
	state                            HistoryState
	rows                             []historyPaintRow
	reset, dirty, enterAlt, leaveAlt bool
}

func RenderWithHistory(previous, next Frame, history HistoryWindow, installed HistoryState) (HistoryRender, error) {
	if err := next.Validate(); err != nil {
		return HistoryRender{}, err
	}
	if err := history.Validate(); err != nil {
		return HistoryRender{}, err
	}
	first := history.Cursor.NextID
	if len(history.Rows) > 0 {
		first = history.Rows[0].ID
	}
	p := HistoryRender{previous: previous, next: next, history: history}
	p.state = HistoryState{EndpointID: next.EndpointID, Cursor: history.Cursor, FirstID: first, Columns: next.Geometry.Cols, AltScreen: next.AltScreen}
	p.enterAlt = next.AltScreen && !installed.AltScreen
	p.leaveAlt = !next.AltScreen && installed.AltScreen
	p.reset = installed.EndpointID != next.EndpointID || previous.EndpointID != next.EndpointID || installed.Columns != next.Geometry.Cols || previous.Geometry != next.Geometry || installed.Cursor.ClearEpoch != history.Cursor.ClearEpoch || installed.Cursor.NextID > history.Cursor.NextID || p.leaveAlt
	p.dirty = p.reset || p.enterAlt || p.leaveAlt || installed.Cursor != history.Cursor || !sameHistoryFrame(previous, next)
	if !p.dirty {
		return p, nil
	}
	if !next.AltScreen {
		start := 0
		if !p.reset {
			for start < len(history.Rows) && history.Rows[start].ID < installed.Cursor.NextID {
				start++
			}
			// Prefix eviction alone is not a loss to the parent: those rows were
			// already delivered. Append while every ID since its cursor remains
			// available; otherwise rebuild only the bounded retained suffix.
			expected := installed.Cursor.NextID
			for _, row := range history.Rows[start:] {
				if row.ID != expected {
					p.reset = true
					break
				}
				expected++
			}
			if expected != history.Cursor.NextID {
				p.reset = true
			}
			// A wrapped, underfull first row needs pristine storage in both parent
			// models. ECH would materialize a copied gap in Zellij; EL2 would clear
			// xterm's incoming wrap flag. Rebuild from typed state instead.
			if next.rowMetadata(0).Wrapped && next.Geometry.Rows > 1 && next.rowMetadata(1).Wrapped && next.rowMetadata(0).UsedColumns < next.Geometry.Cols {
				p.reset = true
			}
			if previous.Geometry == next.Geometry && previous.rowMetadata(0).Wrapped != next.rowMetadata(0).Wrapped && start == len(history.Rows) {
				p.reset = true
			}
		}
		if p.reset {
			start = 0
		}
		var err error
		p.rows, err = reflowHistory(history.Rows[start:], next.Geometry.Cols)
		if err != nil {
			return HistoryRender{}, err
		}
		if !p.reset && len(p.rows) > 0 && p.rows[0].wrapped && p.rows[0].used < next.Geometry.Cols && (len(p.rows) > 1 && p.rows[1].wrapped || len(p.rows) == 1 && history.ContinuesToScreen) {
			p.reset = true
			p.rows, err = reflowHistory(history.Rows, next.Geometry.Cols)
			if err != nil {
				return HistoryRender{}, err
			}
		}
	}
	return p, nil
}
func (p HistoryRender) NextState() HistoryState { return p.state }
func sameHistoryFrame(a, b Frame) bool {
	if a.EndpointID != b.EndpointID || a.Geometry != b.Geometry || a.Cursor != b.Cursor || a.AltScreen != b.AltScreen || len(a.Cells) != len(b.Cells) {
		return false
	}
	for i := range a.Cells {
		if !a.Cells[i].Equal(&b.Cells[i]) {
			return false
		}
	}
	return reflect.DeepEqual(a.Rows, b.Rows)
}

// Reflow retains at most the input cell count and one descriptor per input cell
// or hard row. It never pads to parent width or constructs unbounded strings.
func reflowHistory(rows []HistoryRow, cols int) ([]historyPaintRow, error) {
	var out []historyPaintRow
	current := historyPaintRow{}
	have := false
	flush := func() {
		if have {
			out = append(out, current)
			current = historyPaintRow{}
			have = false
		}
	}
	for i, row := range rows {
		join := i > 0 && row.Meta.Wrapped && row.ID == rows[i-1].ID+1
		if !join {
			flush()
			current.wrapped = i == 0 && row.Meta.Wrapped
			have = true
		}
		for x := 0; x < row.Meta.UsedColumns; x++ {
			c := row.Cells[x]
			if c.Width == 0 {
				continue
			}
			width := max(1, c.Width)
			if width > cols {
				// The retained source survives a one-column viewport. Omit only
				// an indivisible glyph that cannot be painted coherently; a later
				// width change rebuilds the complete text from this same window.
				x += width - 1
				continue
			}
			if current.used+width > cols {
				flush()
				current.wrapped = true
				have = true
			}
			// A previous physical row may carry painted trailing blanks. New
			// logical text replaces that padding instead of counting it as text.
			current.cells = current.cells[:current.used]
			current.cells = append(current.cells, c)
			if width == 2 {
				current.cells = append(current.cells, Cell{})
				x++
			}
			current.used += width
		}
		// Keep erase/background paint beyond the text extent. It remains
		// physical padding and never changes logical-line length or wrapping.
		current.cells = current.cells[:current.used]
		for x := row.Meta.UsedColumns; x < len(row.Cells) && len(current.cells) < cols; x++ {
			current.cells = append(current.cells, row.Cells[x])
		}
	}
	flush()
	return out, nil
}

type historyEmitter struct {
	write  func([]byte) error
	buffer strings.Builder
	err    error
}

func (e *historyEmitter) flush() {
	if e.err == nil && e.buffer.Len() > 0 {
		e.err = e.write([]byte(e.buffer.String()))
		e.buffer.Reset()
	}
}
func (e *historyEmitter) add(s string) {
	if e.err != nil {
		return
	}
	if e.buffer.Len()+len(s) > 64<<10 {
		e.flush()
	}
	if e.err == nil {
		e.buffer.WriteString(s)
	}
}
func (e *historyEmitter) cup(x, y int) { e.add(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1)) }
func (e *historyEmitter) packet(s string) {
	e.flush()
	if e.err == nil {
		e.err = e.write([]byte(s))
	}
}
func (e *historyEmitter) resetStyle() { e.add("\x1b[0m\x1b]8;;\x1b\\") }

// cells requires plain rendition on entry and restores it on return. Emit's
// prologue and the cells/blankTail/wrap helpers maintain that invariant, so
// default-only rows need no per-row SGR or OSC8 traffic.
func (e *historyEmitter) cells(cells []Cell, used int) {
	var style uv.Style
	var link uv.Link
	for x := 0; x < used && e.err == nil; x++ {
		c := cells[x]
		if c.Width == 0 {
			continue
		}
		if !style.Equal(&c.Style) {
			e.add("\x1b[0m" + c.Style.String())
			style = c.Style
		}
		if link != c.Link {
			e.add("\x1b]8;" + c.Link.Params + ";" + c.Link.URL + "\x1b\\")
			link = c.Link
		}
		if c.Content == "" {
			// Moving over an erased cell alone loses its background. ECH paints
			// it while retaining an empty-content cell in the terminal model.
			if c.Style.Bg != nil {
				e.add("\x1b[X")
			}
			e.add("\x1b[C")
		} else {
			e.add(c.Content)
		}
		if c.Width == 2 {
			x++
		}
	}
	if !style.Equal(&uv.Style{}) || link != (uv.Link{}) {
		e.resetStyle()
	}
}

// blankTail paints physical erase cells without extending the logical text
// extent. Restoring with CUP-to-start plus CUF avoids inventing default padding
// in Zellij; callers may immediately establish an outgoing soft wrap.
func (e *historyEmitter) blankTail(cells []Cell, used, cols, y int, soft bool) {
	painted := false
	for x := used; x < len(cells); x++ {
		// Autowrap paints the final early-wide gap without materializing a
		// copied space in Zellij. ECH here would destroy that provenance.
		if soft && x == cols-1 {
			continue
		}
		c := cells[x]
		if c.Style.Bg != nil {
			e.cup(x, y)
			e.add("\x1b[0m" + c.Style.String() + "\x1b[X")
			painted = true
		}
	}
	if painted {
		e.resetStyle()
		e.cup(0, y)
		if used > 0 {
			e.add(fmt.Sprintf("\x1b[%dC", used))
		}
	}
}

// wrap sets the following row's soft link without filling a short source with
// copied spaces. CUF does not materialize padding in Zellij, whereas CUP does.
func (e *historyEmitter) wrap(cells []Cell, used, cols int) {
	if used >= cols {
		e.add("x")
		return
	}
	if cols-used > 1 {
		e.add(fmt.Sprintf("\x1b[%dC", cols-used-1))
	}
	// Early-wide autowrap erases the last source column with the current
	// background. Match its retained blank paint instead of resetting it.
	coloredGap := len(cells) >= cols && cells[cols-1].Style.Bg != nil
	if coloredGap {
		gap := uv.Style{Bg: cells[cols-1].Style.Bg}
		e.add(gap.String())
	}
	e.add("界")
	if coloredGap {
		e.resetStyle()
	}
}
func (p HistoryRender) Emit(write func([]byte) error) error {
	if !p.dirty {
		return nil
	}
	if write == nil {
		return fmt.Errorf("terminal: missing history writer")
	}
	e := historyEmitter{write: write}
	// The bracket spans every chunk. An ALT packet below flushes it as its own
	// write, so the transition stays inside the bracket and its packet whole.
	e.add(syncBegin)
	// Dedicated packets let Presenter account for a completed mode transition
	// even if a later frame chunk fails; neither sequence is split by Emit.
	if p.enterAlt {
		e.packet(altEnter)
	}
	if p.leaveAlt {
		e.packet(altLeave)
	}
	// Re-asserted every frame on purpose, like Render's (#262 M2). The region
	// reset is convergent, not functional: the history push below sets 1;2r
	// and resets it itself before the lower rows.
	e.add("\x1b[?25l\x1b[?6l\x1b[r\x1b[?7h")
	e.resetStyle()
	cols, height := p.next.Geometry.Cols, p.next.Geometry.Rows
	cleanTop := p.reset || p.enterAlt
	if p.reset || p.enterAlt {
		if !p.next.AltScreen {
			e.add("\x1b[3J")
		}
		e.cup(0, 0)
		e.add(fmt.Sprintf("\x1b[%dL\x1b[2K", height))
	}
	if !p.next.AltScreen && len(p.rows) > 0 {
		if height > 1 {
			e.add("\x1b[1;2r")
		}
		for i, row := range p.rows {
			if height > 1 {
				e.cup(0, 1)
				e.add("\x1b[M\x1b[L\x1b[2K")
			}
			e.cup(0, 0)
			if i == 0 && !cleanTop {
				if !row.wrapped {
					e.add("\x1b[2K")
				} else {
					e.add(fmt.Sprintf("\x1b[%dX", cols))
				}
			}
			e.cells(row.cells, row.used)
			soft := p.history.ContinuesToScreen
			if i+1 < len(p.rows) {
				soft = p.rows[i+1].wrapped
			}
			e.blankTail(row.cells, row.used, cols, 0, soft)
			if soft {
				e.wrap(row.cells, row.used, cols)
			}
			if height > 1 {
				e.cup(0, 1)
				e.add("\n")
			} else if !soft {
				e.add("\r\n")
			}
			cleanTop = true
		}
	}
	e.add("\x1b[r")
	// Replace lower rows with canonical empty rows, then construct their desired
	// soft links while painting downward. Nothing here scrolls a viewport row.
	if height > 1 {
		e.cup(0, 1)
		e.add(fmt.Sprintf("\x1b[%dL", height-1))
		for y := 1; y < height; y++ {
			e.cup(0, y)
			e.add("\x1b[2K")
		}
	}
	e.cup(0, 0)
	if !p.next.rowMetadata(0).Wrapped {
		e.add("\x1b[2K")
	} else if !cleanTop {
		e.add(fmt.Sprintf("\x1b[%dX", cols))
	}
	for y := 0; y < height; y++ {
		e.cup(0, y)
		row := p.next.Cells[y*cols : (y+1)*cols]
		meta := p.next.rowMetadata(y)
		e.cells(row, meta.UsedColumns)
		soft := y+1 < height && p.next.rowMetadata(y+1).Wrapped
		e.blankTail(row, meta.UsedColumns, cols, y, soft)
		if soft {
			e.cup(0, y)
			if meta.UsedColumns > 0 {
				e.add(fmt.Sprintf("\x1b[%dC", meta.UsedColumns))
			}
			if meta.UsedColumns == cols {
				// CUF clamps before phantom position; rewrite the last leading cell to
				// acquire pending wrap without changing its value or style.
				last := cols - 1
				if last > 0 && row[last].Width == 0 {
					last--
				}
				e.cup(last, y)
				e.cells(row[last:], cols-last)
			}
			e.wrap(row, meta.UsedColumns, cols)
		}
	}
	e.resetStyle()
	e.add("\x1b[?7h")
	e.add(cursorEpilogue(p.next.Cursor))
	e.add(syncEnd)
	e.flush()
	return e.err
}
