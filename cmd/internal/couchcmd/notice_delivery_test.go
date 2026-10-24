package couchcmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// Stateful backend: admission owns an active attempt until released or cancelled;
// receipts and ownership evolve independently of producers.
type noticeDeliveryWorld struct {
	mu                sync.Mutex
	owner             couchcore.NoticeDeliveryOwner
	attempts          []couchcore.NoticeDeliveryEffect
	receipts          map[string]bool
	active, maxActive int
	entered           chan couchcore.NoticeDeliveryEffect
	release           chan struct{}
	result            couchcore.NoticeDeliveryResult
}

func newNoticeDeliveryWorld() *noticeDeliveryWorld {
	return &noticeDeliveryWorld{owner: couchcore.NoticeDeliveryOwner{State: couchcore.NoticeOwnerEnabled, Root: "/hub", Binding: "hub", Epoch: 1}, receipts: map[string]bool{}, entered: make(chan couchcore.NoticeDeliveryEffect, 4), release: make(chan struct{}), result: couchcore.NoticeDeliveryUncertain}
}
func (w *noticeDeliveryWorld) NoticeOwner(couchcore.NoticeDeliveryOccurrence) couchcore.NoticeDeliveryOwner {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.owner
}
func (w *noticeDeliveryWorld) NoticeOwnerCurrent(owner couchcore.NoticeDeliveryOwner) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return owner.Root == "" || owner.Root == w.owner.Root && owner.Epoch == w.owner.Epoch
}
func (w *noticeDeliveryWorld) NoticeReceipt(id string) (bool, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.receipts[id]
	return ok, v
}
func (w *noticeDeliveryWorld) DeliverNotice(ctx context.Context, e couchcore.NoticeDeliveryEffect) couchcore.NoticeDeliveryResult {
	w.mu.Lock()
	w.attempts = append(w.attempts, e)
	w.active++
	if w.active > w.maxActive {
		w.maxActive = w.active
	}
	w.mu.Unlock()
	select {
	case w.entered <- e:
	case <-ctx.Done():
	}
	select {
	case <-w.release:
	case <-ctx.Done():
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.active--
	return w.result
}
func noticeDeliveryEventually(t *testing.T, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("delivery state did not converge")
}
func noticeDeliveryRetained(d *noticeDelivery) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.state.Entries) + len(d.projections)
}

func TestNoticeDeliveryAdapterBootBindAndUncertain(t *testing.T) {
	d := newNoticeDelivery(testConsole())
	defer d.Close()
	w := newNoticeDeliveryWorld()
	d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: "worker exited"})
	if got := noticeDeliveryRetained(d); got != 1 {
		t.Fatalf("startup retained=%d", got)
	}
	d.Bind(w)
	var admitted couchcore.NoticeDeliveryEffect
	select {
	case admitted = <-w.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("boot event not delivered")
	}
	close(w.release)
	noticeDeliveryEventually(t, func() bool {
		d.mu.Lock()
		defer d.mu.Unlock()
		return len(d.state.Entries) == 1 && d.state.Entries[0].Phase == couchcore.NoticeDeliveryAwaitingReceipt
	})
	// Several ownership/receipt polls cannot resubmit uncertain admission.
	for i := 0; i < 3; i++ {
		d.reconcile()
	}
	w.mu.Lock()
	if len(w.attempts) != 1 {
		t.Fatal("uncertain admission resent")
	}
	w.receipts[admitted.AdmissionID] = true
	w.mu.Unlock()
	noticeDeliveryEventually(t, func() bool { return noticeDeliveryRetained(d) == 0 })
}

