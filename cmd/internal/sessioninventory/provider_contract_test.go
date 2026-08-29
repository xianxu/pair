package sessioninventory

import "testing"

func TestProviderContractFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		agent  Agent
		root   string
		schema string
		want   ProviderContract
	}{
		{name: "claude", agent: AgentClaude, root: "claude-projects", schema: "claude-v1", want: ProviderClaudeJSONLV1},
		{name: "codex", agent: AgentCodex, root: "codex-sessions", schema: "codex-v1", want: ProviderCodexJSONLV1},
		{name: "muse", agent: AgentMuse, root: "muse-sessions", schema: "muse-v1", want: ProviderMuseJSONLV1},
		{name: "agy transcript", agent: AgentAgy, root: "agy-brain", schema: "agy-transcript-v1", want: ProviderAgyTranscriptJSONLV1},
		{name: "agy sqlite", agent: AgentAgy, root: "agy-conversations", schema: "agy-v1"},
		{name: "unknown schema", agent: AgentClaude, root: "claude-projects", schema: "claude-v2"},
		{name: "wrong root", agent: AgentClaude, root: "other", schema: "claude-v1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ProviderContractFor(test.agent, test.root, test.schema)
			if got != test.want || ok != (test.want != "") {
				t.Fatalf("ProviderContractFor() = %q, %v; want %q, %v", got, ok, test.want, test.want != "")
			}
		})
	}
}
