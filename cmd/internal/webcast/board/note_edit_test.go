package board

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestNoteEditRequiresOwnerAndPreservesAuthor(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	id, err := r.bd.Note(NoteIn{Viewer: ana, Owner: "one", Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 2, 3, 4}, Region: recording.Box{1, 2, 30, 40}, Text: "original"})
	if err != nil {
		t.Fatal(err)
	}
	original := *r.bd.st.Notes[0].Note
	for _, owner := range []string{"", "two", broadcast.OperatorNoteOwner} {
		if err := r.bd.Edit(EditIn{Seg: r.seg(), ID: id, Owner: owner, Text: "stolen"}); !errors.Is(err, ErrOwner) {
			t.Fatalf("owner %q: %v", owner, err)
		}
	}
	if err := r.bd.Edit(EditIn{Seg: r.seg(), ID: id, Owner: "one", Text: "edited"}); err != nil {
		t.Fatal(err)
	}
	n := r.bd.st.Notes[0]
	if n.Text != "edited" || !n.Edited || n.Revision != 1 || n.Author != original.Author || n.At.Node != original.At.Node || n.T != original.T || len(r.bd.st.Notes) != 1 {
		t.Fatalf("note %+v", n)
	}
	if original.Text != "original" || len(r.kinds("note-edit")) != 1 {
		t.Fatal("mutated source or duplicate edit")
	}
	if err := r.bd.SetState(StateIn{Viewer: Viewer{ID: "bo", Name: "Bo"}, ID: id, State: recording.NoteResolved}); err != nil {
		t.Fatal(err)
	}
	if n.ResolvedBy == nil || n.ResolvedBy.Name != "Bo" {
		t.Fatalf("resolver %+v", n.ResolvedBy)
	}
	if err := r.bd.Edit(EditIn{Seg: r.seg(), ID: id, Owner: "one", Text: "resurrect"}); !errors.Is(err, ErrUnknown) {
		t.Fatalf("resolved edit %v", err)
	}
}

func TestNoteEditRechecksOrderedLifecycle(t *testing.T) {
	for _, change := range []string{"resolve", "pause", "navigation", "clear"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			id, err := r.bd.Note(NoteIn{Viewer: ana, Owner: "one", Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 10, 10}, Region: recording.Box{1, 1, 10, 10}, Text: "original"})
			if err != nil {
				t.Fatal(err)
			}
			r.before = func() {
				r.before = nil
				switch change {
				case "resolve":
					r.add(&recording.NoteState{ID: id, State: recording.NoteResolved})
				case "pause":
					r.add(&recording.Stop{Why: recording.StopPause})
				case "navigation":
					r.page(recording.SegNavigate)
				case "clear":
					r.add(&recording.AnnotationClear{Who: recording.Author{ID: Operator}})
				}
			}
			if err := r.bd.Edit(EditIn{Seg: 1, ID: id, Owner: "one", Text: "late"}); err == nil {
				t.Fatal("accepted retired edit")
			}
			if len(r.kinds("note-edit")) != 0 {
				t.Fatal("retired edit reached sink")
			}
		})
	}
}

func TestNoteEditRejectsOldCapabilityGeneration(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, err := r.bd.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if err := a.Edit(EditIn{Seg: 1, ID: "old", Owner: "one", Text: "late"}); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
}

