package sessionwatch

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"os"
	"path/filepath"
	"testing"
)

type boundGuardWork struct{ read, decoded int64 }

func (c *boundGuardWork) ReadWork(_ sessioninventory.Artifact, n int64)   { c.read += n }
func (c *boundGuardWork) DecodeWork(_ sessioninventory.Artifact, n int64) { c.decoded += n }

type boundGuardLedger struct {
	sessionledger.OSRuntime
	reads int
}

func (r *boundGuardLedger) ReadFile(path string) ([]byte, error) {
	r.reads++
	return r.OSRuntime.ReadFile(path)
}

func TestBoundPublicationGuardUsesCacheUnderLockAndRejectsChangedLedger(t *testing.T) {
	for _, change := range []string{"append", "replace"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ledger-work.jsonl")
			owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
			launch := mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", BaselineComplete: true})
			binding := func(id string) []byte {
				return mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 1, RootNativeID: id, ConfirmationReason: sessionledger.ConfirmationChosen})
			}
			if err := os.WriteFile(path, append(launch, binding("ours")...), 0600); err != nil {
				t.Fatal(err)
			}
			work := &boundGuardWork{}
			native := sessioninventory.NewOSRuntime(t.TempDir(), dir).WithOwnerLaunchCache(sessioninventory.NewOwnerLaunchCache()).WithWorkObserver(work)
			ledgerIO := &boundGuardLedger{}
			store := sessionledger.LedgerStore{Runtime: ledgerIO}
			read := func() (sessionledger.Current, bool, error) {
				current, ok, _, err := sessioninventory.ReadCurrentLaunchAt(context.Background(), native, owner, sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-work.jsonl"})
				return current, ok, err
			}
			effects := 0
			publish := func() error {
				return store.WithCurrentConfirmationReader(path, owner, 1, "ours", read, func() error { effects++; return nil })
			}
			if err := publish(); err != nil {
				t.Fatal(err)
			}
			before := *work
			for range 10 {
				if err := publish(); err != nil {
					t.Fatal(err)
				}
			}
			if work.read != before.read || work.decoded != before.decoded || ledgerIO.reads != 0 {
				t.Fatalf("unchanged active guard reread: native%+v before%+v legacyReads%d", work, before, ledgerIO.reads)
			}
			if change == "append" {
				f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				_, err = f.Write(launch)
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			} else {
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				tmp := filepath.Join(dir, "replacement")
				if err := os.WriteFile(tmp, append(launch, binding("them")...), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(tmp, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(tmp, path); err != nil {
					t.Fatal(err)
				}
			}
			priorEffects := effects
			if err := publish(); !errors.Is(err, sessionledger.ErrStaleLaunch) {
				t.Fatalf("changed ledger guard=%v", err)
			}
			if effects != priorEffects || work.read <= before.read || work.decoded <= before.decoded {
				t.Fatalf("change not reread before refusing: effects%d/%d work%+v", effects, priorEffects, work)
			}
		})
	}
}
