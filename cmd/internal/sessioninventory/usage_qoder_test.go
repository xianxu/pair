package sessioninventory

import "testing"

// qoderUsage is a usage block captured byte for byte from a Qoder transcript
// (pair:8, 2026-10-17): Qoder writes Claude's shape with every token field 0
// and reports occupancy only as context_usage_ratio (pair#654).
const qoderUsage = `"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0,"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":"standard","cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":0},"inference_geo":"","iterations":[],"speed":"standard","credits":2.8637050828571424,"original_credits":2.8637050828571424,"billable":true,"request_id":"8dd45c60-c3b8-4c98-920d-c7a268dd9820","context_usage_ratio":0.61191}`

// qoderAssistant is the captured record around a usage block, with the
// message content trimmed.
func qoderAssistant(sidechain, usage string) string {
	return `{"type":"assistant","isSidechain":` + sidechain + `,"uuid":"aeb7b232-fc82-4f36-8e6e-6e960110e2fc","timestamp":"2026-10-17T19:26:24.167Z","message":{"id":"202610180326196b4d124bee024717","type":"message","role":"assistant","model":"auto","stop_reason":"end_turn","stop_sequence":null,"content":[{"type":"text","text":"trimmed"}],` + usage + `}}`
}

func TestParseTokenUsageQoderReadsTheContextRatio(t *testing.T) {
	t.Parallel()

	got, ok := ParseTokenUsage(AgentQoder, []byte(qoderAssistant("false", qoderUsage)))
	if !ok || got.ContextRatio != 0.61191 || got.InputTokens != 0 || got.Figure() != "61%" {
		t.Fatalf("Qoder usage = %#v, %v (%q)", got, ok, got.Figure())
	}

	for name, line := range map[string]string{
		"zero tokens, no ratio": qoderAssistant("false", `"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}`),
		"tokens but no ratio":   qoderAssistant("false", `"usage":{"input_tokens":5000}`),
		"zero ratio":            qoderAssistant("false", `"usage":{"context_usage_ratio":0}`),
		"ratio above one":       qoderAssistant("false", `"usage":{"context_usage_ratio":1.2}`),
		"sidechain":             qoderAssistant("true", qoderUsage),
		"no usage":              qoderAssistant("false", `"stop_details":null`),
		"user record":           `{"type":"user","message":{"role":"user","content":"hi"},` + qoderUsage + `}`,
		"malformed":             `not json`,
	} {
		if got, ok := ParseTokenUsage(AgentQoder, []byte(line)); ok || got != (TokenUsage{}) {
			t.Errorf("%s: accepted %#v", name, got)
		}
	}
}

func TestTokenUsageFromJSONLQoderKeepsTheLastRatio(t *testing.T) {
	t.Parallel()

	later := `"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0,"context_usage_ratio":0.631995}`
	data := []byte(qoderAssistant("false", qoderUsage) + "\n" +
		`{"type":"active-leaf","sessionId":"7febf963-ecac-409f-82f9-c1438842f2a5"}` + "\n" +
		qoderAssistant("false", later) + "\n" +
		qoderAssistant("true", `"usage":{"context_usage_ratio":0.9}`) + "\n")
	if got, ok := TokenUsageFromJSONL(AgentQoder, data); !ok || got.Figure() != "63%" {
		t.Fatalf("Qoder last usage = %#v, %v", got, ok)
	}
}

func TestTokenUsageFigure(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		usage TokenUsage
		want  string
	}{
		{TokenUsage{InputTokens: 175}, "175"},
		{TokenUsage{InputTokens: 81_500}, "82k"},
		{TokenUsage{ContextRatio: 0.004}, "0%"},
		{TokenUsage{ContextRatio: 0.395}, "40%"},
		{TokenUsage{ContextRatio: 1}, "100%"},
	} {
		if got := c.usage.Figure(); got != c.want {
			t.Errorf("%#v.Figure() = %q, want %q", c.usage, got, c.want)
		}
	}
}
