package board

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestHistoricalOpenNotesRemainListedAndResolve(t *testing.T) {
	for _, operator := range []bool{false, true} {
		t.Run(map[bool]string{false: "viewer", true: "operator"}[operator], func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			id, err := r.note(r.p1.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			oldNode := r.p1.ID
			r.page(recording.SegNavigate)
			if got := r.bd.Live(); len(got) != 1 || got[0].ID != id {
				t.Fatalf("Couch list lost old note: %+v", got)
			}
			if got := r.bd.OperatorView(); len(got.Notes) != 1 || got.Notes[0].ID != id || got.Notes[0].At.Node != oldNode {
				t.Fatalf("operator list lost frozen note: %+v", got)
			}
			if operator {
				s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
					if doc != r.seg() {
						t.Errorf("resolve uses doc %d, want current %d", doc, r.seg())
					}
					err := r.w.Emit(rec, doc)
					if err != nil {
						done(OperatorRefused, err)
					} else {
						done(OperatorAccepted, nil)
					}
					return nil
				})
				if got := s.SetState("1", r.seg(), StateIn{ID: id, State: recording.NoteResolved}); got.Outcome != OperatorAccepted {
					t.Fatal(got)
				}
			} else if err := r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteResolved}); err != nil {
				t.Fatal(err)
			}
			if len(r.bd.Live()) != 0 || len(r.bd.OperatorView().Notes) != 0 {
				t.Fatal("resolved historical note still listed")
			}
			states := r.kinds("note-state")
			if len(states) != 1 {
				t.Fatalf("state records: %v", states)
			}
		})
	}
}

func TestHistoricalResolveKeepsCurrentAdmissionAndRevisionFences(t *testing.T) {
	for _, mode := range []string{"pause", "navigation", "revision", "permission"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			id, err := r.note(r.p1.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			r.page(recording.SegNavigate)
			want := ErrStale
			switch mode {
			case "pause":
				r.add(&recording.Stop{Why: recording.StopPause})
				want = ErrPaused
			case "navigation":
				r.before = func() { r.before = nil; r.page(recording.SegNavigate) }
			case "revision":
				r.before = func() {
					r.before = nil
					r.add(&recording.NoteState{ID: id, State: recording.NoteResolved})
					r.add(&recording.NoteState{ID: id, State: recording.NoteOpen})
				}
			case "permission":
				if err := r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
					t.Fatal(err)
				}
				want = ErrDisabled
			}
			if err := r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteResolved}); !errors.Is(err, want) {
				t.Fatalf("resolve: %v, want %v", err, want)
			}
		})
	}
}

func TestHistoricalListDoesNotConsumeCurrentPageColorQuota(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	for i := 0; i < broadcast.MaxNotesPerColor; i++ {
		if _, err := r.note(r.p1.ID, 0); err != nil {
			t.Fatal(err)
		}
	}
	r.page(recording.SegNavigate)
	if _, err := r.note(r.p1.ID, 0); err != nil {
		t.Fatalf("old page consumed new page quota: %v", err)
	}
	if len(r.bd.Live()) != broadcast.MaxNotesPerColor+1 {
		t.Fatal("list omitted historical notes")
	}
}

func TestHistoricalOperatorResolveKeepsQueuedPageFence(t *testing.T) {
	for _, mode := range []string{"pause", "navigation", "revision"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			id, err := r.note(r.p1.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			r.page(recording.SegNavigate)
			s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
				switch mode {
				case "pause":
					r.add(&recording.Stop{Why: recording.StopPause})
				case "navigation":
					r.page(recording.SegNavigate)
				case "revision":
					r.add(&recording.NoteState{ID: id, State: recording.NoteResolved})
					r.add(&recording.NoteState{ID: id, State: recording.NoteOpen})
				}
				err := r.w.Emit(rec, doc)
				if err == nil {
					done(OperatorAccepted, nil)
				} else {
					done(OperatorRefused, err)
				}
				return nil
			})
			want := ErrStale
			if mode == "pause" {
				want = ErrPaused
			}
			if got := s.SetState("1", r.seg(), StateIn{ID: id, State: recording.NoteResolved}); got.Outcome != OperatorRefused || got.Error != want.Error() {
				t.Fatalf("resolve escaped %s fence: %+v", mode, got)
			}
			if !hasLive(r.bd.Live(), id) {
				t.Fatal("refused resolve consumed old note")
			}
		})
	}
}
