package launcher

import "github.com/xianxu/pair/cmd/internal/procutil"

// AgentBelongsToServer consumes one process snapshot. A live wrapper belonging
// to an inaccessible old server cannot witness the replacement terminal.
func AgentBelongsToServer(server SessionServerIdentity, agentPID int, table []procutil.Process) bool {
	if server.PID <= 0 || server.Identity == "" || agentPID <= 0 || agentPID == server.PID {
		return false
	}
	rows := make(map[int]procutil.Process, len(table))
	for _, p := range table {
		if _, duplicate := rows[p.PID]; duplicate {
			return false
		}
		rows[p.PID] = p
	}
	if rows[server.PID].Identity != server.Identity {
		return false
	}
	seen := map[int]bool{}
	for pid := agentPID; pid > 0; {
		if seen[pid] {
			return false
		}
		seen[pid] = true
		p, ok := rows[pid]
		if !ok || p.Identity == "" {
			return false
		}
		if pid == server.PID {
			return true
		}
		pid = p.PPID
	}
	return false
}
