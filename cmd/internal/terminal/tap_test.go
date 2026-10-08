package terminal

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

type tapRecord struct {
	frame Frame
	class FrameClass
}

type tapRecorder struct {
	mu   sync.Mutex
	seen []tapRecord
}

func (r *tapRecorder) tap(f Frame, class FrameClass) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen = append(r.seen, tapRecord{f, class})
}

func (r *tapRecorder) count() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.seen) }

func (r *tapRecorder) last(t *testing.T) tapRecord {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.seen) == 0 {
		t.Fatal("tap saw no frame")
	}
	return r.seen[len(r.seen)-1]
}

// presented reads the presenter's retained frame on its own goroutine.
func presented(t *testing.T, p *Presenter) Frame {
	t.Helper()
	var f Frame
	if err := p.call(context.Background(), func(context.Context) error { f = p.previous.Clone(); return nil }); err != nil {
		t.Fatal(err)
	}
	return f
}

func tappedPresenter(t *testing.T) (*Presenter, *ttyio.Fake, *Endpoint, *tapRecorder) {
	t.Helper()
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	r := &tapRecorder{}
	if err := p.SetTap(context.Background(), r.tap); err != nil {
		t.Fatal(err)
	}
	return p, parent, e, r
}

func chromeRow(t *testing.T, text string, cols int) []Cell {
	t.Helper()
	row, err := StyledRows(text, cols, 1)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestTapSeesEveryPaintedFrame(t *testing.T) {
	ctx := context.Background()
	p, _, e, r := tappedPresenter(t)
	check := func(step string) {
		t.Helper()
		got := r.last(t)
		if got.class != FramePublic {
			t.Fatalf("%s: class %v, want public", step, got.class)
		}
		if want := presented(t, p); !reflect.DeepEqual(got.frame, want) {
			t.Fatalf("%s: tapped frame differs from the painted one", step)
		}
	}
	if err := p.Select(ctx, e, Geometry{8, 5}, chromeRow(t, "CHR", 8)); err != nil {
		t.Fatal(err)
	}
	check("select")
	if _, err := e.Feed([]byte("HI"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := p.Present(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := p.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	check("output")
	if err := p.UpdateChrome(ctx, chromeRow(t, "NEW", 8)); err != nil {
		t.Fatal(err)
	}
	check("chrome")
	if got := r.last(t).frame; got.Cells[len(got.Cells)-8].Content != "N" {
		t.Fatalf("chrome update not in tapped frame: %q", got.Cells[len(got.Cells)-8].Content)
	}
	if err := p.Resize(ctx, Geometry{10, 6}, func(Geometry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	check("resize")
	if got := r.last(t).frame.Geometry; got != (Geometry{10, 6}) {
		t.Fatalf("resize geometry %+v", got)
	}
}

func TestTapClassFollowsPanel(t *testing.T) {
	ctx := context.Background()
	p, _, e, r := tappedPresenter(t)
	panel := Frame{Geometry: Geometry{8, 5}, Cells: make([]Cell, 40)}
	if err := p.Panel(ctx, panel, FramePrivate); err != nil {
		t.Fatal(err)
	}
	if got := r.last(t).class; got != FramePrivate {
		t.Fatalf("panel class %v, want private", got)
	}
	selectPresenter(t, p, e)
	if got := r.last(t).class; got != FramePublic {
		t.Fatalf("select after private panel: class %v, want public", got)
	}
	if err := p.Panel(ctx, panel, FramePublic); err != nil {
		t.Fatal(err)
	}
	if got := r.last(t).class; got != FramePublic {
		t.Fatalf("public panel class %v", got)
	}
}

func TestTapGetsAnOwnedClone(t *testing.T) {
	p, _, e, r := tappedPresenter(t)
	selectPresenter(t, p, e)
	got := r.last(t).frame
	retained := presented(t, p)
	got.Cells[0].Content = "X"
	if after := presented(t, p); !reflect.DeepEqual(after, retained) {
		t.Fatal("mutating the tapped frame changed the presenter's retained frame")
	}
}

func TestTapNotCalledOnFailedPaint(t *testing.T) {
	p, parent, e, r := tappedPresenter(t)
	parent.Enqueue(ttyio.WriteStep{Err: errors.New("boom")})
	if err := p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)); err == nil {
		t.Fatal("select succeeded through a failing parent")
	}
	if n := r.count(); n != 0 {
		t.Fatalf("tap saw %d frames from a failed paint", n)
	}
}

func TestSetTapNilStopsDelivery(t *testing.T) {
	ctx := context.Background()
	p, _, e, r := tappedPresenter(t)
	selectPresenter(t, p, e)
	n := r.count()
	if err := p.SetTap(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.UpdateChrome(ctx, chromeRow(t, "AGAIN", 8)); err != nil {
		t.Fatal(err)
	}
	if got := r.count(); got != n {
		t.Fatalf("tap called %d times after removal", got-n)
	}
}
