package terminal

import (
	"fmt"
	"unsafe"
)

// RowMetadata preserves logical lines across cursor-addressed painting.
type RowMetadata struct {
	Wrapped     bool
	UsedColumns int
}
type HistoryCursor struct{ ClearEpoch, NextID uint64 }
type HistoryRow struct {
	ID    uint64
	Cells []Cell
	Meta  RowMetadata
}

// HistoryWindow is an owned bounded primary-screen suffix. Gaps in IDs denote
// loss; the first retained row never attaches to an unavailable predecessor.
type HistoryWindow struct {
	Cursor            HistoryCursor
	Rows              []HistoryRow
	ContinuesToScreen bool
}
type Publication struct {
	Frame   Frame
	History HistoryWindow
}

func (h HistoryWindow) Clone() HistoryWindow {
	rows := make([]HistoryRow, len(h.Rows))
	for i, row := range h.Rows {
		rows[i] = row
		rows[i].Cells = make([]Cell, len(row.Cells))
		for j, c := range row.Cells {
			rows[i].Cells[j] = cloneCell(c)
		}
	}
	h.Rows = rows
	return h
}
func (p Publication) Clone() Publication {
	return Publication{Frame: p.Frame.Clone(), History: p.History.Clone()}
}

func (h HistoryWindow) Validate() error {
	if len(h.Rows) > MaxHistoryLines {
		return fmt.Errorf("terminal: history line limit exceeded")
	}
	cells, bytes := 0, 0
	for i, row := range h.Rows {
		if row.ID >= h.Cursor.NextID || i > 0 && row.ID <= h.Rows[i-1].ID {
			return fmt.Errorf("terminal: invalid history identity")
		}
		if row.Meta.UsedColumns < 0 || row.Meta.UsedColumns > len(row.Cells) {
			return fmt.Errorf("terminal: invalid history extent")
		}
		cells += len(row.Cells)
		for j, c := range row.Cells {
			if err := validateCell(row.Cells, j); err != nil {
				return err
			}
			bytes += int(unsafe.Sizeof(Cell{})) + len(c.Content) + len(c.Link.URL) + len(c.Link.Params)
		}
	}
	if cells > MaxHistoryCells || bytes > 4<<20 {
		return fmt.Errorf("terminal: history payload limit exceeded")
	}
	return nil
}

func rowMetadata(cells []Cell) RowMetadata {
	m := RowMetadata{}
	for x, c := range cells {
		if c.Content != "" {
			m.UsedColumns = max(m.UsedColumns, min(len(cells), x+max(1, c.Width)))
		}
	}
	return m
}
func (f Frame) rowMetadata(y int) RowMetadata {
	if len(f.Rows) == f.Geometry.Rows {
		return f.Rows[y]
	}
	return rowMetadata(f.Cells[y*f.Geometry.Cols : (y+1)*f.Geometry.Cols])
}
