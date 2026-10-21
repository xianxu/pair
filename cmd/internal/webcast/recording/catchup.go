package recording

// CatchUp is what a viewer joining mid-cast receives before the live
// records, from everything recorded so far:
//
//   - the head;
//   - every asset binding (the table outlives segments);
//   - every bare segment and stop, preserving each note's document identity;
//   - all notes, edits, states and clears; only current-document strokes;
//   - the latest control change (ctl) before the latest segment: it
//     carries the whole control state (#466);
//   - every record from the latest segment on, except hover and ptr: of
//     those only the latest of each, in place (they are state, the
//     operator's pointer now, not history a viewer replays; #458).
//
// Order is kept. Applied to a fresh state, the catch-up gives the same DOM,
// assets, DocSeg and all note state as the whole recording.
func CatchUp(recs []Record) []Record {
	hasSketch := false
	for _, r := range recs {
		if sketchMetadata(r) != nil || r.Kind() == "sketch-snapshot" {
			hasSketch = true
			break
		}
	}
	if !hasSketch {
		return catchUpRecords(recs)
	}
	state, err := ReduceCatchUp(recs, int64(^uint64(0)>>1))
	if err != nil {
		return append([]Record(nil), recs...)
	}
	retained := catchUpRecords(recs)
	out := make([]Record, 0, len(retained)+1)
	for _, r := range retained {
		if sketchMetadata(r) == nil && r.Kind() != "sketch-snapshot" && r.Kind() != "ctl" {
			out = append(out, r)
		}
	}
	return append(out, state.SketchSnapshot())
}

func catchUpRecords(recs []Record) []Record {
	last, doc := -1, -1
	for i, r := range recs {
		if s, ok := r.(*Seg); ok {
			last = i
			if s.Why != SegResume {
				doc = i
			}
		}
	}
	if last < 0 {
		return append([]Record(nil), recs...)
	}
	ctl := -1 // the latest control change before the latest segment
	for i := last - 1; i >= 0 && ctl < 0; i-- {
		if _, ok := recs[i].(*Ctl); ok {
			ctl = i
		}
	}
	if doc < 0 {
		doc = 0
	}
	hover, ptr := -1, -1 // the latest of each in the latest segment
	for i := len(recs) - 1; i >= last && (hover < 0 || ptr < 0); i-- {
		switch recs[i].(type) {
		case *Hover:
			if hover < 0 {
				hover = i
			}
		case *Ptr:
			if ptr < 0 {
				ptr = i
			}
		}
	}
	var out []Record
	kept := map[string]bool{} // notes the catch-up carries
	for i, r := range recs {
		switch r := r.(type) {
		case *Seg, *Stop:
			out = append(out, r)
			continue
		case *Head, *Asset:
			if i < last {
				out = append(out, r)
				continue
			}
		case *Note:
			kept[r.ID] = true
			out = append(out, r)
			continue
		case *NoteEdit:
			if kept[r.ID] {
				out = append(out, r)
			}
			continue
		case *NoteState:
			if kept[r.ID] {
				out = append(out, r)
			}
			continue
		case *Stroke:
			if i >= doc {
				out = append(out, r)
			}
			continue
		case *AnnotationClear:
			out = append(out, r)
			clear(kept)
			continue
		case *Ctl:
			if i == ctl {
				out = append(out, r)
				continue
			}
		case *Hover:
			if i != hover {
				continue
			}
		case *Ptr:
			if i != ptr {
				continue
			}
		}
		if i >= last {
			out = append(out, r)
		}
	}
	return out
}
