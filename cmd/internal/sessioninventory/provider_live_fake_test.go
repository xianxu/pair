package sessioninventory_test

import (
	"bytes"
	"os"
	"reflect"
	"sort"
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
