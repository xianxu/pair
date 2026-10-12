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
