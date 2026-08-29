package sessionwatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestPrepareLaunchAuthority(t *testing.T) {
	t.Parallel()
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{
		{StableID: "root-a", NativeID: "native-a", Role: sessioninventory.RoleRoot},
		{StableID: "root-b", NativeID: "native-b", Role: sessioninventory.RoleRoot},
	}}}}
	events := []sessioninventory.NativeEventFact{
		{Agent: sessioninventory.AgentCodex, RootNodeID: "root-a", Position: 5},
		{Agent: sessioninventory.AgentCodex, RootNodeID: "root-a", Position: 9},
	}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}

	t.Run("fresh launch is content-free provisional baseline", func(t *testing.T) {
		store := &fakeLifecycleStore{}
		prepared, err := PrepareLaunch(PrepareLaunchInput{Owner: owner, PairLogOffset: 42, Inventory: inventory, NativeEvents: events}, store)
		if err != nil {
			t.Fatal(err)
		}
		if prepared.Launch.Ordinal != 1 || prepared.Binding != nil || len(store.records) != 1 {
			t.Fatalf("prepared=%#v records=%#v", prepared, store.records)
		}
		want := []sessionledger.NativeWatermark{{RootNativeID: "native-a", EventPosition: 9}, {RootNativeID: "native-b", EventPosition: 0}}
		if !equalWatermarks(prepared.Launch.NativeWatermarks, want) {
			t.Fatalf("watermarks=%#v want=%#v", prepared.Launch.NativeWatermarks, want)
		}
	})

	t.Run("explicit scanner-authorized resume immediately joins generation", func(t *testing.T) {
		store := &fakeLifecycleStore{}
		prepared, err := PrepareLaunch(PrepareLaunchInput{Owner: owner, Inventory: inventory, ResumeNativeID: "native-b"}, store)
		if err != nil || prepared.Binding == nil || prepared.Binding.RootNativeID != "native-b" || len(store.records) != 2 {
			t.Fatalf("prepared=%#v records=%#v err=%v", prepared, store.records, err)
		}
	})

	t.Run("unrecognized resume cannot create a launch", func(t *testing.T) {
		store := &fakeLifecycleStore{}
		if _, err := PrepareLaunch(PrepareLaunchInput{Owner: owner, Inventory: inventory, ResumeNativeID: "unknown"}, store); !errors.Is(err, ErrResumeUnauthorized) {
			t.Fatalf("err=%v", err)
		}
		if len(store.records) != 0 {
			t.Fatalf("records=%#v", store.records)
		}
	})

	t.Run("indeterminate launch reconciles without another generation", func(t *testing.T) {
		store := &fakeLifecycleStore{appendErr: &sessionledger.AppendOutcomeError{Outcome: sessionledger.AppendIndeterminate, Err: errors.New("sync uncertain")}}
		prepared, err := PrepareLaunch(PrepareLaunchInput{Owner: owner, Inventory: inventory}, store)
		if err != nil || prepared.Launch.Ordinal != 1 || store.reconciles != 1 || len(store.records) != 1 {
			t.Fatalf("prepared=%#v records=%#v reconciles=%d err=%v", prepared, store.records, store.reconciles, err)
		}
	})

	t.Run("committed explicit binding advances state and preserves cleanup warning", func(t *testing.T) {
		store := &fakeLifecycleStore{bindingErr: &sessionledger.AppendOutcomeError{Outcome: sessionledger.AppendCommitted, Err: errors.New("unlock failed")}}
		prepared, err := PrepareLaunch(PrepareLaunchInput{Owner: owner, Inventory: inventory, ResumeNativeID: "native-b"}, store)
		if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendCommitted || prepared.Binding == nil || prepared.Binding.RootNativeID != "native-b" {
			t.Fatalf("prepared=%#v outcome=%v err=%v", prepared, sessionledger.AppendOutcomeOf(err), err)
		}
	})
}

func TestPrepareOSLaunchIncrementalCapturesMetadataWithoutBodyReads(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects"}
	runtime.AddRoot(root)
	for i := 0; i < 1573; i++ {
		id := fmt.Sprintf("%08x-1111-4111-8111-%012x", i+1, i+1)
		runtime.PutFile(sessioninventory.FileEntry{
			Artifact:     sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/" + id + ".jsonl"},
			StableFileID: sessioninventory.StableFileID(id), GenerationToken: "gen:1", MutationToken: "ctime:1",
		}, []byte(`{"sessionId":"`+id+`"}`+"\n"))
	}
	store := &fakeLifecycleStore{}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"}
	prepared, err := PrepareRuntimeLaunch(t.TempDir(), owner, "", 0, runtime, store)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Launch.Version != 2 || len(prepared.Launch.LaunchArtifactBoundaries) != 1573 {
		t.Fatalf("launch=%#v", prepared.Launch)
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadAt, ""); got != 0 {
		t.Fatalf("body range reads=%d, want 0", got)
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadFile, ""); got != 0 {
		t.Fatalf("body file reads=%d, want 0", got)
	}
}

