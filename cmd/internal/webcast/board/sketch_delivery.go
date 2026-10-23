package board

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const (
	SketchDeliveryJournalEntries = 256
	SketchDeliveryJournalBytes   = 1 << 20
	SketchDeliveryBatchEntries   = 32
	SketchDeliveryBatchBytes     = 64 << 10
	SketchDeliveryBootstrapBytes = 36 << 20
)

var ErrSketchDeliveryLimit = errors.New("board: sketch delivery exceeds bound")

type SketchDeliveryCursor struct {
	Epoch string `json:"epoch"`
	Seq   uint64 `json:"seq"`
}
type SketchDeliveryBatch struct {
	Epoch   string            `json:"epoch"`
	From    uint64            `json:"from"`
	To      uint64            `json:"to"`
	Patches []json.RawMessage `json:"patches"`
}
type SketchDisplayBegin struct {
	ID     string             `json:"id"`
	Author recording.Author   `json:"author"`
	Doc    int                `json:"doc"`
	Tab    string             `json:"tab"`
	VP     recording.Viewport `json:"vp"`
	T      int64              `json:"t"`
}
type SketchDisplayStrokeBegin struct {
	ID    string           `json:"id"`
	At    recording.NoteAt `json:"at"`
	Width float64          `json:"width"`
}
type SketchDisplayStroke struct {
	Begin     SketchDisplayStrokeBegin `json:"begin"`
	Pts       []recording.SketchPoint  `json:"pts"`
	Visible   bool                     `json:"visible"`
	Sealed    bool                     `json:"sealed"`
	Truncated bool                     `json:"truncated"`
}
type SketchDisplaySketch struct {
	NextSequence uint64                `json:"next_sequence,string,omitempty"`
	Digest       string                `json:"digest,omitempty"`
	Begin        SketchDisplayBegin    `json:"begin"`
	Phase        string                `json:"phase"`
	Reason       string                `json:"reason"`
	Label        string                `json:"label"`
	Strokes      []SketchDisplayStroke `json:"strokes"`
}
type SketchDisplay struct {
	Sketches []SketchDisplaySketch `json:"sketches"`
}
type SketchDeliveryBootstrap struct {
	Epoch   string        `json:"epoch"`
	Seq     uint64        `json:"seq"`
	Display SketchDisplay `json:"display"`
}
type SketchDelivery struct {
	Batch     *SketchDeliveryBatch     `json:"batch,omitempty"`
	Bootstrap *SketchDeliveryBootstrap `json:"bootstrap,omitempty"`
}
type SketchDeliveryStats struct{ Patches, Points, EncodedBytes, Bootstraps uint64 }
type sketchJournalEntry struct {
	seq uint64
	raw json.RawMessage
}
type sketchDeliveryJournal struct {
	epoch   string
	seq     uint64
	entries []sketchJournalEntry
	bytes   int
	stats   SketchDeliveryStats
	subs    map[chan struct{}]struct{}
	closed  bool
}
type sketchDeliveryBefore struct {
	core      recording.SketchBeforeChange
	duplicate bool
}

