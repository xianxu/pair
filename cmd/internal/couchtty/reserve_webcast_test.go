package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func TestRenderWebcastCell(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model StatusModel
		label string
	}{
		{"off", StatusModel{}, ""},
		{"idle", StatusModel{Webcast: WebcastIdle}, WebcastIdleLabel},
		{"live", StatusModel{Webcast: WebcastLive}, WebcastLiveLabel},
		{"after a live screen broadcast", StatusModel{Broadcast: BroadcastLive, Webcast: WebcastLive}, WebcastLiveLabel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := RenderStatusRow(80, tc.model)
			if tc.label == "" {
				if row.Webcast != (ColumnSpan{}) || strings.Contains(row.Body, "Chrome") {
					t.Fatalf("drawn: %q %+v", row.Body, row.Webcast)
				}
				return
			}
			if !strings.Contains(row.Body, tc.label) || row.Webcast.End-row.Webcast.Start != len([]rune(tc.label))+0 {
				t.Fatalf("body %q span %+v", row.Body, row.Webcast)
			}
			if tc.model.Broadcast == BroadcastLive {
				// The screen broadcast's cells keep their fixed columns.
				if !strings.HasPrefix(row.Body, broadcast.LiveSGR+broadcast.LiveLabel) || row.Webcast.Start <= row.Pointer.End {
					t.Fatalf("Chrome cell moved the broadcast's: %q %+v", row.Body, row)
				}
			}
		})
	}
	if row := RenderStatusRow(3, StatusModel{Webcast: WebcastIdle}); row.Webcast.End > 3 {
		t.Fatalf("clipped span %+v", row.Webcast)
	}
}

// The Chrome group (#501) reads `Chrome ⏸ ↗ 👆 👽 💭N`, in both phases, each
// button its own span; the live cell wears LIVE's style without the screen
// broadcast's indicator check reading it as LIVE.
func TestRenderWebcastFocusButtonAndLiveStyle(t *testing.T) {
	for _, cell := range []WebcastCell{WebcastIdle, WebcastLive} {
		row := RenderStatusRow(80, StatusModel{Webcast: cell, WebNotes: 2})
		spans := []ColumnSpan{row.Webcast, row.WebFocus, row.WebPointerLink, row.WebDriveLink, row.WebNotes}
		widths := []int{len([]rune(WebcastLiveLabel)), 1, 2, 2, 3}
		for i, s := range spans {
			if s.End-s.Start != widths[i] || (i > 0 && s.Start != spans[i-1].End+1) {
				t.Fatalf("cell %v: span %d %+v in %+v (%q)", cell, i, s, spans, row.Body)
			}
		}
		want := webcastButtons + " " + broadcast.NoteGlyph + "2"
		if !strings.Contains(row.Body, want) {
			t.Fatalf("group drawn as %q", row.Body)
		}
	}
	if row := RenderStatusRow(80, StatusModel{}); row.WebFocus != (ColumnSpan{}) || row.WebPointerLink != (ColumnSpan{}) || row.WebDriveLink != (ColumnSpan{}) || strings.Contains(row.Body, WebcastFocusLabel) {
		t.Fatalf("button without Chrome: %q", row.Body)
	}
	live := RenderStatusRow(80, StatusModel{Webcast: WebcastLive})
	if !strings.HasPrefix(live.Body, broadcast.LiveSGR+WebcastLiveLabel+"\x1b[0m") {
		t.Fatalf("live cell not in LIVE's style: %q", live.Body)
	}
	if broadcast.IndicatorShown(rowFrame(t, 80, live.Body)) {
		t.Fatal("the screen broadcast reads Chrome ⏸ as its LIVE")
	}
	if idle := RenderStatusRow(80, StatusModel{Webcast: WebcastIdle}); !strings.HasPrefix(idle.Body, WebcastIdleLabel) {
		t.Fatalf("idle cell styled: %q", idle.Body)
	}
}

// webcastButtons is the Chrome group's ↗ 👆 👽 run after the cell (#501).
const webcastButtons = " " + WebcastFocusLabel + " " + broadcast.PointerLabel + " " + WebcastDriveLinkLabel
