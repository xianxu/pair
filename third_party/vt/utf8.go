package vt

import (
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// handlePrint keeps the last printed cluster amendable across Write boundaries.
// A control operation seals it; a transport boundary does not.
func (e *Emulator) handlePrint(r rune) {
	next := string(r)
	if len(next) > e.limits.GraphemeBytes {
		e.flushGrapheme()
		return
	}
	if e.cluster != "" {
		joined := e.cluster + next
		first, width := ansi.FirstGraphemeCluster(joined, ansi.GraphemeWidth)
		if len(first) == len(joined) {
			if len(joined) > e.limits.GraphemeBytes {
				e.clusterDropping = true
				return
			}
			if e.clusterDropping {
				return
			}
			e.cluster = joined
			if width == e.clusterWidth {
				c := *e.scr.CellAt(e.clusterX, e.clusterY)
				c.Content = joined
				e.scr.SetCell(e.clusterX, e.clusterY, &c)
			} else {
				// Undo the provisional cell before a width-changing extension. Wrapping
				// happens once the extended cluster's width is known, including scrolling.
				e.scr.SetCell(e.clusterX, e.clusterY, &e.clusterOld[0])
				if e.clusterX+1 < e.scr.Width() {
					e.scr.SetCell(e.clusterX+1, e.clusterY, &e.clusterOld[1])
				}
				e.scr.setCursor(e.clusterX, e.clusterY, false)
				e.atPhantom = false
				e.handleGrapheme(joined, width)
			}
			return
		}
	}
	e.clusterDropping = false
	e.cluster = next
	_, width := ansi.FirstGraphemeCluster(next, ansi.GraphemeWidth)
	e.handleGrapheme(next, width)
}

// flushGrapheme seals the provisional cluster at semantic boundaries.
func (e *Emulator) flushGrapheme() {
	e.cluster = ""
	e.clusterDropping = false
	e.clusterOld = [2]uv.Cell{}
}

// handleGrapheme handles UTF-8 graphemes.
func (e *Emulator) handleGrapheme(content string, width int) {
	awm := e.isModeSet(ansi.ModeAutoWrap)
	cell := uv.Cell{
		Content: content,
		Width:   width,
		Style:   e.scr.cursorPen(),
		Link:    e.scr.cursorLink(),
	}

	x, y := e.scr.CursorPosition()
	if (e.atPhantom || x+width > e.scr.Width()) && awm {
		// moves cursor down similar to [Terminal.linefeed] except it doesn't
		// respects [ansi.LNM] mode.
		// This will reset the phantom state i.e. pending wrap state.
		e.index()
		_, y = e.scr.CursorPosition()
		x = 0
	}

	// Retain the overwritten cells for a width-changing cluster extension.
	e.clusterX, e.clusterY, e.clusterWidth = x, y, width
	if c := e.scr.CellAt(x, y); c != nil {
		e.clusterOld[0] = *c
	}
	if c := e.scr.CellAt(x+1, y); c != nil {
		e.clusterOld[1] = *c
	}

	// Handle character set mappings
	if len(content) == 1 { //nolint:nestif
		var charset CharSet
		c := content[0]
		if e.gsingle > 1 && e.gsingle < 4 {
			charset = e.charsets[e.gsingle]
			e.gsingle = 0
		} else if c < 128 {
			charset = e.charsets[e.gl]
		} else {
			charset = e.charsets[e.gr]
		}

		if charset != nil {
			if r, ok := charset[c]; ok {
				cell.Content = r
				cell.Width = 1
			}
		}
	}

	if cell.Width == 1 && len(content) == 1 {
		e.lastChar, _ = utf8.DecodeRuneInString(content)
	}

	e.scr.SetCell(x, y, &cell)

	// Handle phantom state at the end of the line
	e.atPhantom = awm && x+cell.Width >= e.scr.Width()
	if !e.atPhantom {
		x += cell.Width
	} else {
		x = e.scr.Width() - 1
	}

	// NOTE: We don't reset the phantom state here, we handle it up above.
	e.scr.setCursor(x, y, false)
}
