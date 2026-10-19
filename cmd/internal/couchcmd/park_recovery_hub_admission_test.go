package couchcmd

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Keep the interrupted target separate from the hub's own canonical slot:
// HubOwnership deliberately forbids moving a hub slot. Both identities are
// private fixture identities, and the caller's verified root remains unchanged.
func movableParkHub(t *testing.T, r *parkAdmissionRig) (string, couchcore.HubSlot) {
	t.Helper()
	primary := r.before.StartingPath
	other := filepath.Join(filepath.Dir(primary), "other-hub")
	state, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{
		{Root: primary, Workspace: filepath.Dir(primary), Slot: couchcore.HubSlot{Repository: primary, Number: 1}},
		{Root: other, Workspace: filepath.Dir(primary), Slot: couchcore.HubSlot{Repository: other, Number: 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	h := r.service.s.authority.hubs
	h.mu.Lock()
	h.state = state
	h.mu.Unlock()
	return other, couchcore.HubSlot{Repository: primary, Number: 0}
}

func moveParkToOtherHub(t *testing.T, r *parkAdmissionRig, other string, slot couchcore.HubSlot) {
	t.Helper()
	h := r.service.s.authority.hubs
	h.mu.Lock()
	next, err := h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{slot}, Caller: r.before.StartingPath, Target: other, Revision: h.state.Revision()})
	if err == nil {
		h.state = next
	}
	h.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
}

func assertParkHubUnchanged(t *testing.T, r *parkAdmissionRig) {
	t.Helper()
	after, err := r.couch.Threads.GetThread(r.before.Address)
	if err != nil || after.Revision != r.before.Revision || after.Park == nil || after.Park.Identity != r.before.Park.Identity || r.effects.Load() != 0 {
		t.Fatalf("unauthorized recovery changed park: %+v err=%v effects=%d", after, err, r.effects.Load())
	}
}

func TestRecoverParkHubMissingCallerCache(t *testing.T) {
	r := newParkAdmissionRig(t, false)
	r.service.s.mu.Lock()
	delete(r.service.s.workspaces, r.caller)
	r.service.s.mu.Unlock()
	req := r.request("missing-caller-cache")
	got := r.call(t, req)
	if got.Code != "refused" || !strings.Contains(got.Error, "refresh") || !strings.Contains(got.Error, "retry") || got.Operation != nil || r.catalog.probes.Load() != 0 || r.callerProbes.Load() != 0 {
		t.Fatalf("missing cache: %+v probes=%d caller=%d", got, r.catalog.probes.Load(), r.callerProbes.Load())
	}
	status := r.call(t, statusRequest(r.caller, req.ID))
	if status.Operation == nil || status.Operation.Status != couchmessage.ReceiptUnknown {
		t.Fatalf("refusal allocated receipt: %+v", status)
	}
	assertParkHubUnchanged(t, r)
}

func TestRecoverParkHubCrossOwnerAdmission(t *testing.T) {
	r := newParkAdmissionRig(t, false)
	other, slot := movableParkHub(t, r)
	moveParkToOtherHub(t, r, other, slot)
	req := r.request("cross-owner")
	got := r.call(t, req)
	if got.Code != "refused" || !strings.Contains(got.Error, "ask that hub") || got.Operation != nil || r.catalog.probes.Load() != 0 || r.callerProbes.Load() != 0 {
		t.Fatalf("cross-hub admission: %+v probes=%d", got, r.catalog.probes.Load())
	}
	status := r.call(t, statusRequest(r.caller, req.ID))
	if status.Operation == nil || status.Operation.Status != couchmessage.ReceiptUnknown {
		t.Fatalf("cross-owner receipt: %+v", status)
	}
	assertParkHubUnchanged(t, r)
}

func TestRecoverParkHubQueuedAuthorityChanges(t *testing.T) {
	for _, dispatch := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			name := "queued"
			if dispatch {
				name = "dispatch"
			}
			if replace {
				name += "/caller"
			} else {
				name += "/owner"
			}
			t.Run(name, func(t *testing.T) {
				r := newParkAdmissionRig(t, dispatch)
				other, slot := movableParkHub(t, r)
				req := r.request("authority-changed")
				if got := r.call(t, req); got.Code != "accepted" {
					t.Fatalf("admission: %+v", got)
				}
				awaitParkAdmission(t, r.catalog.entered, "queued workspace probe")
				if dispatch {
					r.release()
					awaitParkAdmission(t, r.dispatched, "prepared dispatch")
				}
				if replace {
					r.service.world.set(r.caller, func(s *worldSlot) { s.binding.Nonce = "replacement" })
				} else {
					moveParkToOtherHub(t, r, other, slot)
				}
				r.release()
				r.releaseOnce.Do(func() { close(r.dispatchRelease) })
				// Observe the private receipt store directly when the authenticating
				// caller has been replaced; the stale caller must not regain wire access.
				var got couchmessage.OperationReceipt
				testwait.Until(t, "authority refusal receipt", func() bool {
					ops := r.service.slotOps
					ops.mu.Lock()
					got = ops.receipts[receiptKey{r.caller.Scope, r.caller.Tag, req.ID}]
					ops.mu.Unlock()
					return got.Status == couchmessage.ReceiptFailed || got.Status == couchmessage.ReceiptRefused || got.Status == couchmessage.ReceiptSucceeded
				})
				if got.Status == couchmessage.ReceiptSucceeded || got.Detail == "" {
					t.Fatalf("changed authority: %+v", got)
				}
				if !replace && !strings.Contains(got.Detail, "ask that hub") {
					t.Fatalf("wrong ownership refusal: %+v", got)
				}
				assertParkHubUnchanged(t, r)
			})
		}
	}
}

func TestRecoverParkHubRetainedReceiptSkipsUnavailableTargets(t *testing.T) {
	r := newParkAdmissionRig(t, false)
	req := r.request("retained-unavailable")
	accepted := r.call(t, req)
	if accepted.Code != "accepted" || accepted.Operation == nil {
		t.Fatalf("admission: %+v", accepted)
	}
	awaitParkAdmission(t, r.catalog.entered, "queued workspace probe")
	var targetProbes atomic.Int32
	r.service.s.authority.hubTarget = func(context.Context, string) (couchcore.HubSlot, error) {
		targetProbes.Add(1)
		return couchcore.HubSlot{}, errors.New("target authority unavailable")
	}
	r.catalog.mu.Lock()
	r.catalog.Errors = map[string]error{".": errors.New("catalog unavailable"), r.before.StartingPath: errors.New("catalog unavailable")}
	r.catalog.mu.Unlock()
	probes, callerProbes := r.catalog.probes.Load(), r.callerProbes.Load()
	duplicate := r.call(t, req)
	if duplicate.Code != "accepted" || duplicate.Operation == nil || duplicate.Operation.ID != accepted.Operation.ID {
		t.Fatalf("retained duplicate: %+v", duplicate)
	}
	conflict := req
	conflict.Target = "no-longer-enrolled:9"
	got := r.call(t, conflict)
	if got.Code != "id-conflict" {
		t.Fatalf("retained conflict: %+v", got)
	}
	if r.catalog.probes.Load() != probes || r.callerProbes.Load() != callerProbes || targetProbes.Load() != 0 {
		t.Fatalf("retained request probed unavailable target: workspace=%d/%d caller=%d/%d target=%d", r.catalog.probes.Load(), probes, r.callerProbes.Load(), callerProbes, targetProbes.Load())
	}
	assertParkHubUnchanged(t, r)
}
