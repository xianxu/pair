package couchcmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

const startupResendPrompt = "Couch restarted and the hub is available. If your last report to the hub is still unanswered and relevant, resend it now. Otherwise do nothing."

type supervisionSend struct {
	from   couchmessage.Binding
	at     time.Time
	system bool
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
	for key, v := range c.sends {
		if now.Sub(v.at) > 2*couchmessage.ReceiptLifetime {
			delete(c.sends, key)
		}
	}
	if _, exists := c.sends[id]; exists {
		return false
	}
	if len(c.sends) >= couchmessage.MaxReceipts {
		return false
	}
	c.sends[id] = supervisionSend{from: from, at: now, system: system}
	return true
}

// sending is called only for authenticated CLI sends, before resolving hub so
// a live supervisor can retain the fact of a refused hub report (not its body).
func (c *supervisionCoordinator) sending(from couchmessage.Binding, r couchmessage.Request) {
	hub, hasHub := c.service.hubBinding()
	toHub := r.Target == couchmessage.HubTarget || (hasHub && r.Target == hub.Slot)
	if !toHub && (!hasHub || from != hub) {
		return
	}
	submission, connected := c.service.broker.Submission(from)
	if !connected {
		return
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.remember(r.ID, from, now, false) {
		return
	}
	if toHub && (!hasHub || from != hub) {
		c.state.report(from, r.ID, now, submission, completionHint(r.Body))
	}
}
func (c *supervisionCoordinator) systemMessage(m couchmessage.Message) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.sends[m.ID]
	return ok && v.from == m.From && v.system
}
func (c *supervisionCoordinator) delivered(m couchmessage.Message, r couchmessage.Receipt, err error) {
	if err != nil || r.Status != couchmessage.Submitted {
		return
	}
	hub, ok := c.service.hubBinding()
	if !ok || m.From != hub {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.sends[m.ID]; ok && v.from == m.From && !v.system {
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
	hub, hasHub := c.service.hubBinding()
	caller := hub
	if !hasHub {
		connected := c.service.broker.Connected()
		if len(connected) == 0 {
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
	var facts []supervisionFacts
	for _, r := range rows {
		if hasHub && r.Binding == hub {
			continue
		}
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
	effects := c.state.tick(now, facts, hasHub)
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
		from, to, body := hub, fx.binding, startupResendPrompt
		if fx.kind == supervisionNudge {
			from, to = fx.binding, hub
			body = fmt.Sprintf("slot %s: %d min with no later hub message or operator submission observed; inspect with couch --peek %s", fx.binding.Slot, int(fx.age/time.Minute), fx.binding.Slot)
		}
		id, err := newRequestID()
		if err == nil {
			c.mu.Lock()
			remembered := c.remember(id, from, now, true)
			c.mu.Unlock()
			if remembered {
				_, err = c.service.broker.Send(ctx, from, id, couchmessage.Route{Target: to.Slot}, body)
			} else {
				err = fmt.Errorf("supervision observation capacity reached")
			}
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
