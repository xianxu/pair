package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"testing"
)

func TestResourceInventoryWorkerProbesOnlyChangedRows(t *testing.T) {
	probes, workspaces := 0, 0
	address := couchcore.ThreadAddress{RepoScope: "s", Tag: "t"}
	cache := couchcore.NewResourceRootCache()
	inv := &resourceInventory{cache: cache, seen: map[string]string{}, workspace: func(context.Context, string) (couchcore.WorkspaceIdentity, error) {
		workspaces++
		name := "repo:1"
		return couchcore.WorkspaceIdentity{Address: &name, RepoIdentity: "/repo/.git", WorktreeRoot: "/slot"}, nil
	}, probe: func(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error) {
		probes++
		return couchcore.PairSessionBinding{}, nil
	}}
	snap := couchtty.ResourceInventorySnapshot{Rows: []couchcore.ActionableThreadSummary{{Address: address, WorkingPath: "/slot", State: couchcore.ThreadDetached}}}
	inv.refresh(context.Background(), snap)
	inv.refresh(context.Background(), snap)
	if probes != 1 || workspaces != 1 {
		t.Fatalf("repeated IO: probes=%d workspace=%d", probes, workspaces)
	}
	if got := inv.slots(); len(got) != 1 || got[0].OwnershipKnown {
		t.Fatalf("unproved ownership: %+v", got)
	}
	snap.Hosted = []couchcore.LiveTTYObservation{{Address: address, Process: couchcore.ProcessIdentity{PID: 1, Identity: "new"}}}
	inv.refresh(context.Background(), snap)
	if probes != 2 {
		t.Fatal("new helper did not reprobe")
	}
	inv.refresh(context.Background(), couchtty.ResourceInventorySnapshot{})
	if len(inv.slots()) != 1 {
		t.Fatal("pre-refresh console erased seeded inventory")
	}
	inv.refresh(context.Background(), couchtty.ResourceInventorySnapshot{InventoryKnown: true})
	if len(inv.slots()) != 0 {
		t.Fatal("removed slot retained")
	}
}

func TestResourceInventoryKeepsMultipleEmptySlots(t *testing.T) {
	inv := &resourceInventory{seen: map[string]string{}}
	snapshot := couchtty.ResourceInventorySnapshot{Rows: []couchcore.ActionableThreadSummary{
		{Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "repo", Number: 1, WorktreeRoot: "/one"}}},
		{Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "repo", Number: 2, WorktreeRoot: "/two"}}},
	}}
	inv.refresh(context.Background(), snapshot)
	inv.refresh(context.Background(), snapshot)
	rows := inv.slots()
	if len(rows) != 2 || rows[0].Slot != "repo:1" || rows[1].Slot != "repo:2" {
		t.Fatalf("empty slot identity collapsed: %+v", rows)
	}
}

func TestResourceInventorySnapshotRequiresCurrentSessionProofAndNoIO(t *testing.T) {
	a := couchcore.ThreadAddress{RepoScope: "s", Tag: "t"}
	cache := couchcore.NewResourceRootCache()
	revision := cache.Begin(a, "old")
	cache.Complete(a, revision, launcher.SessionOwnerObservation{State: launcher.SessionOwnerOwned, Name: "old", Server: launcher.SessionServerIdentity{PID: 50, Identity: "generation", Session: "old"}}, nil)
	inv := &resourceInventory{cache: cache, rows: []couchcore.ResourceSlot{{Slot: "repo:1", Thread: a, Workspace: couchcore.WorkspaceIdentity{RepoIdentity: "/repo/.git", WorktreeRoot: "/slot"}}}, sessions: map[couchcore.ThreadAddress]string{a: "new"}, workspace: func(context.Context, string) (couchcore.WorkspaceIdentity, error) { panic("snapshot ran IO") }, probe: func(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error) {
		panic("snapshot ran probe")
	}}
	if got := inv.slots()[0]; got.OwnershipKnown || len(got.Roots) != 0 {
		t.Fatalf("adopted old session: %+v", got)
	}
	inv.sessions[a] = "old"
	if got := inv.slots()[0]; !got.OwnershipKnown || len(got.Roots) != 1 || got.Roots[0].Key.PID != 50 {
		t.Fatalf("lost current proof: %+v", got)
	}
	inv.rows[0].Workspace = couchcore.WorkspaceIdentity{}
	if got := inv.slots()[0]; got.OwnershipKnown || got.Reason != "workspace identity unavailable" {
		t.Fatalf("missing workspace claimed: %+v", got)
	}
}
