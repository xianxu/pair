package couchcore

import (
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// A machine: launchd(1) › couch(100) › helper(110) › zellij(120) ›
// wrapper(200) › agent(300) › bash(310) › make(320) › go(330), and a sibling
// of the agent, the draft's nvim (210). A decoy named like the agent, under
// launchd, is never a slot's. pid 400 is already at 10.
func reniceMachine() []procutil.Process {
	p := func(pid, ppid, nice int, id string) procutil.Process {
		return procutil.Process{PID: pid, PPID: ppid, Nice: nice, Identity: id}
	}
	return []procutil.Process{
		p(1, 0, 0, "launchd"), p(100, 1, 0, "couch"), p(110, 100, 0, "helper"), p(120, 110, 0, "zellij"),
		p(200, 120, 0, "wrap@1"), p(300, 200, 0, "agent@1"), p(310, 300, 0, "bash"), p(320, 310, 0, "make"),
		p(330, 320, 0, "go"), p(400, 300, 10, "mcp"), p(210, 200, 0, "nvim"), p(900, 1, 0, "decoy-agent"),
	}
}

func pids(ps []procutil.Process) []int {
	var out []int
	for _, p := range ps {
		out = append(out, p.PID)
	}
	slices.Sort(out)
	return out
}

func TestPlanReniceLowersOnlyTheAgentsDescendants(t *testing.T) {
	got := PlanRenice(reniceMachine(), []ReniceRoot{{Slot: "pair:3", WrapperPID: 200, WrapperStart: "wrap@1", AgentPID: 300}}, 10, 100)
	if len(got) != 1 || !got[0].Confirmed || got[0].Agent.PID != 300 {
		t.Fatalf("%+v", got)
	}
	if l := pids(got[0].Lower); !slices.Equal(l, []int{310, 320, 330}) {
		t.Errorf("lower %v, want the agent's strict descendants below 10: 310 320 330", l)
	}
	if got[0].AtLevel != 1 {
		t.Errorf("already at the level: %d, want 1 (400)", got[0].AtLevel)
	}
}

// The agent is the pid the wrapper reported, confirmed in the same
// snapshot; nothing else selects a tree.
func TestPlanReniceConfirmsTheAgent(t *testing.T) {
	m := reniceMachine()
	for _, c := range []struct {
		name string
		root ReniceRoot
	}{
		{"no agent pid (file unread)", ReniceRoot{WrapperPID: 200, WrapperStart: "wrap@1"}},
		{"a stale pid that is now the decoy", ReniceRoot{WrapperPID: 200, WrapperStart: "wrap@1", AgentPID: 900}},
		{"a pid that is gone", ReniceRoot{WrapperPID: 200, WrapperStart: "wrap@1", AgentPID: 777}},
		{"the wrapper's pid reused (identity differs)", ReniceRoot{WrapperPID: 200, WrapperStart: "wrap@0", AgentPID: 300}},
		{"no wrapper identity", ReniceRoot{WrapperPID: 200, AgentPID: 300}},
		{"the agent pid is the wrapper's grandchild", ReniceRoot{WrapperPID: 200, WrapperStart: "wrap@1", AgentPID: 310}},
		{"the agent pid is the wrapper itself", ReniceRoot{WrapperPID: 200, WrapperStart: "wrap@1", AgentPID: 200}},
	} {
		got := PlanRenice(m, []ReniceRoot{c.root}, 10, 100)
		if got[0].Confirmed || len(got[0].Lower) != 0 {
			t.Errorf("%s: %+v", c.name, got[0])
		}
	}
}

// Never the agent, a wrapper, zellij or Couch, even where the tree would
// reach them; level 0 lowers nothing; a cycle in a snapshot doesn't loop.
func TestPlanReniceNever(t *testing.T) {
	m := reniceMachine()
	// A second slot whose wrapper sits, oddly, under the first slot's agent.
	m = append(m, procutil.Process{PID: 500, PPID: 330, Identity: "wrap@2"}, procutil.Process{PID: 510, PPID: 500, Identity: "agent@2"}, procutil.Process{PID: 520, PPID: 510, Identity: "job"})
	roots := []ReniceRoot{{Slot: "a", WrapperPID: 200, WrapperStart: "wrap@1", AgentPID: 300}, {Slot: "b", WrapperPID: 500, WrapperStart: "wrap@2", AgentPID: 510}}
	got := PlanRenice(m, roots, 10, 100)
	for _, s := range got {
		for _, p := range s.Lower {
			if p.PID == 500 || p.PID == 510 || p.PID == 200 || p.PID == 300 || p.PID == 100 || p.PID == 120 {
				t.Errorf("slot %s would lower %d", s.Slot, p.PID)
			}
		}
	}
	if l := pids(got[1].Lower); !slices.Equal(l, []int{520}) {
		t.Errorf("slot b lowers %v, want 520", l)
	}
	// Couch under an agent (a test run from a slot): Couch and its
	// ancestors stay.
	got = PlanRenice(reniceMachine(), roots[:1], 10, 330)
	if l := pids(got[0].Lower); slices.Contains(l, 330) || slices.Contains(l, 320) || slices.Contains(l, 310) {
		t.Errorf("Couch or its ancestors lowered: %v", l)
	}
	if got := PlanRenice(reniceMachine(), roots[:1], 0, 100); len(got[0].Lower) != 0 || got[0].AtLevel != 0 || !got[0].Confirmed {
		t.Errorf("level 0: %+v", got[0])
	}
	cycle := append(reniceMachine(), procutil.Process{PID: 600, PPID: 601}, procutil.Process{PID: 601, PPID: 600})
	_ = PlanRenice(cycle, roots[:1], 10, 600)
}

func TestPlanReniceTheDecoyIsNeverTouched(t *testing.T) {
	got := PlanRenice(reniceMachine(), []ReniceRoot{{Slot: "pair:3", WrapperPID: 200, WrapperStart: "wrap@1", AgentPID: 300}}, 10, 100)
	if slices.Contains(pids(got[0].Lower), 900) {
		t.Error("a process named like the agent, outside the tree, was selected")
	}
}
