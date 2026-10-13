package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
	"time"
)

func TestSwitchRecoveryRowsKeepVisibleActionAfterReload(t *testing.T) {
	for _, tc := range []struct {
		name             string
		outcome          couchcore.AgentSwitchOutcome
		action           couchcore.AgentSwitchAction
		operation, label string
	}{
		{"waiting", couchcore.AgentSwitchInProgress, couchcore.AgentSwitchWait, "", "Switch in progress"},
		{"before park", couchcore.AgentSwitchOldRunning, couchcore.AgentSwitchRetry, "switch-agent", "Old agent still running"},
		{"after park", couchcore.AgentSwitchNeedsRecovery, couchcore.AgentSwitchRecover, "recover", "Switch needs recovery"},
		{"start failed", couchcore.AgentSwitchNeedsRecovery, couchcore.AgentSwitchRecover, "recover", "Switch needs recovery"},
		{"unknown", couchcore.AgentSwitchNeedsRecovery, couchcore.AgentSwitchResumeOld, "recover", "Resume old thread"},
		{"partial delivery", couchcore.AgentSwitchHandoffAttention, couchcore.AgentSwitchCopyBrief, "copy-orientation", "handoff needs attention"},
		{"complete", couchcore.AgentSwitchReady, couchcore.AgentSwitchContinue, "switch", "New agent ready"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := menuThreads()[0]
			row.State = couchcore.ThreadUnusable
			row.Reason = couchcore.ReasonUnknown
			if tc.action == couchcore.AgentSwitchContinue {
				row.State = couchcore.ThreadLive
				row.Reason = ""
			}
			row.AgentSwitch = &couchcore.AgentSwitchView{Outcome: tc.outcome, Action: tc.action, RequestID: "saved-request", Detail: "exact recovery reason", Brief: "  exact\n\tbrief\n"}
			state := NewMenuState(nil, row.Address)
			state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{row}, Generation: 1})
			if got := enterOperationFor(row); got != tc.operation {
				t.Fatalf("primary %q want %q", got, tc.operation)
			}
			text := rootStateText(row, time.Now())
			if !strings.Contains(text, tc.label) {
				t.Fatalf("row %q lacks %q", text, tc.label)
			}
			if tc.outcome == couchcore.AgentSwitchNeedsRecovery && !strings.Contains(renderedMenuNotice(state), "exact recovery reason") {
				t.Fatalf("no restart notice: %+v", state.Notice)
			}
			if tc.action != couchcore.AgentSwitchContinue && reattachCandidate(row) {
				t.Fatal("unresolved recovery auto-reattaches")
			}
			state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
			switch tc.action {
			case couchcore.AgentSwitchRecover, couchcore.AgentSwitchResumeOld:
				if len(effects) != 1 || effects[0].Operation != "recover" {
					t.Fatalf("effects %+v", effects)
				}
				state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "recover", Attempt: state.InFlight.Attempt, Address: row.Address, Error: "resource unavailable"})
				if enterOperationFor(row) != "recover" || !strings.Contains(state.Notice.Text, "resource unavailable") {
					t.Fatal("recovery failure lost its exit")
				}
			case couchcore.AgentSwitchCopyBrief:
				if len(effects) != 1 || effects[0].CopyOrientation == nil || effects[0].CopyOrientation.Body != row.AgentSwitch.Brief || effects[0].Operation != "" {
					t.Fatalf("copy effects %+v", effects)
				}
			case couchcore.AgentSwitchRetry:
				if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameSwitchAgent {
					t.Fatal("retry did not open existing form")
				}
			}
		})
	}
}
