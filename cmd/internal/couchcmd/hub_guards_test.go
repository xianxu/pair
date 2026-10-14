package couchcmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func guardHubState(t *testing.T) *hubRuntime {
	t.Helper()
	claims, err := couchcore.ParseHubClaims([]byte("repo:2\n"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{
		{Root: "/repo0", Workspace: "/", Slot: couchcore.HubSlot{Repository: "/repo", Number: 0}},
		{Root: "/repo1", Workspace: "/", Slot: couchcore.HubSlot{Repository: "/repo", Number: 1}, Claims: claims},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &hubRuntime{state: state}
}
func TestHubSwitchMutationAdmissionAndQueuedMove(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(map[bool]string{false: "admission", true: "queued"}[queued], func(t *testing.T) {
			r, q, hub, worker, _ := leanSwitchRig(t)
			h := guardHubState(t)
			if queued {
				h.state, _ = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
			}
			r.s.authority.hubs = h
			for _, b := range []couchmessage.Binding{hub, worker} {
				n := 0
				if b == worker {
					n = 2
				}
				r.world.set(b, func(s *worldSlot) { s.workspace.PrimaryRoot = "/repo"; s.workspace.Slot = &n })
			}
			// Refresh cached identities after adding the canonical slot facts.
			r.s.mu.Lock()
			for _, b := range []couchmessage.Binding{hub, worker} {
				r.s.workspaces[b] = r.world.slots[b.Thread()].workspace
			}
			r.s.mu.Unlock()
			got := switchCall(t, r, switchRequest(hub, "guarded", worker.Slot))
			if !queued {
				if got.Code == "accepted" || q.count() != 0 {
					t.Fatalf("cross-hub switch admitted: %+v", got)
				}
				return
			}
			if got.Code != "accepted" {
				t.Fatalf("same-hub refused: %+v", got)
			}
			h.mu.Lock()
			next, err := h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo0", Target: "/repo1", Revision: h.state.Revision()})
			h.state = next
			h.mu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := q.job(0).prepare(context.Background()); err == nil {
				t.Fatal("queued switch crossed moved ownership")
			}
		})
	}
}

func TestHubSlotMutationGuardsEveryOperationAndQueuedMove(t *testing.T) {
	for _, op := range []string{"resume", "reboot", "reap", "relaunch", "reload-context", "prepare"} {
		t.Run(op, func(t *testing.T) {
			r, runner, _ := slotRig(t)
			caller := r.world.add(3)
			r.world.set(caller, func(s *worldSlot) { n := 3; s.workspace.PrimaryRoot = "/repo"; s.workspace.Slot = &n })
			r.s.authority.hubs = guardHubState(t)
			r.s.authority.hubTarget = func(context.Context, string) (couchcore.HubSlot, error) {
				return couchcore.HubSlot{Repository: "/repo", Number: 2}, nil
			}
			req := slotRequest(caller, op, "guarded", "pair:2")
			req.Confirmed = true
			// Live-pane recovery callers need no messaging registration.
			got := r.s.handleHubSlotOperation(context.Background(), caller, req)
			if got.Code != "refused" || runner.count() != 0 {
				t.Fatalf("cross-hub %s admitted: %+v", op, got)
			}
			h := r.s.authority.hubs
			h.state, _ = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
			var check func(context.Context) error
			r.slotOps.guardedRun = func(key, op, target string, opts couchcore.LiveRestartOptions, guard func(context.Context) error, started func(), finished func(any, error)) error {
				check = guard
				return runner.run(key, op, target, opts, started, finished)
			}
			got = r.s.handleHubSlotOperation(context.Background(), caller, req)
			if got.Code != "accepted" || check == nil {
				t.Fatalf("same-hub recovery failed: %+v", got)
			}
			if err := check(context.Background()); err != nil {
				t.Fatal(err)
			}
			r.world.set(caller, func(s *worldSlot) { s.binding.Nonce = "replacement" })
			if err := check(context.Background()); err == nil {
				t.Fatal("replaced caller still authorized")
			}
			r.world.set(caller, func(s *worldSlot) { s.binding.Nonce = caller.Nonce })
			h.state, _ = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo0", Target: "/repo1", Revision: h.state.Revision()})
			if err := check(context.Background()); err == nil {
				t.Fatal("moved target still authorized")
			}
		})
	}
}

func TestHubSwitchReceiptScope(t *testing.T) {
	r, _, clock := slotRig(t)
	r.s.authority.hubs = guardHubState(t)
	r.s.authority.hubTarget = func(context.Context, string) (couchcore.HubSlot, error) {
		return couchcore.HubSlot{Repository: "/repo", Number: 2}, nil
	}
	r.slotOps.receipts[receiptKey{"scope", "owner", "one"}] = couchmessage.OperationReceipt{ID: "one", Op: "switch-agent", Target: "pair:2", Status: couchmessage.ReceiptQueued, Admitted: clock.Now()}
	for _, root := range []string{"/repo0", "/repo1"} {
		_, ok := r.slotOps.switchStatus("one", r.s.hubReceiptFilter(context.Background(), couchcore.WorkspaceIdentity{WorktreeRoot: root}))
		if ok != (root == "/repo1") {
			t.Fatalf("receipt visible to %s = %v", root, ok)
		}
	}
}

func TestHubMutationOwnershipCrossProduct(t *testing.T) {
	state := guardHubState(t).snapshot()
	for _, callerN := range []int{0, 1, 2, 3} {
		for _, targetN := range []int{0, 1, 2, 3} {
			caller := couchcore.WorkspaceIdentity{PrimaryRoot: "/repo", Slot: &callerN}
			if callerN < 2 {
				caller.WorktreeRoot = fmt.Sprintf("/repo%d", callerN)
			}
			target := couchcore.HubSlot{Repository: "/repo", Number: targetN}
			allowed := (callerN == 1 || callerN == 2) == (targetN == 1 || targetN == 2)
			if err := authorizeHubMutation(caller, target, state); (err == nil) != allowed {
				t.Fatalf("caller %d target %d allowed %v err %v", callerN, targetN, allowed, err)
			}
		}
	}
	if err := authorizeHubMutation(couchcore.WorkspaceIdentity{}, couchcore.HubSlot{Repository: "/repo", Number: 3}, state); err == nil {
		t.Fatal("unknown caller permitted")
	}
}
