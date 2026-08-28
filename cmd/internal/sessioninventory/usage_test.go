package sessioninventory

import "testing"

func TestTokenUsageFromJSONLUsesLastAcceptedRootUsage(t *testing.T) {
	t.Parallel()

	claude := []byte(
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-opus","usage":{"input_tokens":100,"cache_creation_input_tokens":50,"cache_read_input_tokens":25}}}` + "\n" +
			`{"type":"assistant","isSidechain":false,"message":{"model":"claude-opus"}}` + "\n" +
			`{"type":"assistant","isSidechain":true,"message":{"model":"claude-opus","usage":{"input_tokens":999}}}` + "\n" +
			`{"type":"assistant","message":{"model":"<synthetic>","usage":{"input_tokens":999}}}` + "\n",
	)
	if got, ok := TokenUsageFromJSONL(AgentClaude, claude); !ok || got.InputTokens != 175 {
		t.Fatalf("Claude usage = %#v, %v", got, ok)
	}

	codex := []byte(
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":40},"total_token_usage":{"input_tokens":999}}}}` + "\n" +
			`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":60}}}}` + "\n" +
			`{"type":"event_msg","payload":{"type":"token_count","info":{}}}` + "\n" +
			`{"type":"event_msg","payload":{"type":"token_count","info":null}}` + "\n",
	)
	if got, ok := TokenUsageFromJSONL(AgentCodex, codex); !ok || got.InputTokens != 60 {
		t.Fatalf("Codex usage = %#v, %v", got, ok)
	}
}

func TestTokenUsageFromJSONLRejectsUnsupportedAndMalformedRecords(t *testing.T) {
	t.Parallel()

	for _, agent := range []Agent{AgentAgy, AgentMuse} {
		if got, ok := TokenUsageFromJSONL(agent, []byte(`{"usage":100}`+"\n")); ok || got != (TokenUsage{}) {
			t.Fatalf("%s usage = %#v, %v", agent, got, ok)
		}
	}
	if got, ok := TokenUsageFromJSONL(AgentClaude, []byte("not json\n")); ok || got != (TokenUsage{}) {
		t.Fatalf("malformed usage = %#v, %v", got, ok)
	}
}
