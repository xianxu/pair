package recording

import "fmt"

const (
	SketchHistoryLimit = 32 << 20
	// The three maximum generated terminal envelopes fit respectively in
	// 8, 4 and 4 KiB, including JSON HTML escaping and the JSONL newline.
	// Their sum is rounded to 16 KiB inside, never beyond, the history cap.
	SketchLabelReserve     = 8 << 10
	SketchEndReserve       = 4 << 10
	SketchInterruptReserve = 4 << 10
	SketchTerminalReserve  = SketchLabelReserve + SketchEndReserve + SketchInterruptReserve
)

type sketchReservation struct{ label, end, interrupt int }

func (r sketchReservation) total() int { return r.label + r.end + r.interrupt }

// SketchBudget is the ordered producer's private admission owner. The zero
// value is ready for a new recording. Bytes are exact encoded JSONL sizes,
// including the newline. Next only inspects the new record and its one entry.
type SketchBudget struct {
	used, reserved int
	entries        map[string]sketchReservation
}

// SketchBudgetDelta is a proposal. Commit it only after State accepts the
// same record, even when the subsequent sink outcome is unknown. A producer
// serializes Next/Commit; proposals must not be reused across admissions.
type SketchBudgetDelta struct {
	used, reserved      int
	id                  string
	entry               sketchReservation
	remove, clear, noop bool
}

func (b *SketchBudget) Used() int     { return b.used }
func (b *SketchBudget) Reserved() int { return b.reserved }

// SketchEncodedBytes counts only sketch history. Legacy DOM records can be
// large and need no extra encoding merely to consult the sketch byte budget.
func SketchEncodedBytes(r Record) (int, error) {
	if sketchMetadata(r) == nil {
		return 0, nil
	}
	line, err := Encode(r)
	if err != nil {
		return 0, err
	}
	return len(line) + 1, nil
}

func (b *SketchBudget) Next(r Record, encodedBytes int, duplicate bool) (SketchBudgetDelta, error) {
	d := SketchBudgetDelta{used: b.used, reserved: b.reserved}
	if duplicate {
		d.noop = true
		return d, nil
	}
	if r == nil {
		return d, fmt.Errorf("sketch budget: nil record")
	}
	switch r := r.(type) {
	case *AnnotationClear:
		d.clear = true
		d.reserved = 0
		return d, nil
	case *Stop:
		if r.Why == StopEnd {
			d.clear = true
			d.reserved = 0
			return d, nil
		}
	}
	if sketchMetadata(r) == nil {
		d.noop = true
		return d, nil
	}
	if encodedBytes < 0 || encodedBytes > SketchHistoryLimit {
		return d, fmt.Errorf("sketch budget: invalid encoded size")
	}
	var before sketchReservation
	switch r := r.(type) {
	case *SketchBegin:
		d.id = r.ID
	case *SketchStrokeBegin:
		d.id = r.Sketch
	case *SketchPoints:
		d.id = r.Sketch
	case *SketchStrokeEnd:
		d.id = r.Sketch
	case *SketchUndo:
		d.id = r.Sketch
	case *SketchRedo:
		d.id = r.Sketch
	case *SketchLabel:
		d.id = r.Sketch
	case *SketchInterrupt:
		d.id = r.Sketch
	default:
		return d, fmt.Errorf("sketch budget: unsupported sketch event")
	}
	before, exists := b.entries[d.id]
	d.entry = before
	switch r.(type) {
	case *SketchBegin:
		if exists {
			return d, fmt.Errorf("sketch budget: existing sketch")
		}
		d.entry = sketchReservation{SketchLabelReserve, SketchEndReserve, SketchInterruptReserve}
	case *SketchStrokeBegin:
		if !exists {
			return d, fmt.Errorf("sketch budget: missing sketch")
		}
		d.entry.end = SketchEndReserve
	case *SketchStrokeEnd:
		if !exists {
			return d, fmt.Errorf("sketch budget: missing sketch")
		}
		if before.end > 0 && encodedBytes > before.end {
			return d, fmt.Errorf("sketch end exceeds terminal allowance")
		}
		d.entry.end = 0
	case *SketchInterrupt:
		if !exists || before.interrupt == 0 || encodedBytes > before.interrupt {
			return d, fmt.Errorf("sketch interrupt exceeds terminal allowance")
		}
		d.entry.end, d.entry.interrupt = 0, 0
	case *SketchLabel:
		if !exists || encodedBytes > before.label {
			return d, fmt.Errorf("sketch label exceeds terminal allowance")
		}
		d.entry = sketchReservation{}
		d.remove = true
	default:
		if !exists {
			return d, fmt.Errorf("sketch budget: missing sketch")
		}
	}
	d.used += encodedBytes
	d.reserved += d.entry.total() - before.total()
	if d.used+d.reserved > SketchHistoryLimit {
		return d, fmt.Errorf("sketch history exceeds %d-byte cap including finalization reservations", SketchHistoryLimit)
	}
	// Implicit ctl/seg interruptions conservatively retain the full reserve:
	// only the reducer owns control transitions. Label/clear/end release it.
	return d, nil
}

func (b *SketchBudget) Commit(d SketchBudgetDelta) {
	if d.noop {
		return
	}
	b.used, b.reserved = d.used, d.reserved
	if d.clear {
		b.entries = nil
		return
	}
	if d.remove {
		delete(b.entries, d.id)
		return
	}
	if b.entries == nil {
		b.entries = make(map[string]sketchReservation)
	}
	b.entries[d.id] = d.entry
}

// RebuildSketchBudget reconstructs producer admission once, in stream order.
// It uses the same reducer for receipt reconciliation and validation; no
// history reconstruction occurs during subsequent chunk admission.
func RebuildSketchBudget(records []Record) (*SketchBudget, error) {
	b := &SketchBudget{}
	s := NewState()
	for _, r := range records {
		if r.Kind() == "sketch-snapshot" {
			return nil, fmt.Errorf("catch-up snapshot is not a persisted record")
		}
		duplicate, err := s.Sketches.Reconcile(r)
		if err != nil {
			return nil, err
		}
		if duplicate {
			continue
		}
		size, err := SketchEncodedBytes(r)
		if err != nil {
			return nil, err
		}
		d, err := b.Next(r, size, false)
		if err != nil {
			return nil, err
		}
		if err = s.Apply(r); err != nil {
			return nil, err
		}
		b.Commit(d)
	}
	return b, nil
}
