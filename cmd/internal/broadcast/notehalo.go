package broadcast

import (
	"image/color"
	"time"
)

// A note's 💭 sits on a chip of its author's color inside a round glow
// (#447 M3): on a full screen a bare emoji is easy to miss. The glow flashes
// at full color when the note is saved, and every open note flashes together
// on one shared beat every NoteFlashEvery until it is resolved. One beat for
// all notes bounds the repaints the flashes cost: about 26 per beat plus 26
// per arrival (saves are rate limited), whatever the number of notes, where
// per-note phases would cost 26 per note per beat. The flash is brief and
// finely stepped (20ms steps), and the resting glow faint: the operator found
// a 1.5s hold, 50ms steps and a stronger rest too obtrusive and choppy.
const (
	NoteFlashEvery = 15 * time.Second
	NoteFlashHold  = 300 * time.Millisecond
	NoteFlashFade  = 500 * time.Millisecond
	noteFadeSteps  = 25
	noteFadeStep   = NoteFlashFade / noteFadeSteps
	// The glow's reach in rows; columns reach twice as far, because a cell
	// is about twice as tall as it is wide.
	noteGlowRows = 2.5
)

// Resting glow, as how far each ring is weakened toward the background
// (paletteTint's t): the ring next to the chip keeps one very light shade,
// and the outer ring fades out entirely, so a quiet note is a small, faint
// halo and only a flash spreads the full glow (the operator's smoke).
const (
	noteGlowNear = 0.9
	noteGlowFar  = 1.0
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

// NoteChipColor is a note's author color at full strength, as its chip is
// drawn; the terminal's notes menu (M4) draws each item's chip with it.
func NoteChipColor(c int, style NoteStyle) color.Color {
	return paletteTint(c, 0, style.BG, style.Blend)
}

// flashShape is how much of a flash shows p into it: 1 while it holds, then
// stepping down, 0 once it is over; next is how long until that changes
// (ok false once over).
func flashShape(p time.Duration) (level float64, next time.Duration, ok bool) {
	switch {
	case p < NoteFlashHold:
		return 1, NoteFlashHold - p, true
	case p < NoteFlashHold+NoteFlashFade:
		k := int((p - NoteFlashHold) / noteFadeStep) // 0..noteFadeSteps-1
		return 1 - float64(k+1)/float64(noteFadeSteps+1), noteFadeStep - (p-NoteFlashHold)%noteFadeStep, true
	}
	return 0, 0, false
}

// beatPhase is how far now is into the shared beat.
func beatPhase(now time.Time) time.Duration {
	return time.Duration(now.UnixNano() % int64(NoteFlashEvery))
}

// noteFlash is how much flash a note saved at at shows at now: its arrival
// flash or the shared beat, whichever is stronger.
func noteFlash(at, now time.Time) float64 {
	arrival := 0.0
	if age := now.Sub(at); age >= 0 {
		arrival, _, _ = flashShape(age)
	}
	beat, _, _ := flashShape(beatPhase(now))
	return max(arrival, beat)
}

// noteGlow is the glow around n on a cols×rows grid: each cell's t (0 is the
// full color), given the flash level. A cell faded all the way out is not
// in it (the outer ring at rest), nor are the chip's own two cells, nor are cells off the grid or on the broadcast's controls.
func noteGlow(n Note, cols, rows, chromeRows int, flash float64) map[cell]float64 {
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
				c.col < 0 || c.col >= cols || c.row < 0 || c.row >= rows || guarded(c, rows, chromeRows) {
				continue
			}
			rest := noteGlowFar
			if d2 <= 1.5*1.5 {
				rest = noteGlowNear
			}
			if t := rest * (1 - flash); t < 1 {
				out[c] = t
			}
		}
	}
	return out
}

// NoteNextChange is how long until the notes' drawing next changes: an
// arrival flash stepping or ending, or the shared beat starting, stepping or
// ending. ok is false when nothing would change: no notes, or only the
// hovered one, which is steady.
func NoteNextChange(notes []Note, now time.Time, hover int) (time.Duration, bool) {
	var next time.Duration
	ok := false
	take := func(d time.Duration) {
		if !ok || d < next {
			next, ok = d, true
		}
	}
	for _, n := range notes {
		if n.ID == hover {
			continue
		}
		if age := now.Sub(n.At); age >= 0 {
			if _, d, live := flashShape(age); live {
				take(d)
			}
		}
		p := beatPhase(now)
		if _, d, live := flashShape(p); live {
			take(d)
		} else {
			take(NoteFlashEvery - p)
		}
	}
	return next, ok
}
