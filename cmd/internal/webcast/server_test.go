package webcast

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
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
	resp, err := castClient.Do(req)
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
	for ev == "capabilities" || ev == "feedback" {
		ev, data = s.next()
	}
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
	srv := httptest.NewServer(NewServer("tok", f, time.Second, nil, nil))
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

// TestServerAssets: <token>/assets/<hash> is the proxy's, empty until the
// recording's store is bound, and only under this cast's token.
func TestServerAssets(t *testing.T) {
	s := NewServer("tok", NewFakeBrowser(), time.Second, nil, nil)
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	store := castassets.NewRecorded(&discardWriter{}, castassets.DefaultPolicy)
	a, err := store.Put(context.Background(), castassets.Capture{URL: "http://x/f.woff2", MIME: "font/woff2", Body: []byte("wOF2")})
	if err != nil {
		t.Fatal(err)
	}
	status := func(path string) (int, string) {
		resp, err := castClient.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if code, _ := status("/tok/assets/" + a.Hash); code != 404 {
		t.Errorf("before the recording opens: %d", code)
	}
	s.Assets().Bind(store)
	if code, body := status("/tok/assets/" + a.Hash); code != 200 || body != "wOF2" {
		t.Errorf("bound: %d %q", code, body)
	}
	if code, _ := status("/other/assets/" + a.Hash); code != 404 {
		t.Errorf("another token: %d", code)
	}
}

// discardWriter is an AssetWriter that keeps nothing (the store then keeps
// bodies itself).
type discardWriter struct{}

func (discardWriter) PutAsset(body []byte) (string, error) { return recording.HashOf(body), nil }

func TestServerRoutes(t *testing.T) {
	srv := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second, nil, nil))
	t.Cleanup(srv.Close)
	for _, r := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/tok/", 200},
		{"GET", "/tok/viewer.js", 200},
		{"GET", "/tok/floating-panel.js", 200},
		{"GET", "/tok/sketch-notes.js", 200},
		{"GET", "/tok/viewer.css", 200},
		{"GET", "/tok/model.js", 200},
		{"GET", "/tok/render.js", 200},
		{"GET", "/tok/css.js", 200},
		{"GET", "/tok/nope", 404},
		{"GET", "/other/", 404},
		{"GET", "/", 404},
		{"POST", "/tok/events", 405},
		{"POST", "/other/", 405},
	} {
		req, _ := http.NewRequest(r.method, srv.URL+r.path, nil)
		resp, err := castClient.Do(req)
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
	s := NewServer("tok", NewFakeBrowser(), time.Second, nil, nil)
	s.maxViewers = 2
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	a, b := openSSE(t, srv.URL+"/tok/events"), openSSE(t, srv.URL+"/tok/events")
	a.nextState()
	b.nextState()
	resp, err := castClient.Get(srv.URL + "/tok/events")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("third viewer: %d", resp.StatusCode)
	}
}

// nextRecs reads n rec events, skipping state events.
func (s *sse) nextRecs(n int) []recording.Record {
	s.t.Helper()
	var out []recording.Record
	for len(out) < n {
		ev, data := s.next()
		if ev == "state" || ev == "capabilities" || ev == "feedback" {
			continue
		}
		if ev != "rec" {
			s.t.Fatalf("got %s %s, want rec", ev, data)
		}
		r, err := recording.Decode([]byte(data))
		if err != nil {
			s.t.Fatalf("rec %q: %v", data, err)
		}
		out = append(out, r)
	}
	return out
}

func sameLines(t *testing.T, got, want []recording.Record) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d", len(got), len(want))
	}
	for i := range want {
		g, _ := recording.Encode(got[i])
		w, _ := recording.Encode(want[i])
		if string(g) != string(w) {
			t.Fatalf("record %d: got %s, want %s", i, g, w)
		}
	}
}