func TestPrepareLaunchV2PublishesExplicitProof(t *testing.T) {
	t.Parallel()
	store := &fakeLifecycleStore{}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"}
	proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: "native-a", ScannerSchema: "claude-v1", ScannerState: json.RawMessage(`{"version":1}`), Artifacts: []sessionledger.ArtifactProof{{StorageRoot: "claude-projects", RelativePath: "a.jsonl", StableFileID: "stable", MutationToken: "mutation"}}}
	prepared, err := PrepareLaunch(PrepareLaunchInput{Owner: owner, ArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}, ResumeNativeID: "native-a", ResumeProof: &proof}, store)
	if err != nil || prepared.Launch.Version != 2 || prepared.Binding == nil || prepared.Binding.Version != 2 || prepared.Binding.AuthorizationProof == nil {
		t.Fatalf("prepared=%#v err=%v", prepared, err)
	}
}

func TestCorruptCatalogColdLaunchStaysMetadataOnly(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "session-inventory-catalog.json"), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects"}
	runtime.AddRoot(root)
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/11111111-1111-4111-8111-111111111111.jsonl"}, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, []byte("private transcript\n"))
	prepared, err := PrepareRuntimeLaunch(dataDir, sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"}, "", 0, runtime, &fakeLifecycleStore{})
	if err != nil || prepared.Launch.Version != 2 || runtime.OperationCount(sessioninventorytest.OperationReadAt, "") != 0 || runtime.OperationCount(sessioninventorytest.OperationReadFile, "") != 0 {
		t.Fatalf("prepared=%#v reads=%d/%d err=%v", prepared, runtime.OperationCount(sessioninventorytest.OperationReadAt, ""), runtime.OperationCount(sessioninventorytest.OperationReadFile, ""), err)
	}
}

func TestObserveAndPersist(t *testing.T) {
	t.Parallel()
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{
		{StableID: "root-a", NativeID: "native-a", Role: sessioninventory.RoleRoot},
		{StableID: "root-b", NativeID: "native-b", Role: sessioninventory.RoleRoot},
	}}}}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	observation := sessioninventory.RoundObservation{ScopeKey: "scope", Tag: "work", Agent: sessioninventory.AgentCodex, RootNodeID: "root-a", PairPositions: []uint64{10}, NativePositions: []uint64{2}}

	t.Run("no completed round stays provisional without writes", func(t *testing.T) {
		store := &fakeLifecycleStore{}
		result, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory}, store, nil)
		if err != nil || len(store.records) != 0 || len(result.Bindings) != 1 || result.Bindings[0].Status != sessioninventory.BindingProvisional {
			t.Fatalf("result=%#v records=%#v err=%v", result, store.records, err)
		}
	})

	t.Run("unique completed round persists then refreshes cache", func(t *testing.T) {
		store := &fakeLifecycleStore{}
		var cached ConfigPayload
		result, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{observation}, Args: []string{"resume", "old", "--flag"}}, store, func(payload ConfigPayload) error {
			cached = payload
			return nil
		})
		if err != nil || len(store.records) != 1 || len(result.Bindings) != 1 || result.Bindings[0].Status != sessioninventory.BindingEstablished {
			t.Fatalf("result=%#v records=%#v err=%v", result, store.records, err)
		}
		if cached.SessionID != "native-a" || len(cached.Args) != 1 || cached.Args[0] != "--flag" {
			t.Fatalf("cached=%#v", cached)
		}
	})

	t.Run("stale generation cannot persist or refresh cache", func(t *testing.T) {
		store := &fakeLifecycleStore{stale: true}
		cacheCalls := 0
		result, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{observation}}, store, func(ConfigPayload) error {
			cacheCalls++
			return nil
		})
		if !errors.Is(err, sessionledger.ErrStaleLaunch) || cacheCalls != 0 || result.Bindings[0].Status != sessioninventory.BindingProvisional {
			t.Fatalf("result=%#v cacheCalls=%d err=%v", result, cacheCalls, err)
		}
	})

	t.Run("cache failure leaves durable binding established", func(t *testing.T) {
		store := &fakeLifecycleStore{}
		result, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{observation}}, store, func(ConfigPayload) error {
			return errors.New("cache unavailable")
		})
		if err != nil || result.Bindings[0].Status != sessioninventory.BindingEstablished {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		found := false
		for _, diagnostic := range result.Diagnostics {
			found = found || diagnostic.Code == sessioninventory.DiagnosticBindingStale
		}
		if !found {
			t.Fatalf("diagnostics=%#v", result.Diagnostics)
		}
	})

	t.Run("indeterminate binding reconciles then establishes", func(t *testing.T) {
		store := &fakeLifecycleStore{bindingErr: &sessionledger.AppendOutcomeError{Outcome: sessionledger.AppendIndeterminate, Err: errors.New("sync uncertain")}}
		result, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{observation}}, store, nil)
		if err != nil || store.reconciles != 1 || result.Bindings[0].Status != sessioninventory.BindingEstablished {
			t.Fatalf("result=%#v reconciles=%d err=%v", result, store.reconciles, err)
		}
	})

	t.Run("committed binding establishes while preserving cleanup warning", func(t *testing.T) {
		store := &fakeLifecycleStore{bindingErr: &sessionledger.AppendOutcomeError{Outcome: sessionledger.AppendCommitted, Err: errors.New("unlock failed")}}
		result, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{observation}}, store, nil)
		if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendCommitted || result.Bindings[0].Status != sessioninventory.BindingEstablished {
			t.Fatalf("result=%#v outcome=%v err=%v", result, sessionledger.AppendOutcomeOf(err), err)
		}
	})
}

