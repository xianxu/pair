package couchcmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// noticeDelivery owns only bounded ephemeral routing state. Publish is called
// with Console locked: it must never call back into Console or perform backend
// IO. Only the backend's cached, callback-free NoticeOwner snapshot is allowed.
type noticeDelivery struct {
	mu                sync.Mutex
	projectionMu      sync.Mutex
	console           *couchtty.Console
	backend           couchtty.NoticeDeliveryBackend
	state             couchcore.NoticeDeliveryState
	admissions        []couchcore.NoticeDeliveryEffect
	projections       []couchcore.NoticeDeliveryEffect
	overflowDelivered uint64
	prefix            string
	sequence          uint64
	ctx               context.Context
	cancel            context.CancelFunc
	wake              chan struct{}
	work              chan struct{}
	wg                sync.WaitGroup
	closeOnce         sync.Once
}

func newNoticeDelivery(console *couchtty.Console) *noticeDelivery {
	prefix, err := newRequestID()
	if err != nil {
		prefix = fmt.Sprintf("notice-%x", time.Now().UnixNano())
	}
	lifetime := context.Background()
	if console != nil {
		lifetime = console.OperationLifetime()
	}
	ctx, cancel := context.WithCancel(lifetime)
	d := &noticeDelivery{console: console, prefix: prefix, ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1), work: make(chan struct{}, 4)}
	d.wg.Add(5)
	go d.run()
	for i := 0; i < 4; i++ {
		go d.worker()
	}
	return d
}

func (d *noticeDelivery) signal() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}
func (d *noticeDelivery) signalWorkers() {
	for i := 0; i < 4; i++ {
		select {
		case d.work <- struct{}{}:
		default:
		}
	}
}

func (d *noticeDelivery) Publish(n couchtty.Notice) {
	d.mu.Lock()
	if d.state.Closed || d.ctx.Err() != nil {
		d.mu.Unlock()
		return
	}
	if n.Cleared {
		d.advanceLocked(couchcore.NoticeDeliveryEvent{Kind: couchcore.NoticeDeliveryConditionCleared, Now: time.Now(), Occurrence: couchcore.NoticeDeliveryOccurrence{Condition: n.Condition}})
		d.mu.Unlock()
		d.signal()
		return
	}
	if n.Condition != "" {
		for _, entry := range d.state.Entries {
			if entry.Occurrence.Condition == n.Condition && entry.Occurrence.Class == n.Class && entry.Occurrence.Source == string(n.Actor) {
				d.mu.Unlock()
				return
			}
		}
	}
	if n.Class == couchcore.NoticeRoutine {
		d.mu.Unlock()
		return
	}
	d.sequence++
	body := n.Body
	if len(body) > couchcore.NoticeDeliveryMaxBodyBytes {
		end := couchcore.NoticeDeliveryMaxBodyBytes
		for end > 0 && !utf8.RuneStart(body[end]) {
			end--
		}
		body = body[:end]
	}
	occurrence := couchcore.NoticeDeliveryOccurrence{ID: d.prefix + "-" + strconv.FormatUint(d.sequence, 10), Condition: n.Condition, Class: n.Class, Origin: n.Origin, OwnerRoot: n.OwnerRoot, SourceRoot: n.SourceRoot, Thread: n.Thread, Label: n.Label, Body: strings.Clone(body), HubHandled: n.HubHandled, Source: string(n.Actor)}
	if n.SourceSlot != nil {
		slot := *n.SourceSlot
		occurrence.SourceSlot = &slot
	}
	// Lock order: Console.mu -> delivery.mu -> sequential backend memory
	// locks (service, hub identity, broker snapshot, automation). NoticeOwner
	// MUST do no IO and invoke no callbacks/Console methods. DeliverNotice is
	// called only after releasing delivery.mu. This captures automation OFF and
	// its generation at the occurrence, before the asynchronous dispatcher runs.
	owner := couchcore.NoticeDeliveryOwner{State: couchcore.NoticeOwnerBooting}
	if d.backend != nil {
		owner = d.backend.NoticeOwner(occurrence)
	}
	// Retain verified ownership before source metadata can be removed. A later
	// live source lookup still wins; without it, use this exact prior root.
	if owner.Root != "" {
		occurrence.OwnerRoot = owner.Root
	}
	route := couchcore.RouteNotice(couchcore.NoticeRouteInput{Class: occurrence.Class, Origin: occurrence.Origin, Owner: owner.State, HubHandled: occurrence.HubHandled})
	if !route.Hub && !route.Bell && !route.Defer {
		d.mu.Unlock()
		return
	}
	if len(d.state.Entries)+len(d.projections) >= couchcore.NoticeDeliveryMaxPending {
		if d.state.OverflowCount < ^uint64(0) {
			d.state.OverflowCount++
		}
		d.mu.Unlock()
		d.signal()
		return
	}
	d.advanceLocked(couchcore.NoticeDeliveryEvent{Kind: couchcore.NoticeDeliveryOccurrenceAdded, Now: time.Now(), Occurrence: occurrence, Owner: owner})
	d.mu.Unlock()
	d.signal()
}

