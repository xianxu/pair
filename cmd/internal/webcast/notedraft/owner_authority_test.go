package notedraft

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Exercise the real capacity fallback: the active capture is held, the eight
// queued captures fill the existing budget, and the next preparation is signed.
func ownerUnavailablePreparation(t *testing.T, s *Service, capture *heldCapture, owner Owner, a board.AnnotationAdmission, p board.PreparedNote) Status {
	t.Helper()
	if _, err := s.Prepare(context.Background(), owner, "held", a, p, true); err != nil {
		t.Fatal(err)
	}
	<-capture.started
	for i := 0; i < 8; i++ {
		if _, err := s.Prepare(context.Background(), owner, fmt.Sprintf("queued-%d", i), a, p, true); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Prepare(context.Background(), owner, "unavailable", a, p, true)
	if err != nil || got.State != "ready" || got.Unavailable != CaptureBusy || !strings.HasPrefix(got.Draft, "u.") {
		t.Fatalf("signed owner fallback: %+v %v", got, err)
	}
	return got
}

func TestOwnerInitialOffUnavailableTokenRestoreSave(t *testing.T) {
	bd, a, p := preparationAuthority(t, true)
	owner := Owner{Scope: "authenticated-console-document", Identity: broadcast.OperatorNoteOwner, Generation: 0}
	capture := &heldCapture{started: make(chan struct{}, 1)}
	var written []recording.Note
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: capture,
		Write: func(_ context.Context, admitted board.AnnotationAdmission, n board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
			if err := admitted.Check(); err != nil {
				return err
			}
			if check != nil {
				if err := check(); err != nil {
					return err
				}
			}
			written = append(written, n.Note)
			settle(board.OperatorAccepted, nil)
			return nil
		}, Release: bd.ReleaseNoteReservation})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	draft := ownerUnavailablePreparation(t, s, capture, owner, a, p)
	// None of these claims can consume the genuine owner token.
	for _, foreign := range []Owner{
		{Scope: owner.Scope, Identity: "viewer", Generation: 0},
		{Scope: "other-document", Identity: owner.Identity, Generation: 0},
		{Scope: owner.Scope, Identity: owner.Identity, Generation: 1},
	} {
		if _, err := s.Save(context.Background(), foreign, "save", draft.Draft, a, board.Viewer{}, 0, "owner text", nil); err == nil {
			t.Fatalf("foreign owner consumed token: %+v", foreign)
		}
	}
	// Even a service-authenticated token for the other surface cannot be
	// restored with owner authority. The operator bit is not a bypass flag.
	otherSurface, err := s.untoken(owner, draft.Draft)
	if err != nil {
		t.Fatal(err)
	}
	otherSurface.Operator = false
	otherToken, err := s.token(otherSurface)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(context.Background(), owner, "save", otherToken, a, board.Viewer{}, 0, "owner text", nil); !errors.Is(err, board.ErrDisabled) {
		t.Fatalf("owner consumed viewer surface: %v", err)
	}
	_, viewerAdmission, _ := preparationAuthority(t, false)
	if _, err := s.Save(context.Background(), owner, "save", draft.Draft, viewerAdmission, board.Viewer{ID: "viewer", Name: "Viewer"}, 0, "owner text", nil); !errors.Is(err, board.ErrDisabled) {
		t.Fatalf("viewer authority consumed operator token: %v", err)
	}
	if len(written) != 0 {
		t.Fatal("foreign token reached writer")
	}
	checks := 0
	for i := 0; i < 2; i++ {
		got, err := s.Save(context.Background(), owner, "save", draft.Draft, a, board.Viewer{ID: "claimed-viewer", Name: "Claimed viewer"}, 0, "owner text", func() error { checks++; return nil })
		if err != nil || got.State != "saved" {
			t.Fatalf("restored owner save: %+v %v", got, err)
		}
	}
	if checks != 1 || len(written) != 1 || written[0].Owner != broadcast.OperatorNoteOwner || written[0].Author.ID != board.Operator || written[0].CropUnavailable != CaptureBusy || written[0].Capture == "" {
		t.Fatalf("owner restoration/write evidence: checks=%d notes=%+v", checks, written)
	}
	if _, err := bd.AdmitAnnotation(); err == nil || bd.AnnotationCapability().Phase != capability.Off {
		t.Fatal("owner token save granted viewer annotation")
	}
}

func TestOwnerInitialOffUnknownSaveReconcilesWithoutDuplicate(t *testing.T) {
	bd, a, p := preparationAuthority(t, true)
	owner := Owner{Scope: "authenticated-console-document", Identity: broadcast.OperatorNoteOwner, Generation: 0}
	capture := &heldCapture{started: make(chan struct{}, 1)}
	var actual *recording.Note
	calls := 0
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: capture, Release: bd.ReleaseNoteReservation,
		Write: func(_ context.Context, admitted board.AnnotationAdmission, n board.NoteReservation, _ func() error, settle func(board.OperatorOutcome, error)) error {
			if err := admitted.Check(); err != nil {
				return err
			}
			calls++
			note := n.Note
			actual = &note
			settle(board.OperatorUnknown, errors.New("owner write acknowledgement lost"))
			return nil
		}, Observe: func(context.Context, string) (*recording.Note, bool, error) { return actual, actual != nil, nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	draft := ownerUnavailablePreparation(t, s, capture, owner, a, p)
	got, err := s.Save(context.Background(), owner, "save", draft.Draft, a, board.Viewer{}, 0, "owner text", nil)
	if err != nil || got.State != "pending" || !strings.Contains(got.Error, "acknowledgement lost") || calls != 1 {
		t.Fatalf("unknown owner save: %+v calls=%d err=%v", got, calls, err)
	}
	if err := s.Cancel(owner, got.Draft); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		got, err = s.Save(context.Background(), owner, "save", draft.Draft, a, board.Viewer{}, 0, "owner text", nil)
		if err != nil || got.State != "saved" || got.Error != "" || calls != 1 || actual == nil || got.NoteID != actual.ID {
			t.Fatalf("owner ACK reconciliation: %+v calls=%d err=%v", got, calls, err)
		}
	}
	if actual.Owner != owner.Identity || actual.Author.ID != board.Operator || bd.AnnotationCapability().Phase != capability.Off {
		t.Fatal("owner attribution or viewer OFF changed")
	}
}
