package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func menuAddress(tag string) couchcore.ThreadAddress {
	return couchcore.ThreadAddress{RepoScope: "scope", Tag: couchcore.ThreadTag(tag)}
}

func menuThreads() []couchcore.ActionableThreadSummary {
	return []couchcore.ActionableThreadSummary{
		{Address: menuAddress("couch-one"), WorkingPath: "/repo/one", Name: "compiler", State: couchcore.ThreadLive},
		{Address: menuAddress("couch-two"), WorkingPath: "/repo/two", Name: "review", State: couchcore.ThreadParked},
	}
}

func reduceKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	return ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: key})
}

func TestReduceMenuRootFilteringPreservesStableSelection(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	for _, r := range "rev" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}

	frame := state.CurrentFrame()
	if frame.Kind != MenuFrameRoot || frame.Filter != "rev" || frame.SelectedAddress != menuAddress("couch-two") {
		t.Fatalf("filtered frame = %+v", frame)
	}
	visible := VisibleMenuThreads(state)
	if len(visible) != 1 || visible[0].Address != menuAddress("couch-two") {
		t.Fatalf("visible = %+v", visible)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if got := state.CurrentFrame().SelectedAddress; got != menuAddress("couch-two") {
		t.Fatalf("clearing filter moved stable selection to %v", got)
	}
}

func TestReduceMenuRootZeroSelectionHasNoEffect(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	for _, r := range "absent" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	if got := state.CurrentFrame().SelectedAddress; got != (couchcore.ThreadAddress{}) {
		t.Fatalf("selection = %v, want none", got)
	}

	for _, key := range []PanelKey{{Kind: KeyEnter}, {Kind: KeyTab}} {
		before := state
		var effects []MenuEffect
		state, effects = reduceKey(state, key)
		if len(effects) != 0 || len(state.Frames) != len(before.Frames) || state.Notice != "no selection" {
			t.Fatalf("key %v changed zero-selection state: state=%+v effects=%+v", key.Kind, state, effects)
		}
	}
}

func TestReduceMenuRootEnterDispatchesExactSwitchOrResume(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	liveState, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	wantLive := []MenuEffect{{Operation: "switch", Args: map[string]string{"repo-scope": "scope", "tag": "couch-one"}}}
	if !reflect.DeepEqual(effects, wantLive) || liveState.Notice != "" {
		t.Fatalf("live enter = state %+v effects %+v", liveState, effects)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	_, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	wantParked := []MenuEffect{{Operation: "resume", Args: map[string]string{"repo-scope": "scope", "tag": "couch-two"}}}
	if !reflect.DeepEqual(effects, wantParked) {
		t.Fatalf("parked enter effects = %+v, want %+v", effects, wantParked)
	}
}

func TestReduceMenuActionAndConfirmationCaptureExactThread(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameActions || frame.Thread != menuAddress("couch-one") || frame.SelectedItem != "park" {
		t.Fatalf("action frame = %+v", frame)
	}

	beforeTab := state
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	if !reflect.DeepEqual(state, beforeTab) || len(effects) != 0 {
		t.Fatalf("Tab changed action frame: state=%+v effects=%+v", state, effects)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameConfirmation || frame.Thread != menuAddress("couch-one") || frame.SelectedItem != "cancel" {
		t.Fatalf("confirmation frame = %+v", frame)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	want := []MenuEffect{{Operation: "park", Args: map[string]string{"repo-scope": "scope", "tag": "couch-one"}}}
	if !reflect.DeepEqual(effects, want) {
		t.Fatalf("park effects = %+v, want %+v", effects, want)
	}
}

func TestReduceMenuActionUsesExistingNameOperation(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameText || frame.Action != "name" {
		t.Fatalf("text frame = %+v", frame)
	}
	for _, r := range "new name" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	want := []MenuEffect{{Operation: "name", Args: map[string]string{
		"repo-scope": "scope", "ref": "couch-one", "name": "new name",
	}}}
	if !reflect.DeepEqual(effects, want) {
		t.Fatalf("name effects = %+v, want %+v", effects, want)
	}
}

func TestReduceMenuBellIsEphemeralAndClearedBySwitch(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	original := state
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventBell, Address: menuAddress("couch-one"), Bell: true})
	if !state.Bells[menuAddress("couch-one")] || original.Bells[menuAddress("couch-one")] {
		t.Fatalf("bell ownership aliased: original=%v next=%v", original.Bells, state.Bells)
	}
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if state.Bells[menuAddress("couch-one")] || len(effects) != 1 || effects[0].Operation != "switch" {
		t.Fatalf("switch did not clear bell: state=%+v effects=%+v", state, effects)
	}
}

func TestReduceMenuTextBoundsUTF8AndRestoresActionFrame(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state.Frames[len(state.Frames)-1].Input = strings.Repeat("a", menuNameLimit-1)

	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'é'})
	if got := state.CurrentFrame().Input; len(got) != menuNameLimit-1 || !utf8.ValidString(got) {
		t.Fatalf("multi-byte overflow changed input: len=%d valid=%v", len(got), utf8.ValidString(got))
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: utf8.RuneError})
	if got := state.CurrentFrame().Input; len(got) != menuNameLimit-1 {
		t.Fatalf("RuneError changed input: len=%d", len(got))
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if frame := state.CurrentFrame(); len(state.Frames) != 2 || frame.Kind != MenuFrameActions || frame.SelectedItem != "name" {
		t.Fatalf("Escape did not restore exact action frame: %+v", state.Frames)
	}
}

