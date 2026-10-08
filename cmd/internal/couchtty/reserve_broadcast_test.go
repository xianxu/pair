package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/terminalcapture"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// rowFrame composes a frame whose last row is the drawn status row, as the
// Presenter does, so the broadcast's indicator check sees what was painted.
func rowFrame(t *testing.T, cols int, body string) terminal.Frame {
	t.Helper()
	child, err := terminal.StyledRows("child output", cols, 2)
	if err != nil {
		t.Fatal(err)
	}
	row, err := terminal.StyledRows(body, cols, 1)
	if err != nil {
		t.Fatal(err)
	}
	f, err := terminal.PanelFrame(terminal.Geometry{Cols: cols, Rows: 3}, append(child, row...), terminal.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestStatusRowBroadcastOffDrawsNothing(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{{Label: "brain", Active: true}}}
	off := RenderStatusRow(80, m)
	if strings.Contains(off.Body, "LIVE") || off.Control != (ColumnSpan{}) {
		t.Fatalf("off broadcast drew a cell: %q %+v", off.Body, off.Control)
	}
}

// The drawer↔checker contract: what RenderStatusRow draws while live is
// exactly what the broadcast accepts as the indicator, and nothing else is.
func TestStatusRowLiveCellSatisfiesIndicator(t *testing.T) {
	for _, c := range []struct {
		cell BroadcastCell
		cols int
		want bool
	}{
		{BroadcastLive, 80, true},
		{BroadcastStarting, 80, false},
		{BroadcastOff, 80, false},
		{BroadcastLive, textwidth.Width(broadcast.LiveLabel) - 1, false},
		{BroadcastLive, textwidth.Width(broadcast.LiveLabel), true},
	} {
		r := RenderStatusRow(c.cols, StatusModel{Broadcast: c.cell, Actors: []StatusActor{{Label: "brain"}}})
		if got := broadcast.IndicatorShown(rowFrame(t, c.cols, r.Body)); got != c.want {
			t.Errorf("cell %v at %d cols: IndicatorShown = %v, want %v (%q)", c.cell, c.cols, got, c.want, r.Body)
		}
	}
}

func TestStatusRowLiveLeadsRecAndShiftsChips(t *testing.T) {
	thread := couchcore.ThreadAddress{RepoScope: "r", Tag: "t"}
	m := StatusModel{
		Broadcast: BroadcastLive,
		Capture:   terminalcapture.Status{Phase: terminalcapture.Recording, LimitBytes: 100, WrittenBytes: 21},
		Actors:    []StatusActor{{Label: "brain", Thread: thread}},
	}
	r := RenderStatusRow(80, m)
	plain := string(ansi.Strip([]byte(r.Body)))
	if want := broadcast.LiveLabel + " REC 21%  brain"; plain != want {
		t.Fatalf("row %q, want %q", plain, want)
	}
	liveWidth := textwidth.Width(broadcast.LiveLabel)
	if r.Control != (ColumnSpan{Start: 0, End: liveWidth}) {
		t.Fatalf("control span %+v", r.Control)
	}
	start := textwidth.Width(broadcast.LiveLabel + " REC 21%  ")
	if len(r.Chips) != 1 || r.Chips[0].Start != start {
		t.Fatalf("chip spans %+v, want start %d", r.Chips, start)
	}
	if got, ok := r.ColumnToActor(start); !ok || got != thread {
		t.Fatal("chip no longer click-accurate after the live cell")
	}
	for col := 0; col < liveWidth; col++ {
		if !r.Control.Contains(col) {
			t.Fatalf("column %d of the red span is not the control", col)
		}
		if _, ok := r.ColumnToActor(col); ok {
			t.Fatalf("column %d maps to an actor", col)
		}
	}
	if r.Control.Contains(liveWidth) {
		t.Fatal("the column past the red span is part of the control")
	}
}

func TestStatusRowStartingCell(t *testing.T) {
	r := RenderStatusRow(80, StatusModel{Broadcast: BroadcastStarting})
	if !strings.HasPrefix(r.Body, broadcast.LiveSGR+broadcast.StartingLabel) {
		t.Fatalf("starting row %q", r.Body)
	}
	if r.Control != (ColumnSpan{Start: 0, End: textwidth.Width(broadcast.StartingLabel)}) {
		t.Fatalf("starting control %+v", r.Control)
	}
}
