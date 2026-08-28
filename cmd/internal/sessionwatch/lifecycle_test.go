package sessionwatch

import (
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
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
	records []sessionledger.Record
	stale   bool
}

func (f *fakeLifecycleStore) Append(_ string, record sessionledger.Record) (sessionledger.Record, error) {
	record.Ordinal = uint64(len(f.records) + 1)
	f.records = append(f.records, record)
	return record, nil
}

func (f *fakeLifecycleStore) AppendBindingIfCurrent(_ string, owner sessionledger.Owner, launchOrdinal uint64, rootNativeID string) (sessionledger.Record, error) {
	if f.stale {
		return sessionledger.Record{}, sessionledger.ErrStaleLaunch
	}
	record := sessionledger.Record{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, LaunchOrdinal: launchOrdinal, RootNativeID: rootNativeID, Ordinal: uint64(len(f.records) + 1)}
	f.records = append(f.records, record)
	return record, nil
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
