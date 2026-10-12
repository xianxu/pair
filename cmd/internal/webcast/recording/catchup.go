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
//   - every record from the latest segment on.
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
	if doc < 0 {
		doc = 0
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
		}
		if i >= last {
			out = append(out, r)
		}
	}
	return out
}
