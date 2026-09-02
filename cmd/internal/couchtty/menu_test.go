package couchtty

import (
	"fmt"
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

func correlatedMenuResult(state MenuState, event MenuEvent) MenuEvent {
	event.Kind = MenuEventOperationResult
	event.Attempt = state.InFlight.Attempt
	return event
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
		if len(effects) != 0 || len(state.Frames) != len(before.Frames) || state.Notice.Text != "no selection" {
			t.Fatalf("key %v changed zero-selection state: state=%+v effects=%+v", key.Kind, state, effects)
		}
	}
}

func TestReduceMenuRootEnterDispatchesExactSwitchOrResume(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	liveState, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	wantLive := []MenuEffect{{Operation: "switch", Attempt: 1, Args: map[string]string{"repo-scope": "scope", "tag": "couch-one"}}}
	if !reflect.DeepEqual(effects, wantLive) || liveState.Notice.Level != MenuNoticeProgress || liveState.Notice.Owner.OperationAttempt != 1 {
		t.Fatalf("live enter = state %+v effects %+v", liveState, effects)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	_, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	wantParked := []MenuEffect{{Operation: "resume", Attempt: 1, Args: map[string]string{"repo-scope": "scope", "tag": "couch-two"}}}
	if !reflect.DeepEqual(effects, wantParked) {
		t.Fatalf("parked enter effects = %+v, want %+v", effects, wantParked)
	}
}

func TestReduceMenuActionAndConfirmationCaptureExactThread(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	// Detach leads the list for a live row: safe before destructive.
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameActions || frame.Thread != menuAddress("couch-one") || frame.SelectedItem != "detach" {
		t.Fatalf("action frame = %+v", frame)
	}
	state = selectMenuItem(state, "park")

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
	want := []MenuEffect{{Operation: "park", Attempt: 1, Args: map[string]string{"repo-scope": "scope", "tag": "couch-one"}}}
	if !reflect.DeepEqual(effects, want) {
		t.Fatalf("park effects = %+v, want %+v", effects, want)
	}
}

func TestReduceMenuActionUsesExistingNameOperation(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state = selectMenuItem(state, "name")
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameText || frame.Action != "name" {
		t.Fatalf("text frame = %+v", frame)
	}
	for _, r := range "new name" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	want := []MenuEffect{{Operation: "name", Attempt: 1, Args: map[string]string{
		"repo-scope": "scope", "ref": "couch-one", "name": "new name",
	}}}
	if !reflect.DeepEqual(effects, want) {
		t.Fatalf("name effects = %+v, want %+v", effects, want)
	}
}

func TestReduceMenuAttentionProjectionIsImmutableByCopy(t *testing.T) {
	threads := menuThreads()
	threads[1].State = couchcore.ThreadLive
	state := NewMenuState(threads, menuAddress("couch-one"))
	state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{
		menuAddress("couch-two"): {{Sequence: 1, Text: "ready"}},
	}
	original := state
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state.Attention[menuAddress("couch-two")][0].Text = "mutated"
	if got := original.Attention[menuAddress("couch-two")][0].Text; got != "ready" {
		t.Fatalf("attention projection aliased: %q", got)
	}
}

func TestReduceMenuTextBoundsUTF8AndRestoresActionFrame(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state = selectMenuItem(state, "name")
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
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRight})
	if frame := state.CurrentFrame(); frame.Agent != "codex" || !frame.AgentSticky || frame.FormField != MenuFieldAgent {
		t.Fatalf("explicit agent choice = %+v", frame)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyUp})
	state, _ = reduceKey(state, PanelKey{Kind: KeyBackspace})
	if frame := state.CurrentFrame(); frame.Agent != "codex" || !frame.AgentSticky || frame.Path != "/repo/on" {
		t.Fatalf("path edit lost sticky agent = %+v", frame)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if !reflect.DeepEqual(state.Frames, origin) {
		t.Fatalf("originating stack = %+v, want %+v", state.Frames, origin)
	}
}

