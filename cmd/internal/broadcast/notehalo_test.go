package broadcast

import (
	"image/color"
	"math/rand"
	"testing"
	"time"
)

func TestPaletteTint(t *testing.T) {
	black := color.RGBA{0, 0, 0, 0xff}
	if got := paletteTint(1, 0, black, true); got != markPalette[1].rgb {
		t.Fatalf("full truecolor %v", got)
	}
	if got := paletteTint(1, 0, black, false); got != markPalette[1].index {
		t.Fatalf("full 256 %v", got)
	}
	if got := paletteTint(0, 0.5, color.RGBA{0xff, 0xff, 0xff, 0xff}, true); got != (color.RGBA{0xff, 0xd7, 0x80, 0xff}) {
		t.Fatalf("midpoint of orange and white %v", got)
	}
	if got := paletteTint(2, 0.5, black, false); got != markPalette[2].ladder[2] {
		t.Fatalf("half way down the ladder %v", got)
	}
	if got := paletteTint(2, 1, black, false); got != markPalette[2].ladder[4] {
		t.Fatalf("bottom of the ladder %v", got)
	}
}

func TestNoteGlowGeometry(t *testing.T) {
	// Half way through a flash both rings show, so the full shape is visible.
	const half = 0.5
	g := noteGlow(Note{Col: 10, Row: 5}, 80, 24, 1, half)
	if _, in := g[cell{10, 5}]; in {
		t.Fatal("the chip is not part of the glow")
	}
	want := map[cell]float64{
		{9, 5}: noteGlowNear, {12, 5}: noteGlowNear, {10, 4}: noteGlowNear, {11, 6}: noteGlowNear,
		{8, 5}: noteGlowNear, {13, 5}: noteGlowNear, // dx 2 → (2/2)² = 1
		{5, 5}: noteGlowFar, {16, 5}: noteGlowFar, // dx 5: the row's ends
		{10, 3}: noteGlowFar, {11, 7}: noteGlowFar, // two rows away
		{7, 3}: noteGlowFar, {14, 7}: noteGlowFar, // dx 3, dy 2: (1.5)²+4 = 6.25, on the edge
	}
	for c, tv := range want {
		if got, in := g[c]; !in || got != tv*(1-half) {
			t.Errorf("%v: %v %v, want %v", c, got, in, tv)
		}
	}
	for _, c := range []cell{{4, 5}, {17, 5}, {6, 3}, {15, 7}, {10, 2}, {10, 8}} {
		if _, in := g[c]; in {
			t.Errorf("%v is outside the round glow", c)
		}
	}
	// At rest only the inner ring shows, in one light shade.
	rest := noteGlow(Note{Col: 10, Row: 5}, 80, 24, 1, 0)
	for c, tv := range rest {
		if _, inShape := g[c]; !inShape || tv != noteGlowNear {
			t.Errorf("%v at %v at rest; want %v", c, tv, noteGlowNear)
		}
	}
	for c, tv := range want {
		if _, in := rest[c]; in != (tv == noteGlowNear) {
			t.Errorf("%v at rest: shown %v; only the inner ring shows", c, in)
		}
	}
	// Flashing: every glow cell at the full color.
	for c, tv := range noteGlow(Note{Col: 10, Row: 5}, 80, 24, 1, 1) {
		if tv != 0 {
			t.Fatalf("%v at %v during a flash", c, tv)
		}
	}
}

func TestNoteGlowStaysOffTheControlsAndTheGrid(t *testing.T) {
	g := noteGlow(Note{Col: StatusGuardCols - 1, Row: 21}, 30, 24, 1, 1) // a flash: the whole glow
	for c := range g {
		if guarded(c, 24, 1) || c.col < 0 || c.row >= 24 {
			t.Fatalf("glow on %v", c)
		}
	}
	if _, in := g[cell{StatusGuardCols, 23}]; !in {
		t.Fatal("the tab bar past the controls can glow")
	}
}

// beat0 is the start of a shared beat (Unix seconds divisible by 15).
var beat0 = time.Unix(1500, 0)

const ms = time.Millisecond

func flashName(f float64) string {
	switch f {
	case 1:
		return "full"
	case 0:
		return "rest"
	}
	return "fading"
}

func TestNoteFlashTiming(t *testing.T) {
	old := beat0.Add(-time.Hour) // long past its arrival: the beat alone
	beat := []struct {
		phase time.Duration
		want  string
	}{
		{0, "full"}, {NoteFlashHold - ms, "full"}, {NoteFlashHold + ms, "fading"},
		{NoteFlashHold + NoteFlashFade - ms, "fading"}, {NoteFlashHold + NoteFlashFade, "rest"}, {14 * time.Second, "rest"},
		{15 * time.Second, "full"},
	}
	for _, c := range beat {
		if got := flashName(noteFlash(old, beat0.Add(c.phase))); got != c.want {
			t.Errorf("beat phase %v: %s, want %s", c.phase, got, c.want)
		}
	}
	// A note saved mid-beat flashes on arrival, on its own.
	saved := beat0.Add(5 * time.Second)
	arrival := []struct {
		age  time.Duration
		want string
	}{
		{0, "full"}, {NoteFlashHold - ms, "full"}, {NoteFlashHold + ms, "fading"}, {NoteFlashHold + NoteFlashFade, "rest"},
		{10 * time.Second, "full"}, // the next shared beat
	}
	for _, c := range arrival {
		if got := flashName(noteFlash(saved, saved.Add(c.age))); got != c.want {
			t.Errorf("arrival age %v: %s, want %s", c.age, got, c.want)
		}
	}
}

