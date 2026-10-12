package fidelity

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestRenderIsStaticAndKeepsState(t *testing.T) {
	b := rec.NewBuilder("r")
	frameDoc := rec.Node{ID: b.ID(), Y: rec.YDoc, C: []rec.Node{b.El("html", nil, b.El("body", nil, b.Txt("in frame")))}}
	host := b.El("div", nil)
	shadow := rec.Node{ID: b.ID(), Y: rec.YShadow, Mode: "closed", C: []rec.Node{b.El("i", nil, b.Txt("closed root"))}}
	host.SR = &shadow
	in := b.El("input", []string{"value", "attr"})
	pw := b.El("input", []string{"type", "password"})
	style := b.El("style", nil, b.Txt(""))
	sel := b.El("select", nil, b.El("option", nil, b.Txt("a")), b.El("option", nil, b.Txt("b")))
	ifr := b.El("iframe", []string{"src", "http://other/"})
	script := b.El("script", nil, b.Txt("alert(1)"))
	btn := b.El("button", []string{"onclick", "go()"}, b.Txt("go"))
	b.Page(rec.SegLive, "http://x/", host, in, pw, style, sel, ifr, script, btn)
	b.Add(&rec.Doc{Doc: frameDoc.ID, Host: ifr.ID, URL: "http://other/", Tree: frameDoc})
	b.Add(&rec.Input{ID: in.ID, V: rec.Str("typed")})
	b.Add(&rec.Input{ID: pw.ID, Secret: true})
	b.Add(&rec.Input{ID: sel.ID, Sel: []int{1}})
	b.Add(&rec.Sheet{ID: b.ID(), Owner: style.ID, Text: "h1{color:red}"})
	st, err := rec.Reduce(b.Recs, 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(st)
	for _, want := range []string{`<template shadowrootmode="closed">`, `closed root`, `value="typed"`, `h1{color:red}</style>`,
		`srcdoc="`, `in frame`, `<option data-fidelity-rid`, `selected`} {
		if !strings.Contains(out, want) {
			t.Errorf("render lacks %s:\n%s", want, out)
		}
	}
	for _, not := range []string{"alert(1)", "onclick", `value="attr"`, `src="http://other/"`} {
		if strings.Contains(out, not) {
			t.Errorf("render has %s:\n%s", not, out)
		}
	}
}

func pngOf(t *testing.T, fill func(x, y int) color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, fill(x, y))
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestDiff(t *testing.T) {
	// The control's corner is left out of the comparison.
	corner := pngOf(t, func(x, y int) color.Color {
		if x >= 100-64 && y >= 100-24 {
			return color.Black
		}
		return color.White
	})
	white := pngOf(t, func(int, int) color.Color { return color.White })
	nearly := pngOf(t, func(int, int) color.Color { return color.RGBA{250, 250, 250, 255} })
	quarter := pngOf(t, func(x, y int) color.Color {
		if x < 50 && y < 50 {
			return color.Black
		}
		return color.White
	})
	for _, c := range []struct {
		a, b []byte
		want float64
	}{{white, corner, 0}, {white, white, 0}, {white, nearly, 0}, {white, quarter, 2500.0 / (10000 - 64*24)}} {
		if got, err := Diff(c.a, c.b); err != nil || got != c.want {
			t.Errorf("diff %v, want %v (%v)", got, c.want, err)
		}
	}
}

// The fidelity of #457's fixtures (recorded by #453's prototype) under this
// renderer: the baseline the recorder's own recordings are compared with.
//
//	FIDELITY_CHROME=1 go test ./cmd/internal/webcast/recorder/internal/fidelity -run Fixtures -v
//	FIDELITY_DIR=<recording> go test ./cmd/internal/webcast/recorder/internal/fidelity -run Dir -v
func TestFidelityFixtures(t *testing.T) {
	if os.Getenv("FIDELITY_CHROME") == "" {
		t.Skip("set FIDELITY_CHROME=1 to replay in a real Chrome")
	}
	for _, name := range []string{"stress", "app", "blog"} {
		rep, err := Run(context.Background(), chrome(t), rec.Fixture(name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s: %s", name, rep)
	}
}

func TestFidelityDir(t *testing.T) {
	dir := os.Getenv("FIDELITY_DIR")
	if dir == "" {
		t.Skip("set FIDELITY_DIR to a recording")
	}
	for _, d := range strings.Split(dir, ",") {
		rep, err := Run(context.Background(), chrome(t), d)
		if err != nil {
			t.Fatalf("%s: %v", d, err)
		}
		t.Logf("%s: %s", d, rep)
	}
}

func chrome(t *testing.T) string {
	bin, err := webcast.FindChrome(os.Getenv, exec.LookPath, func(p string) bool { _, err := os.Stat(p); return err == nil })
	if err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestPickAndSummarize(t *testing.T) {
	a := Pick("a", 1, []Try{{-200, 3}, {0, 1}, {200, 2}})
	none := Pick("b", 2, nil)
	c := Pick("c", 3, []Try{{0, 5}})
	if a.Diff != 1 || a.Offset != 0 || none.Diff != -1 {
		t.Fatalf("pick: %+v %+v", a, none)
	}
	rep := Summarize([]Shot{a, none, c})
	if rep.Skipped != 1 || len(rep.Shots) != 2 || rep.Max != 5 || rep.Median != 5 {
		t.Fatalf("summary %+v", rep)
	}
	if got := Summarize([]Shot{none}); got.Skipped != 1 || len(got.Shots) != 0 {
		t.Fatalf("all skipped: %+v", got)
	}
}
