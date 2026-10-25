package couchcmd

import (
	"context"
	"fmt"
	"strings"
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

// Guards exercise admission's memo and execution's fresh source separately,
// as newMessageService does when authority is supplied at construction.
func wireGuardTestMemo(r *serviceRig, resolve func(context.Context, string) (couchcore.HubSlot, error)) {
	r.s.hubTargets = newHubTargetMemo(r.s.lifetime, resolve, nil)
	r.s.authority.hubTarget = r.s.hubTargets.Fresh
}
func TestHubSwitchMutationAdmissionAndQueuedMove(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(map[bool]string{false: "admission", true: "queued"}[queued], func(t *testing.T) {
			h := guardHubState(t)
			if queued {
				h.state, _ = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
			}
			r, q, hub, worker, _ := leanSwitchRig(t, func(a messageAuthority) messageAuthority {
				a.hubs = h
				return a
			})
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
	for _, op := range []string{"resume", "reboot", "reap", "relaunch", "reload-context", "prepare", "recover-park"} {
		t.Run(op, func(t *testing.T) {
			r, runner, _ := slotRig(t, func(a messageAuthority) messageAuthority {
				a.hubs = guardHubState(t)
				return a
			})
			caller := r.world.add(3)
			r.world.set(caller, func(s *worldSlot) { n := 3; s.workspace.PrimaryRoot = "/repo"; s.workspace.Slot = &n })
			// Register only after the world has its canonical slot facts, then
			// wait for the exact connected binding's cached workspace proof.
			r.attach(caller, "caller")
			r.wrapper(caller)
			r.waitConnected(caller, true)
			r.waitWorkspace(caller)
			r.slotOps.names = func(context.Context) ([]couchcore.RepositoryName, error) {
				return []couchcore.RepositoryName{{Key: "/repo", Dir: "pair"}}, nil
			}
			wireGuardTestMemo(r, func(context.Context, string) (couchcore.HubSlot, error) {
				return couchcore.HubSlot{Repository: "/repo", Number: 2}, nil
			})
			req := slotRequest(caller, op, "guarded", "pair:2")
			req.Confirmed = true
			got := switchCall(t, r, req)
			if got.Code != "refused" || !strings.Contains(got.Error, "belongs to hub") || runner.count() != 0 {
				t.Fatalf("cross-hub %s admitted: %+v", op, got)
			}
			h := r.s.authority.hubs
			h.mu.Lock()
			h.state, _ = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
			h.mu.Unlock()
			var check func(context.Context) error
			r.slotOps.guardedRun = func(key, op, target string, opts couchcore.LiveRestartOptions, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
				check = guard
				return runner.run(key, op, target, opts, started, progress, finished)
			}
			got = switchCall(t, r, req)
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
			h.mu.Lock()
			h.state, _ = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo0", Target: "/repo1", Revision: h.state.Revision()})
			h.mu.Unlock()
			if err := check(context.Background()); err == nil {
				t.Fatal("moved target still authorized")
			}
		})
	}
}

func TestHubSwitchReceiptScope(t *testing.T) {
	r, _, main, _, _ := leanSwitchRig(t, func(a messageAuthority) messageAuthority {
		a.hubs = guardHubState(t)
		return a
	})
	other := r.connect(1)
	r.waitWorkspace(main)
	r.waitWorkspace(other)
	wireGuardTestMemo(r, func(context.Context, string) (couchcore.HubSlot, error) {
		return couchcore.HubSlot{Repository: "/repo", Number: 2}, nil
	})
	r.slotOps.mu.Lock()
	r.slotOps.receipts[receiptKey{"scope", "owner", "one"}] = couchmessage.OperationReceipt{ID: "one", Op: "switch-agent", Target: "pair:2", Status: couchmessage.ReceiptQueued}
	r.slotOps.mu.Unlock()
	for _, caller := range []couchmessage.Binding{main, other} {
		got := switchCall(t, r, statusRequest(caller, "one"))
		want := couchmessage.ReceiptUnknown
		if caller == other {
			want = couchmessage.ReceiptQueued
		}
		if got.Operation == nil || got.Operation.Status != want {
			t.Fatalf("receipt for %s = %+v", caller.Slot, got)
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
