package terminal

import (
	"context"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"sync"
	"testing"
	"time"
)

type chromeControlledWriter struct {
	mu      sync.Mutex
	blocked bool
	out     *ttyio.Fake
}

func (w *chromeControlledWriter) setBlocked(on bool) { w.mu.Lock(); w.blocked = on; w.mu.Unlock() }
func (w *chromeControlledWriter) WriteContext(ctx context.Context, b []byte) (int, error) {
	w.mu.Lock()
	blocked := w.blocked
	w.mu.Unlock()
	if blocked {
		n := min(3, len(b))
		n, _ = w.out.WriteContext(ctx, b[:n])
		return n, context.DeadlineExceeded
	}
	return w.out.WriteContext(ctx, b)
}
func chromeSnapshot(t *testing.T, p *Presenter) ChromeState {
	t.Helper()
	s, err := p.ChromeState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestChromeConfirmationFencesMouseUntilAutomaticRecovery(t *testing.T) {
	ctx := context.Background()
	writer := &chromeControlledWriter{out: ttyio.NewFake()}
	p := NewPresenter(writer, AnyMotion)
	t.Cleanup(func() { writer.setBlocked(false); p.Release(ctx) })
	e, input := newEndpointTest(t, "chrome-confirm")
	if _, err := e.Feed([]byte("\x1b[?1003h\x1b[?1006h"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if s := chromeSnapshot(t, p); s.Confirmed || s.Revision != 0 {
		t.Fatalf("unpainted state %+v", s)
	}
	if err := p.Select(ctx, e, Geometry{8, 5}, make([]Cell, 8)); err != nil {
		t.Fatal(err)
	}
	before := chromeSnapshot(t, p)
	if !before.Confirmed || before.Rows != 1 || before.Revision == 0 {
		t.Fatalf("selected state %+v", before)
	}
	writer.setBlocked(true)
	if err := p.ResizeLayout(ctx, Geometry{8, 5}, make([]Cell, 16), func(Geometry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	pending := chromeSnapshot(t, p)
	if pending.Confirmed || pending.Rows != 2 || pending.Revision <= before.Revision {
		t.Fatalf("partial resize remained confirmed: %+v", pending)
	}
	if err := p.Input(ctx, uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	if err := p.Input(ctx, uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	if err := p.Input(ctx, uv.MouseWheelEvent{X: 1, Y: 1, Button: uv.MouseWheelDown}); err != nil {
		t.Fatal(err)
	}
	if err := e.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if len(input.Bytes()) != 0 {
		t.Fatalf("mouse reached child behind unconfirmed chrome: %q", input.Bytes())
	}
	if err := p.Input(ctx, uv.KeyPressEvent{Code: 'x', Text: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := p.Input(ctx, uv.PasteEvent{Content: "paste"}); err != nil {
		t.Fatal(err)
	}
	if err := e.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if string(input.Bytes()) != "xpaste" {
		t.Fatalf("logical input lost: %q", input.Bytes())
	}
	writer.setBlocked(false)
	waitFor(t, "same chrome revision confirmed by automatic repaint", func() bool {
		s := chromeSnapshot(t, p)
		return s.Confirmed && s.Revision == pending.Revision && s.Rows == 2
	})
	if err := p.Input(ctx, uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	if err := p.Input(ctx, uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	if err := e.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if string(input.Bytes()) == "xpaste" {
		t.Fatal("mouse still fenced after recovery")
	}
}

func TestChromeRevisionOnlyChangesForLayoutAdmissions(t *testing.T) {
	ctx := context.Background()
	p, parent, e, taps := tappedPresenter(t)
	if err := p.Select(ctx, e, Geometry{8, 6}, make([]Cell, 16)); err != nil {
		t.Fatal(err)
	}
	first := chromeSnapshot(t, p)
	if first.Rows != 2 || !first.Confirmed {
		t.Fatalf("two-row select %+v", first)
	}
	if err := p.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if got := chromeSnapshot(t, p); got != first {
		t.Fatalf("body repaint revised chrome: %+v -> %+v", first, got)
	}
	if err := p.UpdateChrome(ctx, make([]Cell, 16)); err != nil {
		t.Fatal(err)
	}
	second := chromeSnapshot(t, p)
	if second.Revision <= first.Revision || !second.Confirmed {
		t.Fatalf("chrome admission %+v", second)
	}
	f := presented(t, p)
	f.EndpointID = ""
	f.Generation = 0
	f.GeometryEpoch = 0
	f.ChromeRows = 1
	seen, written := taps.count(), len(parent.Bytes())
	if err := p.Panel(ctx, f, FramePrivate); err != nil {
		t.Fatal(err)
	}
	third := chromeSnapshot(t, p)
	if third.Rows != 1 || !third.Confirmed || third.Revision <= second.Revision {
		t.Fatalf("panel admission %+v", third)
	}
	if len(parent.Bytes()) != written {
		t.Fatal("metadata-only publication unexpectedly needed drawing bytes")
	}
	if taps.count() != seen+1 || taps.last(t).frame.ChromeRows != 1 {
		t.Fatal("metadata-only publication missed tap")
	}
}

func TestPresenterChromeShapeChangesUseAcknowledgedResize(t *testing.T) {
	ctx := context.Background()
	p, _, e, _ := presenterFixture(t, AnyMotion)
	if err := p.Select(ctx, e, Geometry{8, 6}, make([]Cell, 16)); err != nil {
		t.Fatal(err)
	}
	before := chromeSnapshot(t, p)
	if err := p.UpdateChrome(ctx, make([]Cell, 8)); err == nil {
		t.Fatal("unacknowledged chrome-height change accepted")
	}
	if got := chromeSnapshot(t, p); got != before {
		t.Fatalf("rejected update changed chrome %+v -> %+v", before, got)
	}
	var child Geometry
	if err := p.ResizeLayout(ctx, Geometry{8, 6}, make([]Cell, 8), func(g Geometry) error { child = g; return nil }); err != nil {
		t.Fatal(err)
	}
	if child != (Geometry{8, 5}) {
		t.Fatalf("wrong child size %+v", child)
	}
	if got := chromeSnapshot(t, p); got.Rows != 1 || !got.Confirmed || got.Revision <= before.Revision {
		t.Fatalf("shrunken chrome %+v", got)
	}
	if err := p.ResizeLayout(ctx, Geometry{8, 6}, make([]Cell, 16), func(g Geometry) error { child = g; return nil }); err != nil {
		t.Fatal(err)
	}
	if child != (Geometry{8, 4}) {
		t.Fatalf("wrong child size %+v", child)
	}
	if err := p.Resize(ctx, Geometry{10, 7}, func(g Geometry) error { child = g; return nil }); err != nil {
		t.Fatal(err)
	}
	if child != (Geometry{10, 5}) {
		t.Fatalf("Resize forgot two-row reservation %+v", child)
	}
	if got := chromeSnapshot(t, p); got.Rows != 2 || !got.Confirmed || got.Host != (Geometry{10, 7}) {
		t.Fatalf("resized chrome %+v", got)
	}
}

func TestBodyStallRevokesExistingChromeConfirmation(t *testing.T) {
	ctx := context.Background()
	writer := &chromeControlledWriter{out: ttyio.NewFake()}
	p := NewPresenter(writer, AnyMotion)
	t.Cleanup(func() { writer.setBlocked(false); p.Release(ctx) })
	e, _ := newEndpointTest(t, "body-stall")
	if err := p.Select(ctx, e, Geometry{8, 5}, make([]Cell, 8)); err != nil {
		t.Fatal(err)
	}
	first := chromeSnapshot(t, p)
	writer.setBlocked(true)
	if _, err := e.Feed([]byte("new body"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := p.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	pending := chromeSnapshot(t, p)
	if pending.Confirmed || pending.Revision != first.Revision {
		t.Fatalf("body stall confirmation %+v -> %+v", first, pending)
	}
	writer.setBlocked(false)
	waitFor(t, "body repaint restores existing chrome revision", func() bool { s := chromeSnapshot(t, p); return s.Confirmed && s.Revision == first.Revision })
}
