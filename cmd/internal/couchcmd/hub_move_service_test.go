package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type hubMoveRig struct {
	*serviceRig
	queue     *leanSwitchQueue
	hubs      *hubRuntime
	defs      []couchcore.HubDefinition
	callers   []couchmessage.Binding
	worker    couchmessage.Binding
	published atomic.Int32
}

func newHubMoveRig(t *testing.T) *hubMoveRig {
	h, defs, _ := moveFixture(t)
	r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}}
	r.slotOps = newSlotOperations(nil, nil, time.Now)
	r.decorateAuthority = func(a messageAuthority) messageAuthority {
		a.hubs = h
		a.hubTarget = func(_ context.Context, ref string) (couchcore.HubSlot, error) {
			if !strings.HasPrefix(ref, "pair:") {
				return couchcore.HubSlot{}, fmt.Errorf("unknown target %s", ref)
			}
			n, err := strconv.Atoi(strings.TrimPrefix(ref, "pair:"))
			if err != nil {
				return couchcore.HubSlot{}, err
			}
			return couchcore.HubSlot{Repository: filepath.Join(defs[0].Workspace, "pair"), Number: n}, nil
		}
		return a
	}
	r.init()
	x := &hubMoveRig{serviceRig: r, queue: &leanSwitchQueue{}, hubs: h, defs: defs}
	x.install(hubClaimFiles{})
	for i, def := range defs {
		b := r.world.add(i)
		r.world.set(b, func(s *worldSlot) {
			n := 0
			s.root = def.Root
			s.workspace.WorktreeRoot = def.Root
			s.workspace.PrimaryRoot = def.Slot.Repository
			s.workspace.Slot = &n
			s.workspace.FleetRoot = def.Workspace
		})
		r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("hub-%d", i)))
		r.wrapper(b)
		r.waitConnected(b, true)
		r.waitWorkspace(b)
		x.callers = append(x.callers, b)
	}
	x.worker = r.connect(7)
	return x
}
func (r *hubMoveRig) install(files hubClaimFiles) {
	r.s.hubMoves.Store(&hubMoveService{files: files, publish: func(couchcore.HubOwnership) { r.published.Add(1) }, enqueue: func(key, op string, p func(context.Context) (couchcore.OperationCall, error), start func(), finish func(any, error)) error {
		if op != "move-slot" {
			return fmt.Errorf("unexpected operation %s", op)
		}
		return r.queue.enqueue(key, p, start, finish)
	}})
}
func (r *hubMoveRig) request(caller int, id, target string, to int, persist bool) couchmessage.Request {
	req := callerRequest(r.callers[caller], "move-slot")
	req.ID = id
	req.Target = target
	req.ToHub = r.defs[to].Root
	req.Persist = persist
	return req
}
func (r *hubMoveRig) finish(index int) error {
	job := r.queue.job(index)
	job.started()
	call, err := job.prepare(context.Background())
	var value any
	if err == nil {
		call.Context = context.Background()
		value, err = executeHubMove(call)
	}
	job.finished(value, err)
	return err
}
func (r *hubMoveRig) owner(n int) string {
	return r.hubs.snapshot().Resolve(couchcore.HubSlot{Repository: filepath.Join(r.defs[0].Workspace, "pair"), Number: n}).Hub
}
func (r *hubMoveRig) call(req couchmessage.Request) couchmessage.Response {
	return switchCall(r.t, r.serviceRig, req)
}
func (r *hubMoveRig) assertStatus(caller int, id string, want couchmessage.ReceiptStatus) {
	r.t.Helper()
	got := r.call(statusRequest(r.callers[caller], id))
	if got.Operation == nil || got.Operation.Status != want {
		r.t.Fatalf("status %s = %+v want %v", id, got, want)
	}
}

