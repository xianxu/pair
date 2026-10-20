package couchcmd

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type driveServiceFake struct {
	serviceEndpointFake
	mu                        sync.Mutex
	observation               couchmessage.DriveObservation
	starts, cancels, statuses int
	pending, lost             bool
}

func (e *driveServiceFake) ObserveDrive(ctx context.Context) (couchmessage.DriveObservation, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.observation, ctx.Err()
}
func (e *driveServiceFake) StartDrive(ctx context.Context, id string, start couchmessage.DriveStart) (agentdrive.Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.starts++
	if e.lost {
		return agentdrive.Result{}, errors.New("lost reply")
	}
	if e.pending {
		return agentdrive.Result{}, nil
	}
	return agentdrive.Result{Outcome: agentdrive.OutcomeConfirmed, Tail: []string{"fresh composer"}}, nil
}
func (e *driveServiceFake) DriveStatus(ctx context.Context, id string) (agentdrive.Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.statuses++
	return agentdrive.Result{}, ctx.Err()
}
func (e *driveServiceFake) CancelDrive(ctx context.Context, id string) (agentdrive.Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cancels++
	return agentdrive.Result{Outcome: agentdrive.OutcomeUnconfirmed}, nil
}
func (e *driveServiceFake) counts() (int, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.starts, e.cancels
}
func driveRequest(b couchmessage.Binding, id, target string) couchmessage.Request {
	r := callerRequest(b, "keys")
	r.ID, r.Target = id, target
	r.Drive = &agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}
	return r
}
func driveRig(t *testing.T) (*serviceRig, *leanSwitchQueue, couchmessage.Binding, couchmessage.Binding, *driveServiceFake) {
	r, q, hub, worker, _ := leanSwitchRig(t)
	endpoint := &driveServiceFake{observation: couchmessage.DriveObservation{Idle: true, Instance: "wrapper", Sequence: 1}}
	r.world.mu.Lock()
	r.world.endpoints[worker] = endpoint
	r.world.mu.Unlock()
	r.s.drives.Store(&agentDriveService{enqueue: func(key, op string, deadline time.Time, p func(context.Context) (couchcore.OperationCall, error), start func(), finish func(any, error)) error {
		return q.enqueue(key, p, start, finish)
	}})
	return r, q, hub, worker, endpoint
}
func TestDriveSocketAuthorityDedupAndReceipts(t *testing.T) {
	r, q, hub, worker, e := driveRig(t)
	for _, request := range []couchmessage.Request{driveRequest(worker, "not-hub", hub.Slot), driveRequest(hub, "self", hub.Slot)} {
		if got := switchCall(t, r, request); got.Code == "accepted" {
			t.Fatalf("unauthorized %+v", got)
		}
	}
	req := driveRequest(hub, "drive", worker.Slot)
	got := switchCall(t, r, req)
	if got.Code != "accepted" || q.count() != 1 {
		t.Fatalf("%+v jobs %d", got, q.count())
	}
	job := q.job(0)
	call, err := job.prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	job.started()
	value, err := executeAgentDrive(call)
	job.finished(value, err)
	if starts, _ := e.counts(); starts != 1 {
		t.Fatalf("starts %d", starts)
	}
	r.world.set(worker, func(s *worldSlot) { s.live = false })
	again := switchCall(t, r, req)
	if again.Operation == nil || again.Operation.Status != couchmessage.ReceiptSucceeded || q.count() != 1 {
		t.Fatalf("repeat %+v", again)
	}
	req.Drive.Keys[0].Kind = agentdrive.KeyDown
	if conflict := switchCall(t, r, req); conflict.Code != "id-conflict" {
		t.Fatalf("%+v", conflict)
	}
	status := switchCall(t, r, statusRequest(hub, "drive"))
	if status.Operation == nil || status.Operation.Drive == nil || len(status.Operation.Drive.Tail) != 1 {
		t.Fatalf("status %+v", status)
	}
}
func TestDriveQueuedChangeRefusesBeforeInput(t *testing.T) {
	for _, fault := range []string{"instance", "sequence", "busy", "caller-session", "target-session"} {
		t.Run(fault, func(t *testing.T) {
			r, q, hub, worker, e := driveRig(t)
			got := switchCall(t, r, driveRequest(hub, "queued", worker.Slot))
			if got.Code != "accepted" {
				t.Fatalf("%+v", got)
			}
			switch fault {
			case "instance":
				e.observation.Instance = "replacement"
			case "sequence":
				e.observation.Sequence++
			case "busy":
				e.observation.Idle = false
			default:
				old := r.s.liveness.Load()
				next := make(map[string]couchmessage.SlotLiveness)
				for k, v := range *old {
					next[k] = v
				}
				key := worker.Slot
				if fault == "caller-session" {
					key = hub.Slot
				}
				v := next[key]
				v.Session++
				next[key] = v
				r.s.liveness.Store(&next)
			}
			if _, err := q.job(0).prepare(context.Background()); err == nil {
				t.Fatal("queued change accepted")
			}
			if starts, _ := e.counts(); starts != 0 {
				t.Fatal("input after stale proof")
			}
		})
	}
}
func TestDriveLostAdmissionReplyNeverReplayed(t *testing.T) {
	e := &driveServiceFake{lost: true}
	call := couchcore.OperationCall{Context: context.Background(), TypedPayload: agentDrivePayload{endpoint: e, id: "lost", start: couchmessage.DriveStart{Deadline: time.Now().Add(time.Second)}, callerCurrent: func(context.Context) error { return nil }}}
	value, err := executeAgentDrive(call)
	if err != nil || value.(agentdrive.Result).Outcome != agentdrive.OutcomeUnconfirmed {
		t.Fatalf("%#v %v", value, err)
	}
	starts, cancels := e.counts()
	if starts != 1 || cancels != 1 {
		t.Fatalf("starts %d cancel %d", starts, cancels)
	}
}
func TestDriveCallerLossCancelsOwnedMonitor(t *testing.T) {
	e := &driveServiceFake{pending: true}
	var checks atomic.Int32
	call := couchcore.OperationCall{Context: context.Background(), TypedPayload: agentDrivePayload{endpoint: e, id: "cancel", start: couchmessage.DriveStart{Deadline: time.Now().Add(time.Second)}, callerCurrent: func(context.Context) error {
		if checks.Add(1) == 1 {
			return nil
		}
		return errors.New("caller replaced")
	}}}
	value, err := executeAgentDrive(call)
	if err != nil || value.(agentdrive.Result).Outcome != agentdrive.OutcomeUnconfirmed {
		t.Fatalf("%#v %v", value, err)
	}
	starts, cancels := e.counts()
	if starts != 1 || cancels != 1 {
		t.Fatalf("starts %d cancel %d", starts, cancels)
	}
}

