package board

import (
	"context"
	"errors"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"maps"
	"math"
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
	NoteRemaining    int
	CurrentPage      *sketchimage.PageScene
	Current          *sketchimage.CurrentView
}
type SelectedSketch struct {
	ID             string
	NextSequence   uint64
	Digest         string
	Source, Finish query.SketchSourceFacts
}
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
	out := Selection{Document: b.st.DocSeg, TabID: b.st.Tab.ID, SketchGeneration: b.submissionGeneration}
	for _, candidate := range b.st.Sketches.SubmissionCandidates() {
		witness, ok := b.feedback.sketches[candidate.ID]
		if !ok || !witness.finished {
			// Without its exact accepted witnesses (not labelled yet, or past
			// the witness bound) the sketch cannot be packaged; leave it out
			// rather than refuse the notes and sketches that can (pair#849).
			continue
		}
		out.Sketches = append(out.Sketches, SelectedSketch{ID: candidate.ID, NextSequence: candidate.NextSequence, Digest: candidate.Digest, Source: witness.source, Finish: witness.finish})
	}
	for _, n := range b.st.Notes {
		if n.State == recording.NoteOpen && !b.noteFeedbackHiddenLocked(n.ID, n.Revision) {
			out.Notes = append(out.Notes, SelectedNote{n.ID, n.Revision})
		}
	}
	return out, nil
}

// PressSelection is what a press submits (pair#849): the selection's open notes
// and finished sketches that delivered does not already hold, the oldest
// MaxSubmissionSketches sketches first. What the page shows plays no part.
func (s Selection) PressSelection(delivered core.DeliveredFeedback) Selection {
	out := s
	out.Notes, out.Sketches, out.SketchRemaining, out.NoteRemaining = nil, nil, 0, 0
	out.CurrentPage, out.Current = nil, nil
	for _, n := range s.Notes {
		if _, sent := delivered.Notes[core.PreviewNote{ID: n.ID, Revision: n.Revision}]; !sent {
			out.Notes = append(out.Notes, n)
		}
	}
	for _, v := range s.Sketches {
		if _, sent := delivered.Sketches[core.PreviewItem{RecordingID: delivered.RecordingID, SketchID: v.ID, Sequence: v.NextSequence, Digest: v.Digest}]; sent {
			continue
		}
		if len(out.Sketches) == core.MaxSubmissionSketches {
			out.SketchRemaining++
			continue
		}
		out.Sketches = append(out.Sketches, v)
	}
	return out
}