func TestReduceMenuStartCompletionInteractionOwnsKeys(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude", "codex"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 's'})
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	result := CompletionResult{
		Identity: state.CurrentFrame().CompletionRequest,
		Matches:  CompletionMatches{Paths: []string{"sample/", "src/"}},
	}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventCompletionResult, Completion: &result})
	if len(effects) != 0 || state.CurrentFrame().CompletionSelected != 0 || !reflect.DeepEqual(state.CurrentFrame().CompletionCandidates, result.Matches.Paths) {
		t.Fatalf("completion menu = state %+v effects %+v", state, effects)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	if state.CurrentFrame().CompletionSelected != 1 {
		t.Fatalf("Tab selection = %+v", state.CurrentFrame())
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	if state.CurrentFrame().CompletionSelected != 0 || state.CurrentFrame().FormField != MenuFieldPath {
		t.Fatalf("Down selection = %+v", state.CurrentFrame())
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyUp})
	if state.CurrentFrame().CompletionSelected != 1 {
		t.Fatalf("Up selection = %+v", state.CurrentFrame())
	}
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().Path != "src/" || len(state.CurrentFrame().CompletionCandidates) != 0 {
		t.Fatalf("accepted candidate = state %+v effects %+v", state, effects)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	if state.CurrentFrame().FormField != MenuFieldAgent {
		t.Fatalf("Down did not focus agent: %+v", state.CurrentFrame())
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyRight})
	if state.CurrentFrame().Agent != "codex" {
		t.Fatalf("Right did not select agent: %+v", state.CurrentFrame())
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyUp})
	if state.CurrentFrame().FormField != MenuFieldPath {
		t.Fatalf("Up did not focus path: %+v", state.CurrentFrame())
	}
}

func TestReduceMenuStartCompletionEscapeAndImmediateDot(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: '.'})
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	if len(effects) != 0 || state.CurrentFrame().Path != "./" {
		t.Fatalf("dot completion = state %+v effects %+v", state, effects)
	}
	state, effects = reduceKey(state, PanelKey{Kind: KeyTab})
	result := CompletionResult{Identity: state.CurrentFrame().CompletionRequest, Matches: CompletionMatches{Paths: []string{"./one/", "./two/"}}}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventCompletionResult, Completion: &result})
	depth := len(state.Frames)
	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(state.Frames) != depth || len(state.CurrentFrame().CompletionCandidates) != 0 {
		t.Fatalf("first Escape left form or candidates: %+v", state.Frames)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(state.Frames) != depth-1 {
		t.Fatalf("second Escape did not leave form: %+v", state.Frames)
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
	state = selectMenuItem(state, "name")
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state.Frames[len(state.Frames)-1].Input = "draft"

	reordered := []couchcore.ActionableThreadSummary{menuThreads()[1], menuThreads()[0]}
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: reordered})
	if len(effects) != 0 || state.CurrentFrame().Input != "draft" || state.CurrentFrame().Thread != menuAddress("couch-two") {
		t.Fatalf("reordered refresh lost captured state: state=%+v effects=%+v", state, effects)
	}

	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: menuThreads()[:1]})
	if len(effects) != 0 || len(state.Frames) != 1 || state.CurrentFrame().SelectedAddress != menuAddress("couch-one") ||
		!strings.Contains(state.Notice.Text, "review") || !strings.Contains(state.Notice.Text, "scope/couch-two") {
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
	state = selectMenuItem(state, "park")
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, dispatched := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(dispatched) != 1 || dispatched[0].Operation != "park" {
		t.Fatalf("initial dispatch = %+v", dispatched)
	}

	failed, effects := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: "park", Address: menuAddress("couch-one"), Error: "cleanup failed",
		Inventory: menuThreads(), InventorySet: true,
	}))
	if len(effects) != 0 || failed.CurrentFrame().Kind != MenuFrameActions || failed.Notice.Text != "cleanup failed" {
		t.Fatalf("failed result = state %+v effects %+v", failed, effects)
	}

	parked := menuThreads()
	parked[0].State = couchcore.ThreadParked
	completion := correlatedMenuResult(state, MenuEvent{
		Operation: "park", Address: menuAddress("couch-one"), Success: true,
		Inventory: parked, InventorySet: true,
	})
	succeeded, effects := ReduceMenu(state, completion)
	if len(effects) != 0 || len(succeeded.Frames) != 1 || succeeded.CurrentFrame().SelectedAddress != menuAddress("couch-one") {
		t.Fatalf("successful result = state %+v effects %+v", succeeded, effects)
	}
	again, effects := ReduceMenu(succeeded, completion)
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

	failed, effects := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: "resume", Address: menuAddress("couch-two"), Error: "resume failed",
		Inventory: menuThreads(), InventorySet: true,
	}))
	if len(effects) != 0 || failed.Notice.Text != "resume failed" || failed.CurrentFrame().Kind != MenuFrameRoot || failed.CurrentFrame().SelectedAddress != menuAddress("couch-two") {
		t.Fatalf("root resume failure = state %+v effects %+v", failed, effects)
	}
}

