package sessioninventory_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestAppendOnlyProviderConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		agent    sessioninventory.Agent
		root     string
		relative string
		fixture  string
		schema   string
		validate func(sessioninventory.FileEntry, *sessioninventory.ScannerState, []sessioninventory.FramedJSONLRecord) (sessioninventory.ScannerState, error)
	}{
		{
			name: "claude", agent: sessioninventory.AgentClaude, root: "claude-projects", relative: "-repo/11111111-1111-4111-8111-111111111111.jsonl", schema: "claude-v1",
			fixture: filepath.Join("testdata", "native", "claude", "v1", "claude-projects", "-repo", "11111111-1111-4111-8111-111111111111.jsonl"),
			validate: func(entry sessioninventory.FileEntry, prior *sessioninventory.ScannerState, records []sessioninventory.FramedJSONLRecord) (sessioninventory.ScannerState, error) {
				state, _, err := sessioninventory.ValidateClaudeDelta(entry, prior, records)
				return state, err
			},
		},
		{
			name: "codex", agent: sessioninventory.AgentCodex, root: "codex-sessions", relative: "2026/08/28/rollout-root-019d1111-1111-7111-8111-111111111111.jsonl", schema: "codex-v1",
			fixture: filepath.Join("testdata", "native", "codex", "v1", "codex-sessions", "2026", "08", "28", "rollout-root-019d1111-1111-7111-8111-111111111111.jsonl"),
			validate: func(entry sessioninventory.FileEntry, prior *sessioninventory.ScannerState, records []sessioninventory.FramedJSONLRecord) (sessioninventory.ScannerState, error) {
				state, _, err := sessioninventory.ValidateCodexDelta(entry, prior, records)
				return state, err
			},
		},
		{
			name: "muse", agent: sessioninventory.AgentMuse, root: "muse-sessions", relative: "2026/08/28/77777777-7777-4777-8777-777777777777/session.jsonl", schema: "muse-v1",
			fixture: filepath.Join("testdata", "native", "muse", "v1", "muse-sessions", "2026", "08", "28", "77777777-7777-4777-8777-777777777777", "session.jsonl"),
			validate: func(entry sessioninventory.FileEntry, prior *sessioninventory.ScannerState, records []sessioninventory.FramedJSONLRecord) (sessioninventory.ScannerState, error) {
				state, _, err := sessioninventory.ValidateMuseDelta(entry, prior, records)
				return state, err
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := sessioninventory.ProviderContractFor(test.agent, test.root, test.schema); !ok {
				t.Fatal("fixture has no reviewed provider contract")
			}
			raw, err := os.ReadFile(test.fixture)
			if err != nil {
				t.Fatal(err)
			}
			split := bytes.IndexByte(raw, '\n') + 1
			if split <= 0 || split == len(raw) {
				t.Fatal("fixture does not contain an append boundary")
			}
			assertAppendStateMatchesFull(t, test.agent, test.root, test.relative, raw[:split], raw[split:], test.validate)
		})
	}
	assertAgyAppendStateMatchesFull(t)
}

func assertAppendStateMatchesFull(t *testing.T, agent sessioninventory.Agent, rootName, relative string, prefix, suffix []byte, validate func(sessioninventory.FileEntry, *sessioninventory.ScannerState, []sessioninventory.FramedJSONLRecord) (sessioninventory.ScannerState, error)) {
	t.Helper()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: agent, Name: rootName}
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: rootName, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}, StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1"}
	runtime.AddRoot(root)
	runtime.PutFile(entry, prefix)
	initial := onlyFile(t, runtime, root)
	firstObservation, err := sessioninventory.ObserveStableArtifact(runtime, root, initial, sessioninventory.JSONLFrameState{}, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	state, err := validate(initial, nil, firstObservation.Records)
	if err != nil {
		t.Fatal(err)
	}
	runtime.AppendFile(entry.Artifact, suffix, "ctime:2")
	appended := onlyFile(t, runtime, root)
	secondObservation, err := sessioninventory.ObserveStableArtifact(runtime, root, appended, firstObservation.FrameState, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	incremental, err := validate(appended, &state, secondObservation.Records)
	if err != nil {
		t.Fatal(err)
	}
	allRecords, _, err := sessioninventory.FrameJSONLSuffix(sessioninventory.JSONLFrameState{}, append(append([]byte(nil), prefix...), suffix...), 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	full, err := validate(appended, nil, allRecords)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(incremental, full) {
		t.Fatalf("incremental=%#v\nfull=%#v", incremental, full)
	}
}

func assertAgyAppendStateMatchesFull(t *testing.T) {
	t.Helper()
	nativeID := "55555555-5555-4555-8555-555555555555"
	runtime, database, transcript := incrementalAgyFixture(nativeID)
	raw, err := os.ReadFile(filepath.Join("testdata", "native", "agy", "v1", "agy-brain", nativeID, ".system_generated", "logs", "transcript.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	split := bytes.IndexByte(raw, '\n') + 1
	first, frame, err := sessioninventory.FrameJSONLSuffix(sessioninventory.JSONLFrameState{}, raw[:split], 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err := sessioninventory.ValidateAgyDelta(runtime, database, transcript, nil, first)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := sessioninventory.FrameJSONLSuffix(frame, raw[split:], 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	incremental, _, err := sessioninventory.ValidateAgyDelta(runtime, database, transcript, &state, second)
	if err != nil {
		t.Fatal(err)
	}
	all, _, err := sessioninventory.FrameJSONLSuffix(sessioninventory.JSONLFrameState{}, raw, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	full, _, err := sessioninventory.ValidateAgyDelta(runtime, database, transcript, nil, all)
	if err != nil || !reflect.DeepEqual(incremental, full) {
		t.Fatalf("incremental=%#v full=%#v err=%v", incremental, full, err)
	}
}

func onlyFile(t *testing.T, runtime *sessioninventorytest.FakeRuntime, root sessioninventory.StorageRoot) sessioninventory.FileEntry {
	t.Helper()
	files, err := runtime.ListFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("files=%#v err=%v", files, err)
	}
	return files[0]
}
