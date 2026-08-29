package sessioninventory_test

import (
	"bytes"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestLiveJSONLProviderBehaviorMatchesStatefulFake(t *testing.T) {
	if os.Getenv("PAIR_LIVE_NATIVE_SESSIONS") != "1" {
		t.Skip("set PAIR_LIVE_NATIVE_SESSIONS=1 for installed-provider comparison")
	}
	runtime, err := sessioninventory.DefaultOSRuntime(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, agent := range []sessioninventory.Agent{sessioninventory.AgentClaude, sessioninventory.AgentCodex, sessioninventory.AgentMuse} {
		agent := agent
		t.Run(string(agent), func(t *testing.T) {
			observations, diagnostics := sessioninventory.ObserveAgentMetadata(runtime, agent)
			if len(diagnostics) != 0 && len(observations) == 0 {
				t.Fatalf("metadata unavailable: %#v", diagnostics)
			}
			sort.Slice(observations, func(i, j int) bool { return observations[i].Entry.Size < observations[j].Entry.Size })
			attempts := 0
			for _, observation := range observations {
				if observation.Entry.Artifact.Kind != sessioninventory.ArtifactTranscript || observation.Entry.Size > 8<<20 {
					continue
				}
				attempts++
				raw, readErr := runtime.ReadFile(observation.Entry.Artifact, 8<<20)
				if readErr == nil && liveAppendMatchesFake(t, agent, observation.Entry, raw) {
					return
				}
				if attempts >= 8 {
					break
				}
			}
			t.Fatalf("%s installed append behavior diverged from the stateful fake or had no bounded sample", agent)
		})
	}
	t.Run("agy", func(t *testing.T) {
		if !liveAgyAppendMatchesFake(t, runtime) {
			t.Fatal("agy installed append behavior diverged from the stateful fake or had no bounded joined sample")
		}
	})
}

func liveAgyAppendMatchesFake(t *testing.T, installed sessioninventory.OSRuntime) bool {
	t.Helper()
	observations, _ := sessioninventory.ObserveAgentMetadata(installed, sessioninventory.AgentAgy)
	databases := map[string]sessioninventory.FileEntry{}
	transcripts := map[string]sessioninventory.FileEntry{}
	for _, observation := range observations {
		entry := observation.Entry
		switch entry.Artifact.StorageRoot {
		case "agy-conversations":
			if strings.HasSuffix(entry.Artifact.RelativePath, ".db") {
				databases[strings.TrimSuffix(path.Base(entry.Artifact.RelativePath), ".db")] = entry
			}
		case "agy-brain":
			if parts := strings.Split(entry.Artifact.RelativePath, "/"); len(parts) > 0 {
				transcripts[parts[0]] = entry
			}
		}
	}
	schemaQuery, factsQuery := sessioninventory.AgyProviderContractQueries()
	for id, database := range databases {
		transcript, ok := transcripts[id]
		if !ok || transcript.Size > 8<<20 {
			continue
		}
		raw, err := installed.ReadFile(transcript.Artifact, 8<<20)
		if err != nil {
			continue
		}
		split := bytes.IndexByte(raw, '\n') + 1
		if split <= 0 || split >= len(raw) {
			continue
		}
		header, _, headerErr := installed.ReadAt(database.Artifact, 0, 16)
		schema, schemaErr := installed.QuerySQLite(database.Artifact, schemaQuery, 8<<20)
		facts, factsErr := installed.QuerySQLite(database.Artifact, factsQuery, 8<<20)
		if headerErr != nil || schemaErr != nil || factsErr != nil {
			continue
		}
		fake := sessioninventorytest.NewFakeRuntime()
		fake.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentAgy, Name: database.Artifact.StorageRoot})
		fake.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentAgy, Name: transcript.Artifact.StorageRoot})
		fake.PutFile(database, header)
		fake.PutFile(transcript, raw[:split])
		fake.PutSQLite(database.Artifact, schemaQuery, schema)
		fake.PutSQLite(database.Artifact, factsQuery, facts)
		first, frame, frameErr := sessioninventory.FrameJSONLSuffix(sessioninventory.JSONLFrameState{}, raw[:split], 8<<20)
		initial, _, initialErr := sessioninventory.ValidateAgyDelta(fake, database, transcript, nil, first)
		fake.AppendFile(transcript.Artifact, raw[split:], "live-append")
		second, _, secondFrameErr := sessioninventory.FrameJSONLSuffix(frame, raw[split:], 8<<20)
		incremental, _, incrementalErr := sessioninventory.ValidateAgyDelta(fake, database, transcript, &initial, second)
		all, complete, fullFrameErr := sessioninventory.FrameJSONLSuffix(sessioninventory.JSONLFrameState{}, raw, 8<<20)
		fakeFull, _, fakeFullErr := sessioninventory.ValidateAgyDelta(fake, database, transcript, nil, all)
		installedFull, _, installedFullErr := sessioninventory.ValidateAgyDelta(installed, database, transcript, nil, all)
		if frameErr == nil && initialErr == nil && secondFrameErr == nil && incrementalErr == nil && fullFrameErr == nil && len(complete.IncompleteTail) == 0 && fakeFullErr == nil && installedFullErr == nil && reflect.DeepEqual(incremental, fakeFull) && reflect.DeepEqual(fakeFull, installedFull) {
			return true
		}
	}
	return false
}

