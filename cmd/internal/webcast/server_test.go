package webcast

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sse reads one event stream: each next() is the next (event, data) pair.
type sse struct {
	t    *testing.T
	resp *http.Response
	sc   *bufio.Scanner
}

func openSSE(t *testing.T, url string) *sse {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("events: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	return &sse{t: t, resp: resp, sc: bufio.NewScanner(resp.Body)}
}

func (s *sse) next() (string, string) {
	s.t.Helper()
	done := make(chan struct{})
	var event, data string
	go func() {
		defer close(done)
		for s.sc.Scan() {
			line := s.sc.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data = strings.TrimPrefix(line, "data: ")
			case line == "" && event != "":
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.t.Fatal("no event")
	}
	return event, data
}

func (s *sse) nextState() wireState {
	s.t.Helper()
	ev, data := s.next()
	if ev != "state" {
		s.t.Fatalf("got %s %s, want state", ev, data)
	}
	var w wireState
	json.Unmarshal([]byte(data), &w)
	return w
}

func TestServerStreamsStateAndEnd(t *testing.T) {
	f := NewFakeBrowser()
	f.Navigate("https://app/inbox", "Inbox")
	srv := httptest.NewServer(NewServer("tok", f, time.Second))
	t.Cleanup(srv.Close)
	s := openSSE(t, srv.URL+"/tok/events")
	if got := s.nextState(); got != (wireState{}) {
		t.Fatalf("join while idle: %+v", got)
	}
	f.SetLive(true)
	if got := s.nextState(); got != (wireState{Live: true, URL: "https://app/inbox", Title: "Inbox"}) {
		t.Fatalf("live: %+v", got)
	}
	f.Navigate("https://app/cart", "Cart")
	if got := s.nextState(); got.Title != "Cart" {
		t.Fatalf("navigate: %+v", got)
	}
	f.SetLive(false)
	if got := s.nextState(); got != (wireState{}) {
		t.Fatalf("pause: %+v", got)
	}
	f.QuitByHand()
	if ev, data := s.next(); ev != "end" || !strings.Contains(data, endedReason) {
		t.Fatalf("end: %s %s", ev, data)
	}
}

func TestServerRoutes(t *testing.T) {
	srv := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second))
	t.Cleanup(srv.Close)
	for _, r := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/tok/", 200},
		{"GET", "/tok/viewer.js", 200},
		{"GET", "/tok/viewer.css", 200},
		{"GET", "/tok/nope", 404},
		{"GET", "/other/", 404},
		{"GET", "/", 404},
		{"POST", "/tok/events", 405},
		{"POST", "/other/", 405},
	} {
		req, _ := http.NewRequest(r.method, srv.URL+r.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != r.want {
			t.Errorf("%s %s = %d, want %d", r.method, r.path, resp.StatusCode, r.want)
		}
		if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
			t.Errorf("%s %s: CSP %q", r.method, r.path, csp)
		}
		if resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s %s: headers %v", r.method, r.path, resp.Header)
		}
	}
}

func TestServerCapsViewers(t *testing.T) {
	s := NewServer("tok", NewFakeBrowser(), time.Second)
	s.maxViewers = 2
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	a, b := openSSE(t, srv.URL+"/tok/events"), openSSE(t, srv.URL+"/tok/events")
	a.nextState()
	b.nextState()
	resp, err := http.Get(srv.URL + "/tok/events")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("third viewer: %d", resp.StatusCode)
	}
}
