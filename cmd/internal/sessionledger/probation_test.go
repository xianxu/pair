package sessionledger

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestV3LaunchKeepsRequestedIdentitySeparateFromConfirmation(t *testing.T) {
	raw := []byte(`{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":12,"artifact_boundaries":[],"requested_native_id":"A","request_origin":"resume","baseline_complete":false}` + "\n" + `{"v":3,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"D","confirmation_reason":"correlation"}` + "\n")
	parsed := ParseLedger(raw)
	if len(parsed.Records) != 2 || len(parsed.MalformedOrdinals) != 0 {
		t.Fatalf("v3 records rejected: %+v", parsed)
	}
	current, ok := CurrentLaunch(parsed.Records, Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"})
	if !ok || current.Conflict || current.Binding == nil || current.Binding.RootNativeID != "D" {
		t.Fatalf("current=%+v", current)
	}
	for _, r := range parsed.Records {
		encoded, err := EncodeRecord(r)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err = json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if r.Kind == RecordLaunch && (fields["requested_native_id"] != "A" || fields["baseline_complete"] != false) {
			t.Fatalf("lost request: %s", encoded)
		}
	}
}

func TestConfirmationIdempotentAndRejectsCompetingRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	store := LedgerStore{Runtime: OSRuntime{}}
	owner := Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	launch, err := store.Append(path, Record{Version: 3, Kind: RecordLaunch, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, BaselineComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.ConfirmIfCurrent(path, owner, launch.Ordinal, "D", "correlation", nil)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.ConfirmIfCurrent(path, owner, launch.Ordinal, "D", "correlation", nil)
	if err != nil || again.Ordinal != first.Ordinal {
		t.Fatalf("duplicate=%+v %v", again, err)
	}
	if _, err = store.ConfirmIfCurrent(path, owner, launch.Ordinal, "E", "correlation", nil); err == nil {
		t.Fatal("competing confirmation accepted")
	}
}

func TestConfirmedEffectCannotWriteAfterNewLaunch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	store := LedgerStore{Runtime: OSRuntime{}}
	owner := Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	launch, err := store.Append(path, Record{Version: 3, Kind: RecordLaunch, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, BaselineComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ConfirmIfCurrent(path, owner, launch.Ordinal, "A", "correlation", nil); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Append(path, launch); err != nil {
		t.Fatal(err)
	}
	called := false
	err = store.WithCurrentConfirmation(path, owner, launch.Ordinal, "A", func() error { called = true; return nil })
	if err == nil || called {
		t.Fatalf("stale effect ran: %v %v", called, err)
	}
}

func TestRequestedNativeIDIsSafeAsOpaqueArgvValue(t *testing.T) {
	for _, id := range []string{"--help", "-unsafe", "bad\x00id"} {
		_, err := EncodeRecord(Record{Version: 3, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", RequestedNativeID: id, RequestOrigin: RequestOriginResume, BaselineComplete: true})
		if err == nil {
			t.Fatalf("unsafe requested id accepted: %q", id)
		}
	}
	for _, id := range []string{"opaque/conversation:v2", "future-format_42"} {
		if _, err := EncodeRecord(Record{Version: 3, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", RequestedNativeID: id, RequestOrigin: RequestOriginResume, BaselineComplete: true}); err != nil {
			t.Fatalf("opaque safe id rejected: %v", err)
		}
	}
}
