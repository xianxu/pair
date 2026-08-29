package sessioninventory_test

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestObserveStableArtifactConsumesGrowthThroughStableEOF(t *testing.T) {
	t.Parallel()
	base, root, entry := incrementalArtifactFixture(t, []byte("one\n"))
	runtime := &mutatingReadRuntime{FakeRuntime: base, mutate: func() {
		base.AppendFile(entry.Artifact, []byte("two\n"), "ctime:2")
	}}
	result, err := sessioninventory.ObserveStableArtifact(runtime, root, entry, sessioninventory.JSONLFrameState{}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 2 || string(result.Records[0].Bytes) != "one" || string(result.Records[1].Bytes) != "two" {
		t.Fatalf("records=%#v", result.Records)
	}
	if result.RawObservedOffset != 8 || result.FrameState.ParserCompleteOffset != 8 || result.Fingerprint.MutationToken != "ctime:2" || result.Disputed {
		t.Fatalf("result=%#v", result)
	}
}

func TestObserveStableArtifactDisputesReplacementOrTruncation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*sessioninventorytest.FakeRuntime, sessioninventory.FileEntry)
	}{
		{name: "replacement", mutate: func(runtime *sessioninventorytest.FakeRuntime, entry sessioninventory.FileEntry) {
			runtime.ReplaceFile(entry, []byte("other\n"), "gen:2", "ctime:2")
		}},
		{name: "truncation", mutate: func(runtime *sessioninventorytest.FakeRuntime, entry sessioninventory.FileEntry) {
			runtime.TruncateFile(entry.Artifact, 0, "ctime:2")
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			base, root, entry := incrementalArtifactFixture(t, []byte("one\n"))
			runtime := &mutatingReadRuntime{FakeRuntime: base, mutate: func() { test.mutate(base, entry) }}
			result, err := sessioninventory.ObserveStableArtifact(runtime, root, entry, sessioninventory.JSONLFrameState{}, 1024)
			if !errors.Is(err, sessioninventory.ErrArtifactChanged) || !result.Disputed || len(result.Records) != 0 {
				t.Fatalf("result=%#v err=%v", result, err)
			}
		})
	}
}

func TestObserveStableArtifactDisputesSameSizeMutation(t *testing.T) {
	t.Parallel()
	base, root, entry := incrementalArtifactFixture(t, []byte("one\n"))
	runtime := &mutatingReadRuntime{FakeRuntime: base, mutate: func() {
		base.ReplaceFile(entry, []byte("two\n"), "gen:1", "ctime:2")
	}}
	result, err := sessioninventory.ObserveStableArtifact(runtime, root, entry, sessioninventory.JSONLFrameState{}, 1024)
	if !errors.Is(err, sessioninventory.ErrArtifactChanged) || !result.Disputed {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestIncrementalInventoryReconcilesMetadataBeforeSelectingWork(t *testing.T) {
	t.Parallel()
	runtime, _, entry := incrementalArtifactFixture(t, []byte("one\n"))
	catalog := sessioninventory.Catalog{Version: sessioninventory.CatalogVersion, Generation: 7, Entries: []sessioninventory.CatalogEntry{{
		Agent: sessioninventory.AgentClaude, Artifact: entry.Artifact,
		Fingerprint:   sessioninventory.ArtifactFingerprint{StableFileID: entry.StableFileID, GenerationToken: entry.GenerationToken, MutationToken: entry.MutationToken, Size: entry.Size, BirthTime: entry.BirthTime, ModTime: entry.ModTime},
		Authorization: sessioninventory.AuthorizationAuthorized, ScannerSchema: "claude-v1", ProviderContract: sessioninventory.ProviderClaudeJSONLV1,
		RawObservedOffset: entry.Size, ParserCompleteOffset: entry.Size,
	}}}
	inventory := sessioninventory.NewIncrementalInventory(runtime, catalog)
	snapshot := inventory.Observe(sessioninventory.AgentClaude)
	if snapshot.Delta.BaseGeneration != 7 || len(snapshot.Delta.Reused) != 1 || len(snapshot.Delta.Work) != 0 {
		t.Fatalf("delta=%#v", snapshot.Delta)
	}
	selected := inventory.Select(sessioninventory.TargetRequest{Mode: sessioninventory.TargetDiagnostic, Agent: sessioninventory.AgentClaude}, snapshot)
	if len(selected.Eligible) != 1 {
		t.Fatalf("selected=%#v", selected)
	}
	selected = inventory.Select(sessioninventory.TargetRequest{Mode: sessioninventory.TargetNewLaunch, Agent: sessioninventory.AgentClaude}, snapshot)
	if len(selected.Eligible) != 0 {
		t.Fatalf("catalog-reused artifact became fresh launch work: %#v", selected)
	}
}

func TestObserveLaunchBoundarySuffixFramesOnlyWholePostLaunchRecords(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		prefix string
		suffix string
		want   []string
	}{
		{name: "newline", prefix: "old\n", suffix: "new\n", want: []string{"new"}},
		{name: "incomplete", prefix: "partial", suffix: "-old\nnew\n", want: []string{"new"}},
		{name: "empty", prefix: "", suffix: "new\n", want: []string{"new"}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runtime, root, boundaryEntry := incrementalArtifactFixture(t, []byte(test.prefix))
			boundary := sessioninventory.TargetArtifactBoundary{StorageRoot: boundaryEntry.Artifact.StorageRoot, RelativePath: boundaryEntry.Artifact.RelativePath, StableFileID: boundaryEntry.StableFileID, GenerationToken: boundaryEntry.GenerationToken, MutationToken: boundaryEntry.MutationToken, RawSize: boundaryEntry.Size}
			runtime.AppendFile(boundaryEntry.Artifact, []byte(test.suffix), "ctime:2")
			files, err := runtime.ListFiles(root)
			if err != nil || len(files) != 1 {
				t.Fatalf("metadata=%#v err=%v", files, err)
			}
			observation := sessioninventory.ArtifactObservation{Agent: sessioninventory.AgentClaude, Entry: files[0], ScannerSchema: "claude-v1", ProviderContract: sessioninventory.ProviderClaudeJSONLV1}
			result, err := sessioninventory.ObserveLaunchBoundarySuffix(runtime, boundary, observation)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, record := range result.Records {
				got = append(got, string(record.Bytes))
			}
			if !slices.Equal(got, test.want) {
				t.Fatalf("records=%q want=%q", got, test.want)
			}
		})
	}
}

