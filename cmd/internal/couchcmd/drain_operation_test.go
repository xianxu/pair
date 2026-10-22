package couchcmd

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// drainProbeEndpoint is healthy at enrollment, then exposes controlled
// admission and the exact system bodies independently of coordinator state.
type drainProbeEndpoint struct {
	serviceEndpointFake
	armed, refuse              atomic.Bool
	entered, release, finished chan struct{}
	enterOnce, finishOnce      sync.Once
	bodies                     chan string
}

func newDrainProbeEndpoint() *drainProbeEndpoint {
	return &drainProbeEndpoint{entered: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{}), bodies: make(chan string, 8)}
}
func (e *drainProbeEndpoint) Observe(ctx context.Context) (couchmessage.Observation, error) {
	if e.refuse.Load() {
		return couchmessage.Observation{}, errors.New("fixture refuses this admission")
	}
	if e.armed.Load() {
		e.enterOnce.Do(func() { close(e.entered) })
		defer e.finishOnce.Do(func() { close(e.finished) })
		select {
		case <-e.release:
		case <-ctx.Done():
			return couchmessage.Observation{}, ctx.Err()
		}
	}
	return e.serviceEndpointFake.Observe(ctx)
}
func (e *drainProbeEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.bodies <- m.Body
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}

func drainOperationCall(t *testing.T, r *serviceRig, request couchmessage.Request) couchmessage.Response {
	t.Helper()
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, request, &response); err != nil {
		t.Fatal(err)
	}
	return response
}
func drainOperationStatus(t *testing.T, r *serviceRig, caller couchmessage.Binding, id string) couchmessage.OperationReceipt {
	t.Helper()
	request := callerRequest(caller, "operation-status")
	request.ID = id
	response := drainOperationCall(t, r, request)
	if response.Code != "ok" || response.Operation == nil {
		t.Fatalf("operation status: %+v", response)
	}
	return *response.Operation
}
func waitDrainOperation(t *testing.T, r *serviceRig, caller couchmessage.Binding, id string, want couchmessage.ReceiptStatus) couchmessage.OperationReceipt {
	t.Helper()
	var result couchmessage.OperationReceipt
	testwait.Until(t, "drain operation reaches "+string(want), func() bool {
		result = drainOperationStatus(t, r, caller, id)
		return result.Status == want
	})
	return result
}

// This real-socket29-target regression failed before the fix in2.000721125s:
// context deadline exceeded, nil parent and0 healthy reserves/deliveries.
func TestDrainOperationSlowFirstTargetDoesNotLoseExchange(t *testing.T) {
	r := newServiceRig(t)
	steward := r.connect(0)
	r.s.authority.hub = "/repo0"
	slow := newDrainProbeEndpoint()
	var reserves, deliveries atomic.Int32
	for i := 1; i <= 29; i++ {
		b := r.world.add(i)
		var endpoint couchmessage.DeliveryEndpoint = countedFlagEndpoint{reserves: &reserves, deliveries: &deliveries}
		if i == 1 {
			endpoint = slow
		}
		r.world.mu.Lock()
		r.world.endpoints[b] = endpoint
		r.world.mu.Unlock()
		r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("slow-fanout-%d", i)))
		r.wrapper(b)
		r.waitConnected(b, true)
	}
	slow.armed.Store(true)
	request := flagRequest(steward, "start")
	started := time.Now()
	response := drainOperationCall(t, r, request)
	t.Logf("29-target exchange elapsed=%s operation=%+v healthy_reserves=%d healthy_deliveries=%d", time.Since(started), response.Operation, reserves.Load(), deliveries.Load())
	if response.Code != "accepted" || response.Operation == nil || response.Operation.ID != request.ID || response.Operation.Drain == nil || len(response.Operation.Drain.Children) != 29 {
		t.Fatalf("unregistered mutation: %+v", response)
	}
	select {
	case <-slow.entered:
	case <-testwait.After(t):
		t.Fatal("slow target never entered")
	}
	testwait.Until(t, "all28 healthy targets deliver while first is blocked", func() bool { return deliveries.Load() == 28 })
	select {
	case <-slow.finished:
		t.Fatal("healthy targets progressed only after the slow target finished")
	default:
	}
	parent := drainOperationStatus(t, r, steward, request.ID)
	if parent.Status != couchmessage.ReceiptRunning {
		t.Fatalf("pending child reported finished: %+v", parent)
	}
	close(slow.release)
	waitDrainOperation(t, r, steward, request.ID, couchmessage.ReceiptSucceeded)
}

