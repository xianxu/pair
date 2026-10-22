package board

import (
	"maps"
	"slices"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// OperatorStroke is one recorded chunk, not an accumulated stroke. Seq never
// repeats within a Board, even when the transient history is pruned.
type OperatorStroke struct {
	Seq uint64 `json:"seq"`
	recording.Stroke
	received time.Time
}

// OperatorView is a small projection of the current authoritative Board state.
// FADE_MS in web/annotate.js is the corresponding client fade interval.
const OperatorStrokeFadeMS int64 = 1500
const MaxOperatorStrokes = 128

type OperatorView struct {
	FeedbackMask         LiveFeedbackMask     `json:"feedback_mask"`
	SketchCursor         SketchDeliveryCursor `json:"sketch_cursor"`
	SketchAnchorRevision uint64               `json:"sketch_anchor_revision"`
	Revision             uint64               `json:"revision"`
	Seg                  int                  `json:"seg"`
	Doc                  int                  `json:"doc"`
	T                    int64                `json:"t"`
	Paused               bool                 `json:"paused"`
	Ended                bool                 `json:"ended"`
	Driver               string               `json:"driver"`
	Notes                []*recording.Note    `json:"notes"`
	Strokes              []OperatorStroke     `json:"strokes"`
}

func (b *Board) pruneOperatorLocked(now time.Time) {
	before := len(b.operatorStrokes)
	defer func() {
		if len(b.operatorStrokes) != before {
			b.operatorRevision++
		}
	}()
	b.operatorStrokes = slices.DeleteFunc(b.operatorStrokes, func(s OperatorStroke) bool {
		dt := float64(0)
		for _, p := range s.Pts {
			dt = max(dt, p[2])
		}
		life := OperatorStrokeFadeMS + int64(dt)
		return b.st.T-s.T >= life || now.Sub(s.received) >= time.Duration(life)*time.Millisecond
	})
	if len(b.operatorStrokes) > MaxOperatorStrokes {
		b.operatorStrokes = slices.Clone(b.operatorStrokes[len(b.operatorStrokes)-MaxOperatorStrokes:])
	}
}
func (b *Board) refreshOperatorLocked(r recording.Record) {
	if _, clear := r.(*recording.AnnotationClear); clear {
		b.operatorStrokes = nil
	}
	if b.operatorDoc != b.st.DocSeg {
		b.operatorDoc = b.st.DocSeg
		b.operatorStrokes = nil
	}
	now := time.Now()
	if stroke, ok := r.(*recording.Stroke); ok && len(stroke.Pts) <= broadcast.MaxPointsPerBatch {
		b.operatorSeq++
		copy := *stroke
		copy.Pts = slices.Clone(stroke.Pts)
		b.operatorStrokes = append(b.operatorStrokes, OperatorStroke{Seq: b.operatorSeq, Stroke: copy, received: now})
	}
	b.pruneOperatorLocked(now)
	b.operatorRevision++
	for ch := range b.operatorSubs {
		select {
		case <-ch:
		default:
		}
		ch <- b.operatorViewLocked()
	}
}
func cloneNote(n *recording.Note) *recording.Note {
	c := *n
	c.At.Path = slices.Clone(n.At.Path)
	c.At.Classes = slices.Clone(n.At.Classes)
	c.At.Stable = maps.Clone(n.At.Stable)
	if n.At.Box != nil {
		x := *n.At.Box
		c.At.Box = &x
	}
	if n.Region != nil {
		x := *n.Region
		c.Region = &x
	}
	return &c
}
func (b *Board) operatorViewLocked() OperatorView {
	v := OperatorView{Revision: b.operatorRevision, Seg: b.st.Seg, Doc: b.st.DocSeg, T: b.st.T, Paused: b.st.Paused(), Ended: b.closed || b.st.Ended || b.broken != nil, Notes: []*recording.Note{}, Strokes: []OperatorStroke{}}
	v.SketchCursor = b.sketchDeliveryCursorLocked()
	v.FeedbackMask = cloneFeedbackMask(b.feedbackDisplay.mask)
	v.SketchAnchorRevision = b.st.Sketches.AnchorRevision()
	if b.st.Control.Driver != nil {
		v.Driver = b.st.Control.Driver.Name
		if v.Driver == "" {
			v.Driver = b.st.Control.Driver.ID
		}
	}
	for _, n := range b.st.Notes {
		if n.State == recording.NoteOpen && !b.noteFeedbackHiddenLocked(n.ID, n.Revision) {
			v.Notes = append(v.Notes, cloneNote(n.Note))
		}
	}
	for _, s := range b.operatorStrokes {
		s.Pts = slices.Clone(s.Pts)
		v.Strokes = append(v.Strokes, s)
	}
	return v
}
func (b *Board) OperatorView() OperatorView {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pruneOperatorLocked(time.Now())
	return b.operatorViewLocked()
}

// SubscribeOperator holds only the latest snapshot, with independent copies for
// each subscriber. The cancel function and Close may be called concurrently.
func (b *Board) SubscribeOperator() (<-chan OperatorView, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pruneOperatorLocked(time.Now())
	ch := make(chan OperatorView, 1)
	ch <- b.operatorViewLocked()
	if b.closed {
		close(ch)
		return ch, func() {}
	}
	if b.operatorSubs == nil {
		b.operatorSubs = map[chan OperatorView]struct{}{}
	}
	b.operatorSubs[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.operatorSubs[ch]; ok {
			delete(b.operatorSubs, ch)
			close(ch)
		}
	}
}

// Sketch anchor identity is metadata, independent of visibility and pressure
// points. The worker requests the owned list only when this revision changes.
func (b *Board) SketchAnchorRevision() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.st.Sketches.AnchorRevision()
}
func (b *Board) SketchAnchorNodes(document int) []recording.NodeID {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.st.Sketches.AnchorNodes(document)
}
