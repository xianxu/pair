package board

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"testing"
	"time"
)

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
