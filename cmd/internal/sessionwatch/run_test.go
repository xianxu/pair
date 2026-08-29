package sessionwatch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestRunEstablishesOnlyAfterCompletedCorroboratedRound(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
	native.AddRoot(nativeRoot)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
	artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
	text := "please inspect the durable watcher boundary now"
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, text))
	native.SetProcess("1234", "native-identity", nil, []string{filepath.Join(nativeRoot.Path, filepath.FromSlash(relative))})

	paths := mustScopedPaths(t, dataDir, "work")
	launch := mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = launch
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"

	err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: 20 * time.Millisecond, Poll: time.Millisecond}, runtime)
	if err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].RootNativeID != sid {
		t.Fatalf("records=%#v", runtime.store.records)
	}
	if got := string(runtime.writes[paths.Config("codex")]); !strings.Contains(got, sid) {
		t.Fatalf("config=%s", got)
	}
	if !runtime.hasLog(adapt.Fired) {
		t.Fatalf("logs=%#v", runtime.logs)
	}
}

func TestWatcherIncrementalV2PublishesProofFromOnlyPostBoundaryArtifact(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
	native.AddRoot(nativeRoot)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
	artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
	text := "please inspect the durable watcher boundary now"
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, text))

	paths := mustScopedPaths(t, dataDir, "work")
	if err := os.WriteFile(paths.Catalog(), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: 20 * time.Millisecond, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].Version != 2 || runtime.store.records[0].AuthorizationProof == nil || runtime.store.records[0].RootNativeID != sid {
		t.Fatalf("records=%#v", runtime.store.records)
	}
	catalog, err := (sessioninventory.CatalogStore{Runtime: sessioninventory.CatalogOSRuntime{}}).Read(paths.Catalog())
	if err != nil || len(catalog.Entries) != 1 || catalog.Entries[0].ParserCompleteOffset != int64(len(codexRound(sid, text))) {
		t.Fatalf("catalog=%#v err=%v", catalog, err)
	}
	if got := native.OperationCount(sessioninventorytest.OperationReadFile, ""); got != 0 {
		t.Fatalf("whole-file reads=%d", got)
	}
}

func TestWatcherLaunchBaselineWinsWhenConcurrentCatalogAlreadyContainsNewArtifact(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"}
	native.AddRoot(root)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	text := "launch boundary owns per-launch newness"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/29/rollout-test-" + sid + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, text))
	files, err := native.ListFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("files=%#v err=%v", files, err)
	}
	paths := mustScopedPaths(t, dataDir, "work")
	store := sessioninventory.CatalogStore{Runtime: sessioninventory.CatalogOSRuntime{}}
	_, err = store.Update(paths.Catalog(), func(catalog sessioninventory.Catalog) (sessioninventory.Catalog, error) {
		entry := files[0]
		catalog.Entries = []sessioninventory.CatalogEntry{{Agent: sessioninventory.AgentCodex, Artifact: entry.Artifact, Fingerprint: sessioninventory.ArtifactFingerprint{StableFileID: entry.StableFileID, GenerationToken: entry.GenerationToken, MutationToken: entry.MutationToken, Size: entry.Size}, Authorization: sessioninventory.AuthorizationAuthorized, ScannerSchema: "codex-v1", ProviderContract: sessioninventory.ProviderCodexJSONLV1, RawObservedOffset: entry.Size, ParserCompleteOffset: entry.Size}}
		return catalog, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.Log()] = []byte("## 2026-08-29 01:00:01\n\n" + text + "\n\n---\n\n")
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Nanosecond, Timeout: 10 * time.Millisecond, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].AuthorizationProof == nil {
		t.Fatalf("records=%#v", runtime.store.records)
	}
}

func TestWatcherMigratesOnlyProoflessBoundRootAndPublishesProof(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"}
	native.AddRoot(root)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/29/rollout-test-" + sid + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, "legacy owner"))
	sibling := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/29/rollout-other-123e4567-e89b-12d3-a456-426614174000.jsonl", Kind: sessioninventory.ArtifactTranscript}
	native.PutFile(sessioninventory.FileEntry{Artifact: sibling, StableFileID: "sibling", GenerationToken: "gen:2", MutationToken: "ctime:2"}, []byte("malformed sibling must remain unread\n"))
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	launch := mustLaunchRecord(t, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex"})
	binding := mustLaunchRecord(t, sessionledger.Record{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 1, RootNativeID: sid})
	runtime.files[paths.Ledger()] = append(launch, binding...)
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Nanosecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].AuthorizationProof == nil || runtime.store.records[0].RootNativeID != sid {
		t.Fatalf("migration records=%#v", runtime.store.records)
	}
	if got := native.OperationCount(sessioninventorytest.OperationReadAt, artifact.StorageRoot+":"+artifact.RelativePath); got == 0 {
		t.Fatal("named root was not validated")
	}
	if got := native.OperationCount(sessioninventorytest.OperationReadAt, sibling.StorageRoot+":"+sibling.RelativePath); got != 0 {
		t.Fatalf("proof migration read sibling %d times", got)
	}
}

