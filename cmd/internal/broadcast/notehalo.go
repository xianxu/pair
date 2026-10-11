package broadcast

import (
	"image/color"
	"time"
)

// A note's 💭 sits on a chip of its author's color inside a round glow
// (#447 M3): on a full screen a bare emoji is easy to miss. The glow flashes
// at full color when the note is saved and again every NoteFlashEvery until
// it is resolved; each note's phase is its own age, so notes don't pulse
// together.
const (
	NoteFlashEvery = 15 * time.Second
	NoteFlashHold  = 1500 * time.Millisecond
	NoteFlashFade  = 500 * time.Millisecond
	noteFadeSteps  = 10
	noteFadeStep   = NoteFlashFade / noteFadeSteps
	// The glow's reach in rows; columns reach twice as far, because a cell
	// is about twice as tall as it is wide.
	noteGlowRows = 2.5
)

// Resting glow, as how far each ring is weakened toward the background
// (paletteTint's t): the ring next to the chip, and the outer ring.
const (
	noteGlowNear = 0.5
	noteGlowFar  = 0.75
)

// NoteStyle is what drawing notes needs besides the notes: the operator's
// background and whether it can be blended into (truecolor), the clock, and
// the note the operator is hovering in the notes menu (0 for none), which is
// held at full glow.
type NoteStyle struct {
	BG    color.RGBA
	Blend bool
	Now   time.Time
	Hover int
}

// noteFlash is how much of the flash shows at age: 1 while it holds, then
// stepping down to 0, the resting glow, until the next flash.
func noteFlash(age time.Duration) float64 {
	p := max(age, 0) % NoteFlashEvery
	switch {
	case p < NoteFlashHold:
		return 1
	case p < NoteFlashHold+NoteFlashFade:
		k := int((p - NoteFlashHold) / noteFadeStep) // 0..noteFadeSteps-1
		return 1 - float64(k+1)/float64(noteFadeSteps+1)
	}
	return 0
}

// noteGlow is the glow around n on a cols×rows grid: each cell's t (0 is the
// full color), given the flash level. The chip's own two cells are not in
// it, nor are cells off the grid or on the broadcast's controls.
func noteGlow(n Note, cols, rows int, flash float64) map[cell]float64 {
	out := map[cell]float64{}
	const reach = 2 // whole rows inside noteGlowRows
	for dy := -reach; dy <= reach; dy++ {
		for col := n.Col - 2*reach - 1; col <= n.Col+1+2*reach+1; col++ {
			c := cell{col, n.Row + dy}
			dx := 0
			switch {
			case col < n.Col:
				dx = n.Col - col
			case col > n.Col+1:
				dx = col - n.Col - 1
			}
			d2 := float64(dx*dx)/4 + float64(dy*dy)
			if (dx == 0 && dy == 0) || d2 > noteGlowRows*noteGlowRows ||
				c.col < 0 || c.col >= cols || c.row < 0 || c.row >= rows || guarded(c, rows) {
				continue
			}
			rest := noteGlowFar
			if d2 <= 1.5*1.5 {
				rest = noteGlowNear
			}
			out[c] = rest * (1 - flash)
		}
	}
	return out
}

// NoteNextChange is how long until the notes' drawing next changes (a flash
// starting, stepping or ending); ok is false when nothing changes. A hovered
// note is steady.
func NoteNextChange(notes []Note, now time.Time, hover int) (time.Duration, bool) {
	var next time.Duration
	ok := false
	for _, n := range notes {
		if n.ID == hover {
			continue
		}
		p := max(now.Sub(n.At), 0) % NoteFlashEvery
		var d time.Duration
		switch {
		case p < NoteFlashHold:
			d = NoteFlashHold - p
		case p < NoteFlashHold+NoteFlashFade:
			d = noteFadeStep - (p-NoteFlashHold)%noteFadeStep
		default:
			d = NoteFlashEvery - p
		}
		if !ok || d < next {
			next, ok = d, true
		}
	}
	return next, ok
}
