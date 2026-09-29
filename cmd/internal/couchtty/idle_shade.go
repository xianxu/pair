package couchtty

import (
	"fmt"
	"image/color"
	"math"
	"time"
)

// IdleLevel is how long a live thread has gone without activity (pair#247):
// the operator's input or its agent's work. The tab bar and the switcher both
// classify with IdleLevelFor and style with FadeStyle, so the two surfaces
// cannot disagree about how idle a thread looks.
type IdleLevel uint8

const (
	IdleFresh IdleLevel = iota // under 1 day, unknown, or in the future
	IdleDay                    // at least 1 day
	IdleStale                  // at least 3 days
)

// idleThresholds are the operator's bands (#247, 2026-09-28): three levels,
// with boundaries at one day and three days.
var idleThresholds = [...]time.Duration{24 * time.Hour, 72 * time.Hour}

// IdleLevelFor classifies last against now. An unknown time reads as fresh --
// a live thread that has left no evidence has not been shown to be idle -- and
// so does a time in the future, which only clock skew produces.
func IdleLevelFor(now, last time.Time, known bool) IdleLevel {
	if !known || last.IsZero() || last.After(now) {
		return IdleFresh
	}
	age, level := now.Sub(last), IdleFresh
	for i, threshold := range idleThresholds {
		if age >= threshold {
			level = IdleLevel(i + 1)
		}
	}
	return level
}

// Palette is what Couch knows about the host terminal's colours. FG and BG come
// from the terminal's own answers to OSC 10/11; Known is set only once both
// have arrived. TrueColor and NoColor come from the environment.
type Palette struct {
	FG, BG    color.RGBA
	Known     bool
	TrueColor bool
	NoColor   bool
}

// styleBase names the colour a label is drawn in before fading.
type styleBase uint8

const (
	baseDefault styleBase = iota // the terminal's default foreground
	baseAmber                    // attentionSGR's xterm 220
)

// amberRGB is xterm colour 220, the colour attentionSGR selects.
var amberRGB = color.RGBA{0xff, 0xd7, 0x00, 0xff}

// idleBlend is how far each level mixes its colour toward the background. The
// top weight stops short of the background so an idle label recedes without
// vanishing.
var idleBlend = [...]float64{0, 0.40, 0.65}

// FadeStyle is the SGR a label (baseDefault) or an amber glyph (baseAmber) is
// drawn with at level. Level 0 is byte-for-byte what the bar drew before #247,
// and so is every level under NO_COLOR: the fade is suppressed, nothing else
// changes.
//
// Fading blends toward the terminal's REAL background rather than using fixed
// greys, which invert on a light scheme (#217). Without an answer from the
// terminal, the label falls back to ANSI 90, the one grey the terminal themes
// itself; the two faded levels are then indistinguishable, and the amber is
// left alone rather than guessed at.
func FadeStyle(p Palette, level IdleLevel, base styleBase) string {
	if level == IdleFresh || p.NoColor {
		if base == baseAmber {
			return attentionSGR
		}
		return ""
	}
	if !p.Known {
		if base == baseAmber {
			return attentionSGR
		}
		return "\x1b[90m"
	}
	from := p.FG
	if base == baseAmber {
		from = amberRGB
	}
	c := blend(from, p.BG, idleBlend[level])
	if p.TrueColor {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
	}
	return fmt.Sprintf("\x1b[38;5;%dm", quantize256(c))
}

// blend mixes a toward b by t (0 keeps a, 1 is b), per channel, rounded.
func blend(a, b color.RGBA, t float64) color.RGBA {
	mix := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x)*(1-t) + float64(y)*t))
	}
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 0xff}
}

// cubeLevels are the channel values of xterm's 6x6x6 colour cube (16-231).
var cubeLevels = [...]int{0, 95, 135, 175, 215, 255}

// quantize256 is the xterm-256 colour nearest c, searching the colour cube and
// the grey ramp (232-255). The 16 system colours are skipped: the terminal's
// theme redefines them, so their RGB is not known.
func quantize256(c color.RGBA) int {
	distance := func(r, g, b int) int {
		dr, dg, db := r-int(c.R), g-int(c.G), b-int(c.B)
		return dr*dr + dg*dg + db*db
	}
	nearest := func(v uint8) int {
		best := 0
		for i, level := range cubeLevels {
			if abs(level-int(v)) < abs(cubeLevels[best]-int(v)) {
				best = i
			}
		}
		return best
	}
	ri, gi, bi := nearest(c.R), nearest(c.G), nearest(c.B)
	bestIndex := 16 + 36*ri + 6*gi + bi
	bestDistance := distance(cubeLevels[ri], cubeLevels[gi], cubeLevels[bi])
	for i := 0; i < 24; i++ {
		grey := 8 + 10*i
		if d := distance(grey, grey, grey); d < bestDistance {
			bestIndex, bestDistance = 232+i, d
		}
	}
	return bestIndex
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
