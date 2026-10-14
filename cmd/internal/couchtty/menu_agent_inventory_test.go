package couchtty

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestStartFormsUseProductionAgentInventory(t *testing.T) {
	for _, addSlot := range []bool{false, true} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("add-slot=%v/explicit=%v", addSlot, explicit), func(t *testing.T) {
				row := groupedRow("/workspace/pair", 0, "current")
				state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
				state.RootAgent = "claude"
				var effects []MenuEffect
				if addSlot {
					state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
					state.Frames[len(state.Frames)-1].SelectedItem = "add-slot"
					state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				} else {
					state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
					for _, r := range "/workspace/pair" {
						state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
					}
					state, effects = reduceKey(state, PanelKey{Kind: KeyDown})
				}
				if len(effects) != 1 || effects[0].Preview == nil || effects[0].Preview.Agent != "" {
					t.Fatalf("untouched selection must defer to resolver: %+v", effects)
				}
				initial := *effects[0].Preview
				request := initial
				wantAgent, wantRequested := "muse", ""
				if explicit {
					// Both arrows must work on the real constructor's choices.
					for _, step := range []struct {
						key   PanelKeyKind
						agent string
					}{
						{KeyRight, "codex"}, {KeyLeft, "claude"}, {KeyRight, "codex"},
					} {
						state, effects = reduceKey(state, PanelKey{Kind: step.key})
						if state.CurrentFrame().Agent != step.agent || !state.CurrentFrame().AgentSticky || len(effects) != 1 || effects[0].Preview == nil || effects[0].Preview.Agent != step.agent {
							t.Fatalf("arrow selection %s: frame=%+v effects=%+v", step.agent, state.CurrentFrame(), effects)
						}
					}
					request = *effects[0].Preview
					stale := couchcore.PreparedStart{Resolution: couchcore.StartResolution{Fingerprint: "stale", Profile: couchcore.LaunchProfile{Agent: "muse"}}}
					state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: initial.Generation, Prepared: &stale})
					if state.CurrentFrame().Agent != "codex" || len(effects) != 0 {
						t.Fatal("stale preview replaced explicit selection")
					}
					wantAgent, wantRequested = "codex", "codex"
				}
				prepared := couchcore.PreparedStart{Resolution: couchcore.StartResolution{
					OriginalInput: request.Path, CanonicalPath: request.Path, Action: request.Action,
					RequestedAgent: request.Agent, Profile: couchcore.LaunchProfile{Agent: wantAgent}, Fingerprint: "accepted",
				}}
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: request.Generation, Prepared: &prepared})
				if len(effects) != 0 || state.CurrentFrame().Agent != wantAgent || state.CurrentFrame().AgentSticky != explicit {
					t.Fatalf("preview changed selection semantics: %+v", state.CurrentFrame())
				}
				if !strings.Contains(RenderMenu(state, 100, 30, time.Time{}, false), "agent "+wantAgent) {
					t.Fatal("selected agent missing from form")
				}
				state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				if len(effects) != 1 || effects[0].Operation != "start" || effects[0].Args["agent"] != wantRequested || effects[0].Args["fingerprint"] != "accepted" || effects[0].Args["action"] != "create" {
					t.Fatalf("start did not preserve preview inputs: %+v", effects)
				}
			})
		}
	}
}

func TestMenuAgentInventoryIsCanonicalAndOwned(t *testing.T) {
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	if !slices.Equal(state.Agents, launcher.AgentInventory()) {
		t.Fatalf("agents=%v want canonical inventory %v", state.Agents, launcher.AgentInventory())
	}
	state.Agents[0] = "changed"
	if slices.Contains(NewMenuState(nil, couchcore.ThreadAddress{}).Agents, "changed") {
		t.Fatal("menu states share mutable inventory")
	}
}
