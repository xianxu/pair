package titlepoller

import "testing"

func TestTitleUpdateConfirmsOnlySuccessfulEffect(t *testing.T) {
	var state TitleUpdateState
	desired := TitleUpdate{Target: TitleUpdateTarget{Generation: "launch-a", PaneID: "7"}, Title: "codex (42)"}
	refresh := func() []TitleUpdateEffect {
		return AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates, Titles: []TitleUpdate{desired}})
	}
	first := refresh()
	if len(first) != 1 || first[0].Target != desired.Target || first[0].Title != desired.Title {
		t.Fatalf("first effect = %#v", first)
	}
	// Missing acknowledgement is unknown, so the next refresh retries.
	retry := refresh()
	if len(retry) != 1 || retry[0].ID == first[0].ID {
		t.Fatalf("retry = %#v", retry)
	}
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: retry[0], Success: false}})
	retry = refresh()
	if len(retry) != 1 {
		t.Fatalf("failed effect must retry: %#v", retry)
	}
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: retry[0], Success: true}})
	if got := refresh(); len(got) != 0 {
		t.Fatalf("confirmed title emitted again: %#v", got)
	}
	desired.Title = "codex (43)"
	if got := refresh(); len(got) != 1 || got[0].Title != desired.Title {
		t.Fatalf("changed title = %#v", got)
	}
}

func TestTitleUpdateIgnoresLateCompletions(t *testing.T) {
	var state TitleUpdateState
	target := TitleUpdateTarget{Generation: "launch-a", PaneID: "7"}
	refresh := func(title string) TitleUpdateEffect {
		got := AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates, Titles: []TitleUpdate{{Target: target, Title: title}}})
		if len(got) != 1 {
			t.Fatalf("refresh = %#v", got)
		}
		return got[0]
	}
	old := refresh("old")
	current := refresh("new")
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: old, Success: true}})
	// A late previous desire cannot confirm the current one.
	current = refresh("new")
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: current, Success: true}})
	// Replacing the generation prunes its cache, even for the same pane/title.
	target.Generation = "launch-b"
	replacement := refresh("new")
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: current, Success: true}})
	replacement = refresh("new")
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: replacement, Success: true}})
	got := AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates, Titles: []TitleUpdate{{Target: target, Title: "new"}}})
	if len(got) != 0 {
		t.Fatalf("replacement unconfirmed: %#v", got)
	}
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates})
	if len(state.targets) != 0 {
		t.Fatalf("removed targets retained: %d", len(state.targets))
	}
	refresh("new")
}

func TestTitleUpdatePrunesMissingPanesAndSeparatesTargets(t *testing.T) {
	var state TitleUpdateState
	titles := []TitleUpdate{
		{Target: TitleUpdateTarget{Generation: "a", PaneID: "1"}, Title: "same"},
		{Target: TitleUpdateTarget{Generation: "a", PaneID: "2"}, Title: "same"},
		{Target: TitleUpdateTarget{Generation: "b", PaneID: "1"}, Title: "same"},
	}
	effects := AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates, Titles: titles})
	if len(effects) != 3 {
		t.Fatalf("effects = %#v", effects)
	}
	for _, effect := range effects {
		AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: effect, Success: true}})
	}
	if got := AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates, Titles: titles[:1]}); len(got) != 0 {
		t.Fatalf("retained pane changed: %#v", got)
	}
	if len(state.targets) != 1 {
		t.Fatalf("cache not bounded to snapshot: %d", len(state.targets))
	}
	if got := AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates, Titles: titles}); len(got) != 2 {
		t.Fatalf("removed panes must be unconfirmed: %#v", got)
	}
}

func TestSharedTitleDecisions(t *testing.T) {
	if got := FrameTitle("codex", "42"); got != "codex (42)" {
		t.Fatalf("frame: %q", got)
	}
	if got := PrefixForAge(0); got != "🔴 " {
		t.Fatalf("prefix: %q", got)
	}
	if got := CmuxWorkspaceTitle("🔴 ", "📁pair"); got != "🔴 📁pair" {
		t.Fatalf("workspace: %q", got)
	}
	if ShouldClaimWorkspace("other", "ours", true) {
		t.Fatal("claimed live owner")
	}
	raw := FormatCmuxOwner("ours", "📁pair")
	if tag, session := ParseCmuxOwner(raw); tag != "ours" || session != "📁pair" {
		t.Fatalf("owner: %q %q", tag, session)
	}
}

func TestTitleUpdateUncertainChangedWriteInvalidatesEarlierConfirmation(t *testing.T) {
	var state TitleUpdateState
	target := TitleUpdateTarget{Generation: "a", PaneID: "1"}
	refresh := func(title string) []TitleUpdateEffect {
		return AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: RefreshTitleUpdates, Titles: []TitleUpdate{{Target: target, Title: title}}})
	}
	effect := refresh("old")[0]
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: effect, Success: true}})
	effect = refresh("new")[0]
	AdvanceTitleUpdate(&state, TitleUpdateEvent{Kind: CompleteTitleUpdate, Result: TitleUpdateResult{Effect: effect, Success: false}})
	if got := refresh("old"); len(got) != 1 {
		t.Fatalf("unknown changed write must not preserve old confirmation: %#v", got)
	}
}
