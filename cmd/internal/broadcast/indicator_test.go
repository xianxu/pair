package broadcast

import (
	"testing"

	vt "github.com/charmbracelet/x/vt"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestIndicatorShown(t *testing.T) {
	cases := []struct {
		name  string
		frame func(*testing.T) bool
		want  bool
	}{
		{"live label leads the last row", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, 20, 3, "body", liveChrome("REC 21%")))
		}, true},
		{"label text without the live style", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, 20, 3, "body", LiveLabel+" REC"))
		}, false},
		{"label in another style", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, 20, 3, "body", "\x1b[44m"+LiveLabel+"\x1b[0m"))
		}, false},
		{"label on another row", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, 20, 3, LiveSGR+LiveLabel+"\x1b[0m", "tabs"))
		}, false},
		{"label not at column zero", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, 20, 3, "body", " "+liveChrome("")))
		}, false},
		{"label clipped by a narrow terminal", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, textwidth.Width(LiveLabel)-1, 3, "body", liveChrome("")))
		}, false},
		{"blank resize row", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, 20, 3, "body", ""))
		}, false},
		{"starting label is not live", func(t *testing.T) bool {
			return IndicatorShown(textFrame(t, 20, 3, "body", LiveSGR+StartingLabel+"\x1b[0m"))
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.frame(t); got != c.want {
				t.Fatalf("IndicatorShown = %v, want %v", got, c.want)
			}
		})
	}
}

// The status row's click span and IndicatorShown's cell walk both assume one
// column per glyph. If a width function says otherwise, they drift apart.
func TestIndicatorGlyphWidths(t *testing.T) {
	for _, label := range []string{LiveLabel, StartingLabel} {
		runes := len([]rune(label))
		if w := textwidth.Width(label); w != runes {
			t.Errorf("textwidth.Width(%q) = %d, want %d", label, w, runes)
		}
		if w := ansi.StringWidth(label); w != runes {
			t.Errorf("ansi.StringWidth(%q) = %d, want %d", label, w, runes)
		}
	}
}

// #412: the pointer and control markers are emoji, two columns everywhere
// they're measured or drawn; otherwise the status row's click spans and the
// pointer check drift from what's on screen.
func TestCapabilityMarkerWidths(t *testing.T) {
	for _, label := range []string{PointerLabel, ControlLabel} {
		if w := textwidth.Width(label); w != 2 {
			t.Errorf("textwidth.Width(%q) = %d, want 2", label, w)
		}
		if w := ansi.StringWidth(label); w != 2 {
			t.Errorf("ansi.StringWidth(%q) = %d, want 2", label, w)
		}
		e := vt.NewEmulator(10, 2)
		e.WriteString("\x1b[1;3H" + label)
		if got := e.CursorPosition().X - 2; got != 2 {
			t.Errorf("vt emulator advanced %d for %q, want 2", got, label)
		}
	}
}
