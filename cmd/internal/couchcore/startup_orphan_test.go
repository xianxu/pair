package couchcore

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// #399 M1 review. The operator advice must match the evidence: on 2026-10-06
// killing only the server left `pair wrap` (which ignored SIGTERM) and PPID-1
// `pair title` helpers behind. The refusal names the whole tree.
func TestOrphanRefusalNamesTheWholeProcessTree(t *testing.T) {
	text := orphanStartRefusal("/repo", ThreadAddress{Tag: "couch-1"}, &launcher.SessionServerIdentity{PID: 9090, Session: "📁repo-1"})
	for _, want := range []string{"pkill -TERM -P 9090", "kill 9090", "pkill -KILL", "pair wrap"} {
		if !strings.Contains(text, want) {
			t.Fatalf("refusal %q lacks %q", text, want)
		}
	}
	if strings.Contains(text, "goes with it") {
		t.Fatalf("refusal still claims the agent dies with the server: %q", text)
	}
}

// The reason alone decides: an orphan row whose server details are missing is
// still a running agent, never debris to the one-primary rule.
func TestAnOrphanRowWithoutItsServerStillBlocksStartup(t *testing.T) {
	row := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "couch-1"}, WorkingPath: "/repo",
		State: ThreadUnusable, Reason: ReasonOrphanedServer}
	if _, held := ScopeHoldsOrphanedThread([]ActionableThreadSummary{row}, "s"); !held {
		t.Fatal("an orphan without server details read as debris")
	}
	if text := orphanStartRefusal("/repo", row.Address, nil); !strings.Contains(text, "lost its socket") {
		t.Fatalf("refusal without a server = %q", text)
	}
}

func TestVocabularyConsumersKnowTheOrphan(t *testing.T) {
	if sessionOwnerWord(launcher.SessionOwnerOrphaned) != "orphaned" {
		t.Fatalf("word = %q", sessionOwnerWord(launcher.SessionOwnerOrphaned))
	}
	// An orphan's agent is known to be running: reconcile must never treat its
	// checkout as removable.
	if running, known := AgentRunning(AgentOrphaned); !running || !known {
		t.Fatalf("AgentRunning(orphaned) = %v, %v", running, known)
	}
}

// One ranking: the slot report and the recovery report's many-threads rule
// order the attention states the same way (busy > orphaned > unknown).
func TestOneAgentRankingForBothReports(t *testing.T) {
	if !(agentRank[AgentBusy] > agentRank[AgentOrphaned] && agentRank[AgentOrphaned] > agentRank[AgentUnusableUnknown]) {
		t.Fatalf("rank = %v", agentRank)
	}
	for _, a := range AllEvidenceAgents() {
		if _, ok := agentRank[a]; !ok {
			t.Fatalf("agent %q has no rank", a)
		}
	}
}
