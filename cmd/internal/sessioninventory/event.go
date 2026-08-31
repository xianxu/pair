package sessioninventory

import (
	"encoding/json"
	"strings"
)

type NativeEventKind string

const (
	EventOperator   NativeEventKind = "operator"
	EventAssistant  NativeEventKind = "assistant"
	EventToolCall   NativeEventKind = "tool_call"
	EventToolResult NativeEventKind = "tool_result"
	EventTerminal   NativeEventKind = "terminal"
)

type EventDisposition string

const (
	EventAccepted EventDisposition = "accepted"
	EventIgnored  EventDisposition = "ignored"
	EventNearMiss EventDisposition = "near_miss"
)

// NativeEvent is a content-minimal causal event. Only operator and assistant
// text is retained; tool payloads and terminal details are deliberately not.
// pair:155-concept pure new M2
type NativeEvent struct {
	Kind       NativeEventKind `json:"kind"`
	Text       string          `json:"text"`
	SourceKind string          `json:"source_kind"`
}

func (e NativeEvent) Progress() bool {
	switch e.Kind {
	case EventAssistant, EventToolCall, EventToolResult, EventTerminal:
		return true
	default:
		return false
	}
}

func nativeTextEvent(kind NativeEventKind, text, source string) (NativeEvent, bool) {
	text = NormalizePairText(text)
	return NativeEvent{Kind: kind, Text: text, SourceKind: source}, text != ""
}

func NormalizeNativeEvent(agent Agent, record []byte) ([]NativeEvent, EventDisposition) {
	switch agent {
	case AgentClaude:
		return normalizeClaudeEvent(record)
	case AgentCodex:
		return normalizeCodexEvent(record)
	case AgentAgy:
		return normalizeAgyEvent(record)
	case AgentMuse:
		return normalizeMuseEvent(record)
	default:
		return nil, EventNearMiss
	}
}

type textBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func normalizeClaudeEvent(record []byte) ([]NativeEvent, EventDisposition) {
	var value struct {
		Type        string `json:"type"`
		IsSidechain bool   `json:"isSidechain"`
		Operation   string `json:"operation"`
		Content     string `json:"content"`
		Message     struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if decodeStrictJSON(record, &value) != nil {
		return nil, EventNearMiss
	}
	if value.IsSidechain {
		return nil, EventIgnored
	}
	if value.Type == "queue-operation" {
		switch value.Operation {
		case "enqueue":
			event, ok := nativeTextEvent(EventOperator, value.Content, "queue-operation.enqueue")
			if !ok {
				return nil, EventNearMiss
			}
			return []NativeEvent{event}, EventAccepted
		case "remove":
			return nil, EventIgnored
		default:
			return nil, EventNearMiss
		}
	}
	if value.Type != "user" && value.Type != "assistant" {
		switch value.Type {
		case "attachment", "ai-title", "last-prompt", "atis-latch", "system":
			return nil, EventIgnored
		default:
			return nil, EventNearMiss
		}
	}
	if value.Message.Role != value.Type {
		return nil, EventNearMiss
	}
	var scalar string
	if json.Unmarshal(value.Message.Content, &scalar) == nil {
		kind := EventOperator
		if value.Type == "assistant" {
			kind = EventAssistant
		}
		event, ok := nativeTextEvent(kind, scalar, "message."+value.Type)
		if !ok {
			return nil, EventIgnored
		}
		return []NativeEvent{event}, EventAccepted
	}
	var blocks []textBlock
	if json.Unmarshal(value.Message.Content, &blocks) != nil {
		return nil, EventNearMiss
	}
	if value.Type == "user" {
		var results []NativeEvent
		var texts []string
		for _, block := range blocks {
			switch block.Type {
			case "tool_result":
				results = append(results, NativeEvent{Kind: EventToolResult, SourceKind: "message.user.tool_result"})
			case "text":
				if block.Text != "" {
					texts = append(texts, block.Text)
				}
			default:
				return nil, EventNearMiss
			}
		}
		if len(results) != 0 {
			return results, EventAccepted
		}
		if len(texts) != 0 {
			event, ok := nativeTextEvent(EventOperator, strings.Join(texts, "\n"), "message.user.text")
			if ok {
				return []NativeEvent{event}, EventAccepted
			}
		}
		return nil, EventIgnored
	}
	var events []NativeEvent
	var texts []string
	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		case "tool_use":
			events = append(events, NativeEvent{Kind: EventToolCall, SourceKind: "message.assistant.tool_use"})
		case "thinking":
		default:
			return nil, EventNearMiss
		}
	}
	if len(texts) != 0 {
		if event, ok := nativeTextEvent(EventAssistant, strings.Join(texts, "\n"), "message.assistant.text"); ok {
			events = append([]NativeEvent{event}, events...)
		}
	}
	if len(events) == 0 {
		return nil, EventIgnored
	}
	return events, EventAccepted
}

