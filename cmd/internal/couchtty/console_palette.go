package couchtty

import (
	"image/color"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// paletteQuery asks the host terminal for its default foreground (OSC 10) and
// background (OSC 11). Idle fading blends toward the real background (pair#247),
// because fixed greys invert on a light scheme. Run sends it once, after
// MakeRaw and before the presenter's first write, so it cannot interleave with
// a frame; the answers come back through the one stdin decoder.
const paletteQuery = "\x1b]10;?\x1b\\\x1b]11;?\x1b\\"

// ensureMenuLocked builds the menu state on first use. Anything recorded before
// that -- the colour modes, a palette reply that raced the first attach -- is
// kept: the fresh state must not forget what the terminal already told us.
// Callers hold c.mu.
func (c *Console) ensureMenuLocked(address couchcore.ThreadAddress) {
	if c.menuReady {
		return
	}
	palette := c.menu.Palette
	c.menu = NewMenuState(nil, address)
	c.menu.Notice = infoMenuNotice("thread inventory unavailable")
	c.menu.Palette = palette
	c.menuReady = true
}

// SetColorModes records what the environment says about colour: COLORTERM's
// truecolor/24bit, and NO_COLOR. The caller reads the environment, so the
// console stays testable without it. Call before Run.
func (c *Console) SetColorModes(trueColor, noColor bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.menu.Palette.TrueColor, c.menu.Palette.NoColor = trueColor, noColor
}

// capturePalette records a colour reply. It reports whether the event was one,
// so the caller repaints. Known is set only once both answers are in; a nil
// colour (a malformed reply) is ignored, which leaves the palette unknown and
// the fade on its ANSI 90 fallback.
func (c *Console) capturePalette(event uv.Event) bool {
	var rgba color.RGBA
	foreground := false
	switch e := event.(type) {
	case uv.ForegroundColorEvent:
		if e.Color == nil {
			return false
		}
		rgba, foreground = toRGBA(e.Color), true
	case uv.BackgroundColorEvent:
		if e.Color == nil {
			return false
		}
		rgba = toRGBA(e.Color)
	default:
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	palette := c.menu.Palette
	if foreground {
		palette.FG, c.paletteFG = rgba, true
	} else {
		palette.BG, c.paletteBG = rgba, true
	}
	palette.Known = c.paletteFG && c.paletteBG
	// No ensureMenuLocked here: building the menu is the first attach's job,
	// with its own address. A reply that beats it lands on the unbuilt state,
	// and ensureMenuLocked carries it over.
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventPalette, Palette: palette})
	return true
}

func toRGBA(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xff}
}

// repaintAfterPalette redraws whichever surface is showing, the way a landed
// slot git pass does.
func (c *Console) repaintAfterPalette() {
	c.mu.Lock()
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if panelFocused {
		c.showMenu()
	} else {
		c.repaint()
	}
}
