package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// The fake uses the production pure generation authority; maps below retain
// current-generation payloads only, never decide whether input is authorized.
type drainServiceEndpoint struct {
	mu                  sync.Mutex
	notices, interrupts int
	unknown, idle       bool
	instance, namespace string
	authority           couchmessage.DrainGeneration
	protocol            int
	input               uint64
	proof               func(couchmessage.DrainFence) couchmessage.DrainFence
	receipts            map[string]couchmessage.Receipt
	results             map[string]couchmessage.DrainInterruptResult
	revoked             map[string]bool
}

func (e *drainServiceEndpoint) ownerLocked() error {
	instance := e.instance
	if instance == "" {
		instance = "instance"
	}
	if e.authority.View().Instance == instance && e.authority.View().Namespace == e.namespace {
		return nil
	}
	authority, err := couchmessage.NewDrainGeneration(e.namespace, instance)
	if err != nil {
		return err
	}
	e.authority = authority
	e.receipts = map[string]couchmessage.Receipt{}
	e.results = map[string]couchmessage.DrainInterruptResult{}
	e.revoked = map[string]bool{}
	return nil
}
func (e *drainServiceEndpoint) reduceLocked(event couchmessage.DrainGenerationEvent) error {
	if err := e.ownerLocked(); err != nil {
		return err
	}
	before := e.authority.View().Epoch
	next, err := couchmessage.ReduceDrainGeneration(e.authority, event)
	if err != nil {
		return err
	}
	e.authority = next
	if before != next.View().Epoch {
		e.receipts = map[string]couchmessage.Receipt{}
		e.results = map[string]couchmessage.DrainInterruptResult{}
		e.revoked = map[string]bool{}
	}
	return nil
}
func (e *drainServiceEndpoint) echoLocked(f couchmessage.DrainFence) couchmessage.DrainFence {
	if e.proof != nil {
		return e.proof(f)
	}
	return f
}
func (e *drainServiceEndpoint) Observe(context.Context) (couchmessage.Observation, error) {
	return couchmessage.Observation{LastActivity: time.Now().Add(-time.Hour)}, nil
}
func (e *drainServiceEndpoint) Reserve(context.Context, string, uint64) error { return nil }
func (e *drainServiceEndpoint) Release(context.Context, string) error         { return nil }
func (e *drainServiceEndpoint) ReserveDrain(_ context.Context, id string, sequence uint64, fence couchmessage.DrainFence) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if sequence != 0 || fence.Kind != couchmessage.DrainNoticeControl {
		return errors.New("fake notice reservation authority mismatch")
	}
	return e.reduceLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainAdmitControl, Fence: fence, ID: id})
}
func (e *drainServiceEndpoint) ReleaseDrain(_ context.Context, id string, fence couchmessage.DrainFence) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.authority.Lookup(fence, id)
	return err
}
func (e *drainServiceEndpoint) Deliver(_ context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	fence := m.DrainFence
	if fence.Kind != couchmessage.DrainNoticeControl {
		return couchmessage.Receipt{}, errors.New("fake expected fenced drain notice")
	}
	if err := e.ownerLocked(); err != nil {
		return couchmessage.Receipt{}, err
	}
	if _, err := e.authority.Lookup(fence, m.ID); err != nil {
		return couchmessage.Receipt{}, err
	}
	if r, ok := e.receipts[m.ID]; ok {
		return r, nil
	}
	if !e.authority.Allows(fence, m.ID) {
		return couchmessage.Receipt{Message: m, Status: couchmessage.Cancelled}, nil
	}
	if err := e.reduceLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainBeginInput, Fence: fence, ID: m.ID}); err != nil {
		return couchmessage.Receipt{}, err
	}
	e.notices++
	if err := e.reduceLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainSettleControl, Fence: fence, ID: m.ID, Disposition: couchmessage.DrainControlCommitted}); err != nil {
		return couchmessage.Receipt{}, err
	}
	r := couchmessage.Receipt{Message: m, Status: couchmessage.Submitted}
	e.receipts[m.ID] = r
	return r, nil
}
func (e *drainServiceEndpoint) ObserveDrain(context.Context) (couchmessage.DrainObservation, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.ownerLocked(); err != nil {
		return couchmessage.DrainObservation{}, err
	}
	view := e.authority.View()
	protocol := e.protocol
	if protocol == 0 {
		protocol = 2
	}
	return couchmessage.DrainObservation{Protocol: protocol, Namespace: view.Namespace, Epoch: view.Epoch, Generation: view.Generation, Instance: view.Instance, Input: e.input, Turn: 1, Active: !e.idle, Idle: e.idle, ComposerKnown: true, ComposerEmpty: true, Complete: true}, nil
}
func (e *drainServiceEndpoint) InterruptDrain(_ context.Context, id string, fence couchmessage.DrainFence, o couchmessage.DrainObservation) (couchmessage.DrainInterruptResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if fence.Kind != couchmessage.DrainInterruptControl || o.Protocol != 2 || o.Namespace != e.namespace || o.Instance != fence.Instance {
		return couchmessage.DrainInterruptResult{}, errors.New("fake interrupt observation mismatch")
	}
	if err := e.reduceLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainAdmitControl, Fence: fence, ID: id}); err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	if r, ok := e.results[id]; ok {
		return r, nil
	}
	result := couchmessage.DrainInterruptResult{ID: id, Fence: e.echoLocked(fence), Turn: o.Turn, Outcome: "skipped"}
	disposition := couchmessage.DrainControlNotCommitted
	if !e.idle && o.Active && e.authority.Allows(fence, id) {
		if err := e.reduceLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainBeginInput, Fence: fence, ID: id}); err != nil {
			return couchmessage.DrainInterruptResult{}, err
		}
		e.interrupts++
		result.Outcome = "interrupted"
		disposition = couchmessage.DrainControlCommitted
	}
	if err := e.reduceLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainSettleControl, Fence: fence, ID: id, Disposition: disposition}); err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	e.results[id] = result
	if e.unknown {
		return couchmessage.DrainInterruptResult{}, errors.New("lost reply")
	}
	return result, nil
}
func (e *drainServiceEndpoint) DrainInterruptStatus(_ context.Context, id string, fence couchmessage.DrainFence) (couchmessage.DrainInterruptResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.ownerLocked(); err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	if _, err := e.authority.Lookup(fence, id); err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	if e.unknown {
		return couchmessage.DrainInterruptResult{ID: id, Fence: e.echoLocked(fence), Outcome: "unknown"}, nil
	}
	r, ok := e.results[id]
	if !ok {
		return couchmessage.DrainInterruptResult{}, errors.New("fake unknown interrupt")
	}
	r.Fence = e.echoLocked(fence)
	return r, nil
}
func (e *drainServiceEndpoint) RevokeDrain(_ context.Context, id string, fence couchmessage.DrainFence) (couchmessage.DrainRevokeResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.ownerLocked(); err != nil {
		return couchmessage.DrainRevokeResult{}, err
	}
	disposition := couchmessage.DrainControlNotCommitted
	outcome := "not-committed"
	result := couchmessage.DrainRevokeResult{ID: id, Fence: e.echoLocked(fence)}
	if r, ok := e.receipts[id]; ok {
		disposition = couchmessage.DrainControlCommitted
		outcome = "already-committed"
		result.Receipt = &r
	}
	if r, ok := e.results[id]; ok && r.Outcome == "interrupted" {
		disposition = couchmessage.DrainControlCommitted
		outcome = "already-committed"
		result.Interrupt = &r
	}
	if e.unknown && outcome == "already-committed" {
		disposition = couchmessage.DrainControlUnknown
		outcome = "unknown"
		result.Receipt = nil
		result.Interrupt = nil
	}
	if err := e.reduceLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainRevokeControl, Fence: fence, ID: id, Disposition: disposition}); err != nil {
		return couchmessage.DrainRevokeResult{}, err
	}
	e.revoked[id] = true
	result.Outcome = outcome
	return result, nil
}

