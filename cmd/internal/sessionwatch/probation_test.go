package sessionwatch

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"testing"
	"time"
)

func TestPrepareResumeRecordsTargetWithoutTranscript(t *testing.T) {
	rt := sessioninventorytest.NewFakeRuntime()
	rt.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"})
	store := &fakeLifecycleStore{}
	launch, err := PrepareRuntimeLaunchRequest(t.TempDir(), sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}, "A", sessionledger.RequestOriginResume, 12, rt, store)
	if err != nil || launch.Binding != nil || launch.Launch.Version != 3 || launch.Launch.RequestedNativeID != "A" || !launch.Launch.BaselineComplete || len(store.records) != 1 {
		t.Fatalf("launch=%+v err=%v records=%+v", launch, err, store.records)
	}
	if rt.OperationCount(sessioninventorytest.OperationReadAt, "") != 0 {
		t.Fatal("read native bodies")
	}
}

func TestRunResumeObservesDifferentRootAfterSilence(t *testing.T) {
	rtNative := sessioninventorytest.NewFakeRuntime()
	rtNative.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native"})
	const sid = "019eff64-6ceb-7e72-9d41-a735a97029ac"
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-test-" + sid + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
	dataDir := t.TempDir()
	paths := mustScopedPaths(t, dataDir, "work")
	rt := newWatcherRuntime(rtNative)
	rt.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", RequestedNativeID: "A", RequestOrigin: sessionledger.RequestOriginResume, BaselineComplete: true})
	rt.files[paths.AgentPID()] = []byte("1234")
	rt.modTimes[paths.AgentPID()] = rt.now
	rt.identities["1234"] = "owned"
	start := rt.now
	rt.onSleep = func() {
		if rt.now.Sub(start) > time.Second {
			rtNative.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "file", GenerationToken: "gen", MutationToken: "now"}, codexRound(sid, "hello"))
			rt.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\nhello\n\n---\n\n")
		}
		if rt.now.Sub(start) > 2*time.Minute {
			rt.identities["1234"] = "gone"
		}
	}
	err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, DataDir: dataDir, Timeout: time.Millisecond, Poll: time.Millisecond, SlowPoll: time.Second}, rt)
	if err != nil || len(rt.store.records) != 1 || rt.store.records[0].RootNativeID != sid {
		t.Fatalf("err=%v records=%+v", err, rt.store.records)
	}
}

func TestPrepareIncompleteBaselineStillPersistsResumeTarget(t *testing.T) {
	rt := sessioninventorytest.NewFakeRuntime()
	rt.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"})
	rt.SetError(sessioninventorytest.OperationListFiles, "codex-sessions", errors.New("offline"))
	store := &fakeLifecycleStore{}
	prepared, err := PrepareRuntimeLaunchRequest(t.TempDir(), sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}, "A", sessionledger.RequestOriginResume, 0, rt, store)
	if err != nil || prepared.Launch.BaselineComplete || prepared.Launch.RequestedNativeID != "A" {
		t.Fatalf("launch=%+v err=%v", prepared.Launch, err)
	}
}

func TestChosenFilenameConfirmsWithoutParsingNativeBody(t *testing.T) {
	native := sessioninventorytest.NewFakeRuntime()
	native.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects"})
	const sid = "11111111-1111-4111-8111-111111111111"
	native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "claude-projects", RelativePath: "-repo/" + sid + ".jsonl"}}, []byte("unfamiliar native format"))
	dataDir := t.TempDir()
	paths := mustScopedPaths(t, dataDir, "work")
	rt := newWatcherRuntime(native)
	rt.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", RequestedNativeID: sid, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true})
	err := Run(Options{Agent: "claude", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, DataDir: dataDir, PIDWait: time.Nanosecond, Timeout: time.Nanosecond, Poll: time.Nanosecond}, rt)
	if err != nil || len(rt.store.records) != 1 || rt.store.records[0].ConfirmationReason != "chosen-id" {
		t.Fatalf("err=%v records=%+v", err, rt.store.records)
	}
	if native.OperationCount(sessioninventorytest.OperationReadAt, "") != 0 {
		t.Fatal("handshake parsed body")
	}
}