// A viewer joining mid-cast gets the catch-up, then every record as it
// comes, across a pause, a resume and a navigation.
func TestServerStreamsRecordsFromCatchUp(t *testing.T) {
	recs := recording.ContractRecording(10)
	for _, cut := range []int{1, len(recs) / 2, len(recs) - 2} {
		hub := recording.NewHub(len(recs))
		for _, r := range recs[:cut] {
			hub.Put(r)
		}
		srv := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second, hub, nil))
		s := openSSE(t, srv.URL+"/tok/events")
		want := recording.CatchUp(recs[:cut])
		sameLines(t, s.nextRecs(len(want)), want)
		for _, r := range recs[cut:] {
			hub.Put(r)
		}
		sameLines(t, s.nextRecs(len(recs)-cut), recs[cut:])
		hub.Close()
		s.resp.Body.Close() // the stream outlives the fan-out; the viewer leaves
		srv.Close()
	}
}

// stubLive is a Live whose subscription the test ends by hand, and says
// why (ended: the fan-out's end; else a drop).
type stubLive struct {
	recs  chan recording.Record
	ended bool
}

func (l *stubLive) Subscribe() (<-chan recording.Record, func()) { return l.recs, func() {} }
func (l *stubLive) Ended(<-chan recording.Record) bool           { return l.ended }

// A viewer dropped as too slow has its stream ended, so it reconnects for
// a fresh catch-up.
func TestServerEndsStreamOfDroppedViewer(t *testing.T) {
	live := &stubLive{recs: make(chan recording.Record)}
	srv := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second, live, nil))
	t.Cleanup(srv.Close)
	s := openSSE(t, srv.URL+"/tok/events")
	s.nextState()
	close(live.recs) // dropped
	done := make(chan struct{})
	go func() {
		for s.sc.Scan() {
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a dropped viewer's stream stayed open")
	}
}

// A closed fan-out leaves the viewer with what it has: no reconnect loop,
// and the stream still says when the cast ends.
func TestServerKeepsStreamWhenRecordsEnd(t *testing.T) {
	hub := recording.NewHub(64)
	recs := recording.ContractRecording(2)
	for _, r := range recs {
		hub.Put(r)
	}
	hub.Close()
	f := NewFakeBrowser()
	srv := httptest.NewServer(NewServer("tok", f, time.Second, hub, nil))
	t.Cleanup(srv.Close)
	s := openSSE(t, srv.URL+"/tok/events")
	want := recording.CatchUp(recs)
	sameLines(t, s.nextRecs(len(want)), want)
	f.QuitByHand()
	for {
		switch ev, _ := s.next(); ev {
		case "end":
			return
		case "rec":
			t.Fatal("records again after the fan-out ended")
		case "":
			t.Fatal("the stream ended without end: the viewer would reconnect")
		}
	}
}

func TestServerPresence(t *testing.T) {
	var mu sync.Mutex
	var log []string
	presence := func(id string) func() {
		mu.Lock()
		defer mu.Unlock()
		log = append(log, "join "+id)
		return func() {
			mu.Lock()
			defer mu.Unlock()
			log = append(log, "leave "+id)
		}
	}
	srv := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second, nil, presence))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/tok/events?viewer=v-1", nil)
	resp, err := castClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	bufio.NewReader(resp.Body).ReadString('\n')
	cancel()
	resp.Body.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		got := strings.Join(log, ",")
		mu.Unlock()
		if got == "join v-1,leave v-1" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("presence: %q", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// No id: no seat. A malformed id is refused before anything opens.
	for path, want := range map[string]int{"/tok/events?viewer=a%20b": 400, "/tok/events?viewer=" + strings.Repeat("x", 65): 400} {
		resp, err := castClient.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("%s: %d, want %d", path, resp.StatusCode, want)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(log) != 2 {
		t.Errorf("a refused stream reached presence: %v", log)
	}
}

// The page frame inherits the viewer's policy: the recorded page's styles
// may load, its scripts may not.
func TestServerPolicyKeepsScriptsOut(t *testing.T) {
	script := ""
	for _, d := range strings.Split(contentSecurityPolicy, ";") {
		if f := strings.Fields(d); len(f) > 0 && f[0] == "script-src" {
			script = strings.Join(f[1:], " ")
		}
	}
	if script != "'self'" {
		t.Fatalf("script-src %q, want only 'self'", script)
	}
	for _, want := range []string{"style-src 'self' 'unsafe-inline' blob:", "img-src 'self'", "font-src 'self'"} {
		if !strings.Contains(contentSecurityPolicy, want) {
			t.Errorf("policy lacks %q", want)
		}
	}
}
