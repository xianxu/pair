package webcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// emittingBrowser is a FakeBrowser whose recording's one writer is the
// reference board.Writer over the sink the cast opened, as the recorder's
// queue is for Chrome.
type emittingBrowser struct {
	*FakeBrowser
	w *board.Writer
}

func (e *emittingBrowser) ClearAnnotations(ctx context.Context, id uint64) error {
	if e.w == nil {
		return nil
	}
	return e.w.ClearAnnotations(ctx, id)
}

func (e *emittingBrowser) Emit(r recording.Record, doc int) error { return e.w.Emit(r, doc) }

// recordedBoardCast is a recorded cast with a board, its viewer routes served, and
// the recording's one writer reachable once it opens.
type recordedBoardCast struct {
	c    *Cast
	eb   *emittingBrowser
	ro   *RecordOptions
	dir  string
	base string
}

func startRecordedBoardCast(t *testing.T) *recordedBoardCast {
	t.Helper()
	bc := &recordedBoardCast{dir: t.TempDir(), eb: &emittingBrowser{FakeBrowser: NewFakeBrowser()}}
	c, err := StartCast(context.Background(), CastConfig{
		Launch:        func(_ context.Context, r *RecordOptions) (Browser, error) { bc.ro = r; return bc.eb, nil },
		Host:          broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: 2 * time.Second}),
		Ping:          time.Second,
		RecordingsDir: bc.dir,
		NewLive: func() (recording.Sink, recording.Live) {
			h := recording.NewHub(256)
			return h, h
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.End)
	srv := httptest.NewServer(c.server)
	t.Cleanup(srv.Close)
	bc.c, bc.base = c, srv.URL+"/"+c.pointerToken+"/" // the pointer link annotates (pair#506)
	return bc
}

// open opens the recording (the first go-live) and writes a live page with
// one element, which it returns.
func (bc *recordedBoardCast) open(t *testing.T) recording.Node {
	t.Helper()
	o, err := bc.ro.Out()
	if err != nil {
		t.Fatal(err)
	}
	bc.eb.w = board.NewWriter(o.Sink, nil)
	b := recording.NewBuilder("rec")
	p := b.El("p", []string{"id", "p"}, b.Txt("Hello"))
	b.Page(recording.SegLive, "http://app/", p)
	b.Recs[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "fixture-load", Root: 1}
	for _, r := range b.Recs {
		if err := bc.eb.w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// A recorded cast takes viewers' notes (#470): the board follows the
// recording through its Tee, a posted note reaches the directory and the
// viewers' stream, Couch sees it in Notes, and the operator resolves it.
func TestCastBoardRecordsNotes(t *testing.T) {
	bc := startRecordedBoardCast(t)
	c, eb, dir, base := bc.c, bc.eb, bc.dir, bc.base
	if err := c.SetCapability(context.Background(), CapabilityAnnotation, true); err != nil {
		t.Fatal(err)
	}
	if code, text := post(t, base+"note", "application/json", noteBody(1, 5, 0)); code != http.StatusConflict || text != refuseNotLive {
		t.Fatalf("a note before the recording opens: %d %q", code, text)
	}

	p := bc.open(t)
	notes, cancel := c.Notes()
	defer cancel()
	<-notes
	if code, text := post(t, base+"note", "application/json", noteBody(1, p.ID, 2)); code != http.StatusNoContent {
		t.Fatalf("note: %d %q", code, text)
	}
	live := <-notes
	if len(live) != 1 || live[0].Text != "hi" || live[0].Color != 2 || live[0].Author.ID != "ana" {
		t.Fatalf("Couch's notes %+v", live)
	}
	if err := c.ResolveNote(live[0].ID); err != nil {
		t.Fatal(err)
	}
	if live := <-notes; len(live) != 0 {
		t.Fatalf("after the operator resolved it: %+v", live)
	}
	eb.w.Close() // as the recorder does when the recording ends: flushes the directory
	if _, open := <-notes; open {
		t.Fatalf("closing the recording closes the board")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("recordings %v", entries)
	}
	body, _ := os.ReadFile(filepath.Join(dir, entries[0].Name(), recording.FileNotes))
	if lines := strings.Count(string(body), "\n"); lines != 2 || !strings.Contains(string(body), `"state":"resolved"`) {
		t.Fatalf("notes.jsonl:\n%s", body)
	}
}

// pair#493: a live cast's notes and strokes are on disk while it is still
// recording, not only once it stops. The operator's 12:01 cast held its note
// and strokes in the recorder's buffers until the cast ended at 14:29, so the
// live replay and `couch --broadcast-web-query DIR notes` saw none. Nothing
// here closes the recording: the reads below are what a replay and a query
// make mid-cast.
func TestCastBoardNotesAndStrokesReachDiskWhileLive(t *testing.T) {
	bc := startRecordedBoardCast(t)
	c, eb, dir, base := bc.c, bc.eb, bc.dir, bc.base
	if err := c.SetCapability(context.Background(), CapabilityAnnotation, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { eb.w.Close() })
	p := bc.open(t)
	notes, cancel := c.Notes()
	defer cancel()
	<-notes
	if code, text := post(t, base+"note", "application/json", noteBody(1, p.ID, 2)); code != http.StatusNoContent {
		t.Fatalf("note: %d %q", code, text)
	}
	<-notes // the board took it
	if code, text := post(t, base+"stroke", "application/json", `{"viewer":{"id":"ana"},"seg":1,"id":"s1","color":1,"pts":[[1,2,0],[3,4,16]]}`); code != http.StatusNoContent {
		t.Fatalf("stroke: %d %q", code, text)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("recordings %v", entries)
	}
	recDir := filepath.Join(dir, entries[0].Name())
	// Replay's reader, mid-cast.
	rec, err := recording.ReadDir(recDir)
	if err != nil {
		t.Fatal(err)
	}
	var note, stroke bool
	for _, r := range rec.Records {
		switch r := r.(type) {
		case *recording.Note:
			note = note || r.Text == "hi"
		case *recording.Stroke:
			stroke = true
		}
	}
	if !note || !stroke {
		t.Fatalf("while live, the recording holds note=%v stroke=%v (%d records)", note, stroke, len(rec.Records))
	}
	// The query an agent runs, mid-cast.
	var out, errout strings.Builder
	if code := query.Run([]string{recDir, "notes"}, nil, &out, &errout); code != 0 || !strings.Contains(out.String(), "hi") {
		t.Fatalf("notes query while live: exit %d\n%s%s", code, out.String(), errout.String())
	}
}
