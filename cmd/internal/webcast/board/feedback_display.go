package board

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const MaxFeedbackMaskBytes = 64 << 20

var ErrFeedbackMaskLimit = errors.New("board: feedback mask exceeds wire budget")

// Reserve the longest revision envelope up front. Pruning only removes entries,
// so later revision growth cannot make an admitted full mask exceed its budget.
func feedbackMaskBudget(mask LiveFeedbackMask, limit int) error {
	mask.Revision = math.MaxUint64
	data, err := json.Marshal(mask)
	if err != nil {
		return err
	}
	if len(data) > limit {
		return ErrFeedbackMaskLimit
	}
	return nil
}

type FeedbackSketchVersion struct {
	ID           string `json:"id"`
	NextSequence uint64 `json:"next_sequence,string"`
	Digest       string `json:"digest"`
}
type FeedbackNoteVersion struct {
	ID       string `json:"id"`
	Revision uint64 `json:"revision,string"`
}
type LiveFeedbackMask struct {
	Scope    string                  `json:"scope"`
	Revision uint64                  `json:"revision,string"`
	Sketches []FeedbackSketchVersion `json:"sketches"`
	Notes    []FeedbackNoteVersion   `json:"notes"`
}
type feedbackDisplay struct {
	mask     LiveFeedbackMask
	sketches map[string]FeedbackSketchVersion
	notes    map[string]uint64
	subs     map[chan LiveFeedbackMask]struct{}
}

func cloneFeedbackMask(m LiveFeedbackMask) LiveFeedbackMask {
	m.Sketches = append([]FeedbackSketchVersion{}, m.Sketches...)
	m.Notes = append([]FeedbackNoteVersion{}, m.Notes...)
	return m
}

// matchDeliveredFeedback intersects exact versions in authoritative creation
// order. Count every input/current visit; no points or receipts are inspected.
func matchDeliveredFeedback(current []SelectedSketch, notes []SelectedNote, delivered []SelectedSketch, deliveredNotes []SelectedNote) (LiveFeedbackMask, int) {
	visits := 0
	sketchSet := make(map[FeedbackSketchVersion]struct{}, len(delivered))
	for _, s := range delivered {
		visits++
		sketchSet[FeedbackSketchVersion{ID: s.ID, NextSequence: s.NextSequence, Digest: s.Digest}] = struct{}{}
	}
	noteSet := make(map[SelectedNote]struct{}, len(deliveredNotes))
	for _, n := range deliveredNotes {
		visits++
		noteSet[n] = struct{}{}
	}
	out := LiveFeedbackMask{Sketches: []FeedbackSketchVersion{}, Notes: []FeedbackNoteVersion{}}
	for _, s := range current {
		visits++
		v := FeedbackSketchVersion{ID: s.ID, NextSequence: s.NextSequence, Digest: s.Digest}
		if _, ok := sketchSet[v]; ok {
			out.Sketches = append(out.Sketches, v)
		}
	}
	for _, n := range notes {
		visits++
		if _, ok := noteSet[n]; ok {
			out.Notes = append(out.Notes, FeedbackNoteVersion{ID: n.ID, Revision: n.Revision})
		}
	}
	return out, visits
}

func (b *Board) currentFeedbackVersionsLocked() ([]SelectedSketch, []SelectedNote) {
	var sketches []SelectedSketch
	for _, v := range b.st.Sketches.SubmissionCandidates() {
		sketches = append(sketches, SelectedSketch{ID: v.ID, NextSequence: v.NextSequence, Digest: v.Digest})
	}
	var notes []SelectedNote
	for _, n := range b.st.Notes {
		if n.State == recording.NoteOpen {
			notes = append(notes, SelectedNote{ID: n.ID, Revision: n.Revision})
		}
	}
	return sketches, notes
}

// SetDeliveredFeedback projects Store-confirmed delivery into this Board's
// display lifetime. It never writes or resolves recorded annotations.
func (b *Board) SetDeliveredFeedback(sketches []SelectedSketch, notes []SelectedNote) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrClosed
	}
	if b.broken != nil {
		return ErrBroken
	}
	return b.setDeliveredFeedbackLocked(sketches, notes, MaxFeedbackMaskBytes)
}

