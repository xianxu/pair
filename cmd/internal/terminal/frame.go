package terminal

import (
	"fmt"
	"image/color"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type Cell = uv.Cell

// Cursor coordinates are zero based. Shape is DECSCUSR's shape family:
// 0/default or 1/block, 2/underline, 3/bar; Blink is independent.
type Cursor struct {
	X, Y           int
	Visible, Blink bool
	Shape          int
}

// Frame is an owned immutable publication. Call Clone before retaining data
// supplied by a mutable backend or another owner.
type Frame struct {
	EndpointID                string
	Generation, GeometryEpoch uint64
	Geometry                  Geometry
	Cells                     []Cell
	Rows                      []RowMetadata
	Cursor                    Cursor
	AltScreen                 bool
}

func (f Frame) Clone() Frame {
	f.Cells = cloneCells(f.Cells)
	f.Rows = append([]RowMetadata(nil), f.Rows...)
	return f
}

func cloneCells(cells []Cell) []Cell {
	if cells == nil {
		return nil
	}
	owned := make([]Cell, len(cells))
	for i, c := range cells {
		owned[i] = cloneCell(c)
	}
	return owned
}

func cloneCell(c Cell) Cell {
	c.Style.Fg = cloneColor(c.Style.Fg)
	c.Style.Bg = cloneColor(c.Style.Bg)
	c.Style.UnderlineColor = cloneColor(c.Style.UnderlineColor)
	return c
}

func cloneColor(c color.Color) color.Color {
	switch c.(type) {
	case nil, ansi.BasicColor, ansi.IndexedColor, ansi.RGBColor, color.RGBA, color.NRGBA, color.RGBA64, color.NRGBA64:
		return c
	}
	r, g, b, a := c.RGBA()
	return color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: uint16(a)}
}

func plainText(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validLink(c Cell) bool {
	return len(c.Link.URL) <= MaxLinkBytes && len(c.Link.Params) <= MaxLinkBytes && plainText(c.Link.URL) && plainText(c.Link.Params)
}

func (f Frame) Validate() error {
	if err := f.Geometry.Validate(); err != nil {
		return err
	}
	if len(f.Cells) != f.Geometry.Cols*f.Geometry.Rows {
		return fmt.Errorf("terminal: frame cell count does not match geometry")
	}
	if f.Cursor.X < 0 || f.Cursor.X >= f.Geometry.Cols || f.Cursor.Y < 0 || f.Cursor.Y >= f.Geometry.Rows || f.Cursor.Shape < 0 || f.Cursor.Shape > 3 {
		return fmt.Errorf("terminal: invalid frame cursor")
	}
	if len(f.Rows) != 0 && len(f.Rows) != f.Geometry.Rows {
		return fmt.Errorf("terminal: row metadata count does not match geometry")
	}
	for y, m := range f.Rows {
		if m.UsedColumns < 0 || m.UsedColumns > f.Geometry.Cols {
			return fmt.Errorf("terminal: invalid row metadata %d", y)
		}
	}
	for y := 0; y < f.Geometry.Rows; y++ {
		row := f.Cells[y*f.Geometry.Cols : (y+1)*f.Geometry.Cols]
		for x := range row {
			if err := validateCell(row, x); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateCell(cells []Cell, i int) error {
	c := cells[i]
	if !plainText(c.Content) || len(c.Content) > MaxClusterBytes || !validLink(c) {
		return fmt.Errorf("terminal: unsafe cell %d", i)
	}
	if c.Width < 0 || c.Width > 2 {
		return fmt.Errorf("terminal: unsupported cell width at %d", i)
	}
	if c.Content != "" {
		cluster, width := ansi.FirstGraphemeCluster(c.Content, ansi.GraphemeWidth)
		if len(cluster) != len(c.Content) || width != c.Width {
			return fmt.Errorf("terminal: content does not fit cell %d", i)
		}
	}
	if c.Width == 0 {
		// Zero cells are also the backend's ordinary blank representation.
		if c.Content != "" {
			return fmt.Errorf("terminal: content in continuation at %d", i)
		}
	}
	if c.Width == 2 && (i == len(cells)-1 || cells[i+1].Width != 0 || cells[i+1].Content != "") {
		return fmt.Errorf("terminal: broken wide cell at %d", i)
	}
	return nil
}
