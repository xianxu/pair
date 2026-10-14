package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/durablefile"
)

func moveFixture(t *testing.T) (*hubRuntime, []couchcore.HubDefinition, couchcore.HubSlot) {
	t.Helper()
	root, _ := filepath.EvalSymlinks(t.TempDir())
	var defs []couchcore.HubDefinition
	for _, name := range []string{"ops", "bench", "third"} {
		p := filepath.Join(root, name)
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
		doc, _ := couchcore.ParseHubClaims(nil)
		defs = append(defs, couchcore.HubDefinition{Root: p, Workspace: root, Slot: couchcore.HubSlot{Repository: p, Number: 0}, Claims: doc})
	}
	state, err := couchcore.NewHubOwnership(defs)
	if err != nil {
		t.Fatal(err)
	}
	return &hubRuntime{state: state}, defs, couchcore.HubSlot{Repository: filepath.Join(root, "pair"), Number: 7}
}
func TestHubMoveEphemeralThenPersistAndRestart(t *testing.T) {
	h, defs, slot := moveFixture(t)
	move := couchcore.HubMove{Caller: defs[1].Root, Target: defs[1].Root, Slots: []couchcore.HubSlot{slot}, Revision: h.snapshot().Revision()}
	verify := func() error { return nil }
	if _, err := h.applyMove(context.Background(), move, false, hubClaimFiles{}, verify); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(defs[1].Root, ".hub")); !os.IsNotExist(err) {
		t.Fatalf("ephemeral wrote file %v", err)
	}
	move.Revision = h.snapshot().Revision()
	if _, err := h.applyMove(context.Background(), move, true, hubClaimFiles{}, verify); err != nil {
		t.Fatal(err)
	}
	doc, err := ReadHubClaimFile(defs[1].Root)
	if err != nil {
		t.Fatal(err)
	}
	defs[1].Claims = doc
	restarted, err := couchcore.NewHubOwnership(defs)
	if err != nil {
		t.Fatal(err)
	}
	if got := restarted.Resolve(slot).Hub; got != defs[1].Root {
		t.Fatalf("restart owner %s", got)
	}
	move.Target = defs[0].Root
	move.Revision = h.snapshot().Revision()
	if _, err := h.applyMove(context.Background(), move, true, hubClaimFiles{}, verify); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(defs[1].Root, ".hub"))
	if err != nil || len(raw) != 0 {
		t.Fatalf("move back %q %v", raw, err)
	}
}
func TestHubMoveFailureKeepsOwnershipAndStaleMoveRefuses(t *testing.T) {
	h, defs, slot := moveFixture(t)
	move := couchcore.HubMove{Caller: defs[1].Root, Target: defs[1].Root, Slots: []couchcore.HubSlot{slot}, Revision: h.snapshot().Revision()}
	files := hubClaimFiles{write: func(string, []byte, os.FileMode) error { return errors.New("disk full") }}
	if _, err := h.applyMove(context.Background(), move, true, files, func() error { return nil }); err == nil {
		t.Fatal("accepted failed persistence")
	}
	if h.snapshot().Resolve(slot).Hub != defs[0].Root {
		t.Fatal("failure moved slot")
	}
	if _, err := h.applyMove(context.Background(), move, false, hubClaimFiles{}, func() error { return errors.New("disconnected") }); err == nil {
		t.Fatal("accepted revoked caller")
	}
	if _, err := h.applyMove(context.Background(), move, false, hubClaimFiles{}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	move.Target = defs[2].Root
	if _, err := h.applyMove(context.Background(), move, false, hubClaimFiles{}, func() error { return nil }); err == nil {
		t.Fatal("accepted stale revision")
	}
}

func TestHubMoveSatisfiedPersistDoesNotInvalidateQueuedRevision(t *testing.T) {
	h, defs, slot := moveFixture(t)
	move := couchcore.HubMove{Caller: defs[1].Root, Target: defs[1].Root, Slots: []couchcore.HubSlot{slot}, Revision: h.snapshot().Revision()}
	if _, err := h.applyMove(context.Background(), move, true, hubClaimFiles{}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	before := h.snapshot().Revision()
	move.Revision = before
	if _, err := h.applyMove(context.Background(), move, true, hubClaimFiles{}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if h.snapshot().Revision() != before {
		t.Fatal("already satisfied move changed revision")
	}
}

// Exercise the production replan, not just an adapter retry of the old edits.
func TestHubMoveRetryConfirmsAlreadyVisibleClaims(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			h, defs, slot := moveFixture(t)
			// A broad main claim makes target publication and source exact removal
			// both necessary, covering partial and fully visible multi-file moves.
			if err := os.WriteFile(filepath.Join(defs[0].Root, ".hub"), []byte("pair\npair:7\n"), 0600); err != nil {
				t.Fatal(err)
			}
			doc, err := ReadHubClaimFile(defs[0].Root)
			if err != nil {
				t.Fatal(err)
			}
			defs[0].Claims = doc
			h.state, err = couchcore.NewHubOwnership(defs)
			if err != nil {
				t.Fatal(err)
			}
			move := couchcore.HubMove{Caller: defs[0].Root, Target: defs[1].Root, Slots: []couchcore.HubSlot{slot}, Revision: h.snapshot().Revision()}
			writes := 0
			failedRoot := ""
			files := hubClaimFiles{write: func(path string, raw []byte, mode os.FileMode) error {
				writes++
				// Model the real failure boundary: visible atomic rename, no confirmed
				// parent-directory sync. The temporary file is in the same directory.
				temp := path + ".test-temp"
				if err := os.WriteFile(temp, raw, mode); err != nil {
					return err
				}
				if err := os.Rename(temp, path); err != nil {
					return err
				}
				if writes == failAt {
					failedRoot = filepath.Dir(path)
					return errors.New("directory sync failed")
				}
				return durablefile.SyncDirectory(filepath.Dir(path))
			}}
			verify := func() error { return nil }
			if _, err := h.applyMove(context.Background(), move, true, files, verify); err == nil {
				t.Fatal("claimed failed sync succeeded")
			}
			files = hubClaimFiles{syncDirectory: func(root string) error {
				if root == failedRoot {
					return errors.New("directory sync still failing")
				}
				return durablefile.SyncDirectory(root)
			}}
			if _, err := h.applyMove(context.Background(), move, true, files, verify); err == nil {
				t.Fatal("replanned retry claimed persistence without confirming directory sync")
			}
			if h.snapshot().Revision() != move.Revision || h.snapshot().Resolve(slot).Hub != defs[0].Root {
				t.Fatal("failed reconciliation published ownership")
			}
			if _, err := h.applyMove(context.Background(), move, true, hubClaimFiles{}, verify); err != nil {
				t.Fatal(err)
			}
			for i := range defs {
				defs[i].Claims, err = ReadHubClaimFile(defs[i].Root)
				if err != nil {
					t.Fatal(err)
				}
			}
			restarted, err := couchcore.NewHubOwnership(defs)
			if err != nil {
				t.Fatal(err)
			}
			if restarted.Resolve(slot).Hub != defs[1].Root {
				t.Fatal("confirmed retry did not survive restart")
			}
		})
	}
}
