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
