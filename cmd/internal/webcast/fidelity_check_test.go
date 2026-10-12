package webcast

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/fidelity"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
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

// Frame records come from disk: a file outside frames/ or an implausible
// size is refused before anything is read or laid out.
func TestFrameViewport(t *testing.T) {
	ok := []struct {
		f    recording.Frame
		want image.Point
	}{
		{recording.Frame{File: "frames/0.png", W: 640, H: 400, Scale: 0.5}, image.Pt(1280, 800)},
		{recording.Frame{File: "frames/3.png", W: 640, H: 400}, image.Pt(1280, 800)}, // the fixtures carry no scale
		{recording.Frame{File: "frames/1.png", W: 100, H: 50, Scale: 1.0 / 3}, image.Pt(300, 150)},
	}
	for _, c := range ok {
		if got, err := frameViewport(&c.f); err != nil || got != c.want {
			t.Errorf("%+v: %v %v", c.f, got, err)
		}
	}
	for _, f := range []recording.Frame{
		{File: "../events.jsonl", W: 640, H: 400, Scale: 0.5},
		{File: "/etc/passwd", W: 640, H: 400, Scale: 0.5},
		{File: "frames/../../x.png", W: 640, H: 400, Scale: 0.5},
		{File: "assets/0.png", W: 640, H: 400, Scale: 0.5},
		{File: "frames/0.png", W: 0, H: 400, Scale: 0.5},
		{File: "frames/0.png", W: 640, H: 400, Scale: 2},
		{File: "frames/0.png", W: 1 << 20, H: 400, Scale: 0.5},
	} {
		if _, err := frameViewport(&f); err == nil {
			t.Errorf("%+v accepted", f)
		}
	}
}
