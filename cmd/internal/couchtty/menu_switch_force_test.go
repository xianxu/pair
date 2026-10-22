package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestForceSwitchHasOneLocalConfirmation(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, scenario := range []string{"accept", "cancel", "changed", "no-offer"} {
			t.Run(agent+"/"+scenario, func(t *testing.T) {
				state := NewMenuState(menuThreads(), menuAddress("couch-one"))
				p := couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: agent, SourceRevision: 1, Profile: couchcore.LaunchProfile{Agent: agent, Argv: []string{}}, Fingerprint: "initial"}
				appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: p.Address, Agent: agent, SwitchStage: 1, SwitchPrepared: &p, SelectedItem: "force-switch"})
				state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
				if len(effects) != 1 || effects[0].Preview == nil || !effects[0].Preview.SwitchForce {
					t.Fatalf("force bypassed preview: %+v", effects)
				}
				final := p
				final.Force = &couchcore.AgentSwitchForceOffer{}
				if scenario == "changed" {
					final.SourceRevision++
				}
				if scenario == "no-offer" {
					final.Force = nil
				}
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: effects[0].Preview.Generation, SwitchPrepared: &final})
				if len(effects) != 0 {
					t.Fatal("force ran before confirmation")
				}
				if scenario == "changed" || scenario == "no-offer" {
					if state.CurrentFrame().SwitchStage != 1 {
						t.Fatal("invalid offer reached confirmation")
					}
					return
				}
				body := RenderMenuView(state, 100, 20, time.Now(), false).Body
				if !strings.Contains(body, "Force switch? The current turn's unsaved progress may be lost.") || !strings.Contains(body, "Uncommitted file count is unknown") {
					t.Fatalf("missing exact risk/unknown count: %s", body)
				}
				narrow := state
				narrow.Notice = errorMenuNotice("ordinary failure")
				narrow.CopyFeedback = CopyFeedback{Kind: CopyFeedbackFailed, Text: "clipboard failed"}
				compact := RenderMenuView(narrow, 40, 10, time.Now(), false).Body
				words := strings.Join(strings.Fields(compact), " ")
				if !strings.Contains(words, "unsaved progress may be lost.") || !strings.Contains(compact, "Cancel") || !strings.Contains(words, "Uncommitted file count is unknown") {
					t.Fatalf("compact confirmation lost risk or controls: %s", compact)
				}
				if state.CurrentFrame().SelectedItem != "cancel" {
					t.Fatal("force confirmation defaults to destructive choice")
				}
				if scenario == "cancel" {
					state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
					if len(effects) != 0 || state.CurrentFrame().SwitchStage != 1 {
						t.Fatal("cancelled force had effects")
					}
					return
				}
				state, effects = reduceKey(state, PanelKey{Kind: KeyLeft})
				if len(effects) != 0 {
					t.Fatal("focus acted")
				}
				state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
				if len(effects) != 1 || effects[0].Operation != "switch-agent" {
					t.Fatalf("one confirmation did not execute: %+v", effects)
				}
				if _, ok := effects[0].TypedPayload.(*couchcore.AgentSwitchForceConsent); !ok {
					t.Fatal("force was not local typed consent")
				}
				for key := range effects[0].Args {
					if strings.Contains(key, "force") {
						t.Fatal("force serialized in operation args")
					}
				}
			})
		}
	}
}

func TestForceSwitchNoticeCountsKeptFiles(t *testing.T) {
	n := 4
	if got := forceSwitchFileNotice(&n); got != "4 uncommitted files stay as they are for the new agent" {
		t.Fatal(got)
	}
}

func forceSwitchConfirmationState() MenuState {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	p := &couchcore.PreparedAgentSwitch{Address: menuAddress("couch-one"), SourceAgent: "claude", Profile: couchcore.LaunchProfile{Agent: "codex"}, Force: &couchcore.AgentSwitchForceOffer{}}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Thread: p.Address, Agent: "codex", SwitchStage: 2, SwitchPrepared: p, SelectedItem: "cancel"})
	return state
}

func TestForceSwitchCancellationPreservesInputAndSnapshot(t *testing.T) {
	for _, route := range []struct {
		name string
		key  PanelKey
	}{{"escape", PanelKey{Kind: KeyEscape}}, {"cancel", PanelKey{Kind: KeyEnter}}} {
		t.Run(route.name, func(t *testing.T) {
			state := forceSwitchConfirmationState()
			snapshot := cloneMenuState(state)
			prepared := state.CurrentFrame().SwitchPrepared
			offer := prepared.Force
			before := RenderMenuView(snapshot, 100, 20, time.Time{}, false).Body
			next, effects := reduceKey(state, route.key)
			if len(effects) != 0 || next.CurrentFrame().SwitchStage != 1 || next.CurrentFrame().SwitchPrepared.Force != nil {
				t.Fatalf("cancellation did not discard the next state's offer: %+v, %+v", next.CurrentFrame(), effects)
			}
			if state.CurrentFrame().SwitchStage != 2 || prepared.Force != offer || snapshot.CurrentFrame().SwitchPrepared.Force != offer {
				t.Fatal("cancellation mutated the input or retained snapshot's force offer")
			}
			if next.CurrentFrame().SwitchPrepared == prepared {
				t.Fatal("cancellation retained the shared preview pointer")
			}
			if after := RenderMenuView(snapshot, 100, 20, time.Time{}, false).Body; after != before {
				t.Fatal("cancellation changed the retained snapshot's rendering")
			}
		})
	}
}

func TestForceSwitchCancellationConcurrentSnapshotRender(t *testing.T) {
	for _, route := range []struct {
		name string
		key  PanelKey
	}{{"escape", PanelKey{Kind: KeyEscape}}, {"cancel", PanelKey{Kind: KeyEnter}}} {
		t.Run(route.name, func(t *testing.T) {
			state := forceSwitchConfirmationState()
			snapshot := cloneMenuState(state)
			start, done := make(chan struct{}), make(chan struct{})
			go func() {
				defer close(done)
				<-start
				for range 100 {
					RenderMenuView(snapshot, 100, 20, time.Time{}, false)
				}
			}()
			close(start)
			for range 100 {
				reduceKey(state, route.key)
			}
			<-done
		})
	}
}