func (b *Board) setDeliveredFeedbackLocked(sketches []SelectedSketch, notes []SelectedNote, limit int) error {
	current, currentNotes := b.currentFeedbackVersionsLocked()
	mask, _ := matchDeliveredFeedback(current, currentNotes, sketches, notes)
	mask.Scope = b.feedbackDisplay.mask.Scope
	if err := feedbackMaskBudget(mask, limit); err != nil {
		return err
	}
	if b.installFeedbackMaskLocked(mask) {
		b.submissionChangedLocked()
	}
	return nil
}

func (b *Board) installFeedbackMaskLocked(mask LiveFeedbackMask) bool {
	d := &b.feedbackDisplay
	if slices.Equal(d.mask.Sketches, mask.Sketches) && slices.Equal(d.mask.Notes, mask.Notes) {
		return false
	}
	mask.Scope = d.mask.Scope
	mask.Revision = d.mask.Revision + 1
	d.mask = mask
	d.sketches = make(map[string]FeedbackSketchVersion, len(mask.Sketches))
	for _, v := range mask.Sketches {
		d.sketches[v.ID] = v
	}
	d.notes = make(map[string]uint64, len(mask.Notes))
	for _, v := range mask.Notes {
		d.notes[v.ID] = v.Revision
	}
	for ch := range d.subs {
		select {
		case <-ch:
		default:
		}
		ch <- cloneFeedbackMask(mask)
	}
	b.refreshOperatorLocked(nil)
	return true
}

func (b *Board) noteFeedbackHiddenLocked(id string, revision uint64) bool {
	v, ok := b.feedbackDisplay.notes[id]
	return ok && v == revision
}

// Most ordered records take no mask work. A changed masked identity is pruned
// once; subsequent pressure chunks take only the constant-size map lookup.
func (b *Board) pruneFeedbackMaskLocked(r recording.Record) {
	d := &b.feedbackDisplay
	prune := false
	if key := recording.SketchReceiptKey(r); key != "" {
		id, _, _ := strings.Cut(key, "\x00")
		if old, ok := d.sketches[id]; ok {
			v, found := b.st.Sketches.Metadata(id)
			prune = !found || v.NextSequence != old.NextSequence || v.Digest != old.Digest
		}
	} else {
		switch v := r.(type) {
		case *recording.NoteEdit:
			_, prune = d.notes[v.ID]
		case *recording.NoteState:
			_, prune = d.notes[v.ID]
		case *recording.AnnotationClear:
			prune = len(d.mask.Sketches) > 0 || len(d.mask.Notes) > 0
		}
	}
	if !prune {
		return
	}
	var sketches []SelectedSketch
	for _, v := range d.mask.Sketches {
		sketches = append(sketches, SelectedSketch{ID: v.ID, NextSequence: v.NextSequence, Digest: v.Digest})
	}
	var notes []SelectedNote
	for _, v := range d.mask.Notes {
		notes = append(notes, SelectedNote{ID: v.ID, Revision: v.Revision})
	}
	current, currentNotes := b.currentFeedbackVersionsLocked()
	mask, _ := matchDeliveredFeedback(current, currentNotes, sketches, notes)
	b.installFeedbackMaskLocked(mask)
}

func (b *Board) FeedbackMask() LiveFeedbackMask {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneFeedbackMask(b.feedbackDisplay.mask)
}
func (b *Board) SubscribeFeedbackMask() (<-chan LiveFeedbackMask, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan LiveFeedbackMask, 1)
	ch <- cloneFeedbackMask(b.feedbackDisplay.mask)
	if b.closed {
		close(ch)
		return ch, func() {}
	}
	if b.feedbackDisplay.subs == nil {
		b.feedbackDisplay.subs = map[chan LiveFeedbackMask]struct{}{}
	}
	b.feedbackDisplay.subs[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.feedbackDisplay.subs[ch]; ok {
			delete(b.feedbackDisplay.subs, ch)
			close(ch)
		}
	}
}
