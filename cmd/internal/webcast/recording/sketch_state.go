package recording

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
)

// SketchProjection is an owned, deterministic snapshot. Appending points never
// builds this projection; consumers request it off the recording hot path.
type SketchProjection struct {
	Active       string       `json:"active"`
	HistoryBytes int          `json:"history_bytes"`
	Sketches     []SketchView `json:"sketches"`
}
type SketchView struct {
	Begin        *SketchBegin       `json:"begin"`
	Phase        string             `json:"phase"`
	Reason       string             `json:"reason"`
	Label        string             `json:"label"`
	Strokes      []SketchStrokeView `json:"strokes"`
	Visible      []string           `json:"visible"`
	Redo         []string           `json:"redo"`
	NextSequence uint64             `json:"next_sequence"`
	Receipts     []SketchReceipt    `json:"receipts"`
}
type SketchStrokeView struct {
	Begin     *SketchStrokeBegin `json:"begin"`
	Pts       []SketchPoint      `json:"pts"`
	NextChunk uint64             `json:"next_chunk"`
	Sealed    bool               `json:"sealed"`
	Truncated bool               `json:"truncated"`
}
type SketchReceipt struct {
	Operation string `json:"operation"`
	Sequence  uint64 `json:"sequence"`
	Digest    string `json:"digest"`
	Semantic  string `json:"semantic"`
}
type sketchHistory struct {
	begin                      *SketchBegin
	phase, reason, label, open string
	strokes                    map[string]*SketchStrokeView
	shown                      map[string]bool
	order, visible, redo       []string
	next                       uint64
	receipts                   map[string]SketchReceipt
	receiptOrder               []string
	points                     int
}

// SketchState owns history by identity, with O(new points) append and constant
// stack changes. The zero value is ready for use.
type SketchState struct {
	active         string
	historyBytes   int
	order          []string
	sketches       map[string]*sketchHistory
	work           struct{ points, stacks uint64 }
	anchors        map[int]map[NodeID]bool
	anchorRevision uint64
}

// SketchID returns a record's owning sketch identity, or empty for other kinds.
// Live admission and receipt reconciliation share this projection.
func SketchID(r Record) string {
	switch r := r.(type) {
	case *SketchBegin:
		return r.ID
	case *SketchStrokeBegin:
		return r.Sketch
	case *SketchPoints:
		return r.Sketch
	case *SketchStrokeEnd:
		return r.Sketch
	case *SketchUndo:
		return r.Sketch
	case *SketchRedo:
		return r.Sketch
	case *SketchLabel:
		return r.Sketch
	case *SketchInterrupt:
		return r.Sketch
	}
	return ""
}

func sketchID(r Record) string { return SketchID(r) }

// Reconcile is read-only. Producers call it before stamping or writing, so
// an exact retry cannot emit another JSONL line or consume another byte.
func (s *SketchState) Reconcile(r Record) (bool, error) {
	o := sketchMetadata(r)
	if o == nil {
		return false, nil
	}
	if err := r.check(); err != nil {
		return false, err
	}
	h := s.sketches[sketchID(r)]
	if h == nil {
		return false, nil
	}
	semantic, err := SketchSemantic(r)
	if err != nil {
		return false, err
	}
	if old, ok := h.receipts[o.Operation]; ok {
		if old.Sequence != o.Sequence || old.Digest != o.Digest || old.Semantic != semantic {
			return false, errors.New("sketch operation identity conflict")
		}
		return true, nil
	}
	if o.Sequence != h.next {
		return false, fmt.Errorf("sketch sequence %d; expected %d", o.Sequence, h.next)
	}
	return false, nil
}

type SketchContext struct {
	Live, Ended bool
	Doc         int
	Tab, Driver string
}

// SketchDelta is fully validated; only the state owner can commit it.
type SketchDelta struct{ commit func() }

