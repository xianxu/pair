package couchcmd

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestLifecycleNoticesSetting(t *testing.T) {
	for _, tc := range []struct {
		value     string
		on, valid bool
	}{{"", true, true}, {"on", true, true}, {"off", false, true}, {"yes", false, false}} {
		on, err := lifecycleNoticesEnabled(func(string) string { return tc.value })
		if on != tc.on || (err == nil) != tc.valid {
			t.Fatalf("%q: %v %v", tc.value, on, err)
		}
	}
}

func TestLifecyclePublisherUsesCanonicalSystemReceiptAndRestartBarrier(t *testing.T) {
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub, worker := r.world.add(1), r.world.add(2)
	endpoint := &supervisionControlledEndpoint{}
	r.world.endpoints[hub] = endpoint
	r.attach(hub, "hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	r.attach(worker, "worker")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	p := startLifecyclePublisher(r.s, func(string) {})
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	p.observe(couchtty.LifecycleObservation{Kind: "recovered", Path: "/repo2", Incarnation: "worker-one", Observation: 1, Initiation: "automatic"})
	// No summary is eligible while the existing restart pass is collecting.
	p.mu.Lock()
	queued := len(p.ready)
	p.mu.Unlock()
	if queued != 0 {
		t.Fatal("published before drain")
	}
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	testwait.Until(t, "canonical lifecycle receipt", func() bool {
		endpoint.mu.Lock()
		var id string
		if len(endpoint.messages) > 0 {
			id = endpoint.messages[0].ID
		}
		endpoint.mu.Unlock()
		if id == "" {
			return false
		}
		receipt, ok, _ := r.s.broker.ReceiptSnapshot(id)
		if !ok {
			return false
		}
		if receipt.Message.Origin != couchmessage.OriginSystem || !strings.Contains(receipt.Message.Body, "recovered") || !strings.Contains(receipt.Message.Body, worker.Slot) {
			t.Fatalf("receipt: %+v", receipt)
		}
		return true
	})
}

func lifecyclePublisherRig(t *testing.T, ep couchmessage.DeliveryEndpoint, notices ...func(string)) (*serviceRig, *lifecyclePublisher, couchmessage.Binding) {
	t.Helper()
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub := r.world.add(1)
	r.world.endpoints[hub] = ep
	r.attach(hub, "hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	notice := func(string) {}
	if len(notices) > 0 {
		notice = notices[0]
	}
	return r, startLifecyclePublisher(r.s, notice), hub
}
func lifecycleSlotObservation(n int, observation uint64) couchtty.LifecycleObservation {
	return couchtty.LifecycleObservation{Kind: "recovered", Path: fmt.Sprintf("/repo%d", n), Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "pair", Number: n, PrimaryRoot: "/repo1", WorktreeRoot: fmt.Sprintf("/repo%d", n)}}, Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: couchcore.ThreadTag(fmt.Sprintf("t%d", n))}, Incarnation: fmt.Sprintf("%d:start", n), Observation: observation, Initiation: "automatic"}
}
func lifecycleNextMessage(t *testing.T, ep *controlledEndpoint) couchmessage.Message {
	t.Helper()
	select {
	case m := <-ep.started:
		return m
	case <-testwait.After(t):
		t.Fatal("no lifecycle message")
		return couchmessage.Message{}
	}
}
func lifecycleCanonicalSubmitted(t *testing.T, r *serviceRig, m couchmessage.Message) {
	t.Helper()
	testwait.Until(t, "canonical submitted lifecycle receipt", func() bool {
		receipt, known, busy := r.s.broker.ReceiptSnapshot(m.ID)
		if !known || busy || receipt.Status != couchmessage.Submitted {
			return false
		}
		if receipt.Message != m || receipt.Message.Origin != couchmessage.OriginSystem {
			t.Fatalf("canonical mismatch: %+v", receipt)
		}
		return true
	})
}
func TestLifecycleBindingKeyBoundsFullIdentity(t *testing.T) {
	b := couchmessage.Binding{Slot: "pair:1", Repository: strings.Repeat("/deep", 150), Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: 1, Start: "start"}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	key := lifecycleBindingKey(b)
	if len(key) > 512 {
		t.Fatalf("binding key exceeds collector limit: %d", len(key))
	}
	b.Nonce = "replacement"
	if lifecycleBindingKey(b) == key {
		t.Fatal("full identity lost")
	}
}
func TestLifecyclePublisherRetiresSubmittedAndPublishesNewFact(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(lifecycleSlotObservation(2, 1))
	first := lifecycleNextMessage(t, ep)
	later := lifecycleSlotObservation(2, 2)
	later.Kind = "parked"
	p.observe(later)
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, first)
	second := lifecycleNextMessage(t, ep)
	if second.ID == first.ID || !strings.Contains(second.Body, "parked") || second.Body == first.Body {
		t.Fatalf("new snapshot: %+v", second)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, second)
}
func TestLifecyclePublisherTwentySlotRestartOneCanonicalSummary(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	for n := 2; n < 22; n++ {
		p.observe(lifecycleSlotObservation(n, uint64(n)))
	}
	p.refresh()
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	m := lifecycleNextMessage(t, ep)
	if strings.Count(m.Body, "recovered") != 20 || !strings.Contains(m.Body, "restart") {
		t.Fatalf("restart summary: %s", m.Body)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
}
func TestLifecyclePublisherShutdownJoinsBlockedDelivery(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(lifecycleSlotObservation(2, 1))
	_ = lifecycleNextMessage(t, ep)
	done := make(chan struct{})
	go func() { r.s.Close(); close(done) }()
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("publisher shutdown blocked")
	}
}

