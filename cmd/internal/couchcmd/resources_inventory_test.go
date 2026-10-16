package couchcmd

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"strings"
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

func TestResourceInventoryStartupSampleWhileProbeBlocked(t *testing.T) {
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	a := couchcore.ThreadAddress{RepoScope: "s", Tag: "live"}
	inv := &resourceInventory{seen: map[string]string{}, probe: func(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error) {
		close(entered)
		<-release
		return couchcore.PairSessionBinding{}, nil
	}}
	go func() {
		defer close(done)
		inv.refresh(context.Background(), couchtty.ResourceInventorySnapshot{InventoryKnown: true, Rows: []couchcore.ActionableThreadSummary{{Address: a, State: couchcore.ThreadLive}}})
	}()
	<-entered
	defer func() { close(release); <-done }()
	root := couchcore.ResourceProcessKey{PID: 10, Identity: "root"}
	child := couchcore.ResourceProcessKey{PID: 11, Identity: "child"}
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "root"}, {PID: 11, PPID: 10, Identity: "child"}}}
	s := testResourceSampler(f)
	s.slots = inv.snapshot
	s.state.Provenance = []couchcore.ResourceProvenance{{ResourceRoot: couchcore.ResourceRoot{Key: child, Slot: "r:1", Thread: a}, Root: root}}
	s.pass(context.Background())
	if len(s.view().Orphans) != 0 || s.view().Processes[0].OrphanKnown {
		t.Fatalf("startup invented orphan: %+v", s.view())
	}
	if got := inv.slots(); len(got) != 1 || got[0].Thread != a {
		t.Fatalf("blocked proof hid known slot metadata: %+v", got)
	}
}

func TestResourceInventoryOverCapPrioritizesLiveRows(t *testing.T) {
	inv := &resourceInventory{seen: map[string]string{}}
	snap := couchtty.ResourceInventorySnapshot{InventoryKnown: true}
	for i := 0; i < couchcore.ResourceMaxSlots; i++ {
		snap.Rows = append(snap.Rows, couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: "s", Tag: couchcore.ThreadTag(fmt.Sprint(i))}, State: couchcore.ThreadParked})
	}
	live := couchcore.ThreadAddress{RepoScope: "s", Tag: "live"}
	snap.Rows = append(snap.Rows, couchcore.ActionableThreadSummary{Address: live, State: couchcore.ThreadLive})
	inv.refresh(context.Background(), snap)
	found := false
	for _, row := range inv.slots() {
		found = found || row.Thread == live
	}
	if !found {
		t.Fatal("current live row silently omitted by cap")
	}
	inventory := inv.snapshot()
	if inventory.Complete || inventory.Omitted != 1 {
		t.Fatalf("cap concealed incompleteness: %+v", inventory)
	}
	sampler := testResourceSampler(&samplerReader{})
	sampler.slots = inv.snapshot
	sampler.pass(context.Background())
	if !strings.Contains(strings.Join(sampler.view().Errors, " "), "omitted 1 rows") {
		t.Fatal("query concealed omitted rows")
	}

}

func TestResourceInventoryCanceledProofRetainsPublishedMetadata(t *testing.T) {
	entered, done := make(chan struct{}), make(chan struct{})
	inv := &resourceInventory{probe: func(ctx context.Context, _ couchcore.ThreadAddress) (couchcore.PairSessionBinding, error) {
		close(entered)
		<-ctx.Done()
		return couchcore.PairSessionBinding{}, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := couchcore.ThreadAddress{RepoScope: "s", Tag: "a"}
	go func() {
		defer close(done)
		inv.refresh(ctx, couchtty.ResourceInventorySnapshot{InventoryKnown: true, Rows: []couchcore.ActionableThreadSummary{{Address: a}}})
	}()
	<-entered
	if s := inv.snapshot(); s.Complete || len(s.Slots) != 1 {
		t.Fatalf("pending inventory unavailable: %+v", s)
	}
	cancel()
	<-done
	if s := inv.snapshot(); s.Complete || len(s.Slots) != 1 {
		t.Fatalf("cancellation proved absence: %+v", s)
	}
}
