package broadcast

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// PlaceholderText replaces a private frame's body for viewers.
const PlaceholderText = "The operator is switching threads"

// ViewerFrame is the frame viewers get for f. A private frame (the switcher,
// which lists the whole fleet) becomes a placeholder of the same geometry
// unless showSwitcher is set. The placeholder keeps all declared chrome rows, the tab bar:
// it shows the same thread labels as every public frame, so it reveals nothing
// new, and it keeps the LIVE indicator in front of viewers too.
func ViewerFrame(f terminal.Frame, class terminal.FrameClass, showSwitcher bool) (terminal.Frame, error) {
	if class == terminal.FramePublic || showSwitcher {
		return f, nil
	}
	if err := f.Validate(); err != nil {
		return terminal.Frame{}, err
	}
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	bodyRows := rows - f.ChromeRows
	body, err := terminal.StyledRows(placeholderBody(cols, bodyRows), cols, bodyRows)
	if err != nil {
		return terminal.Frame{}, err
	}
	child, err := terminal.PanelFrame(terminal.Geometry{Cols: cols, Rows: bodyRows}, body, terminal.Cursor{})
	if err != nil {
		return terminal.Frame{}, err
	}
	out, err := terminal.Compose(child, f.Geometry, f.Cells[bodyRows*cols:])
	if err != nil {
		return terminal.Frame{}, err
	}
	if len(f.Rows) == rows {
		copy(out.Rows[bodyRows:], f.Rows[bodyRows:])
	}
	return out, nil
}

// placeholderBody centres PlaceholderText in a body of rows×cols, truncated
// to the width when the terminal is narrower than the text.
func placeholderBody(cols, rows int) string {
	text := PlaceholderText
	for textwidth.Width(text) > cols {
		text = text[:len(text)-1]
	}
	pad := (cols - textwidth.Width(text)) / 2
	return strings.Repeat("\n", rows/2) + strings.Repeat(" ", pad) + text
}
