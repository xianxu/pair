package webcast

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// replayRoot is a recordings root holding copies of the named fixtures.
func replayRoot(t *testing.T, names ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, n := range names {
		if err := os.CopyFS(filepath.Join(root, n), os.DirFS(recording.Fixture(n))); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func replayGet(t *testing.T, s http.Handler, method, path string) (*http.Response, string) {
	t.Helper()
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest(method, path, nil))
	res := rr.Result()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func TestReplayServerRoutes(t *testing.T) {
	root := replayRoot(t, "segments", "blog")
	os.WriteFile(filepath.Join(root, "stray.txt"), []byte("not a recording"), 0o644)
	os.Mkdir(filepath.Join(root, "empty"), 0o755)
	s := NewReplayServer("tok", root)

	res, body := replayGet(t, s, http.MethodGet, "/tok/")
	if res.StatusCode != 200 || !strings.Contains(body, `href="r/segments/"`) || !strings.Contains(body, `href="r/blog/"`) || strings.Contains(body, "empty") || strings.Contains(body, "stray") {
		t.Fatalf("listing %d:\n%s", res.StatusCode, body)
	}
	if !strings.Contains(res.Header.Get("Content-Security-Policy"), "script-src 'self'") {
		t.Errorf("listing CSP: %q", res.Header.Get("Content-Security-Policy"))
	}

	res, body = replayGet(t, s, http.MethodGet, "/tok/r/segments/")
	if res.StatusCode != 200 || !strings.Contains(body, `id="timeline"`) || !strings.Contains(res.Header.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatalf("viewer page %d", res.StatusCode)
	}
	for _, m := range []string{"viewer.js", "replay.js", "model.js", "render.js", "css.js", "viewer.css", "replay.css"} {
		if res, _ := replayGet(t, s, http.MethodGet, "/tok/r/segments/"+m); res.StatusCode != 200 {
			t.Errorf("%s: %d", m, res.StatusCode)
		}
	}

	rec, err := recording.ReadDir(recording.Fixture("blog"))
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	for _, x := range rec.Records {
		line, _ := recording.Encode(x)
		want.Write(append(line, '\n'))
	}
	if res, body := replayGet(t, s, http.MethodGet, "/tok/r/blog/records"); res.StatusCode != 200 || body != want.String() {
		t.Fatalf("records %d: %d bytes, want %d", res.StatusCode, len(body), want.Len())
	}
	served := 0
	for _, x := range rec.Records {
		if a, ok := x.(*recording.Asset); ok && strings.HasPrefix(a.MIME, "image/") {
			if res, _ := replayGet(t, s, http.MethodGet, "/tok/r/blog/assets/"+a.Hash); res.StatusCode != 200 {
				t.Errorf("asset %s: %d", a.URL, res.StatusCode)
			}
			served++
		}
	}
	if served == 0 {
		t.Fatal("blog served no image: the check saw nothing")
	}
}

// Assets are served before the records are asked for, too (a reload that
// hits the cache for records).
func TestReplayServerAssetsWithoutRecords(t *testing.T) {
	s := NewReplayServer("tok", replayRoot(t, "blog"))
	rec, _ := recording.ReadDir(recording.Fixture("blog"))
	for _, x := range rec.Records {
		if a, ok := x.(*recording.Asset); ok && strings.HasPrefix(a.MIME, "image/") {
			if res, _ := replayGet(t, s, http.MethodGet, "/tok/r/blog/assets/"+a.Hash); res.StatusCode != 200 {
				t.Fatalf("asset %s: %d", a.URL, res.StatusCode)
			}
			return
		}
	}
	t.Fatal("no image asset in blog")
}

// Only recordings directly under the root are served.
func TestReplayServerConfinesToRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "recordings")
	os.Mkdir(root, 0o755)
	if err := os.CopyFS(filepath.Join(outer, "secret"), os.DirFS(recording.Fixture("segments"))); err != nil {
		t.Fatal(err)
	}
	os.Symlink(filepath.Join(outer, "secret"), filepath.Join(root, "link"))
	os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o644)
	if err := os.CopyFS(filepath.Join(root, "ok"), os.DirFS(recording.Fixture("segments"))); err != nil {
		t.Fatal(err)
	}
	s := NewReplayServer("tok", root)
	for _, p := range []string{
		"/tok/r/../secret/records", "/tok/r/%2e%2e/records", "/tok/r/link/records", "/tok/r/file/records",
		"/tok/r/missing/records", "/tok/r/ok", "/tok/r/ok/nope.js", "/tok/r/.ok/records", "/tok/x/ok/records",
		"/other/r/ok/records", "/tok/r//records",
	} {
		if res, _ := replayGet(t, s, http.MethodGet, p); res.StatusCode != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", p, res.StatusCode)
		}
	}
	if res, _ := replayGet(t, s, http.MethodGet, "/tok/r/ok/records"); res.StatusCode != 200 {
		t.Errorf("ok: %d", res.StatusCode)
	}
	if res, _ := replayGet(t, s, http.MethodPost, "/tok/r/ok/records"); res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", res.StatusCode)
	}
}

func TestReplayServerListsNothingYet(t *testing.T) {
	s := NewReplayServer("tok", filepath.Join(t.TempDir(), "none"))
	if res, body := replayGet(t, s, http.MethodGet, "/tok/"); res.StatusCode != 200 || !strings.Contains(body, "No recordings yet") {
		t.Fatalf("%d %s", res.StatusCode, body)
	}
}

func TestListRecordingsReadsHeadAndEnd(t *testing.T) {
	got, err := ListRecordings(replayRoot(t, "segments", "driven"))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Listed{}
	for _, l := range got {
		byID[l.ID] = l
	}
	if l := byID["segments"]; l.Duration.Milliseconds() != 9000 || l.Notes != 2 || l.Started == "" {
		t.Errorf("segments: %+v", l)
	}
	if l := byID["driven"]; l.Duration.Milliseconds() != 9500 {
		t.Errorf("driven: %+v", l)
	}
}

func TestReplayLink(t *testing.T) {
	if got := ReplayLink("http://127.0.0.1:5/tok/", "rec-1", 1530); got != "http://127.0.0.1:5/tok/r/rec-1/#t=1530" {
		t.Errorf("%s", got)
	}
	if got := ReplayLink("http://h/tok", "a", -3); got != "http://h/tok/r/a/#t=0" {
		t.Errorf("%s", got)
	}
}

// Every id the recorder names a recording with (#458) is one the replay
// serves.
func TestReplayAcceptsTheRecordersIDs(t *testing.T) {
	for _, at := range []time.Time{time.Now(), time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)} {
		if id := recordingID(at); !ValidRecordingID(id) {
			t.Errorf("the recorder's id %q is not a replayable id", id)
		}
	}
}
