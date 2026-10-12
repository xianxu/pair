package webcast

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/fidelity"
)

// Only the replays a report can show are kept, however long the cast.
func TestKeepWorstBoundsReplays(t *testing.T) {
	replays := map[string][]byte{}
	var shots []fidelity.Shot
	for i := 0; i < 50; i++ {
		f := fmt.Sprintf("frames/%d.png", i)
		shots = append(shots, fidelity.Shot{File: f, Diff: float64((i * 37) % 50)})
		replays[f] = []byte{1}
		keepWorst(replays, shots)
		if len(replays) > fidelity.MaxWorst {
			t.Fatalf("%d replays kept", len(replays))
		}
	}
	rep := fidelity.Summarize("r", shots, 0)
	for _, i := range rep.Worst {
		if _, ok := replays[rep.Shots[i].File]; !ok {
			t.Errorf("worst %+v has no replay", rep.Shots[i])
		}
	}
}

func encoded(t *testing.T, w, h int, black image.Rectangle) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{255, 255, 255, 255}
			if image.Pt(x, y).In(black) {
				c = color.NRGBA{0, 0, 0, 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func TestSideBySideCrops(t *testing.T) {
	tab := encoded(t, 200, 100, image.Rectangle{})
	cases := []struct {
		replay []byte
		w, h   int
	}{
		{encoded(t, 200, 100, image.Rect(50, 50, 60, 60)), 3*(10+2*fidelity.CropPad) + 2*fidelity.Gap, 10 + 2*fidelity.CropPad},
		{encoded(t, 200, 100, image.Rectangle{}), 3*200 + 2*fidelity.Gap, 100}, // nothing differs: whole
		{encoded(t, 120, 100, image.Rectangle{}), 3*120 + 2*fidelity.Gap, 100}, // another size: the common part
	}
	for i, c := range cases {
		out, err := sideBySide(tab, c.replay)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != c.w || b.Dy() != c.h {
			t.Errorf("case %d: %v, want %dx%d", i, b, c.w, c.h)
		}
	}
}
