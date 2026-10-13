package notedraft

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"strings"
	"testing"
)

func TestSaveUnavailableTokenDoesNotNeedStoreCapacity(t *testing.T) {
	cap := &heldCapture{started: make(chan struct{}, 1)}
	var written []*recording.Note
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: cap, Write: func(_ context.Context, _ board.AnnotationAdmission, r board.NoteReservation, _ func() error, settle func(board.OperatorOutcome, error)) error {
		n := r.Note
		written = append(written, &n)
		settle(board.OperatorAccepted, nil)
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, p := preparation(t)
	owner := Owner{Scope: "viewer", Generation: 1}
	now := s.opts.Now()
	token, err := s.token(&captureRecord{ID: opaqueID(), Owner: owner, Request: "prepare", RecordingPath: s.opts.RecordingPath, Prepared: p, State: State{Phase: Ready, Unavailable: StoreFull}, Created: now, Expires: now.Add(DraftTTL)})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		got, err := s.Save(context.Background(), owner, "save", token, a, board.Viewer{ID: "viewer", Name: "Viewer"}, 0, "written despite full store", nil)
		if err != nil || got.State != "saved" {
			t.Fatal(got, err)
		}
	}
	if len(written) != 1 || written[0].CropUnavailable != StoreFull || written[0].Text != "written despite full store" {
		t.Fatal(written)
	}
	ids, err := s.disk.list()
	if err != nil || len(ids) != 0 {
		t.Fatal(ids, err)
	}
	// A caller cannot use another authenticated owner's preparation.
	if _, err = s.Save(context.Background(), Owner{Scope: "foreign", Generation: 1}, "save", token, a, board.Viewer{ID: "viewer", Name: "Viewer"}, 0, "no", nil); err == nil {
		t.Fatal("foreign token saved")
	}
}

func TestRefusedSaveReleasesTextAndClearsDurableIntent(t *testing.T) {
	cap := &heldCapture{started: make(chan struct{}, 1)}
	calls := 0
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: cap, Write: func(_ context.Context, _ board.AnnotationAdmission, _ board.NoteReservation, _ func() error, settle func(board.OperatorOutcome, error)) error {
		calls++
		if calls == 1 {
			settle(board.OperatorRefused, errors.New("writer refused before dispatch"))
		} else {
			settle(board.OperatorAccepted, nil)
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, p := preparation(t)
	owner := Owner{Scope: "viewer", Generation: 1}
	now := s.opts.Now()
	r := &captureRecord{ID: opaqueID(), Owner: owner, Request: "prepare", RecordingPath: s.opts.RecordingPath, Prepared: p, State: State{Phase: Ready, Unavailable: CaptureBusy}, Created: now, Expires: now.Add(DraftTTL)}
	data, _ := json.Marshal(r)
	if err = s.disk.put(r.ID, data, nil); err != nil {
		t.Fatal(err)
	}
	s.records[r.ID] = r
	got, err := s.Save(context.Background(), owner, "save-1", r.ID, a, board.Viewer{ID: "viewer", Name: "Viewer"}, 0, "original", nil)
	if err != nil || got.Error == "" {
		t.Fatal(got, err)
	}
	data, _, err = s.disk.read(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	var persisted captureRecord
	if err = json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.State.Phase != Ready || persisted.SaveRequest != "" || persisted.NoteID != "" {
		t.Fatalf("stale save intent: %+v", persisted)
	}
	got, err = s.Save(context.Background(), owner, "save-2", r.ID, a, board.Viewer{ID: "viewer", Name: "Viewer"}, 0, "edited", nil)
	if err != nil || got.State != "saved" {
		t.Fatal(got, err)
	}
}

func TestUnknownSavePreservesReasonAndEvidenceUntilWriterObservation(t *testing.T) {
	cap := &heldCapture{started: make(chan struct{}, 1)}
	var actual *recording.Note
	calls := 0
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: cap,
		Write: func(_ context.Context, _ board.AnnotationAdmission, r board.NoteReservation, _ func() error, settle func(board.OperatorOutcome, error)) error {
			calls++
			n := r.Note
			actual = &n
			settle(board.OperatorUnknown, errors.New("disk synchronization failed"))
			return nil
		}, Observe: func(context.Context, string) (*recording.Note, bool, error) { return actual, true, nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, p := preparation(t)
	owner := Owner{Scope: "viewer", Generation: 1}
	now := s.opts.Now()
	r := &captureRecord{ID: opaqueID(), Owner: owner, Request: "prepare", RecordingPath: s.opts.RecordingPath, Prepared: p, State: State{Phase: Ready, Unavailable: CaptureBusy}, Created: now, Expires: now.Add(DraftTTL)}
	data, _ := json.Marshal(r)
	if err = s.disk.put(r.ID, data, nil); err != nil {
		t.Fatal(err)
	}
	s.records[r.ID] = r
	got, err := s.Save(context.Background(), owner, "save", r.ID, a, board.Viewer{ID: "viewer", Name: "Viewer"}, 0, "original", nil)
	if err != nil || !strings.Contains(got.Error, "disk synchronization failed") || !strings.Contains(got.Error, "retry Save note") {
		t.Fatalf("lost actionable reason: %+v %v", got, err)
	}
	if err = s.Cancel(owner, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.disk.read(r.ID); err != nil {
		t.Fatal("unknown save evidence lost", err)
	}
	got, err = s.Save(context.Background(), owner, "save", r.ID, a, board.Viewer{ID: "viewer", Name: "Viewer"}, 0, "original", nil)
	if err != nil || got.State != "saved" || got.Error != "" || calls != 1 {
		t.Fatalf("ack reconciliation %+v calls=%d err=%v", got, calls, err)
	}
}
