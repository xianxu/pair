package couchcore

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// #399 M1/M2 review (rule, family operator-advice-contradicts-evidence): the
// advice is the reap mechanism, never a hand-written kill recipe -- two such
// recipes were each wrong about the 2026-10-06 tree. The gesture it names is
// the switcher's recover, which runs the tested reap (descendants first, the
// helpers outside the tree, identity-gated) and then resume.
func TestOrphanRefusalNamesTheReapMechanism(t *testing.T) {
	text := orphanStartRefusal("/repo", ThreadAddress{Tag: "couch-1"}, &launcher.SessionServerIdentity{PID: 9090, Session: "📁repo-1"})
	for _, want := range []string{"server PID 9090 lost its socket", "Tab → recover", "couch-1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("refusal lacks %q:\n%s", want, text)
		}
	}
	for _, wrong := range []string{"kill ", "pkill", "goes with it"} {
		if strings.Contains(text, wrong) {
			t.Fatalf("refusal still carries a hand-written recipe (%q):\n%s", wrong, text)
		}
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
	if text := orphanStartRefusal("/repo", row.Address, nil); !strings.Contains(text, "lost its socket") || !strings.Contains(text, "Tab → recover") {
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