func liveAppendMatchesFake(t *testing.T, agent sessioninventory.Agent, installed sessioninventory.FileEntry, raw []byte) bool {
	t.Helper()
	for split := bytes.IndexByte(raw, '\n') + 1; split > 0 && split < len(raw); {
		prefix, suffix := raw[:split], raw[split:]
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Agent: agent, Name: installed.Artifact.StorageRoot}
		entry := sessioninventory.FileEntry{Artifact: installed.Artifact, StableFileID: "live-replay", GenerationToken: "gen:1", MutationToken: "ctime:1"}
		runtime.AddRoot(root)
		runtime.PutFile(entry, prefix)
		initial := onlyFile(t, runtime, root)
		first, firstErr := sessioninventory.ObserveStableArtifact(runtime, root, initial, sessioninventory.JSONLFrameState{}, 8<<20)
		state, validateErr := validateLiveRecords(agent, initial, nil, first.Records)
		if firstErr == nil && validateErr == nil && !state.Disputed && state.FirstRecordValidated {
			runtime.AppendFile(entry.Artifact, suffix, "ctime:2")
			appended := onlyFile(t, runtime, root)
			second, secondErr := sessioninventory.ObserveStableArtifact(runtime, root, appended, first.FrameState, 8<<20)
			incremental, incrementalErr := validateLiveRecords(agent, appended, &state, second.Records)
			all, frame, frameErr := sessioninventory.FrameJSONLSuffix(sessioninventory.JSONLFrameState{}, raw, 8<<20)
			full, fullErr := validateLiveRecords(agent, appended, nil, all)
			return secondErr == nil && incrementalErr == nil && frameErr == nil && len(frame.IncompleteTail) == 0 && fullErr == nil && reflect.DeepEqual(incremental, full)
		}
		next := bytes.IndexByte(raw[split:], '\n')
		if next < 0 {
			break
		}
		split += next + 1
	}
	return false
}

func validateLiveRecords(agent sessioninventory.Agent, entry sessioninventory.FileEntry, prior *sessioninventory.ScannerState, records []sessioninventory.FramedJSONLRecord) (sessioninventory.ScannerState, error) {
	switch agent {
	case sessioninventory.AgentClaude:
		state, _, err := sessioninventory.ValidateClaudeDelta(entry, prior, records)
		return state, err
	case sessioninventory.AgentCodex:
		state, _, err := sessioninventory.ValidateCodexDelta(entry, prior, records)
		return state, err
	case sessioninventory.AgentMuse:
		state, _, err := sessioninventory.ValidateMuseDelta(entry, prior, records)
		return state, err
	default:
		return sessioninventory.ScannerState{}, sessioninventory.ErrArtifactChanged
	}
}
