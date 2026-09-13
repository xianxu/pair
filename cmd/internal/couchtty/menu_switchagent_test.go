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
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Preview.SwitchArgv != "[]" {
		t.Fatalf("empty args inherited: %+v", effects)
	}
	latest := effects[0].Preview.Generation
	state, _ = reduceKey(state, PanelKey{Kind: KeyUp})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	prepared.Profile.Argv = []string{}
	prepared.Fingerprint = "final"
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: latest, SwitchPrepared: &prepared})
	if state.CurrentFrame().SwitchStage != 1 || state.CurrentFrame().Input != "x" {
		t.Fatalf("stale result accepted: %+v", state.CurrentFrame())
	}
}

func TestSwitchAgentParametersContainFinalButtonsAndDispatchAcceptedPreview(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	p := couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: "claude", Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{"--model", "a b"}}, Fingerprint: "accepted"}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: p.Address, Agent: "codex", SwitchStage: 1, SwitchPrepared: &p, Input: "--model 'a b'", SelectedItem: "parameters"})
	body := RenderMenuView(state, 100, 20, time.Now(), false).Body
	if !strings.Contains(body, "Cancel") || !strings.Contains(body, "claude → codex") || !strings.Contains(body, "▸ parameters") {
		t.Fatalf("parameter form omits final action: %s", body)
	}
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().SelectedItem != "switch" {
		t.Fatalf("input Enter must only move focus: %+v %+v", state.CurrentFrame(), effects)
	}
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Preview == nil || effects[0].Preview.SwitchArgv != `["--model","a b"]` {
		t.Fatalf("Switch did not resolve edited argv: %+v", effects)
	}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: effects[0].Preview.Generation, SwitchPrepared: &p})
	if len(effects) != 1 || effects[0].Args["fingerprint"] != "accepted" || effects[0].Args["argv"] != `["--model","a b"]` || state.CurrentFrame().SwitchStage != 1 {
		t.Fatalf("accepted Switch requires extra confirmation: %+v %+v", state.CurrentFrame(), effects)
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
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Thread: menuAddress("couch-one"), Action: "switch-agent", SwitchStage: 1, Input: "--model draft"})
	state, _ = dispatchThreadOperation(state, "switch-agent", menuAddress("couch-one"))
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "switch-agent", Address: menuAddress("couch-one"), Attempt: state.InFlight.Attempt, Error: "preferences changed; review again"})
	if state.CurrentFrame().SwitchStage != 1 || state.CurrentFrame().Input != "--model draft" {
		t.Fatalf("cannot re-review after refusal: %+v", state.CurrentFrame())
	}
}

func TestSwitchAgentParameterFocusCyclesAndCancelDropsPendingSubmit(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	p := couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: "claude", Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{}}, Fingerprint: "accepted"}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: p.Address, Agent: "codex", SwitchStage: 1, SwitchPrepared: &p, SelectedItem: "parameters"})
	for _, want := range []string{"switch", "cancel", "parameters"} {
		state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
		if state.CurrentFrame().SelectedItem != want {
			t.Fatalf("focus %q want %q", state.CurrentFrame().SelectedItem, want)
		}
		view := RenderMenuView(state, 40, 10, time.Now(), false)
		if (view.Cursor != nil) != (want == "parameters") {
			t.Fatalf("cursor does not follow %s focus: %+v", want, view.Cursor)
		}
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	generation := effects[0].Preview.Generation
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameRoot {
		t.Fatal("Cancel did not leave parameter form untouched")
	}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, SwitchPrepared: &p})
	if len(effects) != 0 || state.InFlight.Operation != "" {
		t.Fatal("cancelled submit dispatched after preview completion")
	}
}

func TestSwitchAgentSourceChangeRequiresReviewOnSameParameterScreen(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	p := couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: "claude", Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{}}, Fingerprint: "initial"}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: p.Address, Agent: "codex", SwitchStage: 1, SwitchPrepared: &p, SelectedItem: "switch"})
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	changed := p
	changed.SourceAgent = "agy"
	changed.Fingerprint = "changed"
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: effects[0].Preview.Generation, SwitchPrepared: &changed})
	if len(effects) != 0 || state.CurrentFrame().SwitchStage != 1 || !strings.Contains(RenderMenuView(state, 100, 20, time.Now(), false).Body, "agy → codex") {
		t.Fatal("source changed without updated same-screen review")
	}
}

func TestSwitchAgentSameAgentReplacementRequiresAnotherAcceptance(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	p := couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: "claude", SourceRevision: 1, Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{}}, Fingerprint: "initial"}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: p.Address, Agent: "codex", SwitchStage: 1, SwitchPrepared: &p, SelectedItem: "switch"})
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	changed := p
	changed.SourceRevision = 2
	changed.Fingerprint = "replacement"
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: effects[0].Preview.Generation, SwitchPrepared: &changed})
	if len(effects) != 0 || state.InFlight.Operation != "" || !strings.Contains(state.Notice.Text, "Source changed") {
		t.Fatal("silently accepted same-agent replacement", state.Notice)
	}
}
