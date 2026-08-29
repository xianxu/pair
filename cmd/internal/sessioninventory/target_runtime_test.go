package sessioninventory_test

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestTargetExplicitResumeReadsOnlyNamedArtifact(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(root)
	wantedID := "019d1111-1111-7111-8111-111111111111"
	otherID := "019d2222-2222-7222-8222-222222222222"
	put := func(id string) sessioninventory.Artifact {
		artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/28/rollout-test-" + id + ".jsonl"}
		runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: sessioninventory.StableFileID("stable-" + id), GenerationToken: sessioninventory.GenerationToken("gen-" + id), MutationToken: sessioninventory.MutationToken("mutation-" + id)}, []byte(
			`{"type":"session_meta","payload":{"id":"`+id+`","parent_thread_id":null,"source":"cli"}}`+"\n"))
		return artifact
	}
	wanted := put(wantedID)
	other := put(otherID)

	if !sessioninventory.NativeSessionCandidateExists(runtime, sessioninventory.AgentCodex, wantedID) {
		t.Fatal("named root was not found")
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadAt, wanted.StorageRoot+":"+wanted.RelativePath); got == 0 {
		t.Fatal("named artifact was not validated")
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadAt, other.StorageRoot+":"+other.RelativePath); got != 0 {
		t.Fatalf("unrelated artifact ReadAt count=%d, want 0", got)
	}
}

func TestTargetCatalogReuseReadsNoArtifactBody(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(root)
	id := "019d1111-1111-7111-8111-111111111111"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/28/rollout-test-" + id + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
	entry := sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen", MutationToken: "mutation"}
	runtime.PutFile(entry, []byte("previously validated bytes\n"))
	observations, _ := sessioninventory.ObserveAgentMetadata(runtime, sessioninventory.AgentCodex)
	fingerprint := sessioninventory.ArtifactFingerprint{StableFileID: observations[0].Entry.StableFileID, GenerationToken: observations[0].Entry.GenerationToken, MutationToken: observations[0].Entry.MutationToken, Size: observations[0].Entry.Size}
	fact := sessioninventory.Fact{Agent: sessioninventory.AgentCodex, NativeID: id, Role: sessioninventory.RoleRoot, Resumable: true, Artifacts: []sessioninventory.Artifact{artifact}}
	catalog := sessioninventory.Catalog{Version: sessioninventory.CatalogVersion, Entries: []sessioninventory.CatalogEntry{{Agent: sessioninventory.AgentCodex, Artifact: artifact, Fingerprint: fingerprint, Authorization: sessioninventory.AuthorizationAuthorized, Facts: []sessioninventory.Fact{fact}}}}

	if !sessioninventory.CatalogSessionCandidateExists(catalog, observations, sessioninventory.AgentCodex, id) {
		t.Fatal("authorized unchanged catalog root was not reused")
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadAt, ""); got != 0 {
		t.Fatalf("ReadAt count=%d, want 0", got)
	}
}
