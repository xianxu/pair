package couchcmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

const startupResendPrompt = "Couch restarted and the hub is available. If your last report to the hub is still unanswered and relevant, resend it now. Otherwise do nothing."

type supervisionSend struct {
	from     couchmessage.Binding
	at       time.Time
	system   bool
	prompt   string
	admitted bool
	// Retain ownership at initiation through queueing and delivery. The snapshot
	// is immutable and shares the pending send lifetime, not receipt history.
	ownership *couchcore.HubOwnership
	hubRoot   string
}

// supervisionCoordinator adapts live evidence to the pure rules. Never hold mu
// across broker calls: endpoint delivery may synchronously call back here.
type supervisionCoordinator struct {
	mu      sync.Mutex
	state   *supervisionState
	sends   map[string]supervisionSend
	service *messageService
	now     func() time.Time
	notice  func(string)
}

func newSupervisionCoordinator(s *messageService, notice func(string)) *supervisionCoordinator {
	return &supervisionCoordinator{state: newSupervisionState(), sends: map[string]supervisionSend{}, service: s, now: time.Now, notice: notice}
}
func (c *supervisionCoordinator) remember(id string, from couchmessage.Binding, now time.Time, system bool) bool {
	if _, exists := c.sends[id]; exists {
		return false
	}
	c.sends[id] = supervisionSend{from: from, at: now, system: system}
	return true
}

// sending is called only for authenticated CLI sends, before resolving hub so
// a live supervisor can retain the fact of a refused hub report (not its body).
func (c *supervisionCoordinator) sending(from couchmessage.Binding, r couchmessage.Request) bool {
	hub, hasHub := c.service.hubBindingFor(from)
	toHub := r.Target == couchmessage.HubTarget || (hasHub && r.Target == hub.Slot)
	if !toHub && (!hasHub || from != hub) {
		return false
	}
	submission, connected := c.service.broker.Submission(from)
	if !connected {
		return false
	}
	// Existing accepted requests are idempotent. Failed attempts retain only the
	// worker's latest report, not a second receipt history with a competing cap.
	if _, err := c.service.broker.Status(from, r.ID); err == nil {
		return false
	}
	now := c.now()
	var ownership *couchcore.HubOwnership
	var hubRoot string
	if hasHub && from == hub && !toHub && c.service.authority.hubs != nil {
		snapshot := c.service.authority.hubs.snapshot()
		ownership = &snapshot
		c.service.mu.Lock()
		hubRoot = c.service.workspaces[from].WorktreeRoot
		c.service.mu.Unlock()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if toHub && (!hasHub || from != hub) {
		c.state.report(from, r.ID, now, submission, completionHint(r.Body))
		return false
	}
	if hasHub && from == hub && !toHub {
		if !c.remember(r.ID, from, now, false) {
			return false
		}
		v := c.sends[r.ID]
		v.ownership, v.hubRoot = ownership, hubRoot
		c.sends[r.ID] = v
		return true
	}
	return false
}

// admitted reconciles only a registration owned by this Send invocation.
func (c *supervisionCoordinator) admitted(id string, receipt *couchmessage.Receipt) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if receipt == nil {
		delete(c.sends, id)
		return
	}
	if v, ok := c.sends[id]; ok {
		v.admitted = true
		c.sends[id] = v
	}
}

// sendGenerated shares normal Broker.Send policy, including its busy queue.
// Metadata exists only while admission/delivery is outstanding; the broker is
// the single capacity owner. No report or reply is dropped to fit a cache.
func (c *supervisionCoordinator) sendGenerated(ctx context.Context, from, to couchmessage.Binding, id, body string, prompt bool) (couchmessage.Receipt, error) {
	c.mu.Lock()
	c.remember(id, from, c.now(), true)
	if prompt {
		v := c.sends[id]
		v.prompt = to.Slot
		c.sends[id] = v
	}
	c.mu.Unlock()
	receipt, err := c.service.broker.SendSystem(ctx, from, id, couchmessage.Route{Target: to.Slot}, body)
	if err != nil {
		c.admitted(id, nil)
	} else {
		c.admitted(id, &receipt)
	}
	return receipt, err
}

