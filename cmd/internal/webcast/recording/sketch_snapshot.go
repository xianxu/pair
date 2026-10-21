package recording

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// SketchSnapshot is a catch-up bootstrap, never an events.jsonl kind. Its
// authority is granted only by NewCatchUpState, not by ordinary disk decoding.
type SketchSnapshot struct {
	Header
	Control  Control          `json:"control"`
	Sketches SketchProjection `json:"sketches"`
}

func (*SketchSnapshot) Kind() string { return "sketch-snapshot" }
func (r *SketchSnapshot) check() error {
	if r.T < 0 {
		return errors.New("negative snapshot time")
	}
	return nil
}
func NewCatchUpState() *State { s := NewState(); s.allowSketchSnapshot = true; return s }
func ReduceCatchUp(recs []Record, t int64) (*State, error) {
	s := NewCatchUpState()
	for i, r := range recs {
		if r.Time() > t {
			break
		}
		if err := s.Apply(r); err != nil {
			return s, fmt.Errorf("catch-up record %d: %w", i, err)
		}
	}
	return s, nil
}
func (s *State) SketchSnapshot() *SketchSnapshot {
	return &SketchSnapshot{Header: Header{T: s.T}, Control: cloneSketchControl(s.Control), Sketches: s.Sketches.Snapshot()}
}
func cloneSketchControl(c Control) Control {
	out := Control{Pending: slices.Clone(c.Pending)}
	if c.Driver != nil {
		d := *c.Driver
		out.Driver = &d
	}
	return out
}
func validateSnapshotControl(c Control) error {
	seen := map[string]bool{}
	if c.Driver != nil {
		if c.Driver.ID == "" {
			return errors.New("snapshot driver has no identity")
		}
		seen[c.Driver.ID] = true
	}
	for _, p := range c.Pending {
		if p.ID == "" || seen[p.ID] {
			return errors.New("invalid snapshot control identities")
		}
		seen[p.ID] = true
	}
	return nil
}