type drainServiceFleet struct {
	mu        sync.Mutex
	members   map[string]drainLiveMember
	endpoints map[couchmessage.Binding]*drainServiceEndpoint
}

func (f *drainServiceFleet) snapshot() map[string]drainLiveMember {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]drainLiveMember{}
	for k, v := range f.members {
		out[k] = v
	}
	return out
}
func (f *drainServiceFleet) idle() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, m := range f.members {
		m.Observation.Idle = true
		m.Observation.Sequence++
		f.members[k] = m
		e := f.endpoints[m.Binding]
		e.mu.Lock()
		e.idle = true
		e.mu.Unlock()
	}
}
func newDrainServiceFixture(t *testing.T) (*drainCoordinator, *drainServiceFleet, *FakeDrainQueue) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	b := couchmessage.NewBroker(ctx, time.Now, nil)
	t.Cleanup(func() { _ = b.Close() })
	store := newDrainStore(t.TempDir())
	namespace, err := couchmessage.DrainNamespaceKey(store.namespace)
	if err != nil {
		t.Fatal(err)
	}
	f := &drainServiceFleet{members: map[string]drainLiveMember{}, endpoints: map[couchmessage.Binding]*drainServiceEndpoint{}}
	for i := 0; i < 3; i++ {
		binding := couchmessage.Binding{Slot: fmt.Sprintf("pair:%d", i), Repository: "/repo/.git", Scope: "scope", Tag: fmt.Sprint(i), Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: 1, Start: "start"}
		e := &drainServiceEndpoint{namespace: namespace}
		if err := b.Register(binding, e); err != nil {
			t.Fatal(err)
		}
		f.endpoints[binding] = e
		f.members[binding.Slot] = drainLiveMember{Binding: binding, Session: "admitted", Observation: couchcore.DrainObservation{Known: true, Connected: true}}
	}
	q := NewFakeDrainQueue()
	c, err := newDrainCoordinator(ctx, drainCoordinatorOptions{Broker: b, Queue: q, Store: store, Now: time.Now, Snapshot: f.snapshot, Endpoint: func(b couchmessage.Binding) couchmessage.DrainEndpoint { return f.endpoints[b] }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.close)
	return c, f, q
}
func waitDrainService(t *testing.T, c *drainCoordinator, predicate func(couchcore.DrainBoard) bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for {
		c.reconcile(context.Background())
		if predicate(c.view()) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("board %+v", c.view())
		}
		time.Sleep(5 * time.Millisecond)
	}
}
func TestDrainServiceThreeMemberGracefulAckIdle(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	from := f.snapshot()["pair:0"].Binding
	b, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "restart", false, time.Minute, from)
	if err != nil || b.ID == "" {
		t.Fatal(b, err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		for _, r := range b.Rows {
			if r.Reason != "active turn" {
				return false
			}
		}
		return true
	})
	for workspace, m := range f.snapshot() {
		if _, err := c.ack(context.Background(), b.ID, workspace, m.Identity()); err != nil {
			t.Fatal(err)
		}
	}
	if c.view().State == couchcore.DrainDrained {
		t.Fatal("ack reused active turn")
	}
	f.idle()
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainDrained })
	for _, e := range f.endpoints {
		e.mu.Lock()
		n, i := e.notices, e.interrupts
		e.mu.Unlock()
		if n != 1 || i != 0 {
			t.Fatalf("effects notice=%d interrupt=%d", n, i)
		}
	}
}
func TestDrainServiceFastLostReplyNoRepeat(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	for _, e := range f.endpoints {
		e.unknown = true
	}
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", true, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainStuck })
	for _, e := range f.endpoints {
		e.mu.Lock()
		if e.notices != 0 || e.interrupts != 1 {
			t.Fatalf("unsafe repeat %+v", e)
		}
		e.unknown = false
		e.mu.Unlock()
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		for _, r := range b.Rows {
			if r.Reason != "active turn" {
				return false
			}
		}
		return true
	})
	for _, e := range f.endpoints {
		e.mu.Lock()
		i := e.interrupts
		e.mu.Unlock()
		if i != 1 {
			t.Fatal("repeated Esc")
		}
	}
}
func TestDrainServiceCancellationOwnOwner(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	_ = q.Pause(context.Background(), "other", "manual", false)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.cancel(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainCancelled })
	snapshot, _ := q.Status(context.Background())
	if len(snapshot.PauseOwners) != 1 || snapshot.PauseOwners[0].Owner != "other" {
		t.Fatal(snapshot)
	}
}

