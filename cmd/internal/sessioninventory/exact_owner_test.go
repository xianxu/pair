package sessioninventory_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestReadCurrentLaunchAtWarmLedgerDoesNoBodyWorkOrListing(t *testing.T) {
	rt, ledger := launchCacheFixture()
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	read := func() sessionledger.Current {
		t.Helper()
		got, ok, diagnostics, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, owner, ledger)
		if err != nil || !ok || len(diagnostics) != 0 {
			t.Fatalf("current=%+v ok=%v diagnostics=%+v err=%v", got, ok, diagnostics, err)
		}
		return got
	}
	if got := read(); got.Launch.Ordinal != 1 {
		t.Fatalf("cold=%+v", got)
	}
	beforeReads := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl")
	beforeDecode := rt.decoded.Load()
	if beforeReads == 0 || beforeDecode == 0 {
		t.Fatal("cold read/decode counters disconnected")
	}
	for range 20 {
		read()
	}
	if n := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl"); n != beforeReads {
		t.Fatalf("warm body reads %d -> %d", beforeReads, n)
	}
	if n := rt.decoded.Load(); n != beforeDecode {
		t.Fatalf("warm decoded bytes %d -> %d", beforeDecode, n)
	}
	if n := rt.OperationCount(sessioninventorytest.OperationListFiles, ""); n != 0 {
		t.Fatalf("listed %d times", n)
	}
}

func TestReadCurrentLaunchAtChangedReplacedAndConflictingLedger(t *testing.T) {
	rt, ledger := launchCacheFixture()
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	read := func() sessionledger.Current {
		t.Helper()
		got, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, owner, ledger)
		if err != nil || !ok {
			t.Fatalf("got=%+v ok=%v err=%v", got, ok, err)
		}
		return got
	}
	read()
	rt.AppendFile(ledger, []byte(v1Launch), "two")
	if got := read(); got.Launch.Ordinal != 2 {
		t.Fatalf("append=%+v", got)
	}
	rt.PutFile(sessioninventory.FileEntry{Artifact: ledger, StableFileID: "replacement", MutationToken: "replacement"}, []byte(v1Launch))
	if got := read(); got.Launch.Ordinal != 1 {
		t.Fatalf("replacement reused old generation: %+v", got)
	}
	binding := func(id string) string {
		return fmt.Sprintf("{\"v\":1,\"kind\":\"binding\",\"scope_key\":\"scope\",\"tag\":\"work\",\"agent\":\"codex\",\"launch_ordinal\":1,\"root_native_id\":%q}\n", id)
	}
	rt.AppendFile(ledger, []byte(binding("A")+binding("B")), "conflict")
	if got := read(); !got.Conflict || got.Binding != nil {
		t.Fatalf("conflict not preserved: %+v", got)
	}
}

func TestReadCurrentLaunchAtRejectsWrongArtifactWithoutIO(t *testing.T) {
	for _, ledger := range []sessioninventory.Artifact{{StorageRoot: "other", RelativePath: "ledger-work.jsonl"}, {StorageRoot: "pair-data", RelativePath: "ledger-other.jsonl"}, {StorageRoot: "pair-data", RelativePath: "../ledger-work.jsonl"}, {StorageRoot: "pair-data", RelativePath: "nested/ledger-work.jsonl"}} {
		t.Run(ledger.StorageRoot+ledger.RelativePath, func(t *testing.T) {
			rt, _ := launchCacheFixture()
			_, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}, ledger)
			if err == nil || ok {
				t.Fatalf("accepted invalid ledger: ok=%v err=%v", ok, err)
			}
			if n := rt.OperationCount(sessioninventorytest.OperationStatFile, "") + rt.OperationCount(sessioninventorytest.OperationReadAt, "") + rt.OperationCount(sessioninventorytest.OperationListFiles, ""); n != 0 {
				t.Fatalf("invalid artifact performed %d IO operations", n)
			}
		})
	}
}

func TestReadCurrentLaunchAtDriftingReadCannotPublishStaleAuthority(t *testing.T) {
	rt, ledger := launchCacheFixture()
	mutation := 0
	rt.readHook = func() {
		mutation++
		rt.PutFile(sessioninventory.FileEntry{Artifact: ledger, MutationToken: sessioninventory.MutationToken(fmt.Sprint(mutation))}, []byte(v1Launch))
	}
	_, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}, ledger)
	if !errors.Is(err, sessioninventory.ErrArtifactChanged) || ok {
		t.Fatalf("drift gave authority: ok=%v err=%v", ok, err)
	}
	rt.readHook = nil
	got, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}, ledger)
	if err != nil || !ok || got.Launch.Ordinal != 1 {
		t.Fatalf("stable read after drift: got=%+v ok=%v err=%v", got, ok, err)
	}
}

func TestReadCurrentLaunchAtMissingAndMalformedLedgerNeverUsesCachedAuthority(t *testing.T) {
	rt, ledger := launchCacheFixture()
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	if _, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, owner, ledger); err != nil || !ok {
		t.Fatalf("prime: %v %v", ok, err)
	}
	rt.SetError(sessioninventorytest.OperationStatFile, "pair-data:ledger-work.jsonl", os.ErrNotExist)
	if _, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, owner, ledger); err == nil || ok {
		t.Fatalf("missing returned authority: %v %v", ok, err)
	}
	rt.SetError(sessioninventorytest.OperationStatFile, "pair-data:ledger-work.jsonl", nil)
	rt.PutFile(sessioninventory.FileEntry{Artifact: ledger, MutationToken: "malformed"}, []byte("not-json\n"))
	if _, ok, diagnostics, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, owner, ledger); err != nil || ok || len(diagnostics) == 0 {
		t.Fatalf("malformed: ok=%v diagnostics=%+v err=%v", ok, diagnostics, err)
	}
}

func TestReadCurrentLaunchAtPreservesProviderSpecificAndLatestOwnerReductions(t *testing.T) {
	rt, ledger := launchCacheFixture()
	rt.AppendFile(ledger, []byte(strings.ReplaceAll(v1Launch, "codex", "claude")), "claude-launch")
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	got, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, owner, ledger)
	if err != nil || !ok || got.Launch.Ordinal != 1 || got.Launch.Agent != "codex" {
		t.Fatalf("provider-specific reduction changed: %+v %v %v", got, ok, err)
	}
	owner.Agent = ""
	got, ok, _, err = sessioninventory.ReadCurrentLaunchAt(context.Background(), rt, owner, ledger)
	if err != nil || !ok || got.Launch.Ordinal != 2 || got.Launch.Agent != "claude" {
		t.Fatalf("latest owner reduction changed: %+v %v %v", got, ok, err)
	}
}
