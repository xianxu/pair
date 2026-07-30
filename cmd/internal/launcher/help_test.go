package launcher

import (
	"strings"
	"testing"
)

// The circular sentence is the #132 bug: pressing the help key produced a CLI
// synopsis whose last line said "In-session keybindings are on Alt+h." Usage may
// POINT at the keybind surface, but never back at the key just pressed.
func TestUsageTextIsNotCircular(t *testing.T) {
	got := UsageText()
	if strings.Contains(got, "keybindings are on Alt+h") {
		t.Error("usage still refers keybindings back to Alt+h")
	}
	if !strings.Contains(got, "pair keys") {
		t.Error("usage should point at `pair keys` for keybindings")
	}
}