func TestObserveLaunchBoundarySuffixRejectsTruncationAndReplacement(t *testing.T) {
	t.Parallel()
	for _, replacement := range []bool{false, true} {
		runtime, root, entry := incrementalArtifactFixture(t, []byte("old\n"))
		boundary := sessioninventory.TargetArtifactBoundary{StorageRoot: entry.Artifact.StorageRoot, RelativePath: entry.Artifact.RelativePath, StableFileID: entry.StableFileID, GenerationToken: entry.GenerationToken, MutationToken: entry.MutationToken, RawSize: entry.Size}
		if replacement {
			runtime.ReplaceFile(entry, []byte("new\n"), "gen:2", "ctime:2")
		} else {
			runtime.TruncateFile(entry.Artifact, 1, "ctime:2")
		}
		files, _ := runtime.ListFiles(root)
		observation := sessioninventory.ArtifactObservation{Agent: sessioninventory.AgentClaude, Entry: files[0], ScannerSchema: "claude-v1", ProviderContract: sessioninventory.ProviderClaudeJSONLV1}
		if _, err := sessioninventory.ObserveLaunchBoundarySuffix(runtime, boundary, observation); !errors.Is(err, sessioninventory.ErrArtifactChanged) {
			t.Fatalf("replacement=%v err=%v", replacement, err)
		}
	}
}

type mutatingReadRuntime struct {
	*sessioninventorytest.FakeRuntime
	once   sync.Once
	mutate func()
}

func (r *mutatingReadRuntime) ReadAt(artifact sessioninventory.Artifact, offset, limit int64) ([]byte, bool, error) {
	raw, eof, err := r.FakeRuntime.ReadAt(artifact, offset, limit)
	r.once.Do(r.mutate)
	return raw, eof, err
}

func incrementalArtifactFixture(t *testing.T, content []byte) (*sessioninventorytest.FakeRuntime, sessioninventory.StorageRoot, sessioninventory.FileEntry) {
	t.Helper()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/native/claude"}
	entry := sessioninventory.FileEntry{
		Artifact:     sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "project/019eff64-6ceb-7e72-9d41-a735a97029ac.jsonl", Kind: sessioninventory.ArtifactTranscript},
		StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1",
	}
	runtime.AddRoot(root)
	runtime.PutFile(entry, content)
	files, err := runtime.ListFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("fixture metadata=%#v, %v", files, err)
	}
	return runtime, root, files[0]
}
