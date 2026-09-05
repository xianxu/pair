package sessionwatch

import (
	"errors"
	"slices"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// LifecycleRecordsFromEvents projects only identity-bearing Codex lifecycle
// envelopes from one scanner-authorized root beyond a durable native-event
// watermark. The native record offset, not journal position, is semantic
// identity.
func LifecycleRecordsFromEvents(events []sessioninventory.NativeEventFact, rootNodeID string, after uint64, launchOrdinal uint64, artifactGeneration, transcriptPath string) ([]LifecycleRecord, uint64) {
	ordered := append([]sessioninventory.NativeEventFact(nil), events...)
	slices.SortFunc(ordered, func(a, b sessioninventory.NativeEventFact) int {
		if a.Position < b.Position {
			return -1
		}
		if a.Position > b.Position {
			return 1
		}
		return 0
	})
	watermark := after
	var records []LifecycleRecord
	for _, fact := range ordered {
		if fact.RootNodeID != rootNodeID || fact.Position <= after {
			continue
		}
		if fact.Position > watermark {
			watermark = fact.Position
		}
		outcome := ""
		switch {
		case fact.Event.Kind == sessioninventory.EventTurnStart && fact.Event.SourceKind == "event_msg.task_started":
			outcome = "started"
		case fact.Event.Kind == sessioninventory.EventTerminal && fact.Event.SourceKind == "event_msg.task_complete":
			outcome = "completed"
		case fact.Event.Kind == sessioninventory.EventTerminal && fact.Event.SourceKind == "event_msg.turn_aborted":
			outcome = "aborted"
		}
		if outcome == "" || fact.Event.TurnID == "" || fact.Event.Timestamp.IsZero() {
			continue
		}
		recordOffset := int64(fact.Position>>8) - 1
		if recordOffset < 0 {
			continue
		}
		records = append(records, LifecycleRecord{
			Version: 1, Agent: string(sessioninventory.AgentCodex), LaunchOrdinal: launchOrdinal,
			ArtifactGeneration: artifactGeneration, Source: "transcript", Outcome: outcome,
			TurnID: fact.Event.TurnID, TranscriptPath: transcriptPath,
			TranscriptOffset: recordOffset, EventTimestamp: fact.Event.Timestamp,
		})
	}
	return records, watermark
}

func publishAuthorizedLifecycle(opts Options, rt Runtime, journalPath string, tracked map[string]sessioninventory.TargetValidation, events []sessioninventory.NativeEventFact, rootNodeID string, after uint64) (uint64, error) {
	var generation, transcriptPath string
	for _, validation := range tracked {
		if sessioninventory.StableID("node", string(sessioninventory.AgentCodex), validation.State.NativeID) != rootNodeID || len(validation.Observations) != 1 {
			continue
		}
		transcriptPath = validation.Observations[0].Entry.Artifact.RelativePath
		for _, result := range validation.Results {
			// GenerationID, not GenerationToken: the raw token is optional and
			// in practice never present -- Linux never populates it and APFS
			// reports st_gen as 0 -- so requiring it rejected every artifact on
			// every platform, and no lifecycle record was ever written.
			generation = result.Fingerprint.GenerationID()
			break
		}
		break
	}
	if generation == "" || transcriptPath == "" {
		return after, errors.New("authorized Codex lifecycle artifact identity is unavailable")
	}
	records, watermark := LifecycleRecordsFromEvents(events, rootNodeID, after, opts.LaunchOrdinal, generation, transcriptPath)
	appendRecord := opts.AppendLifecycle
	if appendRecord == nil {
		appendRecord = func(path string, record LifecycleRecord) error {
			return AppendLifecycleRecord(sessionledger.OSRuntime{}, path, record)
		}
	}
	for _, record := range records {
		if err := appendRecord(journalPath, record); err != nil {
			return after, err
		}
		rt.Log(adapt.Fired, "codex lifecycle "+record.Outcome+" turn_id="+record.TurnID)
	}
	return watermark, nil
}
