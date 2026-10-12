package webcast

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// boardCast is a cast with a board as #458 wires it: the recorder's records
// and the board's go through one writer into the hub viewers stream from and
// into the board, which follows them.
type boardCast struct {
	url string
	hub *recording.Hub
	w   *board.Writer
	b   *recording.Builder
	p   recording.Node // a paragraph on the page
}

func newBoardCast(t *testing.T) *boardCast {
	t.Helper()
	c := &boardCast{hub: recording.NewHub(256), b: recording.NewBuilder("rec")}
	bd := board.New(notes.Structural{}, func(r recording.Record, doc int) error { return c.w.Emit(r, doc) })
	c.w = board.NewWriter(recording.Tee(bd, c.hub), nil)
	srv := httptest.NewServer(NewServer("tok", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(bd))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { c.hub.Close() })
	c.url = srv.URL + "/tok/"
	c.p = c.b.El("p", []string{"id", "p"}, c.b.Txt("Hello"))
	c.b.Page(recording.SegLive, "http://app/", c.p)
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
	resp, err := http.Post(url, contentType, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	text, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(text))
}

func noteBody(seg int, node recording.NodeID, color int) string {
	return fmt.Sprintf(`{"viewer":{"id":"ana","name":"Ana"},"seg":%d,"node":%d,"box":[1,2,3,4],"color":%d,"text":"hi"}`, seg, node, color)
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
	for _, route := range []string{"note", "stroke", "note-state"} {
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
		resp, err := http.DefaultClient.Do(req)
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