func TestWatcherV1UnboundLaunchFailsClosedWithoutCorpusScan(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	native.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"})
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex"})
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Nanosecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if got := native.OperationCount(sessioninventorytest.OperationListFiles, ""); got != 0 {
		t.Fatalf("legacy watcher listed native corpus %d times", got)
	}
}

func TestPersistTrackedCatalogDoesNotRegressNewestOrDisputedEntry(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catalog.json")
	store := sessioninventory.CatalogStore{Runtime: sessioninventory.CatalogOSRuntime{}}
	validation := func(size int64, mutation string) map[string]sessioninventory.TargetValidation {
		artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/29/rollout-id.jsonl", Kind: sessioninventory.ArtifactTranscript}
		entry := sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: sessioninventory.MutationToken(mutation), Size: size}
		state := sessioninventory.ScannerState{Version: sessioninventory.ScannerStateVersion, Agent: sessioninventory.AgentCodex, NativeID: "id", IdentityAnchor: "id", Role: sessioninventory.RoleRoot, ScannerSchema: "codex-v1", FirstRecordValidated: true}
		fact, err := sessioninventory.ScannerStateFact(state, []sessioninventory.Artifact{artifact})
		if err != nil {
			t.Fatal(err)
		}
		return map[string]sessioninventory.TargetValidation{"id": {State: state, Fact: fact, Observations: []sessioninventory.ArtifactObservation{{Agent: sessioninventory.AgentCodex, Entry: entry, ScannerSchema: "codex-v1", ProviderContract: sessioninventory.ProviderCodexJSONLV1}}, Results: map[string]sessioninventory.IncrementalResult{"codex-sessions\x00" + artifact.RelativePath: {Fingerprint: sessioninventory.ArtifactFingerprint{StableFileID: "stable", GenerationToken: "gen:1", MutationToken: sessioninventory.MutationToken(mutation), Size: size}, RawObservedOffset: size, FrameState: sessioninventory.JSONLFrameState{ParserCompleteOffset: size}}}}}
	}
	if err := persistTrackedCatalog(store, path, validation(20, "ctime:2")); err != nil {
		t.Fatal(err)
	}
	if err := persistTrackedCatalog(store, path, validation(10, "ctime:1")); err != nil {
		t.Fatal(err)
	}
	catalog, err := store.Read(path)
	if err != nil || len(catalog.Entries) != 1 || catalog.Entries[0].RawObservedOffset != 20 {
		t.Fatalf("catalog=%#v err=%v", catalog, err)
	}
	_, err = store.Update(path, func(current sessioninventory.Catalog) (sessioninventory.Catalog, error) {
		current.Entries[0].Authorization = sessioninventory.AuthorizationDisputed
		current.Entries[0].ScannerState = json.RawMessage(`{"disputed":true}`)
		return current, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := persistTrackedCatalog(store, path, validation(30, "ctime:3")); err != nil {
		t.Fatal(err)
	}
	catalog, err = store.Read(path)
	if err != nil || catalog.Entries[0].Authorization != sessioninventory.AuthorizationDisputed {
		t.Fatalf("catalog=%#v err=%v", catalog, err)
	}
}

func TestWatcherCatalogFailureNeverPublishesProoflessV2Binding(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"}
	native.AddRoot(root)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	text := "catalog must commit before binding authority"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/29/rollout-test-" + sid + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, text))
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.catalogRuntime = failingCatalogRuntime{}
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.Log()] = []byte("## 2026-08-29 01:00:01\n\n" + text + "\n\n---\n\n")
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Nanosecond, Timeout: time.Nanosecond, Poll: time.Nanosecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 0 {
		t.Fatalf("proofless binding published after catalog failure: %#v", runtime.store.records)
	}
}

