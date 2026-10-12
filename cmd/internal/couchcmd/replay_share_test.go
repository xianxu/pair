package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// noCouch is a broker nobody answers.
func noCouch(context.Context, string, any, any) error { return errors.New("dial unix: no such file") }

// testClock advances only when slept.
func testClock() slotPollClock {
	now := time.Unix(1_000_000, 0)
	return slotPollClock{now: func() time.Time { return now }, sleep: func(d time.Duration) { now = now.Add(d) }}
}

// fakeBroker answers replay requests from a script: each request takes the
// next answer, the last repeating. It records the ops asked.
type fakeBroker struct {
	answers []couchmessage.Response
	ops     []string
}

func (b *fakeBroker) call(_ context.Context, _ string, req, res any) error {
	r := req.(couchmessage.Request)
	if err := couchmessage.ValidateRequest(r); err != nil {
		return err
	}
	b.ops = append(b.ops, r.Op+" "+r.Target)
	next := b.answers[0]
	if len(b.answers) > 1 {
		b.answers = b.answers[1:]
	}
	*res.(*couchmessage.Response) = next
	return nil
}

func replayAnswer(state, base string) couchmessage.Response {
	return couchmessage.Response{Code: "ok", Replay: &couchmessage.ReplayShare{State: state, Base: base}}
}

// replayStore is a Couch store holding recording rec-1.
func replayStore(t *testing.T) string {
	t.Helper()
	store := t.TempDir()
	os.MkdirAll(filepath.Join(recordingsDir(store), "rec-1"), 0o755)
	return store
}

