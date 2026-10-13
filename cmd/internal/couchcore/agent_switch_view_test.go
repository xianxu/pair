package couchcore

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func viewRecovery() *SwitchRecovery {
	now := time.Unix(1000, 0)
	return &SwitchRecovery{RequestID: "request", Phase: AgentSwitchUnconfirmed, Content: SwitchContent{Kind: SwitchContentAuthored, Body: " exact brief\n"}, AdmittedAt: now, OverallDeadline: now.Add(time.Minute), Target: &threadrecord.SwitchLaunchWitness{Helper: &threadrecord.Incarnation{State: "live"}}}
}
func TestAgentSwitchViewFailureRows(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		phase                   AgentSwitchPhase
		source, target, restore Liveness
		out                     AgentSwitchOutcome
		action                  AgentSwitchAction
	}{
		{"brief overall timeout", AgentSwitchRefused, Live, Dead, Dead, AgentSwitchOldRunning, AgentSwitchRetry},
		{"busy timeout", AgentSwitchRefused, Live, Dead, Dead, AgentSwitchOldRunning, AgentSwitchRetry},
		{"park fails before exit", AgentSwitchFailure, Live, Dead, Dead, AgentSwitchOldRunning, AgentSwitchRetry},
		{"park fails after exit", AgentSwitchFailure, Dead, Dead, Dead, AgentSwitchNeedsRecovery, AgentSwitchRecover},
		{"park completion unknown", AgentSwitchFailure, Unknown, Dead, Dead, AgentSwitchNeedsRecovery, AgentSwitchResumeOld},
		{"target start absent", AgentSwitchFailure, Dead, Dead, Dead, AgentSwitchNeedsRecovery, AgentSwitchRecover},
		{"target start uncertain", AgentSwitchFailure, Dead, Unknown, Dead, AgentSwitchNeedsRecovery, AgentSwitchResumeOld},
		{"delivery partial live target", AgentSwitchUnconfirmed, Dead, Live, Dead, AgentSwitchHandoffAttention, AgentSwitchCopyBrief},
		{"delivery partial exited target", AgentSwitchUnconfirmed, Dead, Dead, Dead, AgentSwitchNeedsRecovery, AgentSwitchRecover},
		{"restart before park", AgentSwitchWaitingBrief, Live, Dead, Dead, AgentSwitchOldRunning, AgentSwitchRetry},
		{"restart after park", AgentSwitchSwitching, Dead, Unknown, Dead, AgentSwitchNeedsRecovery, AgentSwitchResumeOld},
		{"restart registered no delivery", AgentSwitchDeliveryPending, Dead, Live, Dead, AgentSwitchHandoffAttention, AgentSwitchCopyBrief},
		{"recovery own failure", AgentSwitchFailure, Dead, Dead, Dead, AgentSwitchNeedsRecovery, AgentSwitchRecover},
		{"recovered original source", AgentSwitchFailure, Live, Dead, Dead, AgentSwitchOldRunning, AgentSwitchRetry},
		{"recovered restored source", AgentSwitchRefused, Dead, Dead, Live, AgentSwitchOldRunning, AgentSwitchRetry},
		{"confirmed ready", AgentSwitchSucceeded, Dead, Live, Dead, AgentSwitchReady, AgentSwitchContinue},
		{"conflicting survivors", AgentSwitchSucceeded, Live, Live, Dead, AgentSwitchNeedsRecovery, AgentSwitchRecover},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := viewRecovery()
			r.Phase = tc.phase
			r.Error = "concrete failure"
			got := ProjectAgentSwitch(r, AgentSwitchEvidence{Source: tc.source, Target: tc.target, Restoration: tc.restore, Now: r.AdmittedAt})
			if got.Outcome != tc.out || got.Action != tc.action || got.RequestID != r.RequestID || got.Label() == "" || got.ActionLabel() == "" {
				t.Fatalf("%+v", got)
			}
			if got.Action == AgentSwitchCopyBrief {
				if got.Brief != r.Content.Body {
					t.Fatal("rewrote brief")
				}
			} else if got.Brief != "" {
				t.Fatal("brief disclosed outside copy action")
			}
		})
	}
}
func TestAgentSwitchViewEveryPhaseAndUnknownEvidence(t *testing.T) {
	for _, phase := range []AgentSwitchPhase{AgentSwitchWaitingBrief, AgentSwitchWaitingSettled, AgentSwitchSwitching, AgentSwitchDeliveryPending, AgentSwitchSucceeded, AgentSwitchRefused, AgentSwitchFailure, AgentSwitchUnconfirmed} {
		r := viewRecovery()
		r.Phase = phase
		e := AgentSwitchEvidence{ActiveRequestID: r.RequestID, Now: r.AdmittedAt}
		got := ProjectAgentSwitch(r, e)
		active := phase == AgentSwitchWaitingBrief || phase == AgentSwitchWaitingSettled || phase == AgentSwitchSwitching || phase == AgentSwitchDeliveryPending
		if active {
			if got.Outcome != AgentSwitchInProgress || got.Action != AgentSwitchWait {
				t.Fatal(phase, got)
			}
		} else if got.Outcome != AgentSwitchNeedsRecovery || got.Action != AgentSwitchResumeOld {
			t.Fatal(phase, got)
		}
		e.Now = r.OverallDeadline
		got = ProjectAgentSwitch(r, e)
		if got.Action == AgentSwitchWait {
			t.Fatal("expired spinner", phase)
		}
		e.Now = r.AdmittedAt
		e.ActiveRequestID = "other"
		got = ProjectAgentSwitch(r, e)
		if got.Action == AgentSwitchWait {
			t.Fatal("foreign request revived spinner", phase)
		}
	}
	if got := ProjectAgentSwitch(nil, AgentSwitchEvidence{}); got != (AgentSwitchView{}) {
		t.Fatal(got)
	}
}
func TestAgentSwitchViewFreshAndRegistrationWitness(t *testing.T) {
	r := viewRecovery()
	r.Content = SwitchContent{Kind: SwitchContentFresh}
	e := AgentSwitchEvidence{Source: Dead, Target: Live, Restoration: Dead}
	got := ProjectAgentSwitch(r, e)
	if got.Outcome != AgentSwitchReady {
		t.Fatal(got)
	}
	r.Target.Helper.State = "creating"
	got = ProjectAgentSwitch(r, e)
	if got.Outcome != AgentSwitchNeedsRecovery {
		t.Fatal("partial target falsely ready", got)
	}
	r.Target.Helper.State = "live"
	r.Content = SwitchContent{}
	got = ProjectAgentSwitch(r, e)
	if got.Action == AgentSwitchCopyBrief || got.Outcome == AgentSwitchReady {
		t.Fatal("missing content falsely usable", got)
	}
}
func TestSwitchRecoveryThreadConversionClone(t *testing.T) {
	saved := viewRecovery()
	saved.Source.Profile = threadrecord.LaunchProfile{Agent: "codex", Argv: []string{"arg"}}
	saved.TargetProfile = threadrecord.LaunchProfile{Agent: "claude", Argv: []string{}}
	record := ThreadRecord{SwitchRecovery: saved}
	persisted := toPersistedThreadRecord(record)
	round := fromPersistedThreadRecord(persisted)
	cloned := cloneThreadRecord(record)
	if !reflect.DeepEqual(round.SwitchRecovery, saved) {
		t.Fatal("conversion lost saved evidence")
	}
	persisted.SwitchRecovery.Source.Profile.Argv[0] = "persisted"
	round.SwitchRecovery.Source.Profile.Argv[0] = "round"
	cloned.SwitchRecovery.Source.Profile.Argv[0] = "clone"
	if saved.Source.Profile.Argv[0] != "arg" {
		t.Fatal("conversion or clone aliased recovery")
	}
}
