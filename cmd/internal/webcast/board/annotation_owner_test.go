package board

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestTrustedOperatorAuthoringLeavesViewerAnnotationOff(t *testing.T) {
	for _, kind := range []string{"note", "pointer"} {
		t.Run(kind, func(t *testing.T) {
			r := newRigWithAnnotation(t, false).page(recording.SegLive)
			s := r.bd.NewOperator(func(record recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
				err := r.w.Emit(record, doc)
				if err != nil {
					done(OperatorRefused, err)
				} else {
					done(OperatorAccepted, nil)
				}
				return nil
			})
			if _, err := r.bd.AdmitAnnotation(); !errors.Is(err, ErrDisabled) {
				t.Fatalf("default viewer admission: %v", err)
			}
			var reply OperatorReply
			if kind == "note" {
				reply = s.Note("1", NoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 20, 20}, Region: recording.Box{1, 1, 10, 10}, Text: "owner note"})
			} else {
				reply = s.Stroke("1", StrokeIn{Seg: r.seg(), ID: "owner-pointer", Pts: [][3]float64{{1, 2, 0}}})
			}
			if reply.Outcome != OperatorAccepted {
				t.Fatalf("trusted operator %s with viewers OFF: %+v", kind, reply)
			}
			if got := r.bd.AnnotationCapability(); got.Phase != capability.Off || got.Generation != 0 {
				t.Fatalf("owner authoring granted viewers or changed generation: %+v", got)
			}
			_, err := r.bd.Note(NoteIn{Viewer: Viewer{ID: Operator}, Owner: "operator", Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 20, 20}, Region: recording.Box{1, 1, 10, 10}, Text: "claimed owner"})
			if !errors.Is(err, ErrDisabled) {
				t.Fatalf("claimed operator identity acquired viewer admission: %v", err)
			}
		})
	}
}

func TestOwnerAnnotationAuthorityPhasesAndGeneration(t *testing.T) {
	for _, phase := range []capability.Phase{capability.Off, capability.On, capability.Disabling, capability.Ended} {
		for _, outcome := range []capability.CleanupOutcome{capability.NoCleanup, capability.Running, capability.Retryable, capability.Broken} {
			if (phase == capability.Disabling) != (outcome != capability.NoCleanup) {
				continue
			}
			t.Run(fmt.Sprintf("phase%d/outcome%d", phase, outcome), func(t *testing.T) {
				b := New(notes.Structural{}, nil)
				b.annotation = capability.CapabilityState{Phase: phase, Generation: 7, Outcome: outcome, Attempt: 1}
				a, err := b.AdmitOperatorAnnotationGeneration(7)
				allowed := phase == capability.Off || phase == capability.On
				if (err == nil) != allowed {
					t.Fatalf("admit=%v phase=%v", err, phase)
				}
				if !allowed {
					return
				}
				if err = a.Check(); err != nil {
					t.Fatal(err)
				}
				b.annotation.Generation++
				if err = a.Check(); !errors.Is(err, ErrDisabled) {
					t.Fatalf("retired authority: %v", err)
				}
				if _, err = b.AdmitOperatorAnnotationGeneration(7); !errors.Is(err, ErrDisabled) {
					t.Fatalf("stale ingress: %v", err)
				}
			})
		}
	}
	b := New(notes.Structural{}, nil)
	if err := (AnnotationAdmission{board: b}).Check(); !errors.Is(err, ErrDisabled) {
		t.Fatalf("zero authority: %v", err)
	}
}

func TestOwnerPreparedNoteKeepsRoleAndOriginalGeneration(t *testing.T) {
	r := newRigWithAnnotation(t, false).page(recording.SegLive)
	a, err := r.bd.AdmitOperatorAnnotationGeneration(0)
	if err != nil {
		t.Fatal(err)
	}
	in := PrepareNoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 20, 20}, Region: recording.Box{1, 1, 10, 10}}
	p, err := a.PrepareNote(in, true)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PreparedNote
	if err = json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ReservePreparedNote(decoded, ana, 0, "text", "", true); !errors.Is(err, ErrDisabled) {
		t.Fatalf("wire authority: %v", err)
	}
	restored, err := a.RestorePreparedNote(decoded, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.ReservePreparedNote(restored, ana, 0, "owner", "", true); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ReservePreparedNote(p, ana, 0, "viewer", "", false); !errors.Is(err, ErrDisabled) {
		t.Fatalf("surface transfer: %v", err)
	}
	if err = r.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	viewer, err := r.bd.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = viewer.ReservePreparedNote(p, ana, 0, "viewer", "", true); !errors.Is(err, ErrDisabled) {
		t.Fatalf("role transfer: %v", err)
	}
	if _, err = a.RestorePreparedNote(decoded, true); !errors.Is(err, ErrDisabled) {
		t.Fatalf("retired restore: %v", err)
	}
	if _, err = a.PrepareNote(in, true); !errors.Is(err, ErrDisabled) {
		t.Fatalf("retired prepare: %v", err)
	}
}

func TestOwnerAnnotationCannotTransferToViewerSurface(t *testing.T) {
	r := newRigWithAnnotation(t, false).page(recording.SegLive)
	a, err := r.bd.AdmitOperatorAnnotationGeneration(0)
	if err != nil {
		t.Fatal(err)
	}
	in := PrepareNoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 20, 20}, Region: recording.Box{1, 1, 10, 10}}
	if _, err = a.PrepareNote(in, false); !errors.Is(err, ErrDisabled) {
		t.Fatalf("owner prepared viewer surface: %v", err)
	}
	if err = a.Stroke(StrokeIn{Viewer: ana, Seg: r.seg(), ID: "viewer", Pts: [][3]float64{{1, 2, 0}}}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("owner used viewer stroke: %v", err)
	}
}

func TestOwnerAnnotationWaitsForOrderedCleanup(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	entered, release := make(chan struct{}), make(chan struct{})
	r.bd.WithAnnotationCleanup(func(context.Context, uint64) error { close(entered); <-release; return nil })
	result := make(chan error, 1)
	go func() { result <- r.bd.SetAnnotationAllowed(context.Background(), false) }()
	<-entered
	generation := r.bd.AnnotationCapability().Generation
	if _, err := r.bd.AdmitOperatorAnnotationGeneration(generation); !errors.Is(err, ErrDisabled) {
		t.Errorf("owner crossed pending clear: %v", err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	a, err := r.bd.AdmitOperatorAnnotationGeneration(generation)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Check(); err != nil {
		t.Fatalf("settled OFF: %v", err)
	}
	if _, err = r.bd.AdmitAnnotation(); !errors.Is(err, ErrDisabled) {
		t.Fatalf("cleanup granted viewer: %v", err)
	}
}
