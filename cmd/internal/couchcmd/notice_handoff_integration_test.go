package couchcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// Pause before the real guarded ingress, not inside hub ownership's effect gate:
// a protocol move remains able to complete while admissions await their turn.
type noticeHandoffGate struct {
	service           *messageService
	entered           chan couchcore.NoticeDeliveryEffect
	mu                sync.Mutex
	releases          map[string]chan struct{}
	attempts          map[string]int
	active, maxActive int
	unknown           atomic.Bool
}

func (g *noticeHandoffGate) NoticeOwner(o couchcore.NoticeDeliveryOccurrence) couchcore.NoticeDeliveryOwner {
	owner := g.service.NoticeOwner(o)
	if g.unknown.Load() {
		owner.State = couchcore.NoticeOwnerBooting
		owner.Binding = ""
	}
	return owner
}
func (g *noticeHandoffGate) NoticeOwnerCurrent(owner couchcore.NoticeDeliveryOwner) bool {
	return g.service.NoticeOwnerCurrent(owner)
}
func (g *noticeHandoffGate) NoticeReceipt(id string) (bool, bool) { return g.service.NoticeReceipt(id) }
func (g *noticeHandoffGate) DeliverNotice(ctx context.Context, e couchcore.NoticeDeliveryEffect) couchcore.NoticeDeliveryResult {
	g.mu.Lock()
	g.active++
	if g.active > g.maxActive {
		g.maxActive = g.active
	}
	g.attempts[e.Occurrence.Body]++
	release := g.releases[e.Occurrence.Body]
	g.mu.Unlock()
	defer func() { g.mu.Lock(); g.active--; g.mu.Unlock() }()
	select {
	case g.entered <- e:
	case <-ctx.Done():
		return couchcore.NoticeDeliveryRefused
	}
	select {
	case <-release:
	case <-ctx.Done():
		return couchcore.NoticeDeliveryRefused
	}
	return g.service.DeliverNotice(ctx, e)
}
func (g *noticeHandoffGate) release(body string) {
	g.mu.Lock()
	ch := g.releases[body]
	g.mu.Unlock()
	close(ch)
}
func (g *noticeHandoffGate) next(t *testing.T) couchcore.NoticeDeliveryEffect {
	t.Helper()
	select {
	case e := <-g.entered:
		return e
	case <-time.After(time.Second):
		t.Fatal("guarded admission did not begin")
		return couchcore.NoticeDeliveryEffect{}
	}
}
func noticeHandoffFixture(t *testing.T) (*hubMoveRig, *noticeDelivery, *noticeHandoffGate) {
	t.Helper()
	r := newHubMoveRig(t)
	r.disconnect(r.worker)
	number := 7
	r.world.set(r.worker, func(s *worldSlot) {
		s.workspace.PrimaryRoot = filepath.Join(r.defs[0].Workspace, "pair")
		s.workspace.Slot = &number
		s.workspace.FleetRoot = r.defs[0].Workspace
	})
	r.wrapper(r.worker)
	r.waitConnected(r.worker, true)
	r.waitWorkspace(r.worker)
	// Epochs on the original root deliberately differ from the destination's.
	for _, action := range []string{"off", "on"} {
		if _, err := r.s.controlAutomation(r.defs[0].Root, action); err != nil {
			t.Fatal(err)
		}
	}
	original := r.s.NoticeOwner(couchcore.NoticeDeliveryOccurrence{Thread: bindingThread(r.worker)})
	if original.Root != r.defs[0].Root || original.Epoch != 2 || original.State != couchcore.NoticeOwnerEnabled {
		t.Fatalf("fixture original witness: %+v", original)
	}
	r.disconnect(r.callers[1])
	at := time.Now().Add(-time.Hour)
	r.world.mu.Lock()
	r.world.endpoints[r.callers[1]] = hubQuietEndpoint{at: at}
	r.world.mu.Unlock()
	r.wrapper(r.callers[1], couchmessage.Observation{LastActivity: at, Sequence: 2})
	r.waitConnected(r.callers[1], true)
	r.waitWorkspace(r.callers[1])
	d := pausedNoticeDelivery()
	d.ctx, d.cancel = context.WithCancel(t.Context())
	g := &noticeHandoffGate{service: r.s, entered: make(chan couchcore.NoticeDeliveryEffect, 8), releases: map[string]chan struct{}{}, attempts: map[string]int{}}
	for i := 0; i < 4; i++ {
		g.releases[fmt.Sprintf("hold-%d", i)] = make(chan struct{})
	}
	g.releases["fifth"] = make(chan struct{})
	d.Bind(g)
	d.wg.Add(4)
	for i := 0; i < 4; i++ {
		go d.worker()
	}
	t.Cleanup(d.Close)
	return r, d, g
}
func noticeHandoffPublish(d *noticeDelivery, r *hubMoveRig, body string) {
	d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(r.worker), Body: body})
}
func noticeHandoffHoldFour(t *testing.T, r *hubMoveRig, d *noticeDelivery, g *noticeHandoffGate) {
	t.Helper()
	for i := 0; i < 4; i++ {
		noticeHandoffPublish(d, r, fmt.Sprintf("hold-%d", i))
		g.next(t)
	}
	noticeHandoffPublish(d, r, "fifth")
}
func noticeHandoffMove(t *testing.T, r *hubMoveRig) {
	t.Helper()
	response := r.call(r.request(1, "notice-handoff", "pair:7", 1, false))
	if response.Code != "accepted" {
		t.Fatalf("move admission: %+v", response)
	}
	if err := r.finish(0); err != nil {
		t.Fatal(err)
	}
}
func noticeHandoffEntry(d *noticeDelivery, body string) (couchcore.NoticeDeliveryEntry, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, entry := range d.state.Entries {
		if entry.Occurrence.Body == body {
			return entry, true
		}
	}
	return couchcore.NoticeDeliveryEntry{}, false
}
func noticeHandoffBells(d *noticeDelivery, body string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, effect := range d.projections {
		if effect.Occurrence.Body == body {
			n++
		}
	}
	return n
}

