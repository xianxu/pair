package vt

import (
	"image/color"
	"slices"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// RowMetadata describes logical text, independently of cell paint damage.
// Wrapped continues the previous physical row. UsedColumns excludes unprinted
// trailing padding, but includes explicitly printed spaces and wide cells.
type RowMetadata struct {
	Wrapped     bool
	UsedColumns int
	// clipped retains an indivisible glyph while the viewport is narrower than
	// its width. It is immutable, cleared by writes, and never exposed as paint.
	clipped *uv.Cell
}

type HistoryRow struct {
	ID    uint64
	Cells []uv.Cell
	Meta  RowMetadata
}

// HistorySnapshot owns its rows/cells. IDs count append attempts (including
// rejected rows); gaps are loss boundaries. Clear never resets NextID.
type HistorySnapshot struct {
	ClearEpoch, NextID uint64
	Rows               []HistoryRow
	ContinuesToScreen  bool
}

func (e *Emulator) RowMetadata(y int) RowMetadata { return e.scr.rowMetadata(y) }
func (s *Screen) rowMetadata(y int) RowMetadata {
	if y < 0 || y >= s.Height() {
		return RowMetadata{}
	}
	m := s.rows[y]
	m.UsedColumns = usedColumns(s.buf.Line(y))
	return m
}
func usedColumns(line uv.Line) int {
	used := 0
	for x, c := range line {
		if c.Content != "" {
			used = max(used, min(len(line), x+max(1, c.Width)))
		}
	}
	return used
}
func (e *Emulator) HistorySnapshot() HistorySnapshot {
	s := &e.scrs[0]
	h := HistorySnapshot{ContinuesToScreen: s.rowMetadata(0).Wrapped}
	if b := s.scrollback; b != nil {
		h.ClearEpoch, h.NextID = b.clearEpoch, b.nextID
		if len(b.ids) == 0 || b.ids[len(b.ids)-1]+1 != b.nextID {
			h.ContinuesToScreen = false
		}
		h.Rows = make([]HistoryRow, len(b.lines))
		for i, line := range b.lines {
			cells := slices.Clone(line)
			for j := range cells {
				cells[j].Style.Fg = historyColor(cells[j].Style.Fg)
				cells[j].Style.Bg = historyColor(cells[j].Style.Bg)
				cells[j].Style.UnderlineColor = historyColor(cells[j].Style.UnderlineColor)
			}
			h.Rows[i] = HistoryRow{ID: b.ids[i], Cells: cells, Meta: b.metadata[i]}
		}
	}
	return h
}
func historyColor(c color.Color) color.Color {
	switch c.(type) {
	case nil, ansi.BasicColor, ansi.IndexedColor, ansi.RGBColor, color.RGBA, color.NRGBA, color.RGBA64, color.NRGBA64:
		return c
	}
	r, g, b, a := c.RGBA()
	return color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: uint16(a)}
}

// legacyLines preserves the existing String/Render display contract while
// CellAt exposes the distinction between an unprinted blank and a printed space.
func (s *Screen) legacyLines() uv.Lines {
	lines := make(uv.Lines, s.Height())
	for y := range lines {
		lines[y] = slices.Clone(s.buf.Line(y))
		for x := range lines[y] {
			if lines[y][x].Width == 1 && lines[y][x].Content == "" {
				lines[y][x].Content = " "
			}
		}
	}
	return lines
}

// historyLine includes a temporarily clipped indivisible glyph in retained
// source text, while CellAt continues to expose only valid viewport paint.
func (s *Screen) historyLine(y int) (uv.Line, RowMetadata) {
	m := s.rowMetadata(y)
	if c := m.clipped; c != nil {
		m.clipped = nil
		m.UsedColumns = c.Width
		return uv.Line{*c, {}}, m
	}
	return s.buf.Line(y), m
}
