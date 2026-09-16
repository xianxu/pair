package terminal

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func TestHistoryDefaultRowsUseOnlyFrameStyleBoundaries(t *testing.T) {
	publication, frame := historyFixture(t, "A\r\nB\r\nC\r\nD\r\nE")
	wire, _ := renderHistoryBytes(t, Frame{}, frame, publication.History, HistoryState{})
	boundary := []byte("\x1b[0m\x1b]8;;\x1b\\")
	if got := bytes.Count(wire, boundary); got != 2 {
		t.Fatalf("default-only rebuild has%d style/link resets; want prologue+final only", got)
	}
}

func TestHistoryCellPainterReturnsPlainRendition(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cell   Cell
		fg, bg int
		bold   bool
	}{
		{"default", Cell{Content: "A", Width: 1}, -1, -1, false},
		{"style", Cell{Content: "A", Width: 1, Style: uv.Style{Fg: ansi.BasicColor(1), Bg: ansi.BasicColor(4), Attrs: uv.AttrBold}}, 1, 4, true},
		{"link", Cell{Content: "A", Width: 1, Link: uv.Link{Params: "id=one", URL: "https://example.com/a;b"}}, -1, -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var wire bytes.Buffer
			e := historyEmitter{write: func(p []byte) error { _, err := wire.Write(p); return err }}
			e.resetStyle()
			e.cells([]Cell{tc.cell}, 1)
			e.cells([]Cell{{Content: "B", Width: 1}}, 1)
			e.flush()
			if e.err != nil {
				t.Fatal(e.err)
			}
			got := runHistoryOracle(t, 4, 2, wire.String())[0]
			a, b := got.Cells[0][0], got.Cells[0][1]
			if a.Text != "A" || a.FG != tc.fg || a.BG != tc.bg || a.Bold != tc.bold || b.Text != "B" || b.FG != -1 || b.BG != -1 || b.Bold {
				t.Fatalf("rendition leaked or lost: %+v", got.Cells[0])
			}
			if tc.name == "link" {
				open := "\x1b]8;id=one;https://example.com/a;b\x1b\\"
				close := "\x1b]8;;\x1b\\"
				tail := strings.SplitN(wire.String(), open, 2)
				if len(tail) != 2 || !strings.Contains(tail[1], close+"B") {
					t.Fatalf("link not closed before plain row: %q", wire.String())
				}
			}
		})
	}
}

func TestHistoryStyledLinkedWrapWireOracle(t *testing.T) {
	source := "\x1b[31m\x1b]8;id=wrap;https://example.com/a;b\x1b\\AA界BB\x1b]8;;\x1b\\\x1b[0mZ\r\nHARD\r\nONE\r\nTWO\r\nTHR\r\nFOUR"
	publication, frame := historyFixture(t, source)
	wire, _ := renderHistoryBytes(t, Frame{}, frame, publication.History, HistoryState{})
	// StyledRows owns and prints the complete four-column chrome row.
	directWire := "\x1b[1;4r" + source + "\x1b[5;1HCHR "
	direct := runHistoryOracle(t, 4, 5, directWire)[0]
	got := runHistoryOracle(t, 4, 5, string(wire))[0]
	if !reflect.DeepEqual(got.History, direct.History) || !reflect.DeepEqual(got.Cells, direct.Cells) || !reflect.DeepEqual(got.Wraps, direct.Wraps) {
		t.Fatalf("styled wrap changed cells/history: got%+v want%+v", got, direct)
	}
	if len(got.Links) == 0 || got.Links[len(got.Links)-1] != ";" {
		t.Fatalf("link left open: %v", got.Links)
	}
	nativeDirect := nativeHistoryDump(t, 4, 5, directWire)
	native := nativeHistoryDump(t, 4, 5, string(wire))
	if native != nativeDirect || native != "AA界BBZ\nHARD\nONE\nTWO\nTHR\nFOUR\nCHR\n" {
		t.Fatalf("native styled copy got%q want%q", native, nativeDirect)
	}
}

func TestHistoryDefaultWideViewportWrapOracle(t *testing.T) {
	source := "AA界B"
	publication, frame := historyFixture(t, source)
	wire, _ := renderHistoryBytes(t, Frame{}, frame, publication.History, HistoryState{})
	directWire := source + "\x1b[5;1HCHR "
	direct := runHistoryOracle(t, 4, 5, directWire)[0]
	got := runHistoryOracle(t, 4, 5, string(wire))[0]
	if !reflect.DeepEqual(got.Cells, direct.Cells) || !reflect.DeepEqual(got.Wraps, direct.Wraps) {
		t.Fatalf("default wide viewport wrap differs: got%+v want%+v", got, direct)
	}
	nativeDirect := nativeHistoryDump(t, 4, 5, directWire)
	native := nativeHistoryDump(t, 4, 5, string(wire))
	if native != nativeDirect || native != "AA界B\n\n\nCHR\n" {
		t.Fatalf("native viewport copy got%q want%q", native, nativeDirect)
	}
}
