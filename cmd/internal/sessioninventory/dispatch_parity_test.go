package sessioninventory_test

import (
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

// BR-18: every session-side dispatch reachable from the agent inventory is
// probed through one fixture per agent. Deleting any agent's case from
// ProviderContractFor, artifactScannerShape, observationNativeID,
// ValidateTargetWork, AdvanceTargetValidation or NativeEventsFromRecords
// turns at least one sub-assertion red.
func TestEveryAgentDispatchParity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		agent    sessioninventory.Agent
		root     string
		schema   string
		nativeID string
		relative string
	}{
		{sessioninventory.AgentClaude, "claude-projects", "claude-v1", "11111111-1111-4111-8111-111111111111", "-repo/11111111-1111-4111-8111-111111111111.jsonl"},
		{sessioninventory.AgentCodex, "codex-sessions", "codex-v1", "019d1111-1111-7111-8111-111111111111", "2026/08/28/rollout-root-019d1111-1111-7111-8111-111111111111.jsonl"},
		{sessioninventory.AgentMuse, "muse-sessions", "muse-v1", "77777777-7777-4777-8777-777777777777", "2026/08/28/77777777-7777-4777-8777-777777777777/session.jsonl"},
		{sessioninventory.AgentQoder, "qoder-projects", "qoder-v1", "11111111-1111-4111-8111-111111111111", "-repo/11111111-1111-4111-8111-111111111111.jsonl"},
	} {
		tc := tc
		t.Run(string(tc.agent), func(t *testing.T) {
			t.Parallel()
			runtime := sessioninventorytest.NewFakeRuntime()
			loadNativeFixture(t, runtime, tc.agent, tc.root, filepath.Join("testdata", "native", string(tc.agent), "v1", tc.root))

			result := sessioninventory.ScannerForAgent(tc.agent)(runtime)
			if len(result.Facts) == 0 {
				t.Fatal("scanner produced no facts")
			}
			found := false
			for _, fact := range result.Facts {
				if fact.NativeID == tc.nativeID {
					found = true
				}
			}
			if !found {
				t.Fatalf("scanner did not produce fact with nativeID %s", tc.nativeID)
			}

			if _, ok := sessioninventory.ProviderContractFor(tc.agent, tc.root, tc.schema); !ok {
				t.Fatal("ProviderContractFor unrecognized")
			}

			root := sessioninventory.StorageRoot{Agent: tc.agent, Name: tc.root}
			files, err := runtime.ListFiles(root)
			if err != nil || len(files) == 0 {
				t.Fatal("no files in root")
			}
			var entry sessioninventory.FileEntry
			for _, f := range files {
				if f.Artifact.RelativePath == tc.relative {
					entry = f
					break
				}
			}
			entry.Artifact.Kind = sessioninventory.ArtifactTranscript
			observed, err := sessioninventory.ObserveStableArtifact(runtime, root, entry, sessioninventory.JSONLFrameState{}, 8<<20)
			if err != nil {
				t.Fatalf("ObserveStableArtifact: %v", err)
			}

			state, diagnostics := validateAgentDelta(tc.agent, entry, nil, observed.Records)
			if state.Disputed {
				t.Fatalf("ValidateDelta disputed: %#v", diagnostics)
			}
			if !state.FirstRecordValidated {
				t.Fatal("first record not validated")
			}

			events, _ := sessioninventory.NativeEventsFromRecords(tc.agent, string(tc.agent)+":"+tc.nativeID, observed.Records)
			if len(events) == 0 {
				t.Fatal("no events from fixture records")
			}
		})
	}
}

func validateAgentDelta(agent sessioninventory.Agent, entry sessioninventory.FileEntry, prior *sessioninventory.ScannerState, records []sessioninventory.FramedJSONLRecord) (sessioninventory.ScannerState, []sessioninventory.Diagnostic) {
	switch agent {
	case sessioninventory.AgentClaude:
		state, d, _ := sessioninventory.ValidateClaudeDelta(entry, prior, records)
		return state, d
	case sessioninventory.AgentCodex:
		state, d, _ := sessioninventory.ValidateCodexDelta(entry, prior, records)
		return state, d
	case sessioninventory.AgentMuse:
		state, d, _ := sessioninventory.ValidateMuseDelta(entry, prior, records)
		return state, d
	case sessioninventory.AgentQoder:
		state, d, _ := sessioninventory.ValidateQoderDelta(entry, prior, records)
		return state, d
	}
	return sessioninventory.ScannerState{}, nil
}
