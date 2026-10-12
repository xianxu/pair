package notes

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// defaultMargin is the space around the notes in a crop, in CSS px.
const defaultMargin = 48

const unbounded = 1e9

// crops computes a submission's screenshot regions; see Structural.Crops.
// resolve finds an anchor's element now, for its landmark.
func crops(st *recording.State, ats []recording.NoteAt, margin float64, hints Hints,
	resolve func(*recording.State, recording.NoteAt) (recording.NodeID, bool)) ([]recording.Box, error) {
	var out []recording.Box
	page := pageBox(st)
	for _, at := range ats {
		id, live := resolve(st, at)
		b, err := anchorBox(st, at, id)
		if err != nil {
			return nil, err
		}
		page = page.Union(b) // clipping never cuts a note's own box
		c := pad(b, margin)
		if live {
			if land, ok := landmark(st.Node(id)); ok {
				if lb, ok := st.Boxes[land.ID]; ok {
					c = snap(c, lb, snapHeight(st))
				}
			}
		}
		out = append(out, c)
	}
	out = cluster(out, margin)
	for i := range out {
		out[i] = clip(out[i], page)
		if hints != nil {
			if h, ok := hints.Crop(st.URL, out[i]); ok {
				out[i] = h
			}
		}
	}
	return out, nil
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

// snap widens c to its landmark per axis: always across (a header or nav
// goes full width), and down only when the landmark is at most maxH tall
// (otherwise the height stays the notes plus margin).
func snap(c, land recording.Box, maxH float64) recording.Box {
	u := c.Union(land)
	if land[3] > maxH {
		u[1], u[3] = c[1], c[3]
	}
	return u
}

// snapHeight is the tallest landmark a crop snaps to vertically: half the
// viewport.
func snapHeight(st *recording.State) float64 {
	if st.VP.H > 0 {
		return float64(st.VP.H) / 2
	}
	return 400
}

// cluster merges crops less than margin apart, repeatedly, so notes near
// each other share a crop and notes far apart get one each.
func cluster(cs []recording.Box, margin float64) []recording.Box {
	for merged := true; merged; {
		merged = false
		for i := 0; i < len(cs) && !merged; i++ {
			for j := i + 1; j < len(cs); j++ {
				if gap(cs[i], cs[j]) <= margin {
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

// clip keeps c on the page.
func clip(c, page recording.Box) recording.Box {
	x0, y0 := max(c[0], page[0]), max(c[1], page[1])
	x1, y1 := min(c[0]+c[2], page[0]+page[2]), min(c[1]+c[3], page[1]+page[3])
	return recording.Box{x0, y0, x1 - x0, y1 - y0}
}