// reconcile checks the existing receipts, not delivery payloads. Terminal
// outcomes include queued expiry/cancellation with no endpoint callback.
func (c *supervisionCoordinator) reconcile() {
	c.mu.Lock()
	pending := make(map[string]supervisionSend, len(c.sends))
	for id, v := range c.sends {
		if v.admitted {
			pending[id] = v
		}
	}
	c.mu.Unlock()
	for id, v := range pending {
		receipt, known, delivering := c.service.broker.ReceiptSnapshot(id)
		// A terminal receipt may precede endpoint return on disconnect. The job
		// owns provenance until all delivery accounting has finished.
		if delivering || (known && !receipt.Status.Terminal()) {
			continue
		}
		c.mu.Lock()
		current, exists := c.sends[id]
		if exists && current == v {
			delete(c.sends, id)
		} else {
			exists = false
		}
		c.mu.Unlock()
		if exists && v.prompt != "" && (!known || receipt.Status != couchmessage.Submitted) {
			detail := "outcome unavailable"
			if known {
				detail = string(receipt.Status)
				if receipt.Detail != "" {
					detail += ": " + receipt.Detail
				}
			}
			c.notice(fmt.Sprintf("startup resend prompt to %s unconfirmed: %s; hub should request fresh status", v.prompt, detail))
		}
	}
}
func (c *supervisionCoordinator) delivered(m couchmessage.Message, r couchmessage.Receipt, err error) {
	if err != nil || r.Status != couchmessage.Submitted {
		return
	}
	hub, ok := c.service.hubBindingFor(m.To)
	if !ok || m.From != hub {
		return
	}
	c.service.mu.Lock()
	identity, known := c.service.workspaces[m.To]
	c.service.mu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.sends[m.ID]; ok && v.from == m.From && !v.system {
		if c.service.authority.hubs != nil && (v.ownership == nil || !known || v.hubRoot == "" || hubOwner(*v.ownership, identity) != v.hubRoot) {
			return
		}
		c.state.reply(m.To, v.at)
	}
}
func (c *supervisionCoordinator) waitState(b couchmessage.Binding) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state.reportID(b), c.state.notified(b)
}
func (c *supervisionCoordinator) waitNudged(b couchmessage.Binding, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.nudged(b, id)
}
func (c *supervisionCoordinator) tick(ctx context.Context, now time.Time, muted map[string]bool) {
	c.reconcile()
	hub, hasHub := c.service.hubBinding()
	caller := hub
	if !hasHub {
		connected := c.service.broker.Connected()
		if len(connected) == 0 {
			if log := c.service.facts.Load(); log != nil {
				log.snapshot(now, nil, nil, c.service.broker)
			}
			c.mu.Lock()
			c.state.tick(now, nil, false)
			c.mu.Unlock()
			return
		}
		caller = connected[0]
	}
	rows, err := c.service.broker.Actors(ctx, caller)
	if err != nil {
		return
	}
	live := c.service.liveness.Load()
	if log := c.service.facts.Load(); log != nil {
		liveness := map[string]couchmessage.SlotLiveness{}
		if live != nil {
			liveness = *live
		}
		log.snapshot(now, rows, liveness, c.service.broker)
	}
	available := map[couchmessage.Binding]bool{}
	var facts []supervisionFacts
	for _, r := range rows {
		if c.service.isHubBinding(r.Binding) {
			continue
		}
		_, available[r.Binding] = c.service.hubBindingFor(r.Binding)
		f := supervisionFacts{binding: r.Binding, known: r.Known, resting: r.Resting, pending: r.Pending, quiet: now.Sub(r.LastActivity)}
		if live != nil {
			if l, ok := (*live)[r.Binding.Slot]; ok && l.Binding == r.Binding {
				f.settled = l.Settled != nil && *l.Settled
			}
		}
		f.muted = muted[r.Binding.Slot]
		f.submission, _ = c.service.broker.Submission(r.Binding)
		facts = append(facts, f)
	}
	c.mu.Lock()
	effects := c.state.tick(now, facts, hasHub, available)
	c.mu.Unlock()
	for _, fx := range effects {
		if ctx.Err() != nil {
			return
		}
		if !c.service.isConnected(fx.binding) {
			continue
		}
		if fx.kind != supervisionPrompt {
			c.mu.Lock()
			current := c.state.reportID(fx.binding) == fx.report
			c.mu.Unlock()
			if !current {
				continue
			}
		}
		if fx.kind == supervisionEscalate {
			c.notice(fmt.Sprintf("slot %s: %d min with no later hub message or operator submission observed; inspect with couch --peek %s", fx.binding.Slot, int(fx.age/time.Minute), fx.binding.Slot))
			continue
		}
		hub, available := c.service.hubBindingFor(fx.binding)
		if !available {
			continue
		}
		from, to, body := hub, fx.binding, startupResendPrompt
		if fx.kind == supervisionNudge {
			from, to = fx.binding, hub
			body = fmt.Sprintf("slot %s: %d min with no later hub message or operator submission observed; inspect with couch --peek %s", fx.binding.Slot, int(fx.age/time.Minute), fx.binding.Slot)
		}
		id, err := newRequestID()
		if err == nil {
			_, err = c.sendGenerated(ctx, from, to, id, body, fx.kind == supervisionPrompt)
		}
		if err != nil {
			if fx.kind == supervisionPrompt {
				c.notice(fmt.Sprintf("startup resend prompt to %s unavailable: %v; hub should request fresh status", to.Slot, err))
			}
			continue
		}
		if fx.kind == supervisionNudge {
			c.waitNudged(fx.binding, fx.report)
		}
	}
}