// snapshotReceiptRecord checks the semantic receipt through the actual codec,
// not a second permissive JSON schema. Writer-assigned times are irrelevant.
func snapshotReceiptRecord(receipt SketchReceipt) (Record, error) {
	kind, body, ok := strings.Cut(receipt.Semantic, ":")
	if !ok {
		return nil, errors.New("invalid snapshot receipt semantic")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("invalid snapshot receipt object")
	}
	m["t"] = json.RawMessage("0")
	m["k"], _ = json.Marshal(kind)
	m["digest"], _ = json.Marshal(receipt.Digest)
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	r, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	o := sketchMetadata(r)
	if o == nil || o.Operation != receipt.Operation || o.Sequence != receipt.Sequence {
		return nil, errors.New("invalid snapshot receipt identity")
	}
	semantic, err := SketchSemantic(r)
	if err != nil || semantic != receipt.Semantic {
		return nil, errors.New("noncanonical snapshot receipt")
	}
	return r, nil
}
func restoreSketchState(p SketchProjection) (SketchState, error) {
	fail := func(msg string) (SketchState, error) { return SketchState{}, errors.New(msg) }
	if p.HistoryBytes < 0 || p.HistoryBytes > SketchHistoryLimit || len(p.Sketches) > SketchMaxCount {
		return fail("snapshot history bound")
	}
	out := SketchState{active: p.Active, historyBytes: p.HistoryBytes, sketches: map[string]*sketchHistory{}}
	semanticBytes := 0
	activeCount := 0
	for _, v := range p.Sketches {
		if v.Begin == nil || v.Begin.check() != nil {
			return fail("invalid snapshot begin")
		}
		id := v.Begin.ID
		if out.sketches[id] != nil || v.NextSequence == 0 || v.NextSequence > MaxSketchInteger || uint64(len(v.Receipts)) != v.NextSequence || len(v.Strokes) > SketchMaxStrokes {
			return fail("invalid snapshot history identities")
		}
		if !oneOf(v.Phase, "drawing", "complete", "interrupted", "cleared") {
			return fail("invalid snapshot phase")
		}
		if v.Reason != "" && !oneOf(v.Reason, SketchInterruptExpiry, SketchInterruptNavigation, SketchInterruptHandoff, SketchInterruptReset, SketchInterruptEnd) {
			return fail("invalid snapshot interruption")
		}
		if v.Phase == "drawing" && (v.Label != "" || v.Reason != "") {
			return fail("invalid drawing metadata")
		}
		if v.Phase == "complete" && (v.Label == "" || v.Reason != "") {
			return fail("invalid completion metadata")
		}
		if v.Phase == "interrupted" && v.Reason == "" {
			return fail("missing interruption evidence")
		}
		if v.Label != "" {
			label := &SketchLabel{SketchOperation: v.Begin.SketchOperation, Sketch: id, Label: v.Label}
			if label.check() != nil {
				return fail("invalid snapshot label")
			}
		}
		begin, err := cloneSketchBegin(v.Begin)
		if err != nil {
			return SketchState{}, err
		}
		h := &sketchHistory{begin: begin, phase: v.Phase, reason: v.Reason, label: v.Label, strokes: map[string]*SketchStrokeView{}, visible: slices.Clone(v.Visible), redo: slices.Clone(v.Redo), next: v.NextSequence, receipts: map[string]SketchReceipt{}}
		strokeSemantics := map[string]string{}
		strokeDigests := map[string]string{}
		pointWitness := map[string][]SketchPoint{}
		chunkWitness := map[string]uint64{}
		labelWitness := ""
		for i, receipt := range v.Receipts {
			if receipt.Sequence != uint64(i) || !validSketchID(receipt.Operation) || !ValidHash(receipt.Digest) {
				return fail("invalid snapshot receipt sequence")
			}
			if _, ok := h.receipts[receipt.Operation]; ok {
				return fail("duplicate snapshot operation")
			}
			if len(receipt.Semantic) > p.HistoryBytes-semanticBytes {
				return fail("snapshot receipt byte bound")
			}
			r, err := snapshotReceiptRecord(receipt)
			if err != nil {
				return SketchState{}, err
			}
			if sketchID(r) != id {
				return fail("snapshot receipt belongs to another sketch")
			}
			if i == 0 {
				if _, ok := r.(*SketchBegin); !ok {
					return fail("snapshot first receipt is not begin")
				}
				semantic, _ := SketchSemantic(begin)
				if semantic != receipt.Semantic || begin.Digest != receipt.Digest {
					return fail("snapshot begin differs from receipt")
				}
			} else if _, ok := r.(*SketchBegin); ok {
				return fail("snapshot second begin")
			}
			if stroke, ok := r.(*SketchStrokeBegin); ok {
				if _, exists := strokeSemantics[stroke.ID]; exists {
					return fail("duplicate snapshot stroke begin")
				}
				strokeSemantics[stroke.ID] = receipt.Semantic
				strokeDigests[stroke.ID] = receipt.Digest
			}
			switch event := r.(type) {
			case *SketchPoints:
				if _, exists := strokeSemantics[event.ID]; !exists || event.Chunk != chunkWitness[event.ID] {
					return fail("snapshot point receipt out of order")
				}
				pointWitness[event.ID] = append(pointWitness[event.ID], event.Pts...)
				chunkWitness[event.ID]++
				if len(pointWitness[event.ID]) > SketchMaxPointsPerStroke {
					return fail("snapshot point receipt bound")
				}
			case *SketchLabel:
				if labelWitness != "" {
					return fail("snapshot repeated label receipt")
				}
				labelWitness = event.Label
			}
			h.receipts[receipt.Operation] = receipt
			h.receiptOrder = append(h.receiptOrder, receipt.Operation)
			semanticBytes += len(receipt.Semantic)
			if semanticBytes > p.HistoryBytes {
				return fail("snapshot receipt byte bound")
			}
		}
		if labelWitness != v.Label {
			return fail("snapshot label differs from receipt")
		}
		for _, view := range v.Strokes {
			if view.Begin == nil || view.Begin.check() != nil || view.Begin.Sketch != id || h.strokes[view.Begin.ID] != nil || len(view.Pts) > SketchMaxPointsPerStroke || view.NextChunk > MaxSketchInteger {
				return fail("invalid snapshot stroke")
			}
			if view.NextChunk != chunkWitness[view.Begin.ID] || !slices.Equal(view.Pts, pointWitness[view.Begin.ID]) {
				return fail("snapshot points differ from receipts")
			}
			semantic, _ := SketchSemantic(view.Begin)
			if strokeSemantics[view.Begin.ID] != semantic || strokeDigests[view.Begin.ID] != view.Begin.Digest {
				return fail("snapshot stroke differs from receipt")
			}
			if view.NextChunk > uint64(len(view.Pts)) || uint64(len(view.Pts)) > view.NextChunk*SketchMaxPointsPerChunk || view.Truncated && !view.Sealed {
				return fail("invalid snapshot chunks")
			}
			for _, point := range view.Pts {
				if !sketchFinite(point[0]) || !sketchFinite(point[1]) || !sketchFinite(point[2]) || point[2] < 0 || point[2] > 1 {
					return fail("invalid snapshot point")
				}
			}
			if !view.Sealed {
				if h.open != "" || v.Phase != "drawing" {
					return fail("invalid snapshot open stroke")
				}
				h.open = view.Begin.ID
			}
			owned := view
			owned.Begin, err = cloneSketchStroke(view.Begin)
			if err != nil {
				return SketchState{}, err
			}
			owned.Pts = append([]SketchPoint{}, view.Pts...)
			h.strokes[view.Begin.ID] = &owned
			h.order = append(h.order, view.Begin.ID)
			h.points += len(view.Pts)
		}
		if len(strokeSemantics) != len(h.strokes) || h.points > SketchMaxPointsPerSketch {
			return fail("snapshot stroke/point bound")
		}
		seen := map[string]bool{}
		for _, stack := range [][]string{h.visible, h.redo} {
			for _, sid := range stack {
				stroke := h.strokes[sid]
				if stroke == nil || !stroke.Sealed || len(stroke.Pts) == 0 || seen[sid] {
					return fail("invalid snapshot stack")
				}
				seen[sid] = true
			}
		}
		if v.Phase == "drawing" {
			activeCount++
			if p.Active != id {
				return fail("snapshot active mismatch")
			}
		}
		if err := validateSketchViewHistory(v); err != nil {
			return SketchState{}, err
		}
		out.sketches[id] = h
		out.order = append(out.order, id)
	}
	if p.Active != "" && activeCount != 1 || p.Active == "" && activeCount != 0 {
		return fail("snapshot active identity missing")
	}
	return out, nil
}
func (s *State) installSketchSnapshot(r *SketchSnapshot) error {
	if !s.allowSketchSnapshot {
		return errors.New("sketch snapshot is catch-up only")
	}
	if err := validateSnapshotControl(r.Control); err != nil {
		return err
	}
	next, err := restoreSketchState(r.Sketches)
	if err != nil {
		return err
	}
	if s.Ended && next.active != "" {
		return errors.New("ended snapshot still drawing")
	}
	if next.active != "" {
		active := next.sketches[next.active]
		driver := r.Control.Driving()
		if driver == "" {
			driver = OperatorID
		}
		if active.begin.Author.ID != driver {
			return errors.New("snapshot active author is not driving")
		}
		if active.begin.Doc != s.DocSeg || active.begin.Tab != s.Tab.ID {
			return errors.New("snapshot active source differs")
		}
	}
	s.Sketches = next
	s.Control = cloneSketchControl(r.Control)
	s.ctlSeen = true
	return nil
}

