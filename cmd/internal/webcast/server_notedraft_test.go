package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if !ok || n.Owner != "private-A" || n.CropUnavailable == "" {
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

func TestDraftOwnerUsesLiveGateProof(t *testing.T) {
	c := newBoardCast(t)
	svc, err := notedraft.New(notedraft.Options{Root: t.TempDir(), RecordingPath: t.TempDir(), Capture: unavailableDraftCapture{}, Write: func(ctx context.Context, a board.AnnotationAdmission, n board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
		if err := check(); err != nil {
			return err
		}
		err := c.w.Emit(&n.Note, n.Document)
		if err != nil {
			settle(board.OperatorRefused, err)
		} else {
			settle(board.OperatorAccepted, nil)
		}
		return err
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	locks := &broadcast.LinkLocks{}
	if err := locks.Mint("pointer", "tok", 4, nil); err != nil {
		t.Fatal(err)
	}
	defer locks.End()
	srv := NewServer("view", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(c.bd).WithPointer("tok").WithLocks(locks).WithNoteDrafts(svc, func(r *http.Request) string { return broadcast.NoteOwnerFromContext(r.Context()) })
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: time.Second})
	mt, err := host.Mount(context.Background(), srv, "tok/")
	if err != nil {
		t.Fatal(err)
	}
	defer mt.Unmount()
	unlock := func() *http.Cookie {
		r := httptest.NewRequest("POST", "/tok/unlock", strings.NewReader(fmt.Sprintf(`{"code":%q}`, locks.LockFor("tok").Code())))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		host.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		return w.Result().Cookies()[0]
	}
	first, foreign := unlock(), unlock()
	request := func(cookie *http.Cookie, route string, body io.Reader) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/tok/"+route, body)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Couch-Annotation-Generation", "1")
		r.Header.Set(broadcast.NoteOwnerHeader, broadcast.OperatorNoteOwner)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		host.ServeHTTP(w, r)
		return w
	}
	prep := fmt.Sprintf(`{"request":"prepare1","seg":1,"node":%d,"box":[1,2,30,40],"region":[1,2,20,20]}`, c.p.ID)
	for _, cookie := range []*http.Cookie{nil, {Name: "couch_session", Value: "invalid"}} {
		if w := request(cookie, "prepare-note", strings.NewReader(prep)); w.Code != 401 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := request(first, "prepare-note", strings.NewReader(prep))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var status notedraft.Status
	json.Unmarshal(w.Body.Bytes(), &status)
	for end := time.Now().Add(time.Second); status.State == "pending" && time.Now().Before(end); {
		time.Sleep(600 * time.Millisecond)
		w = request(first, "draft-status", strings.NewReader(`{"request":"prepare1"}`))
		json.Unmarshal(w.Body.Bytes(), &status)
	}
	if status.State != "ready" {
		t.Fatal(status)
	}
	payload, _ := json.Marshal(map[string]any{"request": "save1", "draft": status.Draft, "viewer": map[string]string{"id": "ana", "name": "Ana"}, "color": 0, "text": "hello"})
	if w := request(foreign, "save-note", strings.NewReader(string(payload))); w.Code == 200 {
		t.Fatal("foreign session saved draft")
	}
	w = request(first, "save-note", strings.NewReader(string(payload)))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &status)
	n, ok := c.bd.ObservedNote(status.NoteID)
	if !ok || len(n.Owner) != 64 || n.Owner == broadcast.OperatorNoteOwner || n.Capture == "" {
		t.Fatal(n, ok)
	}
	w = request(first, "prepare-note", &rotateWebNoteBody{Reader: strings.NewReader(strings.ReplaceAll(prep, "prepare1", "prepare2")), rotate: func() {
		if err := locks.Rotate("pointer"); err != nil {
			t.Fatal(err)
		}
	}})
	if w.Code != http.StatusForbidden {
		t.Fatal("revoked during body read was not refused by ownership check", w.Code, w.Body.String())
	}
}
