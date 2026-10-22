package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		r.Header.Set("X-Couch-Annotation-Generation", "1")
		r.Header.Set(drive.SeatHeader, key)
		r.Header.Set("X-Couch-Drive-Generation", "1")
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
		r.Header.Set("X-Couch-Annotation-Generation", "1")
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
	r.Header.Set("X-Couch-Annotation-Generation", "1")
	r.Header.Set(drive.SeatHeader, key)
	r.Header.Set("X-Couch-Drive-Generation", "1")
	w := httptest.NewRecorder()
	f.srv.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || f.ctrl.Control().Driving() != viewer {
		t.Fatalf("late release revoked newgrant: code=%d control=%+v", w.Code, f.ctrl.Control())
	}
}

func TestAnnotationGenerationRejectsPreIngressAcrossOffOn(t *testing.T) {
	for _, route := range []string{"stroke", "note", "note-state", "note-edit"} {
		t.Run(route, func(t *testing.T) {
			c := newBoardCast(t)
			s := NewServer("view", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(c.bd).WithPointer("point")
			old := c.bd.AnnotationCapability().Generation
			if err := c.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			if err := c.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
				t.Fatal(err)
			}
			// Current-generation work must still succeed after re-enable.
			fresh := httptest.NewRequest(http.MethodPost, "/point/note", strings.NewReader(noteBody(1, c.p.ID, 0)))
			fresh.Header.Set("Content-Type", "application/json")
			fresh.Header.Set("X-Couch-Annotation-Generation", fmt.Sprint(c.bd.AnnotationCapability().Generation))
			answer := httptest.NewRecorder()
			s.ServeHTTP(answer, fresh)
			if answer.Code != http.StatusNoContent {
				t.Fatalf("fresh note: %d %s", answer.Code, answer.Body.String())
			}
			for _, generation := range []string{fmt.Sprint(old), "", "bogus"} {
				body := &capabilityBody{Reader: strings.NewReader(`{}`)}
				r := httptest.NewRequest(http.MethodPost, "/point/"+route, body)
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("X-Couch-Annotation-Generation", generation)
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != http.StatusForbidden || body.reads != 0 {
					t.Fatalf("generation=%q: code=%d reads=%d", generation, w.Code, body.reads)
				}
			}
		})
	}
}

func TestDriveGenerationRejectsPreIngressAcrossOffOn(t *testing.T) {
	for _, op := range []string{"request", "release", "input"} {
		t.Run(op, func(t *testing.T) {
			f := newDriveFixture(t, time.Minute)
			id, key := f.hello(t, "Ana")
			old := f.ctrl.DriveCapability().Generation
			if err := f.ctrl.SetDriveAllowed(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			if err := f.ctrl.SetDriveAllowed(context.Background(), true); err != nil {
				t.Fatal(err)
			}
			if op != "request" {
				if err := f.ctrl.Request(key); err != nil {
					t.Fatal(err)
				}
				if err := f.ctrl.Grant(id); err != nil {
					t.Fatal(err)
				}
			}
			before := len(f.stream.Records())
			route, body := "control", `{"op":"`+op+`"}`
			if op == "input" {
				route, body = "input", f.click("q")
			}
			for _, gen := range []string{fmt.Sprint(old), "", "bad"} {
				r := httptest.NewRequest(http.MethodPost, "/drv/drive/"+route, strings.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set(drive.SeatHeader, key)
				r.Header.Set("X-Couch-Drive-Generation", gen)
				w := httptest.NewRecorder()
				f.srv.ServeHTTP(w, r)
				if w.Code != http.StatusForbidden {
					t.Fatalf("%s gen%q accepted: %d %s", op, gen, w.Code, w.Body.String())
				}
			}
			if len(f.stream.Records()) != before {
				t.Fatal("retired generation changed recording")
			}
			if code, body := f.post(route, key, body); code != http.StatusNoContent {
				t.Fatalf("fresh generation: %d %s", code, body)
			}
		})
	}
}

func TestSketchAuthorityProjectionKeepsOriginEpoch(t *testing.T) {
	c := newBoardCast(t)
	f := newDriveFixture(t, time.Minute)
	srv := f.srv.WithBoard(c.bd).WithPointer("point")
	initial := srv.viewerCapabilities(linkDrive).Drawing
	if initial.Epoch == 0 || initial.Holder != "operator" || !initial.Ready {
		t.Fatalf("initial: %+v", initial)
	}
	_, key := f.hello(t, "Ana")
	if err := f.ctrl.Request(key); err != nil {
		t.Fatal(err)
	}
	if got := srv.viewerCapabilities(linkDrive).Drawing; got != initial {
		t.Fatalf("pending changed origin: %+v", got)
	}
	for _, role := range []castLink{linkView, linkPointer} {
		if srv.viewerCapabilities(role).Drawing.Ready {
			t.Fatal("nondriving role may draw")
		}
	}
}
