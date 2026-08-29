package sessioninventory

import "testing"

func TestColdAuthorizationMatrixSelectsOnlyPostBoundaryArtifacts(t *testing.T) {
	t.Parallel()
	for _, agent := range []Agent{AgentClaude, AgentCodex, AgentMuse} {
		agent := agent
		t.Run(string(agent), func(t *testing.T) {
			old, fresh := targetObservation(agent, "old"), targetObservation(agent, "fresh")
			result := SelectTargetWork(TargetRequest{Mode: TargetNewLaunch, Agent: agent, Baseline: []TargetArtifactBoundary{{StorageRoot: old.Entry.Artifact.StorageRoot, RelativePath: old.Entry.Artifact.RelativePath}}}, []ArtifactObservation{old, fresh})
			if result.Unavailable || len(result.Eligible) != 1 || result.Eligible[0].Entry.Artifact.RelativePath != fresh.Entry.Artifact.RelativePath {
				t.Fatalf("result=%#v", result)
			}
		})
	}
}

func TestColdAuthorizationMatrixAgyRequiresNewJoinedPair(t *testing.T) {
	t.Parallel()
	id := "55555555-5555-4555-8555-555555555555"
	database := ArtifactObservation{Agent: AgentAgy, Entry: FileEntry{Artifact: Artifact{StorageRoot: "agy-conversations", RelativePath: id + ".db"}}}
	transcript := ArtifactObservation{Agent: AgentAgy, Entry: FileEntry{Artifact: Artifact{StorageRoot: "agy-brain", RelativePath: id + "/.system_generated/logs/transcript.jsonl"}}}
	for _, test := range []struct {
		name     string
		baseline []TargetArtifactBoundary
		want     int
	}{
		{name: "both new", want: 2},
		{name: "database old", baseline: []TargetArtifactBoundary{{StorageRoot: database.Entry.Artifact.StorageRoot, RelativePath: database.Entry.Artifact.RelativePath}}},
		{name: "transcript old", baseline: []TargetArtifactBoundary{{StorageRoot: transcript.Entry.Artifact.StorageRoot, RelativePath: transcript.Entry.Artifact.RelativePath}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := SelectTargetWork(TargetRequest{Mode: TargetNewLaunch, Agent: AgentAgy, Baseline: test.baseline}, []ArtifactObservation{database, transcript})
			if len(result.Eligible) != test.want {
				t.Fatalf("eligible=%#v", result.Eligible)
			}
		})
	}
}

func TestTargetWorkNeverWidensProoflessOrNamedRequests(t *testing.T) {
	t.Parallel()
	first, second := targetObservation(AgentClaude, "first"), targetObservation(AgentClaude, "second")
	artifact := first.Entry.Artifact
	nativeID, _, _, _ := claudePathFact(artifact.RelativePath)
	for _, test := range []struct {
		name        string
		request     TargetRequest
		want        int
		unavailable bool
	}{
		{name: "established proof", request: TargetRequest{Mode: TargetEstablished, Agent: AgentClaude, NativeID: nativeID, AuthorizedArtifacts: []Artifact{artifact}}, want: 1},
		{name: "established proofless", request: TargetRequest{Mode: TargetEstablished, Agent: AgentClaude, NativeID: nativeID}, unavailable: true},
		{name: "activity proofless", request: TargetRequest{Mode: TargetActivity, Agent: AgentClaude, NativeID: nativeID}, unavailable: true},
		{name: "explicit named", request: TargetRequest{Mode: TargetExplicitResume, Agent: AgentClaude, NativeID: nativeID}, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := SelectTargetWork(test.request, []ArtifactObservation{first, second})
			if len(result.Eligible) != test.want || result.Unavailable != test.unavailable {
				t.Fatalf("result=%#v", result)
			}
		})
	}
}

func targetObservation(agent Agent, id string) ArtifactObservation {
	artifact := Artifact{Kind: ArtifactTranscript}
	switch agent {
	case AgentClaude:
		artifact.StorageRoot = "claude-projects"
		artifact.RelativePath = "-repo/11111111-1111-4111-8111-111111111111.jsonl"
		if id == "fresh" || id == "second" {
			artifact.RelativePath = "-repo/22222222-2222-4222-8222-222222222222.jsonl"
		}
	case AgentCodex:
		artifact.StorageRoot = "codex-sessions"
		artifact.RelativePath = "2026/08/28/rollout-" + id + "-019d1111-1111-7111-8111-111111111111.jsonl"
		if id == "fresh" || id == "second" {
			artifact.RelativePath = "2026/08/28/rollout-" + id + "-019d2222-2222-7222-8222-222222222222.jsonl"
		}
	case AgentMuse:
		artifact.StorageRoot = "muse-sessions"
		artifact.RelativePath = "2026/08/28/77777777-7777-4777-8777-777777777777/session.jsonl"
		if id == "fresh" || id == "second" {
			artifact.RelativePath = "2026/08/28/88888888-8888-4888-8888-888888888888/session.jsonl"
		}
	}
	return ArtifactObservation{Agent: agent, Entry: FileEntry{Artifact: artifact}}
}
