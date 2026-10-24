package couchcmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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
	p.observe(couchtty.LifecycleObservation{Kind: "recovered", Path: "/repo2", Incarnation: "worker-one", Observation: 1, Initiation: "operator"})
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

func TestLifecyclePublisherDropsMovedOwnership(t *testing.T) {
	for _, phase := range []string{"before-drain", "frozen", "admitted", "uncertain"} {
		t.Run(phase, func(t *testing.T) {
			h, defs, slot := moveFixture(t)
			r := newServiceRig(t)
			r.s.authority.hubs = h
			r.s.automation = newHubAutomationOwner(map[string]bool{defs[0].Root: true, defs[1].Root: true}, couchcore.StartupUnknown, nil)
			r.s.hubMoves.Store(&hubMoveService{})
			endpoints := []*controlledEndpoint{newControlledEndpoint(), newControlledEndpoint()}
			var bindings []couchmessage.Binding
			for i := range endpoints {
				b := r.world.add(i)
				r.world.set(b, func(s *worldSlot) {
					n := 0
					s.root, s.workspace.WorktreeRoot = defs[i].Root, defs[i].Root
					s.workspace.PrimaryRoot, s.workspace.Slot = defs[i].Slot.Repository, &n
				})
				r.world.endpoints[b] = endpoints[i]
				r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("hub-%d", i)))
				r.wrapper(b)
				r.waitConnected(b, true)
				bindings = append(bindings, b)
			}
			p := startLifecyclePublisher(r.s, func(string) {})
			p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
			e := lifecycleSlotObservation(7, 1)
			e.Initiation = "operator"
			e.Target.Slot.PrimaryRoot = slot.Repository
			p.observe(e)
			var release func()
			if phase == "frozen" {
				var err error
				release, err = holdHubEffect(t.Context(), h)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if release != nil {
						release()
					}
				})
			}
			var old couchmessage.Message
			if phase != "before-drain" {
				p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
				if phase == "frozen" {
					testwait.Until(t, "frozen part waiting at admission gate", func() bool {
						p.mu.Lock()
						defer p.mu.Unlock()
						for _, a := range p.attempts {
							if !a.started {
								return true
							}
						}
						return false
					})
				} else {
					old = lifecycleNextMessage(t, endpoints[0], p)
					if phase == "uncertain" {
						endpoints[0].finish <- couchmessage.Receipt{Status: couchmessage.Indeterminate}
						testwait.Until(t, "original uncertain receipt", func() bool {
							receipt, known, busy := r.s.broker.ReceiptSnapshot(old.ID)
							return known && !busy && receipt.Status == couchmessage.Indeterminate
						})
					}
				}
			}
			move := func(gated bool) {
				t.Helper()
				call := couchcore.OperationCall{Name: "move-slot", Context: t.Context(), TypedPayload: hubMoveCall{service: r.s, caller: bindings[1], move: couchcore.HubMove{Caller: defs[1].Root, Target: defs[1].Root, Slots: []couchcore.HubSlot{slot}, Revision: h.snapshot().Revision()}}}
				var err error
				if gated {
					_, err = dispatchHubOperation(h, call, executeHubMove)
				} else {
					_, err = executeHubMove(call)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			move(release == nil)
			if release != nil {
				release()
				release = nil
			}
			if phase == "before-drain" {
				p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
			}
			var m couchmessage.Message
			select {
			case m = <-endpoints[1].started:
			case stale := <-endpoints[0].started:
				t.Fatalf("pending projection reached previous owner: %+v", stale)
			case <-testwait.After(t):
				t.Fatal("no ownership-change notice")
			}
			if m.Body != "Couch hub changed; earlier lifecycle notices were dropped; check Slots" {
				t.Fatalf("moved projection replayed instead of resnapshot notice: %+v", m)
			}
			endpoints[1].finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
			lifecycleCanonicalSubmitted(t, r, m)
			if phase == "admitted" {
				endpoints[0].finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
				lifecycleCanonicalSubmitted(t, r, old)
			}
			wantOld := 0
			if phase == "admitted" || phase == "uncertain" {
				wantOld = 1
			}
			if endpoints[0].deliveries() != wantOld {
				t.Fatal("old projection was submitted or replayed after move")
			}
			e.Observation, e.Kind = 2, "parked"
			p.observe(e)
			next := lifecycleNextMessage(t, endpoints[1], p)
			if !strings.Contains(next.Body, "pair:7: parked") || strings.Contains(next.Body, "recovered") {
				t.Fatalf("new fact routing: %+v", next)
			}
			endpoints[1].finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
			lifecycleCanonicalSubmitted(t, r, next)
			p.mu.Lock()
			generation := p.generation
			p.mu.Unlock()
			move(true) // an idempotent move does not drop new state or announce again
			p.mu.Lock()
			unchanged := generation == p.generation
			p.mu.Unlock()
			if !unchanged {
				t.Fatal("idempotent ownership move discarded projection")
			}
		})
	}
}
func lifecycleSlotObservation(n int, observation uint64) couchtty.LifecycleObservation {
	return couchtty.LifecycleObservation{Kind: "recovered", Path: fmt.Sprintf("/repo%d", n), Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "pair", Number: n, PrimaryRoot: "/repo1", WorktreeRoot: fmt.Sprintf("/repo%d", n)}}, Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: couchcore.ThreadTag(fmt.Sprintf("t%d", n))}, Incarnation: fmt.Sprintf("%d:start", n), Observation: observation, Initiation: "automatic"}
}

