package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"sync/atomic"
	"testing"
	"time"
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
	service := &messageService{broker: b}
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
	d, b, _, hub := resourceBrokerRig(t)
	state := resourceTestAlert(time.Now())
	original := *hub
	hub.Slot = "absent:0"
	_ = d.dispatch(context.Background(), &state)
	if state.Episodes[0].Phase != "pending" {
		t.Fatal("refusal resolved episode")
	}
	*hub = original
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
