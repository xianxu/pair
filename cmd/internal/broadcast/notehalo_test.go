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
	g := noteGlow(Note{Col: 10, Row: 5}, 80, 24, 0)
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
		if got, in := g[c]; !in || got != tv {
			t.Errorf("%v: %v %v, want %v", c, got, in, tv)
		}
	}
	for _, c := range []cell{{4, 5}, {17, 5}, {6, 3}, {15, 7}, {10, 2}, {10, 8}} {
		if _, in := g[c]; in {
			t.Errorf("%v is outside the round glow", c)
		}
	}
	// Flashing: every glow cell at the full color.
	for c, tv := range noteGlow(Note{Col: 10, Row: 5}, 80, 24, 1) {
		if tv != 0 {
			t.Fatalf("%v at %v during a flash", c, tv)
		}
	}
}

func TestNoteGlowStaysOffTheControlsAndTheGrid(t *testing.T) {
	g := noteGlow(Note{Col: StatusGuardCols - 1, Row: 21}, 30, 24, 0)
	for c := range g {
		if guarded(c, 24) || c.col < 0 || c.row >= 24 {
			t.Fatalf("glow on %v", c)
		}
	}
	if _, in := g[cell{StatusGuardCols, 23}]; !in {
		t.Fatal("the tab bar past the controls can glow")
	}
}

func TestNoteFlashTiming(t *testing.T) {
	cases := []struct {
		age  time.Duration
		want string
	}{
		{0, "full"}, {1400 * time.Millisecond, "full"}, {1600 * time.Millisecond, "fading"},
		{1950 * time.Millisecond, "fading"}, {2100 * time.Millisecond, "rest"}, {14 * time.Second, "rest"},
		{15 * time.Second, "full"}, {16600 * time.Millisecond, "fading"}, {18 * time.Second, "rest"},
	}
	for _, c := range cases {
		f := noteFlash(c.age)
		got := "fading"
		switch f {
		case 1:
			got = "full"
		case 0:
			got = "rest"
		}
		if got != c.want {
			t.Errorf("age %v: %v (%v), want %s", c.age, got, f, c.want)
		}
	}
}

func TestNoteNextChange(t *testing.T) {
	t0 := time.Unix(1000, 0)
	at := func(time.Duration) []Note { return []Note{{ID: 1, At: t0}} }
	cases := []struct{ age, want time.Duration }{
		{0, NoteFlashHold},
		{time.Second, 500 * time.Millisecond},
		{1600 * time.Millisecond, 50 * time.Millisecond},
		{3 * time.Second, 12 * time.Second},
		{15 * time.Second, NoteFlashHold},
	}
	for _, c := range cases {
		d, ok := NoteNextChange(at(c.age), t0.Add(c.age), 0)
		if !ok || d != c.want {
			t.Errorf("age %v: %v %v, want %v", c.age, d, ok, c.want)
		}
	}
	if _, ok := NoteNextChange(nil, t0, 0); ok {
		t.Fatal("no notes, no change")
	}
	if _, ok := NoteNextChange([]Note{{ID: 1, At: t0}}, t0, 1); ok {
		t.Fatal("a hovered note is steady")
	}
	// The soonest of several notes: at t0+3s note 1 is 3s into its cycle
	// (12s to its next flash) and note 2 is 17s old, 2s into its cycle (13s).
	d, _ := NoteNextChange([]Note{{ID: 1, At: t0}, {ID: 2, At: t0.Add(-14 * time.Second)}}, t0.Add(3*time.Second), 0)
	if d != 12*time.Second {
		t.Fatalf("soonest %v, want 12s", d)
	}
}

func TestDrawNotesChipAndGlow(t *testing.T) {
	f := textFrame(t, 30, 10, "\n\n\n\n\nsome text on row five here", "")
	bg := color.RGBA{0x10, 0x10, 0x10, 0xff}
	t0 := time.Unix(1000, 0)
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
			board.Add(Note{Col: rng.Intn(24), Row: rng.Intn(7), Color: rng.Intn(PointerColors), Text: "x", At: t0.Add(-time.Duration(rng.Intn(20000)) * time.Millisecond)})
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