// lifecycleNextMessage waits for the next delivered notice. A timeout names
// the publisher's state and the cached workspaces it reads
// (pair#693), so a missing notice says where the fact stopped.
func lifecycleNextMessage(t *testing.T, ep *controlledEndpoint, p *lifecyclePublisher) couchmessage.Message {
	t.Helper()
	select {
	case m := <-ep.started:
		return m
	case <-testwait.After(t):
		t.Fatal("no lifecycle message: " + lifecyclePublisherState(p))
		return couchmessage.Message{}
	}
}

// lifecyclePublisherState is a bounded summary of where facts sit: waiting
// for an identity, cached identities, hub roots, ready and in-flight tokens,
// and the message service's verified workspaces.
func lifecyclePublisherState(p *lifecyclePublisher) string {
	var workspaces []string
	p.service.mu.Lock()
	for b, id := range p.service.workspaces {
		workspaces = append(workspaces, b.Slot+"="+id.WorktreeRoot)
	}
	p.service.mu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	var unresolved, identities, roots []string
	for path, u := range p.unresolved {
		unresolved = append(unresolved, fmt.Sprintf("%s#%d(%s)", path, u.fact.Observation, u.fact.Target.Kind))
	}
	for path := range p.identities {
		identities = append(identities, path)
	}
	for root := range p.roots {
		roots = append(roots, root)
	}
	state := fmt.Sprintf("unresolved=%v identities=%v roots=%v ready=%d attempts=%d workspaces=%v", unresolved, identities, roots, len(p.ready), len(p.attempts), workspaces)
	if len(state) > 4096 {
		state = state[:4096] + "..."
	}
	return state
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
	first := lifecycleNextMessage(t, ep, p)
	later := lifecycleSlotObservation(2, 2)
	later.Kind = "parked"
	p.observe(later)
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, first)
	second := lifecycleNextMessage(t, ep, p)
	if second.ID == first.ID || !strings.Contains(second.Body, "parked") || second.Body == first.Body {
		t.Fatalf("new snapshot: %+v", second)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, second)
}
func TestLifecyclePublisherTwentyExplicitActionsDuringRestartOneCanonicalSummary(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	for n := 2; n < 22; n++ {
		fact := lifecycleSlotObservation(n, uint64(n))
		fact.Initiation = "operator"
		p.observe(fact)
	}
	p.refresh()
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	m := lifecycleNextMessage(t, ep, p)
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
	_ = lifecycleNextMessage(t, ep, p)
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
func TestLifecyclePublisherBusyTwentyExplicitActionsDuringRestart(t *testing.T) {
	ep := &lifecycleBusyEndpoint{controlledEndpoint: newControlledEndpoint(), busy: true, reserved: make(chan struct{}, 1)}
	r, p, _ := lifecyclePublisherRig(t, ep)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	for n := 2; n < 22; n++ {
		fact := lifecycleSlotObservation(n, uint64(n))
		fact.Initiation = "operator"
		p.observe(fact)
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
	m := lifecycleNextMessage(t, ep.controlledEndpoint, p)
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
	first := lifecycleNextMessage(t, oldEP, p)
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
	m := lifecycleNextMessage(t, ep, p)
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
	p := &lifecyclePublisher{service: r.s, ready: map[uint64]couchcore.LifecycleNoticeEffect{}, attempts: map[uint64]lifecycleAttempt{}, roots: map[string]bool{}, identities: map[string]couchcore.WorkspaceIdentity{}, unresolved: map[string]unresolvedFact{}, diagnostics: map[string]string{}, wake: make(chan struct{}, 1), results: make(chan lifecycleAdmission, 128)}
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	fact := lifecycleSlotObservation(2, 1)
	fact.Initiation = "operator"
	p.observe(fact)
	p.refresh()
	e := lifecycleSlotObservation(3, 2)
	e.Target = couchcore.ThreadTarget{}
	e.Initiation = "peer"
	p.observe(e)
	worker := r.world.add(3)
	r.attach(worker, "worker3")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	r.s.workers.Add(1)
	go func() { defer r.s.workers.Done(); p.run(r.s.lifetime) }()
	m := lifecycleNextMessage(t, ep, p)
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
	fact := lifecycleSlotObservation(2, 1)
	fact.Initiation = "operator"
	p.observe(fact)
	e := lifecycleSlotObservation(9, 2)
	e.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary}
	e.Initiation = "peer"
	p.observe(e)
	e = lifecycleSlotObservation(10, 3)
	e.Target = couchcore.ThreadTarget{}
	e.Initiation = "peer"
	p.observe(e)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	m := lifecycleNextMessage(t, ep, p)
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
	m := lifecycleNextMessage(t, ep, p)
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
	first := lifecycleNextMessage(t, ep, p)
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
	second := lifecycleNextMessage(t, ep, p)
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
	r.waitWorkspace(worker)
	p.refresh()
	e := lifecycleSlotObservation(2, 1)
	e.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: e.Address}
	p.observe(e)
	m := lifecycleNextMessage(t, ep, p)
	if !strings.Contains(m.Body, worker.Slot) {
		t.Fatalf("registered primary slot missing: %s", m.Body)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
}

// A fact can reach the publisher before its slot's verified workspace is
// cached: the Console reports a start at launch, and the wrapper's session
// connects and verifies later (pair#693). An ordinary-target fact then waits
// for the identity instead of being dropped, and is published once the
// workspace is cached. Observed before the worker connects at all, the widest
// form of that window, so no interleaving luck is involved.
func TestLifecyclePublisherOrdinaryFactWaitsForItsWorkspace(t *testing.T) {
	ep := newControlledEndpoint()
	r, p, _ := lifecyclePublisherRig(t, ep)
	worker := r.world.add(2)
	e := lifecycleSlotObservation(2, 1)
	e.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: e.Address}
	p.observe(e)
	r.attach(worker, "worker")
	r.wrapper(worker)
	m := lifecycleNextMessage(t, ep, p)
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

// Facts waiting for an identity are bounded (pair#693): newest per path, and
// at 128 paths a new one evicts the oldest observation instead of being
// refused, so paths that never verify cannot crowd out one that will.
func TestLifecycleUnresolvedFactsAreBoundedOldestFirst(t *testing.T) {
	p := &lifecyclePublisher{unresolved: map[string]unresolvedFact{}}
	for i := 1; i <= 128; i++ {
		p.retainUnresolvedLocked(couchtty.LifecycleObservation{Path: fmt.Sprint("/never-", i), Observation: uint64(i)})
	}
	p.retainUnresolvedLocked(couchtty.LifecycleObservation{Path: "/never-5", Observation: 2})
	if p.unresolved["/never-5"].fact.Observation != 5 {
		t.Fatal("an older observation replaced a newer one")
	}
	p.retainUnresolvedLocked(couchtty.LifecycleObservation{Path: "/slot", Observation: 200})
	if len(p.unresolved) != 128 {
		t.Fatalf("unresolved grew to %d", len(p.unresolved))
	}
	if _, ok := p.unresolved["/never-1"]; ok {
		t.Fatal("the oldest observation survived the cap")
	}
	if p.unresolved["/slot"].fact.Observation != 200 {
		t.Fatal("a new path was refused at the cap")
	}
}

// The restart-drain diagnostic counts only untargeted facts: an ordinary row
// that never verifies may simply not be a slot, which is no missing metadata
// (pair#693).
func TestLifecycleDrainReportsOnlyUntargetedFacts(t *testing.T) {
	ep := newControlledEndpoint()
	diagnostic := make(chan string, 4)
	_, p, _ := lifecyclePublisherRig(t, ep, func(message string) {
		select {
		case diagnostic <- message:
		default:
		}
	})
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	e := lifecycleSlotObservation(9, 1)
	e.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary}
	e.Initiation = "peer"
	p.observe(e)
	e = lifecycleSlotObservation(10, 2)
	e.Target = couchcore.ThreadTarget{}
	e.Initiation = "peer"
	p.observe(e)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	for {
		select {
		case message := <-diagnostic:
			if strings.Contains(message, "metadata unverified") {
				if message != "Lifecycle: metadata unverified for 1 slot(s) at restart drain" {
					t.Fatalf("drain diagnostic: %q", message)
				}
				return
			}
		case <-testwait.After(t):
			t.Fatal("no drain diagnostic for the untargeted fact: " + lifecyclePublisherState(p))
		}
	}
}

