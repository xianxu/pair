package sessioninventory_test

import (
	"errors"
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
