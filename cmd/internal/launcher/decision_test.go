package launcher

import "testing"

func TestDecideLaunchForcedResumeAttachesWhenSessionBlocksReuse(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
		Sessions: []Session{{Name: "pair-demo", State: SessionDetached}},
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionAttach || decision.Tag != "demo" || decision.SessionName != "pair-demo" {
		t.Fatalf("decision = %#v, want attach demo/pair-demo", decision)
	}
}

func TestDecideLaunchForcedResumeCreatesWhenSessionDoesNotBlockReuse(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
		Sessions: []Session{{Name: "pair-demo", State: SessionExited}},
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionCreate || decision.Tag != "demo" || decision.SessionName != "pair-demo" || decision.PromptName {
		t.Fatalf("decision = %#v, want create demo/pair-demo without prompt", decision)
	}
}

func TestDecideLaunchEmptyStateCreatesNextFreeTagWithPrompt(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{Agent: "codex"}, SessionSnapshot{
		BaseTag: "pair",
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionCreate || decision.Tag != "pair" || decision.SessionName != "pair-pair" || !decision.PromptName {
		t.Fatalf("decision = %#v, want create next free tag with prompt", decision)
	}
}

func TestDecideLaunchShowsPickerWhenDetachedOrHistoricalExist(t *testing.T) {
	for _, snap := range []SessionSnapshot{
		{BaseTag: "pair", Sessions: []Session{{Name: "pair-other", State: SessionDetached}}},
		{BaseTag: "pair", Historical: []HistoricalTag{{Tag: "pair-old"}}},
	} {
		decision, err := DecideLaunch(LaunchArgs{Agent: "claude"}, snap)
		if err != nil {
			t.Fatalf("DecideLaunch returned error: %v", err)
		}
		if decision.Action != ActionPick {
			t.Fatalf("decision = %#v, want picker", decision)
		}
	}
}

func TestDecideLaunchHistoricalSelectionCreatesByTag(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{Agent: "claude", SelectedTag: "pair-old"}, SessionSnapshot{
		BaseTag:    "pair",
		Historical: []HistoricalTag{{Tag: "pair-old"}},
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionCreate || decision.Tag != "pair-old" || decision.SessionName != "pair-pair-old" || decision.PromptName {
		t.Fatalf("decision = %#v, want create historical tag without prompt", decision)
	}
}