func TestReduceMenuStartFormKeepsStickyAgentAndOriginatingStack(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude", "codex"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	origin := append([]MenuFrame(nil), state.Frames...)

	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameStart || frame.FormField != MenuFieldPath || frame.Agent != "claude" {
		t.Fatalf("start frame = %+v", frame)
	}
	for _, r := range "/repo/one" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRight})
	if frame := state.CurrentFrame(); frame.Agent != "codex" || !frame.AgentSticky || frame.FormField != MenuFieldAgent {
		t.Fatalf("explicit agent choice = %+v", frame)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyBackspace})
	if frame := state.CurrentFrame(); frame.Agent != "codex" || !frame.AgentSticky || frame.Path != "/repo/on" {
		t.Fatalf("path edit lost sticky agent = %+v", frame)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if !reflect.DeepEqual(state.Frames, origin) {
		t.Fatalf("originating stack = %+v, want %+v", state.Frames, origin)
	}
}

func TestReduceMenuStartFormIsBoundedAndDoesNotNest(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state.Frames[len(state.Frames)-1].Path = strings.Repeat("p", menuTextLimit)
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	if len(state.Frames) != 2 || len(state.CurrentFrame().Path) != menuTextLimit {
		t.Fatalf("start form exceeded structural/input bound: %+v", state.Frames)
	}
}

func TestReduceMenuRefreshReconcilesFramesByIdentity(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state.Frames[len(state.Frames)-1].Input = "draft"

	reordered := []couchcore.ActionableThreadSummary{menuThreads()[1], menuThreads()[0]}
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: reordered})
	if len(effects) != 0 || state.CurrentFrame().Input != "draft" || state.CurrentFrame().Thread != menuAddress("couch-two") {
		t.Fatalf("reordered refresh lost captured state: state=%+v effects=%+v", state, effects)
	}

	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: menuThreads()[:1]})
	if len(effects) != 0 || len(state.Frames) != 1 || state.CurrentFrame().SelectedAddress != menuAddress("couch-one") ||
		!strings.Contains(state.Notice, "review") || !strings.Contains(state.Notice, "scope/couch-two") {
		t.Fatalf("hidden target reconciliation = state %+v effects %+v", state, effects)
	}
}

func TestReduceMenuReconcileKeepsGlobalStartAndDropsInvalidOrigin(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state.Frames[len(state.Frames)-1].Path = "/typed/path"

	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: menuThreads()[1:]})
	if len(state.Frames) != 2 || state.Frames[0].Kind != MenuFrameRoot || state.Frames[1].Kind != MenuFrameStart || state.Frames[1].Path != "/typed/path" {
		t.Fatalf("global start did not survive origin reconciliation: %+v", state.Frames)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(state.Frames) != 1 || state.CurrentFrame().SelectedAddress != menuAddress("couch-two") {
		t.Fatalf("restored root = %+v", state.Frames)
	}
}

