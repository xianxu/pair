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
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// emittingBrowser is a FakeBrowser whose recording's one writer is the
// reference board.Writer over the sink the cast opened, as the recorder's
// queue is for Chrome.
type emittingBrowser struct {
	*FakeBrowser
	w *board.Writer
}

func (e *emittingBrowser) Emit(r recording.Record, doc int) error { return e.w.Emit(r, doc) }

// A recorded cast takes viewers' notes (#470): the board follows the
// recording through its Tee, a posted note reaches the directory and the
// viewers' stream, Couch sees it in Notes, and the operator resolves it.
func TestCastBoardRecordsNotes(t *testing.T) {
	dir := t.TempDir()
	eb := &emittingBrowser{FakeBrowser: NewFakeBrowser()}
	var ro *RecordOptions
	c, err := StartCast(context.Background(), CastConfig{
		Launch:        func(_ context.Context, r *RecordOptions) (Browser, error) { ro = r; return eb, nil },
		Host:          broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: 2 * time.Second}),
		Ping:          time.Second,
		RecordingsDir: dir,
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
	base := srv.URL + "/" + c.token + "/"
	if code, text := post(t, base+"note", "application/json", noteBody(1, 5, 0)); code != http.StatusConflict || text != refuseNotLive {
		t.Fatalf("a note before the recording opens: %d %q", code, text)
	}

	o, err := ro.Out()
	if err != nil {
		t.Fatal(err)
	}
	eb.w = board.NewWriter(o.Sink, nil)
	b := recording.NewBuilder("rec")
	p := b.El("p", []string{"id", "p"}, b.Txt("Hello"))
	b.Page(recording.SegLive, "http://app/", p)
	for _, r := range b.Recs {
		if err := eb.w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
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
