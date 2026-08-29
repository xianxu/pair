package sessioninventory

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCatalogCloneAndValidate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 29, 12, 0, 0, 123, time.UTC)
	catalog := Catalog{
		Version:    CatalogVersion,
		Generation: 4,
		Entries: []CatalogEntry{{
			Agent:                AgentClaude,
			Artifact:             Artifact{StorageRoot: "claude-projects", RelativePath: "-repo/root.jsonl", Kind: ArtifactTranscript},
			Fingerprint:          ArtifactFingerprint{StableFileID: "dev:1:ino:2", GenerationToken: "gen:3", MutationToken: "ctime:4", Size: 8, BirthTime: &now, ModTime: &now},
			Authorization:        AuthorizationAuthorized,
			ScannerSchema:        "claude-v1",
			ProviderContract:     ProviderClaudeJSONLV1,
			RawObservedOffset:    8,
			ParserCompleteOffset: 8,
			ScannerState:         json.RawMessage(`{"native_id":"root"}`),
		}},
	}
	if err := ValidateCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	cloned := CloneCatalog(catalog)
	cloned.Entries[0].ScannerState[0] = '['
	if catalog.Entries[0].ScannerState[0] != '{' {
		t.Fatal("CloneCatalog aliased scanner state")
	}

	catalog.Entries[0].Authorization = AuthorizationUnknown
	if err := ValidateCatalog(catalog); err == nil {
		t.Fatal("zero authorization state accepted")
	}
}

func TestMergeCatalogPublicationNeverRegressesSameArtifact(t *testing.T) {
	t.Parallel()
	base := CatalogEntry{
		Agent:         AgentCodex,
		Artifact:      Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/29/rollout-id.jsonl"},
		Fingerprint:   ArtifactFingerprint{StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:2", Size: 20},
		Authorization: AuthorizationAuthorized, ScannerSchema: "codex-v1", ProviderContract: ProviderCodexJSONLV1,
		RawObservedOffset: 20, ParserCompleteOffset: 20, ScannerState: json.RawMessage(`{"cursor":20}`),
	}
	older := cloneCatalogEntry(base)
	older.Fingerprint.MutationToken = "ctime:1"
	older.Fingerprint.Size = 10
	older.RawObservedOffset = 10
	older.ParserCompleteOffset = 10
	older.ScannerState = json.RawMessage(`{"cursor":10}`)
	if got := MergeCatalogPublication(base, older); got.RawObservedOffset != 20 || string(got.ScannerState) != `{"cursor":20}` {
		t.Fatalf("stale writer regressed entry: %#v", got)
	}
	disputed := cloneCatalogEntry(older)
	disputed.Authorization = AuthorizationDisputed
	if got := MergeCatalogPublication(base, disputed); got.Authorization != AuthorizationDisputed {
		t.Fatalf("dispute was lost: %#v", got)
	}
	if got := MergeCatalogPublication(disputed, base); got.Authorization != AuthorizationDisputed {
		t.Fatalf("stale authorization erased dispute: %#v", got)
	}
	crossed := cloneCatalogEntry(base)
	crossed.Fingerprint.MutationToken = "ctime:3"
	crossed.Fingerprint.Size = 30
	crossed.RawObservedOffset = 30
	crossed.ParserCompleteOffset = 10
	if got := MergeCatalogPublication(base, crossed); got.RawObservedOffset != 20 || got.ParserCompleteOffset != 20 {
		t.Fatalf("crossed cursors regressed parser state: %#v", got)
	}
}
