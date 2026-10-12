package fidelity

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fill(w, h int, at func(x, y int) color.Color) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, at(x, y))
		}
	}
	return img
}

func pngOf(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func white(int, int) color.Color { return color.White }

func TestDiff(t *testing.T) {
	// The control's corner is left out of the comparison.
	corner := fill(100, 100, func(x, y int) color.Color {
		if x >= 100-64 && y >= 100-24 {
			return color.Black
		}
		return color.White
	})
	plain := fill(100, 100, white)
	nearly := fill(100, 100, func(int, int) color.Color { return color.RGBA{250, 250, 250, 255} })
	quarter := fill(100, 100, func(x, y int) color.Color {
		if x < 50 && y < 50 {
			return color.Black
		}
		return color.White
	})
	for _, c := range []struct {
		a, b image.Image
		want float64
	}{{plain, corner, 0}, {plain, plain, 0}, {plain, nearly, 0}, {plain, quarter, 2500.0 / (10000 - 64*24)}} {
		if got, err := Diff(pngOf(t, c.a), pngOf(t, c.b)); err != nil || got != c.want {
			t.Errorf("diff %v, want %v (%v)", got, c.want, err)
		}
	}
	if _, err := DiffImages(plain, fill(50, 100, white)); err == nil {
		t.Error("sizes that differ must be an error")
	}
}

func TestSummarize(t *testing.T) {
	var shots []Shot
	for i, d := range []float64{0.5, 9, 1, 4, 0.2, 2, 3, 3.5} {
		shots = append(shots, Shot{File: "frames/" + string(rune('0'+i)) + ".png", T: int64(i * 1000), Diff: d})
	}
	rep := Summarize("rec", shots, 2)
	if rep.Median != 2 && rep.Median != 3 {
		t.Errorf("median %v", rep.Median)
	}
	if rep.Max != 9 || rep.Skipped != 2 || rep.Frames != 10 || rep.Tolerance != Tolerance {
		t.Errorf("report %+v", rep)
	}
	// Over the tolerance: 9, 4, 3.5 (3 is at it, not over).
	if rep.Over != 3 {
		t.Errorf("over %d, want 3", rep.Over)
	}
	for _, s := range rep.Shots {
		if s.Over != (s.Diff > Tolerance) {
			t.Errorf("shot %+v: over is %v", s, s.Over)
		}
	}
	// The worst: every shot over the tolerance, and at least three, worst first.
	var worst []float64
	for _, i := range rep.Worst {
		worst = append(worst, rep.Shots[i].Diff)
	}
	if want := []float64{9, 4, 3.5}; !equal(worst, want) {
		t.Errorf("worst %v, want %v", worst, want)
	}
	// Within tolerance, the three worst are still shown.
	calm := Summarize("rec", []Shot{{Diff: 1}, {Diff: 0.1}, {Diff: 2}, {Diff: 0.5}}, 0)
	if len(calm.Worst) != 3 || calm.Shots[calm.Worst[0]].Diff != 2 || calm.Over != 0 {
		t.Errorf("calm %+v", calm)
	}
	// Many over: capped.
	var many []Shot
	for i := 0; i < 30; i++ {
		many = append(many, Shot{Diff: 10 + float64(i)})
	}
	if got := Summarize("rec", many, 0); len(got.Worst) != MaxWorst || got.Shots[got.Worst[0]].Diff != 39 {
		t.Errorf("many: %d worst, first %v", len(got.Worst), got.Shots[got.Worst[0]].Diff)
	}
	// The caller's shots are left as they were.
	in := []Shot{{Diff: 9}}
	if Summarize("rec", in, 0); in[0].Over || in[0].Crop != "" {
		t.Errorf("Summarize wrote into its input: %+v", in[0])
	}
	if empty := Summarize("rec", nil, 3); len(empty.Shots) != 0 || empty.Skipped != 3 || empty.Within() {
		t.Errorf("empty %+v", empty)
	}
}

func equal(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDiffBoxAndSideBySide(t *testing.T) {
	tab := fill(200, 100, white)
	replay := fill(200, 100, func(x, y int) color.Color {
		if x >= 40 && x < 60 && y >= 10 && y < 20 {
			return color.Black
		}
		return color.White
	})
	box, ok := DiffBox(tab, replay)
	if !ok || box != image.Rect(40-CropPad, 10-CropPad, 60+CropPad, 20+CropPad) {
		t.Fatalf("box %v %v", box, ok)
	}
	// Clipped to the image.
	edge := fill(200, 100, func(x, y int) color.Color {
		if x < 2 && y < 2 {
			return color.Black
		}
		return color.White
	})
	if box, _ := DiffBox(tab, edge); box.Min != image.Pt(0, 0) {
		t.Errorf("edge box %v", box)
	}
	if _, ok := DiffBox(tab, tab); ok {
		t.Error("equal images have no box")
	}
	img := SideBySide(tab, replay, box)
	w, h := box.Dx(), box.Dy()
	if b := img.Bounds(); b.Dx() != 3*w+2*Gap || b.Dy() != h {
		t.Fatalf("side by side %v for a %dx%d box", b, w, h)
	}
	// The mask panel is red where the pixels differ, and the replay panel
	// shows the replay.
	at := image.Pt(40-box.Min.X, 10-box.Min.Y)
	if r, g, b, _ := img.At(2*(w+Gap)+at.X, at.Y).RGBA(); r>>8 != 255 || g>>8 > 64 || b>>8 > 64 {
		t.Errorf("mask pixel %d %d %d", r>>8, g>>8, b>>8)
	}
	if r, _, _, _ := img.At(w+Gap+at.X, at.Y).RGBA(); r != 0 {
		t.Errorf("replay panel at a black pixel: %d", r)
	}
	if r, _, _, _ := img.At(at.X, at.Y).RGBA(); r>>8 != 255 {
		t.Errorf("tab panel at a white pixel: %d", r)
	}
}

func TestReportFiles(t *testing.T) {
	rep := Summarize("20261012T101010-abcd", []Shot{
		{File: "frames/1.png", T: 1200, Diff: 0.4},
		{File: "frames/2.png", T: 5300, Diff: 7.25, Offset: 150, AtT: 12.5},
		{File: "frames/3.png", T: 10400, Diff: 1.1},
	}, 1)
	md := rep.Markdown()
	for _, want := range []string{"20261012T101010-abcd", "median 1.10%", "max 7.25%", "tolerance 3%", "1 of 3",
		"t=5300", "t+150 ms (12.50% at t)", "worst-1.png", "frames/2.png", "1 frame had nothing live to replay"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q:\n%s", want, md)
		}
	}
	if s := rep.Summary(); !strings.Contains(s, "median 1.10%") || !strings.Contains(s, "max 7.25%") || !strings.Contains(s, "1 of 3 frames over 3%") {
		t.Errorf("summary %q", s)
	}
	dir := t.TempDir()
	crops := make([][]byte, len(rep.Worst))
	for i := range crops {
		crops[i] = pngOf(t, fill(4, 4, white))
	}
	if err := Write(dir, rep, crops); err != nil {
		t.Fatal(err)
	}
	var back Report
	raw, err := os.ReadFile(filepath.Join(dir, ReportDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &back); err != nil || back.Max != 7.25 || back.Shots[1].Crop != ReportDir+"/worst-1.png" {
		t.Fatalf("report.json %+v %v", back, err)
	}
	for _, f := range []string{"report.md", "worst-1.png", "worst-2.png", "worst-3.png"} {
		if _, err := os.Stat(filepath.Join(dir, ReportDir, f)); err != nil {
			t.Error(err)
		}
	}
	// A write a crash cut short is swept by the next one.
	if err := os.Mkdir(filepath.Join(dir, ".fidelity-crashed"), 0o700); err != nil {
		t.Fatal(err)
	}
	// A second run replaces the first: no stale crop survives.
	small := Summarize("r", []Shot{{File: "frames/1.png", Diff: 0.1}}, 0)
	if err := Write(dir, small, [][]byte{pngOf(t, fill(4, 4, white))}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ReportDir, "worst-2.png")); !os.IsNotExist(err) {
		t.Errorf("a stale crop survived a rerun: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".fidelity-*")); len(left) != 0 {
		t.Errorf("temporary directories left: %v", left)
	}
	if err := Write(dir, small, nil); err == nil {
		t.Error("crops must match the worst shots")
	}
}
