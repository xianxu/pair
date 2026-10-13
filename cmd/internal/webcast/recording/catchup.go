package recording

// CatchUp is what a viewer joining mid-cast receives before the live
// records, from everything recorded so far:
//
//   - the head;
//   - every asset binding (the table outlives segments);
//   - the segment the current document began in (the latest that isn't a
//     resume: a resume shows the same document as before its pause), as a
//     bare seg record, so the notes keep their segment;
//   - notes, note states and strokes since then;
//   - the latest control change (ctl) before the latest segment: it
//     carries the whole control state (#466);
//   - every record from the latest segment on, except hover and ptr: of
//     those only the latest of each, in place (they are state, the
//     operator's pointer now, not history a viewer replays; #458).
//
// Order is kept. Applied to a fresh state, the catch-up gives the same DOM,
// assets, DocSeg and current-document notes as the whole recording.
func CatchUp(recs []Record) []Record {
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
		if i == doc && doc < last {
			// The segment the document began in: its number is where
			// the document's notes count from (State.DocSeg).
			out = append(out, r)
			continue
		}
		switch r := r.(type) {
		case *Head, *Asset:
			if i < last {
				out = append(out, r)
				continue
			}
		case *Note:
			if i >= doc {
				kept[r.ID] = true
				out = append(out, r)
			}
			continue
		case *NoteState:
			// A note from an earlier document stays behind, and so do its
			// state changes.
			if i >= doc && kept[r.ID] {
				out = append(out, r)
			}
			continue
		case *Stroke:
			if i >= doc {
				out = append(out, r)
			}
			continue
		case *AnnotationClear:
			if i >= doc {
				out = append(out, r)
				clear(kept)
			}
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