func TestNoticeDeliveryAdapterBoundsAndClose(t *testing.T) {
	for _, n := range []int{256, 512} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			d := newNoticeDelivery(testConsole())
			w := newNoticeDeliveryWorld()
			d.Bind(w)
			for i := 0; i < n; i++ {
				d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: strings.Repeat("x", 6000), Label: fmt.Sprint(i)})
				d.mu.Lock()
				if len(d.state.Entries)+len(d.projections) > 128 {
					t.Fatal("mailbox exceeds budget")
				}
				for _, e := range d.state.Entries {
					if len(e.Occurrence.Body) > 4096 {
						t.Fatal("body exceeds budget")
					}
				}
				d.mu.Unlock()
			}
			for i := 0; i < 4; i++ {
				select {
				case <-w.entered:
				case <-time.After(3 * time.Second):
					t.Fatal("four workers not started")
				}
			}
			d.Close()
			w.mu.Lock()
			defer w.mu.Unlock()
			if w.active != 0 || w.maxActive > 4 {
				t.Fatalf("active=%d max=%d", w.active, w.maxActive)
			}
			if got := noticeDeliveryRetained(d); got != 0 {
				t.Fatalf("shutdown retains %d", got)
			}
			d.Publish(couchtty.Notice{Class: couchcore.NoticeHubRequested, Body: "late"})
			if noticeDeliveryRetained(d) != 0 {
				t.Fatal("post-close retained")
			}
		})
	}
}

func TestNoticeDeliveryAdapterDisabledDoesNotReplay(t *testing.T) {
	d := newNoticeDelivery(testConsole())
	defer d.Close()
	w := newNoticeDeliveryWorld()
	w.owner.State = couchcore.NoticeOwnerDisabled
	d.Bind(w)
	d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: "disabled"})
	noticeDeliveryEventually(t, func() bool { return noticeDeliveryRetained(d) == 0 })
	w.mu.Lock()
	w.owner.State = couchcore.NoticeOwnerEnabled
	w.mu.Unlock()
	for i := 0; i < 3; i++ {
		d.reconcile()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.attempts) != 0 {
		t.Fatal("automation off/on replayed")
	}
}

func TestNoticeDeliveryAdapterClearFencesQueuedAndCopiesSource(t *testing.T) {
	d := newNoticeDelivery(testConsole())
	defer d.Close()
	slot := couchcore.HubSlot{}
	d.Publish(couchtty.Notice{Class: couchcore.NoticeLivePicker, Condition: "picker:exact-thread", SourceSlot: &slot, Body: "pick one"})
	d.mu.Lock()
	if len(d.state.Entries) != 1 || d.state.Entries[0].Occurrence.SourceSlot == &slot {
		t.Fatal("source slot not copied")
	}
	d.mu.Unlock()
	d.Publish(couchtty.Notice{Class: couchcore.NoticeLivePicker, Condition: "picker:exact-thread", Cleared: true})
	w := newNoticeDeliveryWorld()
	d.Bind(w)
	d.reconcile()
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.attempts) != 0 {
		t.Fatal("cleared boot condition admitted")
	}
	if noticeDeliveryRetained(d) != 0 {
		t.Fatal("cleared condition retained")
	}
}

func TestNoticeDeliveryAdapterBellDoesNotWaitForAdmissions(t *testing.T) {
	d := newNoticeDelivery(testConsole())
	defer d.Close()
	w := newNoticeDeliveryWorld()
	d.Bind(w)
	for i := 0; i < 4; i++ {
		d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: "worker ended"})
	}
	for i := 0; i < 4; i++ {
		select {
		case <-w.entered:
		case <-time.After(3 * time.Second):
			t.Fatal("admission workers did not fill")
		}
	}
	d.Publish(couchtty.Notice{Class: couchcore.NoticeHubRequested, Body: "operator attention"})
	noticeDeliveryEventually(t, func() bool { d.mu.Lock(); defer d.mu.Unlock(); return len(d.projections) == 0 })
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.active != 4 {
		t.Fatalf("bell waited on admissions: active=%d", w.active)
	}
}

func TestNoticeDeliveryAdapterRoutineDoesNotConsumeCapacity(t *testing.T) {
	d := newNoticeDelivery(nil)
	defer d.Close()
	for i := 0; i < 128; i++ {
		d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: "pending"})
	}
	d.Publish(couchtty.Notice{Class: couchcore.NoticeRoutine, Body: "ordinary feedback"})
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state.OverflowCount != 0 {
		t.Fatal("routine feedback counted as delivery loss")
	}
}

