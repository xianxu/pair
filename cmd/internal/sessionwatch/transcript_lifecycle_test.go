package sessionwatch

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestLifecycleRecordsFromEventsPublishesCompletedShortTurnInOrder(t *testing.T) {
	stamp := time.Unix(100, 0).UTC()
	position := func(offset uint64) uint64 { return ((offset + 1) << 8) }
	events := []sessioninventory.NativeEventFact{
		{RootNodeID: "other", Position: position(1), Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTerminal, SourceKind: "event_msg.task_complete", TurnID: "wrong", Timestamp: stamp}},
		{RootNodeID: "root", Position: position(20), Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTerminal, SourceKind: "event_msg.task_complete", TurnID: "turn-1", Timestamp: stamp.Add(time.Second)}},
		{RootNodeID: "root", Position: position(10), Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTurnStart, SourceKind: "event_msg.task_started", TurnID: "turn-1", Timestamp: stamp}},
	}
	records, watermark := LifecycleRecordsFromEvents(events, "root", 0, 7, "gen:1", "rollout.jsonl")
	if len(records) != 2 || records[0].Outcome != "started" || records[1].Outcome != "completed" || records[0].TurnID != "turn-1" || records[1].TurnID != "turn-1" {
		t.Fatalf("records=%+v", records)
	}
	if records[0].TranscriptOffset != 10 || records[1].TranscriptOffset != 20 || watermark != position(20) {
		t.Fatalf("offsets/watermark=%+v %d", records, watermark)
	}
}

func TestLifecycleRecordsFromEventsUsesWatermarkAndRejectsUnkeyedOrUnknown(t *testing.T) {
	stamp := time.Unix(100, 0).UTC()
	position := func(offset uint64) uint64 { return ((offset + 1) << 8) }
	events := []sessioninventory.NativeEventFact{
		{RootNodeID: "root", Position: position(10), Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTurnStart, SourceKind: "event_msg.task_started", TurnID: "old", Timestamp: stamp}},
		{RootNodeID: "root", Position: position(20), Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTerminal, SourceKind: "event_msg.future", TurnID: "future", Timestamp: stamp}},
		{RootNodeID: "root", Position: position(30), Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTerminal, SourceKind: "event_msg.task_complete", Timestamp: stamp}},
	}
	records, watermark := LifecycleRecordsFromEvents(events, "root", position(10), 7, "gen:1", "rollout.jsonl")
	if len(records) != 0 || watermark != position(30) {
		t.Fatalf("records=%+v watermark=%d", records, watermark)
	}
}
