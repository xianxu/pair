package couchschedule

import (
	"testing"
	"time"
)

func TestUnstartedTimeoutSummaryCanBeRecovered(t *testing.T) {
	for _, kind := range []string{"manual", "script", "notify"} {
		for _, eventKind := range []EventKind{ReservationDurable, ExecutionRefused} {
			t.Run(kind+"/"+string(eventKind), func(t *testing.T) {
				source := "scheduled"
				if kind == "manual" {
					source = "manual"
				}
				state := phaseStateFixture(t, source, "reserved")
				state.Spec.ScriptRun = kind != "notify"
				at := state.Active.Deadline
				event := Event{Kind: eventKind, At: at, Attempt: state.Active.ID, Revision: state.Active.Revision, Outcome: "timeout", Notice: "execution admission deadline elapsed"}
				candidate, effects := Advance(state, event)
				if len(effects) != 0 || candidate.Last == nil || candidate.Last.Outcome != "timeout" || candidate.Last.Exit != nil {
					t.Fatalf("timeout producer changed: %+v %v", candidate, effects)
				}
				decision := PlanSettlement(state, candidate.Last, Event{At: at.Add(time.Second)})
				if decision.Kind == SettlementQuarantined || decision.State.Last == nil || !settlementKnownSummaryEqual(*decision.State.Last, *candidate.Last) || len(decision.Effects) != 0 {
					t.Fatalf("producer's own timeout facts rejected by canonical adoption: %+v", decision)
				}
			})
		}
	}
}

func TestExplanatoryTimeoutRequiresAbsentExit(t *testing.T) {
	state := phaseStateFixture(t, "manual", "reserved")
	candidate, _ := Advance(state, Event{Kind: ReservationDurable, At: state.Active.Deadline, Attempt: state.Active.ID, Revision: state.Active.Revision})
	code := 1
	candidate.Last.Exit = &code
	if validRecoveredSummary(*candidate.Last) {
		t.Fatal("explanatory pre-execution timeout claimed native exit")
	}
}
