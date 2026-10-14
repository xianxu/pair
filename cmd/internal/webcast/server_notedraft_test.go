package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type unavailableDraftCapture struct{}

func (unavailableDraftCapture) Capture(context.Context, board.PreparedNote) (notedraft.Evidence, []byte, error) {
	return notedraft.Evidence{Crop: core.Crop{Unavailable: notedraft.CaptureBusy}}, nil, nil
}
func TestViewerDraftPrepareSavePrivateOwner(t *testing.T) {
	c := newBoardCast(t)
	svc, err := notedraft.New(notedraft.Options{Root: t.TempDir(), RecordingPath: t.TempDir(), Capture: unavailableDraftCapture{}, Write: func(ctx context.Context, a board.AnnotationAdmission, n board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
		err := c.w.Emit(&n.Note, n.Document)
		if err != nil {
			settle(board.OperatorRefused, err)
		} else {
			settle(board.OperatorAccepted, nil)
		}
		return nil
	}, Release: c.bd.ReleaseNoteReservation})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	owner := func(r *http.Request) string {
		if v := r.URL.Query().Get("owner"); v != "" {
			if v == "none" {
				return ""
			}
			return v
		}
		return "private-A"
	}
	s := NewServer("view", NewFakeBrowser(), time.Second, nil, nil).WithBoard(c.bd).WithPointer("tok").WithNoteDrafts(svc, owner)
	server := httptest.NewServer(s)
	defer server.Close()
	url := server.URL + "/tok/"
	code, body := post(t, url+"prepare-note", "application/json", fmt.Sprintf(`{"request":"prepare1","seg":1,"node":%d,"box":[1,2,30,40],"region":[1,2,20,20]}`, c.p.ID))
	if code != 200 {
		t.Fatal(code, body)
	}
	var status notedraft.Status
	json.Unmarshal([]byte(body), &status)
	for end := time.Now().Add(time.Second); status.State == "pending" && time.Now().Before(end); {
		_, body = post(t, url+"draft-status", "application/json", `{"request":"prepare1"}`)
		json.Unmarshal([]byte(body), &status)
	}
	if status.State != "ready" {
		t.Fatal(status)
	}
	payload, _ := json.Marshal(map[string]any{"request": "save1", "draft": status.Draft, "viewer": map[string]string{"id": "ana", "name": "Ana"}, "color": 0, "text": "hello"})
	code, body = post(t, url+"save-note", "application/json", string(payload))
	if code != 200 {
		t.Fatal(code, body)
	}
	json.Unmarshal([]byte(body), &status)
	if status.State != "saved" || status.NoteID == "" {
		t.Fatal(status)
	}
	n, ok := c.bd.ObservedNote(status.NoteID)
	if !ok || n.CropUnavailable == "" {
		t.Fatal(n, ok)
	}
	// A request's attribution or guessed public viewer ID never supplies ownership.

	code, _ = post(t, url+"draft-status?owner=private-B", "application/json", `{"request":"prepare1"}`)
	if code == 200 {
		t.Fatal("another private owner read draft")
	}

	code, _ = post(t, url+"draft-status?owner=none", "application/json", `{"request":"prepare1"}`)
	if code != 403 {
		t.Fatal(code)
	}
	code, _ = post(t, server.URL+"/view/prepare-note", "application/json", `{}`)
	if code != 403 {
		t.Fatal(code)
	}
}

func TestViewerFinalAuthenticatedStreamDisconnectRevokesOnlyItsDrafts(t *testing.T) {
	c := newBoardCast(t)
	svc, err := notedraft.New(notedraft.Options{Root: t.TempDir(), RecordingPath: t.TempDir(), Capture: unavailableDraftCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	server := NewServer("view", NewFakeBrowser(), time.Second, nil, nil).WithBoard(c.bd).WithPointer("tok").WithNoteDrafts(svc, func(r *http.Request) string { return r.Header.Get("X-Test-Owner") })
	open := func(owner, viewer string) (context.CancelFunc, <-chan struct{}) {
		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest(http.MethodGet, "/tok/events?viewer="+viewer, nil).WithContext(ctx)
		req.Header.Set("X-Test-Owner", owner)
		writer := &draftStreamWriter{ResponseRecorder: httptest.NewRecorder(), ready: make(chan struct{})}
		done := make(chan struct{})
		go func() { defer close(done); server.ServeHTTP(writer, req) }()
		select {
		case <-writer.ready:
		case <-time.After(time.Second):
			cancel()
			t.Fatal("stream did not open")
		}
		return cancel, done
	}
	one, doneOne := open("private-A", "public1")
	defer one()
	two, doneTwo := open("private-A", "different-public")
	defer two()
	other, doneOther := open("private-B", "public1")
	defer other()
	a, err := c.bd.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := a.PrepareNote(board.PrepareNoteIn{Seg: 1, Node: c.p.ID, Box: recording.Box{1, 2, 30, 40}, Region: recording.Box{1, 2, 20, 20}}, false)
	if err != nil {
		t.Fatal(err)
	}
	owner := func(private string) notedraft.Owner {
		req := httptest.NewRequest(http.MethodGet, "/tok/prepare-note", nil)
		req.Header.Set("X-Test-Owner", private)
		return notedraft.Owner{Scope: server.noteOwnerScope(req), Generation: 1}
	}
	for _, private := range []string{"private-A", "private-B"} {
		if _, err = svc.Prepare(context.Background(), owner(private), "prepare", a, prepared, false); err != nil {
			t.Fatal(err)
		}
	}
	one()
	<-doneOne
	if got, err := svc.Status(owner("private-A"), "prepare"); err != nil || got.State == "cancelled" {
		t.Fatal("another live stream lost draft", got, err)
	}
	two()
	<-doneTwo
	if got, err := svc.Status(owner("private-A"), "prepare"); err != nil || got.State != "cancelled" {
		t.Fatal("final disconnect retained draft", got, err)
	}
	if got, err := svc.Status(owner("private-B"), "prepare"); err != nil || got.State == "cancelled" {
		t.Fatal("public viewer ID crossed owners", got, err)
	}
	other()
	<-doneOther
}

type draftStreamWriter struct {
	*httptest.ResponseRecorder
	ready chan struct{}
	once  sync.Once
}

func (w *draftStreamWriter) Flush() { w.ResponseRecorder.Flush(); w.once.Do(func() { close(w.ready) }) }