// The projection cap reports itself with the pane to inspect instead of
// silently dropping the hub.
func TestLifecycleProjectionCapacityDiagnostic(t *testing.T) {
	ep := newControlledEndpoint()
	diagnostic := make(chan string, 4)
	_, p, _ := lifecyclePublisherRig(t, ep, func(message string) {
		select {
		case diagnostic <- message:
		default:
		}
	})
	p.mu.Lock()
	for i := 0; i < 128; i++ {
		p.roots[fmt.Sprintf("hub-%d", i)] = true
	}
	p.mu.Unlock()
	p.observe(lifecycleSlotObservation(2, 1))
	for {
		select {
		case message := <-diagnostic:
			if message == "Lifecycle: hub projection full; inspect Slots" {
				return
			}
		case <-testwait.After(t):
			t.Fatal("no capacity diagnostic: " + lifecyclePublisherState(p))
		}
	}
}

// An ordinary fact waits for its identity at most lifecycleOrdinaryWait; an
// untargeted one waits for the restart drain (pair#693).
func TestLifecycleOrdinaryFactExpires(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	p := &lifecyclePublisher{unresolved: map[string]unresolvedFact{}, now: func() time.Time { return now }}
	ordinary := couchtty.LifecycleObservation{Path: "/ordinary", Observation: 1, Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary}}
	p.retainUnresolvedLocked(ordinary)
	p.retainUnresolvedLocked(couchtty.LifecycleObservation{Path: "/untargeted", Observation: 2})
	now = now.Add(lifecycleOrdinaryWait)
	p.reducePendingLocked()
	if _, ok := p.unresolved["/ordinary"]; !ok {
		t.Fatal("an ordinary fact expired inside its wait")
	}
	now = now.Add(time.Second)
	p.reducePendingLocked()
	if _, ok := p.unresolved["/ordinary"]; ok {
		t.Fatal("an ordinary fact outlived its wait")
	}
	if _, ok := p.unresolved["/untargeted"]; !ok {
		t.Fatal("an untargeted fact expired before the drain")
	}
}