func TestDrainServiceRecordStrictAndUnknownPreservation(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	r := c.record
	c.mu.Unlock()
	raw, err := encodeDrainServiceRecord(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	_ = json.Unmarshal(raw, &object)
	object["future"] = json.RawMessage(`{"preserved":true}`)
	var state map[string]json.RawMessage
	_ = json.Unmarshal(object["state"], &state)
	state["FutureState"] = json.RawMessage(`"keep"`)
	object["state"], _ = json.Marshal(state)
	raw, _ = json.Marshal(object)
	decoded, err := decodeDrainServiceRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	rewritten, err := encodeDrainServiceRecord(decoded, raw)
	if err != nil || !bytes.Contains(rewritten, []byte(`"FutureState":"keep"`)) || !bytes.Contains(rewritten, []byte(`"future":{"preserved":true}`)) {
		t.Fatal(string(rewritten), err)
	}
	for _, bad := range []string{`{"schema_version":1}`, strings.Replace(string(raw), `"Fast":false`, `"Fast":null`, 1), strings.Replace(string(raw), `"Phase":`+fmt.Sprintf("%q", r.State.Phase), `"Phase":"unsafe"`, 1)} {
		if _, err := decodeDrainServiceRecord([]byte(bad)); err == nil {
			t.Fatalf("accepted malformed known state %s", bad)
		}
	}
}
func TestDrainServiceRecoverHoldsAndRequiresCancel(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	c.close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	broker := couchmessage.NewBroker(ctx, time.Now, nil)
	defer broker.Close()
	recovered, err := newDrainCoordinator(ctx, drainCoordinatorOptions{Broker: broker, Queue: q, Store: c.opts.Store, Now: time.Now, Snapshot: f.snapshot, Endpoint: c.opts.Endpoint})
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.close()
	if recovered.view().State != couchcore.DrainStuck {
		t.Fatal("recovered old proof", recovered.view())
	}
	_, err = recovered.ack(ctx, recovered.view().ID, "pair:0", f.snapshot()["pair:0"].Identity())
	if err == nil {
		t.Fatal("recovery accepted ack")
	}
	if _, err = recovered.start(ctx, "00000000-0000-4000-8000-000000000002", "different", false, time.Minute, f.snapshot()["pair:0"].Binding); err == nil {
		t.Fatal("recovery replaced owner")
	}
}
func TestDrainServiceDeadlineNoAckDoesNotResume(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	f.idle()
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Second, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		if b.State != couchcore.DrainStuck {
			return false
		}
		for _, r := range b.Rows {
			if r.Reason != "no ack" {
				return false
			}
		}
		return true
	})
	s, _ := q.Status(context.Background())
	if len(s.PauseOwners) != 1 {
		t.Fatal("timeout released owner")
	}
	for _, r := range c.view().Rows {
		if r.Reason != "no ack" {
			t.Fatalf("quiet missing ack: %+v", r)
		}
	}
}

func TestDrainServiceCensusBlocksQuietFleet(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	var incomplete atomic.Bool
	incomplete.Store(true)
	c.opts.Census = func() []couchcore.DrainRow {
		if incomplete.Load() {
			return []couchcore.DrainRow{{Workspace: "unsupported:1", State: couchcore.DrainStuck, Reason: "wrapper unregistered"}}
		}
		return nil
	}
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		for _, r := range b.Rows {
			if r.Workspace == "unsupported:1" {
				continue
			}
			if r.Reason != "active turn" {
				return false
			}
		}
		return true
	})
	for workspace, m := range f.snapshot() {
		_, _ = c.ack(context.Background(), c.view().ID, workspace, m.Identity())
	}
	f.idle()
	c.reconcile(context.Background())
	if c.view().State != couchcore.DrainStuck {
		t.Fatal("incomplete census granted completion", c.view())
	}
	incomplete.Store(false)
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainDrained })
}

