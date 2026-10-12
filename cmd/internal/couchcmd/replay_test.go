package couchcmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestParseReplayArgs(t *testing.T) {
	for _, c := range []struct {
		args []string
		want replayArgs
		err  string
	}{
		{nil, replayArgs{}, ""},
		{[]string{"rec-1"}, replayArgs{target: "rec-1"}, ""},
		{[]string{"rec-1", "--at", "1530"}, replayArgs{target: "rec-1", at: 1530, hasAt: true}, ""},
		{[]string{"--at", "0", "./dir"}, replayArgs{target: "./dir", hasAt: true}, ""},
		{[]string{"--at", "5"}, replayArgs{}, "--at needs a recording"},
		{[]string{"a", "--at"}, replayArgs{}, "--at needs a time"},
		{[]string{"a", "--at", "-1"}, replayArgs{}, "a time in ms"},
		{[]string{"a", "--at", "1.5"}, replayArgs{}, "a time in ms"},
		{[]string{"a", "b"}, replayArgs{}, "one recording"},
		{[]string{"--json"}, replayArgs{}, "unknown flag"},
	} {
		got, err := parseReplayArgs(c.args)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%q: error %v, want %q", c.args, err, c.err)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%q: %+v %v, want %+v", c.args, got, err, c.want)
		}
	}
	inv, err := ParseCLI([]string{replayFlag, "rec", "--at", "9"}, couchcore.Operations())
	if err != nil || inv.kind != cliWebReplay || len(inv.args) != 3 {
		t.Errorf("ParseCLI: %+v %v", inv, err)
	}
	if _, err := ParseCLI([]string{replayFlag, "--at", "9"}, couchcore.Operations()); err == nil {
		t.Error("ParseCLI accepted --at without a recording")
	}
}

func TestResolveReplay(t *testing.T) {
	store := t.TempDir()
	recs := filepath.Join(store, "webcast", "recordings")
	os.MkdirAll(filepath.Join(recs, "rec-1"), 0o755)
	other := filepath.Join(t.TempDir(), "elsewhere")
	os.Mkdir(other, 0o755)
	other, _ = filepath.EvalSymlinks(other) // macOS's temp dir is under a link
	link := filepath.Join(t.TempDir(), "linked")
	os.Symlink(other, link)
	os.WriteFile(filepath.Join(recs, "plain"), []byte("x"), 0o644)
	for _, c := range []struct{ target, root, id, err string }{
		{"", recs, "", ""},
		{"rec-1", recs, "rec-1", ""},
		{other, filepath.Dir(other), "elsewhere", ""},
		// A link resolves to the directory the server will serve.
		{link, filepath.Dir(other), "elsewhere", ""},
		{"missing", "", "", "no recording"},
		{"plain", "", "", "no recording"},
		{"../x", "", "", "not a recording directory"},
		{filepath.Join(t.TempDir(), "bad name"), "", "", "not a recording directory"},
	} {
		root, id, err := resolveReplay(c.target, recs)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%q: error %v, want %q", c.target, err, c.err)
			}
			continue
		}
		if err != nil || root != c.root || id != c.id {
			t.Errorf("%q: %q %q %v", c.target, root, id, err)
		}
	}
}

// lineWriter hands over the first line written to it.
type lineWriter struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	line chan string
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	if i := bytes.IndexByte(w.buf.Bytes(), '\n'); i >= 0 && w.line != nil {
		w.line <- string(w.buf.Bytes()[:i])
		w.line = nil
	}
	return len(p), nil
}

// The CLI serves a recording on loopback, prints its #t= link, and stops
// when its context ends.
func TestRunReplayServesTheLink(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "segments")
	if err := os.CopyFS(dir, os.DirFS(recording.Fixture("segments"))); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	out := &lineWriter{line: make(chan string, 1)}
	done := make(chan int, 1)
	go func() { done <- runReplay(ctx, []string{dir, "--at", "5500"}, "", out, io.Discard) }()
	var link string
	select {
	case link = <-out.line:
	case code := <-done:
		t.Fatalf("exited %d before printing a link", code)
	case <-time.After(10 * time.Second):
		t.Fatal("no link")
	}
	if !regexp.MustCompile(`^http://127\.0\.0\.1:\d+/[A-Za-z0-9_-]+/r/segments/#t=5500$`).MatchString(link) {
		t.Fatalf("link %q", link)
	}
	res, err := http.Get(strings.TrimSuffix(link, "#t=5500") + "records")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.HasPrefix(b, []byte(`{"t":0,"k":"head"`)) {
		t.Fatalf("records %d %.60s", res.StatusCode, b)
	}
	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not stop")
	}
}

func TestRunReplayRefusals(t *testing.T) {
	var errs bytes.Buffer
	if code := runReplay(context.Background(), nil, "", io.Discard, &errs); code != 2 || !strings.Contains(errs.String(), "no store") {
		t.Errorf("no store: %d %s", code, errs.String())
	}
	errs.Reset()
	if code := runReplay(context.Background(), []string{"nope"}, t.TempDir(), io.Discard, &errs); code != 1 || !strings.Contains(errs.String(), "no recording") {
		t.Errorf("missing: %d %s", code, errs.String())
	}
}
