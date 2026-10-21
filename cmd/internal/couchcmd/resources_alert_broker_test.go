package couchcmd

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type resourceBrokerEndpoint struct {
	serviceEndpointFake
	busy       atomic.Bool
	deliveries chan couchmessage.Message
}

func (e *resourceBrokerEndpoint) Reserve(ctx context.Context, _ string, _ uint64) error {
	if e.busy.Load() {
		return couchmessage.ErrRecipientBusy
	}
	return ctx.Err()
}
func (e *resourceBrokerEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	select {
	case e.deliveries <- m:
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
	return couchmessage.Receipt{Status: couchmessage.Submitted}, nil
}
func resourceBrokerRig(t *testing.T) (resourceAlertDispatcher, *couchmessage.Broker, *resourceBrokerEndpoint, *couchmessage.Binding) {
	t.Helper()
	b := couchmessage.NewBroker(context.Background(), time.Now, func(context.Context, couchmessage.Binding) (bool, error) { return true, nil })
	t.Cleanup(func() { _ = b.Close() })
	hub := couchmessage.Binding{Slot: "ops:0", Repository: "/repo/.git", Scope: "scope", Tag: "hub", Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: 100, Start: "start"}
	ep := &resourceBrokerEndpoint{deliveries: make(chan couchmessage.Message, 8)}
	if err := b.Register(hub, ep); err != nil {
		t.Fatal(err)
	}
	service := &messageService{
		broker:     b,
		authority:  messageAuthority{hub: "/repo"},
		automation: newHubAutomationOwner(map[string]bool{"/repo": true}, couchcore.StartupUnknown, nil),
		workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{hub: {WorktreeRoot: "/repo"}},
	}
	coordinator := newSupervisionCoordinator(service, func(string) {})
	d := resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return hub, true }, send: func(ctx context.Context, to couchmessage.Binding, id, body string) (couchmessage.Receipt, error) {
		return coordinator.sendGenerated(ctx, to, to, id, body, false)
	}}
	return d, b, ep, &hub
}
func resourceTestAlert(at time.Time) couchcore.ResourceState {
	e, _ := couchcore.AdvanceResourceEpisode(couchcore.ResourceEpisode{Key: "rss/200/original", Metric: "process-rss"}, couchcore.ResourceEpisodeEvent{At: at, Known: true, High: true, RequiredSamples: 1, Effect: couchcore.ResourceEffect{Slot: "pair:9", Process: couchcore.ResourceProcessKey{PID: 200, Identity: "original"}, Name: "writer", Value: 9 << 30}})
	return couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{e}}
}
func TestResourceAlertBrokerExactHubSelfDeliveryWithoutWorker(t *testing.T) {
	d, b, ep, hub := resourceBrokerRig(t)
	state := resourceTestAlert(time.Now())
	if err := d.dispatch(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-ep.deliveries:
		if m.From != *hub || m.To != *hub {
			t.Fatalf("impersonated worker: %+v", m)
		}
	case <-time.After(time.Second):
		t.Fatal("no self delivery")
	}
	r, ok, _ := b.ReceiptSnapshot(state.Episodes[0].Effect.ID)
	if !ok || r.Message.From != *hub || state.Episodes[0].Phase != "notified" || state.Episodes[0].Delivery != couchcore.ResourceDeliveryAdmitted {
		t.Fatal("no admitted episode")
	}
	_ = d.dispatch(context.Background(), &state)
	select {
	case <-ep.deliveries:
		t.Fatal("duplicate admission")
	default:
	}
}

