package board

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func prepareNoteFixture(t *testing.T, r *rig, operator bool) (AnnotationAdmission, PreparedNote) {
	t.Helper()
	a, err := r.bd.AdmitAnnotation()
	if operator {
		a, err = r.bd.admitOperatorAnnotation()
	}
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.PrepareNote(PrepareNoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{10, 20, 30, 40}, Region: recording.Box{12, 22, 20, 10}}, operator)
	if err != nil {
		t.Fatal(err)
	}
	return a, p
}

func TestPreparedNotePreservesAnchorAfterOriginalDisappears(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, p := prepareNoteFixture(t, r, false)
	want := cloneNote(&recording.Note{At: p.At, Region: &p.Region})
	r.add(&recording.Rm{ID: r.p1.ID})
	reserved, err := a.ReservePreparedNote(p, ana, 2, "saved later", "capture-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if reserved.Document != p.Document || !reflect.DeepEqual(reserved.Note.At, want.At) || !reflect.DeepEqual(reserved.Note.Region, want.Region) || reserved.Note.Text != "saved later" || reserved.Note.Capture != "capture-1" {
		t.Fatalf("changed prepared evidence: %+v", reserved)
	}
	p.At.Path[0] = "mutated"
	p.At.Box[0] = 999
	if !reflect.DeepEqual(reserved.Note.At, want.At) {
		t.Fatal("reservation aliases prepared anchor")
	}
	if _, ok := r.bd.ObservedNote(reserved.Note.ID); ok {
		t.Fatal("reservation is not a recorded note")
	}
	if err = r.w.Emit(&reserved.Note, reserved.Document); err != nil {
		t.Fatal(err)
	}
	got, ok := r.bd.ObservedNote(reserved.Note.ID)
	if !ok {
		t.Fatal("recorded note missing")
	}
	got.At.Path[0] = "mutated"
	got.At.Box[0] = 999
	got.Region[0] = 999
	again, ok := r.bd.ObservedNote(reserved.Note.ID)
	if !ok || !reflect.DeepEqual(again.At, want.At) || !reflect.DeepEqual(again.Region, want.Region) {
		t.Fatal("observation aliases recorded state")
	}
}

func TestPreparedNoteAdmissionDocumentAndAttribution(t *testing.T) {
	for _, kind := range []string{"cross-board", "disabled", "off-on", "navigation", "paused", "operator-role", "spoofed-author", "invalid-color", "empty-text", "invalid-capture", "driving"} {
		t.Run(kind, func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			operator := kind == "driving"
			a, p := prepareNoteFixture(t, r, operator)
			who, color, text, capture := ana, 0, "body", "capture-1"
			var want error
			switch kind {
			case "cross-board":
				other := newRig(t).page(recording.SegLive)
				a, _ = other.bd.AdmitAnnotation()
				want = ErrDisabled
			case "disabled":
				if err := r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
					t.Fatal(err)
				}
				want = ErrDisabled
			case "off-on":
				if err := r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
					t.Fatal(err)
				}
				if err := r.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
					t.Fatal(err)
				}
				a, _ = r.bd.AdmitAnnotation()
				want = ErrDisabled
			case "navigation":
				r.page(recording.SegNavigate)
				want = ErrStale
			case "paused":
				r.add(&recording.Stop{Why: recording.StopPause})
				want = ErrPaused
			case "operator-role":
				operator = true
				want = ErrDisabled
			case "spoofed-author":
				who = Viewer{ID: Operator}
				want = ErrShape
			case "invalid-color":
				color = -1
				want = ErrShape
			case "empty-text":
				text = " "
				want = ErrShape
			case "invalid-capture":
				capture = "../escape"
				want = ErrShape
			case "driving":
				r.bd.mu.Lock()
				r.bd.st.Control.Driver = &recording.Author{ID: "viewer"}
				r.bd.mu.Unlock()
				want = ErrDriving
			}
			_, err := a.ReservePreparedNote(p, who, color, text, capture, operator)
			if !errors.Is(err, want) {
				t.Fatalf("got %v want %v", err, want)
			}
			if len(r.bd.pending) != 0 {
				t.Fatal("refused reservation consumed capacity")
			}
		})
	}
	r := newRig(t).page(recording.SegLive)
	a, p := prepareNoteFixture(t, r, true)
	n, err := a.ReservePreparedNote(p, ana, 0, "operator text", "", true)
	if err != nil || n.Note.Author.ID != Operator || n.Note.Author.Name != "Operator" {
		t.Fatalf("operator attribution %+v %v", n, err)
	}
}

