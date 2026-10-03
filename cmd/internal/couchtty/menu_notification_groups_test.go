package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestMenuNotificationGroupsFollowSlotIndent(t *testing.T) {
	primary := presentationOrdinary("/src/pair", "primary")
	slot := menuSlotRow(2, "secondary")
	for _, row := range []*couchcore.ActionableThreadSummary{&primary, &slot} {
		row.State = couchcore.ThreadLive
		row.PublishedSummary = "work"
	}
	for _, mode := range []MenuRootView{MenuViewNormal, MenuViewFocus} {
		for _, color := range []bool{false, true} {
			state := NewMenuState([]couchcore.ActionableThreadSummary{primary, slot}, primary.Address)
			state.Frames[0].View = mode
			state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{
				primary.Address: {{Text: ""}, {Text: "primary first"}, {Text: ""}, {Text: "primary last"}, {Text: ""}},
				slot.Address:    {{Text: ""}, {Text: "slot only"}, {Text: ""}},
			}
			view := RenderMenuView(state, 80, 20, time.Time{}, color)
			plain := string(ansi.Strip([]byte(view.Body)))
			for _, want := range []string{"\r\n  ├─ primary first\r\n  └─ primary last", "\r\n    └─ slot only"} {
				if !strings.Contains(plain, want) {
					t.Fatalf("mode %v color %v missing %q: %q", mode, color, want, plain)
				}
			}
			if len(view.Extents) != 2 {
				t.Fatalf("extents: %+v", view.Extents)
			}
			for i, row := range []couchcore.ActionableThreadSummary{primary, slot} {
				extent := view.Extents[i]
				wantLines := 3
				if i == 1 {
					wantLines = 2
				}
				if extent.End-extent.Start != wantLines {
					t.Fatalf("empty messages created rows: %+v", extent)
				}
				for y := extent.Start; y < extent.End; y++ {
					key, address, ok := view.PointToRow(y, 4)
					if !ok || key != menuRowKey(row) || address != row.Address {
						t.Fatalf("row %d lost owner", y)
					}
				}
			}
			next, _ := reduceKey(state, PanelKey{Kind: KeyDown})
			if next.CurrentFrame().SelectedKey != slot.RowKey {
				t.Fatalf("navigation did not skip notifications: %+v", next.CurrentFrame())
			}
			state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{slot.Address: {{Text: ""}}}
			empty := RenderMenuView(state, 80, 20, time.Time{}, color)
			if len(empty.Extents) != 2 || empty.Extents[0].End-empty.Extents[0].Start != 1 || empty.Extents[1].End-empty.Extents[1].Start != 1 {
				t.Fatalf("empty notification groups occupy rows: %+v", empty.Extents)
			}
		}
	}
}

func TestMenuNotificationGroupsClipWithoutChangingLastMessage(t *testing.T) {
	slot := menuSlotRow(2, "secondary")
	slot.State = couchcore.ThreadLive
	primary := presentationOrdinary("/src/pair", "primary")
	primary.State = couchcore.ThreadLive
	state := NewMenuState([]couchcore.ActionableThreadSummary{primary, slot}, primary.Address)
	messages := make([]AttentionMessage, 12)
	for i := range messages {
		messages[i].Text = strings.Repeat("界", 40)
	}
	state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{slot.Address: messages}
	for _, height := range []int{10, 20} {
		view := RenderMenuView(state, 40, height, time.Time{}, false)
		plain := string(ansi.Strip([]byte(view.Body)))
		assertRenderedBounds(t, plain, 40, height)
		if !strings.Contains(plain, "\r\n    ├─ ") {
			t.Fatalf("missing clipped branch: %q", plain)
		}
		if strings.Contains(plain, "└─") != (height == 20) {
			t.Fatalf("height %d changed logical final connector: %q", height, plain)
		}
		lines := strings.Split(plain, "\r\n")
		for y, line := range lines {
			if strings.ContainsAny(line, "├└") {
				key, address, ok := view.PointToRow(y, 4)
				if !ok || key != slot.RowKey || address != slot.Address {
					t.Fatalf("clipped message lost owner at %d", y)
				}
			}
		}
	}
}

func TestMenuNotificationGroupKeepsSelectedOwnerWhenTooTall(t *testing.T) {
	slot := menuSlotRow(2, "secondary")
	slot.State = couchcore.ThreadLive
	state := NewMenuState([]couchcore.ActionableThreadSummary{slot}, slot.Address)
	messages := make([]AttentionMessage, 12)
	for i := range messages {
		messages[i].Text = "message"
	}
	state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{slot.Address: messages}
	view := RenderMenuView(state, 40, 10, time.Time{}, false)
	plain := string(ansi.Strip([]byte(view.Body)))
	if !strings.Contains(plain, "pair:2") || strings.Contains(plain, "└─") {
		t.Fatalf("oversized selected group lost its owner or skipped to the tail: %q", plain)
	}
	assertRenderedBounds(t, plain, 40, 10)
}
