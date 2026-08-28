package sessioninventory

import (
	"bytes"
	"encoding/json"
)

type TokenUsage struct {
	InputTokens int `json:"input_tokens"`
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
	default:
		return TokenUsage{}, false
	}
}
