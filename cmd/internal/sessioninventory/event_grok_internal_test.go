package sessioninventory

import "testing"

// The normalizer table is generated from the two sets it is built on, so a kind
// added to either is exercised by construction; one unknown kind pins the drift
// signal.
func TestNormalizeGrokEventKinds(t *testing.T) {
	record := func(method, kind, extra string) []byte {
		return []byte(`{"timestamp":1791400000,"method":"` + method + `","params":{"sessionId":"11111111-1111-4111-8111-111111111111","update":{"sessionUpdate":"` + kind + `"` + extra + `}}}`)
	}
	for kind, want := range grokMappedKinds {
		extra := `,"content":{"type":"text","text":"hello"}`
		if kind == "tool_call_update" {
			extra = `,"status":"completed"`
		}
		method := "session/update"
		if kind == "turn_completed" {
			method = "_x.ai/session/update"
		}
		events, disposition := NormalizeNativeEvent(AgentGrok, record(method, kind, extra))
		if disposition != EventAccepted || len(events) != 1 || events[0].Kind != want {
			t.Errorf("%s: events=%#v disposition=%v, want one %s", kind, events, disposition, want)
		}
	}
	for kind := range grokIgnoredKinds {
		if _, disposition := NormalizeNativeEvent(AgentGrok, record("session/update", kind, "")); disposition != EventIgnored {
			t.Errorf("%s: disposition=%v, want ignored", kind, disposition)
		}
	}
	for name, raw := range map[string][]byte{
		"unknown kind":        record("session/update", "brand_new_update", ""),
		"unknown method":      record("session/request", "user_message_chunk", `,"content":{"type":"text","text":"x"}`),
		"malformed":           []byte(`{"method":`),
		"empty operator text": record("session/update", "user_message_chunk", `,"content":{"type":"text","text":"  "}`),
	} {
		if _, disposition := NormalizeNativeEvent(AgentGrok, raw); disposition != EventNearMiss {
			t.Errorf("%s: disposition=%v, want near-miss", name, disposition)
		}
	}
	// A tool update that has not finished is progress noise, not a result.
	if _, disposition := NormalizeNativeEvent(AgentGrok, record("session/update", "tool_call_update", `,"status":"in_progress"`)); disposition != EventIgnored {
		t.Errorf("in-progress tool update: disposition=%v, want ignored", disposition)
	}
	// The operator text is what round qualification matches, so it survives.
	events, _ := NormalizeNativeEvent(AgentGrok, record("session/update", "user_message_chunk", `,"content":{"type":"text","text":"first line\nsecond line"}`))
	if len(events) != 1 || events[0].Kind != EventOperator || events[0].Text == "" {
		t.Errorf("operator event = %#v", events)
	}
}
