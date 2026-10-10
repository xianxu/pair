package wrapcmd

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// Tail renders the last n rows of the agent pane (scrollback, then screen)
// with couchmessage's markup (pair#425). It reads the emulator the wrapper
// already keeps, so it stores nothing. The cues are generic terminal
// bookkeeping -- faint, reverse, the cursor -- never a reading of the agent.
func (m *terminalModel) Tail(n int) couchmessage.Tail {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		s := m.final
		rows := make([][]uv.Cell, s.Height)
		for y := range rows {
			rows[y] = s.Cells[y*s.Width : (y+1)*s.Width]
		}
		return renderTail(rows, n, s.Cursor.X, s.Cursor.Y, vt.Cursor{Hidden: !s.CursorVisible})
	}
	var rows [][]uv.Cell
	if !m.emulator.IsAltScreen() {
		// The alternate screen has no scrollback of its own.
		history := m.emulator.Scrollback().Lines()
		if len(history) > n {
			history = history[len(history)-n:]
		}
		for _, line := range history {
			rows = append(rows, []uv.Cell(line))
		}
	}
	width, height := m.emulator.Width(), m.emulator.Height()
	for y := 0; y < height; y++ {
		row := make([]uv.Cell, width)
		for x := range row {
			if cell := m.emulator.CellAt(x, y); cell != nil {
				row[x] = *cell
			}
		}
		rows = append(rows, row)
	}
	cursor := m.emulator.Cursor()
	return renderTail(rows, n, cursor.X, len(rows)-height+cursor.Y, cursor)
}

// agentTail is the endpoint's tail probe. Like settledNow, it reads whichever
// terminal model the proxy holds now.
func (p *proxy) agentTail(n int) couchmessage.Tail {
	if p.terminal == nil {
		return couchmessage.Tail{Lines: []string{}}
	}
	return p.terminal.Tail(n)
}

var tailShapes = map[vt.CursorStyle]string{
	vt.CursorDefault: "default", vt.CursorBlock: "block", vt.CursorUnderline: "underline", vt.CursorBar: "bar",
}

// renderTail keeps the last n rows, dropping blank rows below the cursor, and
// renders each one. cursorX/cursorY index rows.
func renderTail(rows [][]uv.Cell, n, cursorX, cursorY int, cursor vt.Cursor) couchmessage.Tail {
	end := len(rows)
	for end > 0 && end-1 > cursorY && tailRowBlank(rows[end-1]) {
		end--
	}
	start := max(end-n, 0)
	tail := couchmessage.Tail{Lines: []string{}}
	c := &couchmessage.TailCursor{Shape: tailShapes[cursor.Style], Steady: cursor.Steady, Hidden: cursor.Hidden}
	if cursorY >= start && cursorY < end {
		c.Row, c.Col = cursorY-start+1, cursorX+1
	}
	tail.Cursor = c
	for y := start; y < end; y++ {
		marker := -1
		if y == cursorY && !cursor.Hidden {
			marker = cursorX
		}
		tail.Lines = append(tail.Lines, renderTailRow(rows[y], marker))
	}
	return couchmessage.BoundTail(tail)
}

// tailVisible reports whether a cell shows anything a reader could see: a
// glyph, or a reversed blank (an agent's drawn cursor).
func tailVisible(c uv.Cell) bool {
	return (c.Content != "" && c.Content != " ") || c.Style.Attrs&uv.AttrReverse != 0
}

func tailRowBlank(row []uv.Cell) bool {
	for _, c := range row {
		if tailVisible(c) {
			return false
		}
	}
	return true
}

// renderTailRow renders one row; marker is the cursor column, or -1. Spans
// close at the row's end, so every line stands alone. A plain blank between
// two cells of the same style stays inside their span (an agent may dim only
// the words), so a span changes only at a visible cell.
func renderTailRow(row []uv.Cell, marker int) string {
	end := 0
	for x, c := range row {
		if tailVisible(c) {
			end = x + max(c.Width, 1)
		}
	}
	end = max(end, marker+1)
	var b, pending strings.Builder
	dim, rev := false, false
	closeSpans := func() {
		if rev {
			b.WriteString("‹/rev›")
		}
		if dim {
			b.WriteString("‹/dim›")
		}
		dim, rev = false, false
	}
	for x := 0; x < end; x++ {
		var c uv.Cell
		if x < len(row) {
			c = row[x]
		}
		if x == marker {
			pending.WriteString("‹cursor›")
		}
		if c.Content == "" && c.Width == 0 && x > 0 && x < len(row) && row[x-1].Width > 1 {
			continue // the second column of a wide glyph
		}
		if !tailVisible(c) {
			pending.WriteByte(' ')
			continue
		}
		wantDim, wantRev := c.Style.Attrs&uv.AttrFaint != 0, c.Style.Attrs&uv.AttrReverse != 0
		if wantDim != dim || wantRev != rev {
			closeSpans()
			b.WriteString(pending.String())
			pending.Reset()
			if wantDim {
				b.WriteString("‹dim›")
			}
			if wantRev {
				b.WriteString("‹rev›")
			}
			dim, rev = wantDim, wantRev
		}
		b.WriteString(pending.String())
		pending.Reset()
		// A literal ‹ is doubled, so screen text cannot spoof the markup.
		b.WriteString(strings.ReplaceAll(c.Content, "‹", "‹‹"))
	}
	closeSpans()
	b.WriteString(strings.TrimRight(pending.String(), " "))
	return b.String()
}
