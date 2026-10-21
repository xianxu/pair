package couchschedule

import (
	"fmt"
	"testing"
)

func TestAdvanceAutomationDisabledConsumesDueAndLeavesManualIndependent(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, manual := range []bool{false, true} {
			t.Run(fmt.Sprintf("script=%t/manual=%t", direct, manual), func(t *testing.T) {
				state, due := taskFixture(t, direct)
				kind := ClockObserved
				if manual {
					kind = ManualRequested
				}
				next, effects := Advance(state, Event{Kind: kind, At: due, RunID: "automation-off", HubOnline: true, AutomationDisabled: true})
				if len(effects) != 0 {
					t.Fatal("emitted premature effect")
				}
				if manual {
					if next.Active == nil || next.Active.Source != "manual" || !next.NextDue.Equal(state.NextDue) {
						t.Fatalf("automation switch blocked manual: %+v", next)
					}
				} else {
					if next.Active != nil || next.Last == nil || next.Last.Outcome != "skipped" || next.Last.Notice != "automation off, skipped" || !next.NextDue.After(due) || !next.Last.DueAt.Equal(state.NextDue) {
						t.Fatalf("automation-off due not consumed as skip: %+v", next)
					}
					replay, _ := Advance(next, Event{Kind: ClockObserved, At: due, RunID: "replay", AutomationDisabled: true})
					if replay.Last.ID != next.Last.ID {
						t.Fatal("replayed consumed slot")
					}
				}
			})
		}
	}
}
