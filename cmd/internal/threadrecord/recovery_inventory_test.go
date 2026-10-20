package threadrecord

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestRecoveryCandidateInventoryEntryPreservesExactFingerprint(t *testing.T) {
	at := time.Date(2026, 10, 20, 12, 30, 45, 123, time.UTC)
	candidate := RecoveryCandidate{
		ArtifactRoot: "codex-sessions", ArtifactRelativePath: "2026/10/rollout.jsonl",
		StableFileID: "dev:ino", GenerationToken: "generation", MutationToken: "mutation", Size: 1234, ModTime: at,
	}
	want := sessioninventory.FileEntry{
		Artifact:     sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/10/rollout.jsonl"},
		StableFileID: "dev:ino", GenerationToken: "generation", MutationToken: "mutation", Size: 1234, ModTime: &at,
	}
	got := candidate.InventoryEntry()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fingerprint changed: got %#v, want %#v", got, want)
	}
	*got.ModTime = at.Add(time.Second)
	if !candidate.ModTime.Equal(at) {
		t.Fatal("inventory entry aliases persisted candidate time")
	}
}