func TestNoteNextChange(t *testing.T) {
	old := []Note{{ID: 1, At: beat0.Add(-time.Hour)}}
	for _, c := range []struct{ phase, want time.Duration }{
		{0, NoteFlashHold},
		{NoteFlashHold - 100*ms, 100 * ms},
		{NoteFlashHold + 3*ms, noteFadeStep - 3*ms},
		{3 * time.Second, 12 * time.Second},
	} {
		if d, ok := NoteNextChange(old, beat0.Add(c.phase), 0); !ok || d != c.want {
			t.Errorf("beat phase %v: %v %v, want %v", c.phase, d, ok, c.want)
		}
	}
	// An arrival mid-beat: its own flash ends before the next beat.
	saved := beat0.Add(5 * time.Second)
	if d, _ := NoteNextChange([]Note{{ID: 2, At: saved}}, saved, 0); d != NoteFlashHold {
		t.Fatalf("arrival: %v, want %v", d, NoteFlashHold)
	}
	if d, _ := NoteNextChange([]Note{{ID: 2, At: saved}}, saved.Add(3*time.Second), 0); d != 7*time.Second {
		t.Fatalf("after the arrival flash: %v, want the beat in 7s", d)
	}
	if _, ok := NoteNextChange(nil, beat0, 0); ok {
		t.Fatal("no notes, no change")
	}
	if _, ok := NoteNextChange(old, beat0, 1); ok {
		t.Fatal("a hovered note is steady")
	}
}

// One beat for every note: the number of repaints the flashes cost doesn't
// grow with the number of notes (the envelope at the 48-note limit).
func TestNoteFlashRepaintsDontGrowWithNotes(t *testing.T) {
	count := func(notes []Note) int {
		n := 0
		for now := beat0; now.Before(beat0.Add(NoteFlashEvery)); n++ {
			d, ok := NoteNextChange(notes, now, 0)
			if !ok {
				break
			}
			now = now.Add(d)
		}
		return n
	}
	var many []Note
	for i := 0; i < MaxNotesPerColor*PointerColors; i++ {
		many = append(many, Note{ID: i + 1, At: beat0.Add(-time.Hour - time.Duration(i)*time.Second)})
	}
	one, all := count(many[:1]), count(many)
	if one != all || all > noteFadeSteps+3 {
		t.Fatalf("repaints per beat: one note %d, 48 notes %d", one, all)
	}
}

func TestDrawNotesChipAndGlow(t *testing.T) {
	f := textFrame(t, 30, 10, "\n\n\n\n\nsome text on row five here", "")
	bg := color.RGBA{0x10, 0x10, 0x10, 0xff}
	t0 := beat0 // saved on a beat: flashing at t0, resting 5s later
	n := Note{ID: 1, Col: 10, Row: 5, Color: 1, At: t0}
	resting := DrawNotes(f, []Note{n}, NoteStyle{BG: bg, Blend: true, Now: t0.Add(5 * time.Second)})
	chip := markPalette[1].rgb
	if resting.Cells[5*30+10].Style.Bg != chip || resting.Cells[5*30+11].Style.Bg != chip {
		t.Fatal("the chip is the author's full color")
	}
	near := resting.Cells[5*30+9]
	if near.Style.Bg != paletteTint(1, noteGlowNear, bg, true) || near.Content != f.Cells[5*30+9].Content {
		t.Fatalf("near glow cell %+v", near)
	}
	flashing := DrawNotes(f, []Note{n}, NoteStyle{BG: bg, Blend: true, Now: t0})
	if flashing.Cells[5*30+9].Style.Bg != chip {
		t.Fatal("the glow is the full color during a flash")
	}
	hovered := DrawNotes(f, []Note{n}, NoteStyle{BG: bg, Blend: true, Now: t0.Add(5 * time.Second), Hover: 1})
	if hovered.Cells[5*30+9].Style.Bg != chip {
		t.Fatal("a hovered note glows at full color")
	}
	// Two notes side by side: neither glow paints the other's chip.
	m := Note{ID: 2, Col: 13, Row: 5, Color: 4, At: t0}
	both := DrawNotes(f, []Note{n, m}, NoteStyle{BG: bg, Blend: true, Now: t0.Add(5 * time.Second)})
	if both.Cells[5*30+13].Style.Bg != markPalette[4].rgb || both.Cells[5*30+11].Style.Bg != chip {
		t.Fatal("a glow painted another note's chip")
	}
}

// Over random notes, glow colors and wide text, the drawing (backgrounds
// included) is the same in any note order.
func TestDrawNotesOrderIndependentWithGlow(t *testing.T) {
	rng := rand.New(rand.NewSource(4473))
	t0 := time.Unix(1000, 0)
	for iter := 0; iter < 200; iter++ {
		f := textFrame(t, 24, 8, "字字ab字\n\n字a字a字a字a\n\n\nxx字字", "")
		var board NoteBoard
		for k := 0; k < 5; k++ {
			board.Add(Note{Box: CellBox{rng.Intn(23), rng.Intn(7), 2 + rng.Intn(8), 1 + rng.Intn(4)}, Color: rng.Intn(PointerColors), Text: "x", At: t0.Add(-time.Duration(rng.Intn(20000)) * time.Millisecond)}, 24, 8, 1)
		}
		notes := board.List()
		style := NoteStyle{BG: color.RGBA{0, 0, 0, 0xff}, Blend: true, Now: t0}
		a := DrawNotes(f, notes, style)
		rev := append([]Note(nil), notes...)
		for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
			rev[i], rev[j] = rev[j], rev[i]
		}
		b := DrawNotes(f, rev, style)
		for i := range a.Cells {
			if a.Cells[i].Content != b.Cells[i].Content || a.Cells[i].Style.Bg != b.Cells[i].Style.Bg {
				t.Fatalf("iter %d: cell %d differs by note order", iter, i)
			}
		}
	}
}