func TestNoticeDeliveryAdapterConsoleStopCancelsAdmissions(t *testing.T) {
	console := testConsole()
	d := newNoticeDelivery(console)
	defer d.Close()
	w := newNoticeDeliveryWorld()
	d.Bind(w)
	d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: "worker ended"})
	select {
	case <-w.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("admission did not begin")
	}
	console.Stop()
	select {
	case <-d.ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Console.Stop did not cancel delivery lifetime")
	}
	noticeDeliveryEventually(t, func() bool { w.mu.Lock(); defer w.mu.Unlock(); return w.active == 0 })
}

func TestNoticeDeliveryAdapterFullQueueCoalescesCondition(t *testing.T) {
	d := newNoticeDelivery(nil)
	defer d.Close()
	d.Publish(couchtty.Notice{Class: couchcore.NoticeLivePicker, Condition: "picker:exact-thread", Body: "pick one"})
	for i := 1; i < 128; i++ {
		d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: "pending"})
	}
	d.Publish(couchtty.Notice{Class: couchcore.NoticeLivePicker, Condition: "picker:exact-thread", Body: "still pick one"})
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state.OverflowCount != 0 {
		t.Fatal("coalesced condition counted as delivery loss")
	}
}

// Pause the dispatcher by driving its production memory transitions directly;
// no goroutine scheduling can race the publication/automation ordering here.
func pausedNoticeDelivery() *noticeDelivery {
	return &noticeDelivery{ctx: context.Background(), prefix: "paused", wake: make(chan struct{}, 1), work: make(chan struct{}, 4)}
}

func TestNoticeDeliveryAdapterCapturesDisabledAtPublication(t *testing.T) {
	d := pausedNoticeDelivery()
	w := newNoticeDeliveryWorld()
	w.owner.State = couchcore.NoticeOwnerDisabled
	d.Bind(w)
	d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: "disabled occurrence"})
	w.mu.Lock()
	w.owner.State = couchcore.NoticeOwnerEnabled
	w.owner.Epoch++
	w.mu.Unlock()
	d.reconcile()
	if len(d.state.Entries) != 0 || len(d.admissions) != 0 {
		t.Fatalf("automation off/on revived occurrence: entries=%d attempts=%d", len(d.state.Entries), len(d.admissions))
	}
}

func TestNoticeDeliveryAdapterPendingEpochCannotRevive(t *testing.T) {
	d := pausedNoticeDelivery()
	w := newNoticeDeliveryWorld()
	d.Bind(w)
	for i := 0; i < 5; i++ {
		d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Body: fmt.Sprint(i)})
	}
	d.reconcile()
	if len(d.admissions) != 4 || len(d.state.Entries) != 5 {
		t.Fatal("fixture did not fill four admission slots")
	}
	w.mu.Lock()
	w.owner.Epoch++
	w.mu.Unlock()
	d.reconcile()
	if len(d.state.Entries) != 4 {
		t.Fatal("pending occurrence survived automation epoch change")
	}
	for _, effect := range d.admissions {
		if effect.Owner.Epoch != 1 {
			t.Fatal("admission adopted later automation epoch")
		}
	}
}

// The real service still resolves ownership and guards ingress; only admission
// ordering is controlled so disconnects happen at exact effect boundaries.
type noticeGuardedAdmission struct {
	service *messageService
	entered chan couchcore.NoticeDeliveryEffect
	release chan struct{}
}

