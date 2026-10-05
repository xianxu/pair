package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestMenuDescriptionTypeaheadBothViewsAndRowKinds(t *testing.T) {
	for name, view := range map[string]MenuRootView{"normal": MenuViewNormal, "focus": MenuViewFocus} {
		for _, kind := range []string{"ordinary", "slot"} {
			for _, source := range []string{"operator", "published"} {
				t.Run(name+"/"+kind+"/"+source, func(t *testing.T) {
					row := menuThreads()[0]
					if kind == "slot" {
						row = menuSlotRow(1, "slot-owner")
					}
					row.State = couchcore.ThreadLive
					row.Description = "Fix Needle regression"
					if source == "published" {
						row.Description = "superseded fallback"
						row.PublishedSummary = "\x1b[31mFix Needle regression\x1b[0m"
					}
					if source == "operator" {
						// The operator description is stored, never matched
						// or displayed (pair#363).
						state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
						state.Frames[0].View, state.Frames[0].Filter = view, "needle"
						if got := VisibleMenuThreads(state); len(got) != 0 {
							t.Fatalf("operator description matched: %+v", got)
						}
						return
					}
					other := menuThreads()[1]
					other.State = couchcore.ThreadLive
					state := NewMenuState([]couchcore.ActionableThreadSummary{row, other}, row.Address)
					state.Frames[0].View = view
					for _, r := range "nEeDlE" {
						state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
					}
					got := VisibleMenuThreads(state)
					if len(got) != 1 || got[0].Address != row.Address {
						t.Fatalf("description match = %+v", got)
					}
					rendered := RenderMenuView(state, 100, 20, time.Time{}, false)
					plain := string(ansi.Strip([]byte(rendered.Body)))
					if !strings.Contains(plain, "Fix Needle regression") {
						t.Fatalf("matched description missing: %q", plain)
					}
					if strings.Contains(plain, "superseded fallback") {
						t.Fatalf("rendered hidden fallback: %q", plain)
					}
					for _, extent := range rendered.Extents {
						for y := extent.Start; y < extent.End; y++ {
							key, _, ok := rendered.PointToRow(y, 1)
							if !ok || key != menuRowKey(row) {
								t.Fatalf("description hit wrong row: %+v", key)
							}
						}
					}
					state.Frames[0].Filter = "superseded"
					if got := VisibleMenuThreads(state); len(got) != 0 {
						t.Fatalf("matched hidden description: %+v", got)
					}
				})
			}
		}
	}
}

func TestMenuDescriptionKeepsExactReferences(t *testing.T) {
	for name, view := range map[string]MenuRootView{"normal": MenuViewNormal, "focus": MenuViewFocus} {
		for _, kind := range []string{"ordinary", "slot"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				first, other := menuThreads()[0], menuThreads()[1]
				query := string(first.Address.Tag)
				if kind == "slot" {
					first, other = menuSlotRow(1, "one"), menuSlotRow(2, "two")
					query = "pair:1"
				}
				first.State, other.State = couchcore.ThreadLive, couchcore.ThreadLive
				first.PublishedSummary, other.PublishedSummary = "unrelated work", "mentions "+query
				state := NewMenuState([]couchcore.ActionableThreadSummary{first, other}, first.Address)
				state.Frames[0].View, state.Frames[0].Filter = view, query
				if got := VisibleMenuThreads(state); len(got) != 1 || got[0].Address != first.Address {
					t.Fatalf("exact reference lost precedence: %+v", got)
				}
			})
		}
	}
}

func TestMenuDescriptionKeepsExactReferencesAcrossRowKinds(t *testing.T) {
	for name, view := range map[string]MenuRootView{"normal": MenuViewNormal, "focus": MenuViewFocus} {
		for _, tc := range []struct {
			name                  string
			targetSlot, otherSlot bool
			query                 string
		}{
			{"ordinary-tag-slot-description", false, true, "exact-owner"},
			{"slot-tag-ordinary-description", true, false, "exact-owner"},
			{"slot-tag-slot-description", true, true, "exact-owner"},
			{"qualified-slot-ordinary-description", true, false, "pair:1"},
			{"numbered-slot-ordinary-description", true, false, ":1"},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				first, other := menuThreads()[0], menuThreads()[1]
				if tc.targetSlot {
					first = menuSlotRow(1, "exact-owner")
				} else {
					first.Address = menuAddress("exact-owner")
				}
				if tc.otherSlot {
					other = menuSlotRow(2, "other-owner")
				}
				first.State, other.State = couchcore.ThreadLive, couchcore.ThreadLive
				first.PublishedSummary, other.PublishedSummary = "target work", "mentions "+tc.query
				state := NewMenuState([]couchcore.ActionableThreadSummary{first, other}, first.Address)
				state.Frames[0].View, state.Frames[0].Filter = view, tc.query
				if got := VisibleMenuThreads(state); len(got) != 1 || menuRowKey(got[0]) != menuRowKey(first) {
					t.Fatalf("mixed exact reference lost precedence: %+v", got)
				}
			})
		}
	}
}