func lifecycleManualPublisher(t *testing.T, endpoints ...couchmessage.DeliveryEndpoint) (*serviceRig, *lifecyclePublisher) {
	t.Helper()
	return lifecycleManualPublisherWithAuthority(t, nil, endpoints...)
}

func lifecycleManualPublisherWithAuthority(t *testing.T, decorate func(messageAuthority) messageAuthority, endpoints ...couchmessage.DeliveryEndpoint) (*serviceRig, *lifecyclePublisher) {
	t.Helper()
	r := newServiceRig(t, decorate)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub := r.world.add(1)
	if len(endpoints) > 0 {
		r.world.endpoints[hub] = endpoints[0]
	}
	r.attach(hub, "hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	p := &lifecyclePublisher{service: r.s, ready: map[uint64]couchcore.LifecycleNoticeEffect{}, attempts: map[uint64]lifecycleAttempt{}, roots: map[string]bool{}, identities: map[string]couchcore.WorkspaceIdentity{}, unresolved: map[string]unresolvedFact{}, diagnostics: map[string]string{}, wake: make(chan struct{}, 1), results: make(chan lifecycleAdmission, 128)}
	return r, p
}

func TestLifecyclePublisherAutomationOffRetiresBeforeOn(t *testing.T) {
	for _, deferred := range []bool{false, true} {
		t.Run(fmt.Sprint(deferred), func(t *testing.T) {
			r, p := lifecycleManualPublisher(t)
			if _, err := r.s.controlAutomation("/repo1", "off"); err != nil {
				t.Fatal(err)
			}
			fact := lifecycleSlotObservation(2, 1)
			if deferred {
				fact.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: fact.Address}
			}
			p.observe(fact)
			if _, err := r.s.controlAutomation("/repo1", "on"); err != nil {
				t.Fatal(err)
			}
			if deferred {
				worker := r.world.add(2)
				r.attach(worker, "worker")
				r.wrapper(worker)
				r.waitConnected(worker, true)
			}
			p.refresh()
			if len(p.ready) != 0 {
				t.Fatal("automation on replayed fact observed off")
			}
			fact.Observation = 2
			p.observe(fact)
			p.refresh()
			if len(p.ready) != 1 {
				t.Fatalf("future fact lost: %s", lifecyclePublisherState(p))
			}
		})
	}
}