func TestOperatorEditCommandShape(t *testing.T) {
	if _, err := ParseOperatorRequest([]byte(`{"generation":"1","op":"edit-note","request":"1","body":{"seg":1,"id":"abc","text":"updated"}}`)); err != nil {
		t.Fatalf("edit command: %v", err)
	}
	for _, body := range []string{
		`{"seg":1,"id":"abc","text":"x","owner":"operator"}`,
		`{"seg":1,"id":"abc","id":"other","text":"x"}`,
		`{"seg":1,"id":"abc","text":" "}`,
	} {
		if _, err := ParseOperatorRequest([]byte(`{"generation":"1","op":"edit-note","request":"1","body":` + body + `}`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestOperatorEditHasNoOverrideOrFalseUnknownProof(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	var settle func(OperatorOutcome, error)
	writes := 0
	s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
		writes++
		if rec.Kind() == "note" {
			if err := r.w.Emit(rec, doc); err != nil {
				return err
			}
			done(OperatorAccepted, nil)
		} else {
			settle = done
		}
		return nil
	})
	n := s.Note("1", NoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 10, 10}, Region: recording.Box{1, 1, 10, 10}, Text: "own"})
	if n.Outcome != OperatorAccepted {
		t.Fatal(n)
	}
	if r.bd.st.Notes[0].Owner != broadcast.OperatorNoteOwner {
		t.Fatal("operator ownership missing")
	}
	a, _ := r.bd.AdmitAnnotation()
	in := EditIn{Seg: r.seg(), ID: n.ID, Text: "edit"}
	if got := s.EditAdmitted(a, "2", in); got.Outcome != OperatorPending || got.ID != "" {
		t.Fatal(got)
	}
	settle(OperatorUnknown, errors.New("uncertain sink"))
	if got := s.Status("2"); got.Outcome != OperatorUnknown {
		t.Fatal("existing note falsely proved edit", got)
	}
	if got := s.EditAdmitted(a, "2", in); got.Outcome != OperatorUnknown || writes != 2 {
		t.Fatalf("replayed %+v writes=%d", got, writes)
	}
	in.Text = "different"
	if got := s.EditAdmitted(a, "2", in); got.Outcome != OperatorRefused {
		t.Fatal("changed payload accepted", got)
	}
	other := r.bd.NewOperator(func(recording.Record, int, AnnotationAdmission, func(OperatorOutcome, error)) error {
		t.Fatal("unauthorized write")
		return nil
	})
	id, err := r.bd.Note(NoteIn{Viewer: ana, Owner: "viewer", Seg: r.seg(), Node: r.p2.ID, Box: recording.Box{1, 1, 10, 10}, Region: recording.Box{1, 1, 10, 10}, Text: "viewer"})
	if err != nil {
		t.Fatal(err)
	}
	if got := other.EditAdmitted(a, "1", EditIn{Seg: r.seg(), ID: id, Text: "override"}); got.Outcome != OperatorRefused || got.Error != ErrOwner.Error() {
		t.Fatal(got)
	}
}

func TestSameLoadNoteEditPreservesCaptureAndRejectsEarlierLoad(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, _ := r.bd.AdmitAnnotation()
	p, err := a.PrepareNote(PrepareNoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 2, 3, 4}, Region: recording.Box{1, 2, 30, 40}}, false)
	if err != nil {
		t.Fatal(err)
	}
	n, err := a.ReservePreparedNote(p, ana, 0, "original", "capture-id", false)
	if err != nil {
		t.Fatal(err)
	}
	n.Note.Owner = "one"
	if err = r.w.Emit(&n.Note, n.Document); err != nil {
		t.Fatal(err)
	}
	old := r.seg()
	if err = r.bd.Edit(EditIn{Seg: r.seg(), ID: n.Note.ID, Owner: "other", Text: "stolen"}); !errors.Is(err, ErrOwner) {
		t.Fatalf("current load owner guard: %v", err)
	}
	r.add(&recording.Stop{Why: recording.StopPause}).page(recording.SegResume)
	if err = r.bd.Edit(EditIn{Seg: r.seg(), ID: n.Note.ID, Owner: "one", Text: "updated"}); err != nil {
		t.Fatal(err)
	}
	got := r.bd.st.Notes[0]
	if got.Text != "updated" || got.Capture != n.Note.Capture || got.At.Node != n.Note.At.Node || *got.Region != *n.Note.Region || got.Revision != 1 {
		t.Fatalf("lost original capture: %+v", got)
	}
	r.page(recording.SegNavigate)
	for _, in := range []EditIn{
		{Seg: old, ID: n.Note.ID, Owner: "one", Text: "stale document"},
		{Seg: r.seg(), ID: n.Note.ID, Owner: "other", Text: "stolen"},
		{Seg: r.seg(), ID: n.Note.ID, Owner: "one", Text: "earlier load"},
	} {
		if err = r.bd.Edit(in); !errors.Is(err, ErrStale) {
			t.Fatalf("earlier-load edit %+v: %v", in, err)
		}
	}
	if len(r.kinds("note-edit")) != 1 || r.bd.st.Notes[0].Text != "updated" || r.bd.st.Notes[0].Revision != 1 {
		t.Fatal("refused edit changed retained history")
	}
}
