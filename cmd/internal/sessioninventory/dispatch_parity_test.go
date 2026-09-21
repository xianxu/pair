package sessioninventory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestEveryAgentDispatchParity(t *testing.T) {
	t.Parallel()
	agents := sessioninventory.SupportedAgents()
	if len(agents) == 0 {
		t.Fatal("SupportedAgents() returned empty")
	}

	for _, agent := range agents {
		agent := agent
		t.Run(string(agent), func(t *testing.T) {
			t.Parallel()
			runtime := sessioninventorytest.NewFakeRuntime()
			rootName := agentFixtureRoot(agent)
			fixtureDir := filepath.Join("testdata", "native", string(agent), "v1", rootName)
			loadNativeFixture(t, runtime, agent, rootName, fixtureDir)

			observations, diagnostics := sessioninventory.ObserveAgentMetadata(runtime, agent)
			if len(observations) == 0 {
				t.Fatalf("ObserveAgentMetadata: no observations, diagnostics=%v", diagnostics)
			}

			if agent == sessioninventory.AgentAgy {
				return
			}

			if _, ok := sessioninventory.ProviderContractFor(agent, rootName, agentSchema(agent)); !ok {
				t.Fatal("ProviderContractFor unrecognized")
			}

			validations, validateDiagnostics := sessioninventory.ValidateTargetWork(runtime, agent, observations)
			if len(validations) == 0 {
				t.Fatalf("ValidateTargetWork: no validations, diagnostics=%v", validateDiagnostics)
			}

			var rootValidation *sessioninventory.TargetValidation
			for i := range validations {
				if validations[i].State.Role == sessioninventory.RoleRoot {
					rootValidation = &validations[i]
					break
				}
			}
			if rootValidation == nil {
				t.Fatalf("no root validation found in %d validations", len(validations))
			}

			events, _ := sessioninventory.NativeEventsFromRecords(agent, string(agent)+":"+rootValidation.State.NativeID, rootValidation.Results[targetKey(rootValidation.Observations[0].Entry.Artifact)].Records)
			if len(events) == 0 {
				t.Fatal("no events from fixture records")
			}
		})
	}
}

func TestAdvanceTargetValidationPerAgent(t *testing.T) {
	t.Parallel()
	for _, agent := range []sessioninventory.Agent{
		sessioninventory.AgentClaude, sessioninventory.AgentCodex,
		sessioninventory.AgentMuse, sessioninventory.AgentQoder,
	} {
		agent := agent
		t.Run(string(agent), func(t *testing.T) {
			t.Parallel()
			runtime := sessioninventorytest.NewFakeRuntime()
			rootName := agentFixtureRoot(agent)
			nativeID := agentFixtureNativeID(agent)
			relative := agentFixtureRelative(agent)
			root := sessioninventory.StorageRoot{Agent: agent, Name: rootName}
			runtime.AddRoot(root)

			raw, err := os.ReadFile(filepath.Join("testdata", "native", string(agent), "v1", rootName, filepath.FromSlash(relative)))
			if err != nil {
				t.Fatal(err)
			}
			birth := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
			entry := sessioninventory.FileEntry{
				Artifact:        sessioninventory.Artifact{StorageRoot: rootName, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript},
				StableFileID:    "dev:1/ino:1",
				GenerationToken: "gen:1",
				MutationToken:   "ctime:1",
				BirthTime:       &birth,
				ModTime:         &birth,
			}
			runtime.PutFile(entry, raw)

			observations, _ := sessioninventory.ObserveAgentMetadata(runtime, agent)
			validations, _ := sessioninventory.ValidateTargetWork(runtime, agent, observations)
			var prior *sessioninventory.TargetValidation
			for i := range validations {
				if validations[i].State.NativeID == nativeID {
					prior = &validations[i]
					break
				}
			}
			if prior == nil {
				t.Fatal("no prior validation for native ID")
			}

			runtime.AppendFile(entry.Artifact, agentAppendRecord(agent, nativeID), "ctime:2")
			current, _ := sessioninventory.ObserveAgentMetadata(runtime, agent)
			var filtered []sessioninventory.ArtifactObservation
			for _, obs := range current {
				if obs.Entry.Artifact.RelativePath == relative {
					filtered = append(filtered, obs)
				}
			}
			advanced, _, advanceErr := sessioninventory.AdvanceTargetValidation(runtime, *prior, filtered)
			if advanceErr != nil {
				t.Fatalf("AdvanceTargetValidation: %v", advanceErr)
			}
			key := targetKey(entry.Artifact)
			if advanced.Results[key].RawObservedOffset <= prior.Results[key].RawObservedOffset {
				t.Fatal("advance did not move offset forward")
			}
		})
	}
}