func (d *noticeDelivery) Bind(backend couchtty.NoticeDeliveryBackend) {
	d.mu.Lock()
	if !d.state.Closed {
		d.backend = backend
	}
	d.mu.Unlock()
	d.signal()
}

func (d *noticeDelivery) advanceLocked(event couchcore.NoticeDeliveryEvent) {
	var effects []couchcore.NoticeDeliveryEffect
	d.state, effects = couchcore.AdvanceNoticeDelivery(d.state, event)
	for _, effect := range effects {
		switch effect.Kind {
		case couchcore.NoticeDeliveryAdmit:
			d.admissions = append(d.admissions, effect)
		case couchcore.NoticeDeliveryBell:
			if effect.Occurrence.Class != couchcore.NoticeLivePicker && effect.Occurrence.Class != couchcore.NoticeJudgedQuestion {
				d.projections = append(d.projections, effect)
			}
			// Overflow count is already retained in state; it needs no queued payload.
		}
	}
	// Remove effects made stale by expiry/clear before reserving another slot.
	keep := d.admissions[:0]
	for _, effect := range d.admissions {
		if d.validLocked(effect) {
			keep = append(keep, effect)
		}
	}
	d.admissions = keep
	d.signalWorkers()
}

func (d *noticeDelivery) validLocked(effect couchcore.NoticeDeliveryEffect) bool {
	if d.state.Closed {
		return false
	}
	for _, entry := range d.state.Entries {
		if entry.Occurrence.ID == effect.Occurrence.ID && entry.Token == effect.Token && entry.Owner == effect.Owner && entry.Phase == couchcore.NoticeDeliverySending {
			return true
		}
	}
	return false
}

func (d *noticeDelivery) run() {
	defer d.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.wake:
		case <-ticker.C:
		}
		if d.ctx.Err() != nil {
			return
		}
		d.reconcile()
		// A bounded projection batch cannot be held behind slow broker admissions.
		for i := 0; i < couchcore.NoticeDeliveryMaxPending+1; i++ {
			if !d.projectOne() {
				break
			}
		}
	}
}

func (d *noticeDelivery) reconcile() {
	d.mu.Lock()
	if d.state.Closed {
		d.mu.Unlock()
		return
	}
	d.advanceLocked(couchcore.NoticeDeliveryEvent{Kind: couchcore.NoticeDeliveryTick, Now: time.Now()})
	backend := d.backend
	entries := append([]couchcore.NoticeDeliveryEntry(nil), d.state.Entries...)
	d.mu.Unlock()
	if backend == nil {
		return
	}
	for _, entry := range entries {
		if d.ctx.Err() != nil {
			return
		}
		switch entry.Phase {
		case couchcore.NoticeDeliveryPending, couchcore.NoticeDeliveryAwaitingDisposition:
			owner := backend.NoticeOwner(entry.Occurrence)
			d.mu.Lock()
			if !d.state.Closed {
				// Match the exact prior fact before using its scoped generation witness.
				// A concurrent completion or accepted handoff supersedes this observation.
				for _, current := range d.state.Entries {
					if current.Occurrence.ID == entry.Occurrence.ID && current.Owner == entry.Owner && current.Token == entry.Token && current.Phase == entry.Phase {
						revoked := !backend.NoticeOwnerCurrent(entry.Owner)
						d.advanceLocked(couchcore.NoticeDeliveryEvent{Kind: couchcore.NoticeDeliveryOwnerChanged, Now: time.Now(), ID: entry.Occurrence.ID, Token: entry.Token, Owner: owner, Revoked: revoked})
						break
					}
				}
			}
			d.mu.Unlock()
		case couchcore.NoticeDeliveryAwaitingReceipt:
			known, terminal := backend.NoticeReceipt(entry.AdmissionID)
			if known && terminal {
				d.mu.Lock()
				if !d.state.Closed {
					d.advanceLocked(couchcore.NoticeDeliveryEvent{Kind: couchcore.NoticeDeliveryReceiptObserved, Now: time.Now(), ID: entry.Occurrence.ID, Token: entry.Token})
				}
				d.mu.Unlock()
			}
		}
	}
}

