package board

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Selection is a frozen set of open notes, across recorded documents.
type Selection struct {
	Document int
	TabID    string
	Notes    []SelectedNote
}
type SelectedNote struct {
	ID       string
	Revision uint64
}

func (b *Board) Selection() (Selection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return Selection{}, ErrClosed
	}
	if b.broken != nil {
		return Selection{}, ErrBroken
	}
	if b.st.Top == nil {
		return Selection{}, ErrNotLive
	}
	out := Selection{Document: b.st.DocSeg, TabID: b.st.Tab.ID}
	for _, n := range b.st.Notes {
		if n.State == recording.NoteOpen {
			out.Notes = append(out.Notes, SelectedNote{n.ID, n.Revision})
		}
	}
	return out, nil
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
			err := admission.resolveSelected(&recording.NoteState{ID: n.ID, State: recording.NoteResolved, IfRevision: &rev}, 0)
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
