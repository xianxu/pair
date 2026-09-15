package terminal

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func evictionRow(id uint64, text string) HistoryRow {
	return HistoryRow{ID: id, Cells: []Cell{{Content: text, Width: 1}}, Meta: RowMetadata{UsedColumns: 1}}
}
func TestHistoryEvictionCursorCoverage(t *testing.T) {
	f := Frame{EndpointID: "eviction", Geometry: Geometry{4, 2}, Cells: make([]Cell, 8)}
	installed := HistoryState{EndpointID: f.EndpointID, Cursor: HistoryCursor{NextID: 2}, FirstID: 0, Columns: 4}
	for _, tc := range []struct {
		name     string
		history  HistoryWindow
		rebuild  bool
		appended int
	}{
		{"overlapping-prefix-eviction", HistoryWindow{Cursor: HistoryCursor{NextID: 3}, Rows: []HistoryRow{evictionRow(1, "B"), evictionRow(2, "C")}}, false, 1},
		{"cursor-is-first-retained", HistoryWindow{Cursor: HistoryCursor{NextID: 4}, Rows: []HistoryRow{evictionRow(2, "C"), evictionRow(3, "D")}}, false, 2},
		{"cursor-fell-before-retained", HistoryWindow{Cursor: HistoryCursor{NextID: 5}, Rows: []HistoryRow{evictionRow(3, "D"), evictionRow(4, "E")}}, true, 2},
		{"gap-after-cursor", HistoryWindow{Cursor: HistoryCursor{NextID: 5}, Rows: []HistoryRow{evictionRow(2, "C"), evictionRow(4, "E")}}, true, 2},
		{"rejected-tail", HistoryWindow{Cursor: HistoryCursor{NextID: 4}, Rows: []HistoryRow{evictionRow(2, "C")}}, true, 1},
		{"clear-epoch", HistoryWindow{Cursor: HistoryCursor{ClearEpoch: 1, NextID: 3}, Rows: []HistoryRow{evictionRow(2, "C")}}, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := RenderWithHistory(f, f, tc.history, installed)
			if err != nil {
				t.Fatal(err)
			}
			var wire bytes.Buffer
			if err := plan.Emit(func(p []byte) error { _, err := wire.Write(p); return err }); err != nil {
				t.Fatal(err)
			}
			if got := bytes.Contains(wire.Bytes(), []byte("\x1b[3J")); got != tc.rebuild {
				t.Fatalf("clear-history=%v want%v", got, tc.rebuild)
			}
			if len(plan.rows) != tc.appended {
				t.Fatalf("serializedrows=%d want%d", len(plan.rows), tc.appended)
			}
		})
	}
}

