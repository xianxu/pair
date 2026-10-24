package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestRenderStatusSendPresence(t *testing.T) {
	for _, tc := range []struct {
		name          string
		model         StatusModel
		terminal, web bool
	}{
		{name: "absent"},
		{name: "flags without casts", model: StatusModel{BroadcastSend: true, WebSend: true}},
		{name: "no pending", model: StatusModel{Broadcast: BroadcastLive, Webcast: WebcastLive}},
		{name: "terminal pending", model: StatusModel{Broadcast: BroadcastLive, BroadcastSend: true}, terminal: true},
		{name: "web pending", model: StatusModel{Webcast: WebcastLive, WebSend: true}, web: true},
		{name: "paused web pending", model: StatusModel{Webcast: WebcastIdle, WebSend: true}, web: true},
		{name: "independent terminal", model: StatusModel{Broadcast: BroadcastLive, Webcast: WebcastLive, BroadcastSend: true}, terminal: true},
		{name: "independent web", model: StatusModel{Broadcast: BroadcastLive, Webcast: WebcastLive, WebSend: true}, web: true},
		{name: "both", model: StatusModel{Broadcast: BroadcastLive, Webcast: WebcastLive, BroadcastSend: true, WebSend: true}, terminal: true, web: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := RenderStatusRow(120, tc.model)
			if (row.BroadcastSend.End > row.BroadcastSend.Start) != tc.terminal || (row.WebSend.End > row.WebSend.Start) != tc.web {
				t.Fatalf("send spans: terminal=%+v web=%+v", row.BroadcastSend, row.WebSend)
			}
			want := 0
			if tc.terminal {
				want++
			}
			if tc.web {
				want++
			}
			if got := strings.Count(row.Body, "⏎"); got != want {
				t.Fatalf("send glyphs=%d want %d: %q", got, want, row.Body)
			}
		})
	}
}

func TestRenderStatusSendClippingAndIndependentSpans(t *testing.T) {
	model := StatusModel{Broadcast: BroadcastLive, BroadcastSend: true, Notes: 12, Webcast: WebcastLive, WebSend: true, WebNotes: 23, WebDrive: "✋名字", WebDrivingOn: true, WebDriverName: "名字", Attention: 2}
	for width := 0; width <= 100; width++ {
		row := RenderStatusRow(width, model)
		plain := string(ansi.Strip([]byte(row.Body)))
		if textwidth.Width(plain) > width {
			t.Fatalf("width %d: overflow %q", width, plain)
		}
		spans := []ColumnSpan{row.Control, row.Pointer, row.Notes, row.BroadcastSend, row.Webcast, row.WebFocus, row.WebPointerLink, row.WebDriveLink, row.WebNotes, row.WebDrive, row.WebSend, row.Attention}
		sends := 0
		for _, send := range []ColumnSpan{row.BroadcastSend, row.WebSend} {
			if send == (ColumnSpan{}) {
				continue
			}
			sends++
			if send.Start < 1 || send.End > width || send.End-send.Start != textwidth.Width("⏎") {
				t.Fatalf("width %d: incomplete send %+v", width, send)
			}
			before := 0
			found := false
			for _, r := range plain {
				if before == send.Start {
					found = r == '⏎'
					break
				}
				before += textwidth.Width(string(r))
			}
			if !found {
				t.Fatalf("width %d: send span %+v not on glyph: %q", width, send, plain)
			}
			for _, other := range spans {
				if other == send || other == (ColumnSpan{}) {
					continue
				}
				if send.Start < other.End && other.Start < send.End {
					t.Fatalf("width %d: overlap %+v %+v", width, send, other)
				}
			}
		}
		if strings.Count(plain, "⏎") != sends {
			t.Fatalf("width %d: unclickable glyph %q", width, plain)
		}
	}
}