func TestPreparedNoteReservationCountsAndReleasesColorCapacity(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, p := prepareNoteFixture(t, r, false)
	var ids []string
	for i := 0; i < broadcast.MaxNotesPerColor; i++ {
		n, err := a.ReservePreparedNote(p, ana, 0, "body", "", false)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, n.Note.ID)
	}
	if _, err := a.ReservePreparedNote(p, ana, 0, "body", "", false); !errors.Is(err, ErrColorFull) {
		t.Fatalf("capacity: %v", err)
	}
	r.bd.ReleaseNoteReservation(ids[0])
	r.bd.ReleaseNoteReservation(ids[0])
	if _, err := a.ReservePreparedNote(p, ana, 0, "body", "", false); err != nil {
		t.Fatal(err)
	}
	if len(r.kinds("note")) != 0 {
		t.Fatal("reserve emitted a note before attachment intent")
	}
}

func TestPrepareNoteRejectsInvalidAnchorAndClampsRegion(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, _ := r.bd.AdmitAnnotation()
	in := PrepareNoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 2, 3, 4}, Region: recording.Box{-50, -50, 100, 100}}
	p, err := a.PrepareNote(in, false)
	if err != nil || p.Region != (recording.Box{0, 0, 50, 50}) || p.At.Text != "First" {
		t.Fatalf("prepare=%+v %v", p, err)
	}
	in.Node = r.p1.C[0].ID
	if _, err := a.PrepareNote(in, false); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	in.Node = r.p1.ID
	in.Region[2] = -1
	if _, err := a.PrepareNote(in, false); !errors.Is(err, ErrShape) {
		t.Fatal(err)
	}
	var zero AnnotationAdmission
	if _, err := zero.PrepareNote(in, false); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
}

func TestPreparedNoteClonesEvidenceCollections(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, p := prepareNoteFixture(t, r, false)
	p.At.Classes = []string{"card"}
	p.At.Stable = map[string]string{"id": "one"}
	n, err := a.ReservePreparedNote(p, ana, 0, "body", "", false)
	if err != nil {
		t.Fatal(err)
	}
	p.At.Classes[0] = "other"
	p.At.Stable["id"] = "other"
	if n.Note.At.Classes[0] != "card" || n.Note.At.Stable["id"] != "one" {
		t.Fatal("reserved note shares capture evidence collections")
	}
	if err = r.w.Emit(&n.Note, n.Document); err != nil {
		t.Fatal(err)
	}
	observed, _ := r.bd.ObservedNote(n.Note.ID)
	observed.At.Classes[0] = "changed"
	observed.At.Stable["id"] = "changed"
	again, _ := r.bd.ObservedNote(n.Note.ID)
	if again.At.Classes[0] != "card" || again.At.Stable["id"] != "one" {
		t.Fatal("observed note shares recorded evidence collections")
	}
}

func TestRestorePreparedNoteRetainsAuthenticatedSnapshotWithoutNode(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, p := prepareNoteFixture(t, r, false)
	wire, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var restored PreparedNote
	if err = json.Unmarshal(wire, &restored); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ReservePreparedNote(restored, ana, 0, "body", "", false); !errors.Is(err, ErrDisabled) {
		t.Fatalf("decoded value gained authority: %v", err)
	}
	r.add(&recording.Rm{ID: r.p1.ID})
	restored, err = a.RestorePreparedNote(restored, false)
	if err != nil {
		t.Fatal(err)
	}
	n, err := a.ReservePreparedNote(restored, ana, 0, "body", "", false)
	gotAnchor, _ := json.Marshal(n.Note.At)
	wantAnchor, _ := json.Marshal(p.At)
	if err != nil || string(gotAnchor) != string(wantAnchor) {
		t.Fatalf("restored note=%+v err=%v", n, err)
	}
	for _, mutate := range []func(*PreparedNote){func(p *PreparedNote) { p.Document++ }, func(p *PreparedNote) { p.TabID = "other" }, func(p *PreparedNote) { p.T = -1 }, func(p *PreparedNote) { p.At.Node = 0 }, func(p *PreparedNote) { p.Region[2] = -1 }, func(p *PreparedNote) { p.At.Kind = "other" }} {
		var value PreparedNote
		if err = json.Unmarshal(wire, &value); err != nil {
			t.Fatal(err)
		}
		mutate(&value)
		if _, err = a.RestorePreparedNote(value, false); err == nil {
			t.Fatal("accepted invalid restored snapshot")
		}
	}
	if err = r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err = a.RestorePreparedNote(p, false); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled restore: %v", err)
	}
}