func TestReduceMenuRootResumeSuccessAppliesReturnedInventory(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	resumed := menuThreads()
	resumed[1].State = couchcore.ThreadLive

	succeeded, effects := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: "resume", Address: menuAddress("couch-two"), Success: true,
		Inventory: resumed, InventorySet: true,
	}))
	thread, found := findMenuThread(succeeded.Inventory, menuAddress("couch-two"))
	if len(effects) != 0 || !found || !thread.Live() || succeeded.CurrentFrame().Kind != MenuFrameRoot || succeeded.CurrentFrame().SelectedAddress != menuAddress("couch-two") {
		t.Fatalf("root resume success = state %+v effects %+v", succeeded, effects)
	}
}

func TestReduceMenuOperationResultRequiresExactCapturedOperation(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state = selectMenuItem(state, "name")
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	state, dispatched := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(dispatched) != 1 || dispatched[0].Operation != "name" {
		t.Fatalf("rename dispatch = %+v", dispatched)
	}
	state.Notice = infoMenuNotice("keep")

	before := state
	got, effects := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: "describe", Address: menuAddress("couch-one"), Error: "unrelated",
		Inventory: menuThreads()[1:], InventorySet: true,
	}))
	if len(effects) != 0 || !reflect.DeepEqual(got, before) {
		t.Fatalf("unrelated completion changed state: got=%+v want=%+v effects=%+v", got, before, effects)
	}
}

func TestReduceMenuOperationResultPreservesHiddenTargetDiagnostic(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state = selectMenuItem(state, "park")
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})

	got, effects := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: "park", Address: menuAddress("couch-one"), Error: "cleanup failed",
		Inventory: menuThreads()[1:], InventorySet: true,
	}))
	if len(effects) != 0 || len(got.Frames) != 1 || !strings.Contains(got.Notice.Text, "compiler") ||
		!strings.Contains(got.Notice.Text, "scope/couch-one") || strings.Contains(got.Notice.Text, "cleanup failed") {
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
	got, effects := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: "start", Address: created.Address, Success: true,
		Inventory: append(menuThreads(), created), InventorySet: true,
	}))
	if len(effects) != 0 || len(got.Frames) != 1 || got.CurrentFrame().SelectedAddress != created.Address {
		t.Fatalf("start completion = state %+v effects %+v", got, effects)
	}
}

func TestReduceMenuStartFailureWithoutCreatedAddressClearsDispatchAndRestoresForm(t *testing.T) {
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

	got, effects := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: "start", Error: "launch failed",
	}))
	if len(effects) != 0 || got.Notice.Text != "launch failed" || got.CurrentFrame().Kind != MenuFrameStart || got.InFlight.Operation != "" {
		t.Fatalf("failed start completion = state %+v effects %+v", got, effects)
	}
}

func TestMenuOperationCorrelationEnumeratesEveryOperationOutcomeAndAddressShape(t *testing.T) {
	target := menuAddress("couch-one")
	created := menuAddress("couch-new")
	for _, operation := range []string{"switch", "resume", "park", "name", "describe"} {
		origin := MenuOperationOrigin{Operation: operation, Attempt: 1, Address: target}
		for _, success := range []bool{false, true} {
			if !menuOperationMatches(origin, MenuEvent{Operation: operation, Attempt: 1, Address: target, Success: success}) {
				t.Errorf("%s success=%t did not match exact target", operation, success)
			}
			if menuOperationMatches(origin, MenuEvent{Operation: operation, Attempt: 1, Success: success}) {
				t.Errorf("%s success=%t matched missing target", operation, success)
			}
			if menuOperationMatches(origin, MenuEvent{Operation: operation, Attempt: 1, Address: created, Success: success}) {
				t.Errorf("%s success=%t matched wrong target", operation, success)
			}
		}
	}

	start := MenuOperationOrigin{Operation: "start", Attempt: 1}
	if !menuOperationMatches(start, MenuEvent{Operation: "start", Attempt: 1, Error: "launch failed"}) {
		t.Error("failed start without result address did not match")
	}
	if menuOperationMatches(start, MenuEvent{Operation: "start", Attempt: 1, Success: true}) {
		t.Error("successful start without created address matched")
	}
	if !menuOperationMatches(start, MenuEvent{Operation: "start", Attempt: 1, Address: created, Success: true}) {
		t.Error("successful start with created address did not match")
	}
	if !menuOperationMatches(start, MenuEvent{Operation: "start", Attempt: 1, Address: created, Error: "post-create failure"}) {
		t.Error("failed start with a created address did not match")
	}
	if menuOperationMatches(start, MenuEvent{Operation: "resume", Attempt: 1, Address: created, Success: true}) {
		t.Error("start origin matched another operation")
	}
}