func TestDriveHubOwnershipRecheckedInsideEffectGate(t *testing.T) {
	endpoint := &driveServiceFake{observation: couchmessage.DriveObservation{Idle: true, Instance: "owned", Sequence: 4}}
	r, b := hubMessageRigWithWorkerEndpoint(t, endpoint)
	r.s.slotOps = newSlotOperations(nil, func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: "/repo/.git", Dir: "pair"}}, nil
	}, time.Now)
	q := &leanSwitchQueue{}
	r.s.drives.Store(&agentDriveService{enqueue: func(key, op string, deadline time.Time, p func(context.Context) (couchcore.OperationCall, error), started func(), finished func(any, error)) error {
		return q.enqueue(key, p, started, finished)
	}})
	// Pin positive settled evidence; this rig already has registered session tokens.
	old := r.s.liveness.Load()
	next := make(map[string]couchmessage.SlotLiveness)
	settled := true
	for k, v := range *old {
		v.Settled = &settled
		next[k] = v
	}
	r.s.liveness.Store(&next)
	if got := switchCall(t, r, driveRequest(b[0], "foreign-hub", b[2].Slot)); got.Code == "accepted" {
		t.Fatalf("cross-hub %+v", got)
	}
	got := switchCall(t, r, driveRequest(b[1], "owned-drive", b[2].Slot))
	if got.Code != "accepted" {
		t.Fatalf("owned %+v", got)
	}
	job := q.job(0)
	call, err := job.prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	call.Context = context.Background()
	h := r.s.authority.hubs
	h.mu.Lock()
	state, err := h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
	if err == nil {
		h.state = state
	}
	h.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	_, err = dispatchHubOperation(h, call, executeAgentDrive)
	if err == nil {
		t.Fatal("ownership changed after prepare but drive ran")
	}
	if starts, _ := endpoint.counts(); starts != 0 {
		t.Fatalf("starts %d", starts)
	}
}
func TestDriveAbsoluteDeadlineStopsBeforeStart(t *testing.T) {
	e := &driveServiceFake{}
	call := couchcore.OperationCall{Context: context.Background(), TypedPayload: agentDrivePayload{endpoint: e, id: "expired", start: couchmessage.DriveStart{Deadline: time.Now().Add(-time.Second)}, callerCurrent: func(context.Context) error { return nil }}}
	_, _ = executeAgentDrive(call)
	if starts, _ := e.counts(); starts != 0 {
		t.Fatal("expired operation reached wrapper start")
	}
}

