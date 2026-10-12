package fidelity

import (
	"image"
	"image/color"
	"image/draw"
)

// CropPad is the margin around the differing pixels in a crop, and Gap
// the space between a side-by-side image's panels (image px).
const (
	CropPad = 8
	Gap     = 4
)

// DiffBox is the bounding box of the pixels that differ between a and b
// (outside the control's corner), padded by CropPad and clipped to the
// image; ok is false when none do. a and b are the same size.
func DiffBox(a, b image.Image) (box image.Rectangle, ok bool) {
	ra, rb := a.Bounds(), b.Bounds()
	mask := ControlCorner(ra.Dx(), ra.Dy())
	for y := 0; y < ra.Dy(); y++ {
		for x := 0; x < ra.Dx(); x++ {
			p := image.Pt(x, y)
			if p.In(mask) || !Differs(a.At(ra.Min.X+x, ra.Min.Y+y), b.At(rb.Min.X+x, rb.Min.Y+y)) {
				continue
			}
			px := image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}
			if !ok {
				box, ok = px, true
			} else {
				box = box.Union(px)
			}
		}
	}
	if !ok {
		return image.Rectangle{}, false
	}
	return box.Inset(-CropPad).Intersect(image.Rect(0, 0, ra.Dx(), ra.Dy())), true
}

// SideBySide is box of the tab's frame, the same box of the replay, and
// the difference: the tab faded, with the differing pixels red.
func SideBySide(tab, replay image.Image, box image.Rectangle) image.Image {
	w, h := box.Dx(), box.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, 3*w+2*Gap, h))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.Gray{0x80}), image.Point{}, draw.Src)
	ta, ra := tab.Bounds().Min.Add(box.Min), replay.Bounds().Min.Add(box.Min)
	draw.Draw(out, image.Rect(0, 0, w, h), tab, ta, draw.Src)
	draw.Draw(out, image.Rect(w+Gap, 0, 2*w+Gap, h), replay, ra, draw.Src)
	red := color.NRGBA{0xff, 0, 0, 0xff}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ct, cr := tab.At(ta.X+x, ta.Y+y), replay.At(ra.X+x, ra.Y+y)
			c := red
			if !Differs(ct, cr) {
				r, g, b := blend(ct)
				l := uint8((0.299*r+0.587*g+0.114*b)/4 + 191) // a light grey of the tab
				c = color.NRGBA{l, l, l, 0xff}
			}
			out.SetNRGBA(2*(w+Gap)+x, y, c)
		}
	}
	return out
}