func TestMenuOperationAttemptRejectsDelayedDuplicateAcrossEveryOperation(t *testing.T) {
	target := menuAddress("couch-one")
	created := menuAddress("couch-new")
	for _, operation := range []string{"switch", "resume", "park", "name", "describe", "start"} {
		for _, success := range []bool{false, true} {
			t.Run(operation+fmt.Sprintf("/success=%t", success), func(t *testing.T) {
				state := NewMenuState(menuThreads(), target)
				originAddress := target
				resultAddress := target
				if operation == "start" {
					originAddress = couchcore.ThreadAddress{}
					resultAddress = couchcore.ThreadAddress{}
					if success {
						resultAddress = created
					}
				}
				state, effects := dispatchMenuOperation(state, MenuEffect{Operation: operation}, originAddress)
				if len(effects) != 1 || effects[0].Attempt == 0 {
					t.Fatalf("attempt A dispatch = state %+v effects %+v", state, effects)
				}
				attemptA := effects[0].Attempt
				resultA := MenuEvent{Kind: MenuEventOperationResult, Operation: operation, Attempt: attemptA, Address: resultAddress, Success: success, Error: "attempt A failed"}
				state, _ = ReduceMenu(state, resultA)
				state, effects = dispatchMenuOperation(state, MenuEffect{Operation: operation}, originAddress)
				if len(effects) != 1 || effects[0].Attempt == attemptA {
					t.Fatalf("attempt B did not get unique identity: state %+v effects %+v", state, effects)
				}
				before := state
				got, emitted := ReduceMenu(state, resultA)
				if len(emitted) != 0 || !reflect.DeepEqual(got, before) {
					t.Fatalf("stale attempt A changed attempt B: got=%+v want=%+v effects=%+v", got, before, emitted)
				}
			})
		}
	}

	exhausted := NewMenuState(menuThreads(), target)
	exhausted.OperationSequence = ^uint64(0)
	got, effects := dispatchMenuOperation(exhausted, MenuEffect{Operation: "switch"}, target)
	if len(effects) != 0 || got.InFlight.Operation != "" || got.Notice.Text != "operation attempt identity exhausted" {
		t.Fatalf("exhausted operation identity authorized work: state=%+v effects=%+v", got, effects)
	}
}

func TestMenuDispatchInstallsOperationProgressBeforeEffect(t *testing.T) {
	target := menuAddress("couch-one")
	tests := []struct {
		operation string
		address   couchcore.ThreadAddress
		want      string
	}{
		{operation: "start", want: "starting thread"},
		{operation: "resume", address: menuAddress("couch-two"), want: "resuming review"},
		{operation: "park", address: target, want: "parking compiler"},
		{operation: "leave", address: target, want: "leaving couch"},
		{operation: "name", address: target, want: "renaming compiler"},
		{operation: "describe", address: target, want: "saving compiler description"},
	}
	for _, tc := range tests {
		t.Run(tc.operation, func(t *testing.T) {
			state := NewMenuState(menuThreads(), target)
			got, effects := dispatchMenuOperation(state, MenuEffect{Operation: tc.operation}, tc.address)
			if len(effects) != 1 || got.Notice.Level != MenuNoticeProgress || got.Notice.Text != tc.want {
				t.Fatalf("dispatch = notice %+v effects %+v, want progress %q before effect", got.Notice, effects, tc.want)
			}
		})
	}
}

func TestMenuNavigationPreservesCurrentOperationProgress(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, effects := dispatchThreadOperation(state, "resume", menuAddress("couch-two"))
	if len(effects) != 1 {
		t.Fatalf("dispatch effects = %+v", effects)
	}
	progress := state.Notice
	state.SpinnerPhase = 2
	got, emitted := ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}})
	if len(emitted) != 0 || got.Notice != progress || got.SpinnerPhase != 2 {
		t.Fatalf("navigation changed progress: got notice=%+v phase=%d, want notice=%+v phase=2", got.Notice, got.SpinnerPhase, progress)
	}
}

