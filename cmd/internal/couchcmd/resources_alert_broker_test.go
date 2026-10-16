package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
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
func resourceBrokerRig(t *testing.T) (resourceAlertDispatcher, *couchmessage.Broker, *resourceBrokerEndpoint, *resourceCheckpoint, *couchmessage.Binding) {
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
	d := resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return hub, true }, connected: func(want couchmessage.Binding) bool { return want == hub }, send: func(ctx context.Context, a resourceAlertAttempt) (couchmessage.Receipt, error) {
		return coordinator.sendGenerated(ctx, a.From, a.To, a.ID, a.Body, false)
	}, lookup: func(_ context.Context, a resourceAlertAttempt) (couchmessage.Receipt, bool) {
		r, ok, _ := b.ReceiptSnapshot(a.ID)
		return r, ok
	}}
	cp := &resourceCheckpoint{Version: 1, State: couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{{Key: "rss/200/start", Metric: "process-rss", Phase: "pending", Effect: &couchcore.ResourceEffect{ID: "resource-real-broker", Metric: "process-rss", Slot: "pair:9", Process: couchcore.ResourceProcessKey{PID: 200, Identity: "old"}, Value: 9 << 30}}}}}
	return d, b, ep, cp, &hub
}
func resourceSaveCopy(dst *[]byte) func(resourceCheckpoint) error {
	return func(cp resourceCheckpoint) error { raw, e := json.Marshal(cp); *dst = raw; return e }
}
func TestResourceAlertBrokerExactHubSelfDeliveryWithoutWorker(t *testing.T) {
	d, b, ep, cp, hub := resourceBrokerRig(t)
	var saved []byte
	save := resourceSaveCopy(&saved)
	if err := d.dispatch(context.Background(), cp, save); err != nil {
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
	r, ok, _ := b.ReceiptSnapshot(cp.State.Episodes[0].Effect.ID)
	if !ok || r.Message.From != *hub || (cp.State.Episodes[0].Phase != "notified" || cp.State.Episodes[0].Delivery != couchcore.ResourceDeliveryAdmitted) {
		t.Fatal("no admitted episode")
	}
	if err := d.dispatch(context.Background(), cp, save); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ep.deliveries:
		t.Fatal("duplicate admission")
	default:
	}
}
func TestResourceAlertBrokerBusyQueuesOnce(t *testing.T) {
	d, b, ep, cp, _ := resourceBrokerRig(t)
	ep.busy.Store(true)
	if err := d.dispatch(context.Background(), cp, func(resourceCheckpoint) error { return nil }); err != nil {
		t.Fatal(err)
	}
	r, ok, _ := b.ReceiptSnapshot(cp.State.Episodes[0].Effect.ID)
	if !ok || r.Status != couchmessage.Queued || (cp.State.Episodes[0].Phase != "notified" || cp.State.Episodes[0].Delivery != couchcore.ResourceDeliveryAdmitted) {
		t.Fatalf("busy not admitted once: %+v %+v", r, cp)
	}
	if err := d.dispatch(context.Background(), cp, func(resourceCheckpoint) error { return nil }); err != nil {
		t.Fatal(err)
	}
	r, _, _ = b.ReceiptSnapshot(r.Message.ID)
	if r.Status != couchmessage.Queued {
		t.Fatal(r)
	}
}
func TestResourceAlertBrokerCrashIntentNeverResendsMissingReceipt(t *testing.T) {
	d, b, ep, cp, _ := resourceBrokerRig(t)
	var saved []byte
	save := resourceSaveCopy(&saved)
	sent := errors.New("simulated crash before broker send")
	d.send = func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, error) {
		return couchmessage.Receipt{}, sent
	}
	if err := d.dispatch(context.Background(), cp, save); err != nil {
		t.Fatal(err)
	}
	var restored resourceCheckpoint
	if err := json.Unmarshal(saved, &restored); err != nil {
		t.Fatal(err)
	}
	d.send = func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, error) {
		t.Fatal("missing receipt authorized resend")
		return couchmessage.Receipt{}, nil
	}
	if err := d.dispatch(context.Background(), &restored, save); err != nil {
		t.Fatal(err)
	}
	if restored.Attempts[0].State != "unknown" || restored.State.Episodes[0].Delivery != couchcore.ResourceDeliveryUnknown {
		t.Fatal(restored.Attempts)
	}
	if _, ok, _ := b.ReceiptSnapshot(restored.Attempts[0].ID); ok {
		t.Fatal("unexpected admission")
	}
	select {
	case <-ep.deliveries:
		t.Fatal("unexpected delivery")
	default:
	}
}
func TestResourceAlertBrokerFailedCheckpointNoAdmission(t *testing.T) {
	d, b, _, cp, _ := resourceBrokerRig(t)
	err := d.dispatch(context.Background(), cp, func(resourceCheckpoint) error { return errors.New("disk full") })
	if err == nil {
		t.Fatal("ignored checkpoint failure")
	}
	if _, ok, _ := b.ReceiptSnapshot(cp.State.Episodes[0].Effect.ID); ok {
		t.Fatal("admitted without checkpoint")
	}
}
func TestResourceAlertRefusedOwnershipMoveDoesNotResend(t *testing.T) {
	d, _, _, cp, hub := resourceBrokerRig(t)
	attempts := 0
	d.send = func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, error) {
		attempts++
		return couchmessage.Receipt{}, couchmessage.ErrUnavailable
	}
	save := func(resourceCheckpoint) error { return nil }
	if err := d.dispatch(context.Background(), cp, save); err != nil {
		t.Fatal(err)
	}
	old := *hub
	hub.Slot = "other:0"
	d.connected = func(b couchmessage.Binding) bool { return b == old || b == *hub }
	if err := d.dispatch(context.Background(), cp, save); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("resent to former owning hub %d times", attempts)
	}
}
func TestResourceAlertBrokerRestartReconcilesAdmittedReceipt(t *testing.T) {
	d, _, _, cp, _ := resourceBrokerRig(t)
	var saved []byte
	writes := 0
	save := func(value resourceCheckpoint) error {
		writes++
		if writes == 2 {
			return errors.New("crash before accepted checkpoint")
		}
		return resourceSaveCopy(&saved)(value)
	}
	if err := d.dispatch(context.Background(), cp, save); err == nil {
		t.Fatal("expected post-admission persistence failure")
	}
	var restored resourceCheckpoint
	if err := json.Unmarshal(saved, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Attempts[0].State != "attempting" {
		t.Fatal("lost write-ahead intent")
	}
	d.send = func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, error) {
		t.Fatal("resent admitted intent")
		return couchmessage.Receipt{}, nil
	}
	if err := d.dispatch(context.Background(), &restored, func(resourceCheckpoint) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if (restored.State.Episodes[0].Phase != "notified" || restored.State.Episodes[0].Delivery != couchcore.ResourceDeliveryAdmitted) || restored.Attempts[0].State != "admitted" {
		t.Fatal("receipt did not reconcile")
	}
}
