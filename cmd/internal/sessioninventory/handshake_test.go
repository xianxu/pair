package sessioninventory

import "testing"

func TestChosenFilenameHandshakeRequiresNewUniqueRootPath(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	candidate := ArtifactObservation{Agent: AgentClaude, Entry: FileEntry{Artifact: Artifact{StorageRoot: "claude-projects", RelativePath: "-repo/" + id + ".jsonl"}}}
	if got := ChosenFilenameHandshake(AgentClaude, id, nil, []ArtifactObservation{candidate}); got == nil {
		t.Fatal("new chosen file not acknowledged")
	}
	baseline := []TargetArtifactBoundary{{StorageRoot: "claude-projects", RelativePath: candidate.Entry.Artifact.RelativePath}}
	if got := ChosenFilenameHandshake(AgentClaude, id, baseline, []ArtifactObservation{candidate}); got != nil {
		t.Fatal("old chosen file acknowledged")
	}
}