func TestDrainServiceOldObservationCannotSealNewAck(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		for _, r := range b.Rows {
			if r.Reason != "active turn" {
				return false
			}
		}
		return true
	})
	member := f.snapshot()["pair:1"]
	c.mu.Lock()
	revision := c.ackRevision["pair:1"]
	c.mu.Unlock()
	if _, err := c.ack(context.Background(), c.view().ID, "pair:1", member.Identity()); err != nil {
		t.Fatal(err)
	}
	c.observeMember("pair:1", member.Identity(), couchcore.DrainObservation{Known: true, Connected: true, Idle: true}, revision)
	c.mu.Lock()
	sealed := c.record.State.Members["pair:1"].AckSealed
	c.mu.Unlock()
	if sealed {
		t.Fatal("pre-ack observation sealed later ack")
	}
}
func TestDrainServiceNewWrapperCannotProveOldInterruptAbsent(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	for _, e := range f.endpoints {
		e.unknown = true
	}
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", true, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainStuck })
	for _, e := range f.endpoints {
		e.mu.Lock()
		e.instance = "replacement"
		e.mu.Unlock()
	}
	_, err = c.cancel(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c.reconcile(context.Background())
	if c.view().State == couchcore.DrainCancelled {
		t.Fatal("new wrapper's empty retained map released old control")
	}
	snapshot, _ := q.Status(context.Background())
	if len(snapshot.PauseOwners) != 1 {
		t.Fatal("uncertain old effect released pause", snapshot)
	}
}

func TestDrainServiceRecoveredNoticeCanCancelWithSameWrapperProof(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		for _, r := range b.Rows {
			if r.Reason != "active turn" {
				return false
			}
		}
		return true
	})
	c.close()
	broker := couchmessage.NewBroker(context.Background(), time.Now, nil)
	defer broker.Close()
	recovered, err := newDrainCoordinator(context.Background(), drainCoordinatorOptions{Broker: broker, Queue: q, Store: c.opts.Store, Now: time.Now, Snapshot: f.snapshot, Endpoint: c.opts.Endpoint})
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.close()
	_, err = recovered.cancel(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, recovered, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainCancelled })
}

func TestDrainServiceRecordRejectsCaseFoldShadow(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	raw, _ := encodeDrainServiceRecord(c.record, nil)
	c.mu.Unlock()
	var object map[string]json.RawMessage
	_ = json.Unmarshal(raw, &object)
	object["State"] = json.RawMessage(`{}`)
	raw, _ = json.Marshal(object)
	if _, err := decodeDrainServiceRecord(raw); err == nil {
		t.Fatal("unknown case alias overwrote known drain authority")
	}
}

func TestDrainServiceRecordPreservesNestedActivityExtensions(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	record := c.record
	c.mu.Unlock()
	record.State.Queue.Active = []couchcore.DrainActivity{{ID: "work", Workspace: "pair:1", Kind: "suite", State: "running"}}
	raw, err := encodeDrainServiceRecord(record, nil)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	_ = json.Unmarshal(raw, &root)
	var state map[string]json.RawMessage
	_ = json.Unmarshal(root["state"], &state)
	var queue map[string]json.RawMessage
	_ = json.Unmarshal(state["Queue"], &queue)
	var activities []map[string]json.RawMessage
	_ = json.Unmarshal(queue["Active"], &activities)
	activities[0]["FutureActivity"] = json.RawMessage(`{"nested":[{"opaque":true}]}`)
	queue["Active"], _ = json.Marshal(activities)
	state["Queue"], _ = json.Marshal(queue)
	root["state"], _ = json.Marshal(state)
	raw, _ = json.Marshal(root)
	decoded, err := decodeDrainServiceRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	decoded.State.Queue.Active[0].State = "interrupting"
	rewritten, err := encodeDrainServiceRecord(decoded, raw)
	if err != nil || !bytes.Contains(rewritten, []byte(`"FutureActivity":{"nested":[{"opaque":true}]}`)) || !bytes.Contains(rewritten, []byte(`"State":"interrupting"`)) {
		t.Fatal("nested activity extension lost", string(rewritten), err)
	}
}

type drainReexecBeforeFinalIdle struct {
	couchmessage.DrainEndpoint
	observations atomic.Int32
}

func (e *drainReexecBeforeFinalIdle) ObserveDrain(ctx context.Context) (couchmessage.DrainObservation, error) {
	o, err := e.DrainEndpoint.ObserveDrain(ctx)
	if e.observations.Add(1) >= 2 {
		o.Instance = "same-binding-reexecuted-wrapper"
	}
	return o, err
}
func TestDrainServiceFinalIdleMustMatchPersistedWrapperInstance(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
	if err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		for _, r := range b.Rows {
			if r.Reason != "active turn" {
				return false
			}
		}
		return true
	})
	c.pass.Lock()
	member := f.snapshot()["pair:1"]
	original := c.opts.Endpoint
	swapped := &drainReexecBeforeFinalIdle{DrainEndpoint: original(member.Binding)}
	c.opts.Endpoint = func(binding couchmessage.Binding) couchmessage.DrainEndpoint {
		if binding == member.Binding {
			return swapped
		}
		return original(binding)
	}
	if _, err := c.ack(context.Background(), c.view().ID, "pair:1", member.Identity()); err != nil {
		c.pass.Unlock()
		t.Fatal(err)
	}
	f.idle()
	c.pass.Unlock()
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		c.mu.Lock()
		m := c.record.State.Members["pair:1"]
		c.mu.Unlock()
		return b.State == couchcore.DrainStuck && !m.Ack && !m.Observation.Known
	})
}

func TestDrainServiceRecordActivityExtensionsFollowStableIdentity(t *testing.T) {
	a := couchcore.DrainActivity{ID: "a", Kind: "suite", Workspace: "pair:1", State: "running"}
	b := couchcore.DrainActivity{ID: "b", Kind: "merge", Workspace: "pair:2", State: "running"}
	old := json.RawMessage(`[{"ID":"a","Kind":"suite","Workspace":"pair:1","State":"running","Head":"","FutureActivity":{"owner":"a"}},{"ID":"b","Kind":"merge","Workspace":"pair:2","State":"running","Head":"","FutureActivity":{"owner":"b"}}]`)
	cases := []struct {
		name    string
		current []couchcore.DrainActivity
		want    []string
	}{
		{"reordered", []couchcore.DrainActivity{b, a}, []string{"b", "a"}},
		{"retired", []couchcore.DrainActivity{b}, []string{"b"}},
		{"new-id", []couchcore.DrainActivity{{ID: "new", Kind: "suite", Workspace: "pair:1", State: "running"}}, []string{""}},
		{"new-kind", []couchcore.DrainActivity{{ID: "a", Kind: "landing", Workspace: "pair:1", State: "running"}}, []string{""}},
		{"new-workspace", []couchcore.DrainActivity{{ID: "a", Kind: "suite", Workspace: "pair:7", State: "running"}}, []string{""}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fresh, _ := json.Marshal(test.current)
			merged, err := mergeDrainRecordUnknown(fresh, old, reflect.TypeOf(test.current))
			if err != nil {
				t.Fatal(err)
			}
			var activities []struct {
				FutureActivity struct {
					Owner string `json:"owner"`
				}
			}
			if err := json.Unmarshal(merged, &activities); err != nil {
				t.Fatal(err)
			}
			for i, want := range test.want {
				if activities[i].FutureActivity.Owner != want {
					t.Fatalf("activity %d extension belongs to %q, want %q: %s", i, activities[i].FutureActivity.Owner, want, merged)
				}
			}
		})
	}
}

