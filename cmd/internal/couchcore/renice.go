package couchcore

import "github.com/xianxu/pair/cmd/internal/procutil"

// The slot renicer's decision (#502): keep the interactive layer (Couch, a
// slot's pair wrapper, zellij, the agent) at normal priority and lower
// only what an agent spawns. Pure: one process-table snapshot in, which
// processes to lower out. couchcmd's renicer is the shell that reads the
// snapshot and applies it.

// ReniceRoot is one slot as the renicer finds it: its wrapper (from the
// message broker's binding, pid and start identity) and the agent pid the
// wrapper reported (its agent-pid file; 0 when unread).
type ReniceRoot struct {
	Slot         string
	WrapperPID   int
	WrapperStart string
	AgentPID     int
}

// ReniceSlot is the decision for one slot. Confirmed says the snapshot
// proves the agent: its parent is the slot's wrapper and the wrapper's
// identity is the binding's. Only a confirmed slot lowers anything. Lower
// is the agent's strict descendants below the level; AtLevel counts those
// already at it or lower.
type ReniceSlot struct {
	Slot      string
	Confirmed bool
	Agent     procutil.Process
	Lower     []procutil.Process
	AtLevel   int
}

// PlanRenice decides, from one snapshot, what to lower to level in each
// slot. The agent is identified only by the pid its wrapper reported and
// confirmed in the same snapshot, never by a name. self is Couch's pid:
// Couch and its ancestors, every slot's agent and wrapper, are never
// lowered, even if the tree would reach them. level 0 lowers nothing (the
// facts are still reported).
func PlanRenice(table []procutil.Process, roots []ReniceRoot, level, self int) []ReniceSlot {
	byPID := make(map[int]procutil.Process, len(table))
	children := make(map[int][]int, len(table))
	for _, p := range table {
		byPID[p.PID] = p
		if p.PPID != p.PID {
			children[p.PPID] = append(children[p.PPID], p.PID)
		}
	}
	never := map[int]bool{}
	for pid, hops := self, 0; pid > 0 && hops < len(table)+1; hops++ {
		never[pid] = true
		p, ok := byPID[pid]
		if !ok || p.PPID == pid {
			break
		}
		pid = p.PPID
	}
	for _, r := range roots {
		never[r.WrapperPID] = true
		never[r.AgentPID] = true
	}
	out := make([]ReniceSlot, 0, len(roots))
	for _, r := range roots {
		s := ReniceSlot{Slot: r.Slot}
		agent, aok := byPID[r.AgentPID]
		wrapper, wok := byPID[r.WrapperPID]
		if r.AgentPID <= 0 || !aok || !wok || agent.PPID != r.WrapperPID || r.WrapperStart == "" || wrapper.Identity != r.WrapperStart {
			out = append(out, s)
			continue
		}
		s.Confirmed, s.Agent = true, agent
		seen := map[int]bool{agent.PID: true}
		queue := append([]int(nil), children[agent.PID]...)
		for len(queue) > 0 {
			pid := queue[0]
			queue = queue[1:]
			if seen[pid] {
				continue
			}
			seen[pid] = true
			queue = append(queue, children[pid]...)
			p := byPID[pid]
			switch {
			case never[pid]:
			case level <= 0 || p.Nice >= level:
				s.AtLevel++
			default:
				s.Lower = append(s.Lower, p)
			}
		}
		out = append(out, s)
	}
	return out
}
