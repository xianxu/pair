package sessioninventory

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/ctxmeter"
)

// pair:155-concept pure new M1
// An agent reports occupancy either as a token count or, like Qoder, only as
// a fraction of a window it does not always name (ContextRatio, pair#654).
type TokenUsage struct {
	InputTokens  int     `json:"input_tokens"`
	ContextRatio float64 `json:"context_ratio,omitempty"`
}

// Figure is the context figure a pane title or pair context shows: a
// percentage for a ratio, else the humanized token count.
func (u TokenUsage) Figure() string {
	if u.ContextRatio > 0 {
		return strconv.Itoa(int(math.Round(u.ContextRatio*100))) + "%"
	}
	return ctxmeter.Humanize(u.InputTokens)
}

// TokenUsageForRoot reads the established scanner-authorized root transcript
// and returns its last accepted root usage record.
func TokenUsageForRoot(runtime Runtime, root Node) (TokenUsage, bool, error) {
	artifact, err := RootTranscript(root)
	if err != nil {
		return TokenUsage{}, false, err
	}
	var last TokenUsage
	found := false
	err = visitJSONLines(runtime, artifact, unlimitedRecordSize, func(line []byte) bool {
		if usage, ok := ParseTokenUsage(root.Agent, line); ok {
			last, found = usage, true
		}
		return false
	})
	return last, found, err
}

func TokenUsageFromJSONL(agent Agent, data []byte) (TokenUsage, bool) {
	var last TokenUsage
	found := false
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if usage, ok := ParseTokenUsage(agent, line); ok {
			last, found = usage, true
		}
	}
	return last, found
}

func ParseTokenUsage(agent Agent, line []byte) (TokenUsage, bool) {
	switch agent {
	case AgentCodex:
		var record struct {
			Type    string `json:"type"`
			Payload struct {
				Type string `json:"type"`
				Info *struct {
					Last *struct {
						InputTokens int `json:"input_tokens"`
					} `json:"last_token_usage"`
				} `json:"info"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &record) != nil || record.Type != "event_msg" || record.Payload.Type != "token_count" || record.Payload.Info == nil || record.Payload.Info.Last == nil || record.Payload.Info.Last.InputTokens < 0 {
			return TokenUsage{}, false
		}
		return TokenUsage{InputTokens: record.Payload.Info.Last.InputTokens}, true
	case AgentClaude:
		var record struct {
			Type        string `json:"type"`
			IsSidechain bool   `json:"isSidechain"`
			Message     struct {
				Model string `json:"model"`
				Usage *struct {
					Input       int `json:"input_tokens"`
					CacheCreate int `json:"cache_creation_input_tokens"`
					CacheRead   int `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &record) != nil || record.Type != "assistant" || record.IsSidechain || record.Message.Model == "<synthetic>" || record.Message.Usage == nil || record.Message.Usage.Input < 0 || record.Message.Usage.CacheCreate < 0 || record.Message.Usage.CacheRead < 0 {
			return TokenUsage{}, false
		}
		return TokenUsage{InputTokens: record.Message.Usage.Input + record.Message.Usage.CacheCreate + record.Message.Usage.CacheRead}, true
	case AgentQoder:
		// Qoder writes Claude's usage shape with every token field 0; its only
		// occupancy figure is the ratio. A zero ratio is no figure.
		var record struct {
			Type        string `json:"type"`
			IsSidechain bool   `json:"isSidechain"`
			Message     struct {
				Usage *struct {
					Ratio *float64 `json:"context_usage_ratio"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &record) != nil || record.Type != "assistant" || record.IsSidechain || record.Message.Usage == nil || record.Message.Usage.Ratio == nil || *record.Message.Usage.Ratio <= 0 || *record.Message.Usage.Ratio > 1 {
			return TokenUsage{}, false
		}
		return TokenUsage{ContextRatio: *record.Message.Usage.Ratio}, true
	case AgentGrok:
		// Each streamed update carries the context occupancy so far; the last
		// one equals grok's own contextTokensUsed (signals.json). The per-turn
		// turn_completed usage sums every model call and is not occupancy.
		var record struct {
			Method string `json:"method"`
			Params struct {
				Meta *struct {
					TotalTokens *int `json:"totalTokens"`
				} `json:"_meta"`
			} `json:"params"`
		}
		if json.Unmarshal(line, &record) != nil || !grokUpdateMethods[record.Method] || record.Params.Meta == nil || record.Params.Meta.TotalTokens == nil || *record.Params.Meta.TotalTokens < 0 {
			return TokenUsage{}, false
		}
		return TokenUsage{InputTokens: *record.Params.Meta.TotalTokens}, true
	default:
		return TokenUsage{}, false
	}
}