func (b *Board) sketchBeforeChangeLocked(r recording.Record) sketchDeliveryBefore {
	duplicate, _ := b.st.Sketches.Reconcile(r)
	return sketchDeliveryBefore{core: b.st.Sketches.BeforeChange(), duplicate: duplicate}
}
func displayBegin(m recording.SketchMetadata) SketchDisplayBegin {
	return SketchDisplayBegin{ID: m.ID, Author: m.Author, Doc: m.Doc, Tab: m.Tab, VP: m.VP, T: m.T}
}
func (b *Board) refreshSketchDeliveryLocked(r recording.Record, before sketchDeliveryBefore) {
	if before.duplicate {
		return
	}
	change, ok := b.st.Sketches.Change(r, before.core)
	if !ok {
		return
	}
	patch := map[string]any{}
	if change.Kind == "annotation-clear" {
		patch["clear"] = true
	} else {
		patch["sketch"] = change.Sketch.ID
		switch r.(type) {
		case *recording.SketchBegin:
			patch["header"] = map[string]any{"begin": displayBegin(change.Sketch), "phase": change.Sketch.Phase, "reason": change.Sketch.Reason, "label": change.Sketch.Label}
		case *recording.SketchLabel, *recording.SketchInterrupt, *recording.Seg, *recording.Stop, *recording.Ctl:
			patch["header"] = map[string]any{"phase": change.Sketch.Phase, "reason": change.Sketch.Reason, "label": change.Sketch.Label}
		}
		header, ok := patch["header"].(map[string]any)
		if !ok {
			header = map[string]any{}
			patch["header"] = header
		}
		header["next_sequence"] = strconv.FormatUint(change.Sketch.NextSequence, 10)
		header["digest"] = change.Sketch.Digest
		if v := change.Stroke; v != nil {
			flags := map[string]any{"id": v.ID, "visible": v.Visible, "sealed": v.Sealed, "truncated": v.Truncated}
			if _, begin := r.(*recording.SketchStrokeBegin); begin {
				delete(flags, "id")
				flags["begin"] = SketchDisplayStrokeBegin{ID: v.ID, At: v.At, Width: v.Width}
			}
			patch["stroke"] = flags
			if len(change.Points) > 0 {
				patch["points"] = map[string]any{"id": v.ID, "offset": change.PointsOffset, "pts": change.Points}
			}
		}
	}
	raw, err := json.Marshal(patch)
	if err != nil {
		b.broken = err
		return
	}
	b.sketchDelivery.append(raw, uint64(len(change.Points)))
}
func (j *sketchDeliveryJournal) init() {
	if j.epoch == "" {
		j.epoch = randomID()
	}
}
func (j *sketchDeliveryJournal) append(raw json.RawMessage, points uint64) {
	j.init()
	if j.closed {
		return
	}
	if j.seq == recording.MaxSketchInteger {
		j.epoch = randomID()
		j.seq = 0
		j.entries = nil
		j.bytes = 0
	}
	j.seq++
	j.stats.Patches++
	j.stats.Points += points
	j.stats.EncodedBytes += uint64(len(raw))
	// A patch larger than a legal batch cannot be delivered incrementally.
	// Advance the high-water and retire continuity; the next read bootstraps.
	probe := SketchDeliveryBatch{Epoch: j.epoch, From: j.seq, To: j.seq, Patches: []json.RawMessage{}}
	encoded, err := json.Marshal(probe)
	if err != nil || len(encoded)+len(raw) > SketchDeliveryBatchBytes || len(raw) > SketchDeliveryJournalBytes {
		j.entries = nil
		j.bytes = 0
	} else {
		j.entries = append(j.entries, sketchJournalEntry{seq: j.seq, raw: raw})
		j.bytes += len(raw)
		for len(j.entries) > SketchDeliveryJournalEntries || j.bytes > SketchDeliveryJournalBytes {
			j.bytes -= len(j.entries[0].raw)
			j.entries[0] = sketchJournalEntry{}
			j.entries = j.entries[1:]
		}
	}
	for ch := range j.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
func (b *Board) sketchDeliveryCursorLocked() SketchDeliveryCursor {
	b.sketchDelivery.init()
	return SketchDeliveryCursor{Epoch: b.sketchDelivery.epoch, Seq: b.sketchDelivery.seq}
}
func (b *Board) OperatorSketchCursor() SketchDeliveryCursor {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sketchDeliveryCursorLocked()
}
func (b *Board) OperatorSketchDeliveryStats() SketchDeliveryStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sketchDelivery.stats
}