func (g *noticeGuardedAdmission) NoticeOwner(o couchcore.NoticeDeliveryOccurrence) couchcore.NoticeDeliveryOwner {
	return g.service.NoticeOwner(o)
}
func (g *noticeGuardedAdmission) NoticeOwnerCurrent(owner couchcore.NoticeDeliveryOwner) bool {
	return g.service.NoticeOwnerCurrent(owner)
}
func (g *noticeGuardedAdmission) NoticeReceipt(id string) (bool, bool) {
	return g.service.NoticeReceipt(id)
}
func (g *noticeGuardedAdmission) DeliverNotice(ctx context.Context, e couchcore.NoticeDeliveryEffect) couchcore.NoticeDeliveryResult {
	select {
	case g.entered <- e:
	case <-ctx.Done():
		return couchcore.NoticeDeliveryRefused
	}
	select {
	case <-g.release:
	case <-ctx.Done():
		return couchcore.NoticeDeliveryRefused
	}
	return g.service.DeliverNotice(ctx, e)
}
func noticeDeliveryWithoutOwnerLoop(t *testing.T, service *messageService) (*noticeDelivery, *noticeGuardedAdmission) {
	t.Helper()
	d := pausedNoticeDelivery()
	d.ctx, d.cancel = context.WithCancel(t.Context())
	g := &noticeGuardedAdmission{service: service, entered: make(chan couchcore.NoticeDeliveryEffect, 4), release: make(chan struct{})}
	d.Bind(g)
	d.wg.Add(4)
	for i := 0; i < 4; i++ {
		go d.worker()
	}
	t.Cleanup(d.Close)
	return d, g
}
func noticeEnterAdmission(t *testing.T, g *noticeGuardedAdmission) couchcore.NoticeDeliveryEffect {
	t.Helper()
	select {
	case e := <-g.entered:
		return e
	case <-time.After(time.Second):
		t.Fatal("admission did not reach guard")
		return couchcore.NoticeDeliveryEffect{}
	}
}
func noticeNonzeroEpoch(t *testing.T, service *messageService) {
	t.Helper()
	for _, action := range []string{"off", "on"} {
		if _, err := service.controlAutomation("/repo1", action); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNoticeDeliveryOwnerLossPendingFallsBack(t *testing.T) {
	for _, revoked := range []bool{false, true} {
		t.Run(fmt.Sprintf("automation-revoked-%t", revoked), func(t *testing.T) {
			r, slots := hubMessageRig(t)
			noticeNonzeroEpoch(t, r.s)
			d, g := noticeDeliveryWithoutOwnerLoop(t, r.s)
			for i := 0; i < 4; i++ {
				d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(slots[2]), Body: fmt.Sprintf("held-%d", i)})
				noticeEnterAdmission(t, g)
			}
			d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(slots[2]), Body: "pending-owner-loss"})
			if revoked {
				noticeNonzeroEpoch(t, r.s)
			}
			r.s.broker.Disconnect(slots[1])
			d.reconcile()
			d.mu.Lock()
			defer d.mu.Unlock()
			want := 1
			if revoked {
				want = 0
			}
			if len(d.projections) != want || want > 0 && d.projections[0].Occurrence.Body != "pending-owner-loss" {
				t.Fatalf("owner-loss fallback=%+v entries=%d wantBell=%d", d.projections, len(d.state.Entries), want)
			}
			if len(d.state.Entries) != 4 {
				t.Fatal("pending owner-loss did not retire exactly once")
			}
		})
	}
}

func TestNoticeDeliveryOwnerLossAtGuardFallsBack(t *testing.T) {
	for _, revoked := range []bool{false, true} {
		t.Run(fmt.Sprintf("automation-revoked-%t", revoked), func(t *testing.T) {
			r, slots := hubMessageRig(t)
			noticeNonzeroEpoch(t, r.s)
			d, g := noticeDeliveryWithoutOwnerLoop(t, r.s)
			d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(slots[2]), Body: "guard-owner-loss"})
			effect := noticeEnterAdmission(t, g)
			if revoked {
				noticeNonzeroEpoch(t, r.s)
			}
			r.s.broker.Disconnect(slots[1])
			close(g.release)
			noticeDeliveryEventually(t, func() bool { d.mu.Lock(); defer d.mu.Unlock(); return len(d.state.Entries) == 0 })
			d.mu.Lock()
			defer d.mu.Unlock()
			want := 1
			if revoked {
				want = 0
			}
			if len(d.projections) != want {
				t.Fatalf("confirmed refusal projections=%d want=%d", len(d.projections), want)
			}
			if _, known, _ := r.s.broker.ReceiptSnapshot(effect.AdmissionID); known {
				t.Fatal("disconnected guard admitted a broker message")
			}
		})
	}
}

