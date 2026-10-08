package broadcast

import (
	"fmt"
	"image/color"
)

// Theme is the operator's terminal palette, so viewers see the colours the
// operator sees: default foreground and background, and the 16 ANSI colours
// that indexed SGR colours 0–15 name. Each is "#rrggbb"; empty means the
// terminal didn't say, and the viewer keeps its own default for it.
type Theme struct {
	Foreground string     `json:"foreground,omitempty"`
	Background string     `json:"background,omitempty"`
	ANSI       [16]string `json:"ansi"`
}

// Hex formats a colour as "#rrggbb"; nil is "".
func Hex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