func TestLifecyclePublisherStartupBaselineDroppedEvenWhenOn(t *testing.T) {
	r, p := lifecycleManualPublisher(t)
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	p.observe(lifecycleSlotObservation(2, 1))
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	p.refresh()
	if len(p.ready) != 0 {
		t.Fatal("startup baseline automatically injected into surviving hub")
	}
	if _, err := r.s.controlAutomation("/repo1", "off"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.s.controlAutomation("/repo1", "on"); err != nil {
		t.Fatal(err)
	}
	p.refresh()
	if len(p.ready) != 0 {
		t.Fatal("startup facts reappeared after automation cycle")
	}
	explicit := lifecycleSlotObservation(3, 2)
	explicit.Initiation = "operator"
	p.observe(explicit)
	p.refresh()
	if len(p.ready) != 1 {
		t.Fatal("new explicit lifecycle fact did not remain eligible")
	}
}

func TestLifecyclePublisherAutomationEpochRetiresReadyBacklog(t *testing.T) {
	r, p := lifecycleManualPublisher(t)
	p.observe(lifecycleSlotObservation(2, 1))
	p.refresh()
	if len(p.ready) != 1 {
		t.Fatal("expected ready job before policy changed")
	}
	if _, err := r.s.controlAutomation("/repo1", "off"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.s.controlAutomation("/repo1", "on"); err != nil {
		t.Fatal(err)
	}
	p.refresh()
	if len(p.ready) != 0 {
		t.Fatal("off-on replayed a previously eligible job")
	}
	p.observe(lifecycleSlotObservation(2, 2))
	p.refresh()
	if len(p.ready) != 1 {
		t.Fatal("policy cycle blocked future facts")
	}
}

func TestLifecyclePublisherAutomationEpochFencesDelayedAdmission(t *testing.T) {
	ep := newControlledEndpoint()
	ownership, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{{Root: "/repo1", Workspace: "/", Slot: couchcore.HubSlot{Repository: "/repo1"}}})
	if err != nil {
		t.Fatal(err)
	}
	h := &hubRuntime{state: ownership}
	r, p := lifecycleManualPublisherWithAuthority(t, func(a messageAuthority) messageAuthority {
		a.hubs = h
		return a
	}, ep)
	release, err := holdHubEffect(t.Context(), h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if release != nil {
			release()
		}
	})
	r.s.workers.Add(1)
	go func() { defer r.s.workers.Done(); p.run(r.s.lifetime) }()
	p.observe(lifecycleSlotObservation(2, 1))
	oldID := ""
	testwait.Until(t, "old job blocked before admission", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		for _, a := range p.attempts {
			if !a.started {
				oldID = a.id
				return true
			}
		}
		return false
	})
	if _, err := r.s.controlAutomation("/repo1", "off"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.s.controlAutomation("/repo1", "on"); err != nil {
		t.Fatal(err)
	}
	// Retire at the policy boundary before releasing the older delayed worker.
	p.refresh()
	release()
	release = nil
	testwait.Until(t, "suppressed admission retired", func() bool { p.mu.Lock(); defer p.mu.Unlock(); return len(p.attempts) == 0 && len(p.ready) == 0 })
	p.mu.Lock()
	diagnostics := len(p.diagnostics)
	p.mu.Unlock()
	if diagnostics != 0 {
		t.Fatal("intentional suppression produced failure diagnostic")
	}
	// The broker never admitted the old body; a fresh fact starts a new input.
	if _, known, _ := r.s.broker.ReceiptSnapshot(oldID); known {
		t.Fatal("old policy epoch admitted generated input")
	}
	p.observe(lifecycleSlotObservation(2, 2))
	message := lifecycleNextMessage(t, ep, p)
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, message)
	if ep.deliveries() != 1 {
		t.Fatal("old body replayed")
	}
}