func TestHubMoveSocketRolesAndBatchAtomicity(t *testing.T) {
	r := newHubMoveRig(t)
	worker := r.request(0, "worker", "pair:7", 1, false)
	worker.Scope = r.worker.Scope
	worker.Tag = r.worker.Tag
	worker.Session = r.worker.Session
	worker.Nonce = r.worker.Nonce
	if got := r.call(worker); got.Code != "refused" {
		t.Fatalf("worker admitted: %+v", got)
	}
	// An extra hub can take main's slot, then move its own slot.
	if got := r.call(r.request(1, "take", "pair:7", 1, false)); got.Code != "accepted" {
		t.Fatal(got)
	}
	if err := r.finish(0); err != nil {
		t.Fatal(err)
	}
	if got := r.call(r.request(1, "give", "pair:7", 2, false)); got.Code != "accepted" {
		t.Fatal(got)
	}
	if err := r.finish(1); err != nil {
		t.Fatal(err)
	}
	before := r.hubs.snapshot().Revision()
	// First member belongs to main, second to a third hub: reject the whole batch.
	if got := r.call(r.request(1, "batch", "pair:8,pair:7", 1, false)); got.Code != "refused" {
		t.Fatalf("foreign batch admitted %+v", got)
	}
	if r.queue.count() != 2 || r.owner(8) != r.defs[0].Root || r.owner(7) != r.defs[2].Root || r.hubs.snapshot().Revision() != before {
		t.Fatal("rejected batch changed state")
	}
}

func TestHubMoveSocketQueuedStaleAuthority(t *testing.T) {
	for _, fault := range []string{"revision", "caller", "disconnect", "caller-after-prepare"} {
		t.Run(fault, func(t *testing.T) {
			r := newHubMoveRig(t)
			if got := r.call(r.request(1, "queued", "pair:7", 1, false)); got.Code != "accepted" {
				t.Fatal(got)
			}
			r.assertStatus(1, "queued", couchmessage.ReceiptQueued)
			var prepared couchcore.OperationCall
			if fault == "caller-after-prepare" {
				var err error
				prepared, err = r.queue.job(0).prepare(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				prepared.Context = context.Background()
			}
			switch fault {
			case "revision":
				h := r.hubs
				h.mu.Lock()
				next, err := h.state.Apply(couchcore.HubMove{Caller: r.defs[0].Root, Target: r.defs[2].Root, Revision: h.state.Revision(), Slots: []couchcore.HubSlot{{Repository: filepath.Join(r.defs[0].Workspace, "pair"), Number: 8}}})
				h.state = next
				h.mu.Unlock()
				if err != nil {
					t.Fatal(err)
				}
			case "caller", "caller-after-prepare":
				r.world.set(r.callers[1], func(s *worldSlot) { s.binding.Nonce = "replacement" })
			case "disconnect":
				r.disconnect(r.callers[1])
			}
			if fault == "caller-after-prepare" {
				job := r.queue.job(0)
				job.started()
				value, err := executeHubMove(prepared)
				job.finished(value, err)
				if err == nil {
					t.Fatal("caller revoked after prepare still moved slot")
				}
			} else if err := r.finish(0); err == nil {
				t.Fatal("stale queued move succeeded")
			}
			if r.owner(7) != r.defs[0].Root || r.published.Load() != 0 {
				t.Fatal("stale move changed projection")
			}
			// Restore record proof for status-only recovery; registration is not required.
			if fault == "caller" || fault == "caller-after-prepare" {
				r.world.set(r.callers[1], func(s *worldSlot) { s.binding.Nonce = r.callers[1].Nonce })
			}
			if fault != "disconnect" {
				r.assertStatus(1, "queued", couchmessage.ReceiptFailed)
			}
		})
	}
}

func TestHubMoveSocketPersistenceFailurePreservesRuntime(t *testing.T) {
	r := newHubMoveRig(t)
	r.install(hubClaimFiles{write: func(string, []byte, os.FileMode) error { return errors.New("disk full") }})
	if got := r.call(r.request(1, "persist", "pair:7", 1, true)); got.Code != "accepted" {
		t.Fatal(got)
	}
	if err := r.finish(0); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("write failure %v", err)
	}
	r.assertStatus(1, "persist", couchmessage.ReceiptFailed)
	if r.owner(7) != r.defs[0].Root || r.published.Load() != 0 {
		t.Fatal("failed persistence moved slot")
	}
}

