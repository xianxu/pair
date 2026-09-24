package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"slices"
	"strings"
	"testing"
)

func addSlotMenu(t *testing.T, row couchcore.ActionableThreadSummary) (MenuState, []MenuEffect) {
	t.Helper()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.RootAgent = "codex"
	if !slices.Contains(menuActionsFor(state, row), "add-slot") {
		t.Fatal("repository row missing Add slot")
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "add-slot"
	return reduceKey(state, PanelKey{Kind: KeyEnter})
}

func TestAddSlotPrefillsExactRepositoryAndUsesCreatePreview(t *testing.T) {
	for _, root := range []string{"/workspace/pair", "/other/pair"} {
		for _, number := range []int{0, 2} {
			row := groupedRow(root, number, "current")
			if number == 0 {
				row.StartingPath = root + "/cmd/internal"
			}
			state, effects := addSlotMenu(t, row)
			frame := state.CurrentFrame()
			if frame.Kind != MenuFrameStart || frame.Path != root || frame.FormField != MenuFieldAgent || frame.AgentSticky {
				t.Fatalf("form %+v", frame)
			}
			if len(effects) != 1 || effects[0].Preview == nil || effects[0].Preview.Path != root || effects[0].Preview.Action != couchcore.StartCreate || effects[0].Preview.Agent != "" {
				t.Fatalf("preview %+v", effects)
			}
			prepared := couchcore.PreparedStart{Resolution: couchcore.StartResolution{OriginalInput: root, CanonicalPath: root + "-slot", Action: couchcore.StartCreate, Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{}}, Fingerprint: "reviewed"}}
			state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: effects[0].Preview.Generation, Prepared: &prepared})
			if len(effects) != 0 {
				t.Fatal("preview launched without Enter")
			}
			state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
			if len(effects) != 1 || effects[0].Operation != "start" || effects[0].Args["path"] != root || effects[0].Args["action"] != "create" || effects[0].Args["fingerprint"] != "reviewed" {
				t.Fatalf("commit %+v", effects)
			}
		}
	}
}
func TestAddSlotCancelAndPreviewRefusalDoNotLaunch(t *testing.T) {
	row := groupedRow("/workspace/pair", 0, "current")
	state, effects := addSlotMenu(t, row)
	generation := effects[0].Preview.Generation
	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	prepared := couchcore.PreparedStart{Resolution: couchcore.StartResolution{Fingerprint: "late"}}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, Prepared: &prepared})
	if state.CurrentFrame().Kind != MenuFrameActions || len(effects) != 0 {
		t.Fatal("cancel accepted late result")
	}
	state, effects = addSlotMenu(t, row)
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: effects[0].Preview.Generation, Error: "resume parked thread first"})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameStart || !strings.Contains(state.Notice.Text, "parked") {
		t.Fatalf("refusal %+v %+v", state, effects)
	}
}
func TestAddSlotDoesNotGuessUnknownOrMalformedRepository(t *testing.T) {
	row := groupedRow("/workspace/pair", 0, "current")
	row.StartingPath = "/elsewhere"
	if slices.Contains(menuActionsFor(NewMenuState(nil, row.Address), row), "add-slot") {
		t.Fatal("guessed unknown root")
	}
	row = groupedRow("/workspace/pair", 1, "current")
	row.Target.Address = row.Address // contradictory tagged target
	row.StartingPath = ""
	if slices.Contains(menuActionsFor(NewMenuState(nil, row.Address), row), "add-slot") {
		t.Fatal("trusted malformed slot")
	}
}