func TestNoticeDeliveryOwnerLossWithoutSourceMetadata(t *testing.T) {
	for _, pending := range []bool{false, true} {
		for _, revoked := range []bool{false, true} {
			t.Run(fmt.Sprintf("pending-%t-revoked-%t", pending, revoked), func(t *testing.T) {
				r, slots := hubMessageRig(t)
				noticeNonzeroEpoch(t, r.s)
				d, g := noticeDeliveryWithoutOwnerLoop(t, r.s)
				if pending {
					for i := 0; i < 4; i++ {
						d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(slots[2]), Body: fmt.Sprintf("held-%d", i)})
						noticeEnterAdmission(t, g)
					}
				}
				// Thread is the only producer provenance: no SourceSlot or OwnerRoot hint.
				d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(slots[2]), Body: "removed-source-owner-loss"})
				if !pending {
					noticeEnterAdmission(t, g)
				}
				if revoked {
					noticeNonzeroEpoch(t, r.s)
				}
				r.disconnect(slots[2])
				r.disconnect(slots[1])
				noticeDeliveryEventually(t, func() bool {
					r.s.mu.Lock()
					defer r.s.mu.Unlock()
					_, source := r.s.workspaces[slots[2]]
					_, owner := r.s.workspaces[slots[1]]
					return !source && !owner
				})
				if pending {
					d.reconcile()
				} else {
					close(g.release)
					noticeDeliveryEventually(t, func() bool { d.mu.Lock(); defer d.mu.Unlock(); return len(d.state.Entries) == 0 })
				}
				d.mu.Lock()
				defer d.mu.Unlock()
				want := 1
				if revoked {
					want = 0
				}
				if len(d.projections) != want {
					t.Fatalf("exact removed-owner fallback=%d want=%d", len(d.projections), want)
				}
			})
		}
	}
}

func TestNoticeDeliveryOwnerLossAfterBootResolution(t *testing.T) {
	r, slots := hubMessageRig(t)
	noticeNonzeroEpoch(t, r.s)
	d, g := noticeDeliveryWithoutOwnerLoop(t, r.s)
	d.mu.Lock()
	d.backend = nil
	d.mu.Unlock()
	d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(slots[2]), Body: "buffered-before-bind"})
	d.Bind(g)
	for i := 0; i < 4; i++ {
		d.Publish(couchtty.Notice{Class: couchcore.NoticeChildExited, Thread: bindingThread(slots[2]), Body: fmt.Sprintf("held-%d", i)})
		noticeEnterAdmission(t, g)
	}
	d.reconcile() // Booting episode now resolves, but all four admissions are held.
	r.disconnect(slots[2])
	r.disconnect(slots[1])
	noticeDeliveryEventually(t, func() bool {
		r.s.mu.Lock()
		defer r.s.mu.Unlock()
		_, source := r.s.workspaces[slots[2]]
		_, owner := r.s.workspaces[slots[1]]
		return !source && !owner
	})
	d.reconcile()
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.projections) != 1 || d.projections[0].Occurrence.Body != "buffered-before-bind" {
		t.Fatal("boot resolution failed to retain exact owner provenance")
	}
}