func TestHubMoveSocketReceiptRepeatAndIntent(t *testing.T) {
	r := newHubMoveRig(t)
	req := r.request(0, "move", "pair:7", 1, false)
	for i := 0; i < 2; i++ {
		if got := r.call(req); got.Code != "accepted" {
			t.Fatal(got)
		}
	}
	if r.queue.count() != 1 {
		t.Fatal("duplicate queued twice")
	}
	changed := req
	changed.Persist = true
	if got := r.call(changed); got.Code != "id-conflict" {
		t.Fatalf("persist reused ID: %+v", got)
	}
	changed = req
	changed.ToHub = r.defs[2].Root
	if got := r.call(changed); got.Code != "id-conflict" {
		t.Fatalf("destination reused ID: %+v", got)
	}
	if err := r.finish(0); err != nil {
		t.Fatal(err)
	}
	r.assertStatus(0, "move", couchmessage.ReceiptSucceeded)
	if got := r.call(req); got.Code != "accepted" || got.Operation == nil || got.Operation.Status != couchmessage.ReceiptSucceeded {
		t.Fatalf("lost reply retry: %+v", got)
	}
	if r.queue.count() != 1 || r.published.Load() != 1 {
		t.Fatal("repeat performed another effect")
	}
}

func TestHubMoveSocketEphemeralThenNewPersistentRequest(t *testing.T) {
	r := newHubMoveRig(t)
	if got := r.call(r.request(1, "ephemeral", "pair:7", 1, false)); got.Code != "accepted" {
		t.Fatal(got)
	}
	if err := r.finish(0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.defs[1].Root, ".hub")); !os.IsNotExist(err) {
		t.Fatalf("ephemeral file = %v", err)
	}
	if got := r.call(r.request(1, "durable", "pair:7", 1, true)); got.Code != "accepted" {
		t.Fatal(got)
	}
	if err := r.finish(1); err != nil {
		t.Fatal(err)
	}
	r.assertStatus(1, "durable", couchmessage.ReceiptSucceeded)
	raw, err := os.ReadFile(filepath.Join(r.defs[1].Root, ".hub"))
	if err != nil || string(raw) != "pair:7\n" {
		t.Fatalf("durable claims %q %v", raw, err)
	}
	// Reconstruct from disk: no runtime overrides participate.
	defs := append([]couchcore.HubDefinition(nil), r.defs...)
	for i := range defs {
		doc, err := ReadHubClaimFile(defs[i].Root)
		if err != nil {
			t.Fatal(err)
		}
		defs[i].Claims = doc
	}
	state, err := couchcore.NewHubOwnership(defs)
	if err != nil {
		t.Fatal(err)
	}
	if owner := state.Resolve(couchcore.HubSlot{Repository: filepath.Join(defs[0].Workspace, "pair"), Number: 7}).Hub; owner != defs[1].Root {
		t.Fatalf("restart owner %s", owner)
	}
}

// TestHubMoveAdmissionUsesWarmSlotsAndRechecksFresh (pair#726 BR-2): move
// admission names its slots from the memo with no fresh resolution, and the
// queued job refuses if a fresh resolution no longer matches.
func TestHubMoveAdmissionUsesWarmSlotsAndRechecksFresh(t *testing.T) {
	r := newHubMoveRig(t)
	resolve := r.s.authority.hubTarget
	var resolutions atomic.Int64
	shift := atomic.Bool{}
	r.s.hubTargets = newHubTargetMemo(r.s.lifetime, func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
		resolutions.Add(1)
		slot, err := resolve(ctx, ref)
		if shift.Load() {
			slot.Number += 2
		}
		return slot, err
	}, nil)
	r.s.authority.hubTarget = r.s.hubTargets.Fresh
	if _, err := r.s.hubTargets.Fresh(t.Context(), "pair:7"); err != nil {
		t.Fatal(err)
	}
	resolutions.Store(0)
	if got := r.call(r.request(1, "take", "pair:7", 1, false)); got.Code != "accepted" {
		t.Fatalf("warm move admission %+v", got)
	}
	if n := resolutions.Load(); n != 0 {
		t.Fatalf("warm move admission ran %d fresh resolutions", n)
	}
	shift.Store(true)
	if _, err := r.queue.job(0).prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "resolved differently") {
		t.Fatalf("queued move with a moved slot: %v", err)
	}
	if r.owner(7) != r.defs[0].Root {
		t.Fatal("a refused move changed ownership")
	}
}
