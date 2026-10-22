package board

import (
	"encoding/json"
	"errors"
	"math"
	"slices"

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

// matchDeliveredFeedback retains the highest Store-confirmed prefix at or below
// each retained identity. Equal sketch sequences require the current digest;
// older watermarks protect catch-up frames without hiding newer current ink.
// Two current passes and one delivered pass keep work O(C+D), with owned scalars.
func matchDeliveredFeedback(current []SelectedSketch, notes []SelectedNote, delivered []SelectedSketch, deliveredNotes []SelectedNote) (LiveFeedbackMask, int) {
	visits := 0
	currentSketches := make(map[string]SelectedSketch, len(current))
	for _, v := range current {
		visits++
		currentSketches[v.ID] = v
	}
	currentNotes := make(map[string]uint64, len(notes))
	for _, v := range notes {
		visits++
		currentNotes[v.ID] = v.Revision
	}
	sketches := make(map[string]FeedbackSketchVersion)
	for _, v := range delivered {
		visits++
		latest, ok := currentSketches[v.ID]
		if !ok || v.NextSequence == 0 || v.NextSequence > latest.NextSequence || (v.NextSequence == latest.NextSequence && v.Digest != latest.Digest) {
			continue
		}
		best, found := sketches[v.ID]
		if !found || v.NextSequence > best.NextSequence || (v.NextSequence == best.NextSequence && v.Digest < best.Digest) {
			sketches[v.ID] = FeedbackSketchVersion{ID: v.ID, NextSequence: v.NextSequence, Digest: v.Digest}
		}
	}
	noteVersions := make(map[string]uint64)
	for _, v := range deliveredNotes {
		visits++
		latest, ok := currentNotes[v.ID]
		if !ok || v.Revision > latest {
			continue
		}
		best, found := noteVersions[v.ID]
		if !found || v.Revision > best {
			noteVersions[v.ID] = v.Revision
		}
	}
	out := LiveFeedbackMask{Sketches: []FeedbackSketchVersion{}, Notes: []FeedbackNoteVersion{}}
	for _, v := range current {
		visits++
		if best, ok := sketches[v.ID]; ok {
			out.Sketches = append(out.Sketches, best)
		}
	}
	for _, v := range notes {
		visits++
		if best, ok := noteVersions[v.ID]; ok {
			out.Notes = append(out.Notes, FeedbackNoteVersion{ID: v.ID, Revision: best})
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
		notes = append(notes, SelectedNote{ID: n.ID, Revision: n.Revision})
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

// Retained IDs keep their confirmed prefix across edits, resolution and newer
// sketch versions. AnnotationClear is the reducer operation that removes them;
// normal pressure/DOM/note events do no mask matching or historical copying.
func (b *Board) pruneFeedbackMaskLocked(r recording.Record) {
	if _, ok := r.(*recording.AnnotationClear); ok {
		b.installFeedbackMaskLocked(LiveFeedbackMask{Sketches: []FeedbackSketchVersion{}, Notes: []FeedbackNoteVersion{}})
	}
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
