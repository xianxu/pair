package couchcmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestHubMutationPayloadRechecksAfterPreparation(t *testing.T) {
	original := &struct{ ID string }{"prepared"}
	moved := false
	guard := func(context.Context) error {
		if moved {
			return errors.New("ownership changed")
		}
		return nil
	}
	payload := hubMutationPayload{original: original, guard: guard}
	got, err := unwrapHubMutation(context.Background(), payload)
	if err != nil || got != original {
		t.Fatalf("original payload %v %v", got, err)
	}
	moved = true
	if _, err := unwrapHubMutation(context.Background(), payload); err == nil {
		t.Fatal("prepared mutation bypassed ownership recheck")
	}
}
func TestHubMutationPayloadLegacyUnchanged(t *testing.T) {
	for _, original := range []any{nil, "legacy", &couchcore.OperationCall{Name: "resume"}} {
		got, err := unwrapHubMutation(context.Background(), original)
		if err != nil || got != original {
			t.Fatalf("legacy payload changed: %v %v", got, err)
		}
	}
}

func TestHubPreparedSwitchRechecksOwnershipAndSourceAtEffect(t *testing.T) {
	for _, fault := range []string{"ownership", "source"} {
		t.Run(fault, func(t *testing.T) {
			h := guardHubState(t)
			var err error
			h.state, err = h.state.Apply(couchcore.HubMove{Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision(), Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}})
			if err != nil {
				t.Fatal(err)
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
				r.s.mu.Lock()
				r.s.workspaces[b] = r.world.slots[b.Thread()].workspace
				r.s.mu.Unlock()
			}
			if got := switchCall(t, r, switchRequest(hub, "prepared", worker.Slot)); got.Code != "accepted" {
				t.Fatal(got)
			}
			call, err := q.job(0).prepare(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := call.TypedPayload.(hubMutationPayload); !ok {
				t.Fatal("switch did not retain effect-time guard")
			}
			if fault == "ownership" {
				h.mu.Lock()
				next, e := h.state.Apply(couchcore.HubMove{Caller: "/repo0", Target: "/repo1", Revision: h.state.Revision(), Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}})
				h.state = next
				h.mu.Unlock()
				if e != nil {
					t.Fatal(e)
				}
			} else {
				live, _ := r.s.LivenessForThread(worker.Scope, worker.Tag)
				settled := false
				live.Settled = &settled
				r.s.liveness.Store(&map[string]couchmessage.SlotLiveness{worker.Slot: live})
			}
			effected := false
			if _, err := dispatchHubOperation(h, call, func(couchcore.OperationCall) (any, error) { effected = true; return nil, nil }); err == nil {
				t.Fatal("prepared switch executed with stale authority")
			}
			if effected {
				t.Fatal("effect ran before authority check")
			}
		})
	}
}

func TestCandidateResumeRechecksHubAuthorityBeforePreparationAndEffect(t *testing.T) {
	for _, refuseAt := range []int{1, 2} {
		t.Run(map[int]string{1: "prepare", 2: "effect"}[refuseAt], func(t *testing.T) {
			r := newColdResumeRig(t)
			hubs := guardHubState(t)
			guardCalls, effects := 0, 0
			guard := func(context.Context) error {
				guardCalls++
				if guardCalls == refuseAt {
					return errors.New("ownership changed")
				}
				return nil
			}
			r.console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				return dispatchHubOperation(hubs, call, func(couchcore.OperationCall) (any, error) {
					effects++
					return nil, nil
				})
			})
			done := make(chan error, 1)
			ops := consoleSlotOperations(r.console, r.c, nil)
			if err := ops.candidate("guarded-candidate", r.address, "candidate-token", func() {}, nil, func(_ any, err error) { done <- err }, guard); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "ownership changed") {
					t.Fatalf("candidate resume refusal: %v", err)
				}
			case <-testwait.After(t):
				t.Fatal("candidate resume did not finish")
			}
			if guardCalls != refuseAt || effects != 0 {
				t.Fatalf("candidate bypassed authority: guards=%d effects=%d", guardCalls, effects)
			}
		})
	}
}
