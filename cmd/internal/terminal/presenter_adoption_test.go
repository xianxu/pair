package terminal

import (
	"context"
	"errors"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"testing"
	"time"
)

func TestPresenterEOFSettlesPendingCancellation(t *testing.T) {
	p, _, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	input.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.call(context.Background(), func(context.Context) error { return p.cancelDrag(ctx) }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	e.EndInput()
	next, _ := newEndpointTest(t, "successor")
	if err := p.Select(context.Background(), next, Geometry{8, 5}, make([]Cell, 8)); err != nil {
		t.Fatalf("dead child blocked successor:%v", err)
	}
	if p.View().Admitted != "successor" || p.View().PendingMouseRelease != "" {
		t.Fatalf("view:%+v", p.View())
	}
}

func TestPresenterZeroChromeResizesWithoutLosingRow(t *testing.T) {
	w := ttyio.NewFake()
	e, err := NewEndpoint("single", Geometry{8, 1}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	p := NewPresenter(w, ChildRequested)
	defer p.Release(context.Background())
	if err = p.Select(context.Background(), e, Geometry{8, 1}, nil); err != nil {
		t.Fatal(err)
	}
	if err = p.Resize(context.Background(), Geometry{10, 1}, func(Geometry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	f, err := e.Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if f.Geometry.Rows != 1 {
		t.Fatalf("rows=%d", f.Geometry.Rows)
	}
}
