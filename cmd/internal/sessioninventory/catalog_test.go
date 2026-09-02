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

// GenerationID answers "WHICH INCARNATION of this artifact path" -- the
// question ArtifactGeneration exists to answer in a lifecycle record.
//
// It must never be empty for a populated fingerprint, because a consumer that
// treats it as required identity would otherwise refuse every artifact on every
// platform pair supports. GenerationToken is the ideal source and is
// unavailable everywhere: Linux never populates it, the portable fallback never
// populates it, and darwin populates it only from st_gen, which APFS reports as
// 0 for unprivileged callers.
func TestArtifactFingerprintGenerationID(t *testing.T) {
	birth := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	other := time.Date(2026, 8, 29, 1, 0, 0, 0, time.UTC)

	// Only emptiness is a contract; the exact spelling is not, so it is not
	// asserted. What IS asserted is the incarnation property, below.
	for _, test := range []struct {
		name        string
		fingerprint ArtifactFingerprint
		wantEmpty   bool
	}{
		{"generation token present", ArtifactFingerprint{StableFileID: "dev:1:ino:2", GenerationToken: "gen:7", BirthTime: &birth}, false},
		{"no generation token, birth time present", ArtifactFingerprint{StableFileID: "dev:1:ino:2", BirthTime: &birth}, false},
		{"no generation token, no birth time", ArtifactFingerprint{StableFileID: "dev:1:ino:2"}, false},
		{"no identity at all", ArtifactFingerprint{}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.fingerprint.GenerationID()
			if (got == "") != test.wantEmpty {
				t.Fatalf("GenerationID() = %q, wantEmpty=%v", got, test.wantEmpty)
			}
		})
	}

	// A filesystem-provided generation token still wins, because it also
	// distinguishes inode REUSE, which dev:ino alone cannot.
	withToken := ArtifactFingerprint{StableFileID: "dev:1:ino:2", GenerationToken: "gen:7", BirthTime: &birth}
	if withToken.GenerationID() != "gen:7" {
		t.Fatalf("GenerationID() = %q, want the filesystem token to win", withToken.GenerationID())
	}

	// The property that matters: a replaced artifact must not share an
	// incarnation identity with the one it replaced.
	base := ArtifactFingerprint{StableFileID: "dev:1:ino:2", BirthTime: &birth}
	replaced := ArtifactFingerprint{StableFileID: "dev:1:ino:9", BirthTime: &birth}
	recreated := ArtifactFingerprint{StableFileID: "dev:1:ino:2", BirthTime: &other}
	if base.GenerationID() == replaced.GenerationID() {
		t.Fatal("a different inode shares an incarnation identity")
	}
	if base.GenerationID() == recreated.GenerationID() {
		t.Fatal("a recreated file shares an incarnation identity with its predecessor")
	}
	if base.GenerationID() != (ArtifactFingerprint{StableFileID: "dev:1:ino:2", BirthTime: &birth}).GenerationID() {
		t.Fatal("the same incarnation produced two identities")
	}
}
