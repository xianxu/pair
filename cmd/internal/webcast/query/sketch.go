package query

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// SketchSelection identifies an exact accepted prefix, not a moving latest
// lookup. Digest is the last receipt's digest; NextSequence includes begin.
type SketchSelection struct {
	ID           string `json:"id"`
	NextSequence uint64 `json:"next_sequence"`
	Digest       string `json:"digest"`
}
type SketchSourceFacts struct {
	PageLoad PageLoadFacts `json:"page_load,omitzero"`
	T        int64         `json:"t"`
	// RecordIndex is zero-based and inclusive. Render this exact input prefix:
	// source time alone cannot distinguish subsequent records at the same time.
	RecordIndex int                `json:"record_index"`
	Doc         int                `json:"doc"`
	Tab         string             `json:"tab"`
	URL         string             `json:"url"`
	Title       string             `json:"title"`
	VP          recording.Viewport `json:"vp"`
	Scroll      [2]float64         `json:"scroll"`
}
type SketchView struct {
	// HistoryRecordIndex is the exact inclusive accepted/contextual history boundary.
	HistoryRecordIndex int                  `json:"history_record_index"`
	History            recording.SketchView `json:"history"`
	HistoryT           int64                `json:"history_t"`
	Source             SketchSourceFacts    `json:"source"`
	Elements           DeclaredElementFacts `json:"elements"`
}
type sketchQueryStats struct{ Records, SketchEvents, SourceNodes int }
type sketchQueryHistory struct {
	state       *recording.State
	source      SketchSourceFacts
	elements    elementFactSource
	selected    SketchSelection
	recordIndex int
	frozen      bool
}