// OperatorSketchSince takes a bootstrap and its exact high-water under the
// same Board lock. Normal batches copy only retained encoded patches.
func (b *Board) OperatorSketchSince(cursor SketchDeliveryCursor) (SketchDelivery, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.broken != nil {
		return SketchDelivery{}, ErrBroken
	}
	current := b.sketchDeliveryCursorLocked()
	j := &b.sketchDelivery
	gap := cursor.Epoch != current.Epoch || cursor.Seq > current.Seq
	if !gap && cursor.Seq < current.Seq {
		gap = len(j.entries) == 0 || cursor.Seq+1 < j.entries[0].seq
	}
	if gap {
		frame := &SketchDeliveryBootstrap{Epoch: current.Epoch, Seq: current.Seq, Display: sketchDisplayProjection(b.st.Sketches.Snapshot())}
		encoded, err := json.Marshal(frame)
		if err != nil {
			return SketchDelivery{}, err
		}
		if len(encoded) > SketchDeliveryBootstrapBytes {
			return SketchDelivery{}, ErrSketchDeliveryLimit
		}
		j.stats.Bootstraps++
		return SketchDelivery{Bootstrap: frame}, nil
	}
	if cursor.Seq == current.Seq {
		return SketchDelivery{}, nil
	}
	frame := &SketchDeliveryBatch{Epoch: current.Epoch, From: cursor.Seq + 1, To: cursor.Seq, Patches: []json.RawMessage{}}
	size := 0
	for _, entry := range j.entries {
		if entry.seq <= cursor.Seq {
			continue
		}
		if len(frame.Patches) >= SketchDeliveryBatchEntries {
			break
		}
		// Encode the small envelope only; retained patch bytes are never encoded
		// again while deciding the batch boundary.
		envelope, _ := json.Marshal(SketchDeliveryBatch{Epoch: frame.Epoch, From: frame.From, To: entry.seq, Patches: []json.RawMessage{}})
		commas := len(frame.Patches)
		if len(envelope)+size+len(entry.raw)+commas > SketchDeliveryBatchBytes {
			break
		}
		frame.Patches = append(frame.Patches, bytes.Clone(entry.raw))
		frame.To = entry.seq
		size += len(entry.raw)
	}
	if len(frame.Patches) == 0 {
		return SketchDelivery{}, ErrSketchDeliveryLimit
	}
	return SketchDelivery{Batch: frame}, nil
}
func sketchDisplayProjection(projection recording.SketchProjection) SketchDisplay {
	out := SketchDisplay{Sketches: []SketchDisplaySketch{}}
	for _, s := range projection.Sketches {
		if s.Phase == "cleared" {
			continue
		}
		b := s.Begin
		v := SketchDisplaySketch{Begin: SketchDisplayBegin{ID: b.ID, Author: b.Author, Doc: b.Doc, Tab: b.Tab, VP: b.VP, T: b.T}, Phase: s.Phase, Reason: s.Reason, Label: s.Label, Strokes: []SketchDisplayStroke{}}
		v.NextSequence = s.NextSequence
		if len(s.Receipts) > 0 {
			v.Digest = s.Receipts[len(s.Receipts)-1].Digest
		}
		visible := map[string]bool{}
		for _, id := range s.Visible {
			visible[id] = true
		}
		for _, p := range s.Strokes {
			v.Strokes = append(v.Strokes, SketchDisplayStroke{Begin: SketchDisplayStrokeBegin{ID: p.Begin.ID, At: p.Begin.At, Width: p.Begin.Width}, Pts: p.Pts, Visible: visible[p.Begin.ID] || s.Phase == "drawing" && !p.Sealed && len(p.Pts) > 0, Sealed: p.Sealed, Truncated: p.Truncated})
		}
		out.Sketches = append(out.Sketches, v)
	}
	return out
}
func (b *Board) SubscribeSketchDelivery() (<-chan struct{}, func()) {
	b.mu.Lock()
	j := &b.sketchDelivery
	j.init()
	ch := make(chan struct{}, 1)
	if j.closed {
		close(ch)
	} else {
		if j.subs == nil {
			j.subs = map[chan struct{}]struct{}{}
		}
		j.subs[ch] = struct{}{}
		ch <- struct{}{}
	}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := j.subs[ch]; ok {
			delete(j.subs, ch)
			close(ch)
		}
	}
}
func (b *Board) closeSketchDeliveryLocked() {
	j := &b.sketchDelivery
	if j.closed {
		return
	}
	j.closed = true
	for ch := range j.subs {
		close(ch)
	}
	j.subs = nil
	j.entries = nil
	j.bytes = 0
}
