package wrapcmd

import (
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestTerminalRowTextPreservesErasedSpacesAndWideContinuations(t *testing.T) {
	s := terminalSnapshot{Width: 6, Height: 1, Cells: []uv.Cell{
		{Content: "界", Width: 2}, {Width: 0}, {Width: 1}, {Content: "A", Width: 1}, {Width: 1}, {Content: "B", Width: 1},
	}}
	if got := terminalRowText(s, 0); got != "界 A B" {
		t.Fatalf("rendered row = %q, want erased spaces without wide continuation space", got)
	}
}