func TestDrainServiceRecordRejectsAmbiguousActivityExtensions(t *testing.T) {
	activity := json.RawMessage(`{"ID":"a","Kind":"suite","Workspace":"pair:1","State":"running","Head":""}`)
	one := json.RawMessage("[" + string(activity) + "]")
	duplicate := json.RawMessage("[" + string(activity) + "," + string(activity) + "]")
	for _, pair := range [][2]json.RawMessage{{one, duplicate}, {duplicate, one}, {json.RawMessage(`[{"ID":"","Kind":"suite"}]`), one}} {
		if _, err := mergeDrainRecordUnknown(pair[0], pair[1], reflect.TypeOf([]couchcore.DrainActivity{})); err == nil {
			t.Fatal("accepted ambiguous or absent activity identity")
		}
	}
}

func drainServiceRecordEpoch(t *testing.T, c *drainCoordinator) uint64 {
	t.Helper()
	c.mu.Lock()
	raw, err := json.Marshal(c.record)
	c.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	var epoch uint64
	if err := json.Unmarshal(fields["epoch"], &epoch); err != nil || epoch == 0 {
		t.Fatalf("persisted drain has no positive epoch: %s", raw)
	}
	return epoch
}
func TestDrainServiceEpochAllocationAndCancelledRetry(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	ctx := context.Background()
	from := f.snapshot()["pair:0"].Binding
	id1 := "00000000-0000-4000-8000-000000000001"
	id2 := "00000000-0000-4000-8000-000000000002"
	if _, err := c.start(ctx, id1, "restart", false, time.Minute, from); err != nil {
		t.Fatal(err)
	}
	if epoch := drainServiceRecordEpoch(t, c); epoch != 1 {
		t.Fatalf("first epoch %d", epoch)
	}
	duplicate, err := c.start(ctx, id1, "restart", false, time.Minute, from)
	if err != nil || duplicate.ID != id1 || drainServiceRecordEpoch(t, c) != 1 {
		t.Fatal("active retry allocated an epoch", duplicate, err)
	}
	if _, err := c.cancel(ctx); err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainCancelled })
	cancelled, err := c.start(ctx, id1, "restart", false, time.Minute, from)
	if err != nil || cancelled.State != couchcore.DrainCancelled || drainServiceRecordEpoch(t, c) != 1 {
		t.Fatal("delayed cancelled retry revived generation", cancelled, err)
	}
	if _, err := c.start(ctx, id2, "restart", false, time.Minute, from); err != nil {
		t.Fatal(err)
	}
	if epoch := drainServiceRecordEpoch(t, c); epoch != 2 {
		t.Fatalf("next epoch %d", epoch)
	}
}
func TestDrainServiceEpochSurvivesRecovery(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	ctx := context.Background()
	if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding); err != nil {
		t.Fatal(err)
	}
	before := drainServiceRecordEpoch(t, c)
	c.close()
	broker := couchmessage.NewBroker(ctx, time.Now, nil)
	defer broker.Close()
	recovered, err := newDrainCoordinator(ctx, drainCoordinatorOptions{Broker: broker, Queue: q, Store: c.opts.Store, Now: time.Now, Snapshot: f.snapshot, Endpoint: c.opts.Endpoint})
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.close()
	if recovered.view().State != couchcore.DrainStuck || drainServiceRecordEpoch(t, recovered) != before {
		t.Fatal("recovery changed epoch or restored completion", recovered.view())
	}
}
func TestDrainServiceEpochOverflowPrecedesIntentAndEffects(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	ctx := context.Background()
	from := f.snapshot()["pair:0"].Binding
	if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000001", "", false, time.Minute, from); err != nil {
		t.Fatal(err)
	}
	_ = drainServiceRecordEpoch(t, c)
	if _, err := c.cancel(ctx); err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainCancelled })
	c.pass.Lock()
	c.mu.Lock()
	raw, _ := json.Marshal(c.record)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	fields["epoch"] = json.RawMessage(`18446744073709551615`)
	raw, _ = json.Marshal(fields)
	if err := json.Unmarshal(raw, &c.record); err != nil {
		t.Fatal(err)
	}
	c.raw = raw
	c.mu.Unlock()
	if err := c.opts.Store.Write(ctx, raw); err != nil {
		c.pass.Unlock()
		t.Fatal(err)
	}
	c.pass.Unlock()
	before, _ := c.opts.Store.Read(ctx)
	if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000002", "", false, time.Minute, from); err == nil {
		t.Fatal("epoch overflow admitted new generation")
	}
	after, _ := c.opts.Store.Read(ctx)
	if !bytes.Equal(before, after) {
		t.Fatal("epoch overflow rewrote durable intent")
	}
	status, _ := q.Status(ctx)
	if len(status.PauseOwners) != 0 {
		t.Fatal("epoch overflow paused queue", status)
	}
}
func TestDrainServiceUnknownCancellationBlocksEpochRetirement(t *testing.T) {
	c, f, _ := newDrainServiceFixture(t)
	ctx := context.Background()
	for _, e := range f.endpoints {
		e.unknown = true
	}
	from := f.snapshot()["pair:0"].Binding
	if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000001", "", true, time.Minute, from); err != nil {
		t.Fatal(err)
	}
	epoch := drainServiceRecordEpoch(t, c)
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainStuck })
	if _, err := c.cancel(ctx); err != nil {
		t.Fatal(err)
	}
	c.reconcile(ctx)
	if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000002", "new request", false, time.Minute, from); err == nil {
		t.Fatal("unknown cancellation retired its epoch")
	}
	if drainServiceRecordEpoch(t, c) != epoch {
		t.Fatal("unknown cancellation changed epoch")
	}
	for _, e := range f.endpoints {
		e.mu.Lock()
		e.unknown = false
		e.mu.Unlock()
	}
	if _, err := c.cancel(ctx); err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainCancelled })
	if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000002", "new request", false, time.Minute, from); err != nil {
		t.Fatal(err)
	}
	if drainServiceRecordEpoch(t, c) != epoch+1 {
		t.Fatal("confirmed cancellation did not admit next epoch")
	}
}

