package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// boardCast is a cast with a board as #458 wires it: the recorder's records
// and the board's go through one writer into the hub viewers stream from and
// into the board, which follows them. url is the pointer link, which may
// annotate; viewURL is the view-only link (pair#506).
type boardCast struct {
	url     string
	viewURL string
	bd      *board.Board
	hub     *recording.Hub
	w       *board.Writer
	b       *recording.Builder
	p       recording.Node // a paragraph on the page
}

func newBoardCast(t *testing.T) *boardCast {
	t.Helper()
	c := &boardCast{hub: recording.NewHub(256), b: recording.NewBuilder("rec")}
	bd := board.New(notes.Structural{}, func(r recording.Record, doc int) error { return c.w.Emit(r, doc) })
	c.w = board.NewWriter(recording.Tee(bd, c.hub), nil)
	bd.WithAnnotationCleanup(c.w.ClearAnnotations)
	if err := bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewServer("view", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(bd).WithPointer("tok"))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { c.hub.Close() })
	c.url, c.viewURL, c.bd = srv.URL+"/tok/", srv.URL+"/view/", bd
	c.p = c.b.El("p", []string{"id", "p"}, c.b.Txt("Hello"))
	c.b.Page(recording.SegLive, "http://app/", c.p)
	c.b.Recs[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "server-load", Root: 1}
	c.flush(t)
	return c
}

func (c *boardCast) flush(t *testing.T) {
	t.Helper()
	for _, r := range c.b.Recs {
		if err := c.w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	c.b.Recs = nil
}

func post(t *testing.T, url, contentType, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Couch-Annotation-Generation", "1")
	resp, err := castClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	text, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(text))
}

func noteBody(seg int, node recording.NodeID, color int) string {
	return fmt.Sprintf(`{"viewer":{"id":"ana","name":"Ana"},"seg":%d,"node":%d,"box":[1,2,3,4],"region":[1,2,30,40],"color":%d,"text":"hi"}`, seg, node, color)
}

// A note posted by one viewer reaches another's stream as a rec.
func TestBoardNoteReachesEveryViewer(t *testing.T) {
	c := newBoardCast(t)
	other := openSSE(t, c.url+"events?viewer=bob")
	other.nextRecs(3) // head, seg, doc
	if code, text := post(t, c.url+"note", "application/json", noteBody(1, c.p.ID, 0)); code != http.StatusNoContent {
		t.Fatalf("note: %d %q", code, text)
	}
	r := other.nextRecs(1)[0]
	n, ok := r.(*recording.Note)
	if !ok || n.Author.ID != "ana" || n.At.Sel != "#p" || n.Text != "hi" {
		t.Fatalf("the other viewer got %+v", r)
	}
	if code, _ := post(t, c.url+"stroke", "application/json", `{"viewer":{"id":"ana"},"seg":1,"id":"s1","color":1,"pts":[[1,2,0]]}`); code != http.StatusNoContent {
		t.Fatalf("stroke: %d", code)
	}
	if s, ok := other.nextRecs(1)[0].(*recording.Stroke); !ok || s.ID != "ana.s1" {
		t.Fatalf("stroke on the stream: %+v", s)
	}
	body, _ := json.Marshal(map[string]any{"viewer": map[string]string{"id": "bob"}, "id": n.ID, "state": "resolved"})
	if code, text := post(t, c.url+"note-state", "application/json", string(body)); code != http.StatusNoContent {
		t.Fatalf("resolve: %d %q", code, text)
	}
	if s, ok := other.nextRecs(1)[0].(*recording.NoteState); !ok || s.State != "resolved" {
		t.Fatalf("note state on the stream: %+v", s)
	}
}

func TestBoardAnswers(t *testing.T) {
	c := newBoardCast(t)
	cases := []struct {
		name, route, ctype, body string
		code                     int
		text                     string
	}{
		{"wrong content type", "note", "text/plain", noteBody(1, c.p.ID, 0), 415, ""},
		{"malformed", "note", "application/json", `{"x":1}`, 400, refuseInvalid},
		{"too large", "stroke", "application/json", strings.Repeat(" ", board.MaxStrokeBody+1), 413, ""},
		{"stale segment", "note", "application/json", noteBody(2, c.p.ID, 0), 409, refuseStale},
		{"unknown note", "note-state", "application/json", `{"viewer":{"id":"a"},"id":"nope","state":"resolved"}`, 409, refuseUnknown},
	}
	for _, k := range cases {
		code, text := post(t, c.url+k.route, k.ctype, k.body)
		if code != k.code || (k.text != "" && text != k.text) {
			t.Errorf("%s: %d %q, want %d %q", k.name, code, text, k.code, k.text)
		}
	}
	c.b.Add(&recording.Stop{Why: recording.StopEnd})
	c.flush(t)
	time.Sleep(time.Second) // refill the notes' budget
	if code, text := post(t, c.url+"note", "application/json", noteBody(1, c.p.ID, 0)); code != http.StatusGone || text != refuseEnded {
		t.Errorf("after the end: %d %q", code, text)
	}
}

