package couchcmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

func coordinatorFixture(t *testing.T) (*switchCoordinator, *time.Time, couchmessage.Binding, couchmessage.Request, switchAdmission) {
	t.Helper()
	now := time.Unix(1000, 0)
	ops := newSlotOperations(nil, nil, func() time.Time { return now })
	c := newSwitchCoordinator(ops, time.Minute, 2*time.Minute)
	c.observe = func(context.Context, switchAdmission) couchcore.AgentSwitchEvent {
		return couchcore.AgentSwitchEvent{Source: "source", Alive: true, SettledKnown: true, Settled: true, GitKnown: true, Clean: true}
	}
	return c, &now, couchmessage.Binding{Scope: "scope", Tag: "caller"}, couchmessage.Request{Op: "switch-agent", ID: "id", Agent: "codex", Target: "repo:2", Confirmed: true}, switchAdmission{Source: "source", Target: "canonical:2", Content: couchcore.SwitchContent{Kind: couchcore.SwitchContentFresh}, Fallback: couchcore.SwitchContent{Kind: couchcore.SwitchContentFallback, Body: "previous session"}}
}
func TestAgentSwitchCoordinatorActivationAndDuplicates(t *testing.T) {
	c, _, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	count := 0
	c.enqueue = func(_ switchAdmission, _ couchcore.SwitchContent, started func(), finished func(any, error)) error {
		count++
		started()
		finished(couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted}, nil)
		return nil
	}
	first := c.admit(ctx, caller, r, a)
	if first.Code != "accepted" {
		t.Fatal(first)
	}
	c.step(ctx)
	if count != 0 {
		t.Fatal("effect before response activation")
	}
	if got := c.admit(ctx, caller, r, a); got.Code != "accepted" {
		t.Fatal(got)
	}
	changed := r
	changed.Agent = "claude"
	if got := c.admit(ctx, caller, changed, a); got.Code != "id-conflict" {
		t.Fatal(got)
	}
	c.activate(r.ID)
	c.step(ctx)
	c.step(ctx)
	if count != 1 {
		t.Fatal(count)
	}
	got := c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptSucceeded {
		t.Fatal(got.Operation)
	}
	other := caller
	other.Tag = "other"
	got = c.ops.handle(ctx, other, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptUnknown {
		t.Fatal("receipt leaked")
	}
}
func TestAgentSwitchCoordinatorCallbackAndDelivery(t *testing.T) {
	c, _, caller, r, a := coordinatorFixture(t)
	a.Content = couchcore.SwitchContent{}
	ctx := context.Background()
	requests := 0
	enqueues := 0
	c.requestBrief = func(context.Context, switchAdmission, string) error { requests++; return nil }
	orient := orientation.Request{SchemaVersion: 1, Tag: "target", Agent: "codex", Attempt: "attempt", Body: " exact brief\n"}
	c.enqueue = func(a switchAdmission, content couchcore.SwitchContent, started func(), finished func(any, error)) error {
		enqueues++
		if content.Body != orient.Body {
			t.Fatal("rewrote brief")
		}
		started()
		c.delivery(a.Address, orient, orientation.DeliveryState{Phase: orientation.DeliverySubmitted}, nil)
		finished(couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted, Orientation: &orient}, nil)
		return nil
	}
	if got := c.admit(ctx, caller, r, a); got.Code != "accepted" {
		t.Fatal(got)
	}
	c.activate(r.ID)
	c.step(ctx)
	if requests != 1 || enqueues != 0 {
		t.Fatal(requests, enqueues)
	}
	cb := couchmessage.Request{ID: "callback", SwitchRequest: r.ID, Agent: r.Agent, Body: orient.Body}
	if got := c.callback(ctx, caller, cb, "other"); got.Code == "accepted" {
		t.Fatal("foreign callback")
	}
	if got := c.callback(ctx, caller, cb, a.Source); got.Code != "accepted" {
		t.Fatal(got)
	}
	c.step(ctx)
	if enqueues != 0 {
		t.Fatal("switched before callback response")
	}
	c.activate(r.ID)
	c.step(ctx)
	got := c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptSucceeded || requests != 1 || enqueues != 1 {
		t.Fatal(got.Operation, requests, enqueues)
	}
}
func TestAgentSwitchCoordinatorFallbackCancellationAndCapacity(t *testing.T) {
	c, now, caller, r, a := coordinatorFixture(t)
	a.Content = couchcore.SwitchContent{}
	ctx := context.Background()
	c.requestBrief = func(context.Context, switchAdmission, string) error { return errors.New("send refused") }
	c.admit(ctx, caller, r, a)
	c.activate(r.ID)
	c.step(ctx)
	got := c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptRefused {
		t.Fatal(got.Operation)
	}
	*now = now.Add(couchmessage.ReceiptRetention + time.Second)
	c.step(ctx)
	if len(c.entries) != 0 {
		t.Fatal("retained expired switch")
	}
	for i := 0; i < maxSlotOperationReceipts; i++ {
		c.ops.receipts[receiptKey{id: time.Duration(i).String()}] = couchmessage.OperationReceipt{Status: couchmessage.ReceiptQueued}
	}
	if got := c.admit(ctx, caller, r, a); got.Code != "overloaded" {
		t.Fatal(got)
	}
}

