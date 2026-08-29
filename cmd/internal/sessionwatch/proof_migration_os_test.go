package sessionwatch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestMigrateProoflessBindingsUpgradesPersistedLegacyOwner(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	dataDir := t.TempDir()
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionDir := filepath.Join(home, ".codex", "sessions", "2026", "08", "29")
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "rollout-test-"+sid+".jsonl"), codexRound(sid, "legacy persisted owner"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dataDir, "ledger-work.jsonl")
	store := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	launch, err := store.Append(path, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendBindingIfCurrent(path, owner, launch.Ordinal, sid); err != nil {
		t.Fatal(err)
	}
	if err := MigrateProoflessBindings(home, dataDir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := sessionledger.CurrentLaunch(sessionledger.ParseLedger(raw).Records, owner)
	if !ok || current.Binding == nil || current.Binding.AuthorizationProof == nil || current.Binding.RootNativeID != sid {
		t.Fatalf("current=%#v ok=%v", current, ok)
	}
}