type lifecycleBusyEndpoint struct {
	*controlledEndpoint
	mu       sync.Mutex
	busy     bool
	reserved chan struct{}
}

func (e *lifecycleBusyEndpoint) Reserve(ctx context.Context, _ string, _ uint64) error {
	e.mu.Lock()
	busy := e.busy
	e.mu.Unlock()
	select {
	case e.reserved <- struct{}{}:
	default:
	}
	if busy {
		return couchmessage.ErrRecipientBusy
	}
	return ctx.Err()
}
func TestLifecyclePublisherBusyTwentySlotRestart(t *testing.T) {
	ep := &lifecycleBusyEndpoint{controlledEndpoint: newControlledEndpoint(), busy: true, reserved: make(chan struct{}, 1)}
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	for n := 2; n < 22; n++ {
		p.observe(lifecycleSlotObservation(n, uint64(n)))
	}
	p.refresh()
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	select {
	case <-ep.reserved:
	case <-testwait.After(t):
		t.Fatal("not queued at busy hub")
	}
	if ep.deliveries() != 0 {
		t.Fatal("busy hub received input")
	}
	ep.mu.Lock()
	ep.busy = false
	ep.mu.Unlock()
	m := lifecycleNextMessage(t, ep.controlledEndpoint)
	if strings.Count(m.Body, "recovered") != 20 {
		t.Fatalf("not one20-slot summary: %s", m.Body)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
	r.s.Close()
	if ep.deliveries() != 1 {
		t.Fatalf("restart sent%d turns", ep.deliveries())
	}
}
func TestLifecyclePublisherReplacementNeverReplaysOldInput(t *testing.T) {
	oldEP := newControlledEndpoint()
	r, p, old := lifecyclePublisherRig(t, oldEP)
	p.observe(lifecycleSlotObservation(2, 1))
	first := lifecycleNextMessage(t, oldEP)
	next := old
	next.Nonce = "replacement"
	ep := newControlledEndpoint()
	r.world.mu.Lock()
	r.world.endpoints[next] = ep
	r.world.mu.Unlock()
	r.world.set(old, func(s *worldSlot) { s.binding = next })
	r.wrapper(next)
	r.waitConnected(next, true)
	r.waitConnected(old, false)
	later := lifecycleSlotObservation(2, 2)
	later.Kind = "parked"
	p.observe(later)
	p.signal()
	m := lifecycleNextMessage(t, ep)
	if m.To != next || m.ID == first.ID || strings.Contains(m.Body, "recovered") || !strings.Contains(m.Body, "parked") {
		t.Fatalf("replacement replay: %+v", m)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
	oldReceipt, known, _ := r.s.broker.ReceiptSnapshot(first.ID)
	if !known || oldReceipt.Status != couchmessage.Indeterminate || oldReceipt.Message.To != old {
		t.Fatalf("old receipt: %+v", oldReceipt)
	}
}
func TestLifecyclePublisherDrainsResolvableMetadataBeforeFreeze(t *testing.T) {
	ep := newControlledEndpoint()
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub := r.world.add(1)
	r.world.endpoints[hub] = ep
	r.attach(hub, "hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	// Start the shell after producing its inputs, so a concurrent refresh cannot
	// hide omission of the drain boundary's atomic metadata reduction.
	p := &lifecyclePublisher{service: r.s, ready: map[uint64]couchcore.LifecycleNoticeEffect{}, attempts: map[uint64]lifecycleAttempt{}, roots: map[string]bool{}, identities: map[string]couchcore.WorkspaceIdentity{}, unresolved: map[string]couchtty.LifecycleObservation{}, diagnostics: map[string]string{}, wake: make(chan struct{}, 1), results: make(chan lifecycleAdmission, 128)}
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	p.observe(lifecycleSlotObservation(2, 1))
	p.refresh()
	e := lifecycleSlotObservation(3, 2)
	e.Target = couchcore.ThreadTarget{}
	p.observe(e)
	worker := r.world.add(3)
	r.attach(worker, "worker3")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	r.s.workers.Add(1)
	go func() { defer r.s.workers.Done(); p.run(r.s.lifetime) }()
	m := lifecycleNextMessage(t, ep)
	if !strings.Contains(m.Body, "pair:2") || !strings.Contains(m.Body, "pair:3") {
		t.Fatalf("cohort frozen before resolving metadata: %s", m.Body)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
}
func TestLifecyclePublisherOrdinaryUnknownDoesNotHoldDrain(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	p.observe(lifecycleSlotObservation(2, 1))
	e := lifecycleSlotObservation(9, 2)
	e.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary}
	p.observe(e)
	e = lifecycleSlotObservation(10, 3)
	e.Target = couchcore.ThreadTarget{}
	p.observe(e)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	m := lifecycleNextMessage(t, ep)
	if strings.Contains(m.Body, "pair:9") || strings.Contains(m.Body, "pair:10") {
		t.Fatal("unknown metadata claimed slot evidence")
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
}
func TestLifecyclePublisherStaleNeedsRelaunchAddressCannotOverwriteCurrentSlot(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	worker := r.world.add(2)
	r.attach(worker, "worker")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	p.refresh()
	stale := lifecycleSlotObservation(2, 1)
	stale.Kind = "needs-relaunch"
	stale.Address.Tag = "old-thread"
	p.observe(stale)
	current := lifecycleSlotObservation(2, 2)
	current.Kind = "started"
	p.observe(current)
	m := lifecycleNextMessage(t, ep)
	if strings.Contains(m.Body, "needs-relaunch") || !strings.Contains(m.Body, "started") {
		t.Fatalf("stale address emitted: %s", m.Body)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
}

func TestLifecyclePublisherUncertainAdmissionRetiresOnceAndAllowsNewRevision(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(lifecycleSlotObservation(2, 1))
	first := lifecycleNextMessage(t, ep)
	var token uint64
	testwait.Until(t, "first admitted attempt", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		for k, a := range p.attempts {
			if a.id == first.ID && a.admitted {
				token = k
				return true
			}
		}
		return false
	})
	// Model a lost admission acknowledgement while canonical delivery continues.
	// Results are the publisher's existing broker boundary, not a retry seam.
	p.results <- lifecycleAdmission{token: token, err: couchmessage.ErrUncertain}
	later := lifecycleSlotObservation(2, 2)
	later.Kind = "parked"
	p.observe(later)
	testwait.Until(t, "new revision canonically queued behind uncertain input", func() bool {
		p.mu.Lock()
		var ids []string
		for _, a := range p.attempts {
			if a.id != first.ID {
				ids = append(ids, a.id)
			}
		}
		p.mu.Unlock()
		for _, id := range ids {
			receipt, known, _ := r.s.broker.ReceiptSnapshot(id)
			if known && receipt.Status == couchmessage.Queued && receipt.Message.Origin == couchmessage.OriginSystem && strings.Contains(receipt.Message.Body, "parked") {
				return true
			}
		}
		return false
	})
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, first)
	second := lifecycleNextMessage(t, ep)
	if second.ID == first.ID || strings.Contains(second.Body, "recovered") {
		t.Fatal("uncertain input replayed")
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, second)
}

func TestLifecyclePublisherAttemptCapacityRefusesWithoutSending(t *testing.T) {
	ep := newControlledEndpoint()
	diagnostic := make(chan string, 4)
	r, p, _ := lifecyclePublisherRig(t, ep, func(message string) {
		select {
		case diagnostic <- message:
		default:
		}
	})
	p.mu.Lock()
	for i := uint64(1000); i < 1128; i++ {
		p.attempts[i] = lifecycleAttempt{id: fmt.Sprint("unknown-", i), hub: "/hub", admitted: true, cancel: func() {}}
	}
	p.mu.Unlock()
	p.observe(lifecycleSlotObservation(2, 1))
	for {
		select {
		case message := <-diagnostic:
			if strings.Contains(message, "capacity") {
				if ep.deliveries() != 0 {
					t.Fatal("capacity-exhausted publisher sent input")
				}
				r.s.Close()
				return
			}
		case <-testwait.After(t):
			t.Fatal("capacity was not refused")
		}
	}
}

func TestLifecyclePublisherRegisteredPrimarySlotIsNotOrdinaryContent(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	worker := r.world.add(2)
	r.attach(worker, "worker")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	p.refresh()
	e := lifecycleSlotObservation(2, 1)
	e.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: e.Address}
	p.observe(e)
	m := lifecycleNextMessage(t, ep)
	if !strings.Contains(m.Body, worker.Slot) {
		t.Fatalf("registered primary slot missing: %s", m.Body)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
}

func TestLifecyclePublisherReplacementBeforeAdmissionReleasesFrozenToken(t *testing.T) {
	p := &lifecyclePublisher{ready: map[uint64]couchcore.LifecycleNoticeEffect{}, attempts: map[uint64]lifecycleAttempt{}, diagnostics: map[string]string{}}
	p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeConfirmed, Fact: couchcore.LifecycleNoticeFact{Hub: "/hub", Slot: "pair:2", Incarnation: "worker", Observation: 1, Kind: couchcore.LifecycleStarted}})
	p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "one", Ready: true})
	p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "two", Ready: true})
	if len(p.ready) != 1 {
		t.Fatalf("replacement stranded frozen token: %+v", p.ready)
	}
	for _, fx := range p.ready {
		if fx.Incarnation != "two" {
			t.Fatalf("old recipient retained: %+v", fx)
		}
	}
}