// The sweep's alert must survive the real broker exactly once per episode, in a
// body that names both the evidence and the reap's gate (#583).
func TestResourceAlertBrokerDeliversTheOrphanNvimOnce(t *testing.T) {
	d, _, ep, hub := resourceBrokerRig(t)
	state := orphanNvimAlertState(time.Now(), "/fleet/worktree/pair-slot1/pair")
	if err := d.dispatch(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-ep.deliveries:
		want := "Couch resource monitor: unattributed embedded nvim pid=4242 age=312h5m0s in /fleet/worktree/pair-slot1/pair; no live slot owns it. Inspect with couch --resources; reap with couch --reap-orphans --confirm."
		if m.Body != want {
			t.Fatalf("body %q", m.Body)
		}
		if m.From != *hub || m.To != *hub {
			t.Fatalf("impersonated worker: %+v", m.From)
		}
	case <-time.After(time.Second):
		t.Fatal("no orphan alert")
	}
	if state.Episodes[0].Phase != "notified" || state.Episodes[0].Delivery != couchcore.ResourceDeliveryAdmitted {
		t.Fatalf("episode %+v", state.Episodes[0])
	}
	_ = d.dispatch(context.Background(), &state)
	select {
	case m := <-ep.deliveries:
		t.Fatalf("duplicate orphan alert %+v", m)
	default:
	}
}

