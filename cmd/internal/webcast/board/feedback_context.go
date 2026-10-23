package board

import (
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

// feedbackContexts owns only scalar witnesses, never reducer or record pointers.
// records is the next absolute stream ordinal: the initial head is index zero.
type feedbackContexts struct {
	records  int
	sketches map[string]feedbackSketchContext
}
type feedbackSketchContext struct {
	source, finish query.SketchSourceFacts
	finished       bool
}

func (b *Board) feedbackContextLocked(index int) query.SketchSourceFacts {
	s := b.st
	v := query.SketchSourceFacts{T: s.T, RecordIndex: index, Doc: s.DocSeg, Tab: s.Tab.ID, URL: s.URL, Title: s.Title, VP: s.VP}
	if s.Top != nil {
		v.Scroll = s.Scrolls[s.Top.ID]
	}
	return v
}

// Called exactly once after every successfully applied ordered record. Duplicate
// receipts still occupy stream ordinals but cannot replace accepted witnesses.
func (b *Board) refreshFeedbackContextLocked(r recording.Record, duplicate bool) {
	index := b.feedback.records
	b.feedback.records++
	switch v := r.(type) {
	case *recording.SketchBegin:
		if duplicate {
			return
		}
		if b.feedback.sketches == nil {
			b.feedback.sketches = make(map[string]feedbackSketchContext)
		}
		if len(b.feedback.sketches) >= recording.SketchMaxCount {
			return
		}
		source := b.feedbackContextLocked(index)
		source.Doc, source.Tab, source.VP = v.Doc, v.Tab, v.VP
		b.feedback.sketches[v.ID] = feedbackSketchContext{source: source}
	case *recording.SketchLabel:
		if duplicate {
			return
		}
		witness, ok := b.feedback.sketches[v.Sketch]
		if !ok {
			return
		}
		witness.finish = b.feedbackContextLocked(index)
		witness.finished = true
		b.feedback.sketches[v.Sketch] = witness
		b.submissionChangedLocked()
	case *recording.Note, *recording.NoteEdit, *recording.NoteState:
		// Successful reducer events create a note or advance its revision. Invalid
		// events never reach this seam, and unrelated records do no note scans.
		b.submissionChangedLocked()
	case *recording.AnnotationClear:
		clear(b.feedback.sketches)
		b.submissionChangedLocked()
	}
}

// currentFeedbackPageLocked pins the absolute ordered recording boundary. No
// journal is copied, and the caller already owns the Board mutex.
func (b *Board) currentFeedbackPageLocked() sketchimage.PageScene {
	if b.st.Head == nil || b.st.Top == nil || b.feedback.records == 0 {
		return sketchimage.PageScene{}
	}
	return sketchimage.PageScene{RecordingID: b.st.Head.ID, SketchSourceFacts: b.feedbackContextLocked(b.feedback.records - 1)}
}
