package sessionwatch

import (
	"errors"
	"slices"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// LifecycleRecordsFromEvents projects only identity-bearing Codex lifecycle
// envelopes from one scanner-authorized root beyond a durable native-event
// watermark. The native record offset, not journal position, is semantic
// identity.
func LifecycleRecordsFromEvents(events []sessioninventory.NativeEventFact, rootNodeID string, after uint64, launchOrdinal uint64, artifactGeneration, transcriptPath string) ([]LifecycleRecord, uint64) {
	pubs, watermark := lifecyclePublicationsFromEvents(events, rootNodeID, after, launchOrdinal, artifactGeneration, transcriptPath)
	records := make([]LifecycleRecord, 0, len(pubs))
	for _, p := range pubs {
		records = append(records, p.record)
	}
	return records, watermark
}

func lifecyclePublicationsFromEvents(events []sessioninventory.NativeEventFact, rootNodeID string, after uint64, launchOrdinal uint64, artifactGeneration, transcriptPath string) ([]lifecyclePublication, uint64) {
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
	var records []lifecyclePublication
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
		records = append(records, lifecyclePublication{record: LifecycleRecord{
			Version: 1, Agent: string(sessioninventory.AgentCodex), LaunchOrdinal: launchOrdinal,
			ArtifactGeneration: artifactGeneration, Source: "transcript", Outcome: outcome,
			TurnID: fact.Event.TurnID, TranscriptPath: transcriptPath,
			TranscriptOffset: recordOffset, EventTimestamp: fact.Event.Timestamp,
		}, position: fact.Position})
	}
	return records, watermark
}

func lifecycleArtifactIdentity(validation sessioninventory.TargetValidation) (generation, path string, err error) {
	if len(validation.Observations) != 1 {
		return "", "", errors.New("authorized Codex lifecycle artifact identity is unavailable")
	}
	path = validation.Observations[0].Entry.Artifact.RelativePath
	for _, result := range validation.Results {
		generation = result.Fingerprint.GenerationID()
		break
	}
	if generation == "" || path == "" {
		return "", "", errors.New("authorized Codex lifecycle artifact identity is unavailable")
	}
	return generation, path, nil
}

func publishLifecycleBatch(opts Options, rt Runtime, journalPath string, state boundFollowState) (boundFollowState, error) {
	appendRecord := opts.AppendLifecycle
	if appendRecord == nil {
		appendRecord = func(path string, record LifecycleRecord) error {
			return AppendLifecycleRecord(sessionledger.OSRuntime{}, path, record)
		}
	}
	for state.phase == followPublishing {
		record := state.pending.publications[state.index].record
		err := appendRecord(journalPath, record)
		if commitoutcome.Of(err) != commitoutcome.Committed {
			return reduceBoundFollow(state, boundFollowEvent{failure: err}), err
		}
		state = reduceBoundFollow(state, boundFollowEvent{ack: &record})
		rt.Log(adapt.Fired, "codex lifecycle "+record.Outcome+" turn_id="+record.TurnID)
		if err != nil {
			rt.Log(adapt.NearMiss, "lifecycle committed with cleanup warning: "+err.Error())
		}
	}
	return state, nil
}