// NextSketch validates without changing s, including all referenced identities.
func NextSketch(s *SketchState, r Record, c SketchContext) (SketchDelta, error) {
	fail := func(msg string) (SketchDelta, error) { return SketchDelta{}, errors.New(msg) }
	if sketchMetadata(r) == nil {
		return fail("not a sketch event")
	}
	dup, err := s.Reconcile(r)
	if err != nil {
		return SketchDelta{}, err
	}
	if dup {
		return SketchDelta{commit: func() {}}, nil
	}
	o := sketchMetadata(r)
	semantic, err := SketchSemantic(r)
	if err != nil {
		return SketchDelta{}, err
	}
	// Canonical known fields bound authoritative state; preserved wire-only
	// unknown members remain on original records and are charged by SketchBudget.
	size, err := sketchModelEncodedBytes(r, semantic)
	if err != nil {
		return SketchDelta{}, err
	}
	if size > SketchHistoryLimit-s.historyBytes {
		return fail("sketch history byte limit")
	}
	h := s.sketches[sketchID(r)]
	var effect func()
	if b, ok := r.(*SketchBegin); ok {
		if h != nil {
			return fail("sketch identity already used")
		}
		if o.Sequence != 0 {
			return fail("sketch begins at sequence zero")
		}
		if s.active != "" {
			return fail("another sketch is drawing")
		}
		if len(s.order) >= SketchMaxCount {
			return fail("sketch count limit")
		}
		if !c.Live || c.Ended || b.Doc != c.Doc || b.Tab != c.Tab {
			return fail("sketch source is not current and live")
		}
		author := c.Driver
		if author == "" {
			author = OperatorID
		}
		if b.Author.ID != author {
			return fail("sketch author is not driving")
		}
		owned, err := cloneSketchBegin(b)
		if err != nil {
			return SketchDelta{}, err
		}
		h = &sketchHistory{begin: owned, phase: "drawing", strokes: map[string]*SketchStrokeView{}, receipts: map[string]SketchReceipt{}}
		effect = func() {
			if s.sketches == nil {
				s.sketches = map[string]*sketchHistory{}
			}
			s.sketches[b.ID] = h
			s.order = append(s.order, b.ID)
			s.active = b.ID
		}
	} else {
		if h == nil {
			return fail("unknown sketch")
		}
		switch v := r.(type) {
		case *SketchLabel:
			if c.Ended {
				return fail("cast ended before sketch label")
			}
			if h.label != "" || h.phase == "cleared" || h.phase == "complete" {
				return fail("sketch label is final")
			}
			if h.open != "" {
				return fail("end stroke before label")
			}
			effect = func() {
				h.label = v.Label
				if h.phase == "drawing" {
					h.phase = "complete"
				}
				if s.active == v.Sketch {
					s.active = ""
				}
			}
		case *SketchStrokeEnd:
			stroke := h.strokes[v.ID]
			if stroke == nil {
				return fail("unknown stroke")
			}
			if h.phase == "cleared" || h.phase == "complete" {
				return fail("sketch history is closed")
			}
			if stroke.Sealed {
				effect = func() {}
			} else {
				if h.open != v.ID {
					return fail("stroke is not open")
				}
				effect = func() { s.seal(h, v.Truncated) }
			}
		default:
			if h.phase != "drawing" || s.active != sketchID(r) {
				return fail("sketch history is closed")
			}
			if _, interrupt := r.(*SketchInterrupt); !interrupt && (!c.Live || c.Ended || h.begin.Doc != c.Doc || h.begin.Tab != c.Tab) {
				return fail("sketch drawing is paused or stale")
			}
			switch v := r.(type) {
			case *SketchStrokeBegin:
				if h.open != "" {
					return fail("another stroke is open")
				}
				if h.strokes[v.ID] != nil {
					return fail("stroke identity already used")
				}
				if len(h.order) >= SketchMaxStrokes {
					return fail("sketch stroke limit")
				}
				owned, err := cloneSketchStroke(v)
				if err != nil {
					return SketchDelta{}, err
				}
				effect = func() {
					h.strokes[v.ID] = &SketchStrokeView{Begin: owned, Pts: []SketchPoint{}}
					h.order = append(h.order, v.ID)
					h.open = v.ID
					s.addAnchor(h.begin.Doc, owned.At.Node)
					h.redo = nil
				}
			case *SketchPoints:
				stroke := h.strokes[v.ID]
				if stroke == nil || h.open != v.ID || stroke.Sealed {
					return fail("stroke is not open")
				}
				if v.Chunk != stroke.NextChunk {
					return fail("stroke chunk is out of sequence")
				}
				if len(v.Pts) > SketchMaxPointsPerStroke-len(stroke.Pts) || len(v.Pts) > SketchMaxPointsPerSketch-h.points {
					return fail("sketch point limit")
				}
				pts := slices.Clone(v.Pts)
				for i := range pts {
					pts[i] = normalizeSketchPoint(pts[i])
				}
				effect = func() {
					stroke.Pts = append(stroke.Pts, pts...)
					stroke.NextChunk++
					h.points += len(pts)
					s.work.points += uint64(len(pts))
				}
			case *SketchUndo:
				if h.open != "" || len(h.visible) == 0 || h.visible[len(h.visible)-1] != v.ID {
					return fail("undo requires the visible stack top")
				}
				effect = func() {
					h.visible = h.visible[:len(h.visible)-1]
					h.redo = append(h.redo, v.ID)
					h.shown[v.ID] = false
					s.work.stacks++
				}
			case *SketchRedo:
				if h.open != "" || len(h.redo) == 0 || h.redo[len(h.redo)-1] != v.ID {
					return fail("redo requires the redo stack top")
				}
				effect = func() {
					h.redo = h.redo[:len(h.redo)-1]
					h.visible = append(h.visible, v.ID)
					h.shown[v.ID] = true
					s.work.stacks++
				}
			case *SketchInterrupt:
				effect = func() { s.interrupt(v.Reason) }
			default:
				return fail("unknown sketch event")
			}
		}
	}
	receipt := SketchReceipt{Operation: o.Operation, Sequence: o.Sequence, Digest: o.Digest, Semantic: semantic}
	return SketchDelta{commit: func() {
		effect()
		h.receipts[o.Operation] = receipt
		h.receiptOrder = append(h.receiptOrder, o.Operation)
		h.next++
		s.historyBytes += size
	}}, nil
}
func (s *SketchState) seal(h *sketchHistory, truncated bool) {
	if h.open == "" {
		return
	}
	stroke := h.strokes[h.open]
	stroke.Sealed = true
	stroke.Truncated = truncated
	if len(stroke.Pts) > 0 {
		h.visible = append(h.visible, h.open)
		if h.shown == nil {
			h.shown = map[string]bool{}
		}
		h.shown[h.open] = true
	}
	h.open = ""
}
func (s *SketchState) interrupt(reason string) {
	if s.active == "" {
		return
	}
	h := s.sketches[s.active]
	s.seal(h, true)
	h.phase = "interrupted"
	h.reason = reason
	s.active = ""
}
func (s *SketchState) pause() {
	if s.active != "" {
		s.seal(s.sketches[s.active], true)
	}
}
func (s *SketchState) clear() {
	if len(s.anchors) > 0 {
		s.anchors = nil
		s.anchorRevision++
	}
	for _, id := range s.order {
		s.seal(s.sketches[id], true)
		s.sketches[id].phase = "cleared"
		s.sketches[id].open = ""
	}
	s.active = ""
}
func cloneSketchBegin(b *SketchBegin) (*SketchBegin, error) {
	raw, err := Encode(b)
	if err != nil {
		return nil, err
	}
	r, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	owned := r.(*SketchBegin)
	owned.wire = nil
	return owned, nil
}
func cloneSketchStroke(b *SketchStrokeBegin) (*SketchStrokeBegin, error) {
	raw, err := Encode(b)
	if err != nil {
		return nil, err
	}
	r, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	owned := r.(*SketchStrokeBegin)
	owned.wire = nil
	if owned.At.Box != nil {
		for i, v := range *owned.At.Box {
			if v == 0 {
				owned.At.Box[i] = 0
			}
		}
	}
	return owned, nil
}
func (s *SketchState) Snapshot() SketchProjection { return s.snapshotForDocuments(nil) }

