package notes

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// defaultMargin is the space around a note's box in its crop, in CSS px.
const defaultMargin = 16

const unbounded = 1e9

// noteCrops computes a submission's screenshot regions (#488): each note's
// box (see noteBox) plus the margin, crops that overlap merged into one,
// each clipped to the page. Clipping never cuts a note's own box. resolve
// finds an older note's element now, for its layout box.
func noteCrops(st *recording.State, ns []*recording.Note, margin float64,
	resolve func(*recording.State, recording.NoteAt) (recording.NodeID, bool)) ([]recording.Box, error) {
	var out []recording.Box
	page := pageBox(st)
	for _, n := range ns {
		b, err := noteBox(st, n, resolve)
		if err != nil {
			return nil, err
		}
		page = page.Union(b)
		out = append(out, pad(b, margin))
	}
	out = merge(out)
	for i := range out {
		out[i] = clip(out[i], page)
	}
	return out, nil
}

// noteBox is the area a note is about: its region, which every note made
// since #488 has; an older one's anchor box (anchorBox).
func noteBox(st *recording.State, n *recording.Note,
	resolve func(*recording.State, recording.NoteAt) (recording.NodeID, bool)) (recording.Box, error) {
	if n.Region != nil {
		return *n.Region, nil
	}
	id, _ := resolve(st, n.At)
	return anchorBox(st, n.At, id)
}

// anchorBox is the box a note was made on: its own (a viewer-made note's box
// is the viewer's, computed at click time), else the layout's for the
// element it resolves to now (id).
func anchorBox(st *recording.State, at recording.NoteAt, id recording.NodeID) (recording.Box, error) {
	if at.Box != nil {
		return *at.Box, nil
	}
	if b, ok := st.Boxes[id]; ok {
		return b, nil
	}
	return recording.Box{}, fmt.Errorf("notes: no box for node %d", at.Node)
}

func pad(b recording.Box, m float64) recording.Box {
	return recording.Box{b[0] - m, b[1] - m, b[2] + 2*m, b[3] + 2*m}
}

// merge joins crops that overlap, repeatedly, so overlapping notes share
// one image and the rest get one each.
func merge(cs []recording.Box) []recording.Box {
	for merged := true; merged; {
		merged = false
		for i := 0; i < len(cs) && !merged; i++ {
			for j := i + 1; j < len(cs); j++ {
				if gap(cs[i], cs[j]) < 0 {
					cs[i] = cs[i].Union(cs[j])
					cs = append(cs[:j], cs[j+1:]...)
					merged = true
					break
				}
			}
		}
	}
	return cs
}

// gap is the distance between two boxes along the axis they are furthest
// apart on; zero or less when they overlap.
func gap(a, b recording.Box) float64 {
	dx := max(b[0]-(a[0]+a[2]), a[0]-(b[0]+b[2]))
	dy := max(b[1]-(a[1]+a[3]), a[1]-(b[1]+b[3]))
	return max(dx, dy)
}

// pageBox is the top document's extent: its root element's box, else the
// viewport's width with no bottom (unbounded is 1e9 px).
func pageBox(st *recording.State) recording.Box {
	if st.Top != nil {
		for _, k := range st.Top.Kids {
			if b, ok := st.Boxes[k.ID]; ok && k.Y == recording.YEl {
				return b
			}
		}
	}
	w := float64(st.VP.W)
	if w == 0 {
		w = unbounded
	}
	return recording.Box{0, 0, w, unbounded}
}

// ClampToPage is a viewer's note box kept on the page (#488): the part of
// b inside the page (pageBox, widened to the viewport). false when none of
// it is: there is nothing there to note.
func ClampToPage(st *recording.State, b recording.Box) (recording.Box, bool) {
	c := clip(b, pageBox(st).Union(recording.Box{0, 0, float64(st.VP.W), float64(st.VP.H)}))
	return c, c[2] > 0 && c[3] > 0
}

// clip keeps c on the page.
func clip(c, page recording.Box) recording.Box {
	x0, y0 := max(c[0], page[0]), max(c[1], page[1])
	x1, y1 := min(c[0]+c[2], page[0]+page[2]), min(c[1]+c[3], page[1]+page[3])
	return recording.Box{x0, y0, x1 - x0, y1 - y0}
}
