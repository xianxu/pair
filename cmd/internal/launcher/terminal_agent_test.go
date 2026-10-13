package launcher

import (
	"github.com/xianxu/pair/cmd/internal/procutil"
	"testing"
)

func TestTerminalAgentRequiresExactServerAndLiveDescendant(t *testing.T) {
	server := SessionServerIdentity{PID: 10, Identity: "server-generation"}
	table := []procutil.Process{{PID: 10, PPID: 1, Identity: "server-generation"}, {PID: 20, PPID: 10, Identity: "wrapper-generation"}, {PID: 30, PPID: 20, Identity: "agent-generation"}}
	if !AgentBelongsToServer(server, 30, "agent-generation", table) {
		t.Fatal("rejected live agent under exact server")
	}
	for _, pid := range []int{0, 10, 99} {
		if AgentBelongsToServer(server, pid, "agent-generation", table) {
			t.Fatalf("accepted non-agent %d", pid)
		}
	}
	table[2].Identity = "reused-agent"
	if AgentBelongsToServer(server, 30, "agent-generation", table) {
		t.Fatal("stale readiness accepted reused agent pid with unchanged ancestry")
	}
	table[2].Identity = "agent-generation"
	if AgentBelongsToServer(server, 30, "", table) {
		t.Fatal("legacy readiness attested identity")
	}
	table[0].Identity = "replacement"
	if AgentBelongsToServer(server, 30, "agent-generation", table) {
		t.Fatal("accepted reused server pid")
	}
	table[0].Identity = server.Identity
	table[1].PPID = 999
	if AgentBelongsToServer(server, 30, "agent-generation", table) {
		t.Fatal("accepted agent under another server")
	}
	table[1].PPID = 30
	if AgentBelongsToServer(server, 30, "agent-generation", table) {
		t.Fatal("accepted ancestry cycle")
	}
}
