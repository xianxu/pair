package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type drainServiceEndpoint struct {
	mu                  sync.Mutex
	notices, interrupts int
	unknown             bool
	idle                bool
	instance            string
	receipts            map[string]couchmessage.Receipt
	results             map[string]couchmessage.DrainInterruptResult
	revoked             map[string]bool
}

func (e *drainServiceEndpoint) Observe(context.Context) (couchmessage.Observation, error) {
	return couchmessage.Observation{LastActivity: time.Now().Add(-time.Hour)}, nil
}
func (e *drainServiceEndpoint) Reserve(context.Context, string, uint64) error { return nil }
func (e *drainServiceEndpoint) Release(context.Context, string) error         { return nil }
func (e *drainServiceEndpoint) Deliver(_ context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.revoked[m.ID] {
		return couchmessage.Receipt{Message: m, Status: couchmessage.Cancelled}, nil
	}
	e.notices++
	r := couchmessage.Receipt{Message: m, Status: couchmessage.Submitted}
	if e.receipts == nil {
		e.receipts = map[string]couchmessage.Receipt{}
	}
	e.receipts[m.ID] = r
	return r, nil
}
func (e *drainServiceEndpoint) ObserveDrain(context.Context) (couchmessage.DrainObservation, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	instance := e.instance
	if instance == "" {
		instance = "instance"
	}
	return couchmessage.DrainObservation{Instance: instance, Turn: 1, Active: !e.idle, Idle: e.idle, ComposerKnown: true, ComposerEmpty: true, Complete: true}, nil
}
func (e *drainServiceEndpoint) InterruptDrain(_ context.Context, id string, o couchmessage.DrainObservation) (couchmessage.DrainInterruptResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.results == nil {
		e.results = map[string]couchmessage.DrainInterruptResult{}
	}
	if r, ok := e.results[id]; ok {
		return r, nil
	}
	r := couchmessage.DrainInterruptResult{ID: id, Instance: o.Instance, Turn: o.Turn, Outcome: "interrupted"}
	if e.revoked[id] {
		r.Outcome = "skipped"
	} else {
		e.interrupts++
	}
	e.results[id] = r
	if e.unknown {
		return couchmessage.DrainInterruptResult{}, errors.New("lost reply")
	}
	return r, nil
}
func (e *drainServiceEndpoint) DrainInterruptStatus(_ context.Context, id string) (couchmessage.DrainInterruptResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.unknown {
		return couchmessage.DrainInterruptResult{ID: id, Instance: "instance", Outcome: "unknown"}, nil
	}
	return e.results[id], nil
}
func (e *drainServiceEndpoint) RevokeDrain(_ context.Context, id string) (couchmessage.DrainRevokeResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.revoked == nil {
		e.revoked = map[string]bool{}
	}
	e.revoked[id] = true
	if r, ok := e.receipts[id]; ok {
		return couchmessage.DrainRevokeResult{ID: id, Instance: "instance", Outcome: "already-committed", Receipt: &r}, nil
	}
	if r, ok := e.results[id]; ok && r.Outcome == "interrupted" {
		if e.unknown {
			return couchmessage.DrainRevokeResult{ID: id, Instance: "instance", Outcome: "unknown"}, nil
		}
		return couchmessage.DrainRevokeResult{ID: id, Instance: "instance", Outcome: "already-committed", Interrupt: &r}, nil
	}
	return couchmessage.DrainRevokeResult{ID: id, Instance: "instance", Outcome: "not-committed"}, nil
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
	f := &drainServiceFleet{members: map[string]drainLiveMember{}, endpoints: map[couchmessage.Binding]*drainServiceEndpoint{}}
	for i := 0; i < 3; i++ {
		binding := couchmessage.Binding{Slot: fmt.Sprintf("pair:%d", i), Repository: "/repo/.git", Scope: "scope", Tag: fmt.Sprint(i), Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: 1, Start: "start"}
		e := &drainServiceEndpoint{}
		if err := b.Register(binding, e); err != nil {
			t.Fatal(err)
		}
		f.endpoints[binding] = e
		f.members[binding.Slot] = drainLiveMember{Binding: binding, Session: "admitted", Observation: couchcore.DrainObservation{Known: true, Connected: true}}
	}
	q := NewFakeDrainQueue()
	c, err := newDrainCoordinator(ctx, drainCoordinatorOptions{Broker: b, Queue: q, Store: newDrainStore(t.TempDir()), Now: time.Now, Snapshot: f.snapshot, Endpoint: func(b couchmessage.Binding) couchmessage.DrainEndpoint { return f.endpoints[b] }})
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