func TestMenuSpinnerTickRequiresExactProgressIdentity(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, effects := dispatchThreadOperation(state, "resume", menuAddress("couch-two"))
	if len(effects) != 1 {
		t.Fatalf("dispatch effects = %+v", effects)
	}
	exact, _ := ReduceMenu(state, MenuEvent{Kind: MenuEventTick, Attempt: effects[0].Attempt})
	if exact.SpinnerPhase != 1 || exact.Notice != state.Notice {
		t.Fatalf("exact tick = notice %+v phase %d, want preserved notice and phase 1", exact.Notice, exact.SpinnerPhase)
	}
	stale, _ := ReduceMenu(exact, MenuEvent{Kind: MenuEventTick, Attempt: effects[0].Attempt + 1})
	if stale.SpinnerPhase != exact.SpinnerPhase || stale.Notice != exact.Notice {
		t.Fatalf("stale tick changed progress: before=%+v/%d after=%+v/%d", exact.Notice, exact.SpinnerPhase, stale.Notice, stale.SpinnerPhase)
	}
}

func TestMenuResolvingProgressOwnsExactPreviewGeneration(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Preview == nil {
		t.Fatalf("preview effects = %+v", effects)
	}
	generation := effects[0].Preview.Generation
	if state.Notice.Level != MenuNoticeProgress || state.Notice.Text != "resolving" || state.Notice.Owner.PreviewGeneration != generation {
		t.Fatalf("resolving notice = %+v, want generation %d", state.Notice, generation)
	}
	got, _ := ReduceMenu(state, MenuEvent{Kind: MenuEventTick, Generation: generation})
	if got.SpinnerPhase != 1 {
		t.Fatalf("preview tick phase = %d, want 1", got.SpinnerPhase)
	}
}

func TestMenuExactSuccessClearsOnlyItsOwnedProgress(t *testing.T) {
	target := menuAddress("couch-two")
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, effects := dispatchThreadOperation(state, "resume", target)
	if len(effects) != 1 {
		t.Fatalf("dispatch effects = %+v", effects)
	}
	event := MenuEvent{
		Kind: MenuEventOperationResult, Operation: "resume", Attempt: effects[0].Attempt,
		Address: target, Success: true,
	}
	cleared, _ := ReduceMenu(state, event)
	if cleared.Notice != (MenuNotice{}) {
		t.Fatalf("owned progress survived exact success: %+v", cleared.Notice)
	}

	state.Notice = errorMenuNotice("inventory failed")
	preserved, _ := ReduceMenu(state, event)
	if preserved.Notice != state.Notice {
		t.Fatalf("exact success erased unrelated error: got %+v want %+v", preserved.Notice, state.Notice)
	}
}

func TestMenuEscapeCancelsResolvingProgress(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	if state.Notice.Level != MenuNoticeProgress {
		t.Fatalf("precondition notice = %+v", state.Notice)
	}
	got, effects := reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(effects) != 0 || got.CurrentFrame().Kind != MenuFrameRoot || got.Notice != (MenuNotice{}) {
		t.Fatalf("escape = frame %+v notice %+v effects %+v", got.CurrentFrame(), got.Notice, effects)
	}
}

func TestMenuUnrelatedInfoDoesNotReplaceCurrentProgress(t *testing.T) {
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state.Notice = MenuNotice{
		Level: MenuNoticeProgress, Text: "resolving",
		Owner: MenuProgressOwner{PreviewGeneration: 7},
	}
	got, _ := ReduceMenu(state, MenuEvent{Kind: MenuEventRefreshStarted})
	if got.Notice != state.Notice || !got.RefreshPending {
		t.Fatalf("refresh start replaced progress: got %+v want %+v", got.Notice, state.Notice)
	}
}

func TestMenuEditingPendingPreviewClearsObsoleteProgress(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	priorGeneration := state.Notice.Owner.PreviewGeneration
	got, effects := reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	if len(effects) != 0 || got.Notice != (MenuNotice{}) || got.CurrentFrame().Generation == priorGeneration {
		t.Fatalf("preview edit = notice %+v generation %d effects %+v; want cleared obsolete progress and new generation", got.Notice, got.CurrentFrame().Generation, effects)
	}
}

func TestMenuParkHotkeyOpensSemanticParkOrLeaveConfirmation(t *testing.T) {
	target := menuAddress("couch-one")
	for _, operation := range []string{"park", "leave"} {
		t.Run(operation, func(t *testing.T) {
			state := NewMenuState(menuThreads(), target)
			got, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: operation, Address: target})
			if len(effects) != 0 || got.CurrentFrame().Kind != MenuFrameConfirmation || got.CurrentFrame().Action != operation || got.CurrentFrame().SelectedItem != "cancel" {
				t.Fatalf("hotkey state=%+v effects=%+v", got, effects)
			}
			got, _ = ReduceMenu(got, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}})
			got, effects = ReduceMenu(got, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
			if len(effects) != 1 || effects[0].Operation != operation || effects[0].Attempt == 0 {
				t.Fatalf("confirmed %s = state %+v effects %+v", operation, got, effects)
			}
		})
	}
}