func TestNoticeHandoffPendingDisposition(t *testing.T) {
	for _, scenario := range []string{"enabled", "unavailable", "disabled", "revoked-before", "old-off-after", "unknown"} {
		t.Run(scenario, func(t *testing.T) {
			r, d, g := noticeHandoffFixture(t)
			noticeHandoffHoldFour(t, r, d, g)
			if scenario == "revoked-before" {
				for _, action := range []string{"off", "on"} {
					if _, err := r.s.controlAutomation(r.defs[0].Root, action); err != nil {
						t.Fatal(err)
					}
				}
			}
			if scenario == "disabled" {
				if _, err := r.s.controlAutomation(r.defs[1].Root, "off"); err != nil {
					t.Fatal(err)
				}
			}
			noticeHandoffMove(t, r)
			if scenario == "unavailable" {
				r.disconnect(r.callers[1])
			}
			if scenario == "unknown" {
				g.unknown.Store(true)
			}
			before := r.world.counts()
			d.reconcile()
			if got := r.world.counts(); got != before {
				t.Fatalf("reconciliation performed discovery: %v -> %v", before, got)
			}
			entry, present := noticeHandoffEntry(d, "fifth")
			switch scenario {
			case "unavailable":
				if present || noticeHandoffBells(d, "fifth") != 1 {
					t.Fatalf("handoff destination loss lost fallback: present=%v bells=%d", present, noticeHandoffBells(d, "fifth"))
				}
				return
			case "disabled", "revoked-before":
				if present || noticeHandoffBells(d, "fifth") != 0 {
					t.Fatalf("feed-only handoff retained obligation: %+v present=%v", entry, present)
				}
				if scenario == "disabled" {
					if _, err := r.s.controlAutomation(r.defs[1].Root, "on"); err != nil {
						t.Fatal(err)
					}
					d.reconcile()
					if _, exists := noticeHandoffEntry(d, "fifth"); exists {
						t.Fatal("disabled destination replayed occurrence")
					}
				}
				return
			case "unknown":
				if !present || entry.Phase != couchcore.NoticeDeliveryPending || noticeHandoffBells(d, "fifth") != 0 {
					t.Fatalf("unknown readiness erased obligation: %+v present=%v", entry, present)
				}
				g.unknown.Store(false)
				d.reconcile()
				entry, present = noticeHandoffEntry(d, "fifth")
			}
			if !present || entry.Owner.Root != r.defs[1].Root || entry.Owner.Epoch != 0 {
				t.Fatalf("pending handoff compared counters across roots: %+v present=%v", entry, present)
			}
			if scenario == "old-off-after" {
				for _, action := range []string{"off", "on"} {
					if _, err := r.s.controlAutomation(r.defs[0].Root, action); err != nil {
						t.Fatal(err)
					}
				}
				d.reconcile()
				if _, exists := noticeHandoffEntry(d, "fifth"); !exists {
					t.Fatal("former root revoked accepted handoff")
				}
			}
			g.release("hold-0")
			effect := g.next(t)
			if effect.Occurrence.Body != "fifth" || effect.Owner.Root != r.defs[1].Root || effect.Owner.Binding != lifecycleBindingKey(r.callers[1]) {
				t.Fatalf("handoff admission targeted wrong root: %+v", effect)
			}
			g.release("fifth")
			noticeDeliveryEventually(t, func() bool {
				receipt, known, _ := r.s.broker.ReceiptSnapshot(effect.AdmissionID)
				return known && receipt.Message.To == r.callers[1]
			})
			g.mu.Lock()
			defer g.mu.Unlock()
			if g.attempts["fifth"] != 1 || g.maxActive > 4 {
				t.Fatalf("handoff attempt/workers grew: attempts=%v maxActive=%d", g.attempts, g.maxActive)
			}
		})
	}
}