func TestWatcherLaunchBoundaryNeverReadsPreexistingArtifact(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"}
	native.AddRoot(root)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, "old transcript text that must remain unread"))
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{{StorageRoot: root.Name, RelativePath: relative, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1", RawSize: int64(len(codexRound(sid, "old transcript text that must remain unread")))}}})
	runtime.onSleep = func() { runtime.identities["1234"] = "changed" }
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: 20 * time.Millisecond, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if got := native.OperationCount(sessioninventorytest.OperationReadAt, ""); got != 0 || len(runtime.store.records) != 0 {
		t.Fatalf("range reads=%d records=%#v", got, runtime.store.records)
	}
}

func TestWatcherIncrementalReadsOnlyAppendedProgressOnSecondPoll(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"}
	native.AddRoot(root)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
	text := "please inspect the durable watcher boundary now"
	prefix := []byte(`{"timestamp":"2026-08-28T01:00:00Z","type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + text + `"}]}}` + "\n")
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, prefix)
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.onSleep = func() {
		native.AppendFile(artifact, []byte(`{"type":"response_item","payload":{"type":"function_call"}}`+"\n"), "ctime:2")
		runtime.onSleep = nil
	}
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: time.Second, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].AuthorizationProof == nil {
		t.Fatalf("records=%#v", runtime.store.records)
	}
	if got := native.OperationCount(sessioninventorytest.OperationReadAt, artifact.StorageRoot+":"+artifact.RelativePath); got != 2 {
		t.Fatalf("range reads=%d, want one initial and one suffix read", got)
	}
}

func TestRunIgnoresUnrelatedOpenFilesWhenRoundIsUnique(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
	native.AddRoot(nativeRoot)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
	text := "please inspect the durable watcher boundary now"
	native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, text))
	native.SetProcess("1234", "native-identity", nil, []string{"/tmp/unrelated.txt"})

	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"

	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: 20 * time.Millisecond, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].RootNativeID != sid {
		t.Fatalf("records=%#v", runtime.store.records)
	}
}

func TestRunEstablishesUniqueRoundWhenProcessIdentityIsUnavailable(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
	native.AddRoot(nativeRoot)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
	text := "please inspect the durable watcher boundary now"
	native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, text))

	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now

	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: 20 * time.Millisecond, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].RootNativeID != sid {
		t.Fatalf("records=%#v", runtime.store.records)
	}
}

func TestReadPairLogDiagnosesReadFailure(t *testing.T) {
	t.Parallel()
	runtime := newWatcherRuntime(sessioninventorytest.NewFakeRuntime())
	runtime.readErrors = map[string]error{"/pair/log.md": errors.New("disk unavailable")}
	raw, diagnostics := readPairLog(runtime, "/pair/log.md", sessioninventory.AgentCodex)
	if raw != nil || len(diagnostics) != 1 || diagnostics[0].Code != sessioninventory.DiagnosticStorageUnreadable {
		t.Fatalf("raw=%q diagnostics=%#v", raw, diagnostics)
	}
}

func TestRunDoesNotUseChronologyToResolveRepeatedRound(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
	native.AddRoot(root)
	text := "please inspect the durable watcher boundary now"
	for index, sid := range []string{"019eff64-6ceb-7e72-9d41-a735a97029ac", "123e4567-e89b-12d3-a456-426614174000"} {
		relative := "2026/08/28/rollout-" + string(rune('a'+index)) + "-" + sid + ".jsonl"
		native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}}, codexRound(sid, text))
	}
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.onSleep = func() { runtime.identities["1234"] = "changed" }
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"

	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: time.Second, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 0 || len(runtime.writes) != 0 {
		t.Fatalf("records=%#v writes=%#v", runtime.store.records, runtime.writes)
	}
}