// SketchesAt scans the ordered recording once through inclusive history time.
// Without selections it returns every sketch in creation order. Selections
// freeze their exact accepted prefixes and refuse missing or changed versions.
// History and declared facts are owned values; no live page is consulted.
func (q *Querier) SketchesAt(at int64, selected ...SketchSelection) ([]SketchView, error) {
	views, _, err := q.sketchesAt(at, selected)
	return views, err
}
func (q *Querier) sketchesAt(at int64, selected []SketchSelection) ([]SketchView, sketchQueryStats, error) {
	var stats sketchQueryStats
	fail := func(err error) ([]SketchView, sketchQueryStats, error) { return nil, stats, err }
	if at < 0 {
		return fail(fmt.Errorf("query: negative sketch history time"))
	}
	wanted := map[string]SketchSelection{}
	for _, sel := range selected {
		if sel.ID == "" || sel.NextSequence == 0 || sel.NextSequence > recording.MaxSketchInteger || !recording.ValidHash(sel.Digest) {
			return fail(fmt.Errorf("query: invalid sketch selection"))
		}
		if _, exists := wanted[sel.ID]; exists {
			return fail(fmt.Errorf("query: duplicate sketch selection %q", sel.ID))
		}
		wanted[sel.ID] = sel
	}
	state := recording.NewState()
	histories := map[string]*sketchQueryHistory{}
	order := []string{}
	for i, r := range q.recs {
		if r.Time() > at {
			break
		}
		stats.Records++
		if err := state.Apply(r); err != nil {
			return fail(unreadable(i, r, err))
		}
		id := querySketchID(r)
		if begin, ok := r.(*recording.SketchBegin); ok && histories[id] == nil {
			sel, requested := wanted[id]
			if len(selected) > 0 && !requested {
				continue
			}
			// A tiny per-sketch reducer reuses the transition owner, without copying
			// DOM or repeatedly projecting every other sketch's historical points.
			local := recording.NewState()
			local.Head = state.Head
			local.Live = state.Live
			local.Ended = state.Ended
			local.Seg = state.Seg
			local.DocSeg = state.DocSeg
			local.Tab = state.Tab
			local.VP = state.VP
			local.Control = state.Control
			source := SketchBeginSourceFacts(state, begin, i)
			facts := captureElementFacts(state)
			stats.SourceNodes += facts.inspected
			histories[id] = &sketchQueryHistory{state: local, source: source, elements: facts, selected: sel, recordIndex: i}
			order = append(order, id)
		}
		if id != "" {
			h := histories[id]
			if h == nil || h.frozen {
				continue
			}
			if err := h.state.Apply(r); err != nil {
				return fail(unreadable(i, r, err))
			}
			if h.state.T == r.Time() {
				h.recordIndex = i
			}
			stats.SketchEvents++
			if h.selected.NextSequence > 0 {
				meta, _ := h.state.Sketches.Metadata(id)
				if meta.NextSequence == h.selected.NextSequence {
					h.frozen = true
				}
			}
		} else {
			switch r.(type) {
			case *recording.Seg, *recording.Stop, *recording.Ctl, *recording.AnnotationClear:
				for _, id := range order {
					h := histories[id]
					if h.frozen {
						continue
					}
					if err := h.state.Apply(r); err != nil {
						return fail(unreadable(i, r, err))
					}
					h.recordIndex = i
				}
			}
		}
	}
	if len(selected) > 0 && len(histories) != len(selected) {
		return fail(fmt.Errorf("query: selected sketch missing at history cutoff"))
	}
	out := make([]SketchView, 0, len(order))
	for _, id := range order {
		h := histories[id]
		history := h.state.Sketches.Snapshot().Sketches[0]
		if h.selected.NextSequence > 0 && (!h.frozen || history.NextSequence != h.selected.NextSequence || history.Receipts[len(history.Receipts)-1].Digest != h.selected.Digest) {
			return fail(fmt.Errorf("query: sketch %q selected prefix differs or is missing", id))
		}
		anchors := make([]recording.NodeID, 0, len(history.Strokes))
		var ink *recording.Box
		visible := map[string]bool{}
		for _, id := range history.Visible {
			visible[id] = true
		}
		for _, stroke := range history.Strokes {
			anchors = append(anchors, stroke.Begin.At.Node)
			if history.Phase == "cleared" || !visible[stroke.Begin.ID] && stroke.Sealed {
				continue
			}
			for _, p := range stroke.Pts {
				// Pressure changes paint width; keep that width when finding nearby facts.
				radius := stroke.Begin.Width * (.25 + .75*p[2]) / 2
				box := recording.Box{p[0] - radius, p[1] - radius, 2 * radius, 2 * radius}
				if ink == nil {
					ink = &box
				} else {
					union := ink.Union(box)
					ink = &union
				}
			}
		}
		out = append(out, SketchView{HistoryRecordIndex: h.recordIndex, History: history, HistoryT: h.state.T, Source: h.source, Elements: h.elements.facts(anchors, ink)})
	}
	return out, stats, nil
}
func querySketchID(r recording.Record) string {
	switch r := r.(type) {
	case *recording.SketchBegin:
		return r.ID
	case *recording.SketchStrokeBegin:
		return r.Sketch
	case *recording.SketchPoints:
		return r.Sketch
	case *recording.SketchStrokeEnd:
		return r.Sketch
	case *recording.SketchUndo:
		return r.Sketch
	case *recording.SketchRedo:
		return r.Sketch
	case *recording.SketchLabel:
		return r.Sketch
	case *recording.SketchInterrupt:
		return r.Sketch
	}
	return ""
}

// SketchText names both independent clocks and the historical recording handles.
func SketchText(v SketchView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "sketch %s: %s", v.History.Begin.ID, v.History.Phase)
	if v.History.Label == "" {
		b.WriteString(" (unfinished: label required)")
	} else {
		fmt.Fprintf(&b, " %q", v.History.Label)
	}
	fmt.Fprintf(&b, "\nsource t=%d record=%d doc=%d tab=%s %s\nhistory t=%d next_sequence=%d author=%s color=%s visible=%d redo=%d", v.Source.T, v.Source.RecordIndex, v.Source.Doc, v.Source.Tab, v.Source.URL, v.HistoryT, v.History.NextSequence, v.History.Begin.Author.ID, v.History.Begin.Author.Color, len(v.History.Visible), len(v.History.Redo))
	if v.History.Reason != "" {
		fmt.Fprintf(&b, " reason=%s", v.History.Reason)
	}
	b.WriteByte('\n')
	if v.Source.PageLoad != (PageLoadFacts{}) {
		fmt.Fprintf(&b, "page_load: target=%q loader=%q root=%q\n", v.Source.PageLoad.Target, v.Source.PageLoad.Loader, v.Source.PageLoad.Root)
	}
	for _, f := range v.Elements.Anchors {
		fmt.Fprintf(&b, "anchor @%d <%s> id=%q", f.Node, f.Tag, f.ID)
		if f.Unavailable != "" {
			fmt.Fprintf(&b, " unavailable=%q", f.Unavailable)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
