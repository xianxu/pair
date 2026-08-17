package wrapcmd

import (
	"strconv"
	"sync"
)

// Muse renders its input composer as a single prompt line "› " (U+203A, UTF-8 e2 9f a9 20)
// with FG 38;2;90;160;255. Empty and non-empty prompts both contain that glyph.
// When the composer is active the glyph is painted at the cursor row and the
// cursor is left visible at col 3 (just after "› "). We track rows where the
// prompt glyph was painted and consider the composer active when the visible
// cursor is on or adjacent to such a row. This is the Muse analogue of
// codexComposerTracker's BG+EL heuristic, but prompt-anchored — observed in
// scrollback-fix-tty-muse.raw (e.g. 30;1H "› " empty, 9;1H "› work on #140" filled).
const museComposerMinRows = 1

var musePrompt = []byte{0xe2, 0x9f, 0xa9}

type museComposerTracker struct {
	mu sync.Mutex

	rows int
	cols int

	cursorRow     int
	cursorCol     int
	cursorVisible bool

	// bg retained only for SGR parsing side-effects (cursor tracking needs
	// to consume SGR, but Muse prompt detection does not use BG).
	bg          string
	promptRows  map[int]bool
	pending     []byte
}

type museComposerState struct {
	rows          int
	cols          int
	cursorRow     int
	cursorCol     int
	cursorVisible bool
	promptRows    map[int]bool
}

func newMuseComposerTracker() *museComposerTracker {
	return &museComposerTracker{
		cursorRow:  1,
		cursorCol:  1,
		promptRows: map[int]bool{},
	}
}

func (t *museComposerTracker) resize(rows, cols int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rows = rows
	t.cols = cols
	if rows <= 0 || cols <= 0 {
		t.cursorRow = 0
		t.cursorCol = 0
		t.promptRows = map[int]bool{}
		return
	}
	if t.cursorRow < 1 || t.cursorRow > rows {
		t.cursorRow = clampInt(t.cursorRow, 1, rows)
	}
	if t.cursorCol < 1 || t.cursorCol > cols {
		t.cursorCol = clampInt(t.cursorCol, 1, cols)
	}
	for row := range t.promptRows {
		if row < 1 || row > rows {
			delete(t.promptRows, row)
		}
	}
}

func (t *museComposerTracker) feed(data []byte) {
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
		// Detect prompt glyph "›" (e2 9f a9) as composer chrome.
		if b == 0xe2 && i+2 < len(data) && data[i+1] == 0x9f && data[i+2] == 0xa9 {
			t.notePromptRow()
			// One glyph, advance past 3 bytes. Cursor moves one cell (plus
			// the following space will be handled next iteration).
			t.cursorCol++
			t.clampCursor()
			i += 3
			continue
		}
		// Partial prompt at tail — carry over.
		if b == 0xe2 && len(data)-i < 3 {
			t.pending = append([]byte(nil), data[i:]...)
			return
		}
		if b == 0xe2 && i+1 < len(data) && data[i+1] == 0x9f && len(data)-i < 3 {
			t.pending = append([]byte(nil), data[i:]...)
			return
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

func (t *museComposerTracker) state() museComposerState {
	t.mu.Lock()
	defer t.mu.Unlock()

	rows := make(map[int]bool, len(t.promptRows))
	for row, ok := range t.promptRows {
		rows[row] = ok
	}
	return museComposerState{
		rows:          t.rows,
		cols:          t.cols,
		cursorRow:     t.cursorRow,
		cursorCol:     t.cursorCol,
		cursorVisible: t.cursorVisible,
		promptRows:    rows,
	}
}

func (s museComposerState) active() bool {
	if s.rows <= 0 || s.cols <= 0 || !s.cursorVisible || s.cursorRow <= 0 {
		return false
	}
	count := 0
	for row := range s.promptRows {
		if row >= s.cursorRow-1 && row <= s.cursorRow+1 {
			count++
		}
	}
	return count >= museComposerMinRows
}

func (t *museComposerTracker) applyEscape(seq []byte) {
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
		// K alone does not imply prompt — prompt detection is byte-driven.
		// But an EL that clears the prompt line should not keep stale state:
		// if the line was repainted without prompt, it will be handled by
		// the absence of a subsequent prompt mark. We do not auto-clear here.
	}
}

func (t *museComposerTracker) notePromptRow() {
	if t.cursorRow <= 0 {
		return
	}
	t.promptRows[t.cursorRow] = true
}

func (t *museComposerTracker) applyEraseDisplay(params string) {
	parts := splitParams(params)
	mode := intParam(parts[0], 0)
	for row := range t.promptRows {
		switch mode {
		case 0:
			if row >= t.cursorRow {
				delete(t.promptRows, row)
			}
		case 1:
			if row <= t.cursorRow {
				delete(t.promptRows, row)
			}
		case 2, 3:
			delete(t.promptRows, row)
		}
	}
}

func (t *museComposerTracker) applyCursorPosition(params string) {
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

func (t *museComposerTracker) applySGR(params string) {
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

func (t *museComposerTracker) clampCursor() {
	if t.rows > 0 {
		t.cursorRow = clampInt(t.cursorRow, 1, t.rows)
	}
	if t.cols > 0 {
		t.cursorCol = clampInt(t.cursorCol, 1, t.cols)
	}
}
