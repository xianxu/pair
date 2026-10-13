package webcast

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func lockedNoteEditServer(t *testing.T, c *boardCast) (string, *broadcast.LinkLocks) {
	t.Helper()
	locks := &broadcast.LinkLocks{}
	for _, token := range []string{"view", "tok"} {
		if err := locks.Mint(token, token, 4, nil); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(locks.End)
	srv := NewServer("view", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(c.bd).WithPointer("tok").WithLocks(locks)
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: time.Second})
	mt, err := host.Mount(context.Background(), srv, "tok/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mt.Unmount)
	return mt.Base(), locks
}

func TestWebNoteEditUsesCookieNotPublicIdentity(t *testing.T) {
	c := newBoardCast(t)
	base, locks := lockedNoteEditServer(t, c)
	link := base + "/tok/"
	request := func(client *http.Client, route, body string) (int, string) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, link+route, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Couch-Annotation-Generation", "1")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(data)
	}
	clients := make([]*http.Client, 2)
	for i := range clients {
		jar, _ := cookiejar.New(nil)
		clients[i] = &http.Client{Jar: jar, Timeout: 5 * time.Second}
		if status, body := request(clients[i], "unlock", fmt.Sprintf(`{"code":%q}`, locks.LockFor("tok").Code())); status != 200 {
			t.Fatalf("unlock %d %s", status, body)
		}
	}
	if status, body := request(clients[0], "note", noteBody(1, c.p.ID, 0)); status != 204 {
		t.Fatalf("note %d %s", status, body)
	}
	live := c.bd.Live()
	id := live[0].ID
	edit := fmt.Sprintf(`{"seg":1,"id":%q,"text":"updated"}`, id)
	if status, body := request(clients[0], "note-edit", edit); status != 204 {
		t.Fatalf("own edit %d %s", status, body)
	}
	if status, _ := request(clients[1], "note-edit", edit); status != 403 {
		t.Fatalf("other edit %d", status)
	}
	spoof := fmt.Sprintf(`{"seg":1,"id":%q,"text":"stolen","owner":"operator"}`, id)
	if status, _ := request(clients[1], "note-edit", spoof); status != 400 {
		t.Fatalf("owner injection %d", status)
	}
	if got := c.bd.Live(); len(got) != 1 || got[0].Text != "updated" || got[0].Author != live[0].Author {
		t.Fatalf("notes %+v", got)
	}
	if status, body := request(clients[0], "unlock", fmt.Sprintf(`{"code":%q}`, locks.LockFor("tok").Code())); status != 200 {
		t.Fatalf("fresh unlock %d %s", status, body)
	}
	if status, body := request(clients[0], "note-edit", edit); status != 403 {
		t.Fatalf("fresh session edited previous session's note: %d %s", status, body)
	}
}

type rotateWebNoteBody struct {
	io.Reader
	rotate func()
}

func (b *rotateWebNoteBody) Read(p []byte) (int, error) {
	if b.rotate != nil {
		f := b.rotate
		b.rotate = nil
		f()
	}
	return b.Reader.Read(p)
}

func TestWebNoteEditRefusesRotationDuringBodyRead(t *testing.T) {
	c := newBoardCast(t)
	locks := &broadcast.LinkLocks{}
	if err := locks.Mint("pointer", "tok", 4, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(locks.End)
	srv := NewServer("view", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(c.bd).WithPointer("tok").WithLocks(locks)
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: time.Second})
	mt, err := host.Mount(context.Background(), srv, "tok/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mt.Unmount)
	unlock := httptest.NewRequest(http.MethodPost, "/tok/unlock", strings.NewReader(fmt.Sprintf(`{"code":%q}`, locks.LockFor("tok").Code())))
	unlock.Header.Set("Content-Type", "application/json")
	uw := httptest.NewRecorder()
	host.ServeHTTP(uw, unlock)
	cookies := uw.Result().Cookies()
	if uw.Code != 200 || len(cookies) != 1 {
		t.Fatalf("unlock: %d", uw.Code)
	}
	request := func(route string, body io.Reader) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/tok/"+route, body)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Couch-Annotation-Generation", "1")
		r.AddCookie(cookies[0])
		w := httptest.NewRecorder()
		host.ServeHTTP(w, r)
		return w
	}
	if w := request("note", strings.NewReader(noteBody(1, c.p.ID, 0))); w.Code != 204 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	original := c.bd.Live()[0]
	body := &rotateWebNoteBody{Reader: strings.NewReader(fmt.Sprintf(`{"seg":1,"id":%q,"text":"late"}`, original.ID)), rotate: func() {
		if _, err := locks.LockFor("tok").Rotate(); err != nil {
			t.Fatal(err)
		}
	}}
	w := request("note-edit", body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("revoked edit: %d %s", w.Code, w.Body.String())
	}
	if got := c.bd.Live()[0].Text; got != original.Text {
		t.Fatalf("revoked edit changed note: %q", got)
	}
}
