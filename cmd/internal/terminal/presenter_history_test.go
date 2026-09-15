package terminal

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPresenterHistorySwitchPanelAndAltReleaseOracle(t *testing.T) {
	ctx := context.Background()
	w := ttyio.NewFake()
	p := NewPresenter(w, ChildRequested)
	defer p.Release(ctx)
	e, err := NewEndpoint("history", Geometry{4, 4}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	source := "AAA界Z\r\nAA  BZ\r\nHARD\r\nONE\r\nTWO\r\nTHR\r\nFOUR"
	if _, err = e.Feed([]byte(source), time.Now()); err != nil {
		t.Fatal(err)
	}
	chrome, err := StyledRows("CHR", 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err = p.Select(ctx, e, Geometry{4, 5}, chrome); err != nil {
			t.Fatal(err)
		}
	}
	panel := Frame{Geometry: Geometry{4, 5}, Cells: make([]Cell, 20)}
	if err = p.Panel(ctx, panel); err != nil {
		t.Fatal(err)
	}
	if err = p.Select(ctx, e, Geometry{4, 5}, chrome); err != nil {
		t.Fatal(err)
	}
	expected := runHistoryOracle(t, 4, 5, "\x1b[1;4r"+source+"\x1b[5;1HCHR")[0]
	got := runHistoryOracle(t, 4, 5, string(w.Bytes()))[0]
	if !reflect.DeepEqual(got.History, expected.History) || !reflect.DeepEqual(got.Lines, expected.Lines) || !reflect.DeepEqual(got.Wraps, expected.Wraps) {
		t.Fatalf("history/panel restoration got%+v want%+v", got, expected)
	}
	if _, err = e.Feed([]byte("\x1b[?1049hALT"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err = p.Present(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err = p.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if !p.altOwned {
		t.Fatal("alternate buffer ownership not recorded")
	}
	if err = p.Release(ctx); err != nil {
		t.Fatal(err)
	}
	released := runHistoryOracle(t, 4, 5, string(w.Bytes()))[0]
	if !reflect.DeepEqual(released.History, expected.History) || !reflect.DeepEqual(released.Lines, expected.Lines) {
		t.Fatalf("release lost normal buffer: %+v", released)
	}
}

func TestPresenterReleaseCancelsStringBeforeLeavingOwnedAlt(t *testing.T) {
	ctx := context.Background()
	w := ttyio.NewFake()
	p := NewPresenter(w, ChildRequested)
	defer p.Release(ctx)
	e, err := NewEndpoint("alt", Geometry{8, 3}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Feed([]byte("MAIN"), time.Now())
	if err = p.Select(ctx, e, Geometry{8, 3}, nil); err != nil {
		t.Fatal(err)
	}
	e.Feed([]byte("\x1b[?1049hALT"), time.Now())
	if err = p.Select(ctx, e, Geometry{8, 3}, nil); err != nil {
		t.Fatal(err)
	}
	w.Enqueue(ttyio.WriteStep{Limit: 5, Err: io.ErrUnexpectedEOF})
	_ = p.call(ctx, func(ctx context.Context) error { return p.fail(p.write(ctx, []byte("\x1b]0;interrupted"), true)) })
	if err = p.Release(ctx); err != nil {
		t.Fatal(err)
	}
	got := runHistoryOracle(t, 8, 3, string(w.Bytes()))[0]
	if !strings.HasPrefix(got.Lines[0], "MAIN") {
		t.Fatalf("release stayed in alternate buffer:%+v", got)
	}
}