func TestBoardColorLimitAndRate(t *testing.T) {
	c := newBoardCast(t)
	landed, busy, full := 0, 0, 0
	deadline := time.Now().Add(10 * time.Second)
	for full == 0 && time.Now().Before(deadline) {
		switch code, text := post(t, c.url+"note", "application/json", noteBody(1, c.p.ID, 3)); {
		case code == http.StatusNoContent:
			landed++
		case code == http.StatusTooManyRequests && text == refuseBusy:
			busy++
			time.Sleep(100 * time.Millisecond)
		case code == http.StatusConflict && text == refuseColorFull:
			full++
		default:
			t.Fatalf("note: %d %q", code, text)
		}
	}
	if landed != 8 || full != 1 || busy == 0 {
		t.Errorf("landed %d, refused full %d, refused busy %d: want 8 landed, then full, and the rate to bite", landed, full, busy)
	}
}

func TestBoardRoutesNeedABoard(t *testing.T) {
	srv := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second, nil, nil))
	t.Cleanup(srv.Close)
	for _, route := range []string{"note", "stroke", "note-state", "note-edit"} {
		if code, _ := post(t, srv.URL+"/tok/"+route, "application/json", "{}"); code != http.StatusNotFound {
			t.Errorf("POST %s without a board: %d", route, code)
		}
	}
	c := newBoardCast(t)
	for _, r := range []struct {
		method, path string
		want         int
	}{
		{"GET", "note", 404},
		{"POST", "events", 405},
		{"POST", "", 405},
	} {
		req, _ := http.NewRequest(r.method, c.url+r.path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp, err := castClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != r.want {
			t.Errorf("%s %s: %d, want %d", r.method, r.path, resp.StatusCode, r.want)
		}
	}
	if code, _ := post(t, strings.Replace(c.url, "/tok/", "/other/", 1)+"note", "application/json", noteBody(1, c.p.ID, 0)); code != http.StatusMethodNotAllowed {
		t.Errorf("POST under another token: %d", code)
	}
}

// The notes module's GETs: collab.js always (the terminal viewer's file),
// palette.json only when the cast takes notes, so the viewer mounts the
// module only then.
func TestBoardAssets(t *testing.T) {
	plain := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second, nil, nil))
	t.Cleanup(plain.Close)
	c := newBoardCast(t)
	get := func(url string) (int, string, string) {
		resp, err := castClient.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), string(b)
	}
	if code, _, _ := get(plain.URL + "/tok/palette.json"); code != 404 {
		t.Errorf("palette.json without a board: %d", code)
	}
	code, ctype, body := get(c.url + "palette.json")
	var palette paletteAnswer
	if code != 200 || ctype != "application/json" || json.Unmarshal([]byte(body), &palette) != nil || len(palette.Palette) != 6 {
		t.Errorf("palette.json: %d %s %s", code, ctype, body)
	}
	for _, base := range []string{plain.URL + "/tok/", c.url} {
		if code, ctype, body := get(base + "collab.js"); code != 200 || !strings.HasPrefix(ctype, "text/javascript") || !strings.Contains(body, "export function noteDialog") {
			t.Errorf("collab.js at %s: %d %s", base, code, ctype)
		}
	}
	for _, f := range []string{"annotate.js", "annotate.css"} {
		if code, _, _ := get(c.url + f); code != 200 {
			t.Errorf("%s: %d", f, code)
		}
	}
}