func TestFatalLaunchBaselineDiagnostic(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		diagnostic sessioninventory.Diagnostic
		want       bool
	}{
		{"unreadable root transcript", sessioninventory.Diagnostic{Code: sessioninventory.DiagnosticTurnUnusable, Detail: "root transcript unreadable"}, true},
		{"multiple root transcripts", sessioninventory.Diagnostic{Code: sessioninventory.DiagnosticTurnUnusable, Detail: "root requires exactly one transcript artifact"}, true},
		{"near miss", sessioninventory.Diagnostic{Code: sessioninventory.DiagnosticTurnUnusable, Detail: "native event was not allowlisted"}, false},
		{"unrelated diagnostic", sessioninventory.Diagnostic{Code: sessioninventory.DiagnosticNodeMalformed, Detail: "root transcript unreadable"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := fatalLaunchBaselineDiagnostic(test.diagnostic); got != test.want {
				t.Fatalf("fatalLaunchBaselineDiagnostic(%#v) = %v, want %v", test.diagnostic, got, test.want)
			}
		})
	}
}

type fakeLifecycleStore struct {
	records      []sessionledger.Record
	stale        bool
	appendErr    error
	bindingErr   error
	reconcileErr error
	reconciles   int
}

func (f *fakeLifecycleStore) Append(_ string, record sessionledger.Record) (sessionledger.Record, error) {
	record.Ordinal = uint64(len(f.records) + 1)
	f.records = append(f.records, record)
	return record, f.appendErr
}

func (f *fakeLifecycleStore) AppendBindingIfCurrent(_ string, owner sessionledger.Owner, launchOrdinal uint64, rootNativeID string) (sessionledger.Record, error) {
	if f.stale {
		return sessionledger.Record{}, sessionledger.ErrStaleLaunch
	}
	record := sessionledger.Record{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, LaunchOrdinal: launchOrdinal, RootNativeID: rootNativeID, Ordinal: uint64(len(f.records) + 1)}
	f.records = append(f.records, record)
	return record, f.bindingErr
}

func (f *fakeLifecycleStore) AppendBindingProofIfCurrent(_ string, owner sessionledger.Owner, launchOrdinal uint64, proof sessionledger.AuthorizationProof) (sessionledger.Record, error) {
	if f.stale {
		return sessionledger.Record{}, sessionledger.ErrStaleLaunch
	}
	record := sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, LaunchOrdinal: launchOrdinal, RootNativeID: proof.RootNativeID, AuthorizationProof: &proof, Ordinal: uint64(len(f.records) + 1)}
	f.records = append(f.records, record)
	return record, f.bindingErr
}

func (f *fakeLifecycleStore) Reconcile(_ string, record sessionledger.Record) error {
	f.reconciles++
	if f.reconcileErr != nil {
		return f.reconcileErr
	}
	for _, candidate := range f.records {
		if candidate.Ordinal == record.Ordinal {
			return nil
		}
	}
	return errors.New("record missing")
}

func equalWatermarks(left, right []sessionledger.NativeWatermark) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
