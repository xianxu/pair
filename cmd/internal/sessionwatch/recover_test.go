package sessionwatch

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type repairRuntime struct {
	OSRuntime
	native sessioninventory.Runtime
}

func (r repairRuntime) NativeRuntime(_, _ string) sessioninventory.Runtime { return r.native }

func TestRepairPreviewAndApplyPreserveConfig(t *testing.T) {
	opts, rt, store := repairFixture(t, false)
	paths, _ := artifactpath.ResolveScoped(opts.DataDir, opts.Tag)
	config, _ := paths.ConfigChecked(opts.Agent)
	original := []byte(`{"args":["--dangerously-custom"]}`)
	if err := os.WriteFile(config, original, 0600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(paths.Ledger())
	got, err := Recover(opts, rt, store)
	if err != nil || got.Status != "repairable" || got.NativeID == "" || got.Applied {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	after, _ := os.ReadFile(paths.Ledger())
	if !bytes.Equal(before, after) {
		t.Fatal("preview changed ledger")
	}
	if _, err := os.Stat(paths.SessionInventoryCatalog()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview wrote catalog: %v", err)
	}
	opts.Apply = true
	got, err = Recover(opts, rt, store)
	if err != nil || !got.Applied {
		t.Fatalf("apply=%+v err=%v", got, err)
	}
	after, _ = os.ReadFile(config)
	if !bytes.Equal(original, after) {
		t.Fatal("repair changed saved arguments")
	}
	got, err = Recover(opts, rt, store)
	if err != nil || got.Status != "already-bound" || got.Applied {
		t.Fatalf("retry=%+v err=%v", got, err)
	}
}

func TestRepairAmbiguousRoundsAbstain(t *testing.T) {
	opts, rt, store := repairFixture(t, true)
	opts.Apply = true
	got, err := Recover(opts, rt, store)
	if err != nil || got.NativeID != "" || got.Applied {
		t.Fatalf("ambiguous=%+v err=%v", got, err)
	}
}

func TestRepairIncompleteBaselineAbstains(t *testing.T) {
	opts, rt, store := repairFixture(t, false)
	paths := mustScopedPaths(t, opts.DataDir, opts.Tag)
	_, err := store.Append(paths.Ledger(), sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: opts.ScopeKey, Tag: opts.Tag, Agent: opts.Agent})
	if err != nil {
		t.Fatal(err)
	}
	opts.Apply = true
	got, err := Recover(opts, rt, store)
	if err != nil || got.Applied || got.Status != "unavailable" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func repairFixture(t *testing.T, duplicate bool) (RecoveryOptions, repairRuntime, sessionledger.LedgerStore) {
	t.Helper()
	opts := RecoveryOptions{Agent: "codex", Tag: "work", ScopeKey: "scope", DataDir: t.TempDir()}
	native := sessioninventorytest.NewFakeRuntime()
	native.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "unused"})
	text := "please inspect the durable watcher boundary now"
	ids := []string{"019eff64-6ceb-7e72-9d41-a735a97029ac"}
	if duplicate {
		ids = append(ids, "123e4567-e89b-12d3-a456-426614174000")
	}
	for _, id := range ids {
		native.PutFile(sessioninventory.FileEntry{StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1", Artifact: sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-test-" + id + ".jsonl", Kind: sessioninventory.ArtifactTranscript}}, codexRound(id, text))
	}
	paths := mustScopedPaths(t, opts.DataDir, opts.Tag)
	store := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	_, err := store.Append(paths.Ledger(), sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: opts.ScopeKey, Tag: opts.Tag, Agent: opts.Agent})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Log(), []byte("## 2026-08-28 01:00:01\n\n"+text+"\n\n---\n\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return opts, repairRuntime{native: native}, store
}

type supersedingRepairStore struct{ sessionledger.LedgerStore }

func (s supersedingRepairStore) ConfirmIfCurrent(path string, owner sessionledger.Owner, ordinal uint64, id, reason string, proof *sessionledger.AuthorizationProof) (sessionledger.Record, error) {
	_, err := s.Append(path, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, BaselineComplete: true})
	if err != nil {
		return sessionledger.Record{}, err
	}
	return s.LedgerStore.ConfirmIfCurrent(path, owner, ordinal, id, reason, proof)
}
func TestRepairApplyCannotBindSupersedingLaunch(t *testing.T) {
	opts, rt, store := repairFixture(t, false)
	opts.Apply = true
	got, err := Recover(opts, rt, supersedingRepairStore{store})
	if !errors.Is(err, sessionledger.ErrStaleLaunch) || got.Applied {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
func TestRepairCLIScopeRequiredAndHelp(t *testing.T) {
	for _, args := range [][]string{{"codex", "work"}, {"codex", "work", "--scope-key"}, {"codex", "work", "--unknown"}} {
		var stdout, stderr bytes.Buffer
		if code := RunRepairCLI(args, func(string) string { return "" }, &stdout, &stderr); code != 2 || stderr.Len() == 0 {
			t.Fatalf("args=%v code=%d error=%s", args, code, &stderr)
		}
	}
	var stdout, stderr bytes.Buffer
	if code := RunRepairCLI([]string{"--help"}, func(string) string { return "" }, &stdout, &stderr); code != 0 || stdout.Len() == 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
	}
}

func TestRepairV3CompleteBaselineUsesSameCorrelation(t *testing.T) {
	opts, rt, store := repairFixture(t, false)
	paths := mustScopedPaths(t, opts.DataDir, opts.Tag)
	_, err := store.Append(paths.Ledger(), sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: opts.ScopeKey, Tag: opts.Tag, Agent: opts.Agent, BaselineComplete: true, RequestedNativeID: "019eff64-6ceb-7e72-9d41-a735a97029ac", RequestOrigin: sessionledger.RequestOriginResume})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Recover(opts, rt, store)
	if err != nil || got.Status != "repairable" || got.LaunchOrdinal != 2 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestRepairDoesNotUseAnotherOwnersLaunch(t *testing.T) {
	opts, rt, store := repairFixture(t, false)
	opts.ScopeKey = "another-scope"
	opts.Apply = true
	got, err := Recover(opts, rt, store)
	if err != nil || got.Applied || got.NativeID != "" || got.LaunchOrdinal != 0 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

type uncertainRepairStore struct{ sessionledger.LedgerStore }

func (s uncertainRepairStore) ConfirmIfCurrent(path string, owner sessionledger.Owner, ordinal uint64, id, reason string, proof *sessionledger.AuthorizationProof) (sessionledger.Record, error) {
	record, err := s.LedgerStore.ConfirmIfCurrent(path, owner, ordinal, id, reason, proof)
	if err != nil {
		return record, err
	}
	return record, &sessionledger.AppendOutcomeError{Outcome: sessionledger.AppendIndeterminate, Err: errors.New("sync result uncertain")}
}
func TestRepairReconcilesUncertainPublicationWithoutDuplicate(t *testing.T) {
	opts, rt, store := repairFixture(t, false)
	opts.Apply = true
	got, err := Recover(opts, rt, uncertainRepairStore{store})
	if err != nil || !got.Applied {
		t.Fatalf("unreconciled repair: %+v %v", got, err)
	}
	paths := mustScopedPaths(t, opts.DataDir, opts.Tag)
	raw, _ := os.ReadFile(paths.Ledger())
	parsed := sessionledger.ParseLedger(raw)
	if len(parsed.Records) != 2 {
		t.Fatalf("duplicate retry: %+v", parsed)
	}
}
