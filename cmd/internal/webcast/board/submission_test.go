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
			if err := r.w.Emit(&recording.NoteState{ID: id, State: state}, r.bd.st.DocSeg); err != nil {
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
	newID, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.bd.ResolveSelection(context.Background(), sel); err != nil {
		t.Fatal(err)
	}
	if live := r.bd.Live(); len(live) != 2 || !hasLive(live, id) || !hasLive(live, newID) {
		t.Fatal("reopened historical or new page note resolved")
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

func TestSubmissionSelectsCurrentLoadAndResolvesPreviouslyAdmittedHistory(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	first, _ := r.note(r.p1.ID, 0)
	original, _ := r.bd.Selection()
	r.page(recording.SegNavigate)
	second, _ := r.note(r.p1.ID, 1)
	selected, err := r.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Notes) != 1 || selected.Notes[0].ID != second {
		t.Fatalf("selection includes another load: %+v", selected)
	}
	history, err := recording.Reduce(r.mem.Records(), 1<<62)
	if err != nil || len(history.Notes) != 2 || history.Notes[0].ID != first || history.Notes[0].State != recording.NoteOpen {
		t.Fatalf("prior load lost from recording: %+v, %v", history, err)
	}
	if len(original.Notes) != 1 || original.Notes[0].ID != first {
		t.Fatalf("navigation changed admitted identity: %+v", original)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err = r.bd.ResolveSelection(ctx, original); err != nil {
		t.Fatal(err)
	}
	if r.bd.st.Notes[0].ID != first || r.bd.st.Notes[0].State != recording.NoteResolved || r.bd.st.Notes[1].State != recording.NoteOpen {
		t.Fatalf("historical acknowledgement resolved wrong exact version: %+v", r.bd.st.Notes)
	}
	selected, err = r.bd.Selection()
	if err != nil || len(selected.Notes) != 1 || selected.Notes[0].ID != second {
		t.Fatalf("historical ack consumed wrong selection: %+v %v", selected, err)
	}
}

func TestSubmissionHistoricalWriterRequiresGuardedResolution(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	id, _ := r.note(r.p1.ID, 0)
	selected, _ := r.bd.Selection()
	revision := selected.Notes[0].Revision
	r.page(recording.SegNavigate)
	for _, change := range []*recording.NoteState{{ID: id, State: recording.NoteResolved}, {ID: id, State: recording.NoteOpen, IfRevision: &revision}} {
		if err := r.w.Emit(change, 0); !errors.Is(err, ErrStale) {
			t.Fatalf("unguarded historical mutation accepted: %v", err)
		}
	}
	if err := r.w.Emit(&recording.NoteState{ID: id, State: recording.NoteResolved, IfRevision: &revision}, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.w.Emit(&recording.NoteState{ID: id, State: recording.NoteResolved, IfRevision: &revision}, 0); !errors.Is(err, ErrStale) {
		t.Fatalf("historical revision reused: %v", err)
	}
}
