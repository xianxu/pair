package sessioninventory

import (
	"testing"
	"time"
)

func TestValidateCodexLifecycleConformance(t *testing.T) {
	stamp := time.Unix(100, 0).UTC()
	valid := []NativeEventFact{
		{RootNodeID: "root", Position: 10, Event: NativeEvent{Kind: EventTurnStart, SourceKind: "event_msg.task_started", TurnID: "turn", Timestamp: stamp}},
		{RootNodeID: "root", Position: 20, Event: NativeEvent{Kind: EventTerminal, SourceKind: "event_msg.task_complete", TurnID: "turn", Timestamp: stamp.Add(time.Second)}},
	}
	if err := ValidateCodexLifecycleConformance(valid); err != nil {
		t.Fatal(err)
	}
	for name, events := range map[string][]NativeEventFact{
		"missing opener":    {valid[1]},
		"cross root":        {valid[0], {RootNodeID: "other", Position: 20, Event: valid[1].Event}},
		"missing timestamp": {{RootNodeID: "root", Position: 10, Event: NativeEvent{Kind: EventTurnStart, SourceKind: "event_msg.task_started", TurnID: "turn"}}, valid[1]},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCodexLifecycleConformance(events); err == nil {
				t.Fatal("invalid lifecycle envelopes passed conformance")
			}
		})
	}
}
