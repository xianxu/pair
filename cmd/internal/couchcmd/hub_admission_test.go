package couchcmd

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// hubAdmissionRig wires admission's warm sources as production does: the
// fresh resolver behind the memo, counted at the two edges that spawn
// processes in production (workspace resolution runs sdlc; the target
// resolver runs Discover's sdlc and git probes).
type hubAdmissionRig struct {
	*serviceRig
	runner    *fakeSlotRunner
	caller    couchmessage.Binding
	spawns    atomic.Int64
	resolve   atomic.Pointer[func(context.Context, string) (couchcore.HubSlot, error)]
	guardSeen func(context.Context) error
}

func newHubAdmissionRig(t *testing.T) *hubAdmissionRig {
	t.Helper()
	r, runner, _ := slotRig(t)
	h := &hubAdmissionRig{serviceRig: r, runner: runner}
	h.caller = r.connect(3) // a hub CLI request comes from the connected hub slot
	r.world.set(h.caller, func(s *worldSlot) { n := 3; s.workspace.PrimaryRoot = "/repo"; s.workspace.Slot = &n })
	r.s.mu.Lock()
	r.s.workspaces[h.caller] = r.world.slots[h.caller.Thread()].workspace
	r.s.mu.Unlock()
	r.slotOps.names = func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: "/repo", Dir: "pair"}}, nil
	}
	r.s.authority.hubs = guardHubState(t)
	workspace := r.s.authority.workspace
	r.s.authority.workspace = func(ctx context.Context, root string) (couchcore.WorkspaceIdentity, error) {
		h.spawns.Add(1)
		return workspace(ctx, root)
	}
	h.setResolver(func(context.Context, string) (couchcore.HubSlot, error) {
		return couchcore.HubSlot{Repository: "/repo", Number: 5}, nil
	})
	r.s.hubTargets = newHubTargetMemo(r.s.lifetime, func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
		h.spawns.Add(1)
		return (*h.resolve.Load())(ctx, ref)
	}, nil)
	r.s.authority.hubTarget = r.s.hubTargets.Fresh
	r.slotOps.guardedRun = func(key, op, target string, opts couchcore.LiveRestartOptions, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		h.guardSeen = guard
		return runner.run(key, op, target, opts, started, progress, finished)
	}
	return h
}

func (h *hubAdmissionRig) setResolver(f func(context.Context, string) (couchcore.HubSlot, error)) {
	h.resolve.Store(&f)
}

func (h *hubAdmissionRig) admit(id string) couchmessage.Response {
	return h.admitOp("resume", id)
}

func (h *hubAdmissionRig) admitOp(op, id string) couchmessage.Response {
	req := slotRequest(h.caller, op, id, "pair:5")
	req.Confirmed = op != "resume"
	return switchCall(h.t, h.serviceRig, req)
}

// TestHubSlotAdmissionSpawnsNothingWhenWarm is the hub's requirement: with a
// warm memo and a connected caller, admission runs no sdlc or git process;
// the queued guard still re-proves authority on fresh discovery.
func TestHubSlotAdmissionSpawnsNothingWhenWarm(t *testing.T) {
	for _, op := range []string{"resume", "relaunch"} {
		t.Run(op, func(t *testing.T) { testHubSlotAdmissionWarm(t, op) })
	}
}

func testHubSlotAdmissionWarm(t *testing.T, op string) {
	h := newHubAdmissionRig(t)
	if _, err := h.s.hubTargets.Fresh(t.Context(), "pair:5"); err != nil {
		t.Fatal(err)
	}
	h.spawns.Store(0)
	if got := h.admitOp(op, "warm"); got.Code != "accepted" {
		t.Fatalf("warm %s admission %+v", op, got)
	}
	if n := h.spawns.Load(); n != 0 {
		t.Fatalf("warm admission spawned %d sdlc/git resolutions", n)
	}
	if h.guardSeen == nil {
		t.Fatal("no queued guard")
	}
	if err := h.guardSeen(t.Context()); err != nil {
		t.Fatalf("queued guard: %v", err)
	}
	if h.spawns.Load() == 0 {
		t.Fatal("the queued guard did not re-prove authority on fresh discovery")
	}
}

// TestHubSlotAdmissionColdRefusesWarmingThenAdmits: a resolver slower than
// the budget yields a typed refusal inside the budget with nothing queued;
// once that resolution finishes, the next admission admits.
func TestHubSlotAdmissionColdRefusesWarmingThenAdmits(t *testing.T) {
	h := newHubAdmissionRig(t)
	release := make(chan struct{})
	h.setResolver(func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
		select {
		case <-release:
			return couchcore.HubSlot{Repository: "/repo", Number: 5}, nil
		case <-ctx.Done():
			return couchcore.HubSlot{}, ctx.Err()
		}
	})
	start := time.Now()
	got := h.admit("cold")
	elapsed := time.Since(start)
	if got.Code != couchmessage.CodeWarming || got.RetryAfterMillis <= 0 || got.Operation != nil || !strings.Contains(got.Error, "warming") || !strings.Contains(got.Error, "target") {
		t.Fatalf("cold admission %+v", got)
	}
	if elapsed >= couchmessage.AdmissionTimeout {
		t.Fatalf("refusal took %v, past the 2 s exchange", elapsed)
	}
	if h.runner.count() != 0 || h.guardSeen != nil {
		t.Fatal("a refused admission queued work")
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if got = h.admit("warmed"); got.Code == "accepted" || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got.Code != "accepted" {
		t.Fatalf("after the resolution finished: %+v", got)
	}
}

// TestHubSlotAdmissionStaleMemoRefusesBeforeAnyEffect: the memo still names
// a same-hub slot after the reference moved to another hub; admission admits
// on the warm source, and the queued guard refuses on fresh discovery.
func TestHubSlotAdmissionStaleMemoRefusesBeforeAnyEffect(t *testing.T) {
	h := newHubAdmissionRig(t)
	if _, err := h.s.hubTargets.Fresh(t.Context(), "pair:5"); err != nil {
		t.Fatal(err)
	}
	h.setResolver(func(context.Context, string) (couchcore.HubSlot, error) {
		return couchcore.HubSlot{Repository: "/repo", Number: 2}, nil // owned by the other hub
	})
	if got := h.admit("stale"); got.Code != "accepted" {
		t.Fatalf("stale admission %+v", got)
	}
	if err := h.guardSeen(t.Context()); err == nil || !strings.Contains(err.Error(), "belongs to hub") {
		t.Fatalf("queued guard on fresh discovery: %v", err)
	}
}
