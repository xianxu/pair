package couchcore

import (
	"os"
	"strings"
	"testing"
)

// TestShowReportsASlotWithoutAThread: a deleted slot directory has no thread
// record left (it lived inside the directory), yet --show of its path must
// still report the slot's resources and the plan.
func TestShowReportsASlotWithoutAThread(t *testing.T) {
	s := newObservedSlot(t)
	if err := os.RemoveAll(s.layout.Env()); err != nil {
		t.Fatal(err)
	}
	env := newTestEnv(t, "/repo")
	env.Couch.SlotIO = s.f
	result, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": s.layout.Host(), "repo-scope": "scope"})
	if err != nil {
		t.Fatal(err)
	}
	show, ok := result.(ShowResult)
	if !ok || show.Slot == nil {
		t.Fatalf("show = %#v, want a slot report", result)
	}
	if len(show.Threads) != 0 {
		t.Fatalf("threads = %+v, want none", show.Threads)
	}
	env0, _ := show.Slot.Observation.Get(ResourceEnv)
	reg, _ := show.Slot.Observation.Get(ResourceRegistration)
	if env0.State != StateAbsent || reg.Sub != SubStale {
		t.Fatalf("observation %+v", show.Slot.Observation)
	}
	if show.Slot.Address != "repo-name:1" || !strings.HasPrefix(SlotPlanSummary(show.Slot.Plan, show.Slot.PlanError), string(StepMkdirEnv)) {
		t.Fatalf("report %q plan %q", show.Slot.Address, SlotPlanSummary(show.Slot.Plan, show.Slot.PlanError))
	}
}

// TestShowOfAThreadOutsideASlotHasNoSlotReport keeps --show of an ordinary
// thread as it was.
func TestShowOfAThreadOutsideASlotHasNoSlotReport(t *testing.T) {
	env := newTestEnv(t, "/repo")
	created := createOperationThread(t, env.Couch)
	result, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": string(created.Address.Tag), "repo-scope": created.Address.RepoScope})
	if err != nil {
		t.Fatal(err)
	}
	show := result.(ShowResult)
	if show.Slot != nil || len(show.Threads) != 1 {
		t.Fatalf("show = %+v", show)
	}
}

func TestSlotAgentEvidenceTakesTheMostActiveRow(t *testing.T) {
	rows := []ThreadSummary{{State: ThreadParked}, {State: ThreadDetached}}
	if got := slotAgentEvidence(rows); got != AgentDetached {
		t.Fatalf("got %s, want detached: a parked sibling must not hide a running agent", got)
	}
	if got := slotAgentEvidence(nil); got != AgentNone {
		t.Fatalf("no rows = %s", got)
	}
}

func TestSlotPlanSummary(t *testing.T) {
	if got := SlotPlanSummary(SlotPlan{}, ""); got != "nothing to do" {
		t.Fatal(got)
	}
	got := SlotPlanSummary(SlotPlan{Stops: []PlanStop{{Resource: ResourceRegistration, Class: StopUnknown, Reason: "git worktree list: boom"}}}, "")
	if got != "stops at registration (unknown): git worktree list: boom" {
		t.Fatal(got)
	}
}
