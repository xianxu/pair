package vt

import (
	"io"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func historyEmulator(t *testing.T, w, h int) *Emulator {
	t.Helper()
	e := NewEmulator(w, h)
	e.SetReplyWriter(io.Discard)
	t.Cleanup(func() { e.Close() })
	return e
}

func TestPairHistoryWrapMetadata(t *testing.T) {
	for _, wire := range []string{"AAA界Z\r\nHARD\r\n", "AA  BZ\r\nHARD\r\n"} {
		for split := 0; split <= len(wire); split++ {
			e := historyEmulator(t, 4, 2)
			e.WriteString(wire[:split])
			e.WriteString(wire[split:])
			h := e.HistorySnapshot()
			if len(h.Rows) != 2 || h.Rows[0].ID != 0 || h.Rows[1].ID != 1 || h.NextID != 2 {
				t.Fatalf("split %d: %#v", split, h)
			}
			if h.Rows[0].Meta.Wrapped || !h.Rows[1].Meta.Wrapped || h.ContinuesToScreen {
				t.Fatalf("wrap metadata %#v", h)
			}
			want := 3
			if wire[2] == ' ' {
				want = 4
			}
			if h.Rows[0].Meta.UsedColumns != want {
				t.Fatalf("used=%d want=%d", h.Rows[0].Meta.UsedColumns, want)
			}
			if want == 4 && h.Rows[0].Cells[3].Content != " " {
				t.Fatal("printed trailing space lost")
			}
		}
	}
}

func TestPairHistoryIdentityClearAndLoss(t *testing.T) {
	e := historyEmulator(t, 4, 2)
	e.SetScrollbackSize(2)
	e.WriteString("A\r\nB\r\nC\r\nD\r\n")
	h := e.HistorySnapshot()
	if len(h.Rows) != 2 || h.Rows[0].ID != 1 || h.Rows[1].ID != 2 || h.NextID != 3 {
		t.Fatalf("eviction identities %#v", h)
	}
	e.ClearScrollback()
	e.ClearScrollback()
	cleared := e.HistorySnapshot()
	if len(cleared.Rows) != 0 || cleared.NextID != 3 || cleared.ClearEpoch != h.ClearEpoch+2 {
		t.Fatalf("clear %#v", cleared)
	}
	sb := e.Scrollback()
	sb.setLimits(2, 1, 10000)
	sb.Push(uv.Line{{Content: "A", Width: 1}, {Content: "B", Width: 1}})
	sb.Push(uv.Line{{Content: "C", Width: 1}})
	h = e.HistorySnapshot()
	if len(h.Rows) != 1 || h.Rows[0].ID != 4 || h.NextID != 5 {
		t.Fatalf("rejected row identity %#v", h)
	}
}

func TestPairHistorySnapshotsOwnCells(t *testing.T) {
	e := historyEmulator(t, 4, 2)
	e.WriteString("A\r\nB\r\nC")
	h := e.HistorySnapshot()
	h.Rows[0].Cells[0].Content = "mutation"
	if e.HistorySnapshot().Rows[0].Cells[0].Content != "A" {
		t.Fatal("snapshot aliases backend")
	}
	e.WriteString("\r\nD\r\nE")
	if h.Rows[0].Cells[0].Content != "mutation" {
		t.Fatal("subsequent output changed snapshot")
	}
}

func TestPairBlankProvenanceAndLegacyRender(t *testing.T) {
	e := historyEmulator(t, 5, 1)
	e.WriteString("A \x1b[4GB")
	if c := e.CellAt(2, 0); c.Content != "" || c.Width != 1 {
		t.Fatalf("unprinted blank %#v", c)
	}
	if e.CellAt(1, 0).Content != " " {
		t.Fatal("printed space lost")
	}
	if e.String() != "A  B" {
		t.Fatalf("legacy String %q", e.String())
	}
	if e.RowMetadata(0).UsedColumns != 4 {
		t.Fatal("wrong content extent")
	}
	e.WriteString("\x1b[2G\x1b[X")
	if e.CellAt(1, 0).Content != "" {
		t.Fatal("erase became printed space")
	}
	e.WriteString("\x1b[1G界\x1b[2GZ")
	if e.CellAt(0, 0).Content != "" || e.CellAt(0, 0).Width != 1 {
		t.Fatal("partial-wide cleanup became printed space")
	}
	if e.String() != " Z B" {
		t.Fatalf("partial wide %q", e.String())
	}
}

func TestPairHistoryPrimaryIsolationAndRegion(t *testing.T) {
	e := historyEmulator(t, 4, 3)
	e.WriteString("A\r\nB\r\nC\r\nD")
	before := e.HistorySnapshot()
	e.WriteString("\x1b[?1049h1\r\n2\r\n3\r\n4\x1b[2J\x1b[3J\x1b[?1049l")
	after := e.HistorySnapshot()
	if after.NextID != before.NextID || after.ClearEpoch != before.ClearEpoch || len(after.Rows) != len(before.Rows) {
		t.Fatal("alternate affected primary history")
	}
	if e.scrs[1].Scrollback() != nil && e.scrs[1].Scrollback().Len() != 0 {
		t.Fatal("alternate retained scrollback")
	}
	e.WriteString("\x1b[2;3r\x1b[3;1H\n")
	if e.HistorySnapshot().NextID != before.NextID {
		t.Fatal("region below top leaked into history")
	}
}

func TestPairRowMetadataMovesAndClears(t *testing.T) {
	e := historyEmulator(t, 4, 4)
	e.WriteString("ABCDEFGHIK")
	if !e.RowMetadata(1).Wrapped || !e.RowMetadata(2).Wrapped {
		t.Fatal("missing wrap")
	}
	e.WriteString("\x1b[2;1H\x1b[L")
	if e.RowMetadata(1).Wrapped || !e.RowMetadata(2).Wrapped || !e.RowMetadata(3).Wrapped {
		t.Fatal("IL did not move metadata")
	}
	e.WriteString("\x1b[M")
	if !e.RowMetadata(1).Wrapped || !e.RowMetadata(2).Wrapped || e.RowMetadata(3).Wrapped {
		t.Fatal("DL did not move metadata")
	}
	e.WriteString("\x1b[1;1H\n")
	if e.RowMetadata(1).Wrapped {
		t.Fatal("LF failed to seal next row")
	}
	e.WriteString("\x1b[2J")
	for y := 0; y < 4; y++ {
		if e.RowMetadata(y).Wrapped || e.RowMetadata(y).UsedColumns != 0 {
			t.Fatal("clear retained row metadata")
		}
	}
}

func TestPairHistoryResizeAndHeldTail(t *testing.T) {
	e := historyEmulator(t, 4, 2)
	e.WriteString("ABCDEFGHI")
	h := e.HistorySnapshot()
	if len(h.Rows) != 1 || !h.ContinuesToScreen || h.Rows[0].Meta.UsedColumns != 4 {
		t.Fatalf("tail %#v", h)
	}
	e.Resize(2, 2)
	after := e.HistorySnapshot()
	if after.NextID != h.NextID+1 || after.ClearEpoch != h.ClearEpoch || len(after.Rows[0].Cells) != 4 {
		t.Fatal("resize changed history")
	}
	e.WriteString("\x1bc")
	if e.HistorySnapshot().NextID != after.NextID || e.RowMetadata(0).Wrapped {
		t.Fatal("RIS identity/metadata mismatch")
	}
}

func TestPairED2HistoryPreservesBlankLogicalSeparators(t *testing.T) {
	e := historyEmulator(t, 4, 4)
	e.WriteString("A\r\n\r\nB\x1b[2J")
	h := e.HistorySnapshot()
	if len(h.Rows) != 3 || h.Rows[1].Meta.UsedColumns != 0 {
		t.Fatalf("blank hard separator lost %#v", h)
	}
	e2 := historyEmulator(t, 2, 3)
	e2.WriteString("A   B\x1b[2J")
	h = e2.HistorySnapshot()
	if len(h.Rows) != 3 || h.Rows[1].Meta.UsedColumns != 2 || !h.Rows[1].Meta.Wrapped || !h.Rows[2].Meta.Wrapped {
		t.Fatalf("printed-space soft row lost %#v", h)
	}
}
