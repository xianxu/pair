package couchcmd

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// controlledEndpoint is a recipient whose delivery finishes when the test
// says, or never: it stands for a wrapper mid-paste.
type controlledEndpoint struct {
	mu        sync.Mutex
	delivered int
	started   chan couchmessage.Message
	finish    chan couchmessage.Receipt
}

func newControlledEndpoint() *controlledEndpoint {
	return &controlledEndpoint{started: make(chan couchmessage.Message, 4), finish: make(chan couchmessage.Receipt, 1)}
}
func (e *controlledEndpoint) Observe(ctx context.Context) (couchmessage.Observation, error) {
	return couchmessage.Observation{LastActivity: time.Now().Add(-time.Hour), Sequence: 1}, ctx.Err()
}
func (e *controlledEndpoint) Reserve(ctx context.Context, _ string, _ uint64) error { return ctx.Err() }
func (e *controlledEndpoint) Release(context.Context, string) error                 { return nil }
func (e *controlledEndpoint) Deliver(ctx context.Context, m couchmessage.Message) (couchmessage.Receipt, error) {
	e.mu.Lock()
	e.delivered++
	e.mu.Unlock()
	e.started <- m
	select {
	case r := <-e.finish:
		r.Message = m
		return r, nil
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
}
func (e *controlledEndpoint) deliveries() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.delivered
}

func (r *serviceRig) send(from couchmessage.Binding, id, target string) couchmessage.Response {
	req := callerRequest(from, "send")
	req.ID, req.Target, req.Body = id, target, "work"
	return r.s.handle(context.Background(), req)
}

