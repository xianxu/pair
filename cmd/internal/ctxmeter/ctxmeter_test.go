package ctxmeter

import (
	"strings"
	"testing"
)

func TestContextTokensClaude_SumsThreeInputs_SkipsSidechainAndSynthetic(t *testing.T) {
	// real turn (300) → sidechain (small) → synthetic (0). Want the last REAL one: 300.
	jsonl := strings.Join([]string{
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-opus-4-8","usage":{"input_tokens":100,"cache_creation_input_tokens":50,"cache_read_input_tokens":150}}}`,
		`{"type":"assistant","isSidechain":true,"message":{"model":"claude-opus-4-8","usage":{"input_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`,
		`{"type":"assistant","message":{"model":"<synthetic>","usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`,
	}, "\n")
	got, ok := ContextTokens("claude", strings.NewReader(jsonl))
	if !ok || got != 300 {
		t.Fatalf("got (%d,%v) want (300,true)", got, ok)
	}
}

func TestContextTokensCodex_LastTokenUsageNotTotal(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":60287},"total_token_usage":{"input_tokens":38425074}}}}`,
		`{"type":"response_item","payload":{"type":"message"}}`,
	}, "\n")
	got, ok := ContextTokens("codex", strings.NewReader(jsonl))
	if !ok || got != 60287 {
		t.Fatalf("got (%d,%v) want (60287,true)", got, ok)
	}
}

func TestContextTokensAgy_None(t *testing.T) {
	if _, ok := ContextTokens("agy", strings.NewReader(`{"type":"PLANNER_RESPONSE"}`)); ok {
		t.Fatal("agy should yield no count")
	}
}

func TestContextTokensTolerant_GarbageLines(t *testing.T) {
	jsonl := "not json\n" + `{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\nalso not json"
	got, ok := ContextTokens("claude", strings.NewReader(jsonl))
	if !ok || got != 10 {
		t.Fatalf("got (%d,%v) want (10,true)", got, ok)
	}
}

func TestHumanize(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"}, {999, "999"},
		{1000, "1k"}, {397556, "398k"}, // round half-up
		{999999, "1000k"},                    // k-branch can emit 4 digits
		{1000000, "1.0M"}, {1490000, "1.4M"}, // M-branch floors
	}
	for _, c := range cases {
		if got := Humanize(c.n); got != c.want {
			t.Errorf("Humanize(%d)=%q want %q", c.n, got, c.want)
		}
	}
}
