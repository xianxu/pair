package board

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"slices"
)

// Selection is a frozen set of open notes, across recorded documents.
type Selection struct {
	Document         int
	TabID            string
	Notes            []SelectedNote
	Sketches         []SelectedSketch
	SketchGeneration uint64
	SketchRemaining  int
}
type SelectedSketch = recording.SketchCandidate
type SelectedNote struct {
	ID       string
	Revision uint64
}

func (b *Board) Selection() (Selection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.selectionLocked()
}

func (b *Board) selectionLocked() (Selection, error) {
	if b.closed {
		return Selection{}, ErrClosed
	}
	if b.broken != nil {
		return Selection{}, ErrBroken
	}
	if b.st.Top == nil {
		return Selection{}, ErrNotLive
	}
	out := Selection{Document: b.st.DocSeg, TabID: b.st.Tab.ID, SketchGeneration: b.submissionGeneration, Sketches: b.st.Sketches.SubmissionCandidates()}
	for _, n := range b.st.Notes {
		if n.State == recording.NoteOpen {
			out.Notes = append(out.Notes, SelectedNote{n.ID, n.Revision})
		}
	}
	return out, nil
}

// FreezeSubmissionSelection must be called while the manager owns its preview
// state. The lock order is manager then Board; this operation performs no IO.
func (b *Board) FreezeSubmissionSelection(generation uint64, chosen []SelectedSketch) (Selection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, err := b.selectionLocked()
	if err != nil {
		return Selection{}, err
	}
	if generation != s.SketchGeneration || len(chosen) > 8 {
		return Selection{}, ErrStale
	}
	seen := map[string]bool{}
	for _, v := range chosen {
		if seen[v.ID] || !slices.Contains(s.Sketches, v) {
			return Selection{}, ErrStale
		}
		seen[v.ID] = true
	}
	s.Sketches = append([]SelectedSketch(nil), chosen...)
	return s, nil
}

func (b *Board) SubscribeSubmissionCandidates() (<-chan struct{}, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan struct{}, 1)
	if b.closed {
		close(ch)
		return ch, func() {}
	}
	if b.submissionSubs == nil {
		b.submissionSubs = map[chan struct{}]struct{}{}
	}
	b.submissionSubs[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.submissionSubs[ch]; ok {
			delete(b.submissionSubs, ch)
			close(ch)
		}
	}
}

func (b *Board) submissionChangedLocked() {
	b.submissionGeneration++
	for ch := range b.submissionSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// ResolveSelection waits for actual recorded outcomes. A changed/reopened
// note no longer belongs to this acknowledgment; navigation preserves ownership.
func (b *Board) ResolveSelection(ctx context.Context, s Selection) error {
	admission, err := b.AdmitAnnotation()
	if err != nil {
		return err
	}
	updates, stop := b.Subscribe()
	defer stop()
	sent := map[string]bool{}
	for {
		b.mu.Lock()
		if b.closed {
			b.mu.Unlock()
			return ErrClosed
		}
		if b.broken != nil {
			b.mu.Unlock()
			return ErrBroken
		}
		var pending []SelectedNote
		for _, n := range s.Notes {
			if b.st.NoteUnchanged(n.ID, n.Revision) {
				pending = append(pending, n)
			}
		}
		b.mu.Unlock()
		if len(pending) == 0 {
			return nil
		}
		for _, n := range pending {
			if sent[n.ID] {
				continue
			}
			rev := n.Revision
			err := admission.resolveSelected(&recording.NoteState{ID: n.ID, State: recording.NoteResolved, IfRevision: &rev, Who: &recording.Author{ID: Operator, Name: "Operator"}}, 0)
			if err != nil && !errors.Is(err, ErrStale) {
				return err
			}
			sent[n.ID] = true
		}
		select {
		case _, open := <-updates:
			if !open {
				return ErrClosed
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// resolveSelected shares the admission fence and command ordering with viewer
// mutations, while retaining the writer's conditional revision check.
func (a AnnotationAdmission) resolveSelected(r *recording.NoteState, doc int) error {
	b := a.board
	b.command.Lock()
	defer b.command.Unlock()
	b.mu.Lock()
	err := b.admissionLocked(a.permit)
	b.mu.Unlock()
	if err != nil {
		return err
	}
	return b.emit(r, doc)
}