// SketchesOnPageLoad copies matching sketches without copying earlier loads'
// pressure history. Unknown identities never select historical feedback.
func (s *State) SketchesOnPageLoad(load *PageLoad) SketchProjection {
	return s.Sketches.snapshotForDocuments(func(doc int) bool { return s.OnPageLoad(doc, load) })
}

func (s *SketchState) snapshotForDocuments(include func(int) bool) SketchProjection {
	out := SketchProjection{Active: s.active, HistoryBytes: s.historyBytes, Sketches: []SketchView{}}
	for _, id := range s.order {
		h := s.sketches[id]
		if include != nil && !include(h.begin.Doc) {
			continue
		}
		begin, _ := cloneSketchBegin(h.begin)
		v := SketchView{Begin: begin, Phase: h.phase, Reason: h.reason, Label: h.label, Strokes: []SketchStrokeView{}, Visible: append([]string{}, h.visible...), Redo: append([]string{}, h.redo...), NextSequence: h.next, Receipts: []SketchReceipt{}}
		for _, sid := range h.order {
			stroke := *h.strokes[sid]
			stroke.Begin, _ = cloneSketchStroke(stroke.Begin)
			stroke.Pts = append([]SketchPoint{}, stroke.Pts...)
			v.Strokes = append(v.Strokes, stroke)
		}
		for _, op := range h.receiptOrder {
			v.Receipts = append(v.Receipts, h.receipts[op])
		}
		out.Sketches = append(out.Sketches, v)
	}
	return out
}

