package couchcore

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// #399 M1 review (rule, family operator-advice-contradicts-evidence): advice
// must be steps that would have worked on the documented incident, and the test
// checks their ORDER. On 2026-10-06 killing the server first let `pair wrap`
// (which ignored SIGTERM) and `pair title` helpers reparent to PID 1, where no
// child-of-server command finds them. So: list the tree while the server still
// parents it, kill the descendants, and only then the server.
func TestOrphanRefusalStepsWouldHaveWorkedOnTheIncident(t *testing.T) {
	text := orphanStartRefusal("/repo", ThreadAddress{Tag: "couch-1"}, &launcher.SessionServerIdentity{PID: 9090, Session: "📁repo-1"})
	list := strings.Index(text, "ps -axo pid,ppid,command")
	descendants := strings.Index(text, "kill -KILL <pid>")
	server := strings.Index(text, "kill -KILL 9090")
	if list < 0 || descendants < 0 || server < 0 || !(list < descendants && descendants < server) {
		t.Fatalf("steps out of order (list %d, descendants %d, server %d):\n%s", list, descendants, server, text)
	}
	// No step may signal the server before its descendants are dealt with.
	if first := strings.Index(text, "9090"); strings.Contains(text[:descendants], "kill") && first < descendants && strings.Contains(text[first:descendants], "kill 9090") {
		t.Fatalf("a step kills the server before its descendants:\n%s", text)
	}
	for _, wrong := range []string{"goes with it", "pkill -TERM -P"} {
		if strings.Contains(text, wrong) {
			t.Fatalf("refusal still advises %q, which failed on 2026-10-06:\n%s", wrong, text)
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
