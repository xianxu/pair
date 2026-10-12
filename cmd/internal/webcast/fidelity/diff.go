// Package fidelity scores a recording against the real tab (project
// Principle 6, #478): the share of pixels where the replay of a moment
// differs from the tab's screenshot of it (frames/), a tolerance, and a
// report with the worst moments side by side. Pure: the replay itself is
// webcast.CheckFidelity's, in a headless Chrome.
package fidelity

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// Threshold is pixelmatch's default (0.1) as a share of the largest YIQ
// distance, as #453's harness used: a pixel differs when its distance is
// over it.
const Threshold = 0.1

const maxYIQDelta = 35215.0

// ControlCorner is where the cast's in-page control sits in a half-scale
// screenshot (fixed, 12px from the bottom right): the real tab shows it,
// a replay doesn't, so a comparison leaves it out. Its rectangle, given the
// screenshot's size.
func ControlCorner(w, h int) image.Rectangle { return image.Rect(w-64, h-24, w, h) }

// Diff is the share of pixels that differ between two PNGs of one size,
// outside the control's corner.
func Diff(a, b []byte) (float64, error) {
	ia, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		return 0, err
	}
	ib, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	return DiffImages(ia, ib)
}

// DiffImages is Diff on decoded images.
func DiffImages(a, b image.Image) (float64, error) {
	ra, rb := a.Bounds(), b.Bounds()
	if ra.Dx() != rb.Dx() || ra.Dy() != rb.Dy() {
		return 1, fmt.Errorf("sizes differ: %v vs %v", ra.Size(), rb.Size())
	}
	mask := ControlCorner(ra.Dx(), ra.Dy())
	diff, n := 0, 0
	for y := 0; y < ra.Dy(); y++ {
		for x := 0; x < ra.Dx(); x++ {
			if image.Pt(x, y).In(mask) {
				continue
			}
			n++
			if Differs(a.At(ra.Min.X+x, ra.Min.Y+y), b.At(rb.Min.X+x, rb.Min.Y+y)) {
				diff++
			}
		}
	}
	return float64(diff) / float64(n), nil
}

// Differs is the per-pixel test: the YIQ distance is over Threshold.
func Differs(c1, c2 color.Color) bool { return yiqDelta(c1, c2) > maxYIQDelta*Threshold*Threshold }

// yiqDelta is pixelmatch's color distance, blending alpha over white.
func yiqDelta(c1, c2 color.Color) float64 {
	r1, g1, b1 := blend(c1)
	r2, g2, b2 := blend(c2)
	y := rgb2y(r1, g1, b1) - rgb2y(r2, g2, b2)
	i := rgb2i(r1, g1, b1) - rgb2i(r2, g2, b2)
	q := rgb2q(r1, g1, b1) - rgb2q(r2, g2, b2)
	return 0.5053*y*y + 0.299*i*i + 0.1957*q*q
}

func blend(c color.Color) (float64, float64, float64) {
	r, g, b, a := c.RGBA()
	af := float64(a) / 0xffff
	f := func(v uint32) float64 { return 255 + (float64(v)/0xffff*255-255)*af }
	// RGBA is premultiplied; un-premultiply first.
	if a == 0 {
		return 255, 255, 255
	}
	un := func(v uint32) uint32 { return uint32(float64(v) / af) }
	return f(un(r)), f(un(g)), f(un(b))
}

func rgb2y(r, g, b float64) float64 { return r*0.29889531 + g*0.58662247 + b*0.11448223 }
func rgb2i(r, g, b float64) float64 { return r*0.59597799 - g*0.27417610 - b*0.32180189 }
func rgb2q(r, g, b float64) float64 { return r*0.21147017 - g*0.52261711 + b*0.31114694 }
