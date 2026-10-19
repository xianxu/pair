package sessioninventory

import "testing"

func TestRootTranscriptIDUsesTheScannerParsers(t *testing.T) {
	const id = "0b5c2a1e-9f1d-4c4e-8d5b-3e2f1a0b9c8d"
	cases := []struct {
		agent Agent
		rel   string
		ok    bool
	}{
		{AgentClaude, "-Users-x-pair/" + id + ".jsonl", true},
		{AgentQoder, "-Users-x-pair/" + id + ".jsonl", true},
		{AgentClaude, "-Users-x-pair/" + id + "/subagents/agent-abc.jsonl", false},
		{AgentCodex, "2026/10/19/rollout-2026-10-19T09-00-00-" + id + ".jsonl", true},
		{AgentCodex, "2026/10/19/" + id + ".jsonl", false},
		{AgentGrok, "%2FUsers%2Fx%2Fpair/" + id + "/updates.jsonl", true},
		{AgentGrok, "relative/" + id + "/updates.jsonl", false},
		{Agent("unknown"), id + ".jsonl", false},
	}
	for _, c := range cases {
		got, ok := RootTranscriptID(c.agent, c.rel)
		if ok != c.ok || ok && got != id {
			t.Errorf("RootTranscriptID(%s, %q) = %q, %v; want ok=%v", c.agent, c.rel, got, ok, c.ok)
		}
	}
}