// Every module a viewer script imports is served by serveViewerFile, which
// the live and the replay servers both use: a module only one of them
// served broke the other's whole module graph (#470 met #471 that way).
func TestViewerImportsAreServed(t *testing.T) {
	importRE := regexp.MustCompile(`(?m)^\s*import\s[^;]*?from\s+'\./([A-Za-z0-9_-]+\.js)'`)
	seen, queue := map[string]bool{"viewer.js": true}, []string{"viewer.js"}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		rec := httptest.NewRecorder()
		serveViewerFile(rec, httptest.NewRequest("GET", "/"+name, nil), name)
		if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/javascript") {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Header().Get("Content-Type"))
			continue
		}
		for _, m := range importRE.FindAllStringSubmatch(rec.Body.String(), -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				queue = append(queue, m[1])
			}
		}
	}
	if !seen["annotate.js"] || !seen["collab.js"] || !seen["replay.js"] || !seen["drive.js"] {
		t.Errorf("the walk missed modules: %v", seen)
	}
	// And every stylesheet the page links (#467's drive.css).
	page := httptest.NewRecorder()
	serveViewerFile(page, httptest.NewRequest("GET", "/", nil), "")
	links := regexp.MustCompile(`<link rel="stylesheet" href="([A-Za-z0-9_-]+\.css)">`).FindAllStringSubmatch(page.Body.String(), -1)
	var sheets []string
	for _, m := range links {
		sheets = append(sheets, m[1])
		rec := httptest.NewRecorder()
		serveViewerFile(rec, httptest.NewRequest("GET", "/"+m[1], nil), m[1])
		if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/css") {
			t.Errorf("%s: %d %s", m[1], rec.Code, rec.Header().Get("Content-Type"))
		}
	}
	if !slices.Contains(sheets, "drive.css") || !slices.Contains(sheets, "annotate.css") {
		t.Errorf("the page's stylesheets: %v", sheets)
	}
}

// While the cast is paused every board route refuses with one sentence
// (pair#499), as the drive routes refuse while not live.
func TestBoardRoutesRefuseWhilePaused(t *testing.T) {
	c := newBoardCast(t)
	c.b.Add(&recording.Stop{Why: recording.StopPause})
	c.flush(t)
	for route, body := range map[string]string{
		"stroke":     `{"viewer":{"id":"ana"},"seg":1,"id":"s1","color":1,"pts":[[1,2,0]]}`,
		"note":       noteBody(1, c.p.ID, 0),
		"note-state": `{"viewer":{"id":"ana"},"id":"n1","state":"resolved"}`,
		"note-edit":  `{"seg":1,"id":"n1","text":"edited"}`,
	} {
		if code, text := post(t, c.url+route, "application/json", body); code != http.StatusConflict || text != refusePaused {
			t.Errorf("%s while paused: %d %q", route, code, text)
		}
	}
}

// pair#506: the view-only link may watch but not annotate. The refusal comes
// before admission, so a view-only link spends no body read, in-flight slot
// or rate budget: an over-limit body is refused as view-only (403), not as
// too large (413).
func TestBoardRoutesRefuseTheViewOnlyLink(t *testing.T) {
	c := newBoardCast(t)
	huge := strings.Repeat("x", board.MaxNoteBody+1)
	for _, route := range []string{"stroke", "note", "note-state", "note-edit"} {
		for _, body := range []string{noteBody(1, c.p.ID, 0), huge} {
			if code, text := post(t, c.viewURL+route, "application/json", body); code != http.StatusForbidden || text != refuseViewOnly {
				t.Errorf("POST %s under the view-only link: %d %q", route, code, text)
			}
		}
	}
	if n := len(c.bd.Live()); n != 0 {
		t.Fatalf("a view-only post reached the board: %d notes", n)
	}
	if code, _ := post(t, c.url+"note", "application/json", noteBody(1, c.p.ID, 0)); code != http.StatusNoContent {
		t.Fatalf("a note under the pointer link: %d", code)
	}
	if n := len(c.bd.Live()); n != 1 {
		t.Fatalf("the pointer link's note: %d notes", n)
	}
}

// palette.json answers both links with the swatches' colors, and says which
// may annotate, so a view-only viewer mounts strokes and notes read-only.
func TestPaletteSaysWhetherTheLinkAnnotates(t *testing.T) {
	c := newBoardCast(t)
	for url, want := range map[string]bool{c.url: true, c.viewURL: false} {
		resp, err := http.Get(url + "palette.json")
		if err != nil {
			t.Fatal(err)
		}
		var got paletteAnswer
		err = json.NewDecoder(resp.Body).Decode(&got)
		resp.Body.Close()
		if err != nil || len(got.Palette) != 6 || got.Annotate != want {
			t.Errorf("%spalette.json: %+v %v, want annotate %v", url, got, err, want)
		}
	}
}