func (d *noticeDelivery) projectOne() bool {
	// Close takes this lock before cancelling, so no Console projection can begin
	// after cancellation. Never hold d.mu while entering Console.
	d.projectionMu.Lock()
	defer d.projectionMu.Unlock()
	d.mu.Lock()
	if d.state.Closed || d.ctx.Err() != nil {
		d.mu.Unlock()
		return false
	}
	var effect couchcore.NoticeDeliveryEffect
	overflow := false
	updateOnly := false
	if len(d.projections) > 0 {
		effect = d.projections[0]
	} else if d.state.OverflowCount > d.overflowDelivered {
		overflow = true
		effect.Count = d.state.OverflowCount
		updateOnly = d.overflowDelivered > 0
		effect.Occurrence = couchcore.NoticeDeliveryOccurrence{ID: couchcore.NoticeDeliveryOverflowID, Label: "Couch", Body: fmt.Sprintf("Hub diagnostic delivery capacity exceeded (%d omitted)", effect.Count)}
	} else {
		d.mu.Unlock()
		return false
	}
	d.mu.Unlock()
	n := couchtty.Notice{Class: effect.Occurrence.Class, Origin: effect.Occurrence.Origin, Thread: effect.Occurrence.Thread, Label: effect.Occurrence.Label, Body: effect.Occurrence.Body}
	if d.console != nil {
		d.console.ApplyNoticeAttention(n, effect.Occurrence.ID, updateOnly)
	}
	d.mu.Lock()
	if overflow {
		d.overflowDelivered = effect.Count
	} else {
		d.projections[0] = couchcore.NoticeDeliveryEffect{}
		d.projections = d.projections[1:]
	}
	d.mu.Unlock()
	return true
}

func (d *noticeDelivery) worker() {
	defer d.wg.Done()
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.work:
		}
		for {
			d.mu.Lock()
			if d.state.Closed || d.ctx.Err() != nil {
				d.mu.Unlock()
				return
			}
			if len(d.admissions) == 0 || d.backend == nil {
				d.mu.Unlock()
				break
			}
			effect := d.admissions[0]
			d.admissions[0] = couchcore.NoticeDeliveryEffect{}
			d.admissions = d.admissions[1:]
			valid := d.validLocked(effect)
			backend := d.backend
			d.mu.Unlock()
			if !valid {
				continue
			}
			ctx, cancel := context.WithTimeout(d.ctx, 2*time.Second)
			result := backend.DeliverNotice(ctx, effect)
			cancel()
			// Only a confirmed non-admission can acquire a new local disposition.
			// Admitted/uncertain sends remain receipt-owned and are never rerouted.
			var owner couchcore.NoticeDeliveryOwner
			if result == couchcore.NoticeDeliveryRefused && d.ctx.Err() == nil {
				owner = backend.NoticeOwner(effect.Occurrence)
			}
			d.mu.Lock()
			if !d.state.Closed && d.validLocked(effect) {
				revoked := false
				if result == couchcore.NoticeDeliveryRefused {
					revoked = !backend.NoticeOwnerCurrent(effect.Owner)
				}
				d.advanceLocked(couchcore.NoticeDeliveryEvent{Kind: couchcore.NoticeDeliverySendResult, Now: time.Now(), ID: effect.Occurrence.ID, Token: effect.Token, Result: result, Owner: owner, Revoked: revoked})
			}
			d.mu.Unlock()
			d.signal()
		}
	}
}

func (d *noticeDelivery) Close() {
	d.closeOnce.Do(func() {
		d.projectionMu.Lock()
		d.mu.Lock()
		d.state, _ = couchcore.AdvanceNoticeDelivery(d.state, couchcore.NoticeDeliveryEvent{Kind: couchcore.NoticeDeliveryShutdown, Now: time.Now()})
		d.admissions = nil
		d.projections = nil
		d.cancel()
		d.mu.Unlock()
		d.projectionMu.Unlock()
		d.wg.Wait()
	})
}
