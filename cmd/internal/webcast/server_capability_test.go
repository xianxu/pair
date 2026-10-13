package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
)

type capabilityBody struct {
	io.Reader
	before func()
	reads  int
}

func TestCapabilitiesStreamTracksAuthorityAndLinkRole(t *testing.T) {
	c := newBoardCast(t)
	next := func(s *sse) map[string]any {
		t.Helper()
		for {
			kind, body := s.next()
			if kind != "capabilities" {
				continue
			}
			var state map[string]any
			if err := json.Unmarshal([]byte(body), &state); err != nil {
				t.Fatal(err)
			}
			return state
		}
	}
	point := openSSE(t, c.url+"events")
	watch := openSSE(t, c.viewURL+"events")
	for _, item := range []struct {
		s       *sse
		allowed bool
	}{{point, true}, {watch, false}} {
		state := next(item.s)["annotation"].(map[string]any)
		if state["allowed"] != item.allowed || state["state"] != "on" {
			t.Fatalf("initial %+v", state)
		}
	}
	if err := c.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	for _, s := range []*sse{point, watch} {
		state := next(s)["annotation"].(map[string]any)
		for state["state"] == "turning-off" {
			state = next(s)["annotation"].(map[string]any)
		}
		if state["allowed"] != false || state["state"] != "off" {
			t.Fatalf("off %+v", state)
		}
	}
	joined := next(openSSE(t, c.url+"events"))["annotation"].(map[string]any)
	if joined["allowed"] != false || joined["state"] != "off" {
		t.Fatalf("reconnect %+v", joined)
	}
}

func TestDriveOffRefusesBeforeBodyAndAfterDelayedBody(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		f := newDriveFixture(t, time.Minute)
		_, key := f.hello(t, "Ana")
		offOn := func() {
			if err := f.ctrl.SetDriveAllowed(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			if delayed {
				if err := f.ctrl.SetDriveAllowed(context.Background(), true); err != nil {
					t.Fatal(err)
				}
			}
		}
		body := &capabilityBody{Reader: strings.NewReader(`{"op":"request"}`)}
		if delayed {
			body.before = offOn
		} else {
			offOn()
		}
		r := httptest.NewRequest(http.MethodPost, "/drv/drive/control", body)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set(drive.SeatHeader, key)
		w := httptest.NewRecorder()
		f.srv.ServeHTTP(w, r)
		status, message := drive.Answer(drive.ErrDriveUnavailable)
		if w.Code != status || strings.TrimSpace(w.Body.String()) != message {
			t.Fatalf("delayed=%v response=%d %s", delayed, w.Code, w.Body.String())
		}
		if !delayed && body.reads != 0 {
			t.Fatal("OFF read request body")
		}
		if len(f.ctrl.Control().Pending) != 0 {
			t.Fatal("late body requested drive")
		}
		f.hello(t, "Watching while off")
	}
}

func (b *capabilityBody) Read(p []byte) (int, error) {
	b.reads++
	if b.before != nil {
		f := b.before
		b.before = nil
		f()
	}
	return b.Reader.Read(p)
}

func TestAnnotationOffRefusesBeforeBodyAndAfterDelayedBody(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		c := newBoardCast(t)
		s := NewServer("view", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(c.bd).WithPointer("point")
		offOn := func() {
			if err := c.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			if delayed {
				if err := c.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
					t.Fatal(err)
				}
			}
		}
		body := &capabilityBody{Reader: strings.NewReader(noteBody(1, c.p.ID, 0))}
		if delayed {
			body.before = offOn
		} else {
			offOn()
		}
		r := httptest.NewRequest(http.MethodPost, "/point/note", body)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden || strings.TrimSpace(w.Body.String()) != "Annotation is off." {
			t.Fatalf("delayed=%v response=%d %s", delayed, w.Code, w.Body.String())
		}
		if !delayed && body.reads != 0 {
			t.Fatal("OFF read request body")
		}
		if len(c.bd.Live()) != 0 {
			t.Fatal("late body made a note")
		}
	}
}

func TestCapabilityRoleIntersectionAndBrokenProjection(t *testing.T) {
	for _, annotation := range []bool{false, true} {
		for _, driving := range []bool{false, true} {
			c := newBoardCast(t)
			f := newDriveFixture(t, time.Minute)
			if !annotation {
				if err := c.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
					t.Fatal(err)
				}
			}
			if !driving {
				if err := f.ctrl.SetDriveAllowed(context.Background(), false); err != nil {
					t.Fatal(err)
				}
			}
			srv := f.srv.WithBoard(c.bd).WithPointer("point")
			for _, link := range []castLink{linkView, linkPointer, linkDrive} {
				got := srv.viewerCapabilities(link)
				if got.Annotation.Allowed != (annotation && link.annotates()) || got.Drive.Allowed != (driving && link.drives()) {
					t.Fatalf("annotation=%v drive=%v role=%v: %+v", annotation, driving, link, got)
				}
			}
		}
	}
	c := newBoardCast(t)
	f := newDriveFixture(t, time.Minute)
	c.bd.WithAnnotationCleanup(func(context.Context, uint64) error { return errors.New("recorder gone") })
	if err := c.bd.SetAnnotationAllowed(context.Background(), false); err == nil {
		t.Fatal("missing failure")
	}
	got := f.srv.WithBoard(c.bd).viewerCapabilities(linkDrive)
	if got.Annotation.State != "broken" || got.Drive.State != "broken" || got.Annotation.Allowed || got.Drive.Allowed {
		t.Fatalf("broken cast left controls available: %+v", got)
	}
}

func TestOldReleaseBodyCannotRevokeNewGrant(t *testing.T) {
	f := newDriveFixture(t, time.Minute)
	viewer, key := f.hello(t, "Ana")
	grant := func() {
		if err := f.ctrl.Request(key); err != nil {
			t.Fatal(err)
		}
		if err := f.ctrl.Grant(viewer); err != nil {
			t.Fatal(err)
		}
	}
	grant()
	body := &capabilityBody{Reader: strings.NewReader(`{"op":"release"}`), before: func() {
		if err := f.ctrl.SetDriveAllowed(context.Background(), false); err != nil {
			t.Fatal(err)
		}
		if err := f.ctrl.SetDriveAllowed(context.Background(), true); err != nil {
			t.Fatal(err)
		}
		grant()
	}}
	r := httptest.NewRequest(http.MethodPost, "/drv/drive/control", body)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(drive.SeatHeader, key)
	w := httptest.NewRecorder()
	f.srv.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || f.ctrl.Control().Driving() != viewer {
		t.Fatalf("late release revoked newgrant: code=%d control=%+v", w.Code, f.ctrl.Control())
	}
}