func TestValidateTargetWorkRejectsUnknownAgent(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	agent := sessioninventory.Agent("future")
	root := sessioninventory.StorageRoot{Agent: agent, Name: "future-projects", Path: "/native/future"}
	runtime.AddRoot(root)
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/test.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1"}, []byte(`{"type":"user"}`+"\n"))
	observation := sessioninventory.ArtifactObservation{
		Agent:            agent,
		Entry:            sessioninventory.FileEntry{Artifact: artifact, StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1"},
		ScannerSchema:    "future-v1",
		ProviderContract: "future-jsonl-v1",
	}
	_, diagnostics := sessioninventory.ValidateTargetWork(runtime, agent, []sessioninventory.ArtifactObservation{observation})
	found := false
	for _, d := range diagnostics {
		if d.Code == sessioninventory.DiagnosticSchemaNearMiss {
			found = true
		}
	}
	if !found {
		t.Fatalf("unknown agent did not produce schema_near_miss diagnostic: %v", diagnostics)
	}
}

func TestAdvanceTargetValidationRejectsUnknownAgent(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/native/claude"}
	runtime.AddRoot(root)
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/test.jsonl", Kind: sessioninventory.ArtifactTranscript}
	entry := sessioninventory.FileEntry{Artifact: artifact, StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1"}
	runtime.PutFile(entry, []byte(`{"type":"user","timestamp":"2026-08-28T09:01:00Z","sessionId":"11111111-1111-4111-8111-111111111111","isSidechain":false}`+"\n"))
	prior := sessioninventory.TargetValidation{
		State: sessioninventory.ScannerState{Agent: sessioninventory.Agent("future"), NativeID: "test"},
		Observations: []sessioninventory.ArtifactObservation{{
			Agent: sessioninventory.Agent("future"), Entry: entry, ScannerSchema: "future-v1",
		}},
		Results: map[string]sessioninventory.IncrementalResult{
			targetKey(artifact): {},
		},
	}
	current := []sessioninventory.ArtifactObservation{{
		Agent: sessioninventory.Agent("future"), Entry: entry, ScannerSchema: "future-v1",
	}}
	_, _, err := sessioninventory.AdvanceTargetValidation(runtime, prior, current)
	if err == nil {
		t.Fatal("unknown agent advance did not error")
	}
}

func agentFixtureRoot(agent sessioninventory.Agent) string {
	switch agent {
	case sessioninventory.AgentClaude:
		return "claude-projects"
	case sessioninventory.AgentCodex:
		return "codex-sessions"
	case sessioninventory.AgentMuse:
		return "muse-sessions"
	case sessioninventory.AgentQoder:
		return "qoder-projects"
	case sessioninventory.AgentAgy:
		return "agy-brain"
	}
	return ""
}

func agentSchema(agent sessioninventory.Agent) string {
	switch agent {
	case sessioninventory.AgentClaude:
		return "claude-v1"
	case sessioninventory.AgentCodex:
		return "codex-v1"
	case sessioninventory.AgentMuse:
		return "muse-v1"
	case sessioninventory.AgentQoder:
		return "qoder-v1"
	case sessioninventory.AgentAgy:
		return "agy-transcript-v1"
	}
	return ""
}

func agentFixtureNativeID(agent sessioninventory.Agent) string {
	switch agent {
	case sessioninventory.AgentClaude, sessioninventory.AgentQoder:
		return "11111111-1111-4111-8111-111111111111"
	case sessioninventory.AgentCodex:
		return "019d1111-1111-7111-8111-111111111111"
	case sessioninventory.AgentMuse:
		return "77777777-7777-4777-8777-777777777777"
	}
	return ""
}

func agentFixtureRelative(agent sessioninventory.Agent) string {
	switch agent {
	case sessioninventory.AgentClaude:
		return "-repo/11111111-1111-4111-8111-111111111111.jsonl"
	case sessioninventory.AgentCodex:
		return "2026/08/28/rollout-root-019d1111-1111-7111-8111-111111111111.jsonl"
	case sessioninventory.AgentMuse:
		return "2026/08/28/77777777-7777-4777-8777-777777777777/session.jsonl"
	case sessioninventory.AgentQoder:
		return "-repo/11111111-1111-4111-8111-111111111111.jsonl"
	}
	return ""
}

func targetKey(artifact sessioninventory.Artifact) string {
	return artifact.StorageRoot + "\x00" + artifact.RelativePath
}

func agentAppendRecord(agent sessioninventory.Agent, nativeID string) []byte {
	switch agent {
	case sessioninventory.AgentClaude, sessioninventory.AgentQoder:
		return []byte(`{"type":"user","timestamp":"2026-08-28T09:02:00Z","sessionId":"` + nativeID + `","isSidechain":false}` + "\n")
	case sessioninventory.AgentCodex:
		return []byte(`{"timestamp":"2026-08-28T10:05:00Z","type":"event_msg","payload":{"type":"user_message","message":"next"}}` + "\n")
	case sessioninventory.AgentMuse:
		return []byte(`{"payload_type":"agent.message","payload":{"kind":"text","run_id":"` + nativeID + `","text":"next","sender":"human"}}` + "\n")
	}
	return nil
}