func TestPresenterHistoryEvictionNativeAppendAndRebuild(t *testing.T) {
	ctx := context.Background()
	parent := ttyio.NewFake()
	p := NewPresenter(parent, ChildRequested)
	defer p.Release(ctx)
	e, err := NewEndpoint("eviction", Geometry{4, 2}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	// Exercise the production backend's real bounded eviction, at a small cap.
	e.backend.SetScrollbackSize(2)
	source := "A\r\nB\r\nC\r\nD"
	e.Feed([]byte(source), time.Now())
	chrome, err := StyledRows("CHR", 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Select(ctx, e, Geometry{4, 3}, chrome); err != nil {
		t.Fatal(err)
	}
	before := len(parent.Bytes())
	source += "\r\nE"
	e.Feed([]byte("\r\nE"), time.Now())
	if err := p.Present(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := p.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(parent.Bytes()[before:], []byte("\x1b[3J")) {
		t.Fatal("continuous append rebuilt on bounded prefix eviction")
	}
	pub, err := e.Publication(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(pub.History.Rows) != 2 || pub.History.Rows[0].ID != 1 {
		t.Fatal("export stopped being bounded")
	}
	directWire := "\x1b[1;2r" + source + "\x1b[3;1HCHR"
	want := runHistoryOracle(t, 4, 3, directWire)[0]
	got := runHistoryOracle(t, 4, 3, string(parent.Bytes()))[0]
	if len(got.History) != 3 || !reflect.DeepEqual(got.History, want.History) || !reflect.DeepEqual(got.Lines, want.Lines) {
		t.Fatalf("continuous parent history got%+v want%+v", got, want)
	}
	continuous := string(parent.Bytes())
	// A genuine uncovered gap must discard stale parent history and rebuild the
	// retained suffix. Four unseen admissions exceed this endpoint's two rows.
	before = len(parent.Bytes())
	e.Feed([]byte("\r\nF\r\nG\r\nH\r\nI"), time.Now())
	if err := p.Present(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := p.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(parent.Bytes()[before:], []byte("\x1b[3J")) {
		t.Fatal("loss gap failed to rebuild")
	}
	rebuilt := runHistoryOracle(t, 4, 3, string(parent.Bytes()))[0]
	if len(rebuilt.History) != 2 || rebuilt.History[0].Text != "F" || rebuilt.History[1].Text != "G" {
		t.Fatalf("loss rebuild history %+v", rebuilt.History)
	}
	if native := nativeHistoryDump(t, 4, 3, continuous); native != "A\nB\nC\nD\nE\nCHR\n" {
		t.Fatalf("native continuous copy %q", native)
	}
	if native := nativeHistoryDump(t, 4, 3, string(parent.Bytes())); native != "F\nG\nH\nI\nCHR\n" {
		t.Fatalf("native gap rebuild %q", native)
	}
}

func TestHistoryGeometryChangeRebuilds(t *testing.T) {
	before := Frame{EndpointID: "geometry", Geometry: Geometry{4, 2}, Cells: make([]Cell, 8)}
	state := HistoryState{EndpointID: "geometry", Cursor: HistoryCursor{NextID: 1}, Columns: 4}
	history := HistoryWindow{Cursor: state.Cursor, Rows: []HistoryRow{evictionRow(0, "A")}}
	for _, geometry := range []Geometry{{4, 3}, {5, 2}} {
		after := before
		after.Geometry = geometry
		after.Cells = make([]Cell, geometry.Cols*geometry.Rows)
		wire, _ := renderHistoryBytes(t, before, after, history, state)
		if !bytes.Contains(wire, []byte("\x1b[3J")) {
			t.Fatalf("geometry%+v did not rebuild history", geometry)
		}
	}
}

func TestPresenterHistoryEvictionKeepsSoftLineOracle(t *testing.T) {
	ctx := context.Background()
	parent := ttyio.NewFake()
	p := NewPresenter(parent, ChildRequested)
	defer p.Release(ctx)
	e, err := NewEndpoint("soft-eviction", Geometry{4, 2}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.backend.SetScrollbackSize(2)
	e.Feed([]byte("AAAABBBBCCCCDDDD"), time.Now())
	chrome, _ := StyledRows("CHR", 4, 1)
	if err := p.Select(ctx, e, Geometry{4, 3}, chrome); err != nil {
		t.Fatal(err)
	}
	before := len(parent.Bytes())
	e.Feed([]byte("E"), time.Now())
	if err := p.Present(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := p.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(parent.Bytes()[before:], []byte("\x1b[3J")) {
		t.Fatal("soft append rebuilt on prefix eviction")
	}
	directWire := "\x1b[1;2rAAAABBBBCCCCDDDDE\x1b[3;1HCHR"
	direct := runHistoryOracle(t, 4, 3, directWire)[0]
	got := runHistoryOracle(t, 4, 3, string(parent.Bytes()))[0]
	if !reflect.DeepEqual(got.History, direct.History) || !reflect.DeepEqual(got.Lines, direct.Lines) || !reflect.DeepEqual(got.Wraps, direct.Wraps) {
		t.Fatalf("soft history differs: got%+v want%+v", got, direct)
	}
	expected := nativeHistoryDump(t, 4, 3, directWire)
	native := nativeHistoryDump(t, 4, 3, string(parent.Bytes()))
	if native != expected || native != "AAAABBBBCCCC\nDDDDE\nCHR\n" {
		t.Fatalf("native soft copy got%q want%q", native, expected)
	}
}