// The running Couch shares the recording: the CLI asks once, polls the
// status until it's ready, prints the public link and returns.
func TestRunReplaySharesThroughCouch(t *testing.T) {
	store := replayStore(t)
	b := &fakeBroker{answers: []couchmessage.Response{
		replayAnswer("opening", ""), replayAnswer("opening", ""), replayAnswer("ready", "https://cast.example/tok"),
	}}
	var out, errs bytes.Buffer
	if code := runReplay(context.Background(), []string{"rec-1", "--at", "1530"}, store, b.call, testClock(), &out, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	if got := strings.TrimSpace(out.String()); got != "https://cast.example/tok/r/rec-1/#t=1530" {
		t.Fatalf("link %q", got)
	}
	if want := []string{"replay-share rec-1", "replay-status rec-1", "replay-status rec-1"}; strings.Join(b.ops, ",") != strings.Join(want, ",") {
		t.Fatalf("ops %q", b.ops)
	}
	// A DIR naming the same recording, through a link, is shared too.
	link := filepath.Join(t.TempDir(), "rec-1")
	os.Symlink(filepath.Join(recordingsDir(store), "rec-1"), link)
	b = &fakeBroker{answers: []couchmessage.Response{replayAnswer("ready", "https://cast.example/tok")}}
	out.Reset()
	if code := runReplay(context.Background(), []string{link}, store, b.call, testClock(), &out, &errs); code != 0 || !strings.Contains(out.String(), "/r/rec-1/#t=0") {
		t.Fatalf("DIR: %d %q %s", code, out.String(), errs.String())
	}
}

// Each way sharing fails: what the CLI prints, and whether it serves on
// loopback instead (only when sharing was never possible).
func TestShareReplayFailures(t *testing.T) {
	store := replayStore(t)
	recs := recordingsDir(store)
	for _, c := range []struct {
		name    string
		root    string
		call    messageCall
		wantErr error
		msg     string
	}{
		{"no Couch", recs, noCouch, errNoCouch, ""},
		{"old Couch", recs, (&fakeBroker{answers: []couchmessage.Response{{Code: "invalid-request", Error: "operation requires the calling conversation identity"}}}).call, errCouchPredatesReplay, ""},
		{"outside the store", t.TempDir(), noCouch, errNotInStore, ""},
		{"failed", recs, (&fakeBroker{answers: []couchmessage.Response{replayAnswer("opening", ""), {Code: "ok", Replay: &couchmessage.ReplayShare{State: "failed", Error: "broadcast: the tunnel exited"}}}}).call, nil, "failed: broadcast: the tunnel exited"},
		{"still opening", recs, (&fakeBroker{answers: []couchmessage.Response{replayAnswer("opening", "")}}).call, nil, "still opening in Couch; run this again"},
		{"closed", recs, (&fakeBroker{answers: []couchmessage.Response{replayAnswer("closed", "")}}).call, nil, "closed"},
	} {
		_, err := shareReplay(context.Background(), store, c.root, "rec-1", 0, c.call, testClock())
		if err == nil || (c.wantErr != nil && !errors.Is(err, c.wantErr)) || !strings.Contains(err.Error(), c.msg) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// With no Couch to share it, the CLI says so and serves on loopback, as
// before #477; --local does without asking.
func TestRunReplayFallsBackToLoopback(t *testing.T) {
	store := replayStore(t)
	for _, c := range []struct {
		args []string
		call messageCall
		note string
	}{
		{[]string{"rec-1"}, noCouch, "No running Couch"},
		{[]string{"rec-1", "--local"}, func(context.Context, string, any, any) error { t.Error("--local asked Couch"); return nil }, ""},
	} {
		ctx, cancel := context.WithCancel(context.Background())
		lines := make(chan string, 1)
		out := &lineWriter{line: lines}
		var errs syncBuffer
		done := make(chan int, 1)
		go func() { done <- runReplay(ctx, c.args, store, c.call, testClock(), out, &errs) }()
		select {
		case link := <-lines:
			if !strings.HasPrefix(link, "http://127.0.0.1:") || !strings.HasSuffix(link, "/r/rec-1/#t=0") {
				t.Errorf("%q: link %q", c.args, link)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%q: no link", c.args)
		}
		cancel()
		<-done
		if !strings.Contains(errs.String(), c.note) || !strings.Contains(errs.String(), "this machine only") {
			t.Errorf("%q: stderr %q", c.args, errs.String())
		}
	}
}

// The query CLI's link verb shares through the running Couch and refuses,
// pointing at the replay CLI, when it can't: it never serves itself.
func TestQueryLink(t *testing.T) {
	store := replayStore(t)
	dir := filepath.Join(recordingsDir(store), "rec-1")
	storeFn := func() (string, error) { return store, nil }
	b := &fakeBroker{answers: []couchmessage.Response{replayAnswer("opening", ""), replayAnswer("ready", "https://cast.example/tok")}}
	var out, errs bytes.Buffer
	if code := query.Run([]string{dir, "link", "--at", "250"}, replayLinker(storeFn, b.call, testClock()), &out, &errs); code != 0 || strings.TrimSpace(out.String()) != "https://cast.example/tok/r/rec-1/#t=250" {
		t.Fatalf("%d %q %s", code, out.String(), errs.String())
	}
	out.Reset()
	if code := query.Run([]string{dir, "link"}, replayLinker(storeFn, noCouch, testClock()), &out, &errs); code != 1 || out.Len() != 0 ||
		!strings.Contains(errs.String(), "no running Couch") || !strings.Contains(errs.String(), replayFlag+" "+dir) {
		t.Fatalf("no Couch: %d %q %s", code, out.String(), errs.String())
	}
	errs.Reset()
	if code := query.Run([]string{dir, "link", "--json"}, replayLinker(storeFn, noCouch, testClock()), io.Discard, &errs); code != 2 {
		t.Fatalf("link --json: %d %s", code, errs.String())
	}
}

// syncBuffer is a bytes.Buffer safe to write from the CLI's goroutine
// while the test reads it.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// The whole chain in one process: the CLI's share-and-poll, through the
// broker's validation and handler (JSON both ways, as the socket carries
// it), to the console's replay shares on a Host over a fake tunnel. The
// link it prints plays the recording.
func TestReplayShareThroughTheBroker(t *testing.T) {
	store := t.TempDir()
	if err := os.CopyFS(filepath.Join(recordingsDir(store), "segments"), os.DirFS(recording.Fixture("segments"))); err != nil {
		t.Fatal(err)
	}
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{OpenDelay: 30 * time.Millisecond}, ProbeTimeout: 2 * time.Second})
	shares := webcast.NewReplayShares(host, recordingsDir(store))
	defer shares.Close()
	console := testConsole()
	console.SetReplayShares(shares)
	s := &messageService{}
	call := func(ctx context.Context, _ string, req, res any) error {
		raw, _ := json.Marshal(req)
		var r couchmessage.Request
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		raw, _ = json.Marshal(s.handle(ctx, r))
		return json.Unmarshal(raw, res)
	}
	if _, err := shareReplay(context.Background(), store, recordingsDir(store), "segments", 0, call, testClock()); !errors.Is(err, errCouchPredatesReplay) {
		t.Fatalf("no provider installed: %v", err)
	}
	s.SetReplay(console.ReplayShare)
	clock := slotPollClock{now: time.Now, sleep: time.Sleep}
	link, err := shareReplay(context.Background(), store, recordingsDir(store), "segments", 4200, call, clock)
	if err != nil || !strings.HasSuffix(link.url, "/r/segments/#t=4200") || link.local {
		t.Fatalf("%+v %v", link, err)
	}
	res, err := http.Get(strings.TrimSuffix(link.url, "#t=4200") + "records")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.HasPrefix(body, []byte(`{"t":0,"k":"head"`)) {
		t.Fatalf("records %d %.60s", res.StatusCode, body)
	}
	if _, err := shareReplay(context.Background(), store, recordingsDir(store), "nope", 0, call, clock); err == nil || !strings.Contains(err.Error(), "no such recording") {
		t.Fatalf("unknown recording: %v", err)
	}
	// A Couch with no store shares nothing, and says why.
	bare := testConsole()
	s.SetReplay(bare.ReplayShare)
	if _, err := shareReplay(context.Background(), store, recordingsDir(store), "segments", 0, call, clock); err == nil || !strings.Contains(err.Error(), "no recordings store") {
		t.Fatalf("no store: %v", err)
	}
}
