package couchcmd

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// sendBudgetEndpoint is healthy during enrollment, then holds Observe until
// the send's admission context expires, without starting a timer of its own.
type sendBudgetEndpoint struct {
	serviceEndpointFake
	armed    atomic.Bool
	deadline chan time.Time
}

// A stopped registry loop cannot keep a targeted send past its caller's
// deadline. The fixture releases a broken implementation before joining it.
func TestMessageSendTargetedPostHonorsCallerDeadline(t *testing.T) {
	r := newServiceRig(t)
	from := r.connect(1)
	stalled := &messageService{broker: r.s.broker, authority: r.s.authority,
		lifetime: context.Background(), inbox: make(chan messageInput)}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	send := callerRequest(from, "send")
	send.ID, send.Target, send.Body = "stalled-registry", "pair:2", "work"
	done := make(chan couchmessage.Response, 1)
	go func() { done <- stalled.handle(ctx, send) }()
	select {
	case response := <-done:
		if ctx.Err() == nil || response.Code != "refused" || !strings.Contains(response.Error, "deadline exceeded") {
			t.Fatalf("deadline refusal: %+v, caller error=%v", response, ctx.Err())
		}
	case <-testwait.After(t):
		// Unblock the original lifetime-only post, then join the request.
		<-stalled.inbox
		response := <-done
		t.Fatalf("targeted registry post ignored caller deadline: %+v", response)
	}
}

func (e *sendBudgetEndpoint) Observe(ctx context.Context) (couchmessage.Observation, error) {
	if !e.armed.Load() {
		return e.serviceEndpointFake.Observe(ctx)
	}
	deadline, _ := ctx.Deadline()
	e.deadline <- deadline
	<-ctx.Done()
	return couchmessage.Observation{}, ctx.Err()
}

// The sender check and recipient admission must spend the same budget, so
// even a slow sender check leaves time to return a definite refusal (#634).
func TestMessageSendSlowSenderStillReturnsRefusal(t *testing.T) {
	var armed atomic.Bool
	senderDeadline := make(chan time.Time, 1)
	endpoint := &sendBudgetEndpoint{deadline: make(chan time.Time, 1)}
	r := newServiceRig(t, func(a messageAuthority) messageAuthority {
		thread := a.thread
		a.thread = func(ctx context.Context, b couchmessage.Binding) (string, error) {
			if armed.Load() && b.Slot == "pair:1" {
				deadline, _ := ctx.Deadline()
				senderDeadline <- deadline
				timer := time.NewTimer(couchmessage.AdmissionTimeout * 3 / 4)
				defer timer.Stop()
				select {
				case <-timer.C:
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
			return thread(ctx, b)
		}
		return a
	})
	from := r.connect(1)
	to := r.world.add(2)
	r.world.mu.Lock()
	r.world.endpoints[to] = endpoint
	r.world.mu.Unlock()
	r.attach(to, "h2")
	r.wrapper(to)
	r.waitConnected(to, true)
	endpoint.armed.Store(true)
	armed.Store(true)
	send := callerRequest(from, "send")
	send.ID, send.Target, send.Body = "slow-sender", to.Slot, "work"
	var response couchmessage.Response
	started := time.Now()
	err := couchmessage.Call(context.Background(), r.brokerSock, send, &response)
	t.Logf("send exchange elapsed=%s response=%+v error=%v", time.Since(started), response, err)
	if err != nil || response.Code != "refused" || !strings.Contains(response.Error, "timed out") {
		t.Fatalf("broker refusal did not reach sender: %+v %v", response, err)
	}
	if sender, recipient := <-senderDeadline, <-endpoint.deadline; !sender.Equal(recipient) {
		t.Fatalf("admission restarted its budget: sender deadline=%s recipient deadline=%s", sender, recipient)
	}
	status := callerRequest(from, "status")
	status.ID = send.ID
	var receipt couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, status, &receipt); err != nil || receipt.Receipt == nil || receipt.Receipt.Status != couchmessage.Refused || !strings.Contains(receipt.Receipt.Detail, "timed out") {
		t.Fatalf("retained refusal: %+v %v", receipt, err)
	}
}