func TestNoticeHandoffConfirmedRefusalNeverResends(t *testing.T) {
	for _, scenario := range []string{"enabled", "unavailable", "unknown"} {
		t.Run(scenario, func(t *testing.T) {
			unknown := scenario == "unknown"
			r, d, g := noticeHandoffFixture(t)
			noticeHandoffPublish(d, r, "fifth")
			effect := g.next(t)
			noticeHandoffMove(t, r)
			g.unknown.Store(unknown)
			if scenario == "unavailable" {
				r.disconnect(r.callers[1])
			}
			g.release("fifth")
			noticeDeliveryEventually(t, func() bool {
				entry, present := noticeHandoffEntry(d, "fifth")
				return !present || entry.Phase != couchcore.NoticeDeliverySending
			})
			if unknown {
				entry, present := noticeHandoffEntry(d, "fifth")
				if !present || entry.Phase != couchcore.NoticeDeliveryAwaitingDisposition {
					t.Fatalf("confirmed refusal lost unknown disposition: %+v present=%v", entry, present)
				}
				d.reconcile()
				g.mu.Lock()
				attempts := g.attempts["fifth"]
				g.mu.Unlock()
				if attempts != 1 {
					t.Fatal("unknown disposition resent")
				}
				r.disconnect(r.callers[1])
				g.unknown.Store(false)
				d.reconcile()
			}
			wantBells := 1
			if scenario == "enabled" {
				wantBells = 0
			}
			if noticeHandoffBells(d, "fifth") != wantBells {
				t.Fatalf("refused handoff disposition: bells=%d want=%d", noticeHandoffBells(d, "fifth"), wantBells)
			}
			if _, known, _ := r.s.broker.ReceiptSnapshot(effect.AdmissionID); known {
				t.Fatal("confirmed guard refusal admitted receipt")
			}
			for i := 0; i < 3; i++ {
				d.reconcile()
			}
			g.mu.Lock()
			defer g.mu.Unlock()
			if g.attempts["fifth"] != 1 {
				t.Fatalf("confirmed refusal resent: %v", g.attempts)
			}
		})
	}
}

type noticeHandoffCountedBackend struct {
	owner                    couchcore.NoticeDeliveryOwner
	observations, admissions int
}

func (b *noticeHandoffCountedBackend) NoticeOwner(couchcore.NoticeDeliveryOccurrence) couchcore.NoticeDeliveryOwner {
	b.observations++
	return b.owner
}
func (b *noticeHandoffCountedBackend) NoticeOwnerCurrent(couchcore.NoticeDeliveryOwner) bool {
	b.observations++
	return true
}
func (b *noticeHandoffCountedBackend) NoticeReceipt(string) (bool, bool) {
	b.observations++
	return false, false
}
func (b *noticeHandoffCountedBackend) DeliverNotice(context.Context, couchcore.NoticeDeliveryEffect) couchcore.NoticeDeliveryResult {
	b.admissions++
	return couchcore.NoticeDeliveryRefused
}
func TestNoticeHandoffReconciliationObservationBound(t *testing.T) {
	for _, n := range []int{16, 32} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			d := pausedNoticeDelivery()
			b := &noticeHandoffCountedBackend{owner: couchcore.NoticeDeliveryOwner{State: couchcore.NoticeOwnerBooting, Root: "/origin", Epoch: 2}}
			d.Bind(b)
			for i := 0; i < n; i++ {
				d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: fmt.Sprint(i)})
			}
			for history := 0; history < 2*n; history++ {
				b.observations = 0
				d.reconcile()
				if b.observations < n || b.observations > 2*n || b.admissions != 0 || len(d.state.Entries) != n || cap(d.work) != 4 {
					t.Fatalf("reconciliation exceeded bound: n=%d observations=%d admissions=%d retained=%d workerSlots=%d", n, b.observations, b.admissions, len(d.state.Entries), cap(d.work))
				}
			}
		})
	}
}
