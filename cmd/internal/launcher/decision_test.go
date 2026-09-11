package launcher

import (
	"reflect"
	"strings"
	"testing"
)

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

func TestDecideLaunchExplicitAgentArgsCreateWithoutPicker(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{Agent: "codex", AgentExplicit: true, AgentArgsExplicit: true, AgentArgs: []string{"--sandbox", "workspace-write"}}, SessionSnapshot{
		BaseTag:    "pair",
		Sessions:   []Session{{Name: "pair-pair-old", Tag: "old", State: SessionDetached}},
		Historical: []HistoricalTag{{Tag: "pair"}},
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionCreate || decision.Tag != "pair-2" || !decision.PromptName {
		t.Fatalf("decision = %#v, want prompted create for next free explicit-agent tag", decision)
	}
}

func TestDecideLaunchExplicitAgentWithoutSeparatorUsesPickerEvenWithDefaultArgs(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{Agent: "codex", AgentExplicit: true, AgentArgs: []string{"--sandbox", "workspace-write"}}, SessionSnapshot{
		BaseTag:    "pair",
		Sessions:   []Session{{Name: "pair-old", Tag: "old", State: SessionDetached}},
		Historical: []HistoricalTag{{Tag: "pair"}},
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionPick {
		t.Fatalf("decision = %#v, want picker when args came from repo-agent defaults", decision)
	}
}

func TestDecideLaunchExplicitEmptySeparatorCreatesWithoutPicker(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{Agent: "codex", AgentExplicit: true, AgentArgsExplicit: true}, SessionSnapshot{
		BaseTag:    "pair",
		Sessions:   []Session{{Name: "pair-old", Tag: "old", State: SessionDetached}},
		Historical: []HistoricalTag{{Tag: "pair"}},
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionCreate || decision.Tag != "pair-2" || !decision.PromptName {
		t.Fatalf("decision = %#v, want prompted create for explicit empty --", decision)
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

func TestDecideLaunchUsesAssignedSessionName(t *testing.T) {
	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "work"}, SessionSnapshot{
		SessionNames: map[string]string{"work": "📁📁work-2"},
	})
	if err != nil {
		t.Fatalf("DecideLaunch returned error: %v", err)
	}
	if decision.Action != ActionCreate || decision.Tag != "work" || decision.SessionName != "📁📁work-2" {
		t.Fatalf("decision = %#v, want assigned session name", decision)
	}
}

// launchShape is the one switch DecideLaunch branches on, so which of its
// branches read attach state has one owner (pair#228). The `pair -- x` row is
// the one a plausible mistake gets wrong: the agent is DEFAULTED (AgentExplicit
// false) but the args are explicit, and it must create, not pick. The plan
// review applied an AgentExplicit mutation and the rest of the suite stayed
// green; this row is what catches it.
func TestLaunchShapeNamesTheBranchEachArgsShapeTakes(t *testing.T) {
	for _, c := range []struct {
		name string
		args LaunchArgs
		want launchShape
	}{
		{"a picked historical tag", LaunchArgs{SelectedTag: "demo"}, shapeSelected},
		{"pair resume <tag>", LaunchArgs{ForcedTag: "demo"}, shapeForced},
		{"selected wins over forced, as DecideLaunch tests them", LaunchArgs{SelectedTag: "a", ForcedTag: "b"}, shapeSelected},
		{"pair codex -- --flag", LaunchArgs{Agent: "codex", AgentExplicit: true, AgentArgsExplicit: true}, shapeExplicitArgs},
		{"pair -- x (defaulted agent, explicit args)", LaunchArgs{Agent: "claude", AgentExplicit: false, AgentArgsExplicit: true}, shapeExplicitArgs},
		{"pair codex (no separator)", LaunchArgs{Agent: "codex", AgentExplicit: true}, shapeBare},
		{"bare pair", LaunchArgs{}, shapeBare},
	} {
		if got := launchShapeOf(c.args); got != c.want {
			t.Errorf("%s: launchShapeOf = %v, want %v", c.name, got, c.want)
		}
		if got, want := decisionNeedsAttachState(c.args), c.want == shapeBare; got != want {
			t.Errorf("%s: decisionNeedsAttachState = %v, want %v", c.name, got, want)
		}
	}
}

// A decision that reads only liveness decides the same over a liveness
// snapshot as over a full one -- the differential that makes the launcher's
// liveness snapshot safe. Sessions are named so every wrong answer differs.
func TestDecideLaunchOverLivenessMatchesFullSnapshot(t *testing.T) {
	full := []Session{
		{Name: "pair-demo", State: SessionDetached},
		{Name: "pair-busy", State: SessionAttached},
		{Name: "pair-gone", State: SessionExited},
	}
	liveness := make([]Session, len(full))
	for i, s := range full {
		liveness[i] = s
		if s.State != SessionExited {
			liveness[i].State = SessionLive
		}
	}
	for _, args := range []LaunchArgs{
		{ForcedTag: "demo"}, {ForcedTag: "busy"}, {ForcedTag: "gone"}, {ForcedTag: "fresh"},
		{SelectedTag: "demo"},
		{Agent: "codex", AgentExplicit: true, AgentArgsExplicit: true},
		{Agent: "claude", AgentArgsExplicit: true},
	} {
		want, wantErr := DecideLaunch(args, SessionSnapshot{BaseTag: "demo", Sessions: full})
		got, gotErr := DecideLaunch(args, SessionSnapshot{BaseTag: "demo", Sessions: liveness})
		if (wantErr != nil) != (gotErr != nil) || !reflect.DeepEqual(got, want) {
			t.Errorf("args %+v: over liveness = (%+v, %v), over full = (%+v, %v)", args, got, gotErr, want, wantErr)
		}
	}
}

// The guard: a decision that DOES read attach state, handed a snapshot that
// never asked, refuses rather than reading SessionLive as "not detached" -- the
// silent misread that would skip the picker and mint a new session.
func TestDecideLaunchRefusesAttachStateItWasNotGiven(t *testing.T) {
	_, err := DecideLaunch(LaunchArgs{}, SessionSnapshot{BaseTag: "demo", Sessions: []Session{{Name: "pair-demo", State: SessionLive}}})
	if err == nil || !strings.Contains(err.Error(), "attach state") {
		t.Fatalf("err = %v, want a refusal naming the missing attach state", err)
	}
}
