package wrapcmd

import (
	"strconv"
	"sync"

	"github.com/xianxu/pair/cmd/internal/ansi"
)

const (
	codexComposerBG       = "2;57;57;57"
	codexComposerBandRows = 4
	codexComposerMinRows  = 2
)

type codexComposerTracker struct {
	mu sync.Mutex

	rows int
	cols int

	cursorRow     int
	cursorCol     int
	cursorVisible bool

	bg          string
	paintedRows map[int]bool
	pending     []byte
}

type codexComposerState struct {
	rows          int
	cols          int
	cursorRow     int
	cursorCol     int
	cursorVisible bool
	paintedRows   map[int]bool
}

func newCodexComposerTracker() *codexComposerTracker {
	return &codexComposerTracker{
		cursorRow:   1,
		cursorCol:   1,
		paintedRows: map[int]bool{},
	}
}

func (t *codexComposerTracker) resize(rows, cols int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rows = rows
	t.cols = cols
	if rows <= 0 || cols <= 0 {
		t.cursorRow = 0
		t.cursorCol = 0
		t.paintedRows = map[int]bool{}
		return
	}
	if t.cursorRow < 1 || t.cursorRow > rows {
		t.cursorRow = clampInt(t.cursorRow, 1, rows)
	}
	if t.cursorCol < 1 || t.cursorCol > cols {
		t.cursorCol = clampInt(t.cursorCol, 1, cols)
	}
	for row := range t.paintedRows {
		if row < 1 || row > rows || !t.rowInComposerBand(row) {
			delete(t.paintedRows, row)
		}
	}
}

func (t *codexComposerTracker) feed(data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.pending) > 0 {
		combined := make([]byte, 0, len(t.pending)+len(data))
		combined = append(combined, t.pending...)
		combined = append(combined, data...)
		data = combined
		t.pending = nil
	}

	for i := 0; i < len(data); {
		b := data[i]
		if b == 0x1b {
			if seqLen := ansiSequenceLen(data[i:]); seqLen > 0 {
				t.applyEscape(data[i : i+seqLen])
				i += seqLen
				continue
			}
			if len(data)-i < pendingMax {
				t.pending = append([]byte(nil), data[i:]...)
				return
			}
			i++
			continue
		}
		switch b {
		case '\r':
			t.cursorCol = 1
		case '\n':
			t.cursorRow++
			t.cursorCol = 1
		default:
			if b >= 0x20 && b != 0x7f {
				t.cursorCol++
			}
		}
		t.clampCursor()
		i++
	}
}

func (t *codexComposerTracker) state() codexComposerState {
	t.mu.Lock()
	defer t.mu.Unlock()

	rows := make(map[int]bool, len(t.paintedRows))
	for row, ok := range t.paintedRows {
		rows[row] = ok
	}
	return codexComposerState{
		rows:          t.rows,
		cols:          t.cols,
		cursorRow:     t.cursorRow,
		cursorCol:     t.cursorCol,
		cursorVisible: t.cursorVisible,
		paintedRows:   rows,
	}
}

func (s codexComposerState) active() bool {
	if s.rows <= 0 || s.cols <= 0 || !s.cursorVisible {
		return false
	}
	if !codexRowInComposerBand(s.cursorRow, s.rows) {
		return false
	}
	count := 0
	for row := range s.paintedRows {
		if codexRowInComposerBand(row, s.rows) {
			count++
		}
	}
	return count >= codexComposerMinRows
}

func (t *codexComposerTracker) applyEscape(seq []byte) {
	if len(seq) < 3 || seq[0] != 0x1b || seq[1] != '[' {
		return
	}
	final := seq[len(seq)-1]
	params := string(seq[2 : len(seq)-1])
	switch final {
	case 'H', 'f':
		t.applyCursorPosition(params)
	case 'h', 'l':
		if params == "?25" {
			t.cursorVisible = final == 'h'
		}
	case 'm':
		t.applySGR(params)
	case 'J':
		t.applyEraseDisplay(params)
	case 'K':
		if t.rowInComposerBand(t.cursorRow) {
			if t.bg == codexComposerBG {
				t.paintedRows[t.cursorRow] = true
			} else {
				delete(t.paintedRows, t.cursorRow)
			}
		}
	}
}

func (t *codexComposerTracker) applyEraseDisplay(params string) {
	parts := splitParams(params)
	mode := intParam(parts[0], 0)
	for row := range t.paintedRows {
		switch mode {
		case 0:
			if row >= t.cursorRow {
				delete(t.paintedRows, row)
			}
		case 1:
			if row <= t.cursorRow {
				delete(t.paintedRows, row)
			}
		case 2, 3:
			delete(t.paintedRows, row)
		}
	}
}

func (t *codexComposerTracker) applyCursorPosition(params string) {
	parts := splitParams(params)
	row, col := 1, 1
	if len(parts) > 0 && parts[0] != "" {
		if n, err := strconv.Atoi(parts[0]); err == nil && n > 0 {
			row = n
		}
	}
	if len(parts) > 1 && parts[1] != "" {
		if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
			col = n
		}
	}
	t.cursorRow = row
	t.cursorCol = col
	t.clampCursor()
}

func (t *codexComposerTracker) applySGR(params string) {
	parts := splitParams(params)
	if len(parts) == 0 {
		t.bg = ""
		return
	}
	for i := 0; i < len(parts); i++ {
		p := intParam(parts[i], 0)
		switch p {
		case 0:
			t.bg = ""
		case 49:
			t.bg = ""
		case 48:
			if i+4 < len(parts) && parts[i+1] == "2" {
				t.bg = "2;" + parts[i+2] + ";" + parts[i+3] + ";" + parts[i+4]
				i += 4
			} else if i+2 < len(parts) && parts[i+1] == "5" {
				t.bg = "5;" + parts[i+2]
				i += 2
			}
		}
	}
}

func (t *codexComposerTracker) rowInComposerBand(row int) bool {
	return codexRowInComposerBand(row, t.rows)
}

func codexRowInComposerBand(row, rows int) bool {
	if rows <= 0 || row <= 0 {
		return false
	}
	top := rows - codexComposerBandRows + 1
	if top < 1 {
		top = 1
	}
	return row >= top && row < rows
}

func (t *codexComposerTracker) clampCursor() {
	if t.rows > 0 {
		t.cursorRow = clampInt(t.cursorRow, 1, t.rows)
	}
	if t.cols > 0 {
		t.cursorCol = clampInt(t.cursorCol, 1, t.cols)
	}
}

func ansiSequenceLen(data []byte) int {
	if len(data) < 2 || data[0] != 0x1b {
		return 0
	}
	return ansi.SequenceLen(data)
}

func splitParams(params string) []string {
	if params == "" {
		return []string{"0"}
	}
	raw := splitBytes([]byte(params), ';')
	out := make([]string, len(raw))
	for i, p := range raw {
		out[i] = string(p)
	}
	return out
}

func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
