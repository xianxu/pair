package launcher

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/readiness"
)

func TestMatchReadyRecordRequiresExactIdentityAndLivePID(t *testing.T) {
	expect := ReadyExpectation{Tag: "work", Agent: "codex", Session: "pair-work", Nonce: "nonce-1"}
	record := readiness.ReadyRecord{Tag: "work", Agent: "codex", Session: "pair-work", Nonce: "nonce-1", PID: 123}
	if err := MatchReadyRecord(expect, record, func(pid int) bool { return pid == 123 }); err != nil {
		t.Fatalf("MatchReadyRecord exact live record returned error: %v", err)
	}
	for _, tc := range []struct {
		name string
		rec  readiness.ReadyRecord
		live func(int) bool
		want string
	}{
		{"stale nonce", readiness.ReadyRecord{Tag: "work", Agent: "codex", Session: "pair-work", Nonce: "old", PID: 123}, func(int) bool { return true }, "nonce"},
		{"wrong session", readiness.ReadyRecord{Tag: "work", Agent: "codex", Session: "other", Nonce: "nonce-1", PID: 123}, func(int) bool { return true }, "session"},
		{"dead pid", record, func(int) bool { return false }, "pid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := MatchReadyRecord(expect, tc.rec, tc.live)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("MatchReadyRecord error = %v, want mention %q", err, tc.want)
			}
		})
	}
}

func TestScopedPathsAgentReadyUsesTagAndAgent(t *testing.T) {
	scope, err := ResolveRepoScope("/Users/x/workspace/pair")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}
	paths := NewScopedPaths("/data", scope, "work")
	if got, want := paths.AgentReady("codex"), paths.ScopeDir()+"/agent-ready-work-codex.json"; got != want {
		t.Fatalf("AgentReady = %q, want %q", got, want)
	}
}
