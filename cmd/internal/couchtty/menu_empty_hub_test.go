package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestRootMenuEmptyConfiguredHubsAreDisplayOnly(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "mixed"}[mixed], func(t *testing.T) {
			var rows []couchcore.ActionableThreadSummary
			if mixed {
				rows = []couchcore.ActionableThreadSummary{presentationOrdinary("/src/ops", "live")}
			}
			state := NewMenuState(rows, couchcore.ThreadAddress{})
			state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHubOwnership, HubOwnership: multiHubFixture(t)})
			before := cloneMenuState(state)
			rendered := RenderMenuView(state, 40, 30, time.Time{}, false)
			for _, hub := range []string{"/src/ops", "/src/bench"} {
				text := "Ctrl+Space: start " + hub
				if mixed && hub == "/src/ops" {
					if strings.Contains(rendered.Body, text) {
						t.Fatal("observed hub marked empty")
					}
					continue
				}
				if !strings.Contains(rendered.Body, text) {
					t.Fatalf("missing configured hub action %q: %s", text, rendered.Body)
				}
				for row, line := range strings.Split(rendered.Body, "\r\n") {
					if strings.Contains(line, text) {
						if _, _, ok := rendered.PointToRow(row, 0); ok {
							t.Fatal("empty hub display acquired actor hit extent")
						}
					}
				}
			}
			if !reflect.DeepEqual(cloneMenuState(state), before) {
				t.Fatal("rendering mutated authority or selection")
			}
			if len(rendered.Extents) != len(rows) || len(VisibleMenuThreads(state)) != len(rows) || rendered.Cursor != nil {
				t.Fatalf("empty hub display fabricated interaction: %+v", rendered)
			}
			// Start remains the same explicit form, including on a fully empty list.
			next, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyCtrlSpace}})
			if next.CurrentFrame().Kind != MenuFrameStart {
				t.Fatal("empty hub display blocked existing Start form")
			}
			for _, effect := range effects {
				if effect.Operation != "" {
					t.Fatal("display-only hub launched a process")
				}
			}
		})
	}
}

func TestRootMenuHubPresenceRequiresVerifiedCanonicalInventoryPath(t *testing.T) {
	for _, tc := range []struct {
		name    string
		row     couchcore.ActionableThreadSummary
		missing bool
	}{
		{"verified ordinary", presentationOrdinary("/src/bench", "b"), false},
		{"prefix is another repository", presentationOrdinary("/src/bench-copy", "b"), true},
		{"unverified working path", couchcore.ActionableThreadSummary{Address: menuAddress("unknown"), WorkingPath: "/src/bench", StartingPath: "relative"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewMenuState([]couchcore.ActionableThreadSummary{tc.row}, couchcore.ThreadAddress{})
			state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHubOwnership, HubOwnership: multiHubFixture(t)})
			rendered := RenderMenuView(state, 180, 30, time.Time{}, false)
			if strings.Contains(rendered.Body, "Ctrl+Space: start /src/bench") != tc.missing {
				t.Fatalf("hub presence was guessed: %s", rendered.Body)
			}
		})
	}
}

func TestRootMenuMissingHubDisplayDoesNotDependOnFilterOrAssumedSlotZero(t *testing.T) {
	hubRow := presentationSlot("/src/pair", 9)
	root := hubRow.Target.Slot.WorktreeRoot
	ownership, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{{Root: root, Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/pair", Number: 9}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, present := range []bool{false, true} {
		var inventory []couchcore.ActionableThreadSummary
		if present {
			inventory = []couchcore.ActionableThreadSummary{hubRow}
		}
		state := NewMenuState(inventory, couchcore.ThreadAddress{})
		state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHubOwnership, HubOwnership: ownership})
		state.Frames[0].Filter = "no matching actor"
		rendered := RenderMenuView(state, 180, 20, time.Time{}, false)
		if strings.Contains(rendered.Body, "Ctrl+Space: start "+root) == present {
			t.Fatalf("hub presence used filter or assumed slot zero: %s", rendered.Body)
		}
		if len(rendered.Extents) != 0 {
			t.Fatal("filtered or display-only hub acquired hit target")
		}
	}
}
