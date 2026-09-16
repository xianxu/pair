package vt

import uv "github.com/charmbracelet/ultraviolet"

// resizePrimary reflows the visible logical lines independently of retained
// history, as native Zellij does. Sparse staging is bounded by old cell count;
// the final rectangular allocation is bounded by the validated new geometry.
func (s *Screen) resizePrimary(width, height int, phantom bool) bool {
	if width == s.Width() {
		s.Resize(width, height)
		s.cur.X = min(s.cur.X, width-1)
		s.cur.Y = min(s.cur.Y, height-1)
		return phantom
	}
	type row struct {
		cells uv.Line
		meta  RowMetadata
	}
	var rows []row
	last := s.cur.Y
	for y := 0; y < s.Height(); y++ {
		if paintedColumns(s.buf.Line(y)) > 0 || s.rows[y].clipped != nil {
			last = max(last, y)
		}
	}
	last = min(last, s.Height()-1)
	cur, saved := s.cur.Position, s.saved.Position
	curMapped, savedMapped := false, false
	for start := 0; start <= last; {
		end := start + 1
		for end <= last && s.rows[end].Wrapped {
			end++
		}
		rows = append(rows, row{meta: RowMetadata{Wrapped: s.rows[start].Wrapped}})
		current := len(rows) - 1
		for y := start; y < end; y++ {
			extent := paintedColumns(s.buf.Line(y))
			if s.rows[y].clipped != nil {
				extent = s.rows[y].clipped.Width
			}
			cursorX := s.cur.X
			if phantom && y == s.cur.Y {
				cursorX++
			}
			if y == s.cur.Y {
				extent = max(extent, cursorX)
			}
			// A saved cursor maps into existing content but does not create blank rows.
			for x := 0; x < extent; {
				c := *s.CellAt(x, y)
				if x == 0 && s.rows[y].clipped != nil {
					c = *s.rows[y].clipped
				}
				cw := max(1, c.Width)
				if c.Width == 0 {
					x++
					continue
				}
				if len(rows[current].cells) > 0 && len(rows[current].cells)+cw > width {
					rows = append(rows, row{meta: RowMetadata{Wrapped: true}})
					current++
				}
				if y == s.cur.Y && x == cursorX {
					cur = uv.Pos(len(rows[current].cells), current)
					curMapped = true
				}
				if y == s.saved.Y && x == s.saved.X {
					saved = uv.Pos(len(rows[current].cells), current)
					savedMapped = true
				}
				rows[current].cells = append(rows[current].cells, c)
				if cw == 2 {
					rows[current].cells = append(rows[current].cells, uv.Cell{})
				}
				x += cw
			}
			if y == s.cur.Y && !curMapped {
				cur = uv.Pos(len(rows[current].cells), current)
				curMapped = true
			}
			if y == s.saved.Y && !savedMapped {
				saved = uv.Pos(min(s.saved.X, len(rows[current].cells)), current)
				savedMapped = true
			}
		}
		start = end
	}
	// A cursor at the logical end of a full row remains pending autowrap. A
	// subsequent printable performs the wrap; no synthetic blank line is retained.
	pending := cur.X >= width
	if pending {
		cur.X--
	}
	if saved.X >= width {
		saved.X = width - 1
	}
	drop := max(0, len(rows)-height)
	for i := 0; i < drop; i++ {
		if s.scrollback != nil {
			rows[i].meta.UsedColumns = usedColumns(rows[i].cells)
			s.scrollback.push(rows[i].cells, rows[i].meta)
		}
	}
	next := uv.NewRenderBuffer(width, height)
	next.Fill(&uv.Cell{Width: 1})
	metadata := make([]RowMetadata, height)
	for i := drop; i < len(rows); i++ {
		y := i - drop
		metadata[y] = rows[i].meta
		for x, c := range rows[i].cells {
			if c.Width > width {
				copy := c
				metadata[y].clipped = &copy
			} else if c.Width > 0 && x+c.Width <= width {
				next.SetCell(x, y, &c)
			}
		}
	}
	s.buf = next
	s.rows = metadata
	s.scroll = next.Bounds()
	s.buf.Touched = nil
	s.cur.Position = uv.Pos(max(0, min(width-1, cur.X)), max(0, min(height-1, cur.Y-drop)))
	s.saved.Position = uv.Pos(max(0, min(width-1, saved.X)), max(0, min(height-1, saved.Y-drop)))
	return pending
}

func paintedColumns(line uv.Line) int {
	n := usedColumns(line)
	for x, c := range line {
		if c.Style.Bg != nil || c.Link.URL != "" {
			n = max(n, x+max(1, c.Width))
		}
	}
	return min(n, len(line))
}