func TestMenuLeaveEffectCarriesNoThreadArguments(t *testing.T) {
	target := menuAddress("couch-one")
	state := NewMenuState(menuThreads(), target)
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: "leave", Address: target})
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "leave" || len(effects[0].Args) != 0 {
		t.Fatalf("leave effects = %+v, want one argument-free leave", effects)
	}
}

func TestMenuStartSuccessKeepsReturnedSelectionUntilRefreshAddsRow(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state.CurrentFrame()
	state, effects := dispatchMenuOperation(state, MenuEffect{Operation: "start"}, couchcore.ThreadAddress{})
	if len(effects) != 1 {
		t.Fatalf("start effects = %+v", effects)
	}
	created := menuAddress("couch-created")
	state, _ = ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "start", Attempt: effects[0].Attempt,
		Address: created, Success: true,
	})
	if state.CurrentFrame().Kind != MenuFrameRoot || state.CurrentFrame().SelectedAddress != created {
		t.Fatalf("pre-refresh start selection = %+v, want returned address %v", state.CurrentFrame(), created)
	}
	rows := append(menuThreads(), couchcore.ActionableThreadSummary{Address: created, Name: "created", State: couchcore.ThreadLive})
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: rows, InventorySet: true})
	if state.CurrentFrame().SelectedAddress != created {
		t.Fatalf("post-refresh selection = %v, want %v", state.CurrentFrame().SelectedAddress, created)
	}
}

func TestMenuHorizontalArrowsNavigateFrameHierarchy(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, effects := reduceKey(state, PanelKey{Kind: KeyRight})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameActions {
		t.Fatalf("root Right = state %+v effects %+v, want actions", state, effects)
	}
	state = selectMenuItem(state, "park")
	state, effects = reduceKey(state, PanelKey{Kind: KeyRight})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameConfirmation {
		t.Fatalf("actions Right = state %+v effects %+v, want confirmation", state, effects)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyLeft})
	if state.CurrentFrame().Kind != MenuFrameActions {
		t.Fatalf("confirmation Left = %+v, want actions", state.CurrentFrame())
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyLeft})
	if state.CurrentFrame().Kind != MenuFrameRoot {
		t.Fatalf("actions Left = %+v, want root", state.CurrentFrame())
	}
}

func TestMenuOperationCompletionNeverMistakesReplacementFrameForOrigin(t *testing.T) {
	target := menuAddress("couch-one")
	created := menuAddress("couch-new")
	for _, tc := range []struct {
		operation string
		kind      MenuFrameKind
		action    string
	}{
		{operation: "switch", kind: MenuFrameRoot},
		{operation: "resume", kind: MenuFrameRoot},
		{operation: "park", kind: MenuFrameConfirmation, action: "park"},
		{operation: "name", kind: MenuFrameText, action: "name"},
		{operation: "describe", kind: MenuFrameText, action: "describe"},
		{operation: "start", kind: MenuFrameStart},
	} {
		for _, success := range []bool{false, true} {
			t.Run(tc.operation+fmt.Sprintf("/success=%t", success), func(t *testing.T) {
				state := NewMenuState(menuThreads(), target)
				if tc.kind != MenuFrameRoot {
					if !appendMenuFrame(&state, MenuFrame{Kind: tc.kind, Thread: target, Action: tc.action}) {
						t.Fatal("could not append origin frame")
					}
				}
				originAddress := target
				resultAddress := target
				if tc.operation == "start" {
					originAddress = couchcore.ThreadAddress{}
					resultAddress = couchcore.ThreadAddress{}
					if success {
						resultAddress = created
					}
				}
				state, effects := dispatchMenuOperation(state, MenuEffect{Operation: tc.operation}, originAddress)
				if len(effects) != 1 {
					t.Fatalf("dispatch = %+v", effects)
				}
				originInstance := state.InFlight.FrameInstance
				replacement := state.CurrentFrame()
				state.FrameSequence++
				replacement.Instance = state.FrameSequence
				state.Frames[len(state.Frames)-1] = replacement
				if replacement.Instance == originInstance {
					t.Fatal("replacement reused origin frame identity")
				}
				beforeFrames := append([]MenuFrame(nil), state.Frames...)
				result := correlatedMenuResult(state, MenuEvent{Operation: tc.operation, Address: resultAddress, Success: success, Error: "failed"})
				got, _ := ReduceMenu(state, result)

				// A replacement start frame is an independent global overlay opened
				// after dispatch; unlike a replacement thread-local frame, successful
				// completion does not own it.
				globalSuccess := success && (tc.operation == "resume" || tc.operation == "park")
				if globalSuccess {
					if len(got.Frames) != 1 {
						t.Fatalf("global success did not restore root: %+v", got.Frames)
					}
				} else if !reflect.DeepEqual(got.Frames, beforeFrames) {
					t.Fatalf("completion mutated replacement frame: got=%+v want=%+v", got.Frames, beforeFrames)
				}
			})
		}
	}

	exhausted := NewMenuState(menuThreads(), target)
	exhausted.FrameSequence = ^uint64(0)
	before := append([]MenuFrame(nil), exhausted.Frames...)
	got, effects := reduceKey(exhausted, PanelKey{Kind: KeyTab})
	if len(effects) != 0 || !reflect.DeepEqual(got.Frames, before) || got.Notice.Text != "menu frame identity exhausted" {
		t.Fatalf("exhausted frame identity authorized navigation: state=%+v effects=%+v", got, effects)
	}
}