func TestDrainServiceRejectedEpochRecordCannotAuthorizeCancel(t *testing.T) {
	for _, badField := range []string{"namespace", "epoch", "schema_version"} {
		t.Run(badField, func(t *testing.T) {
			c, f, q := newDrainServiceFixture(t)
			ctx := context.Background()
			if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding); err != nil {
				t.Fatal(err)
			}
			_ = drainServiceRecordEpoch(t, c)
			waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
				for _, r := range b.Rows {
					if r.Reason != "active turn" {
						return false
					}
				}
				return true
			})
			c.close()
			raw, err := c.opts.Store.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var object map[string]json.RawMessage
			_ = json.Unmarshal(raw, &object)
			switch badField {
			case "namespace":
				object[badField] = json.RawMessage(`"wrong-namespace"`)
			case "epoch":
				object[badField] = json.RawMessage(`0`)
			case "schema_version":
				object[badField] = json.RawMessage(`1`)
			}
			raw, _ = json.Marshal(object)
			if err := os.WriteFile(filepath.Join(c.opts.Store.namespace, "drain.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			broker := couchmessage.NewBroker(ctx, time.Now, nil)
			defer broker.Close()
			recovered, err := newDrainCoordinator(ctx, drainCoordinatorOptions{Broker: broker, Queue: q, Store: c.opts.Store, Now: time.Now, Snapshot: f.snapshot, Endpoint: c.opts.Endpoint})
			if err != nil {
				t.Fatal(err)
			}
			defer recovered.close()
			if recovered.view().State != couchcore.DrainStuck {
				t.Fatal("rejected record did not hold", recovered.view())
			}
			if _, err := recovered.cancel(ctx); err == nil {
				t.Fatal("partially decoded rejected record authorized cancellation")
			}
			recovered.mu.Lock()
			schema := recovered.record.SchemaVersion
			controls := len(recovered.record.Controls)
			recovered.mu.Unlock()
			if schema != 0 || controls != 0 {
				t.Fatalf("rejected authority survived: schema=%d controls=%d", schema, controls)
			}
			status, _ := q.Status(ctx)
			if len(status.PauseOwners) != 1 || status.PauseOwners[0].Owner != "couch-drain:00000000-0000-4000-8000-000000000001" {
				t.Fatal("rejected record dropped original queue owner", status)
			}
			for _, endpoint := range f.endpoints {
				endpoint.mu.Lock()
				revoked := len(endpoint.revoked)
				endpoint.mu.Unlock()
				if revoked != 0 {
					t.Fatal("rejected record invoked control revocation")
				}
			}
		})
	}
}