// FreezeSubmissionSelection must be called while the manager owns its preview
// state. The lock order is manager then Board; this operation performs no IO.
// notes must equal the entire ordered list acknowledged by that preview.
func (b *Board) FreezeSubmissionSelection(generation uint64, chosen []SelectedSketch, notes []SelectedNote) (Selection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, err := b.selectionLocked()
	if err != nil {
		return Selection{}, err
	}
	if generation != s.SketchGeneration || len(chosen) > 8 || !slices.Equal(notes, s.Notes) {
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
	err := b.surfaceAdmissionLocked(a.authority, false)
	b.mu.Unlock()
	if err != nil {
		return err
	}
	return b.emit(r, doc)
}

// FreezeVisibleSubmissionSelection is the final scalar-only Board fence. The
// actor resolves UI locators before this call; no IO or journal copies occur
// while the manager -> host -> Board locks are held.
func (b *Board) FreezeVisibleSubmissionSelection(generation uint64, chosen []SelectedSketch, notes []SelectedNote, view core.FeedbackVisibility, resolved []recording.NodeID) (Selection, error) {
	if core.ValidateFeedbackVisibility(view) != nil {
		return Selection{}, ErrStale
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	selection, err := b.selectionLocked()
	if err != nil {
		return Selection{}, err
	}
	page := b.currentFeedbackPageLocked()
	if generation != selection.SketchGeneration || len(chosen) > 8 || !sketchimage.SamePageView(page, view.Scene) || view.MaskScope != b.feedbackDisplay.mask.Scope || view.MaskRevision != b.feedbackDisplay.mask.Revision {
		return Selection{}, ErrStale
	}
	candidates := map[string]SelectedSketch{}
	unsent := 0
	for _, s := range selection.Sketches {
		candidates[s.ID] = s
		hidden, ok := b.feedbackDisplay.sketches[s.ID]
		if !ok || hidden.NextSequence != s.NextSequence || hidden.Digest != s.Digest {
			unsent++
		}
	}
	visible := map[string]core.VisibleSketch{}
	for _, s := range view.Sketches {
		candidate, ok := candidates[s.ID]
		if !ok || s.Sequence != candidate.NextSequence || s.Digest != candidate.Digest {
			return Selection{}, ErrStale
		}
		if hidden, ok := b.feedbackDisplay.sketches[s.ID]; ok && hidden.NextSequence == s.Sequence && hidden.Digest == s.Digest {
			return Selection{}, ErrStale
		}
		visible[s.ID] = s
	}
	currentNotes := map[SelectedNote]bool{}
	for _, n := range selection.Notes {
		currentNotes[n] = true
	}
	visibleNotes := map[SelectedNote]bool{}
	for _, n := range view.Notes {
		key := SelectedNote{n.ID, n.Revision}
		if !currentNotes[key] {
			return Selection{}, ErrStale
		}
		visibleNotes[key] = true
	}
	seenNotes := map[string]bool{}
	for _, n := range notes {
		if seenNotes[n.ID] || !currentNotes[n] || !visibleNotes[n] {
			return Selection{}, ErrStale
		}
		seenNotes[n.ID] = true
	}
	seen := map[string]bool{}
	projections := []sketchimage.PathProjection{}
	anchors := map[recording.NodeID]recording.NoteAt{}
	offset := 0
	for _, s := range chosen {
		if seen[s.ID] || candidates[s.ID] != s {
			return Selection{}, ErrStale
		}
		seen[s.ID] = true
		painted, ok := visible[s.ID]
		if !ok {
			return Selection{}, ErrStale
		}
		tops, ok := b.st.Sketches.StackTops(s.ID)
		if !ok || tops.VisibleCount != len(painted.Paths) {
			return Selection{}, ErrStale
		}
		for _, path := range painted.Paths {
			stroke, ok := b.st.Sketches.StrokeMetadata(s.ID, path.ID)
			if !ok || !stroke.Visible || offset >= len(resolved) {
				return Selection{}, ErrStale
			}
			node := b.st.Node(resolved[offset])
			offset++
			if node == nil || node.Y != recording.YEl {
				return Selection{}, ErrStale
			}
			for current, depth := node, 0; ; depth++ {
				if current == b.st.Top {
					break
				}
				if depth >= 256 || current.Y == recording.YDoc {
					return Selection{}, ErrStale
				}
				if current.Parent != nil {
					current = current.Parent
				} else if current.Y == recording.YShadow && current.Host != nil {
					current = current.Host
				} else {
					return Selection{}, ErrStale
				}
			}
			if known, ok := b.st.Boxes[node.ID]; ok {
				for i, value := range known {
					if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value-path.Box[i]) > 2 {
						return Selection{}, ErrStale
					}
				}
			}
			viewport := recording.Box{page.Scroll[0], page.Scroll[1], float64(page.VP.W), float64(page.VP.H)}
			if !viewport.Contains(path.Clip) {
				return Selection{}, ErrStale
			}
			if b.model == nil {
				return Selection{}, ErrStale
			}
			at, cached := anchors[node.ID]
			if !cached {
				var err error
				at, err = b.model.Anchor(b.st, node.ID)
				if err != nil || at.Node != node.ID {
					return Selection{}, ErrStale
				}
				anchors[node.ID] = at
			}
			box := path.Box
			at.Box = &box
			at.Route = page.URL
			at.Classes = slices.Clone(at.Classes)
			at.Path = slices.Clone(at.Path)
			at.Stable = maps.Clone(at.Stable)
			projections = append(projections, sketchimage.PathProjection{SketchID: s.ID, PathID: path.ID, At: at, Clip: path.Clip})
		}
	}
	if offset != len(resolved) {
		return Selection{}, ErrStale
	}
	selection.Sketches = slices.Clone(chosen)
	selection.Notes = slices.Clone(notes)
	selection.SketchRemaining = unsent - len(chosen)
	selection.NoteRemaining = len(currentNotes) - len(notes)
	if len(chosen) > 0 {
		selection.CurrentPage = &page
		selection.Current = &sketchimage.CurrentView{Projections: projections}
	}
	return selection, nil
}
