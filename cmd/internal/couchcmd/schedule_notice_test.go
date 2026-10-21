package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func scheduleNoticeRig(t *testing.T, ep couchmessage.DeliveryEndpoint) (*serviceRig, scheduleNotice, couchmessage.Binding) {
	t.Helper()
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub := r.world.add(1)
	r.world.endpoints[hub] = ep
	r.attach(hub, "hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	return r, scheduleNotice{service: r.s}, hub
}
func TestScheduleNoticeAuthenticatedExactSystemDelivery(t *testing.T) {
	ep := newControlledEndpoint()
	r, n, hub := scheduleNoticeRig(t, ep)
	frozen, ok := n.Binding("/repo1")
	if !ok {
		t.Fatal("missing hub")
	}
	var got couchmessage.Binding
	if err := json.Unmarshal(frozen, &got); err != nil || got != hub {
		t.Fatal(got, err)
	}
	notice, err := n.Admit(t.Context(), "/repo1", frozen, "schedule-stable-id", "Task due at 2026-10-21T00:00:00Z", nil)
	if err != nil || notice != "" {
		t.Fatal(notice, err)
	}
	var m couchmessage.Message
	select {
	case m = <-ep.started:
	case <-testwait.After(t):
		t.Fatal("not delivered")
	}
	if m.ID != "schedule-stable-id" || m.From != hub || m.To != hub || m.Origin != couchmessage.OriginSystem {
		t.Fatalf("wrong authenticated notice: %+v", m)
	}
	if notice, known, active := n.Observe(m.ID); notice != "" || !known || !active {
		t.Fatal(notice, known, active)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	testwait.Until(t, "terminal scheduler receipt", func() bool {
		notice, known, active := n.Observe(m.ID)
		return notice == "submitted" && known && !active
	})
	if notice, err := n.Admit(t.Context(), "/repo1", frozen, m.ID, m.Body, nil); err != nil || notice != "submitted" {
		t.Fatal(notice, err)
	}
	if ep.deliveries() != 1 {
		t.Fatal("stable id redelivered")
	}
	if _, known, _ := n.Observe("missing"); known {
		t.Fatal("invented receipt")
	}
	_ = r
}
func TestScheduleNoticeFrozenReplacementAndOffline(t *testing.T) {
	ep := newControlledEndpoint()
	r, n, old := scheduleNoticeRig(t, ep)
	frozen, _ := n.Binding("/repo1")
	next := old
	next.Nonce = "replacement"
	nextEP := newControlledEndpoint()
	r.world.mu.Lock()
	r.world.endpoints[next] = nextEP
	r.world.mu.Unlock()
	r.world.set(old, func(s *worldSlot) { s.binding = next })
	r.wrapper(next)
	r.waitConnected(next, true)
	r.waitConnected(old, false)
	if notice, err := n.Admit(t.Context(), "/repo1", frozen, "stale-notice", "due", nil); notice != "dropped: hub changed" || err != nil {
		t.Fatal(notice, err)
	}
	if ep.deliveries() != 0 || nextEP.deliveries() != 0 {
		t.Fatal("replacement received frozen notice")
	}
	current, ok := n.Binding("/repo1")
	if !ok || string(current) == string(frozen) {
		t.Fatal("replacement not frozen freshly")
	}
	r.disconnect(next)
	r.waitConnected(next, false)
	if _, ok := n.Binding("/repo1"); ok {
		t.Fatal("offline binding")
	}
	if notice, err := n.Admit(t.Context(), "/repo1", current, "offline-notice", "due", nil); notice != "dropped: hub unavailable" || err != nil {
		t.Fatal(notice, err)
	}
}

type scheduleNoticeLateEndpoint struct {
	*controlledEndpoint
	release chan struct{}
}

func (e *scheduleNoticeLateEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.started <- m
	<-e.release
	return couchmessage.Receipt{}, ctx.Err()
}
func TestScheduleNoticeTerminalReceiptWaitsForActiveCallback(t *testing.T) {
	ep := &scheduleNoticeLateEndpoint{controlledEndpoint: newControlledEndpoint(), release: make(chan struct{})}
	defer func() {
		select {
		case <-ep.release:
		default:
			close(ep.release)
		}
	}()
	r, n, hub := scheduleNoticeRig(t, ep)
	frozen, _ := n.Binding("/repo1")
	if notice, err := n.Admit(t.Context(), "/repo1", frozen, "active-notice", "due", nil); notice != "" || err != nil {
		t.Fatal(notice, err)
	}
	select {
	case <-ep.started:
	case <-testwait.After(t):
		t.Fatal("not started")
	}
	r.s.broker.Disconnect(hub)
	receipt, known, active := r.s.broker.ReceiptSnapshot("active-notice")
	if !known || !active || !receipt.Status.Terminal() {
		t.Fatal(receipt, known, active)
	}
	if notice, known, active := n.Observe("active-notice"); notice != "" || !known || !active {
		t.Fatal("premature delivery finalization", notice, known, active)
	}
	close(ep.release)
	testwait.Until(t, "callback joined before terminal notice", func() bool {
		notice, known, active := n.Observe("active-notice")
		return known && !active && notice == scheduleNoticeReceipt(receipt)
	})
}
func TestScheduleNoticeReceiptAndAdmissionClassification(t *testing.T) {
	for _, tc := range []struct {
		status couchmessage.Status
		want   string
	}{{couchmessage.Queued, ""}, {couchmessage.Held, ""}, {couchmessage.Delivering, ""}, {couchmessage.Submitted, "submitted"}, {couchmessage.Expired, "expired"}, {couchmessage.Cancelled, "dropped: cancelled"}, {couchmessage.Indeterminate, "uncertain"}, {couchmessage.Refused, "dropped: refused"}} {
		if got := scheduleNoticeReceipt(couchmessage.Receipt{Status: tc.status}); got != tc.want {
			t.Fatal(tc.status, got, tc.want)
		}
	}
	if got := scheduleNoticeAdmissionError(&couchmessage.RefusedError{Reason: "before commit"}); got != "not-admitted" {
		t.Fatal(got)
	}
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, couchmessage.ErrUncertain, errors.New("lost commit acknowledgement")} {
		if got := scheduleNoticeAdmissionError(err); got != "uncertain" {
			t.Fatal(got)
		}
	}
}

func TestScheduleNoticeBusyHubRetainsCanonicalQueue(t *testing.T) {
	ep := &lifecycleBusyEndpoint{controlledEndpoint: newControlledEndpoint(), busy: true, reserved: make(chan struct{}, 1)}
	r, n, _ := scheduleNoticeRig(t, ep)
	frozen, _ := n.Binding("/repo1")
	if notice, err := n.Admit(t.Context(), "/repo1", frozen, "busy-notice", "due", nil); notice != "" || err != nil {
		t.Fatal(notice, err)
	}
	receipt, known, _ := r.s.broker.ReceiptSnapshot("busy-notice")
	if !known || receipt.Status != couchmessage.Queued || receipt.Message.Origin != couchmessage.OriginSystem {
		t.Fatal(receipt, known)
	}
	if notice, known, _ := n.Observe("busy-notice"); notice != "" || !known {
		t.Fatal(notice, known)
	}
	if ep.deliveries() != 0 {
		t.Fatal("busy hub pasted")
	}
	ep.mu.Lock()
	ep.busy = false
	ep.mu.Unlock()
	select {
	case m := <-ep.started:
		if m.ID != "busy-notice" {
			t.Fatal(m)
		}
	case <-testwait.After(t):
		t.Fatal("queue not delivered")
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	testwait.Until(t, "busy queue terminal receipt", func() bool {
		notice, known, active := n.Observe("busy-notice")
		return notice == "submitted" && known && !active
	})
	if ep.deliveries() != 1 {
		t.Fatal("queued notice replayed")
	}
}
func TestScheduleNoticeCancelledGateHasNoAdmission(t *testing.T) {
	ep := newControlledEndpoint()
	r, n, _ := scheduleNoticeRig(t, ep)
	frozen, _ := n.Binding("/repo1")
	r.s.authority.hubs = guardHubState(t)
	release, err := holdHubEffect(t.Context(), r.s.authority.hubs)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	notice, err := n.Admit(ctx, "/repo1", frozen, "gate-notice", "due", nil)
	if notice != "not-admitted" || !errors.Is(err, context.Canceled) {
		t.Fatal(notice, err)
	}
	if _, known, _ := r.s.broker.ReceiptSnapshot("gate-notice"); known || ep.deliveries() != 0 {
		t.Fatal("cancelled gate admitted")
	}
}

type scheduleNoticeDeadlineEndpoint struct{ *controlledEndpoint }

func (e scheduleNoticeDeadlineEndpoint) Reserve(ctx context.Context, _ string, _ uint64) error {
	<-ctx.Done()
	return ctx.Err()
}
func TestScheduleNoticeCanonicalRefusalIsKnownNonAdmission(t *testing.T) {
	ep := scheduleNoticeDeadlineEndpoint{newControlledEndpoint()}
	_, n, _ := scheduleNoticeRig(t, ep)
	frozen, _ := n.Binding("/repo1")
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	notice, err := n.Admit(ctx, "/repo1", frozen, "refused-notice", "due", nil)
	var refused *couchmessage.RefusedError
	if notice != "not-admitted" || !errors.As(err, &refused) || ep.deliveries() != 0 {
		t.Fatal(notice, err)
	}
}
