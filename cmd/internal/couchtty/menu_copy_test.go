package couchtty

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
)

func TestMenuCopyFailureSurvivesOperatorAndBackgroundEvents(t *testing.T) {
	for _, kind := range []CopyFeedbackKind{CopyFeedbackFailed, CopyFeedbackUnconfirmed} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			state := NewMenuState(menuThreads(), menuAddress("couch-one"))
			held := CopyFeedback{Kind: kind, Text: "pbcopy exited 7", payload: "original diagnostic"}
			state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventCopyResult, CopyFeedback: held})
			for _, event := range []MenuEvent{
				{Kind: MenuEventTick}, {Kind: MenuEventRefreshStarted},
				{Kind: MenuEventInventory, Error: "inventory failed"},
				{Kind: MenuEventInventory, Inventory: menuThreads()},
				{Kind: MenuEventPalette}, {Kind: MenuEventActivity},
				{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}},
				{Kind: MenuEventKey, Key: PanelKey{Kind: KeyRune, Rune: 'r'}},
				{Kind: MenuEventKey, Key: PanelKey{Kind: KeyTab}},
			} {
				state, _ = ReduceMenu(state, event)
				if state.CopyFeedback != held {
					t.Fatalf("event %v lost retained copy: %+v", event.Kind, state.CopyFeedback)
				}
			}
		})
	}
}

func TestMenuCopyEscapeConsumesOnlyFeedback(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Frames[0].Filter = "rev"
	state.Notice = errorMenuNotice("ordinary operation failed")
	state.CopyFeedback = CopyFeedback{Kind: CopyFeedbackFailed, Text: "clipboard failed", payload: "secret"}
	want := cloneMenuState(state)
	want.CopyFeedback = CopyFeedback{}
	got, effects := reduceKey(state, PanelKey{Kind: KeyEscape})
	if !reflect.DeepEqual(got, want) || len(effects) != 0 {
		t.Fatalf("Escape changed menu rather than just dismissing copy: got=%+v effects=%+v", got, effects)
	}
}

func TestMenuCopyResultsPreserveOrdinaryNoticeAndReleaseRetryPayload(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Notice = errorMenuNotice("operation failed")
	state.CopyFeedback = CopyFeedback{Kind: CopyFeedbackFailed, payload: "private retry"}
	for _, kind := range []CopyFeedbackKind{CopyFeedbackSuccess, CopyFeedbackEmpty} {
		got, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventCopyResult, CopyFeedback: CopyFeedback{Kind: kind, Text: "result", payload: "must not retain"}})
		if got.Notice != state.Notice || len(effects) != 0 {
			t.Fatal("copy result replaced ordinary notice")
		}
		if got.CopyFeedback.Kind != kind || got.CopyFeedback.payload != "" {
			t.Fatalf("result retained payload: %+v", got.CopyFeedback)
		}
		background, _ := ReduceMenu(got, MenuEvent{Kind: MenuEventTick})
		if background.CopyFeedback != got.CopyFeedback {
			t.Fatal("background erased success")
		}
		next, _ := reduceKey(got, PanelKey{Kind: KeyDown})
		if next.CopyFeedback != (CopyFeedback{}) {
			t.Fatal("ordinary action did not retire success")
		}
	}
}

func TestMenuCopyRenderBothBannersWithinBounds(t *testing.T) {
	for _, frameKind := range []MenuFrameKind{MenuFrameRoot, MenuFrameStart} {
		for _, height := range []int{10, 11, 15} {
			t.Run(fmt.Sprintf("%v/%d", frameKind, height), func(t *testing.T) {
				state := NewMenuState(menuThreads(), menuAddress("couch-one"))
				state.Notice = errorMenuNotice("ordinary failure")
				state.CopyFeedback = CopyFeedback{Kind: CopyFeedbackFailed, Text: strings.Repeat("copy failure ", 20), payload: "invisible secret"}
				state.Frames[0].Kind = frameKind
				state.Frames[0].Filter = "r"
				state.Frames[0].FormField = MenuFieldPath
				state.Frames[0].Path = "/tmp"
				state.Frames[0].CompletionCandidates = []string{"/tmp/one", "/tmp/two", "/tmp/three", "/tmp/four", "/tmp/five", "/tmp/six"}
				rendered := RenderMenuView(state, 40, height, time.Time{}, false)
				plain := string(ansi.Strip([]byte(rendered.Body)))
				assertRenderedBounds(t, plain, 40, height)
				if !strings.Contains(plain, "ordinary failure") || !strings.Contains(plain, "Ctrl+Y") || !strings.Contains(plain, "Esc") || !strings.Contains(plain, "copy failure") {
					t.Fatalf("missing independent banner/recovery at 40x%d: %q", height, plain)
				}
				if strings.Contains(plain, "invisible secret") {
					t.Fatal("retry payload rendered")
				}
				lines := strings.Split(plain, "\r\n")
				for _, extent := range rendered.Extents {
					if extent.Start < 0 || extent.End > len(lines) || extent.End <= extent.Start {
						t.Fatalf("invalid extent: %+v lines=%d", extent, len(lines))
					}
				}
				if frameKind == MenuFrameStart && !strings.Contains(plain, "agent ") {
					t.Fatalf("copy banners clipped required start control: %q", plain)
				}
				if frameKind == MenuFrameRoot {
					for _, extent := range rendered.Extents {
						label := "compiler"
						if extent.Thread == menuAddress("couch-two") {
							label = "review"
						}
						if !strings.Contains(lines[extent.Start], label) {
							t.Fatalf("extent points at banner rather than actor: %+v, line=%q", extent, lines[extent.Start])
						}
					}
				}
				if rendered.Cursor == nil {
					t.Fatal("copy banners hid input cursor")
				}
				if cursor := rendered.Cursor; cursor != nil && (cursor.Row < 1 || cursor.Row > len(lines) || cursor.Col < 1 || cursor.Col > 40) {
					t.Fatalf("cursor outside fitted block: %+v", cursor)
				}
			})
		}
	}
}
