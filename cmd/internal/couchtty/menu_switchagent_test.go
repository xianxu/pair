package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestSwitchAgentActionIsReachableAndCancelsWithoutMutation(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	if !slices.Contains(menuActionItems(state.Inventory[0]), "switch-agent") {
		t.Fatal("live thread lacks switch-agent action")
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "switch-agent"
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().Action != "switch-agent" {
		t.Fatalf("opening switch: %+v %+v", state.CurrentFrame(), effects)
	}
	state, effects = reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameActions || state.InFlight.Operation != "" {
		t.Fatalf("cancel mutated thread: %+v", state)
	}
}

func TestSwitchAgentEditedEmptyAndStalePreview(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "switch-agent"
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	generation := effects[0].Preview.Generation
	prepared := couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: "codex", Profile: couchcore.LaunchProfile{Agent: state.CurrentFrame().Agent, Argv: []string{"--foo"}}, Fingerprint: "initial"}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, SwitchPrepared: &prepared})
	if state.CurrentFrame().Input != "--foo" {
		t.Fatalf("defaults not prefilled: %+v", state.CurrentFrame())
	}
	for range "--foo" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyBackspace})
	}
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Preview.SwitchArgv != "[]" {
		t.Fatalf("empty args inherited: %+v", effects)
	}
	latest := effects[0].Preview.Generation
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	prepared.Profile.Argv = []string{}
	prepared.Fingerprint = "final"
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: latest, SwitchPrepared: &prepared})
	if state.CurrentFrame().SwitchStage != 1 || state.CurrentFrame().Input != "x" {
		t.Fatalf("stale result accepted: %+v", state.CurrentFrame())
	}
}

func TestSwitchAgentFinalButtonsDispatchExactPreview(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	p := couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: "claude", Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{"--model", "a b"}}, Fingerprint: "accepted"}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: p.Address, SwitchStage: 2, SwitchPrepared: &p, SelectedItem: "cancel"})
	body := RenderMenuView(state, 100, 20, time.Now(), false).Body
	if !strings.Contains(body, "▸ Cancel") || !strings.Contains(body, "claude → codex") {
		t.Fatalf("confirmation hidden: %s", body)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Args["fingerprint"] != "accepted" || effects[0].Args["argv"] != `["--model","a b"]` {
		t.Fatalf("dispatch %+v", effects)
	}
}

func TestSwitchAgentFrameSurvivesOwnParkButRejectsUnrelatedLoss(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "switch-agent"
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	parked := menuThreads()
	parked[0].State = couchcore.ThreadParked
	next, _ := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: parked, Generation: 1})
	if next.CurrentFrame().Kind != MenuFrameSwitchAgent {
		t.Fatal("valid parked source lost form")
	}
	lost := menuThreads()
	lost[0].State = couchcore.ThreadUnusable
	next, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: lost, Generation: 2})
	if next.CurrentFrame().Kind == MenuFrameSwitchAgent {
		t.Fatal("unusable source retained switch form")
	}
}

func TestSwitchAgentLifecyclePolicies(t *testing.T) {
	if !endsItsOwnChild("switch-agent") || !operationNeedsProjectionRefresh("switch-agent") {
		t.Fatal("switch-agent omitted from lifecycle policies")
	}
	for _, state := range []couchcore.ActionableThreadState{couchcore.ThreadBusy, couchcore.ThreadUnusable, couchcore.ThreadDetached} {
		if containsMenuItem(menuActionItems(couchcore.ActionableThreadSummary{State: state}), "switch-agent") {
			t.Fatalf("switch offered for %s", state)
		}
	}
}

func TestSwitchAgentRefusalReturnsToEditableParameters(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Thread: menuAddress("couch-one"), Action: "switch-agent", SwitchStage: 2, Input: "--model draft"})
	state, _ = dispatchThreadOperation(state, "switch-agent", menuAddress("couch-one"))
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "switch-agent", Address: menuAddress("couch-one"), Attempt: state.InFlight.Attempt, Error: "preferences changed; review again"})
	if state.CurrentFrame().SwitchStage != 1 || state.CurrentFrame().Input != "--model draft" {
		t.Fatalf("cannot re-review after refusal: %+v", state.CurrentFrame())
	}
}