func TestLifecyclePublisherOriginalPermitReachesMailboxCommit(t *testing.T) {
	endpoint := newControlledEndpoint()
	r, p := lifecycleManualPublisher(t, endpoint)
	hub, ok := r.s.hubBindingAt("/repo1")
	if !ok {
		t.Fatal("fixture needs verified hub")
	}
	seen := flipAutomationAtReserve(t, r.s, hub, "/repo1")
	r.s.workers.Add(1)
	go func() { defer r.s.workers.Done(); p.run(r.s.lifetime) }()
	p.observe(lifecycleSlotObservation(2, 1))
	id := awaitAutomationReserve(t, seen)
	testwait.Until(t, "suppressed lifecycle admission retired", func() bool { p.mu.Lock(); defer p.mu.Unlock(); return len(p.attempts) == 0 && len(p.ready) == 0 })
	if _, known, _ := r.s.broker.ReceiptSnapshot(id); known {
		t.Fatal("stale lifecycle body entered mailbox")
	}
	p.mu.Lock()
	diagnostics := len(p.diagnostics)
	p.mu.Unlock()
	if diagnostics != 0 {
		t.Fatal("intentional suppression produced failure diagnostic")
	}
	p.observe(lifecycleSlotObservation(2, 2))
	message := lifecycleNextMessage(t, endpoint, p)
	endpoint.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, message)
	if endpoint.deliveries() != 1 {
		t.Fatal("suppressed lifecycle body replayed")
	}
}
