package vt

import (
	"slices"
	"unsafe"

	uv "github.com/charmbracelet/ultraviolet"
)

// DefaultScrollbackSize is the default maximum number of lines in the scrollback buffer.
const DefaultScrollbackSize = 10000

// Scrollback represents a scrollback buffer that stores lines scrolled off the screen.
type Scrollback struct {
	lines                            []uv.Line
	maxLines                         int
	maxCells, maxBytes, cells, bytes int
}

// NewScrollback creates a new scrollback buffer with the given maximum number of lines.
func NewScrollback(maxLines int) *Scrollback {
	if maxLines <= 0 {
		maxLines = DefaultScrollbackSize
	}
	return &Scrollback{
		lines:    make([]uv.Line, 0, min(maxLines, 1000)), // Pre-allocate reasonable capacity
		maxLines: maxLines,
	}
}

// Push adds a line to the scrollback buffer.
// If the buffer is full, the oldest line is removed.
func (s *Scrollback) Push(line uv.Line) {
	if s == nil || s.maxLines <= 0 {
		return
	}

	// Find last non-empty cell to trim trailing empty cells.
	// This helps with wrapping and window resizing.
	lastNonEmpty := -1
	for i := len(line) - 1; i >= 0; i-- {
		c := &line[i]
		if !c.IsZero() && !c.Equal(&uv.EmptyCell) {
			lastNonEmpty = i
			break
		}
	}

	// Clone the line content up to and including the last non-empty cell
	cloned := slices.Clone(line[:lastNonEmpty+1])

	nbytes := lineBytes(cloned)
	if (s.maxCells > 0 && len(cloned) > s.maxCells) || (s.maxBytes > 0 && nbytes > s.maxBytes) {
		return
	}
	for len(s.lines) > 0 && (len(s.lines) >= s.maxLines || (s.maxCells > 0 && s.cells+len(cloned) > s.maxCells) || (s.maxBytes > 0 && s.bytes+nbytes > s.maxBytes)) {
		s.evict()
	}
	s.lines = append(s.lines, cloned)
	s.cells += len(cloned)
	s.bytes += nbytes
}

// PushN adds n lines from the buffer starting at line y to the scrollback.
func (s *Scrollback) PushN(buf *uv.RenderBuffer, y, n int) {
	if s == nil || buf == nil || n <= 0 {
		return
	}

	for i := range min(n, buf.Height()-y) {
		if line := buf.Line(y + i); line != nil {
			s.Push(line)
		}
	}
}

// Len returns the number of lines in the scrollback buffer.
func (s *Scrollback) Len() int {
	if s == nil {
		return 0
	}
	return len(s.lines)
}

// MaxLines returns the maximum number of lines the scrollback buffer can hold.
func (s *Scrollback) MaxLines() int {
	if s == nil {
		return 0
	}
	return s.maxLines
}

// SetMaxLines sets the maximum number of lines in the scrollback buffer.
// If the current number of lines exceeds the new maximum, oldest lines are removed.
func (s *Scrollback) SetMaxLines(maxLines int) {
	if s == nil || maxLines <= 0 {
		return
	}

	s.maxLines = maxLines
	for len(s.lines) > maxLines {
		s.evict()
	}
}

// Line returns the line at the given index.
// Index 0 is the oldest line, Len()-1 is the most recent.
// Returns nil if index is out of bounds.
func (s *Scrollback) Line(index int) uv.Line {
	if s == nil || index < 0 || index >= len(s.lines) {
		return nil
	}
	return s.lines[index]
}

// Lines returns all lines in the scrollback buffer.
// Index 0 is the oldest line.
func (s *Scrollback) Lines() []uv.Line {
	if s == nil {
		return nil
	}
	return s.lines
}

// Clear removes all lines from the scrollback buffer.
func (s *Scrollback) Clear() {
	if s == nil {
		return
	}
	clear(s.lines)
	s.lines = s.lines[:0]
	s.cells = 0
	s.bytes = 0
}

// CellAt returns the cell at the given position in the scrollback buffer.
// x is the column, y is the line index (0 = oldest).
// Returns nil if position is out of bounds.
func (s *Scrollback) CellAt(x, y int) *uv.Cell {
	line := s.Line(y)
	if line == nil || x < 0 || x >= len(line) {
		return nil
	}
	return &line[x]
}

func lineBytes(line uv.Line) int {
	n := len(line) * int(unsafe.Sizeof(uv.Cell{}))
	for i := range line {
		n += cellPayload(&line[i])
	}
	return n
}
func (s *Scrollback) evict() {
	s.cells -= len(s.lines[0])
	s.bytes -= lineBytes(s.lines[0])
	s.lines = slices.Delete(s.lines, 0, 1)
}
func (s *Scrollback) setLimits(lines, cells, bytes int) {
	s.maxLines = lines
	s.maxCells = cells
	s.maxBytes = bytes
	for len(s.lines) > 0 && (len(s.lines) > lines || s.cells > cells || s.bytes > bytes) {
		s.evict()
	}
}
