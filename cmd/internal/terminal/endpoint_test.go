package terminal

import (
	"context"
	"errors"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"os"
	"sync"
	"testing"
	"time"
)

func newEndpointTest(t *testing.T, id string) (*Endpoint, *ttyio.Fake) {
	t.Helper()
	out := ttyio.NewFake()
	e, err := NewEndpoint(id, Geometry{8, 4}, out)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Close() })
	return e, out
}
func TestEndpointHiddenQueriesHaveTheirOwnOrigin(t *testing.T) {
	a, aw := newEndpointTest(t, "a")
	b, bw := newEndpointTest(t, "b")
	now := time.Now()
	if _, err := a.Feed([]byte("\x1b[2;3H\x1b[6n"), now); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Feed([]byte("\x1b[3;4H\x1b[6n"), now); err != nil {
		t.Fatal(err)
	}
	if err := a.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := b.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(aw.Bytes()) != "\x1b[2;3R" || string(bw.Bytes()) != "\x1b[3;4R" {
		t.Fatalf("crossed replies %q %q", aw.Bytes(), bw.Bytes())
	}
}
func TestEndpointSyncWithholdsThenRecoversAndCopies(t *testing.T) {
	e, _ := newEndpointTest(t, "one")
	now := time.Unix(100, 0)
	e.Feed([]byte("A"), now)
	before, err := e.Snapshot(now)
	if err != nil {
		t.Fatal(err)
	}
	e.Feed([]byte("\x1b[?2026h\rB"), now)
	held, _ := e.Snapshot(now.Add(100 * time.Millisecond))
	if held.Cells[0].Content != "A" {
		t.Fatal("published incomplete sync")
	}
	recovered, _ := e.Snapshot(now.Add(SyncTimeout))
	if recovered.Cells[0].Content != "B" {
		t.Fatal("sync recovery never published")
	}
	recovered.Cells[0].Content = "tampered"
	fresh, _ := e.Snapshot(now.Add(SyncTimeout))
	if fresh.Cells[0].Content != "B" || before.Cells[0].Content != "A" {
		t.Fatal("mutable frame alias")
	}
	e.Feed([]byte("\x1b[?2026l\rC"), now.Add(time.Second))
	after, _ := e.Snapshot(now.Add(time.Second))
	if after.Cells[0].Content != "C" {
		t.Fatal("sync end")
	}
}
func TestEndpointEffectsAreOnceOnlyAndClipboardReadsStayUnsupported(t *testing.T) {
	e, out := newEndpointTest(t, "one")
	batch, err := e.Feed([]byte("\x1b]52;c;aGk=\a\x1b]777;notify;pair;hello\a\x1b]52;c;?\a"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Effects) != 2 || batch.Effects[0].Kind != ClipboardEffect || string(batch.Effects[0].Data) != "hi" || batch.Effects[1].Kind != NotificationEffect || batch.Effects[1].Text != "hello" {
		t.Fatalf("%+v", batch)
	}
	e.Snapshot(time.Now())
	e.Snapshot(time.Now())
	empty, err := e.Feed(nil, time.Now())
	if err != nil || len(empty.Effects) != 0 {
		t.Fatal("snapshot replayed effects")
	}
	e.Flush(context.Background())
	if string(out.Bytes()) != "\x1b]52;c;\x1b\\" {
		t.Fatalf("clipboard read response %q", out.Bytes())
	}
	for _, effect := range batch.Effects {
		if effect.EndpointID != "one" || effect.Position == 0 || effect.Sequence == 0 {
			t.Fatalf("missing origin %+v", effect)
		}
	}
}
func TestEndpointResizeCommitsOnlyAfterPTYSuccess(t *testing.T) {
	e, _ := newEndpointTest(t, "one")
	before, _ := e.Snapshot(time.Now())
	failure := errors.New("ioctl failed")
	if err := e.Resize(Geometry{10, 5}, func(Geometry) error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("%v", err)
	}
	after, _ := e.Snapshot(time.Now())
	if after.Geometry != before.Geometry || after.GeometryEpoch != before.GeometryEpoch {
		t.Fatal("unconfirmed resize advanced state")
	}
	if err := e.Resize(Geometry{10, 5}, func(Geometry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	resized, _ := e.Snapshot(time.Now())
	if resized.Geometry != (Geometry{10, 5}) || resized.GeometryEpoch <= before.GeometryEpoch {
		t.Fatal("resize not committed")
	}
}
func TestEndpointInputUsesNegotiatedModes(t *testing.T) {
	e, out := newEndpointTest(t, "one")
	e.Feed([]byte("\x1b[>3u\x1b[?1002h\x1b[?1006h"), time.Now())
	if err := e.Send(uv.KeyReleaseEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl}); err != nil {
		t.Fatal(err)
	}
	e.Send(uv.MouseMotionEvent{X: 1, Y: 2, Button: uv.MouseNone})
	e.Send(uv.MouseMotionEvent{X: 1, Y: 2, Button: uv.MouseLeft})
	e.Flush(context.Background())
	if string(out.Bytes()) != "\x1b[13;5:3u\x1b[<32;2;3M" {
		t.Fatalf("wrong negotiated input %q", out.Bytes())
	}
	if e.Modes().Tracking != 1002 {
		t.Fatal("lost drag mode")
	}
}

func TestEndpointSplitOutputSurvivesInputAndSnapshots(t *testing.T) {
	for split := 0; split <= len("e\u0301\x1b[31m界"); split++ {
		t.Run(fmt.Sprint(split), func(t *testing.T) {
			e, _ := newEndpointTest(t, "a")
			p := []byte("e\u0301\x1b[31m界")
			now := time.Now()
			if _, err := e.Feed(p[:split], now); err != nil {
				t.Fatal(err)
			}
			if _, err := e.Snapshot(now); err != nil {
				t.Fatal(err)
			}
			if err := e.Send(uv.KeyPressEvent{Code: 'x', Text: "x"}); err != nil {
				t.Fatal(err)
			}
			if _, err := e.Feed(p[split:], now); err != nil {
				t.Fatal(err)
			}
			f, err := e.Snapshot(now)
			if err != nil {
				t.Fatal(err)
			}
			if f.Cells[0].Content != "e\u0301" || f.Cells[1].Content != "界" || f.Cells[1].Width != 2 {
				t.Fatalf("split %d: %+v", split, f.Cells[:4])
			}
		})
	}
}
func TestEndpointRepeatedSyncBeginDoesNotPostponeRecovery(t *testing.T) {
	e, _ := newEndpointTest(t, "a")
	now := time.Unix(100, 0)
	e.Feed([]byte("\x1b[?2026hA"), now)
	e.Feed([]byte("\x1b[?2026hB"), now.Add(100*time.Millisecond))
	f, err := e.Snapshot(now.Add(SyncTimeout))
	if err != nil || f.Cells[1].Content != "B" {
		t.Fatalf("recovery postponed: %+v %v", f, err)
	}
}
func TestEndpointDecoderNegotiatedDelivery(t *testing.T) {
	e, out := newEndpointTest(t, "a")
	e.Feed([]byte("\x1b[?1h\x1b[?2004h"), time.Now())
	var decoder Decoder
	for _, b := range []byte("\x1b[A\x1b[200~exact\x1b[A\x1b[201~\x1b[1;2R") {
		events, err := decoder.Feed([]byte{b})
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			if !event.Reply {
				if err := e.Send(event.Event); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if err := e.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(out.Bytes()); got != "\x1bOA\x1b[200~exact\x1b[A\x1b[201~" {
		t.Fatalf("wrong delivered input %q", got)
	}
}
func TestEndpointCloseRejectsAllTransactions(t *testing.T) {
	e, _ := newEndpointTest(t, "a")
	e.Close()
	if _, err := e.Feed([]byte("late"), time.Now()); !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}
	if _, err := e.Snapshot(time.Now()); !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}
	if err := e.Send(uv.KeyPressEvent{Code: 'a'}); !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}
}
func TestEndpointInvalidResizeDoesNotCallPTY(t *testing.T) {
	e, _ := newEndpointTest(t, "a")
	called := false
	err := e.Resize(Geometry{MaxCells, 2}, func(Geometry) error { called = true; return nil })
	if err == nil || called {
		t.Fatalf("invalid resize reached PTY: %v %v", called, err)
	}
}

type discardTTY struct{}

func (discardTTY) WriteContext(_ context.Context, p []byte) (int, error) { return len(p), nil }
func BenchmarkEndpointFeedSnapshot(b *testing.B) {
	for _, g := range []Geometry{{80, 24}, {240, 80}} {
		b.Run(fmt.Sprintf("%dx%d", g.Cols, g.Rows), func(b *testing.B) {
			e, err := NewEndpoint("bench", g, discardTTY{})
			if err != nil {
				b.Fatal(err)
			}
			defer e.Close()
			p := []byte("\x1b[Houtput with 色 and e\u0301\x1b[K")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := e.Feed(p, time.Now()); err != nil {
					b.Fatal(err)
				}
				if _, err := e.Snapshot(time.Now()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
func TestEndpointSyncCapturesLatestCompleteStateWithoutPriorSnapshot(t *testing.T) {
	e, _ := newEndpointTest(t, "a")
	now := time.Now()
	e.Feed([]byte("A\x1b[?2026h\rB"), now)
	f, err := e.Snapshot(now)
	if err != nil || f.Cells[0].Content != "A" {
		t.Fatalf("held wrong completed state %+v %v", f.Cells[:1], err)
	}
}
func TestEndpointPublicationDeadlineOnlyWhileWithheld(t *testing.T) {
	e, _ := newEndpointTest(t, "a")
	now := time.Unix(100, 0)
	if !e.NextPublication().IsZero() {
		t.Fatal("idle timer")
	}
	e.Feed([]byte("\x1b[?2026hA"), now)
	if got := e.NextPublication(); !got.Equal(now.Add(SyncTimeout)) {
		t.Fatal(got)
	}
	e.Snapshot(now.Add(SyncTimeout))
	if !e.NextPublication().IsZero() {
		t.Fatal("recovered timer still running")
	}
}
func TestEndpointConcurrentCloseAlwaysJoinsWriter(t *testing.T) {
	e, _ := newEndpointTest(t, "a")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Close()
			select {
			case <-e.input.done:
			default:
				t.Error("Close returned before writer exit")
			}
		}()
	}
	wg.Wait()
}
func TestReplyBufferOverflowCannotBeIgnoredByQueryHandler(t *testing.T) {
	e, _ := newEndpointTest(t, "a")
	if n, err := e.replies.Write(make([]byte, MaxInputBytes+1)); n != 0 || !errors.Is(err, ErrBackpressure) {
		t.Fatal(n, err)
	}
	if _, err := e.Feed(nil, time.Now()); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("ignored reply overflow: %v", err)
	}
}