func TestResourceAlertBrokerBusyQueuesOnce(t *testing.T) {
	d, b, ep, _ := resourceBrokerRig(t)
	ep.busy.Store(true)
	state := resourceTestAlert(time.Now())
	if err := d.dispatch(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	r, ok, _ := b.ReceiptSnapshot(state.Episodes[0].Effect.ID)
	if !ok || r.Status != couchmessage.Queued || state.Episodes[0].Phase != "notified" {
		t.Fatalf("busy not admitted once: %+v", r)
	}
	_ = d.dispatch(context.Background(), &state)
	r, _, _ = b.ReceiptSnapshot(r.Message.ID)
	if r.Status != couchmessage.Queued {
		t.Fatal(r)
	}
}
func TestResourceAlertBrokerUncertainRetryUsesBrokerDedup(t *testing.T) {
	d, b, ep, _ := resourceBrokerRig(t)
	state := resourceTestAlert(time.Now())
	send := d.send
	calls := 0
	d.send = func(ctx context.Context, to couchmessage.Binding, id, body string) (couchmessage.Receipt, error) {
		calls++
		r, err := send(ctx, to, id, body)
		if calls == 1 && err == nil {
			return couchmessage.Receipt{}, couchmessage.ErrUncertain
		}
		return r, err
	}
	_ = d.dispatch(context.Background(), &state)
	if state.Episodes[0].Phase != "pending" {
		t.Fatal("uncertainty did not remain pending")
	}
	_ = d.dispatch(context.Background(), &state)
	if calls != 2 || state.Episodes[0].Phase != "notified" {
		t.Fatal("retry not reconciled by broker")
	}
	select {
	case <-ep.deliveries:
	case <-time.After(time.Second):
		t.Fatal("no delivery")
	}
	if _, ok, _ := b.ReceiptSnapshot(state.Episodes[0].Effect.ID); !ok {
		t.Fatal("no broker admission")
	}
	select {
	case <-ep.deliveries:
		t.Fatal("duplicate delivery")
	default:
	}
}
func TestResourceAlertBrokerTemporaryRefusalRetries(t *testing.T) {
	d, b, ep, hub := resourceBrokerRig(t)
	state := resourceTestAlert(time.Now())
	b.Disconnect(*hub)
	_ = d.dispatch(context.Background(), &state)
	if state.Episodes[0].Phase != "pending" {
		t.Fatal("refusal resolved episode")
	}
	if err := b.Register(*hub, ep); err != nil {
		t.Fatal(err)
	}
	_ = d.dispatch(context.Background(), &state)
	if _, ok, _ := b.ReceiptSnapshot(state.Episodes[0].Effect.ID); !ok || state.Episodes[0].Phase != "notified" {
		t.Fatal("restored hub did not admit")
	}
}
func TestResourceAlertBrokerRecoveryAndFreshRestartRearm(t *testing.T) {
	d, b, _, _ := resourceBrokerRig(t)
	at := time.Now()
	state := resourceTestAlert(at)
	_ = d.dispatch(context.Background(), &state)
	first := state.Episodes[0].Effect.ID
	e, _ := couchcore.AdvanceResourceEpisode(state.Episodes[0], couchcore.ResourceEpisodeEvent{At: at.Add(time.Minute), Known: true})
	e, _ = couchcore.AdvanceResourceEpisode(e, couchcore.ResourceEpisodeEvent{At: at.Add(2 * time.Minute), Known: true, High: true, RequiredSamples: 1, Effect: couchcore.ResourceEffect{Slot: "pair:9"}})
	state.Episodes[0] = e
	_ = d.dispatch(context.Background(), &state)
	second := state.Episodes[0].Effect.ID
	if first == second || state.Episodes[0].Phase != "notified" {
		t.Fatal("recovery did not rearm")
	}
	fresh := resourceTestAlert(at.Add(3 * time.Minute))
	_ = d.dispatch(context.Background(), &fresh)
	third := fresh.Episodes[0].Effect.ID
	if third == first || third == second || fresh.Episodes[0].Phase != "notified" {
		t.Fatal("fresh supervisor state did not permit advisory repeat")
	}
	for _, id := range []string{first, second, third} {
		if _, ok, _ := b.ReceiptSnapshot(id); !ok {
			t.Fatal("missing rearmed admission")
		}
	}
}

// The whole chain with nothing stubbed between its ends: a sample of a real
// process table must reach the hub as one alert, and only one, inside the
// episode. Every seam in between is where a future change could silently drop
// the only notice a 13-day leak ever gets (#583).
func TestResourceAlertBrokerEndToEndFromTheSweep(t *testing.T) {
	d, _, ep, _ := resourceBrokerRig(t)
	f := &samplerReader{
		rows:    []procutil.Process{{PID: 4242, PPID: 1, Identity: "1760000000.5", Name: "/opt/homebrew/bin/nvim"}},
		cwdPath: "/fleet/worktree/pair-slot1/pair/cmd",
	}
	s := testResourceSampler(f)
	at := time.Unix(1760100000, 0)
	s.alerts = d.dispatch
	s.slots = func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: []couchcore.ResourceSlot{{Slot: "pair:1", OwnershipKnown: true, Workspace: couchcore.WorkspaceIdentity{WorktreeRoot: "/fleet/worktree/pair-slot1/pair"}}}}
	}
	delivered := func(when string) couchmessage.Message {
		t.Helper()
		select {
		case m := <-ep.deliveries:
			return m
		case <-time.After(time.Second):
			t.Fatalf("%s: nothing reached the hub", when)
		}
		return couchmessage.Message{}
	}
	quiet := func(when string) {
		t.Helper()
		select {
		case m := <-ep.deliveries:
			t.Fatalf("%s: repeated alert %q", when, m.Body)
		default:
		}
	}
	s.now = func() time.Time { return at }
	s.pass(context.Background())
	select {
	case m := <-ep.deliveries:
		t.Fatalf("one sample alerted the hub: %q", m.Body)
	default:
	}
	s.now = func() time.Time { return at.Add(time.Minute) }
	s.pass(context.Background())
	body := delivered("the second sample").Body
	// The effect binds the age at the sample that fired it — 100059.999995s, the
	// darwin stamp's fraction being microseconds — rounded to a readable minute.
	for _, want := range []string{"pid=4242", "age=27h48m0s", "in /fleet/worktree/pair-slot1/pair", "couch --reap-orphans --confirm"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body lost %q: %q", want, body)
		}
	}
	s.now = func() time.Time { return at.Add(2 * time.Minute) }
	s.pass(context.Background())
	quiet("the third sample")
	// Reaped: the episode clears, so a later leak by the same pid is a new alert.
	f.rows = nil
	s.now = func() time.Time { return at.Add(3 * time.Minute) }
	s.pass(context.Background())
	quiet("after the reap")
	for _, e := range s.state.Episodes {
		if strings.HasPrefix(e.Key, "nvim/") {
			t.Fatalf("episode survived the reap: %+v", e)
		}
	}
}
