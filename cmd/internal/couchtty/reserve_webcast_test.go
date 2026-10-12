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
				if !strings.HasPrefix(row.Body, broadcast.LiveSGR+broadcast.LiveLabel) || row.Webcast.Start <= row.Remote.End {
					t.Fatalf("Chrome cell moved the broadcast's: %q %+v", row.Body, row)
				}
			}
		})
	}
	if row := RenderStatusRow(3, StatusModel{Webcast: WebcastIdle}); row.Webcast.End > 3 {
		t.Fatalf("clipped span %+v", row.Webcast)
	}
}

// The ↗ button (#501) sits right after the Chrome cell, before 💭N, in both
// phases; the live cell wears LIVE's style (#501 revision) without the
// screen broadcast's indicator check reading it as LIVE.
func TestRenderWebcastFocusButtonAndLiveStyle(t *testing.T) {
	for _, cell := range []WebcastCell{WebcastIdle, WebcastLive} {
		row := RenderStatusRow(80, StatusModel{Webcast: cell, WebNotes: 2})
		if row.WebFocus.Start != row.Webcast.End+1 || row.WebFocus.End-row.WebFocus.Start != 1 || row.WebNotes.Start != row.WebFocus.End+1 {
			t.Fatalf("cell %v: spans cell %+v focus %+v notes %+v in %q", cell, row.Webcast, row.WebFocus, row.WebNotes, row.Body)
		}
		if !strings.Contains(row.Body, " "+WebcastFocusLabel+" ") {
			t.Fatalf("no button: %q", row.Body)
		}
	}
	if row := RenderStatusRow(80, StatusModel{}); row.WebFocus != (ColumnSpan{}) || strings.Contains(row.Body, WebcastFocusLabel) {
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