func TestRunRejectsSupersededLaunch(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	paths := mustScopedPaths(t, dataDir, "work")
	first := strings.TrimSpace(string(mustLaunchRecord(t, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex"})))
	second := first
	runtime := newWatcherRuntime(sessioninventorytest.NewFakeRuntime())
	runtime.files[paths.Ledger()] = []byte(first + "\n" + second + "\n")
	runtime.now = time.Unix(10, 0)
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Nanosecond, Timeout: time.Nanosecond, Poll: time.Nanosecond}, runtime); !errors.Is(err, sessionledger.ErrStaleLaunch) {
		t.Fatalf("err=%v", err)
	}
}

func TestPIDFileFreshUsesExactNativeBoundAndLegacySecondTolerance(t *testing.T) {
	bound := time.Unix(100, 500)
	if !pidFileFresh(bound, bound, time.Time{}) || pidFileFresh(time.Unix(100, 499), bound, time.Time{}) || !pidFileFresh(time.Unix(100, 0), time.Time{}, time.Unix(100, 900)) {
		t.Fatal("pid file freshness boundary changed")
	}
}

type watcherRuntime struct {
	now            time.Time
	files          map[string][]byte
	readErrors     map[string]error
	modTimes       map[string]time.Time
	identities     map[string]string
	writes         map[string][]byte
	logs           []adapt.Outcome
	native         sessioninventory.Runtime
	store          *fakeLifecycleStore
	migrator       sessioninventory.ProofMigrator
	catalogRuntime sessioninventory.CatalogStoreRuntime
	onSleep        func()
}

func newWatcherRuntime(native sessioninventory.Runtime) *watcherRuntime {
	return &watcherRuntime{now: time.Unix(100, 0), files: map[string][]byte{}, modTimes: map[string]time.Time{}, identities: map[string]string{}, writes: map[string][]byte{}, native: native, store: &fakeLifecycleStore{}}
}

func (r *watcherRuntime) Now() time.Time { return r.now }
func (r *watcherRuntime) Sleep(duration time.Duration) {
	r.now = r.now.Add(duration)
	if r.onSleep != nil {
		r.onSleep()
	}
}
func (r *watcherRuntime) ReadFile(path string) ([]byte, error) {
	if err := r.readErrors[path]; err != nil {
		return nil, err
	}
	raw, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), raw...), nil
}
func (r *watcherRuntime) ModTime(path string) (time.Time, error) {
	value, ok := r.modTimes[path]
	if !ok {
		return time.Time{}, os.ErrNotExist
	}
	return value, nil
}
func (r *watcherRuntime) ProcessIdentity(pid string) string { return r.identities[pid] }
func (r *watcherRuntime) AtomicWrite(path string, raw []byte) error {
	r.writes[path] = append([]byte(nil), raw...)
	return nil
}
func (r *watcherRuntime) Log(outcome adapt.Outcome, _ string)                { r.logs = append(r.logs, outcome) }
func (r *watcherRuntime) NativeRuntime(_, _ string) sessioninventory.Runtime { return r.native }
func (r *watcherRuntime) LedgerAppender() LedgerAppender                     { return r.store }
func (r *watcherRuntime) CatalogStore() sessioninventory.CatalogStore {
	if r.catalogRuntime != nil {
		return sessioninventory.CatalogStore{Runtime: r.catalogRuntime}
	}
	return sessioninventory.CatalogStore{Runtime: sessioninventory.CatalogOSRuntime{}}
}
func (r *watcherRuntime) ProofMigrator() *sessioninventory.ProofMigrator { return &r.migrator }

type failingCatalogRuntime struct{}

func (failingCatalogRuntime) MkdirAll(string, os.FileMode) error {
	return errors.New("catalog unavailable")
}
func (failingCatalogRuntime) Lock(string) (sessioninventory.CatalogUnlocker, error) {
	return nil, errors.New("catalog unavailable")
}
func (failingCatalogRuntime) ReadFile(string) ([]byte, error) { return nil, os.ErrNotExist }
func (failingCatalogRuntime) CreateTemp(string, string) (sessioninventory.CatalogFile, error) {
	return nil, errors.New("catalog unavailable")
}
func (failingCatalogRuntime) Remove(string) error         { return nil }
func (failingCatalogRuntime) Rename(string, string) error { return errors.New("catalog unavailable") }
func (failingCatalogRuntime) SyncDirectory(string) error  { return errors.New("catalog unavailable") }
func (r *watcherRuntime) hasLog(want adapt.Outcome) bool {
	for _, outcome := range r.logs {
		if outcome == want {
			return true
		}
	}
	return false
}

func codexRound(sid, text string) []byte {
	return []byte(`{"timestamp":"2026-08-28T01:00:00Z","type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + text + `"}]}}` + "\n" +
		`{"type":"response_item","payload":{"type":"function_call"}}` + "\n")
}

func mustLaunchRecord(t *testing.T, record sessionledger.Record) []byte {
	t.Helper()
	raw, err := sessionledger.EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

type scopedTestPaths struct{ dataDir, tag string }

func mustScopedPaths(t *testing.T, dataDir, tag string) scopedTestPaths {
	t.Helper()
	return scopedTestPaths{dataDir: dataDir, tag: tag}
}
func (p scopedTestPaths) Ledger() string   { return filepath.Join(p.dataDir, "ledger-"+p.tag+".jsonl") }
func (p scopedTestPaths) Log() string      { return filepath.Join(p.dataDir, "log-"+p.tag+".md") }
func (p scopedTestPaths) AgentPID() string { return filepath.Join(p.dataDir, "agent-pid-"+p.tag) }
func (p scopedTestPaths) Catalog() string {
	return filepath.Join(p.dataDir, "session-inventory-catalog.json")
}
func (p scopedTestPaths) Config(agent string) string {
	return filepath.Join(p.dataDir, "config-"+p.tag+"-"+agent+".json")
}