func TestModelDriveScopeIntentAndDirtyCheckout(t *testing.T) {
	r, q, hub, worker, _ := driveRig(t)
	r.s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
		t.Fatal("agent input must not probe git cleanliness")
		return couchcore.SlotGitStatus{}, nil
	}
	req := callerRequest(hub, "set-model")
	req.ID, req.Target, req.Confirmed = "model-drive", worker.Slot, true
	req.Drive = &agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "gpt-6.1-sol", Effort: "medium"}}
	got := switchCall(t, r, req)
	if got.Code != "accepted" || q.count() != 1 {
		t.Fatalf("%+v", got)
	}
	req.Drive.Model.Scope = agentdrive.ScopeSession
	if got = switchCall(t, r, req); got.Code != "accepted" || q.count() != 1 {
		t.Fatalf("canonical default scope %+v", got)
	}
	req.Drive.Model.Scope = agentdrive.ScopeDefault
	if got = switchCall(t, r, req); got.Code != "id-conflict" || q.count() != 1 {
		t.Fatalf("changed scope %+v", got)
	}
}

func TestDriveRecoveryOnlyNeedsPositiveWrapperProof(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		keys        agentdrive.KeySequence
		recoverable bool
		accepted    bool
	}{
		{"clear occupied", agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}, true, true},
		{"escape and clear", agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyClearLine}, {Kind: agentdrive.KeyEsc}}, true, true},
		{"missing positive proof", agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}, false, false},
		{"generic enter", agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}, true, false},
		{"mixed clear and text", agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}, {Kind: agentdrive.KeyText, Text: "new input"}}, true, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r, q, hub, worker, e := driveRig(t)
			e.observation.Idle = false
			e.observation.Recoverable = scenario.recoverable
			old := r.s.liveness.Load()
			next := make(map[string]couchmessage.SlotLiveness)
			for k, v := range *old {
				next[k] = v
			}
			live := next[worker.Slot]
			settled := false
			live.Settled = &settled
			next[worker.Slot] = live
			r.s.liveness.Store(&next)
			req := driveRequest(hub, "recovery-proof", worker.Slot)
			req.Drive.Keys = scenario.keys
			response := switchCall(t, r, req)
			if (response.Code == "accepted") != scenario.accepted {
				t.Fatalf("response %+v", response)
			}
			if !scenario.accepted {
				if q.count() != 0 {
					t.Fatal("refused recovery queued work")
				}
				return
			}
			call, err := q.job(0).prepare(context.Background())
			if err != nil {
				t.Fatalf("occupied execution guard: %v", err)
			}
			value, err := executeAgentDrive(call)
			if err != nil || value.(agentdrive.Result).Outcome != agentdrive.OutcomeConfirmed {
				t.Fatalf("execution %#v %v", value, err)
			}
		})
	}
}
func TestDriveQueuedRecoveryRechecksProofAndIdentity(t *testing.T) {
	for _, fault := range []string{"proof lost", "sequence changed", "target session changed", "caller session changed"} {
		t.Run(fault, func(t *testing.T) {
			r, q, hub, worker, e := driveRig(t)
			e.observation.Idle = false
			e.observation.Recoverable = true
			req := driveRequest(hub, "queued-recovery", worker.Slot)
			req.Drive.Keys = agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}
			if got := switchCall(t, r, req); got.Code != "accepted" {
				t.Fatalf("admission %+v", got)
			}
			switch fault {
			case "proof lost":
				e.observation.Recoverable = false
			case "sequence changed":
				e.observation.Sequence++
			default:
				old := r.s.liveness.Load()
				next := make(map[string]couchmessage.SlotLiveness)
				for k, v := range *old {
					next[k] = v
				}
				key := worker.Slot
				if fault == "caller session changed" {
					key = hub.Slot
				}
				v := next[key]
				v.Session++
				next[key] = v
				r.s.liveness.Store(&next)
			}
			if _, err := q.job(0).prepare(context.Background()); err == nil {
				t.Fatal("queued recovery accepted stale proof")
			}
			if starts, _ := e.counts(); starts != 0 {
				t.Fatal("stale recovery wrote input")
			}
		})
	}
}

