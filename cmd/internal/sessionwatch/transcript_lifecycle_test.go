package sessionwatch

import (
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// quietRuntime is the smallest Runtime that publishAuthorizedLifecycle needs:
// it logs, and nothing else on the interface is reached from this path.
type quietRuntime struct{ Runtime }

func (quietRuntime) Log(adapt.Outcome, string) {}

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

// publishAuthorizedLifecycle used to read Fingerprint.GenerationToken directly
// and refuse when it was empty. It is empty on every platform pair supports --
// Linux never populates it, the portable fallback never populates it, and
// darwin populates it only from st_gen, which APFS reports as 0 for
// unprivileged callers. So the Codex lifecycle journal was never written and
// turn-completion notifications never fired.
//
// The Go suite stayed green because its fixtures hardcoded `GenerationToken:
// "gen:1"` -- data production never supplies. This test pins the realistic
// shape: NO generation token, which is what a real observation always looks
// like.
func TestPublishAuthorizedLifecycleWithoutAFilesystemGenerationToken(t *testing.T) {
	const nativeID = "019eff64-6ceb-7e72-9d41-a735a97029ac"
	rootNodeID := sessioninventory.StableID("node", string(sessioninventory.AgentCodex), nativeID)
	relPath := "2026/08/28/rollout-test-" + nativeID + ".jsonl"
	birth := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)

	artifact := sessioninventory.Artifact{
		StorageRoot: "codex-sessions", RelativePath: relPath, Kind: sessioninventory.ArtifactTranscript,
	}
	tracked := map[string]sessioninventory.TargetValidation{
		nativeID: {
			State: sessioninventory.ScannerState{NativeID: nativeID},
			Observations: []sessioninventory.ArtifactObservation{
				{Agent: sessioninventory.AgentCodex, Entry: sessioninventory.FileEntry{Artifact: artifact}},
			},
			Results: map[string]sessioninventory.IncrementalResult{
				"codex-sessions\x00" + relPath: {Fingerprint: sessioninventory.ArtifactFingerprint{
					// Exactly what a real macOS/Linux observation carries.
					StableFileID: "dev:16777230:ino:82772638",
					BirthTime:    &birth,
				}},
			},
		},
	}

	events := []sessioninventory.NativeEventFact{{
		RootNodeID: rootNodeID, Position: 1 << 8,
		Event: sessioninventory.NativeEvent{
			Kind: sessioninventory.EventTerminal, SourceKind: "event_msg.task_complete",
			TurnID: "turn-1", Timestamp: birth,
		},
	}}

	var appended []LifecycleRecord
	opts := Options{
		LaunchOrdinal: 1,
		AppendLifecycle: func(_ string, record LifecycleRecord) error {
			appended = append(appended, record)
			return nil
		},
	}

	watermark, err := publishTestAuthorizedLifecycle(opts, quietRuntime{}, "journal.jsonl", tracked, events, rootNodeID, 0)
	if err != nil {
		t.Fatalf("publishAuthorizedLifecycle: %v -- a real observation was refused", err)
	}
	if watermark != 1<<8 {
		t.Fatalf("watermark = %d, want %d", watermark, 1<<8)
	}
	if len(appended) != 1 {
		t.Fatalf("appended %d records, want 1", len(appended))
	}
	if appended[0].ArtifactGeneration == "" {
		t.Fatal("record carries no artifact generation -- ValidateLifecycleRecord would reject it")
	}
	if err := ValidateLifecycleRecord(appended[0]); err != nil {
		t.Fatalf("produced record fails its own grammar: %v", err)
	}
}

func TestPublishAuthorizedLifecyclePreservesAcknowledgedPrefix(t *testing.T) {
	const sid = "019eff64-6ceb-7e72-9d41-a735a97029ac"
	root := sessioninventory.StableID("node", "codex", sid)
	tracked := map[string]sessioninventory.TargetValidation{sid: {State: sessioninventory.ScannerState{NativeID: sid}, Observations: []sessioninventory.ArtifactObservation{{Entry: sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{RelativePath: "rollout.jsonl"}}}}, Results: map[string]sessioninventory.IncrementalResult{"own": {Fingerprint: sessioninventory.ArtifactFingerprint{StableFileID: "own"}}}}}
	stamp := time.Unix(100, 0)
	events := []sessioninventory.NativeEventFact{
		{RootNodeID: root, Position: 11 << 8, Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTurnStart, SourceKind: "event_msg.task_started", TurnID: "turn", Timestamp: stamp}},
		{RootNodeID: root, Position: 21 << 8, Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventTerminal, SourceKind: "event_msg.task_complete", TurnID: "turn", Timestamp: stamp}},
	}
	calls := 0
	watermark, err := publishTestAuthorizedLifecycle(Options{LaunchOrdinal: 1, AppendLifecycle: func(string, LifecycleRecord) error {
		calls++
		if calls == 2 {
			return fmt.Errorf("publication uncertain")
		}
		return nil
	}}, quietRuntime{}, "journal", tracked, events, root, 0)
	if err == nil || watermark != 11<<8 || calls != 2 {
		t.Fatalf("acknowledged prefix lost: watermark=%d calls=%d err=%v", watermark, calls, err)
	}
}

// Only assemble inputs here: identity, projection and publication all exercise
// the same production functions used by the bound follower.
func publishTestAuthorizedLifecycle(opts Options, rt Runtime, path string, tracked map[string]sessioninventory.TargetValidation, events []sessioninventory.NativeEventFact, root string, after uint64) (uint64, error) {
	for _, validation := range tracked {
		generation, transcript, err := lifecycleArtifactIdentity(validation)
		if err != nil {
			return after, err
		}
		pubs, through := lifecyclePublicationsFromEvents(events, root, after, opts.LaunchOrdinal, generation, transcript)
		state := reduceBoundFollow(boundFollowState{watermark: after}, boundFollowEvent{prepared: &boundPrepared{publications: pubs, watermark: through}})
		state, err = publishLifecycleBatch(opts, rt, path, state)
		return state.watermark, err
	}
	return after, fmt.Errorf("test validation missing")
}
