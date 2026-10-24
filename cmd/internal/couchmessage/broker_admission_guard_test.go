package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type policyEndpoint struct {
	*FakeDeliveryEndpoint
	observe func()
	reserve func() error
}

func (e *policyEndpoint) Observe(ctx context.Context) (Observation, error) {
	if e.observe != nil {
		e.observe()
	}
	return e.FakeDeliveryEndpoint.Observe(ctx)
}
func (e *policyEndpoint) Reserve(ctx context.Context, id string, seq uint64) error {
	if e.reserve != nil {
		if err := e.reserve(); err != nil {
			return err
		}
	}
	return e.FakeDeliveryEndpoint.Reserve(ctx, id, seq)
}

// Exercise the production generated-message admission boundary.
func sendWithAdmissionPolicy(b *Broker, from Binding, id string, route Route, guard func(Binding) (func(), error)) (Receipt, error) {
	return b.SendSystem(context.Background(), from, id, route, "automatic", guard)
}

func TestSystemAdmissionPolicyChecksMailboxCommit(t *testing.T) {
	for _, phase := range []string{"families", "observe", "reserve", "busy-observe", "busy-reserve", "already-busy"} {
		for _, family := range []bool{false, true} {
			if family && (phase == "already-busy" || phase == "busy-observe" || phase == "busy-reserve") {
				continue
			}
			t.Run(fmt.Sprintf("%s/family=%v", phase, family), func(t *testing.T) {
				now := time.Unix(1000, 0)
				b := NewBroker(context.Background(), func() time.Time { return now }, func(context.Context, Binding) (bool, error) { return true, nil })
				defer b.Close()
				from, to := brokerBinding("pair:1"), brokerBinding("ops:0")
				ep := &policyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now.Add(-time.Minute))}
				_ = b.Register(from, newFakeEndpoint(now))
				_ = b.Register(to, ep)
				now = now.Add(time.Minute)
				allowed := true
				blocked := errors.New("automation occurrence retired")
				if phase == "families" || phase == "already-busy" {
					b.SetFamilies(func(context.Context) (map[string]string, error) { allowed = false; return nil, nil })
				}
				if phase == "observe" || phase == "busy-observe" {
					ep.observe = func() { allowed = false }
				}
				if phase == "reserve" || phase == "busy-reserve" {
					ep.reserve = func() error {
						allowed = false
						if phase == "busy-reserve" {
							return ErrRecipientBusy
						}
						return nil
					}
				}
				if phase == "busy-observe" {
					ep.reserve = func() error { return ErrRecipientBusy }
				}
				if phase == "already-busy" {
					b.mu.Lock()
					b.actors[to].reservation = "other"
					b.mu.Unlock()
				}
				route := Route{Target: to.Slot}
				if family {
					route.Target = "ops"
				}
				calls := 0
				_, err := sendWithAdmissionPolicy(b, from, "late", route, func(actual Binding) (func(), error) {
					calls++
					if actual != to {
						t.Errorf("recipient=%+v", actual)
					}
					if !allowed {
						return nil, blocked
					}
					return func() {}, nil
				})
				if !errors.Is(err, blocked) {
					t.Fatalf("policy changed during %s but mailbox admitted: %v (guard calls %d)", phase, err, calls)
				}
				b.mu.Lock()
				a := b.actors[to]
				_, pending := a.state.Pending()
				queued := a.state.Queued()
				inbox := len(a.inbox)
				b.mu.Unlock()
				if pending || queued != 0 || inbox != 0 {
					t.Fatalf("refusal mutated mailbox: pending=%v queued=%d inbox=%d", pending, queued, inbox)
				}
			})
		}
	}
}

