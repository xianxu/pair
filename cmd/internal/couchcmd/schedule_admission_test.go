package couchcmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type heldScheduleAdmission struct {
	scheduleNoticePort
	entered  chan struct{}
	finished chan struct{}
}

func (n heldScheduleAdmission) Admit(ctx context.Context, root string, binding json.RawMessage, id, body string, authority scheduleNoticeAuthority) (string, error) {
	close(n.entered)
	defer close(n.finished)
	return n.scheduleNoticePort.Admit(ctx, root, binding, id, body, authority)
}

func TestScheduleAdmissionGateRechecksDeadlineAndCancellation(t *testing.T) {
	for _, name := range []string{"deadline", "deadline-held", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			c, roots, initial := scheduleServiceFixture(t)
			root := roots[0]
			c.roots = roots[:1]
			hubs := guardHubState(t)
			r := newServiceRig(t, func(a messageAuthority) messageAuthority { a.hubs = hubs; return a })
			r.world.hub = root
			r.restart(nil)
			hub := r.world.add(1)
			r.world.set(hub, func(s *worldSlot) { s.root = root; s.workspace.WorktreeRoot = root })
			ep := newControlledEndpoint()
			r.world.endpoints[hub] = ep
			r.attach(hub, "scheduled-hub")
			r.wrapper(hub)
			r.waitConnected(hub, true)
			release, err := holdHubEffect(t.Context(), r.s.authority.hubs)
			if err != nil {
				t.Fatal(err)
			}
			held := true
			defer func() {
				if held {
					release()
				}
			}()
			n := heldScheduleAdmission{scheduleNoticePort: scheduleNotice{service: r.s}, entered: make(chan struct{}), finished: make(chan struct{})}
			clock := newSchedulerClockFake(initial.Add(50 * time.Second))
			a := newAutomaticScheduler(c, nil, n)
			startSchedulerClockRun(t, a, clock)
			clock.nextTimer(t)
			due := initial.Add(time.Minute)
			clock.advance(t, due)
			select {
			case <-n.entered:
			case <-testwait.After(t):
				t.Fatal("admission did not enter gate wait")
			}
			key := couchschedule.TaskKey{Hub: root, Task: "report"}
			state := clockLedger(t, c).Tasks[key.ID()]
			if state.Active == nil {
				t.Fatal("missing intent")
			}
			id := state.Active.ReceiptID
			// A gate-waiting notice must not own scheduler authority. This also
			// rejects reversing the documented hub-effect -> authority order.
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			if err := c.store.Update(ctx, func(*couchschedule.Ledger) error { return nil }); err != nil {
				t.Fatal("waiting admission blocked authority", err)
			}
			if name == "cancelled" {
				if err := os.RemoveAll(filepath.Join(root, "scripts", "scheduled", "report")); err != nil {
					t.Fatal(err)
				}
				if err := c.refresh(t.Context()); err != nil {
					t.Fatal(err)
				}
			} else {
				clock.advance(t, state.Active.NoticeDeadline)
			}
			if name != "deadline-held" {
				release()
				held = false
			}
			select {
			case <-n.finished:
			case <-testwait.After(t):
				t.Fatal("admission did not finish")
			}
			if receipt, known, _ := r.s.broker.ReceiptSnapshot(id); known {
				t.Fatalf("invalidated notice admitted: %+v", receipt)
			}
			if ep.deliveries() != 0 {
				t.Fatal("invalidated notice delivered")
			}
		})
	}
}

type blockedScheduleReservation struct {
	*controlledEndpoint
	reserving chan struct{}
}

func (e blockedScheduleReservation) Reserve(ctx context.Context, _ string, _ uint64) error {
	close(e.reserving)
	<-ctx.Done()
	return ctx.Err()
}

func TestScheduleAdmissionDeadlineReleasesAuthorityFromBrokerWait(t *testing.T) {
	c, roots, initial := scheduleServiceFixture(t)
	root := roots[0]
	c.roots = roots[:1]
	r := newServiceRig(t)
	r.world.hub = root
	r.restart(nil)
	hub := r.world.add(1)
	r.world.set(hub, func(s *worldSlot) { s.root = root; s.workspace.WorktreeRoot = root })
	ep := blockedScheduleReservation{newControlledEndpoint(), make(chan struct{})}
	r.world.endpoints[hub] = ep
	r.attach(hub, "scheduled-hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	clock := newSchedulerClockFake(initial.Add(time.Minute))
	c.now = clock.Now
	a := newAutomaticScheduler(c, nil, scheduleNotice{service: r.s})
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	var job scheduleJob
	select {
	case job = <-a.jobs:
	case <-testwait.After(t):
		t.Fatal("no notice job")
	}
	done := make(chan string, 1)
	go func() { notice, _ := a.admitNotice(t.Context(), clock, job.state); done <- notice }()
	select {
	case <-ep.reserving:
	case <-testwait.After(t):
		t.Fatal("no broker reservation")
	}
	updated := make(chan error, 1)
	go func() { updated <- c.store.Update(t.Context(), func(*couchschedule.Ledger) error { return nil }) }()
	clock.advance(t, job.state.Active.NoticeDeadline)
	select {
	case notice := <-done:
		if notice != "expired" {
			t.Fatal(notice)
		}
	case <-testwait.After(t):
		t.Fatal("broker admission outlived persisted budget")
	}
	select {
	case err := <-updated:
		if err != nil {
			t.Fatal(err)
		}
	case <-testwait.After(t):
		t.Fatal("deadline did not release scheduler authority")
	}
	if ep.deliveries() != 0 {
		t.Fatal("expired reservation delivered")
	}
}