func normalizeCodexEvent(record []byte) ([]NativeEvent, EventDisposition) {
	var value struct {
		Type    string `json:"type"`
		Payload struct {
			Type    string      `json:"type"`
			Role    string      `json:"role"`
			Content []textBlock `json:"content"`
		} `json:"payload"`
	}
	if decodeStrictJSON(record, &value) != nil {
		return nil, EventNearMiss
	}
	if value.Type == "response_item" {
		switch value.Payload.Type {
		case "message":
			kind, blockKind := EventOperator, "input_text"
			if value.Payload.Role == "assistant" {
				kind, blockKind = EventAssistant, "output_text"
			} else if value.Payload.Role != "user" {
				return nil, EventIgnored
			}
			var texts []string
			for _, block := range value.Payload.Content {
				if block.Type != blockKind {
					return nil, EventNearMiss
				}
				if block.Text != "" {
					texts = append(texts, block.Text)
				}
			}
			if len(texts) == 0 {
				return nil, EventIgnored
			}
			event, ok := nativeTextEvent(kind, strings.Join(texts, "\n"), "response_item.message."+value.Payload.Role)
			if !ok {
				return nil, EventIgnored
			}
			return []NativeEvent{event}, EventAccepted
		case "function_call", "custom_tool_call", "tool_search_call", "web_search_call":
			return []NativeEvent{{Kind: EventToolCall, SourceKind: "response_item." + value.Payload.Type}}, EventAccepted
		case "function_call_output", "custom_tool_call_output", "tool_search_output":
			return []NativeEvent{{Kind: EventToolResult, SourceKind: "response_item." + value.Payload.Type}}, EventAccepted
		case "reasoning", "agent_message":
			return nil, EventIgnored
		default:
			return nil, EventNearMiss
		}
	}
	if value.Type == "event_msg" {
		switch value.Payload.Type {
		case "task_complete", "turn_aborted":
			return []NativeEvent{{Kind: EventTerminal, SourceKind: "event_msg." + value.Payload.Type}}, EventAccepted
		case "token_count", "agent_message", "patch_apply_end", "sub_agent_activity", "task_started", "user_message", "thread_settings_applied", "context_compacted", "web_search_end", "thread_rolled_back":
			return nil, EventIgnored
		default:
			return nil, EventNearMiss
		}
	}
	switch value.Type {
	case "turn_aborted":
		return []NativeEvent{{Kind: EventTerminal, SourceKind: value.Type}}, EventAccepted
	case "session_meta", "turn_context", "world_state", "compacted", "inter_agent_communication_metadata":
		return nil, EventIgnored
	default:
		return nil, EventNearMiss
	}
}