func TestNoticeDeliveryUnknownReadinessPreservesPending(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, action := range []string{"off", "on"} {
		if _, err := r.s.controlAutomation("/repo0", action); err != nil {
			t.Fatal(err)
		}
	}
	d := pausedNoticeDelivery()
	d.Bind(r.s)
	for i := 0; i < 5; i++ {
		d.Publish(couchtty.Notice{Class: couchcore.NoticeDispatcherDiagnostic, Body: "global diagnostic"})
	}
	if len(d.state.Entries) != 5 || len(d.admissions) != 4 {
		t.Fatal("fixture did not occupy four admissions")
	}
	// This is exactly the registry publication-before-EffectConnect cache window:
	// connected has an unrelated binding which has not committed its identity.
	r.s.mu.Lock()
	unrelated := r.s.workspaces[slots[3]]
	delete(r.s.workspaces, slots[3])
	r.s.mu.Unlock()
	r.s.broker.Disconnect(slots[0])
	owner := r.s.NoticeOwner(d.state.Entries[4].Occurrence)
	if owner.State != couchcore.NoticeOwnerBooting {
		t.Fatalf("fixture owner=%+v", owner)
	}
	d.reconcile()
	if len(d.state.Entries) != 5 {
		t.Errorf("unknown readiness cleared pending occurrence: owner=%+v retained=%d", owner, len(d.state.Entries))
	}
	r.s.mu.Lock()
	r.s.workspaces[slots[3]] = unrelated
	r.s.mu.Unlock()
	d.reconcile()
	if len(d.projections) != 1 {
		t.Errorf("completed identity witness lost fallback: bells=%d", len(d.projections))
	}
}

func TestNoticeDeliveryUnknownReadinessDoesNotEraseEpoch(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, action := range []string{"off", "on"} {
		if _, err := r.s.controlAutomation("/repo0", action); err != nil {
			t.Fatal(err)
		}
	}
	d := pausedNoticeDelivery()
	d.Bind(r.s)
	for i := 0; i < 5; i++ {
		d.Publish(couchtty.Notice{Class: couchcore.NoticeDispatcherDiagnostic, Body: "global diagnostic"})
	}
	r.s.mu.Lock()
	unrelated := r.s.workspaces[slots[3]]
	delete(r.s.workspaces, slots[3])
	r.s.mu.Unlock()
	r.s.broker.Disconnect(slots[0])
	d.reconcile()
	if len(d.state.Entries) != 5 {
		t.Fatalf("unknown readiness lost notice before revocation: entries=%d", len(d.state.Entries))
	}
	for _, action := range []string{"off", "on"} {
		if _, err := r.s.controlAutomation("/repo0", action); err != nil {
			t.Fatal(err)
		}
	}
	d.reconcile()
	r.s.mu.Lock()
	r.s.workspaces[slots[3]] = unrelated
	r.s.mu.Unlock()
	d.reconcile()
	if len(d.projections) != 0 {
		t.Fatalf("unknown-readiness transition erased original epoch: bells=%d", len(d.projections))
	}
}

func TestNoticeDeliveryUnknownReadinessAfterGuardRefusal(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, action := range []string{"off", "on"} {
		if _, err := r.s.controlAutomation("/repo0", action); err != nil {
			t.Fatal(err)
		}
	}
	d, g := noticeDeliveryWithoutOwnerLoop(t, r.s)
	d.Publish(couchtty.Notice{Class: couchcore.NoticeDispatcherDiagnostic, Body: "global guard diagnostic"})
	noticeEnterAdmission(t, g)
	r.s.mu.Lock()
	unrelated := r.s.workspaces[slots[3]]
	delete(r.s.workspaces, slots[3])
	r.s.mu.Unlock()
	r.s.broker.Disconnect(slots[0])
	close(g.release)
	noticeDeliveryEventually(t, func() bool {
		d.mu.Lock()
		defer d.mu.Unlock()
		return len(d.state.Entries) == 1 && d.state.Entries[0].Phase == couchcore.NoticeDeliveryAwaitingDisposition
	})
	r.s.mu.Lock()
	r.s.workspaces[slots[3]] = unrelated
	r.s.mu.Unlock()
	d.reconcile()
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.projections) != 1 {
		t.Fatalf("unknown-readiness refusal lost later unavailable fallback: bells=%d", len(d.projections))
	}
}