func TestModelDriveDoesNotUseRecoveryProof(t *testing.T) {
	r, q, hub, worker, e := driveRig(t)
	e.observation.Idle = false
	e.observation.Recoverable = true
	req := callerRequest(hub, "set-model")
	req.ID, req.Target, req.Confirmed = "occupied-model", worker.Slot, true
	req.Drive = &agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "gpt-6.1-sol"}}
	if response := switchCall(t, r, req); response.Code == "accepted" || q.count() != 0 {
		t.Fatalf("model used recovery-only proof: %+v", response)
	}
}

// TestDrivePanelAdmitsOnlyEsc runs the Couch admission and its pre-input
// guard over a wrapper that reports a recognized panel (pair#691).
func TestDrivePanelAdmitsOnlyEsc(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		keys     agentdrive.KeySequence
		accepted bool
	}{
		{"esc", agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}, true},
		{"clear-line", agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}, false},
		{"double esc", agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyEsc}}, false},
		{"text", agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "x"}}, false},
		{"submit", agentdrive.KeySequence{{Kind: agentdrive.KeySubmit}}, false},
		{"esc and text", agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyText, Text: "x"}}, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r, q, hub, worker, e := driveRig(t)
			e.observation.Idle = false
			e.observation.Dismissable = true
			old := r.s.liveness.Load()
			next := make(map[string]couchmessage.SlotLiveness)
			for k, v := range *old {
				next[k] = v
			}
			live := next[worker.Slot]
			settled := false
			live.Settled = &settled
			next[worker.Slot] = live
			r.s.liveness.Store(&next)
			req := driveRequest(hub, "panel", worker.Slot)
			req.Drive.Keys = scenario.keys
			response := switchCall(t, r, req)
			if (response.Code == "accepted") != scenario.accepted {
				t.Fatalf("response %+v", response)
			}
			if !scenario.accepted {
				if q.count() != 0 {
					t.Fatal("refused panel input queued work")
				}
				if !strings.Contains(response.Error, agentdrive.PanelRefusal) {
					t.Fatalf("panel refusal does not name esc: %+v", response)
				}
				return
			}
			if _, err := q.job(0).prepare(context.Background()); err != nil {
				t.Fatalf("panel execution guard: %v", err)
			}
			e.observation.Dismissable = false
			if _, err := q.job(0).prepare(context.Background()); err == nil {
				t.Fatal("guard accepted a closed panel")
			}
		})
	}
}

// TestDriveInterruptNeedsAWorkingTurnProof: the interrupt skips the settled
// check (a working turn is never settled) and is admitted only on the
// wrapper's working-turn proof, which no key request can use (pair#717).
func TestDriveInterruptNeedsAWorkingTurnProof(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		drive         agentdrive.Request
		interruptible bool
		accepted      bool
	}{
		{"interrupt on a working turn", agentdrive.Request{Interrupt: true}, true, true},
		{"interrupt with no working turn", agentdrive.Request{Interrupt: true}, false, false},
		{"esc on a working turn", agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}}, true, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r, q, hub, worker, e := driveRig(t)
			e.observation.Idle = false
			e.observation.Interruptible = scenario.interruptible
			old := r.s.liveness.Load()
			next := make(map[string]couchmessage.SlotLiveness)
			for k, v := range *old {
				next[k] = v
			}
			live := next[worker.Slot]
			settled := false
			live.Settled = &settled
			next[worker.Slot] = live
			r.s.liveness.Store(&next)
			req := driveRequest(hub, "interrupt", worker.Slot)
			drive := scenario.drive
			req.Drive = &drive
			if drive.Interrupt {
				req.Op, req.Confirmed = "interrupt", true
			}
			response := switchCall(t, r, req)
			if (response.Code == "accepted") != scenario.accepted {
				t.Fatalf("response %+v", response)
			}
			if !scenario.accepted {
				if q.count() != 0 {
					t.Fatal("refused request queued work")
				}
				if drive.Interrupt && !strings.Contains(response.Error, agentdrive.InterruptRefusal) {
					t.Fatalf("interrupt refusal %+v", response)
				}
				return
			}
			if _, err := q.job(0).prepare(context.Background()); err != nil {
				t.Fatalf("interrupt guard: %v", err)
			}
			e.observation.Interruptible = false
			if _, err := q.job(0).prepare(context.Background()); err == nil {
				t.Fatal("guard accepted an interrupt after the turn ended")
			}
		})
	}
}
