package vt

import uv "github.com/charmbracelet/ultraviolet"

// CursorStyle is a DECSCUSR shape family. The zero value leaves shape and
// blink to the host terminal's configured default (DECSCUSR 0, RIS; pair #283).
type CursorStyle int

// Cursor styles.
const (
	CursorDefault CursorStyle = iota
	CursorBlock
	CursorUnderline
	CursorBar
)

// Cursor represents a cursor in a terminal.
type Cursor struct {
	Pen  uv.Style
	Link uv.Link

	uv.Position

	Style  CursorStyle
	Steady bool // Not blinking; meaningful only for an explicit Style
	Hidden bool
}
