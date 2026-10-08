package terminal

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

const overlayTint = ansi.IndexedColor(214)

// tintOverlay tints one cell's background while on, recording the classes it
// was called with.
type tintOverlay struct {
	mu      sync.Mutex
	col     int
	row     int
	on      bool
	classes []FrameClass
}

func (o *tintOverlay) apply(f Frame, class FrameClass) Frame {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.classes = append(o.classes, class)
	if !o.on || o.row >= f.Geometry.Rows || o.col >= f.Geometry.Cols {
		return f
	}
	out := f.Clone()
	out.Cells[o.row*f.Geometry.Cols+o.col].Style.Bg = overlayTint
	return out
}

func (o *tintOverlay) set(on bool) { o.mu.Lock(); o.on = on; o.mu.Unlock() }

func TestOverlayReachesPaintAndTap(t *testing.T) {
	ctx := context.Background()
	p, parent, e, r := tappedPresenter(t)
	o := &tintOverlay{col: 2, row: 1, on: true}
	if err := p.SetOverlay(ctx, o.apply); err != nil {
		t.Fatal(err)
	}
	selectPresenter(t, p, e)
	if got := r.last(t).frame.Cells[1*8+2].Style.Bg; got != overlayTint {
		t.Fatalf("tapped frame not tinted: %v", got)
	}
	if got := presented(t, p).Cells[1*8+2].Style.Bg; got != overlayTint {
		t.Fatal("painted frame not tinted")
	}
	if !strings.Contains(string(parent.Bytes()), "\x1b[48;5;214m") {
		t.Fatal("tint never reached the parent terminal")
	}
}

// An overlay that changes nothing paints exactly what no overlay paints.
func TestIdentityOverlayIsByteIdentical(t *testing.T) {
	run := func(withOverlay bool) string {
		ctx := context.Background()
		parent := ttyio.NewFake()
		p := NewPresenter(parent, ChildRequested)
		defer p.Release(ctx)
		e, _ := newEndpointTest(t, "same")
		if withOverlay {
			if err := p.SetOverlay(ctx, func(f Frame, _ FrameClass) Frame { return f }); err != nil {
				t.Fatal(err)
			}
		}
		if err := p.Select(ctx, e, Geometry{8, 5}, make([]Cell, 8)); err != nil {
			t.Fatal(err)
		}
		e.Feed([]byte("hello"), time.Now())
		p.Present(ctx, e)
		p.Flush(ctx)
		return string(parent.Bytes())
	}
	if a, b := run(false), run(true); a != b {
		t.Fatalf("identity overlay changed the bytes:\n%q\n%q", a, b)
	}
}

// Refresh repaints the current selection, so a fading mark advances with no
// child output.
func TestRefreshRepaintsWithoutOutput(t *testing.T) {
	ctx := context.Background()
	p, _, e, r := tappedPresenter(t)
	o := &tintOverlay{col: 0, row: 0}
	p.SetOverlay(ctx, o.apply)
	selectPresenter(t, p, e)
	if r.last(t).frame.Cells[0].Style.Bg == overlayTint {
		t.Fatal("tinted before the overlay was on")
	}
	o.set(true)
	if err := p.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if r.last(t).frame.Cells[0].Style.Bg != overlayTint {
		t.Fatal("Refresh didn't repaint the endpoint with the overlay")
	}
	// A panel too.
	panel := Frame{Geometry: Geometry{8, 5}, Cells: make([]Cell, 40)}
	if err := p.Panel(ctx, panel, FramePublic); err != nil {
		t.Fatal(err)
	}
	o.set(false)
	if err := p.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if r.last(t).frame.Cells[0].Style.Bg == overlayTint {
		t.Fatal("Refresh didn't repaint the panel")
	}
}

func TestOverlaySeesPanelClass(t *testing.T) {
	ctx := context.Background()
	p, _, _, _ := tappedPresenter(t)
	o := &tintOverlay{}
	p.SetOverlay(ctx, o.apply)
	panel := Frame{Geometry: Geometry{8, 5}, Cells: make([]Cell, 40)}
	if err := p.Panel(ctx, panel, FramePrivate); err != nil {
		t.Fatal(err)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.classes) == 0 || o.classes[len(o.classes)-1] != FramePrivate {
		t.Fatalf("overlay classes %v, want the panel's private class", o.classes)
	}
}

// Marks are drawn on the screen, never into the parent's scrollback: history
// rows come from the endpoint, not the overlaid frame.
func TestOverlayNeverReachesScrollback(t *testing.T) {
	ctx := context.Background()
	w := ttyio.NewFake()
	p := NewPresenter(w, ChildRequested)
	defer p.Release(ctx)
	e, err := NewEndpoint("history", Geometry{6, 3}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	o := &tintOverlay{col: 0, row: 0, on: true}
	p.SetOverlay(ctx, o.apply)
	if err := p.Select(ctx, e, Geometry{6, 3}, nil); err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		e.Feed([]byte("line"+string(rune('0'+i))+"\r\n"), time.Now())
		p.Present(ctx, e)
		p.Flush(ctx)
	}
	screen := runHistoryOracle(t, 6, 3, string(w.Bytes()))[0]
	if len(screen.History) == 0 {
		t.Fatal("no scrollback produced; the test proves nothing")
	}
	for i, row := range screen.History {
		for _, c := range row.Cells {
			if c.BG == 214 {
				t.Fatalf("scrollback row %d (%q) carries the overlay tint", i, row.Text)
			}
		}
	}
	if screen.Cells[0][0].BG != 214 {
		t.Fatal("overlay missing from the visible screen")
	}
}