func TestSystemAdmissionPolicyLockSpansCommitAndDedupRemainsCanonical(t *testing.T) {
	now := time.Unix(1000, 0)
	b, hub, ep, senders := queueBroker(t, now, "pair:1")
	var policy sync.Mutex
	calls, releases := 0, 0
	guard := func(to Binding) (func(), error) {
		if to != hub {
			t.Errorf("recipient=%+v", to)
		}
		if b.mu.TryLock() {
			b.mu.Unlock()
			t.Error("guard called without broker lock")
		}
		policy.Lock()
		calls++
		return func() {
			if policy.TryLock() {
				policy.Unlock()
				t.Error("policy released before commit")
			}
			if b.mu.TryLock() {
				b.mu.Unlock()
				t.Error("broker released before policy")
			}
			if _, ok := b.receipts["first"]; !ok {
				t.Error("policy released before receipt publication")
			}
			releases++
			policy.Unlock()
		}, nil
	}
	first, err := sendWithAdmissionPolicy(b, senders["pair:1"], "first", Route{Target: hub.Slot}, guard)
	if err != nil {
		t.Fatal(err)
	}
	delivered(t, ep)
	blocked := errors.New("now off")
	again, err := sendWithAdmissionPolicy(b, senders["pair:1"], "first", Route{Target: hub.Slot}, func(Binding) (func(), error) { t.Error("canonical receipt rechecked policy"); return nil, blocked })
	if err != nil || again.Message != first.Message {
		t.Fatalf("canonical admission retracted: %+v %v", again, err)
	}
	if calls != 1 || releases != 1 {
		t.Fatalf("guard/release=%d/%d", calls, releases)
	}
}

func TestSystemAdmissionPolicyCoversBothQueueCommits(t *testing.T) {
	for _, path := range []string{"already-busy", "busy-after-reserve"} {
		t.Run(path, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b := NewBroker(context.Background(), func() time.Time { return now }, nil)
			defer b.Close()
			from, to := brokerBinding("pair:1"), brokerBinding("ops:0")
			ep := &policyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now)}
			_ = b.Register(from, newFakeEndpoint(now))
			_ = b.Register(to, ep)
			if path == "already-busy" {
				b.mu.Lock()
				b.actors[to].reservation = "other"
				b.mu.Unlock()
			} else {
				ep.reserve = func() error { return ErrRecipientBusy }
			}
			calls, releases := 0, 0
			r, err := sendWithAdmissionPolicy(b, from, "queued", Route{Target: to.Slot}, func(actual Binding) (func(), error) {
				if actual != to {
					t.Errorf("recipient=%+v", actual)
				}
				if b.mu.TryLock() {
					b.mu.Unlock()
					t.Error("guard outside broker lock")
				}
				calls++
				return func() {
					releases++
					if b.mu.TryLock() {
						b.mu.Unlock()
						t.Error("release outside broker lock")
					}
					if b.actors[to].state.Queued() != 1 {
						t.Error("release preceded enqueue")
					}
					if _, ok := b.receipts["queued"]; !ok {
						t.Error("release preceded receipt")
					}
				}, nil
			})
			if err != nil || r.Status != Queued || calls != 1 || releases != 1 {
				t.Fatalf("queued=%+v error=%v guard/release=%d/%d", r, err, calls, releases)
			}
		})
	}
}

func TestSystemAdmissionPolicyDoesNotRetractRetainedQueuedReceipt(t *testing.T) {
	now := time.Unix(1000, 0)
	b, hub, ep, senders := queueBroker(t, now, "pair:1")
	sender := senders["pair:1"]
	if _, err := b.Send(context.Background(), sender, "first", Route{Target: hub.Slot}, "peer"); err != nil {
		t.Fatal(err)
	}
	delivered(t, ep)
	var calls atomic.Int32
	queued, err := sendWithAdmissionPolicy(b, sender, "retained", Route{Target: hub.Slot}, func(Binding) (func(), error) {
		calls.Add(1)
		return func() {}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	retained := queued
	retained.Status = Submitted
	ep.mu.Lock()
	ep.retained.Add(retained)
	ep.mu.Unlock()
	ep.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, b, sender, "retained", Submitted)
	got, err := sendWithAdmissionPolicy(b, sender, "retained", Route{Target: hub.Slot}, func(Binding) (func(), error) {
		t.Error("retained canonical receipt reran policy")
		return nil, errors.New("off")
	})
	if err != nil || got.Message != queued.Message || got.Status != Submitted || calls.Load() != 1 {
		t.Fatalf("canonical receipt=%+v error=%v calls=%d", got, err, calls.Load())
	}
}
