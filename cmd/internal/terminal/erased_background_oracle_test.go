package terminal

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func assertOracleRowPaint(t *testing.T, got, want []oracleCell) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("row paint\ngot %+v\nwant%+v", got, want)
	}
}

func TestPresenterErasedBackgroundViewportAndAltOracle(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, color string
		bg                  int
		rgb                 bool
	}{
		{"normal-indexed", "", "\x1b[41m", 1, false},
		{"alt-indexed", "\x1b[?1049h", "\x1b[48;5;200m", 200, false},
		{"normal-rgb", "", "\x1b[48;2;16;32;48m", 0x102030, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, parent, e, _ := presenterFixture(t, AnyMotion)
			source := tc.prefix + tc.color + "\x1b[2K\x1b[0m\x1b[4GX"
			e.Feed([]byte(source), time.Now())
			selectPresenter(t, p, e)
			direct := runHistoryOracle(t, 8, 5, source)[0]
			got := runHistoryOracle(t, 8, 5, string(parent.Bytes()))[0]
			if direct.Cells[0][0].BG != tc.bg || direct.Cells[0][0].BGRGB != tc.rgb || direct.Cells[0][3].Text != "X" || direct.Cells[0][3].BG != -1 || direct.Cells[0][7].BG != tc.bg {
				t.Fatalf("direct baseline %+v", direct.Cells[0])
			}
			assertOracleRowPaint(t, got.Cells[0], direct.Cells[0])
			f, err := e.Snapshot(time.Now())
			if err != nil {
				t.Fatal(err)
			}
			for _, x := range []int{0, 1, 2, 4, 5, 6, 7} {
				if f.Cells[x].Content != "" || f.Cells[x].Width != 1 {
					t.Fatalf("erased cell became printed text at%d:%+v", x, f.Cells[x])
				}
			}
		})
	}
}

func TestPresenterErasedBackgroundHistoryAppendAndRebuildOracle(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, AnyMotion)
	source := "\x1b[41m\x1b[2K\x1b[0m\x1b[4GX"
	e.Feed([]byte(source), time.Now())
	selectPresenter(t, p, e)
	suffix := "\r\nB\r\nC\r\nD\r\nE"
	e.Feed([]byte(suffix), time.Now())
	if err := p.Present(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if err := p.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	direct := runHistoryOracle(t, 8, 5, "\x1b[1;4r"+source+suffix)[0]
	got := runHistoryOracle(t, 8, 5, string(parent.Bytes()))[0]
	if len(direct.History) != 1 || len(got.History) != 1 {
		t.Fatalf("history counts direct%d got%d", len(direct.History), len(got.History))
	}
	assertOracleRowPaint(t, got.History[0].Cells, direct.History[0].Cells)
	other, err := NewEndpoint("other-erased", Geometry{8, 4}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	selectPresenter(t, p, other)
	selectPresenter(t, p, e)
	rebuilt := runHistoryOracle(t, 8, 5, string(parent.Bytes()))[0]
	if len(rebuilt.History) != 1 {
		t.Fatal("rebuild duplicated history")
	}
	assertOracleRowPaint(t, rebuilt.History[0].Cells, direct.History[0].Cells)
}

func TestPresenterErasedOnlyHistoryRowOracle(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, AnyMotion)
	source := "\x1b[48;2;16;32;48m\x1b[2K\x1b[0m\r\nB\r\nC\r\nD\r\nE"
	e.Feed([]byte(source), time.Now())
	selectPresenter(t, p, e)
	direct := runHistoryOracle(t, 8, 5, "\x1b[1;4r"+source)[0]
	got := runHistoryOracle(t, 8, 5, string(parent.Bytes()))[0]
	if len(direct.History) != 1 || len(got.History) != 1 {
		t.Fatalf("history counts direct%d got%d", len(direct.History), len(got.History))
	}
	for _, c := range direct.History[0].Cells {
		if c.Text != "" || c.BG != 0x102030 {
			t.Fatal("unexpected direct erased-only row")
		}
	}
	assertOracleRowPaint(t, got.History[0].Cells, direct.History[0].Cells)
}

func TestPresenterErasedBackgroundAltReturnOracle(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, AnyMotion)
	normal := "\x1b[41m\x1b[2K\x1b[0m\x1b[4GX"
	alt := "\x1b[?1049h\x1b[44m\x1b[2K\x1b[0m\x1b[3GY"
	e.Feed([]byte(normal), time.Now())
	selectPresenter(t, p, e)
	for _, wire := range []string{alt, "\x1b[?1049l"} {
		e.Feed([]byte(wire), time.Now())
		if err := p.Present(context.Background(), e); err != nil {
			t.Fatal(err)
		}
		if err := p.Flush(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	direct := runHistoryOracle(t, 8, 5, normal+alt+"\x1b[?1049l")[0]
	got := runHistoryOracle(t, 8, 5, string(parent.Bytes()))[0]
	assertOracleRowPaint(t, got.Cells[0], direct.Cells[0])
}

func TestErasedBackgroundSoftGapNativeCopyOracle(t *testing.T) {
	source := "\x1b[41m\x1b[2KAAA界Z\x1b[0m\r\nHARD\r\nONE\r\nTWO\r\nTHR\r\nFOUR"
	publication, frame := historyFixture(t, source)
	wire, _ := renderHistoryBytes(t, Frame{}, frame, publication.History, HistoryState{})
	directWire := "\x1b[1;4r" + source + "\x1b[5;1HCHR"
	direct := runHistoryOracle(t, 4, 5, directWire)[0]
	got := runHistoryOracle(t, 4, 5, string(wire))[0]
	if !reflect.DeepEqual(got.History, direct.History) {
		t.Fatalf("painted soft gap changed history\ngot%+v\nwant%+v", got.History, direct.History)
	}
	nativeDirect := nativeHistoryDump(t, 4, 5, directWire)
	nativePaint := nativeHistoryDump(t, 4, 5, string(wire))
	if nativeDirect != nativePaint {
		t.Fatalf("painted gap copy direct%q painted%q", nativeDirect, nativePaint)
	}
}