func TestIncompleteBaselineAcquiresLaterEpochBeforeConfirming(t *testing.T) {
	native := sessioninventorytest.NewFakeRuntime()
	native.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"})
	const sid = "019eff64-6ceb-7e72-9d41-a735a97029ac"
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-test-" + sid + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "file", GenerationToken: "gen", MutationToken: "before"}, codexRound(sid, "hello"))
	dataDir := t.TempDir()
	paths := mustScopedPaths(t, dataDir, "work")
	rt := newWatcherRuntime(native)
	rt.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", RequestedNativeID: sid, RequestOrigin: sessionledger.RequestOriginResume, BaselineComplete: false})
	rt.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\nhello\n\n---\n\n")
	rt.files[paths.AgentPID()] = []byte("1234")
	rt.modTimes[paths.AgentPID()] = rt.now
	rt.identities["1234"] = "owned"
	sleeps := 0
	rt.onSleep = func() {
		sleeps++
		if sleeps == 2 {
			if len(rt.store.records) != 0 {
				t.Fatal("historical round confirmed incomplete epoch")
			}
			native.AppendFile(artifact, []byte(`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"new epoch input"}]}}`+"\n"+`{"type":"response_item","payload":{"type":"function_call"}}`+"\n"), "after")
			rt.files[paths.Log()] = append(rt.files[paths.Log()], []byte("## 2026-08-28 01:00:02\n\nnew epoch input\n\n---\n\n")...)
		}
		if sleeps > 10 {
			rt.identities["1234"] = "gone"
		}
	}
	err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, DataDir: dataDir, Timeout: time.Millisecond, Poll: time.Millisecond, SlowPoll: time.Millisecond}, rt)
	if err != nil || len(rt.store.records) != 1 || rt.store.records[0].RootNativeID != sid {
		t.Fatalf("err=%v records=%+v", err, rt.store.records)
	}
}

func TestResumeExistingRootConfirmsOnlyAppendedExchange(t *testing.T) {
	native := sessioninventorytest.NewFakeRuntime()
	native.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"})
	const sid = "019eff64-6ceb-7e72-9d41-a735a97029ac"
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-test-" + sid + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
	before := codexRound(sid, "hello")
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "file", GenerationToken: "gen", MutationToken: "before"}, before)
	dataDir := t.TempDir()
	paths := mustScopedPaths(t, dataDir, "work")
	rt := newWatcherRuntime(native)
	rt.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", RequestedNativeID: sid, RequestOrigin: sessionledger.RequestOriginResume, BaselineComplete: true, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath, StableFileID: "file", GenerationToken: "gen", MutationToken: "before", RawSize: int64(len(before))}}})
	rt.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\nhello\n\n---\n\n")
	rt.files[paths.AgentPID()] = []byte("1234")
	rt.modTimes[paths.AgentPID()] = rt.now
	rt.identities["1234"] = "owned"
	sleeps := 0
	rt.onSleep = func() {
		sleeps++
		if sleeps == 1 {
			if len(rt.store.records) != 0 {
				t.Fatal("old exchange confirmed")
			}
			native.AppendFile(artifact, []byte(`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}}`+"\n"+`{"type":"response_item","payload":{"type":"function_call"}}`+"\n"), "after")
		}
		if sleeps > 10 {
			rt.identities["1234"] = "gone"
		}
	}
	err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, DataDir: dataDir, Timeout: time.Millisecond, Poll: time.Millisecond, SlowPoll: time.Millisecond}, rt)
	if err != nil || len(rt.store.records) != 1 || rt.store.records[0].RootNativeID != sid {
		t.Fatalf("err=%v records=%+v", err, rt.store.records)
	}
}
