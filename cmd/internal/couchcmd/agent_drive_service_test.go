package couchcmd

import (
	"context"
	"errors"
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
