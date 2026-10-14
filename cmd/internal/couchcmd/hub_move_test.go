package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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
