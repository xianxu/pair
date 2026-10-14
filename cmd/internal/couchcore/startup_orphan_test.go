package couchcore

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestVocabularyConsumersKnowTheOrphan(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	if !(agentRank[AgentBusy] > agentRank[AgentOrphaned] && agentRank[AgentOrphaned] > agentRank[AgentUnusableUnknown]) {
		t.Fatalf("rank = %v", agentRank)
	}
	for _, a := range AllEvidenceAgents() {
		if _, ok := agentRank[a]; !ok {
			t.Fatalf("agent %q has no rank", a)
		}
	}
}