// selectMenuItem picks an action by NAME rather than by counting Down presses.
// The action list's order is a deliberate product decision (detach leads park,
// safe before destructive) and has changed once already; a positional fixture
// silently retargets to a different operation when it does.
func selectMenuItem(state MenuState, item string) MenuState {
	next := cloneMenuState(state)
	next.Frames[len(next.Frames)-1].SelectedItem = item
	return next
}

func TestMenuOperationCompletionPreservesLaterGlobalStartOverlay(t *testing.T) {
	for _, operation := range []string{"switch", "resume", "park", "name", "describe", "start"} {
		for _, success := range []bool{false, true} {
			t.Run(operation+fmt.Sprintf("/success=%t", success), func(t *testing.T) {
				state := NewMenuState(menuThreads(), menuAddress("couch-one"))
				state.Agents = []string{"claude"}
				state.RootAgent = "claude"
				var effects []MenuEffect
				switch operation {
				case "switch":
					state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				case "resume":
					state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
					state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				case "park":
					state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
					state = selectMenuItem(state, "park")
					state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
					state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
					state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				case "name", "describe":
					state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
					state = selectMenuItem(state, operation)
					state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
					state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				case "start":
					state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
					frame := &state.Frames[len(state.Frames)-1]
					frame.PreviewAccepted = frame.Generation
					frame.PreviewToken = "accepted"
					frame.PreviewPath = "/repo/new"
					frame.PreviewAgent = "claude"
					state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				}
				if len(effects) != 1 || effects[0].Operation != operation {
					t.Fatalf("dispatch = state %+v effects %+v", state, effects)
				}

				if state.CurrentFrame().Kind == MenuFrameText || state.CurrentFrame().Kind == MenuFrameStart {
					state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
				}
				state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
				if state.CurrentFrame().Kind != MenuFrameStart {
					t.Fatalf("could not open later global start: %+v", state.Frames)
				}
				overlay := state.CurrentFrame()
				address := state.InFlight.Address
				if operation == "start" && success {
					address = menuAddress("couch-new")
				}
				got, emitted := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
					Operation: operation, Address: address, Success: success, Error: "failed",
				}))
				if len(emitted) != 0 || got.CurrentFrame().Kind != MenuFrameStart || got.CurrentFrame().Instance != overlay.Instance {
					t.Fatalf("completion discarded later global start: state=%+v effects=%+v want overlay=%+v", got, emitted, overlay)
				}
			})
		}
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
	state = selectMenuItem(state, "park")
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
				if effect.Completion != nil {
					if effect.Operation != "" || effect.Completion.Identity.FrameInstance == 0 || effect.Completion.Identity.Generation == 0 {
						t.Fatalf("key %v emitted malformed completion %+v", key.Kind, effect)
					}
					continue
				}
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

// Leave is reachable from a couch with NO live thread. That is not an exotic
// case: once leaving detaches rather than parks (#170), an all-detached couch is
// the normal state the operator quits from, and the root actor that used to lend
// `leave` an address is gone.
func TestLeaveConfirmationNeedsNoLiveThread(t *testing.T) {
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state.InventoryReady = true

	state, _ = reduceParkHotkey(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: "leave"})

	if len(state.Frames) != 2 || state.Frames[1].Kind != MenuFrameConfirmation {
		t.Fatalf("frames = %+v, want a confirmation frame over an empty inventory", state.Frames)
	}
	if state.Frames[1].Thread != (couchcore.ThreadAddress{}) {
		t.Fatalf("leave confirmation bound thread %+v, want none -- it is about couch",
			state.Frames[1].Thread)
	}
	if state.Notice.Level == MenuNoticeError {
		t.Fatalf("leave refused with %q", state.Notice.Text)
	}
}

