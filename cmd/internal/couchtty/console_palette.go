package couchtty

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// paletteQuery asks the host terminal for its default foreground (OSC 10) and
// background (OSC 11), and its 16 ANSI colours (OSC 4). Idle fading blends
// toward the real background (pair#247), because fixed greys invert on a light
// scheme; a broadcast hands all of them to viewers so they see the operator's
// colours (#395). Run sends it once, after MakeRaw and before the presenter's
// first write, so it cannot interleave with a frame; the answers come back
// through the one stdin decoder, as replies, never as child input.
var paletteQuery = func() string {
	q := "\x1b]10;?\x1b\\\x1b]11;?\x1b\\"
	for i := range 16 {
		q += fmt.Sprintf("\x1b]4;%d;?\x1b\\", i)
	}
	return q
}()

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
	case uv.UnknownOscEvent:
		// OSC 4 replies have no event of their own in ultraviolet.
		if index, rgba, ok := parseOSC4Reply(string(e)); ok {
			c.mu.Lock()
			c.ansiPalette[index], c.ansiKnown[index] = rgba, true
			c.mu.Unlock()
		}
		// Nothing on screen depends on these yet; no repaint.
		return false
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

// parseOSC4Reply reads a terminal's answer to `OSC 4;N;?`, which is
// `OSC 4;N;rgb:R/G/B` ended by BEL or ST, each component 1–4 hex digits. Only
// indices 0–15 are kept: they are the colours SGR 30–37/90–97 name.
func parseOSC4Reply(raw string) (int, color.RGBA, bool) {
	body, ok := strings.CutPrefix(raw, "\x1b]4;")
	if !ok {
		return 0, color.RGBA{}, false
	}
	body = strings.TrimSuffix(strings.TrimSuffix(body, "\x07"), "\x1b\\")
	indexText, spec, ok := strings.Cut(body, ";")
	if !ok {
		return 0, color.RGBA{}, false
	}
	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 || index > 15 {
		return 0, color.RGBA{}, false
	}
	hex, ok := strings.CutPrefix(spec, "rgb:")
	if !ok {
		return 0, color.RGBA{}, false
	}
	parts := strings.Split(hex, "/")
	if len(parts) != 3 {
		return 0, color.RGBA{}, false
	}
	var rgb [3]uint8
	for i, p := range parts {
		if len(p) < 1 || len(p) > 4 {
			return 0, color.RGBA{}, false
		}
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil {
			return 0, color.RGBA{}, false
		}
		// Scale n hex digits to 8 bits: v / (16^n - 1) * 255, rounded.
		maxV := uint64(1)<<(4*len(p)) - 1
		rgb[i] = uint8((v*255 + maxV/2) / maxV)
	}
	return index, color.RGBA{rgb[0], rgb[1], rgb[2], 0xff}, true
}

// broadcastTheme is the operator's palette as far as the terminal has told
// Couch, for a broadcast's viewers. Colours it didn't report stay empty.
func (c *Console) broadcastTheme() broadcast.Theme {
	c.mu.Lock()
	defer c.mu.Unlock()
	var t broadcast.Theme
	if c.paletteFG {
		t.Foreground = broadcast.Hex(c.menu.Palette.FG)
	}
	if c.paletteBG {
		t.Background = broadcast.Hex(c.menu.Palette.BG)
	}
	for i, known := range c.ansiKnown {
		if known {
			t.ANSI[i] = broadcast.Hex(c.ansiPalette[i])
		}
	}
	return t
}

func toRGBA(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xff}
}

// repaintVisible redraws whichever surface is showing: the switcher when the
// panel has focus, otherwise the actor and its status row. Background results
// that change what either view draws -- a slot git pass, an activity pass, a
// palette reply -- all end here. Focus is read at repaint time, so it is the
// surface on screen now that gets redrawn.
func (c *Console) repaintVisible() {
	c.mu.Lock()
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if panelFocused {
		c.showMenu()
	} else {
		c.repaint()
	}
}