// SketchReceiptKey is scoped by sketch and operation identities. Identifiers
// reject control characters, so the separator cannot alias a legal identity.
func SketchReceiptKey(r Record) string {
	o := sketchMetadata(r)
	if o == nil {
		return ""
	}
	return sketchID(r) + "\x00" + o.Operation
}

// Model accounting uses the canonical known projection and envelope. Canonical
// numeric zeros match JavaScript; SketchBudget alone counts original wire bytes.
func sketchModelEncodedBytes(r Record, semantic string) (int, error) {
	var known map[string]json.RawMessage
	if err := json.Unmarshal([]byte(semantic[len(r.Kind())+1:]), &known); err != nil {
		return 0, err
	}
	known["t"], _ = json.Marshal(r.Time())
	known["k"], _ = json.Marshal(r.Kind())
	known["digest"], _ = json.Marshal(sketchMetadata(r).Digest)
	raw, err := json.Marshal(known)
	return len(raw) + 1, err
}
func normalizeSketchPoint(p SketchPoint) SketchPoint {
	for i, v := range p {
		if v == 0 {
			p[i] = 0
		}
	}
	return p
}

// SketchMetadata and SketchStrokeMetadata are owned small query values. They
// never contain historical points, receipt collections or mutable core maps.
type SketchMetadata struct {
	Digest       string   `json:"digest"`
	ID           string   `json:"id"`
	Author       Author   `json:"author"`
	Owner        string   `json:"owner"`
	Doc          int      `json:"doc"`
	Tab          string   `json:"tab"`
	VP           Viewport `json:"vp"`
	T            int64    `json:"t"`
	Phase        string   `json:"phase"`
	Reason       string   `json:"reason"`
	Label        string   `json:"label"`
	NextSequence uint64   `json:"next_sequence"`
	OpenStroke   string   `json:"open_stroke"`
}
type SketchStackTops struct {
	VisibleTop   string `json:"visible_top"`
	RedoTop      string `json:"redo_top"`
	VisibleCount int    `json:"visible_count"`
	RedoCount    int    `json:"redo_count"`
}

// SketchCandidate is small, owned metadata for a labeled visible sketch. It
// never copies pressure points or receipts; order is recording creation order.
type SketchCandidate struct {
	ID           string
	NextSequence uint64
	Digest       string
}

func (s *SketchState) SubmissionCandidates() []SketchCandidate {
	var out []SketchCandidate
	for _, id := range s.order {
		h := s.sketches[id]
		if h.label == "" || len(h.visible) == 0 || (h.phase != "complete" && h.phase != "interrupted") || len(h.receiptOrder) == 0 {
			continue
		}
		last := h.receipts[h.receiptOrder[len(h.receiptOrder)-1]]
		out = append(out, SketchCandidate{ID: id, NextSequence: h.next, Digest: last.Digest})
	}
	return out
}

type SketchStrokeMetadata struct {
	ID         string  `json:"id"`
	At         NoteAt  `json:"at"`
	Width      float64 `json:"width"`
	T          int64   `json:"t"`
	PointCount int     `json:"point_count"`
	NextChunk  uint64  `json:"next_chunk"`
	Visible    bool    `json:"visible"`
	Sealed     bool    `json:"sealed"`
	Truncated  bool    `json:"truncated"`
}
type SketchChange struct {
	Kind         string                `json:"kind"`
	Sketch       SketchMetadata        `json:"sketch"`
	Stroke       *SketchStrokeMetadata `json:"stroke,omitempty"`
	Chunk        uint64                `json:"chunk,omitempty"`
	Points       []SketchPoint         `json:"points,omitempty"`
	PointsOffset int                   `json:"points_offset,omitempty"`
}