// The site a keystroke-only test cannot reach: reconcileMenuFrames runs on the
// next inventory refresh, so a leave confirmation that survives every keypress
// can still be dropped a second later by a background refresh.
func TestLeaveConfirmationSurvivesAnInventoryRefresh(t *testing.T) {
	live := couchcore.ActionableThreadSummary{
		Address: couchcore.ThreadAddress{RepoScope: "s", Tag: "t"}, Name: "one", State: couchcore.ThreadLive,
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{live}, live.Address)
	state.InventoryReady = true
	state, _ = reduceParkHotkey(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: "leave"})
	if len(state.Frames) != 2 {
		t.Fatalf("frames = %+v, want the leave confirmation", state.Frames)
	}

	// Every thread goes away -- exactly what `leave` itself causes.
	state.Inventory = nil
	state = reconcileMenuFrames(state)

	if len(state.Frames) != 2 || state.Frames[1].Kind != MenuFrameConfirmation ||
		state.Frames[1].Action != "leave" {
		t.Fatalf("frames after refresh = %+v, want the leave confirmation retained", state.Frames)
	}
}

// The counterpart: a PARK confirmation is about a thread and must still vanish
// with it. Asserted so the global-frame predicate cannot be widened by accident.
func TestParkConfirmationStillDiesWithItsThread(t *testing.T) {
	live := couchcore.ActionableThreadSummary{
		Address: couchcore.ThreadAddress{RepoScope: "s", Tag: "t"}, Name: "one", State: couchcore.ThreadLive,
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{live}, live.Address)
	state.InventoryReady = true
	state, _ = reduceParkHotkey(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: "park", Address: live.Address})
	if len(state.Frames) != 2 {
		t.Fatalf("frames = %+v, want the park confirmation", state.Frames)
	}

	state.Inventory = nil
	state = reconcileMenuFrames(state)

	if len(state.Frames) != 1 {
		t.Fatalf("frames after refresh = %+v, want the park confirmation dropped with its thread", state.Frames)
	}
}

// A live row leads with detach and a resumable row offers resume: the action
// list is where the operator discovers that park is not the only way to put a
// thread down. Detach first because it is safe and park is not.
func TestMenuActionItemsLeadWithTheSafeAction(t *testing.T) {
	live := couchcore.ActionableThreadSummary{State: couchcore.ThreadLive}
	if got := menuActionItems(live); len(got) == 0 || got[0] != "detach" {
		t.Fatalf("live actions = %v, want detach first", got)
	}
	if !containsMenuItem(menuActionItems(live), "park") {
		t.Fatalf("live actions = %v, want park still offered", menuActionItems(live))
	}
	for _, state := range []couchcore.ActionableThreadState{couchcore.ThreadParked, couchcore.ThreadDetached} {
		row := couchcore.ActionableThreadSummary{State: state}
		got := menuActionItems(row)
		if len(got) == 0 || got[0] != "resume" {
			t.Fatalf("%s actions = %v, want resume first", state, got)
		}
		if containsMenuItem(got, "detach") || containsMenuItem(got, "park") {
			t.Fatalf("%s actions = %v, want no detach/park on a row with no client", state, got)
		}
	}
}

// reduceRootKey routes Enter by what the row can do: live rows switch, parked
// and detached rows both resume.
func TestReduceRootKeyEnterRoutesByRowState(t *testing.T) {
	for _, test := range []struct {
		state couchcore.ActionableThreadState
		want  string
	}{
		{couchcore.ThreadLive, "switch"},
		{couchcore.ThreadParked, "resume"},
		{couchcore.ThreadDetached, "resume"},
	} {
		address := couchcore.ThreadAddress{RepoScope: "scope", Tag: "couch-one"}
		state := NewMenuState([]couchcore.ActionableThreadSummary{
			{Address: address, Name: "one", State: test.state},
		}, address)
		state.InventoryReady = true
		_, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
		if len(effects) != 1 || effects[0].Operation != test.want {
			t.Fatalf("%s Enter = %+v, want one %q effect", test.state, effects, test.want)
		}
	}
}