func (r *serviceRig) waitStatus(from couchmessage.Binding, id string, ok func(couchmessage.Status) bool) couchmessage.Receipt {
	r.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp := r.s.handle(context.Background(), func() couchmessage.Request {
			q := callerRequest(from, "status")
			q.ID = id
			return q
		}())
		if resp.Receipt != nil && ok(resp.Receipt.Status) {
			return *resp.Receipt
		}
		if time.Now().After(deadline) {
			r.t.Fatalf("receipt for %s never settled: %+v", id, resp)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func terminal(s couchmessage.Status) bool { return s.Terminal() }

// An absent recipient refuses at once and dispatches nothing.
func TestLifecycleAbsentEndpoint(t *testing.T) {
	r := newServiceRig(t)
	from := r.connect(1)
	if resp := r.send(from, "to-nobody", "pair:9"); resp.Code != "unavailable" {
		t.Fatalf("exact absent target %+v", resp)
	}
	// The only member of the family is the sender, which is never selected.
	if resp := r.send(from, "to-family", "pair"); resp.Code != "not-dispatched" {
		t.Fatalf("family with no eligible slot %+v", resp)
	}
}

// A wrapper that dies mid-delivery leaves the outcome Indeterminate: the PTY
// may or may not have received the paste, and nothing is sent again.
func TestLifecycleWrapperDiesMidDelivery(t *testing.T) {
	r := newServiceRig(t)
	from := r.connect(1)
	to := r.world.add(2)
	ep := newControlledEndpoint()
	r.world.endpoints[to] = ep
	r.attach(to, "h2")
	_, stop := r.wrapper(to)
	r.waitConnected(to, true)
	if resp := r.send(from, "mid", to.Slot); resp.Code != "accepted" {
		t.Fatalf("send %+v", resp)
	}
	<-ep.started
	stop() // the wrapper's process ends: its session closes
	receipt := r.waitStatus(from, "mid", terminal)
	if receipt.Status != couchmessage.Indeterminate || ep.deliveries() != 1 {
		t.Fatalf("receipt %+v after %d deliveries", receipt, ep.deliveries())
	}
}

// A replacement wrapper takes over the slot, but a delivery already committed
// to the old incarnation ends there — completed or Indeterminate — and is
// never redirected to the new one.
func TestLifecycleReplacementCompletesAtOldIncarnation(t *testing.T) {
	for _, finishFirst := range []bool{true, false} {
		r := newServiceRig(t)
		from := r.connect(1)
		old := r.world.add(2)
		oldEP := newControlledEndpoint()
		r.world.endpoints[old] = oldEP
		r.attach(old, "h2")
		r.wrapper(old)
		r.waitConnected(old, true)
		if resp := r.send(from, "inflight", old.Slot); resp.Code != "accepted" {
			t.Fatalf("send %+v", resp)
		}
		<-oldEP.started
		if finishFirst {
			oldEP.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
			r.waitStatus(from, "inflight", terminal)
		}
		// Relaunch in the same slot: a new nonce, a new wrapper.
		next := old
		next.Nonce = "next-launch"
		newEP := newControlledEndpoint()
		r.world.endpoints[next] = newEP
		r.world.set(old, func(s *worldSlot) { s.binding = next })
		r.wrapper(next)
		r.waitConnected(next, true)
		r.waitConnected(old, false)
		receipt := r.waitStatus(from, "inflight", terminal)
		want := couchmessage.Indeterminate
		if finishFirst {
			want = couchmessage.Submitted
		}
		if receipt.Status != want || receipt.Message.To != old || newEP.deliveries() != 0 || oldEP.deliveries() != 1 {
			t.Fatalf("finishFirst=%v: receipt %+v, new deliveries %d, old %d", finishFirst, receipt, newEP.deliveries(), oldEP.deliveries())
		}
	}
}

// Frames from a displaced incarnation's still-open session change nothing for
// the binding that replaced it — including after an exec, where the old and
// new connections carry a byte-identical binding.
func TestLifecycleDelayedFramesFromOldIncarnationIgnored(t *testing.T) {
	for _, exec := range []bool{false, true} {
		r := newServiceRig(t)
		from := r.connect(1)
		old := r.world.add(2)
		r.attach(old, "h2")
		oldClient, _ := r.wrapper(old)
		r.waitConnected(old, true)
		next := old
		if !exec {
			next.Nonce = "next-launch"
			r.world.set(old, func(s *worldSlot) { s.binding = next })
		}
		before := r.world.counts()[0]
		r.wrapper(next)
		deadline := time.Now().Add(3 * time.Second)
		for r.world.counts()[0] == before || !r.s.isConnected(next) {
			if time.Now().After(deadline) {
				t.Fatalf("exec=%v: replacement never admitted", exec)
			}
			time.Sleep(5 * time.Millisecond)
		}
		r.mu.Lock()
		r.slotGit["/repo2"] = couchcore.SlotGitStatus{Branch: "main-slot2"}
		r.mu.Unlock()
		marker := time.Now().Add(24 * time.Hour)
		oldClient.Update(couchmessage.Observation{LastActivity: marker, Sequence: 99})
		oldClient.Submit(couchmessage.Observation{LastActivity: marker, Sequence: 100, Submission: 7})
		time.Sleep(200 * time.Millisecond)
		resp := r.s.handle(context.Background(), callerRequest(from, "actors"))
		found := false
		for _, a := range resp.Actors {
			if a.Binding == next {
				found = true
				if a.LastActivity.Equal(marker) {
					t.Fatalf("exec=%v: old incarnation's frame reached its replacement", exec)
				}
			} else if a.Binding == old {
				t.Fatalf("exec=%v: displaced binding listed", exec)
			}
		}
		if !found {
			t.Fatalf("exec=%v: replacement not listed: %+v", exec, resp)
		}
	}
}

// The whole lifecycle in one place: start, detach, reattach, wrapper restart
// (exec: identical binding, new connection) and Couch restart. Each step
// updates the registration through the protocol, and no late event from an
// earlier step erases a newer registration.
func TestLifecycleStartDetachReattachReplaceRestart(t *testing.T) {
	r := newServiceRig(t)
	from := r.connect(1)
	b := r.world.add(2)
	r.attach(b, "h2")
	_, stop := r.wrapper(b)
	r.waitConnected(b, true)

	r.attach(b, "")
	r.waitConnected(b, false)
	r.attach(b, "h2b")
	r.waitConnected(b, true)

	// Exec: the same binding on a new connection; the old one closes after.
	r.wrapper(b)
	stop()
	time.Sleep(100 * time.Millisecond)
	r.waitConnected(b, true)
	if resp := r.send(from, "after-exec", b.Slot); resp.Code != "accepted" {
		t.Fatalf("send after exec %+v", resp)
	}

	// Couch restart: surviving wrappers reconnect, the Console replays panes.
	r.restart(map[couchmessage.ThreadKey]couchmessage.PaneHandle{from.Thread(): "h1", b.Thread(): "h2b"})
	r.waitConnected(from, true)
	r.waitConnected(b, true)
	if resp := r.send(from, "after-restart", b.Slot); resp.Code != "accepted" {
		t.Fatalf("send after Couch restart %+v", resp)
	}
}
