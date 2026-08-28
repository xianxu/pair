package sessioninventory

import (
	"slices"
	"testing"
)

func TestNormalizeNativeEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		agent       Agent
		record      string
		disposition EventDisposition
		kinds       []NativeEventKind
		text        string
	}{
		{name: "claude operator", agent: AgentClaude, record: `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"hello"}]}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventOperator}, text: "hello"},
		{name: "claude operator normalized", agent: AgentClaude, record: `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"=== focus ===\r\nhello  \r\n"}]}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventOperator}, text: "hello"},
		{name: "claude assistant and tool", agent: AgentClaude, record: `{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"private"},{"type":"text","text":"working"},{"type":"tool_use","id":"tool"}]}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventAssistant, EventToolCall}, text: "working"},
		{name: "claude tool result is not operator", agent: AgentClaude, record: `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool","content":"private"}]}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventToolResult}},
		{name: "claude sidechain ignored", agent: AgentClaude, record: `{"type":"user","isSidechain":true,"message":{"role":"user","content":"hidden"}}`, disposition: EventIgnored},
		{name: "codex operator", agent: AgentCodex, record: `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventOperator}, text: "hello"},
		{name: "codex assistant", agent: AgentCodex, record: `{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"working"}]}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventAssistant}, text: "working"},
		{name: "codex function call", agent: AgentCodex, record: `{"type":"response_item","payload":{"type":"function_call","name":"tool"}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventToolCall}},
		{name: "codex custom tool result", agent: AgentCodex, record: `{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"tool"}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventToolResult}},
		{name: "codex terminal", agent: AgentCodex, record: `{"type":"event_msg","payload":{"type":"task_complete"}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventTerminal}},
		{name: "codex aborted terminal", agent: AgentCodex, record: `{"type":"turn_aborted","payload":{"reason":"interrupted"}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventTerminal}},
		{name: "codex reasoning ignored", agent: AgentCodex, record: `{"type":"response_item","payload":{"type":"reasoning"}}`, disposition: EventIgnored},
		{name: "agy wrapped operator", agent: AgentAgy, record: `{"type":"USER_INPUT","content":"<USER_REQUEST>hello</USER_REQUEST>"}`, disposition: EventAccepted, kinds: []NativeEventKind{EventOperator}, text: "hello"},
		{name: "agy assistant", agent: AgentAgy, record: `{"type":"PLANNER_RESPONSE","content":"working"}`, disposition: EventAccepted, kinds: []NativeEventKind{EventAssistant}, text: "working"},
		{name: "agy tool step", agent: AgentAgy, record: `{"type":"RUN_COMMAND","content":"private","status":"done"}`, disposition: EventAccepted, kinds: []NativeEventKind{EventToolResult}},
		{name: "agy structured tool step", agent: AgentAgy, record: `{"type":"RUN_COMMAND","content":{"command":"private"},"status":"done"}`, disposition: EventAccepted, kinds: []NativeEventKind{EventToolResult}},
		{name: "agy error", agent: AgentAgy, record: `{"type":"ERROR_MESSAGE","content":"private"}`, disposition: EventAccepted, kinds: []NativeEventKind{EventTerminal}},
		{name: "agy malformed wrapper", agent: AgentAgy, record: `{"type":"USER_INPUT","content":"<USER_REQUEST>one</USER_REQUEST><USER_REQUEST>two</USER_REQUEST>"}`, disposition: EventNearMiss},
		{name: "muse operator", agent: AgentMuse, record: `{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"started","prompt":"hello"}}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventOperator}, text: "hello"},
		{name: "muse assistant", agent: AgentMuse, record: `{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"assistant_message_committed"}}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventAssistant}},
		{name: "muse tool call", agent: AgentMuse, record: `{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"assistant_tool_calls_committed"}}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventToolCall}},
		{name: "muse tool result", agent: AgentMuse, record: `{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"tool_result_batch_committed"}}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventToolResult}},
		{name: "muse terminal", agent: AgentMuse, record: `{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"terminal"}}}`, disposition: EventAccepted, kinds: []NativeEventKind{EventTerminal}},
		{name: "muse unknown run event near miss", agent: AgentMuse, record: `{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"future_progress"}}}`, disposition: EventNearMiss},
		{name: "known metadata ignored", agent: AgentMuse, record: `{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"agent_tree_initialized"}}`, disposition: EventIgnored},
		{name: "unknown source near miss", agent: AgentCodex, record: `{"type":"future","payload":{}}`, disposition: EventNearMiss},
		{name: "malformed json near miss", agent: AgentClaude, record: `{`, disposition: EventNearMiss},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			events, disposition := NormalizeNativeEvent(test.agent, []byte(test.record))
			if disposition != test.disposition {
				t.Fatalf("disposition = %s, want %s; events=%#v", disposition, test.disposition, events)
			}
			gotKinds := make([]NativeEventKind, len(events))
			for i, event := range events {
				gotKinds[i] = event.Kind
			}
			if !slices.Equal(gotKinds, test.kinds) {
				t.Fatalf("kinds = %#v, want %#v", gotKinds, test.kinds)
			}
			if test.text != "" && (len(events) == 0 || events[0].Text != test.text) {
				t.Fatalf("events = %#v, want first text %q", events, test.text)
			}
		})
	}
}

func FuzzNormalizeNativeEvent(f *testing.F) {
	f.Add("claude", []byte(`{"type":"user","message":{"role":"user","content":"sanitized prompt"}}`))
	f.Add("codex", []byte(`{"type":"response_item","payload":{"type":"future"}}`))
	f.Add("agy", []byte(`{"type":"USER_INPUT","content":"<USER_REQUEST>broken"}`))
	f.Add("muse", []byte(`{"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"terminal"}}}`))
	f.Fuzz(func(t *testing.T, agentName string, record []byte) {
		if len(record) > 2<<20 {
			record = record[:2<<20]
		}
		events, disposition := NormalizeNativeEvent(Agent(agentName), record)
		switch disposition {
		case EventAccepted, EventIgnored, EventNearMiss:
		default:
			t.Fatalf("unknown disposition %q", disposition)
		}
		for _, event := range events {
			switch event.Kind {
			case EventOperator, EventAssistant, EventToolCall, EventToolResult, EventTerminal:
			default:
				t.Fatalf("unknown event kind %q", event.Kind)
			}
			if event.Text != NormalizePairText(event.Text) {
				t.Fatalf("event text is not normalized: %q", event.Text)
			}
		}
	})
}
