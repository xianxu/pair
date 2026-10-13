package board

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"testing"
	"time"
)

// The Board already chose a revision, but the writer has not ordered its
// conditional resolution yet. A resolve/reopen in that gap must win.
func TestSubmissionResolutionRechecksRevisionAtWriter(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	id, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	sel, err := r.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Notes) != 1 {
		t.Fatalf("selection: %+v", sel)
	}
	intercepted := 0
	r.bd.emit = func(rec recording.Record, doc int) error {
		change, ok := rec.(*recording.NoteState)
		if !ok || change.ID != id || change.IfRevision == nil || *change.IfRevision != sel.Notes[0].Revision {
			t.Fatalf("missing captured revision at writer boundary: %#v", rec)
		}
		intercepted++
		for _, state := range []string{recording.NoteResolved, recording.NoteOpen} {
			if err := r.w.Emit(&recording.NoteState{ID: id, State: state}, doc); err != nil {
				t.Fatal(err)
			}
		}
		err := r.w.Emit(rec, doc)
		if !errors.Is(err, ErrStale) {
			t.Fatalf("writer accepted superseded note revision: %v", err)
		}
		return err
	}
	if err := r.bd.ResolveSelection(context.Background(), sel); err != nil {
		t.Fatal(err)
	}
	if intercepted != 1 {
		t.Fatalf("conditional writes=%d, want one", intercepted)
	}
	if live := r.bd.Live(); len(live) != 1 || live[0].ID != id {
		t.Fatalf("reopened note consumed: %+v", live)
	}
	if states := r.kinds("note-state"); len(states) != 2 || states[1].(*recording.NoteState).State != recording.NoteOpen {
		t.Fatalf("conditional resolution reached recording: %+v", states)
	}
	if err := r.bd.ResolveSelection(context.Background(), sel); err != nil {
		t.Fatal(err)
	}
	if intercepted != 1 {
		t.Fatal("retry emitted another stale resolution")
	}
}

func TestSubmissionSelectionAndGuardedResolution(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	one, _ := r.note(r.p1.ID, 0)
	sel, err := r.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	two, _ := r.note(r.p2.ID, 1)
	if len(sel.Notes) != 1 || sel.Notes[0].ID != one {
		t.Fatalf("selection %+v", sel)
	}
	if err = r.bd.ResolveSelection(context.Background(), sel); err != nil {
		t.Fatal(err)
	}
	if live := r.bd.Live(); len(live) != 1 || live[0].ID != two {
		t.Fatalf("unselected note consumed: %+v", live)
	}
	if err = r.bd.ResolveSelection(context.Background(), sel); err != nil {
		t.Fatal(err)
	}
}

func TestSubmissionResolutionPreservesChangedAndNewDocument(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	id, _ := r.note(r.p1.ID, 0)
	sel, _ := r.bd.Selection()
	r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteResolved})
	r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteOpen})
	if err := r.bd.ResolveSelection(context.Background(), sel); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.Live()) != 1 {
		t.Fatal("manually reopened note resolved")
	}
	r.page(recording.SegNavigate)
	r.note(r.p1.ID, 0)
	if err := r.bd.ResolveSelection(context.Background(), sel); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.Live()) != 1 {
		t.Fatal("new page note resolved")
	}
}

func TestSubmissionResolutionWaitsForRecordedOutcome(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.note(r.p1.ID, 0)
	sel, _ := r.bd.Selection()
	r.bd.emit = func(recording.Record, int) error { return nil } // accepted but never recorded
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := r.bd.ResolveSelection(ctx, sel); err == nil {
		t.Fatal("queue acceptance treated as resolution")
	}
}