func TestAgentSwitchCoordinatorFallbackAndLateCallback(t *testing.T) {
	c, now, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	a.Content = couchcore.SwitchContent{}
	notices, sends, queued := 0, 0, 0
	unlocked := func() {
		if !c.ops.mu.TryLock() {
			t.Fatal("hook ran under receipt mutex")
		}
		c.ops.mu.Unlock()
	}
	c.requestBrief = func(context.Context, switchAdmission, string) error { unlocked(); sends++; return nil }
	c.notice = func(string) { unlocked(); notices++ }
	c.observe = func(context.Context, switchAdmission) couchcore.AgentSwitchEvent {
		unlocked()
		return couchcore.AgentSwitchEvent{Source: a.Source, Alive: true, SettledKnown: true, Settled: true, GitKnown: true, Clean: true}
	}
	c.enqueue = func(_ switchAdmission, content couchcore.SwitchContent, _ func(), _ func(any, error)) error {
		unlocked()
		queued++
		if content != a.Fallback {
			t.Fatal("wrong fallback")
		}
		return nil
	}
	c.admit(ctx, caller, r, a)
	c.activate(r.ID)
	c.step(ctx)
	*now = now.Add(time.Minute)
	c.step(ctx)
	c.step(ctx)
	if notices != 1 || sends != 1 || queued != 1 {
		t.Fatal(notices, sends, queued)
	}
	cb := couchmessage.Request{SwitchRequest: r.ID, Agent: r.Agent, Body: "late"}
	if got := c.callback(ctx, caller, cb, a.Source); got.Code == "accepted" {
		t.Fatal("accepted late callback")
	}
	c.cancel(r.ID, "shutdown")
	c.step(ctx)
	if queued != 1 {
		t.Fatal("requeued after cancel")
	}
}
func TestAgentSwitchCoordinatorUnactivatedExpiresAndTargetReserved(t *testing.T) {
	c, now, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	c.admit(ctx, caller, r, a)
	second := r
	second.ID = "other"
	if got := c.admit(ctx, caller, second, a); got.Code != "busy" {
		t.Fatal(got)
	}
	*now = now.Add(2 * time.Minute)
	c.step(ctx)
	got := c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptRefused {
		t.Fatal(got.Operation)
	}
	c.activate(r.ID)
	c.step(ctx)
	if got := c.admit(ctx, caller, second, a); got.Code != "accepted" {
		t.Fatal(got)
	}
	c.cancel(second.ID, "response failed")
	got = c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: second.ID})
	if got.Operation.Detail != "response failed" || got.Operation.Status != couchmessage.ReceiptRefused {
		t.Fatal(got.Operation)
	}
}
func TestAgentSwitchCoordinatorDeliveryIdentity(t *testing.T) {
	c, _, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	a.Content = couchcore.SwitchContent{Kind: couchcore.SwitchContentAuthored, Body: "exact"}
	orient := orientation.Request{SchemaVersion: 1, Tag: "target", Agent: r.Agent, Attempt: "attempt", Body: "exact"}
	c.enqueue = func(_ switchAdmission, _ couchcore.SwitchContent, _ func(), finished func(any, error)) error {
		finished(couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted, Orientation: &orient}, nil)
		return nil
	}
	c.admit(ctx, caller, r, a)
	c.activate(r.ID)
	c.step(ctx)
	wrong := orient
	wrong.Attempt = "old"
	c.delivery(a.Address, wrong, orientation.DeliveryState{Phase: orientation.DeliverySubmitted}, nil)
	got := c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptRunning {
		t.Fatal(got.Operation)
	}
	c.delivery(a.Address, orient, orientation.DeliveryState{Phase: orientation.DeliveryIndeterminate}, nil)
	got = c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptFailed || got.Operation.Code != "unconfirmed" {
		t.Fatal(got.Operation)
	}
	c.delivery(a.Address, orient, orientation.DeliveryState{Phase: orientation.DeliverySubmitted}, nil)
	got = c.ops.handle(ctx, caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptFailed {
		t.Fatal("late delivery resurrected terminal")
	}
}