func TestReduceMenuOperationResultDoesNotRedispatchAndRestoresByOutcome(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, dispatched := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(dispatched) != 1 || dispatched[0].Operation != "park" {
		t.Fatalf("initial dispatch = %+v", dispatched)
	}

	failed, effects := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "park", Address: menuAddress("couch-one"), Error: "cleanup failed",
		Inventory: menuThreads(), InventorySet: true,
	})
	if len(effects) != 0 || failed.CurrentFrame().Kind != MenuFrameActions || failed.Notice != "cleanup failed" {
		t.Fatalf("failed result = state %+v effects %+v", failed, effects)
	}

	parked := menuThreads()
	parked[0].State = couchcore.ThreadParked
	succeeded, effects := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "park", Address: menuAddress("couch-one"), Success: true,
		Inventory: parked, InventorySet: true,
	})
	if len(effects) != 0 || len(succeeded.Frames) != 1 || succeeded.CurrentFrame().SelectedAddress != menuAddress("couch-one") {
		t.Fatalf("successful result = state %+v effects %+v", succeeded, effects)
	}
	again, effects := ReduceMenu(succeeded, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "park", Address: menuAddress("couch-one"), Success: true,
		Inventory: parked, InventorySet: true,
	})
	if len(effects) != 0 || !reflect.DeepEqual(again.Frames, succeeded.Frames) {
		t.Fatalf("duplicate completion redispatched or moved state: state=%+v effects=%+v", again, effects)
	}
}

func TestReduceMenuRootResumeFailureUsesCapturedOperationOrigin(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, dispatched := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(dispatched) != 1 || dispatched[0].Operation != "resume" {
		t.Fatalf("root resume dispatch = %+v", dispatched)
	}

	failed, effects := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "resume", Address: menuAddress("couch-two"), Error: "resume failed",
		Inventory: menuThreads(), InventorySet: true,
	})
	if len(effects) != 0 || failed.Notice != "resume failed" || failed.CurrentFrame().Kind != MenuFrameRoot || failed.CurrentFrame().SelectedAddress != menuAddress("couch-two") {
		t.Fatalf("root resume failure = state %+v effects %+v", failed, effects)
	}
}

func TestReduceMenuRootResumeSuccessAppliesReturnedInventory(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	resumed := menuThreads()
	resumed[1].State = couchcore.ThreadLive

	succeeded, effects := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "resume", Address: menuAddress("couch-two"), Success: true,
		Inventory: resumed, InventorySet: true,
	})
	thread, found := findMenuThread(succeeded.Inventory, menuAddress("couch-two"))
	if len(effects) != 0 || !found || !thread.Live() || succeeded.CurrentFrame().Kind != MenuFrameRoot || succeeded.CurrentFrame().SelectedAddress != menuAddress("couch-two") {
		t.Fatalf("root resume success = state %+v effects %+v", succeeded, effects)
	}
}

func TestReduceMenuOperationResultRequiresExactCapturedOperation(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	state, dispatched := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(dispatched) != 1 || dispatched[0].Operation != "name" {
		t.Fatalf("rename dispatch = %+v", dispatched)
	}
	state.Notice = "keep"

	before := state
	got, effects := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "describe", Address: menuAddress("couch-one"), Error: "unrelated",
		Inventory: menuThreads()[1:], InventorySet: true,
	})
	if len(effects) != 0 || !reflect.DeepEqual(got, before) {
		t.Fatalf("unrelated completion changed state: got=%+v want=%+v effects=%+v", got, before, effects)
	}
}

func TestReduceMenuOperationResultPreservesHiddenTargetDiagnostic(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})

	got, effects := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "park", Address: menuAddress("couch-one"), Error: "cleanup failed",
		Inventory: menuThreads()[1:], InventorySet: true,
	})
	if len(effects) != 0 || len(got.Frames) != 1 || !strings.Contains(got.Notice, "compiler") ||
		!strings.Contains(got.Notice, "scope/couch-one") || strings.Contains(got.Notice, "cleanup failed") {
		t.Fatalf("hidden operation target = state %+v effects %+v", got, effects)
	}
}

