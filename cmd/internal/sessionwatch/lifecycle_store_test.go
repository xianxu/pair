package sessionwatch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestLifecycleReconcilesIndeterminateLedgerRowsThroughProductionStore(t *testing.T) {
	t.Parallel()
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{{StableID: "root-a", NativeID: "native-a", Role: sessioninventory.RoleRoot}}}}}

	t.Run("launch", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ledger.jsonl")
		runtime := &onceFailureLedgerRuntime{Runtime: sessionledger.OSRuntime{}, stage: "sync", remaining: 1}
		prepared, err := PrepareLaunch(PrepareLaunchInput{Owner: owner, LedgerPath: path, Inventory: inventory}, sessionledger.LedgerStore{Runtime: runtime})
		if err != nil || prepared.Launch.Ordinal != 1 {
			t.Fatalf("prepared=%#v err=%v", prepared, err)
		}
		parsed := sessionledger.ParseLedger(mustReadLedger(t, path))
		if len(parsed.Records) != 1 {
			t.Fatalf("parsed=%#v", parsed)
		}
	})

	t.Run("live binding", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ledger.jsonl")
		baseStore := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
		launch, err := baseStore.Append(path, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, NativeWatermarks: []sessionledger.NativeWatermark{}})
		if err != nil {
			t.Fatal(err)
		}
		runtime := &onceFailureLedgerRuntime{Runtime: sessionledger.OSRuntime{}, stage: "directory-sync", remaining: 1}
		observation := sessioninventory.RoundObservation{ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: sessioninventory.AgentCodex, RootNodeID: "root-a", PairPositions: []uint64{1}, NativePositions: []uint64{2}}
		resolved, err := ObserveAndPersist(ObserveInput{Owner: owner, LedgerPath: path, LaunchOrdinal: launch.Ordinal, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{observation}}, sessionledger.LedgerStore{Runtime: runtime}, nil)
		if err != nil || len(resolved.Bindings) != 1 || resolved.Bindings[0].Status != sessioninventory.BindingEstablished {
			t.Fatalf("resolved=%#v err=%v", resolved, err)
		}
		current, ok := sessionledger.CurrentLaunch(sessionledger.ParseLedger(mustReadLedger(t, path)).Records, owner)
		if !ok || current.Binding == nil || current.Binding.RootNativeID != "native-a" || len(current.Bindings) != 1 {
			t.Fatalf("current=%#v ok=%v", current, ok)
		}
	})
}

func mustReadLedger(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type onceFailureLedgerRuntime struct {
	sessionledger.Runtime
	stage     string
	remaining int
}

func (r *onceFailureLedgerRuntime) fail(stage string) bool {
	if r.stage != stage || r.remaining == 0 {
		return false
	}
	r.remaining--
	return true
}

func (r *onceFailureLedgerRuntime) OpenAppend(path string, mode os.FileMode) (sessionledger.AppendFile, error) {
	file, err := r.Runtime.OpenAppend(path, mode)
	if err != nil {
		return nil, err
	}
	return &onceFailureLedgerFile{AppendFile: file, runtime: r}, nil
}

func (r *onceFailureLedgerRuntime) SyncDirectory(path string) error {
	if r.fail("directory-sync") {
		return errors.New("injected directory sync failure")
	}
	return r.Runtime.SyncDirectory(path)
}

type onceFailureLedgerFile struct {
	sessionledger.AppendFile
	runtime *onceFailureLedgerRuntime
}

func (f *onceFailureLedgerFile) Sync() error {
	if f.runtime.fail("sync") {
		return errors.New("injected file sync failure")
	}
	return f.AppendFile.Sync()
}