func TestDrainServiceInterruptProofRequiresExactFence(t *testing.T) {
	mutations := map[string]func(couchmessage.DrainFence) couchmessage.DrainFence{
		"namespace": func(f couchmessage.DrainFence) couchmessage.DrainFence {
			f.Namespace = strings.Repeat("a", 64)
			return f
		},
		"generation": func(f couchmessage.DrainFence) couchmessage.DrainFence {
			f.Generation = "00000000-0000-4000-8000-000000000099"
			return f
		},
		"epoch":    func(f couchmessage.DrainFence) couchmessage.DrainFence { f.Epoch++; return f },
		"instance": func(f couchmessage.DrainFence) couchmessage.DrainFence { f.Instance = "replacement"; return f },
		"kind": func(f couchmessage.DrainFence) couchmessage.DrainFence {
			f.Kind = couchmessage.DrainNoticeControl
			return f
		},
	}
	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			c, f, _ := newDrainServiceFixture(t)
			for _, endpoint := range f.endpoints {
				endpoint.proof = mutation
			}
			if _, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", true, time.Minute, f.snapshot()["pair:0"].Binding); err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, c, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainStuck })
			c.reconcile(context.Background())
			for _, endpoint := range f.endpoints {
				endpoint.mu.Lock()
				notices, interrupts := endpoint.notices, endpoint.interrupts
				endpoint.mu.Unlock()
				if notices != 0 || interrupts != 1 {
					t.Fatalf("wrong proof enabled notice or repeated Esc: %d,%d", notices, interrupts)
				}
			}
		})
	}
}
func TestDrainServiceRevokeProofRequiresExactFence(t *testing.T) {
	c, f, q := newDrainServiceFixture(t)
	ctx := context.Background()
	if _, err := c.start(ctx, "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding); err != nil {
		t.Fatal(err)
	}
	waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
		for _, r := range b.Rows {
			if r.Reason != "active turn" {
				return false
			}
		}
		return true
	})
	for _, endpoint := range f.endpoints {
		endpoint.mu.Lock()
		endpoint.proof = func(f couchmessage.DrainFence) couchmessage.DrainFence { f.Epoch++; return f }
		endpoint.mu.Unlock()
	}
	if _, err := c.cancel(ctx); err != nil {
		t.Fatal(err)
	}
	c.reconcile(ctx)
	if c.view().State == couchcore.DrainCancelled {
		t.Fatal("wrong revoke epoch released dispatch")
	}
	status, _ := q.Status(ctx)
	if len(status.PauseOwners) != 1 {
		t.Fatal("wrong revoke epoch released queue owner")
	}
}
func TestDrainServiceFakeBoundedEpochAuthority(t *testing.T) {
	namespace, err := couchmessage.DrainNamespaceKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, cycles := range []int{1000, 2000} {
		t.Run(fmt.Sprint(cycles), func(t *testing.T) {
			endpoint := &drainServiceEndpoint{namespace: namespace}
			peak := 0
			var retired couchmessage.DrainFence
			var retiredID string
			for i := 1; i <= cycles; i++ {
				observed, err := endpoint.ObserveDrain(ctx)
				if err != nil {
					t.Fatal(err)
				}
				fence := couchmessage.DrainFence{Namespace: namespace, Generation: fmt.Sprintf("%08x-0000-4000-8000-%012x", i, i), Epoch: uint64(i), Instance: observed.Instance, Kind: couchmessage.DrainInterruptControl}
				interruptID := fmt.Sprintf("interrupt-%d", i)
				if _, err := endpoint.InterruptDrain(ctx, interruptID, fence, observed); err != nil {
					t.Fatal(err)
				}
				noticeFence := fence
				noticeFence.Kind = couchmessage.DrainNoticeControl
				noticeID := fmt.Sprintf("notice-%d", i)
				if err := endpoint.ReserveDrain(ctx, noticeID, 0, noticeFence); err != nil {
					t.Fatal(err)
				}
				if _, err := endpoint.Deliver(ctx, couchmessage.Message{ID: noticeID, DrainFence: noticeFence}, func(couchmessage.Receipt) {}); err != nil {
					t.Fatal(err)
				}
				if i > 1 {
					if _, err := endpoint.DrainInterruptStatus(ctx, retiredID, retired); err == nil {
						t.Fatal("retired read claimed current outcome")
					}
					if _, err := endpoint.InterruptDrain(ctx, retiredID, retired, observed); err == nil {
						t.Fatal("retired input regained authority")
					}
				}
				if _, err := endpoint.RevokeDrain(ctx, noticeID, noticeFence); err != nil {
					t.Fatal(err)
				}
				if _, err := endpoint.RevokeDrain(ctx, interruptID, fence); err != nil {
					t.Fatal(err)
				}
				endpoint.mu.Lock()
				residue := len(endpoint.receipts) + len(endpoint.results) + len(endpoint.revoked)
				view := endpoint.authority.View()
				endpoint.mu.Unlock()
				peak = max(peak, residue)
				if residue > 4 || view.Epoch != uint64(i) {
					t.Fatalf("epoch %d residue %d view %+v", i, residue, view)
				}
				retired, retiredID = fence, interruptID
			}
			endpoint.mu.Lock()
			notices, interrupts := endpoint.notices, endpoint.interrupts
			endpoint.mu.Unlock()
			if peak != 4 || notices != cycles || interrupts != cycles {
				t.Fatalf("cycles%d peak%d notices%d interrupts%d", cycles, peak, notices, interrupts)
			}
		})
	}
}

func TestDrainServiceAckSamplesCurrentActivity(t *testing.T) {
	for _, tc := range []struct {
		name              string
		submission, input uint64
	}{
		{"registry", 11, 0}, {"wrapper", 0, 11}, {"both", 6, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, f, _ := newDrainServiceFixture(t)
			_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
			if err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
				for _, r := range b.Rows {
					if r.Reason != "active turn" {
						return false
					}
				}
				return true
			})
			// Freeze reconciliation so the registry/wrapper advance before ack while
			// persisted member evidence stays at the older counter.
			c.pass.Lock()
			defer c.pass.Unlock()
			member := f.snapshot()["pair:1"]
			f.mu.Lock()
			member.Observation.Submission = tc.submission
			f.members["pair:1"] = member
			f.mu.Unlock()
			endpoint := f.endpoints[member.Binding]
			endpoint.mu.Lock()
			endpoint.input = tc.input
			endpoint.mu.Unlock()
			c.mu.Lock()
			m := c.record.State.Members["pair:1"]
			m.Observation.Submission = 10
			c.record.State.Members["pair:1"] = m
			oldRevision := c.ackRevision["pair:1"]
			c.mu.Unlock()
			if _, err := c.ack(context.Background(), c.view().ID, "pair:1", member.Identity()); err != nil {
				t.Fatal(err)
			}
			c.mu.Lock()
			m = c.record.State.Members["pair:1"]
			revision := c.ackRevision["pair:1"]
			c.mu.Unlock()
			if !m.Ack || m.AckSubmission != 11 || m.AckSealed {
				t.Fatalf("ack-time baseline: %+v", m)
			}
			c.observeMember("pair:1", member.Identity(), couchcore.DrainObservation{Known: true, Connected: true, Idle: true, Submission: 10}, oldRevision)
			c.mu.Lock()
			m = c.record.State.Members["pair:1"]
			c.mu.Unlock()
			if !m.Ack || m.AckSealed {
				t.Fatalf("pre-ack in-flight observation changed ack: %+v", m)
			}
			c.observeMember("pair:1", member.Identity(), couchcore.DrainObservation{Known: true, Connected: true, Idle: true, Submission: 11}, revision)
			c.mu.Lock()
			m = c.record.State.Members["pair:1"]
			c.mu.Unlock()
			if !m.AckSealed {
				t.Fatalf("preceding activity removed ack: %+v", m)
			}
			c.observeMember("pair:1", member.Identity(), couchcore.DrainObservation{Known: true, Connected: true, Idle: true, Submission: 12}, revision)
			c.mu.Lock()
			m = c.record.State.Members["pair:1"]
			c.mu.Unlock()
			if m.Ack {
				t.Fatal("post-ack activity retained ack")
			}
		})
	}
}

