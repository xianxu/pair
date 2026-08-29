package sessioninventory

import (
	"testing"
	"time"
)

func TestReconcileCatalog(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	artifact := Artifact{StorageRoot: "claude-projects", RelativePath: "-repo/root.jsonl", Kind: ArtifactTranscript}
	prior := CatalogEntry{
		Agent: AgentClaude, Artifact: artifact,
		Fingerprint:   ArtifactFingerprint{StableFileID: "dev:1:ino:2", GenerationToken: "gen:3", MutationToken: "ctime:4", Size: 100, BirthTime: &baseTime, ModTime: &baseTime},
		Authorization: AuthorizationAuthorized, ScannerSchema: "claude-v1", ProviderContract: ProviderClaudeJSONLV1,
		RawObservedOffset: 100, ParserCompleteOffset: 100,
	}
	changedTime := baseTime.Add(time.Second)

	tests := []struct {
		name         string
		catalog      Catalog
		observations []ArtifactObservation
		wantKind     CatalogWorkKind
		wantReused   int
	}{
		{name: "unchanged", catalog: testCatalog(prior), observations: []ArtifactObservation{catalogObservation(prior)}, wantReused: 1},
		{name: "trusted append", catalog: testCatalog(prior), observations: []ArtifactObservation{{Agent: prior.Agent, Entry: FileEntry{Artifact: artifact, StableFileID: "dev:1:ino:2", GenerationToken: "gen:3", MutationToken: "ctime:5", Size: 120, BirthTime: &baseTime, ModTime: &changedTime}, ScannerSchema: "claude-v1", ProviderContract: ProviderClaudeJSONLV1}}, wantKind: CatalogWorkAppend},
		{name: "new", catalog: Catalog{Version: CatalogVersion, Generation: 1}, observations: []ArtifactObservation{catalogObservation(prior)}, wantKind: CatalogWorkNew},
		{name: "replacement generation", catalog: testCatalog(prior), observations: []ArtifactObservation{{Agent: prior.Agent, Entry: FileEntry{Artifact: artifact, StableFileID: "dev:1:ino:2", GenerationToken: "gen:9", MutationToken: "ctime:5", Size: 120, BirthTime: &changedTime, ModTime: &changedTime}, ScannerSchema: "claude-v1", ProviderContract: ProviderClaudeJSONLV1}}, wantKind: CatalogWorkRevalidate},
		{name: "truncated", catalog: testCatalog(prior), observations: []ArtifactObservation{{Agent: prior.Agent, Entry: FileEntry{Artifact: artifact, StableFileID: "dev:1:ino:2", GenerationToken: "gen:3", MutationToken: "ctime:5", Size: 80, BirthTime: &baseTime, ModTime: &changedTime}, ScannerSchema: "claude-v1", ProviderContract: ProviderClaudeJSONLV1}}, wantKind: CatalogWorkRevalidate},
		{name: "same size mutation", catalog: testCatalog(prior), observations: []ArtifactObservation{{Agent: prior.Agent, Entry: FileEntry{Artifact: artifact, StableFileID: "dev:1:ino:2", GenerationToken: "gen:3", MutationToken: "ctime:5", Size: 100, BirthTime: &baseTime, ModTime: &changedTime}, ScannerSchema: "claude-v1", ProviderContract: ProviderClaudeJSONLV1}}, wantKind: CatalogWorkRevalidate},
		{name: "missing generation append", catalog: testCatalog(withGeneration(prior, "")), observations: []ArtifactObservation{{Agent: prior.Agent, Entry: FileEntry{Artifact: artifact, StableFileID: "dev:1:ino:2", MutationToken: "ctime:5", Size: 120, BirthTime: &baseTime, ModTime: &changedTime}, ScannerSchema: "claude-v1", ProviderContract: ProviderClaudeJSONLV1}}, wantKind: CatalogWorkRevalidate},
		{name: "provider version changed", catalog: testCatalog(prior), observations: []ArtifactObservation{{Agent: prior.Agent, Entry: FileEntry{Artifact: artifact, StableFileID: "dev:1:ino:2", GenerationToken: "gen:3", MutationToken: "ctime:5", Size: 120, BirthTime: &baseTime, ModTime: &changedTime}, ScannerSchema: "claude-v2"}}, wantKind: CatalogWorkRevalidate},
		{name: "deleted", catalog: testCatalog(prior), wantKind: CatalogWorkDelete},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			delta := ReconcileCatalog(test.catalog, test.observations)
			if len(delta.Reused) != test.wantReused {
				t.Fatalf("reused = %d, want %d; delta=%#v", len(delta.Reused), test.wantReused, delta)
			}
			if test.wantKind == "" {
				if len(delta.Work) != 0 {
					t.Fatalf("work = %#v, want none", delta.Work)
				}
				return
			}
			if len(delta.Work) != 1 || delta.Work[0].Kind != test.wantKind {
				t.Fatalf("work = %#v, want one %q", delta.Work, test.wantKind)
			}
		})
	}
}

func testCatalog(entry CatalogEntry) Catalog {
	return Catalog{Version: CatalogVersion, Generation: 1, Entries: []CatalogEntry{entry}}
}

func catalogObservation(entry CatalogEntry) ArtifactObservation {
	return ArtifactObservation{Agent: entry.Agent, Entry: FileEntry{Artifact: entry.Artifact, StableFileID: entry.Fingerprint.StableFileID, GenerationToken: entry.Fingerprint.GenerationToken, MutationToken: entry.Fingerprint.MutationToken, Size: entry.Fingerprint.Size, BirthTime: entry.Fingerprint.BirthTime, ModTime: entry.Fingerprint.ModTime}, ScannerSchema: entry.ScannerSchema, ProviderContract: entry.ProviderContract}
}

func withGeneration(entry CatalogEntry, generation GenerationToken) CatalogEntry {
	entry.Fingerprint.GenerationToken = generation
	return entry
}
