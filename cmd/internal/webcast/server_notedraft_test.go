package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type unavailableDraftCapture struct{}

func (unavailableDraftCapture) Capture(context.Context, board.PreparedNote) (notedraft.Evidence, []byte, error) {
	return notedraft.Evidence{}, nil, errors.New("test capture unavailable")
}
func TestViewerDraftPrepareSavePrivateOwner(t *testing.T) {
	c := newBoardCast(t)
	svc, err := notedraft.New(notedraft.Options{Root: t.TempDir(), RecordingPath: "rec", Capture: unavailableDraftCapture{}, Write: func(ctx context.Context, a board.AnnotationAdmission, n board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
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
	if status.State != "ready" && status.State != "failed" {
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