func TestReduceMenuStartCompletionUsesGlobalOperationOrigin(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	frame := &state.Frames[len(state.Frames)-1]
	frame.PreviewAccepted = frame.Generation
	frame.PreviewToken = "accepted"
	frame.PreviewPath = "/repo/new"
	frame.PreviewAgent = "claude"
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "start" {
		t.Fatalf("start dispatch = %+v", effects)
	}
	created := couchcore.ActionableThreadSummary{Address: menuAddress("couch-new"), WorkingPath: "/repo/new", Name: "new", State: couchcore.ThreadLive}
	got, effects := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "start", Address: created.Address, Success: true,
		Inventory: append(menuThreads(), created), InventorySet: true,
	})
	if len(effects) != 0 || len(got.Frames) != 1 || got.CurrentFrame().SelectedAddress != created.Address {
		t.Fatalf("start completion = state %+v effects %+v", got, effects)
	}
}

func TestReduceMenuRefreshKeepsApplicableZeroMatchActionFrame(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	for _, r := range "absent" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameActions || frame.SelectedItem != "" {
		t.Fatalf("zero-match action frame = %+v", frame)
	}

	got, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: menuThreads()})
	if len(effects) != 0 || len(got.Frames) != 2 || got.CurrentFrame().Kind != MenuFrameActions || got.CurrentFrame().Filter != "absent" || got.CurrentFrame().SelectedItem != "" {
		t.Fatalf("unchanged refresh discarded applicable filtered frame: state=%+v effects=%+v", got, effects)
	}
}

func TestReduceMenuConfirmationUsesSharedListFiltering(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	for _, r := range "compiler" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameConfirmation || frame.Filter != "compiler" || frame.SelectedItem != "park" {
		t.Fatalf("filtered confirmation = %+v", frame)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyBackspace})
	if frame := state.CurrentFrame(); frame.Filter != "compile" || frame.SelectedItem != "park" {
		t.Fatalf("confirmation backspace = %+v", frame)
	}
}

func TestReduceMenuGeneratedTracesPreserveStructuralBounds(t *testing.T) {
	keys := []PanelKey{
		{Kind: KeyRune, Rune: 'x'}, {Kind: KeyBackspace}, {Kind: KeyUp},
		{Kind: KeyDown}, {Kind: KeyEnter}, {Kind: KeyTab},
		{Kind: KeyEscape}, {Kind: KeyCtrlSpace},
	}
	declared := map[string]bool{}
	for _, name := range couchcore.OperationNames() {
		declared[name] = true
	}

	var walk func(MenuState, int)
	walk = func(state MenuState, depth int) {
		if depth == 4 {
			return
		}
		for _, key := range keys {
			before := cloneMenuState(state)
			next, effects := reduceKey(state, key)
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("key %v mutated prior state", key.Kind)
			}
			if len(next.Frames) < 1 || len(next.Frames) > 4 || next.Frames[0].Kind != MenuFrameRoot {
				t.Fatalf("key %v produced invalid stack: %+v", key.Kind, next.Frames)
			}
			startFrames := 0
			for _, frame := range next.Frames {
				if frame.Kind == MenuFrameStart {
					startFrames++
				}
				if len(frame.Filter) > menuFilterLimit || len(frame.Input) > menuTextLimit || len(frame.Path) > menuTextLimit ||
					!utf8.ValidString(frame.Filter) || !utf8.ValidString(frame.Input) || !utf8.ValidString(frame.Path) {
					t.Fatalf("key %v violated text bound: %+v", key.Kind, frame)
				}
			}
			if startFrames > 1 {
				t.Fatalf("key %v nested start forms: %+v", key.Kind, next.Frames)
			}
			for _, effect := range effects {
				if effect.Preview != nil {
					if effect.Operation != "" || effect.Preview.Generation == 0 {
						t.Fatalf("key %v emitted malformed preview %+v", key.Kind, effect)
					}
					continue
				}
				if !declared[effect.Operation] {
					t.Fatalf("key %v emitted private operation %+v", key.Kind, effect)
				}
			}
			walk(next, depth+1)
		}
	}

	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude", "codex"}
	state.RootAgent = "claude"
	walk(state, 0)
}