func TestDrainOperationFailedNoticeThenHarmlessLiftAndReplay(t *testing.T) {
	r := newServiceRig(t)
	steward := r.connect(0)
	r.s.authority.hub = "/repo0"
	probe := newDrainProbeEndpoint()
	worker := r.world.add(1)
	r.world.mu.Lock()
	r.world.endpoints[worker] = probe
	r.world.mu.Unlock()
	r.attach(worker, "never-noticed")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	probe.refuse.Store(true)
	start := flagRequest(steward, "start")
	// Deliberately discard the first response: the printed request ID alone
	// must recover both the mapping and the definite admission failure.
	raw, err := json.Marshal(start)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("unix", r.brokerSock)
	if err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, 4+len(raw))
	binary.BigEndian.PutUint32(frame, uint32(len(raw)))
	copy(frame[4:], raw)
	if n, err := conn.Write(frame); err != nil || n != len(frame) {
		conn.Close()
		t.Fatalf("private request write: n%d/%d %v", n, len(frame), err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	failed := waitDrainOperation(t, r, steward, start.ID, couchmessage.ReceiptFailed)
	if len(failed.Drain.Children) != 1 || failed.Drain.Children[0].ID == "" || failed.Drain.Children[0].Phase != couchmessage.DrainChildFailed {
		t.Fatalf("lost child evidence: %+v", failed)
	}
	select {
	case body := <-probe.bodies:
		t.Fatalf("failed notice delivered: %q", body)
	default:
	}
	probe.refuse.Store(false)
	lift := flagRequest(steward, "undrain")
	if got := drainOperationCall(t, r, lift); got.Code != "accepted" {
		t.Fatalf("lift: %+v", got)
	}
	succeeded := waitDrainOperation(t, r, steward, lift.ID, couchmessage.ReceiptSucceeded)
	if len(succeeded.Drain.Children) != 1 || succeeded.Drain.Children[0].Workspace != worker.Slot {
		t.Fatalf("missed un-noticed member: %+v", succeeded)
	}
	select {
	case body := <-probe.bodies:
		if !strings.HasPrefix(body, "drain lifted; continue your current work") || !strings.Contains(body, "If you received no drain notice, no checkpoint or recovery action is required.") {
			t.Fatalf("confusing lift: %q", body)
		}
	case <-testwait.After(t):
		t.Fatal("lift not delivered")
	}
	// Replaying the original ID after lift is a read of held outcome, never a
	// new posture mutation or another notice, even though the cohort changed.
	replay := drainOperationCall(t, r, start)
	if replay.Operation == nil || replay.Operation.ID != start.ID || replay.Operation.Status != couchmessage.ReceiptFailed {
		t.Fatalf("replay: %+v", replay)
	}
	board := drainOperationCall(t, r, flagRequest(steward, "status"))
	if board.Drain == nil || len(board.Drain.Rows) != 0 {
		t.Fatalf("replay reflagged: %+v", board)
	}
	changed := start
	intent := *start.Drain
	intent.Fast = true
	changed.Drain = &intent
	if got := drainOperationCall(t, r, changed); got.Code != "id-conflict" {
		t.Fatalf("conflicting ID accepted: %+v", got)
	}
	if foreign := drainOperationStatus(t, r, worker, start.ID); foreign.Status != couchmessage.ReceiptUnknown {
		t.Fatalf("receipt crossed caller boundary: %+v", foreign)
	}
}

func TestDrainOperationRegisteredBeforeFlagsAndRejectsCapacity(t *testing.T) {
	r := newServiceRig(t)
	steward := r.connect(0)
	r.s.authority.hub = "/repo0"
	r.connect(1)
	ops := r.s.drainOps
	ops.mu.Lock()
	for i := range maxSlotOperationReceipts {
		id := fmt.Sprintf("occupied-%d", i)
		ops.receipts[receiptKey{steward.Scope, steward.Tag, id}] = couchmessage.OperationReceipt{ID: id, Status: couchmessage.ReceiptRunning}
	}
	ops.mu.Unlock()
	response := drainOperationCall(t, r, flagRequest(steward, "start"))
	if response.Code != "overloaded" {
		t.Fatalf("capacity: %+v", response)
	}
	status := drainOperationCall(t, r, flagRequest(steward, "status"))
	if status.Drain == nil || len(status.Drain.Rows) != 0 {
		t.Fatalf("capacity refusal changed flags: %+v", status)
	}
}

func TestDrainOperationLatestNoticeSurvivesOldCompletion(t *testing.T) {
	r := newServiceRig(t)
	steward := r.connect(0)
	r.s.authority.hub = "/repo0"
	probe := newDrainProbeEndpoint()
	worker := r.world.add(1)
	r.world.mu.Lock()
	r.world.endpoints[worker] = probe
	r.world.mu.Unlock()
	r.attach(worker, "ordered-notices")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	probe.armed.Store(true)
	first := flagRequest(steward, "start")
	drainOperationCall(t, r, first)
	select {
	case <-probe.entered:
	case <-testwait.After(t):
		t.Fatal("first admission not started")
	}
	lift := flagRequest(steward, "undrain")
	drainOperationCall(t, r, lift)
	newest := flagRequest(steward, "start")
	newest.Drain.Fast = true
	drainOperationCall(t, r, newest)
	assertLatest := func() {
		t.Helper()
		board := drainOperationCall(t, r, flagRequest(steward, "status"))
		if board.Drain == nil || len(board.Drain.Rows) != 1 || board.Drain.Rows[0].OperationID != newest.ID {
			t.Fatalf("old completion replaced latest notice: %+v", board.Drain)
		}
	}
	assertLatest()
	close(probe.release)
	for _, request := range []couchmessage.Request{first, lift, newest} {
		waitDrainOperation(t, r, steward, request.ID, couchmessage.ReceiptSucceeded)
	}
	assertLatest()
	want := []string{"Drain: finish", "drain lifted; continue", "Drain (fast): stop"}
	for _, prefix := range want {
		select {
		case body := <-probe.bodies:
			if !strings.HasPrefix(body, prefix) {
				t.Fatalf("notice overtook predecessor: got%q want prefix%q", body, prefix)
			}
		case <-testwait.After(t):
			t.Fatal("notice missing")
		}
	}
}
