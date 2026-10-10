package couchmessage

import (
	"fmt"
	"strings"
)

// A slot's tail (pair#425) is its wrapper's in-memory terminal rendered as
// text with light markup: faint runs as ‹dim›…‹/dim›, reverse video as
// ‹rev›…‹/rev›, and ‹cursor› before the cursor cell; a literal ‹ in the
// text is written ‹‹. Pair renders and never
// classifies: whether the slot is busy is the reader's judgement.
const (
	// MaxTailLines bounds one tail request.
	MaxTailLines = 200
	// MaxTailBytes bounds the rendered lines of one tail response, well under
	// MaxFrameBytes; the oldest lines go first and are counted in Truncated.
	MaxTailBytes = 128 * 1024
)

// TailCursor is the cursor relative to the returned lines: Row and Col are
// 1-based, and Row 0 means the cursor is outside them.
type TailCursor struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Shape  string `json:"shape"`
	Steady bool   `json:"steady,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
}

// String is the one-line summary printed under a tail.
func (c TailCursor) String() string {
	switch {
	case c.Row <= 0:
		return "outside the tail"
	case c.Hidden:
		return fmt.Sprintf("hidden at %d,%d", c.Row, c.Col)
	}
	shape := c.Shape
	if c.Steady {
		shape += " steady"
	}
	return fmt.Sprintf("%d,%d %s", c.Row, c.Col, shape)
}

type Tail struct {
	Lines     []string    `json:"lines"`
	Cursor    *TailCursor `json:"cursor,omitempty"`
	Truncated int         `json:"truncated,omitempty"`
}

// ValidTailLines reports whether n is a tail request's line count.
func ValidTailLines(n int) error {
	if n <= 0 || n > MaxTailLines {
		return fmt.Errorf("tail lines must be 1..%d, not %d", MaxTailLines, n)
	}
	return nil
}

// BoundTail drops the oldest lines until the rendered text fits MaxTailBytes,
// moving the cursor's row with them.
func BoundTail(t Tail) Tail {
	size := 0
	for _, line := range t.Lines {
		size += len(line) + 1
	}
	for size > MaxTailBytes && len(t.Lines) > 1 {
		size -= len(t.Lines[0]) + 1
		t.Lines = t.Lines[1:]
		t.Truncated++
		if t.Cursor != nil && t.Cursor.Row > 0 {
			t.Cursor.Row--
		}
	}
	if len(t.Lines) == 1 && len(t.Lines[0]) > MaxTailBytes {
		t.Lines[0] = strings.ToValidUTF8(t.Lines[0][:MaxTailBytes], "")
	}
	return t
}