func TestAgentSwitchCoordinatorCallbackDeadlinePublishesFallback(t *testing.T) {
	c, now, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	a.Content = couchcore.SwitchContent{}
	notices := 0
	c.notice = func(string) {
		if !c.ops.mu.TryLock() {
			t.Fatal("notice under mutex")
		}
		c.ops.mu.Unlock()
		notices++
	}
	c.admit(ctx, caller, r, a)
	*now = now.Add(time.Minute)
	got := c.callback(ctx, caller, couchmessage.Request{SwitchRequest: r.ID, Agent: r.Agent, Body: "late"}, a.Source)
	if got.Code == "accepted" || notices != 1 {
		t.Fatal(got, notices)
	}
}

func TestAgentSwitchCoordinatorCancelledContextCannotEnqueue(t *testing.T) {
	c, _, caller, r, a := coordinatorFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	queued := 0
	c.enqueue = func(switchAdmission, couchcore.SwitchContent, func(), func(any, error)) error { queued++; return nil }
	c.admit(ctx, caller, r, a)
	c.activate(r.ID)
	cancel()
	c.step(ctx)
	if queued != 0 {
		t.Fatal("queued after shutdown")
	}
	got := c.ops.handle(context.Background(), caller, couchmessage.Request{Op: "operation-status", ID: r.ID})
	if got.Operation.Status != couchmessage.ReceiptRefused {
		t.Fatal(got.Operation)
	}
}

func TestAgentSwitchCoordinatorExistingBeforeReprobe(t *testing.T) {
	c, _, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	if _, ok := c.existing(caller, r); ok {
		t.Fatal("unadmitted request exists")
	}
	c.admit(ctx, caller, r, a)
	// Replacing the source changes admission evidence, never the original intent.
	a.Source = "replacement"
	a.Content = couchcore.SwitchContent{}
	if got := c.admit(ctx, caller, r, a); got.Code != "accepted" {
		t.Fatal(got)
	}
	if got, ok := c.existing(caller, r); !ok || got.Code != "accepted" || got.Operation.ID != r.ID {
		t.Fatal(got, ok)
	}
	changed := r
	changed.Body = "different"
	if got, ok := c.existing(caller, changed); !ok || got.Code != "id-conflict" {
		t.Fatal(got, ok)
	}
	other := caller
	other.Tag = "other"
	if _, ok := c.existing(other, r); ok {
		t.Fatal("existing receipt leaked across owners")
	}
}
func TestAgentSwitchCoordinatorCanExecuteBindsCancelledRequest(t *testing.T) {
	c, _, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	c.enqueue = func(switchAdmission, couchcore.SwitchContent, func(), func(any, error)) error { return nil }
	c.admit(ctx, caller, r, a)
	if c.canExecute(r.ID, a.Source) {
		t.Fatal("waiting request can execute")
	}
	c.activate(r.ID)
	c.step(ctx)
	if !c.canExecute(r.ID, a.Source) {
		t.Fatal("switching request cannot execute")
	}
	if c.canExecute("other", a.Source) || c.canExecute(r.ID, "replacement") {
		t.Fatal("uncorrelated request can execute")
	}
	c.cancel(r.ID, "cancelled")
	if c.canExecute(r.ID, a.Source) {
		t.Fatal("cancelled request can execute")
	}
	next := r
	next.ID = "new"
	c.admit(ctx, caller, next, a)
	c.activate(next.ID)
	c.step(ctx)
	if !c.canExecute(next.ID, a.Source) || c.canExecute(r.ID, a.Source) {
		t.Fatal("old request borrowed newer authority")
	}
}

func TestAgentSwitchCoordinatorObservationCarriesChosenContent(t *testing.T) {
	c, now, caller, r, a := coordinatorFixture(t)
	ctx := context.Background()
	a.Content = couchcore.SwitchContent{}
	c.requestBrief = func(context.Context, switchAdmission, string) error { return nil }
	var seen []couchcore.SwitchContent
	c.observe = func(_ context.Context, a switchAdmission) couchcore.AgentSwitchEvent {
		seen = append(seen, a.Content)
		return couchcore.AgentSwitchEvent{Source: a.Source, Alive: true}
	}
	c.admit(ctx, caller, r, a)
	c.activate(r.ID)
	c.step(ctx)
	*now = now.Add(time.Minute)
	c.step(ctx)
	if len(seen) != 2 || seen[0] != (couchcore.SwitchContent{}) || seen[1] != a.Fallback {
		t.Fatal(seen)
	}
}