// This wrapper injects replacement/failure exactly after the authoritative read,
// rather than depending on a scheduler race.
type drainAckEndpoint struct {
	couchmessage.DrainEndpoint
	after  func()
	mutate func(*couchmessage.DrainObservation)
	err    error
}

func (e drainAckEndpoint) ObserveDrain(ctx context.Context) (couchmessage.DrainObservation, error) {
	o, err := e.DrainEndpoint.ObserveDrain(ctx)
	if e.mutate != nil {
		e.mutate(&o)
	}
	if e.after != nil {
		e.after()
	}
	if e.err != nil {
		err = e.err
	}
	return o, err
}
func TestDrainServiceAckRequiresCurrentEvidence(t *testing.T) {
	for _, name := range []string{"unknown registry", "disconnected registry", "missing wrapper", "wrapper failure", "old instance", "old epoch", "other generation", "old protocol", "registry replacement", "intent replacement"} {
		t.Run(name, func(t *testing.T) {
			c, f, _ := newDrainServiceFixture(t)
			_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
			if err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
				for _, r := range b.Rows {
					if r.Reason != "active turn" {
						return false
					}
				}
				return true
			})
			c.pass.Lock()
			defer c.pass.Unlock()
			member := f.snapshot()["pair:1"]
			original := c.opts.Endpoint
			endpoint := drainAckEndpoint{DrainEndpoint: original(member.Binding)}
			switch name {
			case "unknown registry", "disconnected registry":
				f.mu.Lock()
				m := f.members["pair:1"]
				if name == "unknown registry" {
					m.Observation.Known = false
				} else {
					m.Observation.Connected = false
				}
				f.members["pair:1"] = m
				f.mu.Unlock()
			case "wrapper failure":
				endpoint.err = errors.New("read unavailable")
			case "old instance":
				endpoint.mutate = func(o *couchmessage.DrainObservation) { o.Instance = "replaced" }
			case "old epoch":
				endpoint.mutate = func(o *couchmessage.DrainObservation) { o.Epoch--; o.Generation = "" }
			case "other generation":
				endpoint.mutate = func(o *couchmessage.DrainObservation) { o.Generation = "00000000-0000-4000-8000-000000000002" }
			case "old protocol":
				endpoint.mutate = func(o *couchmessage.DrainObservation) { o.Protocol = 1 }
			case "registry replacement":
				endpoint.after = func() {
					f.mu.Lock()
					m := f.members["pair:1"]
					m.Session = "new session"
					f.members["pair:1"] = m
					f.mu.Unlock()
				}
			case "intent replacement":
				endpoint.after = func() { c.mu.Lock(); c.record.Epoch++; c.mu.Unlock() }
			}
			c.opts.Endpoint = func(b couchmessage.Binding) couchmessage.DrainEndpoint {
				if b == member.Binding {
					if name == "missing wrapper" {
						return nil
					}
					return endpoint
				}
				return original(b)
			}
			if _, err := c.ack(context.Background(), c.view().ID, "pair:1", member.Identity()); err == nil {
				t.Fatal("accepted missing or replaced ack evidence")
			}
			c.mu.Lock()
			acked := c.record.State.Members["pair:1"].Ack
			c.mu.Unlock()
			if acked {
				t.Fatal("refused ack changed authority")
			}
		})
	}
}

func TestDrainServiceSnapshotBeforeAckCannotInvalidateIt(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing=%v", missing), func(t *testing.T) {
			c, f, _ := newDrainServiceFixture(t)
			_, err := c.start(context.Background(), "00000000-0000-4000-8000-000000000001", "", false, time.Minute, f.snapshot()["pair:0"].Binding)
			if err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, c, func(b couchcore.DrainBoard) bool {
				for _, r := range b.Rows {
					if r.Reason != "active turn" {
						return false
					}
				}
				return true
			})
			snapshotRead, release := make(chan struct{}), make(chan struct{})
			var once atomic.Bool
			c.pass.Lock()
			c.opts.Snapshot = func() map[string]drainLiveMember {
				live := f.snapshot()
				if once.CompareAndSwap(false, true) {
					if missing {
						delete(live, "pair:1")
					}
					close(snapshotRead)
					<-release
				}
				return live
			}
			c.pass.Unlock()
			// Cleanup releases the paused read even if acknowledgement refuses.
			var released bool
			defer func() {
				if !released {
					close(release)
				}
			}()
			c.signal()
			select {
			case <-snapshotRead:
			case <-time.After(3 * time.Second):
				t.Fatal("snapshot not reached")
			}
			member := f.snapshot()["pair:1"]
			f.mu.Lock()
			member.Observation.Submission = 1
			f.members["pair:1"] = member
			f.mu.Unlock()
			if _, err := c.ack(context.Background(), c.view().ID, "pair:1", member.Identity()); err != nil {
				t.Fatal(err)
			}
			close(release)
			released = true
			c.pass.Lock()
			defer c.pass.Unlock()
			c.mu.Lock()
			m := c.record.State.Members["pair:1"]
			c.mu.Unlock()
			if !m.Ack {
				t.Fatal("registry read begun before ack invalidated its newer baseline")
			}

		})
	}
}
