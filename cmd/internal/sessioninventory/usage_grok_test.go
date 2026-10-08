package sessioninventory_test

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Grok stamps each streamed update with the context occupancy so far
// (params._meta.totalTokens). Measured on a live 1.0.46 session, the last
// value equals signals.json's contextTokensUsed — what grok's own header shows
// against its window. turn_completed's usage is per-turn billing summed over
// every model call and must not feed the context meter.
func TestTokenUsageGrokReadsContextOccupancy(t *testing.T) {
	data := []byte(`{"timestamp":1791400000,"method":"session/update","params":{"sessionId":"s","update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"hi"}},"_meta":{"eventId":"s-2"}}}
{"timestamp":1791400001,"method":"session/update","params":{"sessionId":"s","update":{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"x"}},"_meta":{"totalTokens":55202}}}
{"timestamp":1791400002,"method":"session/update","params":{"sessionId":"s","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"ok"}},"_meta":{"totalTokens":56912}}}
{"timestamp":1791400003,"method":"_x.ai/session/update","params":{"sessionId":"s","update":{"sessionUpdate":"turn_completed","usage":{"inputTokens":255658,"totalTokens":262309}},"_meta":{"eventId":"s-9"}}}
`)
	usage, ok := sessioninventory.TokenUsageFromJSONL(sessioninventory.AgentGrok, data)
	if !ok || usage.InputTokens != 56912 {
		t.Fatalf("usage = %#v ok=%v, want the last _meta.totalTokens 56912", usage, ok)
	}
	for name, line := range map[string]string{
		"negative":       `{"method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk"},"_meta":{"totalTokens":-1}}}`,
		"not a number":   `{"method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk"},"_meta":{"totalTokens":"56912"}}}`,
		"no meta":        `{"method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk"}}}`,
		"other envelope": `{"type":"assistant","_meta":{"totalTokens":5}}`,
	} {
		if _, ok := sessioninventory.ParseTokenUsage(sessioninventory.AgentGrok, []byte(line)); ok {
			t.Errorf("%s: accepted as grok usage", name)
		}
	}
}