func normalizeAgyEvent(record []byte) ([]NativeEvent, EventDisposition) {
	var value struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}
	if decodeStrictJSON(record, &value) != nil {
		return nil, EventNearMiss
	}
	switch value.Type {
	case "USER_INPUT":
		var content string
		if json.Unmarshal(value.Content, &content) != nil {
			return nil, EventNearMiss
		}
		text, ok := agyOperatorText(content)
		if !ok {
			return nil, EventNearMiss
		}
		event, ok := nativeTextEvent(EventOperator, text, "USER_INPUT")
		if !ok {
			return nil, EventIgnored
		}
		return []NativeEvent{event}, EventAccepted
	case "PLANNER_RESPONSE":
		var content string
		if json.Unmarshal(value.Content, &content) != nil {
			return nil, EventNearMiss
		}
		event, ok := nativeTextEvent(EventAssistant, content, "PLANNER_RESPONSE")
		if !ok {
			return nil, EventIgnored
		}
		return []NativeEvent{event}, EventAccepted
	case "RUN_COMMAND", "VIEW_FILE", "GREP_SEARCH", "CODE_ACTION", "GENERIC", "LIST_DIRECTORY", "SEARCH_WEB", "INVOKE_SUBAGENT", "ASK_QUESTION":
		return []NativeEvent{{Kind: EventToolResult, SourceKind: value.Type}}, EventAccepted
	case "ERROR_MESSAGE":
		return []NativeEvent{{Kind: EventTerminal, SourceKind: value.Type}}, EventAccepted
	case "SYSTEM_MESSAGE", "CONVERSATION_HISTORY", "CHECKPOINT", "DIRECTORY_RULES":
		return nil, EventIgnored
	default:
		return nil, EventNearMiss
	}
}

func agyOperatorText(content string) (string, bool) {
	const open, close = "<USER_REQUEST>", "</USER_REQUEST>"
	openCount, closeCount := strings.Count(content, open), strings.Count(content, close)
	if openCount == 0 && closeCount == 0 {
		return content, true
	}
	if openCount != 1 || closeCount != 1 {
		return "", false
	}
	start := strings.Index(content, open) + len(open)
	end := strings.Index(content, close)
	if end < start {
		return "", false
	}
	return content[start:end], true
}

func normalizeMuseEvent(record []byte) ([]NativeEvent, EventDisposition) {
	var value struct {
		RecordType  string `json:"record_type"`
		PayloadType string `json:"payload_type"`
		Payload     struct {
			Kind  string `json:"kind"`
			Event struct {
				Kind   string `json:"kind"`
				Prompt string `json:"prompt"`
			} `json:"event"`
		} `json:"payload"`
	}
	if decodeStrictJSON(record, &value) != nil {
		return nil, EventNearMiss
	}
	if value.RecordType != "event" || value.PayloadType != "runtime.session" {
		return nil, EventNearMiss
	}
	if value.Payload.Kind == "agent_tree_initialized" || value.Payload.Kind == "task" || value.Payload.Kind == "approval" || value.Payload.Kind == "route_facts" || value.Payload.Kind == "metadata" {
		return nil, EventIgnored
	}
	if value.Payload.Kind != "run" {
		return nil, EventNearMiss
	}
	switch value.Payload.Event.Kind {
	case "started":
		event, ok := nativeTextEvent(EventOperator, value.Payload.Event.Prompt, "runtime.session.run.started")
		if !ok {
			return nil, EventNearMiss
		}
		return []NativeEvent{event}, EventAccepted
	case "assistant_message_committed":
		return []NativeEvent{{Kind: EventAssistant, SourceKind: "runtime.session.run.assistant_message_committed"}}, EventAccepted
	case "assistant_tool_calls_committed":
		return []NativeEvent{{Kind: EventToolCall, SourceKind: "runtime.session.run.assistant_tool_calls_committed"}}, EventAccepted
	case "tool_result_batch_committed":
		return []NativeEvent{{Kind: EventToolResult, SourceKind: "runtime.session.run.tool_result_batch_committed"}}, EventAccepted
	case "terminal":
		return []NativeEvent{{Kind: EventTerminal, SourceKind: "runtime.session.run.terminal"}}, EventAccepted
	default:
		return nil, EventNearMiss
	}
}
