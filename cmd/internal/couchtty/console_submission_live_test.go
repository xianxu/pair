package couchtty

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// pair#849: the real manager over a real cast, with the recorder's default
// screenshot cadence. Unchanged notes, shown in the acknowledged preview, are
// admitted by the chord's first prepare.
func TestSubmissionManagerLivePreparesUnchangedNotes(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>notes</title><style>body{margin:0}section{position:absolute;left:80px;top:100px;width:400px;height:200px;background:#eee}</style><section>Public transcript</section>`)
	}))
	defer page.Close()
	bin, err := webcast.FindChrome(os.Getenv, exec.LookPath, func(p string) bool { _, e := os.Stat(p); return e == nil })
	if err != nil {
		t.Fatal(err)
	}
	var chrome *webcast.Chrome
	cast, err := webcast.StartCast(ctx, webcast.CastConfig{RecordingsDir: filepath.Join(t.TempDir(), "recordings"), Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}}), Logf: t.Logf,
		Launch: func(ctx context.Context, opts *webcast.RecordOptions) (webcast.Browser, error) {
			var e error
			chrome, e = webcast.Launch(ctx, webcast.LaunchOptions{Binary: bin, Headless: true, ProfileDir: t.TempDir(), StartURL: page.URL, Record: opts})
			return chrome, e
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cast.End()
	wait := func(what string, condition func() bool) {
		t.Helper()
		for !condition() {
			select {
			case <-ctx.Done():
				t.Fatalf("waiting for %s: %v", what, ctx.Err())
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
	if err = chrome.SetLive(true); err != nil {
		t.Fatal(err)
	}
	// The recorded document gives the anchors' node IDs.
	var doc *recording.Doc
	var anchors []recording.Node
	var walk func(recording.Node)
	walk = func(n recording.Node) {
		if n.Y == "el" && (n.N == "section" || n.N == "form") {
			anchors = append(anchors, n)
		}
		for _, c := range n.C {
			walk(c)
		}
	}
	wait("recorded document", func() bool {
		f, e := os.Open(filepath.Join(cast.RecordingPath(), "events.jsonl"))
		if e != nil {
			return false
		}
		defer f.Close()
		lines := bufio.NewScanner(f)
		lines.Buffer(nil, 1<<24)
		for lines.Scan() {
			if r, e := recording.Decode(lines.Bytes()); e == nil {
				if d, ok := r.(*recording.Doc); ok {
					doc = d
				}
			}
		}
		if doc == nil {
			return false
		}
		anchors = nil
		walk(doc.Tree)
		return len(anchors) > 0
	})
	wait("private console host", func() bool { _, _, e := cast.SubmissionPreviewHost(ctx); return e == nil })
	selected, err := cast.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	want := 4
	for i := 0; i < want; i++ {
		a := anchors[i%len(anchors)]
		box := recording.Box{80, 100, 400, 200}
		if err = chrome.Emit(&recording.Note{ID: fmt.Sprintf("note-%d", i), Owner: "operator", Author: recording.Author{ID: "operator", Name: "Operator", Color: "#ffaf00"}, Text: fmt.Sprintf("note %d", i),
			At: recording.NoteAt{Tag: a.N, Route: page.URL + "/", Node: a.ID, Kind: recording.AnchorElement, Box: &box}}, selected.Document); err != nil {
			t.Fatal(err)
		}
	}
	wait("notes on the board", func() bool { s, e := cast.SubmissionSelection(); return e == nil && len(s.Notes) == want })

	m, _, target := newSubmissionManagerTest(t)
	m.current = func() (submissionCast, error) { return cast, nil }
	m.startSubmissionPreview(cast)
	wait("acknowledged preview", func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.previewState.Phase() == core.PreviewShown
	})
	if status, err := m.request(ctx, "prepare", "press", target); err != nil || status.State != "preparing" {
		t.Fatalf("unchanged notes refused: %+v %v", status, err)
	}
}