func (s *SketchState) Receipt(id, operation string) (SketchReceipt, bool) {
	h := s.sketches[id]
	if h == nil {
		return SketchReceipt{}, false
	}
	r, ok := h.receipts[operation]
	return r, ok
}
func (s *SketchState) Metadata(id string) (SketchMetadata, bool) {
	h := s.sketches[id]
	if h == nil {
		return SketchMetadata{}, false
	}
	b := h.begin
	digest := ""
	if len(h.receiptOrder) > 0 {
		digest = h.receipts[h.receiptOrder[len(h.receiptOrder)-1]].Digest
	}
	return SketchMetadata{Digest: digest, ID: id, Author: b.Author, Owner: b.Owner, Doc: b.Doc, Tab: b.Tab, VP: b.VP, T: b.T, Phase: h.phase, Reason: h.reason, Label: h.label, NextSequence: h.next, OpenStroke: h.open}, true
}
func (s *SketchState) StackTops(id string) (SketchStackTops, bool) {
	h := s.sketches[id]
	if h == nil {
		return SketchStackTops{}, false
	}
	v := SketchStackTops{VisibleCount: len(h.visible), RedoCount: len(h.redo)}
	if len(h.visible) > 0 {
		v.VisibleTop = h.visible[len(h.visible)-1]
	}
	if len(h.redo) > 0 {
		v.RedoTop = h.redo[len(h.redo)-1]
	}
	return v, true
}
func (s *SketchState) StrokeMetadata(id, stroke string) (SketchStrokeMetadata, bool) {
	h := s.sketches[id]
	if h == nil {
		return SketchStrokeMetadata{}, false
	}
	v := h.strokes[stroke]
	if v == nil {
		return SketchStrokeMetadata{}, false
	}
	at := v.Begin.At
	at.Classes = slices.Clone(at.Classes)
	at.Path = slices.Clone(at.Path)
	at.Stable = maps.Clone(at.Stable)
	if at.Box != nil {
		b := *at.Box
		at.Box = &b
	}
	visible := h.phase != "cleared" && (h.shown[stroke] || h.open == stroke && len(v.Pts) > 0)
	return SketchStrokeMetadata{ID: stroke, At: at, Width: v.Begin.Width, T: v.Begin.T, PointCount: len(v.Pts), NextChunk: v.NextChunk, Visible: visible, Sealed: v.Sealed, Truncated: v.Truncated}, true
}

// Change is read after acceptance at the writer boundary. It copies only the
// incoming chunk and the touched stroke's anchor, never prior point history.
type SketchBeforeChange struct{ Sketch, Stroke string }

func (s *SketchState) BeforeChange() SketchBeforeChange {
	out := SketchBeforeChange{Sketch: s.active}
	if h := s.sketches[s.active]; h != nil {
		out.Stroke = h.open
	}
	return out
}

func (s *SketchState) Change(r Record, before SketchBeforeChange) (SketchChange, bool) {
	if sketchMetadata(r) == nil {
		switch r.(type) {
		case *Seg, *Stop, *Ctl, *AnnotationClear:
		default:
			return SketchChange{}, false
		}
		if _, clear := r.(*AnnotationClear); clear {
			return SketchChange{Kind: r.Kind()}, true
		}
		if h := s.sketches[before.Sketch]; h != nil && s.active == before.Sketch && h.open == before.Stroke && h.phase == "drawing" {
			return SketchChange{}, false
		}
	}
	id := sketchID(r)
	if id == "" {
		id = before.Sketch
	}
	meta, ok := s.Metadata(id)
	if !ok {
		return SketchChange{}, false
	}
	out := SketchChange{Kind: r.Kind(), Sketch: meta}
	sid := ""
	if before.Sketch == id {
		sid = before.Stroke
	}
	switch v := r.(type) {
	case *SketchStrokeBegin:
		sid = v.ID
	case *SketchPoints:
		sid = v.ID
		out.Points = slices.Clone(v.Pts)
		for i := range out.Points {
			out.Points[i] = normalizeSketchPoint(out.Points[i])
		}
		out.Chunk = v.Chunk
		if stroke := s.sketches[id].strokes[v.ID]; stroke != nil {
			out.PointsOffset = len(stroke.Pts) - len(v.Pts)
		}
	case *SketchStrokeEnd:
		sid = v.ID
	case *SketchUndo:
		sid = v.ID
	case *SketchRedo:
		sid = v.ID
	}
	if sid != "" {
		stroke, exists := s.StrokeMetadata(id, sid)
		if !exists {
			return SketchChange{}, false
		}
		out.Stroke = &stroke
	}
	return out, true
}
