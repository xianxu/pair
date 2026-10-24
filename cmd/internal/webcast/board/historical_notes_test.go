package board

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestEarlierLoadFreshResolveRefusesWhileAdmittedResolutionRetainsHistory(t *testing.T) {
	for _, operator := range []bool{false, true} {
		t.Run(map[bool]string{false: "viewer", true: "operator"}[operator], func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			id, err := r.note(r.p1.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			admitted, err := r.bd.Selection()
			if err != nil || len(admitted.Notes) != 1 {
				t.Fatalf("admitted: %+v, %v", admitted, err)
			}
			oldNode := r.p1.ID
			r.page(recording.SegNavigate)
			if got := r.bd.Live(); len(got) != 1 || got[0].ID != id {
				t.Fatalf("Couch list lost old note: %+v", got)
			}
			if got := r.bd.OperatorView(); len(got.Notes) != 1 || got.Notes[0].ID != id || got.Notes[0].At.Node != oldNode {
				t.Fatalf("operator list lost frozen note: %+v", got)
			}
			writes := 0
			var session *OperatorSession
			if operator {
				session = r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
					writes++
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
			}
			if operator {
				if got := session.SetState("1", r.seg(), StateIn{ID: id, State: recording.NoteResolved}); got.Outcome != OperatorRefused || got.Error != ErrStale.Error() {
					t.Fatalf("fresh earlier-load resolve: %+v", got)
				}
			} else if err := r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteResolved}); !errors.Is(err, ErrStale) {
				t.Fatalf("fresh earlier-load resolve: %v", err)
			}
			if writes != 0 || len(r.kinds("note-state")) != 0 || !hasLive(r.bd.Live(), id) {
				t.Fatal("earlier-load request reached writer or consumed history")
			}
			current, err := r.note(r.p1.ID, 1)
			if err != nil {
				t.Fatal(err)
			}
			if operator {
				if got := session.SetState("2", r.seg(), StateIn{ID: current, State: recording.NoteResolved}); got.Outcome != OperatorAccepted {
					t.Fatal(got)
				}
				if writes != 1 {
					t.Fatalf("current-load writer calls=%d", writes)
				}
			} else if err := r.bd.SetState(StateIn{Viewer: ana, ID: current, State: recording.NoteResolved}); err != nil {
				t.Fatal(err)
			}
			if !hasLive(r.bd.Live(), id) || hasLive(r.bd.Live(), current) {
				t.Fatal("current resolve consumed wrong note")
			}
			if err := r.bd.ResolveSelection(context.Background(), admitted); err != nil {
				t.Fatal(err)
			}
			if len(r.bd.Live()) != 0 || len(r.bd.OperatorView().Notes) != 0 {
				t.Fatal("admitted historical resolution was not observed")
			}
			if len(r.kinds("note-state")) != 2 || len(r.bd.st.Notes) != 2 {
				t.Fatal("resolution lost retained history or wrote duplicate state")
			}
		})
	}
}

func TestCurrentLoadResolveKeepsAdmissionAndRevisionFences(t *testing.T) {
	for _, mode := range []string{"pause", "navigation", "revision", "permission"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			id, err := r.note(r.p1.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			queued := 0
			want := ErrStale
			switch mode {
			case "pause":
				r.add(&recording.Stop{Why: recording.StopPause})
				want = ErrPaused
			case "navigation":
				r.before = func() { queued++; r.before = nil; r.page(recording.SegNavigate) }
			case "revision":
				r.before = func() {
					queued++
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
			if (mode == "navigation" || mode == "revision") && queued != 1 {
				t.Fatalf("writer fence was not exercised: %d", queued)
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

func TestCurrentLoadOperatorResolveKeepsQueuedPageFence(t *testing.T) {
	for _, mode := range []string{"pause", "navigation", "revision"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			id, err := r.note(r.p1.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			writes := 0
			s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
				writes++
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
			if writes != 1 {
				t.Fatalf("queued writer fence never reached: %d", writes)
			}
			if !hasLive(r.bd.Live(), id) {
				t.Fatal("refused resolve consumed old note")
			}
		})
	}
}