// validateSketchViewHistory reuses the transition owner for operation ordering
// and stack authority. Contextual seals/interruptions are snapshot facts: their
// original ctl/seg/stop records intentionally do not survive live compaction.
func validateSketchViewHistory(view SketchView) error {
	var check SketchState
	strokes := map[string]SketchStrokeView{}
	for _, stroke := range view.Strokes {
		strokes[stroke.Begin.ID] = stroke
	}
	driver := view.Begin.Author.ID
	if driver == OperatorID {
		driver = ""
	}
	context := SketchContext{Live: true, Doc: view.Begin.Doc, Tab: view.Begin.Tab, Driver: driver}
	for _, receipt := range view.Receipts {
		event, err := snapshotReceiptRecord(receipt)
		if err != nil {
			return err
		}
		history := check.sketches[view.Begin.ID]
		if history != nil {
			needsClosed := false
			switch event := event.(type) {
			case *SketchStrokeBegin, *SketchUndo, *SketchRedo, *SketchLabel:
				needsClosed = true
			case *SketchStrokeEnd:
				needsClosed = event.ID == history.open
			}
			if needsClosed && history.open != "" && strokes[history.open].Truncated {
				check.pause()
			}
			if _, label := event.(*SketchLabel); label && view.Reason != "" && history.phase == "drawing" {
				check.interrupt(view.Reason)
			}
		}
		delta, err := NextSketch(&check, event, context)
		if err != nil {
			return fmt.Errorf("snapshot receipt history: %w", err)
		}
		delta.commit()
	}
	history := check.sketches[view.Begin.ID]
	if history == nil {
		return errors.New("snapshot has no receipt history")
	}
	if history.open != "" && strokes[history.open].Sealed && strokes[history.open].Truncated {
		check.pause()
	}
	if history.phase == "drawing" && view.Reason != "" {
		check.interrupt(view.Reason)
	}
	if view.Phase == "cleared" {
		check.clear()
	}
	if history.phase != view.Phase || history.reason != view.Reason || history.label != view.Label || !slices.Equal(history.visible, view.Visible) || !slices.Equal(history.redo, view.Redo) {
		return errors.New("snapshot projection differs from receipt history")
	}
	for id, stroke := range history.strokes {
		projected := strokes[id]
		if projected.Sealed != stroke.Sealed || projected.Truncated != stroke.Truncated {
			return errors.New("snapshot seal differs from receipt history")
		}
	}
	return nil
}
