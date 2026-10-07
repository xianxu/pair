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
	// Add slot is a :0 row's action whatever its agent's state (the Spec
	// table; pair#402): a parked :0 offers it, a slot row does not.
	if slot := groupedRow("/workspace/pair", 2, "current"); slices.Contains(menuActionsFor(NewMenuState(nil, slot.Address), slot), "add-slot") {
		t.Fatal("a slot row offers add slot")
	}
	parked := groupedRow("/workspace/pair", 0, "current")
	parked.State = couchcore.ThreadParked
	if !slices.Contains(menuActionsFor(NewMenuState(nil, parked.Address), parked), "add-slot") {
		t.Fatal("a parked :0 does not offer add slot")
	}
	// The create commit is the same whether :0 is live or parked (pair#402):
	// nothing in the flow resumes :0.
	for _, root := range []string{"/workspace/pair", "/other/pair"} {
		for _, phase := range []couchcore.ActionableThreadState{"", couchcore.ThreadParked} {
			row := groupedRow(root, 0, "current")
			if phase != "" {
				row.State = phase
			}
			row.StartingPath += "/cmd/internal"
			wantPath := root + "/cmd/internal"
			state, effects := addSlotMenu(t, row)
			frame := state.CurrentFrame()
			if frame.Kind != MenuFrameStart || frame.Path != wantPath || frame.FormField != MenuFieldAgent || frame.AgentSticky {
				t.Fatalf("form %+v", frame)
			}
			if len(effects) != 1 || effects[0].Preview == nil || effects[0].Preview.Path != wantPath || effects[0].Preview.Action != couchcore.StartCreate || effects[0].Preview.Agent != "" {
				t.Fatalf("preview %+v", effects)
			}
			prepared := couchcore.PreparedStart{Resolution: couchcore.StartResolution{OriginalInput: wantPath, CanonicalPath: root + "-slot", Action: couchcore.StartCreate, Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{}}, Fingerprint: "reviewed"}}
			state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: effects[0].Preview.Generation, Prepared: &prepared})
			if len(effects) != 0 {
				t.Fatal("preview launched without Enter")
			}
			state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
			if len(effects) != 1 || effects[0].Operation != "start" || effects[0].Args["path"] != wantPath || effects[0].Args["action"] != "create" || effects[0].Args["fingerprint"] != "reviewed" {
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
