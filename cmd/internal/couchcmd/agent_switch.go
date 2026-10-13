package couchcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

type switchAdmission struct {
	Source, Target, Agent, RequestID string
	Requester                        couchmessage.Binding
	Address                          couchcore.ThreadAddress
	Prepared                         couchcore.PreparedAgentSwitch
	Content, Fallback                couchcore.SwitchContent
}

type switchDelivery struct {
	request orientation.Request
	state   orientation.DeliveryState
	err     error
}
type switchEntry struct {
	admission                    switchAdmission
	request                      couchmessage.Request
	model                        couchcore.AgentSwitch
	active, briefSent, observing bool
	delivery                     *orientation.Request
	early                        *switchDelivery
}

// switchCoordinator owns no authentication or process operations. Its hooks run
// outside the shared receipt mutex; the service supplies current source evidence.
// Entries share the existing global receipt capacity and terminal retention.
type switchCoordinator struct {
	ops                         *slotOperations
	now                         func() time.Time
	briefTimeout, settleTimeout time.Duration
	observe                     func(context.Context, switchAdmission) couchcore.AgentSwitchEvent
	enqueue                     func(switchAdmission, couchcore.SwitchContent, func(), func(any, error)) error
	requestBrief                func(context.Context, switchAdmission, string) error
	notice                      func(string)
	wake                        chan struct{}
	entries                     map[receiptKey]*switchEntry
}

func newSwitchCoordinator(ops *slotOperations, brief, settle time.Duration) *switchCoordinator {
	return &switchCoordinator{ops: ops, now: ops.now, briefTimeout: brief, settleTimeout: settle, wake: make(chan struct{}, 1), entries: map[receiptKey]*switchEntry{}}
}
func (c *switchCoordinator) signal() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}
func (c *switchCoordinator) sweepLocked() {
	c.ops.sweepLocked()
	for key := range c.entries {
		if _, ok := c.ops.receipts[key]; !ok {
			delete(c.entries, key)
		}
	}
}
func switchResponse(code, detail string) couchmessage.Response {
	return couchmessage.Response{Code: code, Error: detail}
}
func (c *switchCoordinator) answerLocked(key receiptKey) couchmessage.Response {
	r := c.ops.receipts[key]
	return couchmessage.Response{Code: "accepted", Operation: &r}
}

func (c *switchCoordinator) admit(_ context.Context, caller couchmessage.Binding, r couchmessage.Request, a switchAdmission) couchmessage.Response {
	if !r.Confirmed {
		return switchResponse("confirmation-required", "switch-agent requires --confirm")
	}
	a.RequestID, a.Agent, a.Requester = r.ID, r.Agent, caller
	key := receiptKey{caller.Scope, caller.Tag, r.ID}
	c.ops.mu.Lock()
	defer c.ops.mu.Unlock()
	c.sweepLocked()
	if response, ok := c.existingLocked(key, r); ok {
		return response
	}
	for k, entry := range c.entries {
		if k.id == r.ID {
			return switchResponse("id-conflict", "switch request ID already in use")
		}
		if entry.admission.Target == a.Target && !entry.model.Terminal() {
			return switchResponse("busy", "a switch is already pending for this slot")
		}
	}
	if len(c.ops.receipts) >= maxSlotOperationReceipts {
		return switchResponse("overloaded", "too many slot operations held")
	}
	if a.Target == "" {
		return switchResponse("invalid-request", "missing canonical target")
	}
	now := c.now()
	model, err := couchcore.NewAgentSwitch(couchcore.AgentSwitchSpec{RequestID: r.ID, Source: a.Source, Agent: r.Agent, Content: a.Content, Fallback: a.Fallback, AdmittedAt: now, BriefDeadline: now.Add(c.briefTimeout), OverallDeadline: now.Add(c.settleTimeout)})
	if err != nil {
		return switchResponse("invalid-request", err.Error())
	}
	receipt, err := couchmessage.ApplyReceiptEvent(couchmessage.OperationReceipt{}, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptAdmit, ID: r.ID, Op: r.Op, Target: r.Target, At: now})
	if err != nil {
		return switchResponse("invalid-request", err.Error())
	}
	receipt.Code = string(model.Phase())
	c.ops.receipts[key] = receipt
	c.entries[key] = &switchEntry{admission: a, request: r, model: model}
	return c.answerLocked(key)
}

// existing returns the original receipt before the service probes a source that
// may already have been replaced. Authentication is supplied by the caller;
// intent equality deliberately excludes mutable source observations.
func (c *switchCoordinator) existing(caller couchmessage.Binding, r couchmessage.Request) (couchmessage.Response, bool) {
	c.ops.mu.Lock()
	defer c.ops.mu.Unlock()
	c.sweepLocked()
	return c.existingLocked(receiptKey{caller.Scope, caller.Tag, r.ID}, r)
}
func (c *switchCoordinator) existingLocked(key receiptKey, r couchmessage.Request) (couchmessage.Response, bool) {
	if old := c.entries[key]; old != nil {
		original := old.request
		if original.Op != r.Op || original.Target != r.Target || original.Agent != r.Agent || original.Body != r.Body || original.SwitchRequest != r.SwitchRequest || original.Confirmed != r.Confirmed || original.SameBinary != r.SameBinary || original.ForceUnknown != r.ForceUnknown {
			return switchResponse("id-conflict", "switch request ID already names different content"), true
		}
		return c.answerLocked(key), true
	}
	if _, ok := c.ops.receipts[key]; ok {
		return switchResponse("id-conflict", "request ID already names another operation"), true
	}
	return couchmessage.Response{}, false
}

// canExecute binds a queued transaction to both its request and source. A
// cancelled job cannot borrow a subsequent switch's authority for that source.
func (c *switchCoordinator) canExecute(id, source string) bool {
	c.ops.mu.Lock()
	defer c.ops.mu.Unlock()
	for key, e := range c.entries {
		if key.id == id && e.model.Source() == source && e.model.Phase() == couchcore.AgentSwitchSwitching {
			return true
		}
	}
	return false
}

// activate runs only after the admission/callback acknowledgment was written.
func (c *switchCoordinator) activate(id string) {
	c.ops.mu.Lock()
	for key, e := range c.entries {
		if key.id == id && !e.model.Terminal() {
			e.active = true
		}
	}
	c.ops.mu.Unlock()
	c.signal()
}
func (c *switchCoordinator) cancel(id, reason string) {
	c.ops.mu.Lock()
	for key, e := range c.entries {
		if key.id == id {
			c.cancelLocked(key, e, reason)
		}
	}
	c.ops.mu.Unlock()
	c.signal()
}
func (c *switchCoordinator) cancelLocked(key receiptKey, e *switchEntry, reason string) {
	if e.model.Terminal() {
		return
	}
	next, _, _ := couchcore.AdvanceAgentSwitch(e.model, couchcore.AgentSwitchEvent{Kind: couchcore.AgentSwitchCancel, RequestID: key.id, At: c.now()})
	e.model = next
	c.projectLocked(key, e)
	if reason != "" {
		receipt := c.ops.receipts[key]
		receipt.Detail = reason
		c.ops.receipts[key] = receipt
	}
}
func (c *switchCoordinator) callback(_ context.Context, _ couchmessage.Binding, r couchmessage.Request, source string) couchmessage.Response {
	fallback := false
	c.ops.mu.Lock()
	defer func() {
		c.ops.mu.Unlock()
		if fallback && c.notice != nil {
			c.notice("Authored brief timed out; using previous-session pointers.")
		}
	}()
	c.sweepLocked()
	for key, e := range c.entries {
		if key.id == r.SwitchRequest {
			before := e.model
			next, effect, err := couchcore.AdvanceAgentSwitch(before, couchcore.AgentSwitchEvent{Kind: couchcore.AgentSwitchCallback, RequestID: r.SwitchRequest, Source: source, Agent: r.Agent, Body: r.Body, At: c.now()})
			e.model = next
			c.projectLocked(key, e)
			if err != nil {
				if effect == couchcore.AgentSwitchFallbackNotice {
					fallback = true
					c.signal()
				}
				return switchResponse("refused", err.Error())
			}
			if before.Phase() == couchcore.AgentSwitchWaitingBrief && next.Phase() == couchcore.AgentSwitchWaitingSettled {
				e.active = false
			}
			return c.answerLocked(key)
		}
	}
	return switchResponse("unknown-request", "switch request is unavailable")
}

// projectLocked keeps all operation-status clients on the existing receipt map.
func (c *switchCoordinator) projectLocked(key receiptKey, e *switchEntry) {
	receipt := c.ops.receipts[key]
	if receipt.Status.Terminal() {
		return
	}
	receipt.Code, receipt.Detail = string(e.model.Phase()), e.model.Reason()
	switch e.model.Phase() {
	case couchcore.AgentSwitchSwitching, couchcore.AgentSwitchDeliveryPending:
		receipt.Status = couchmessage.ReceiptRunning
	case couchcore.AgentSwitchSucceeded:
		receipt.Status = couchmessage.ReceiptSucceeded
	case couchcore.AgentSwitchRefused:
		receipt.Status = couchmessage.ReceiptRefused
	case couchcore.AgentSwitchFailure, couchcore.AgentSwitchUnconfirmed:
		receipt.Status = couchmessage.ReceiptFailed
	}
	if receipt.Status.Terminal() {
		receipt.Finished = c.now()
		e.early = nil
	}
	c.ops.receipts[key] = receipt
}

// step snapshots work under the receipt lock and probes outside it. Model
// comparison drops stale observations if a callback/cancel changed the request.
func (c *switchCoordinator) step(ctx context.Context) {
	c.ops.mu.Lock()
	if ctx.Err() != nil {
		for key, e := range c.entries {
			c.cancelLocked(key, e, "Couch stopped before switch completion")
		}
		c.ops.mu.Unlock()
		return
	}
	c.sweepLocked()
	keys := make([]receiptKey, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}
	c.ops.mu.Unlock()
	for _, key := range keys {
		c.stepEntry(ctx, key)
	}
}
func (c *switchCoordinator) stepEntry(ctx context.Context, key receiptKey) {
	c.ops.mu.Lock()
	e := c.entries[key]
	if e == nil || e.model.Terminal() || e.observing {
		c.ops.mu.Unlock()
		return
	}
	before := e.model
	next, effect, _ := couchcore.AdvanceAgentSwitch(before, couchcore.AgentSwitchEvent{Kind: couchcore.AgentSwitchTick, RequestID: key.id, At: c.now()})
	e.model = next
	c.projectLocked(key, e)
	fallback := effect == couchcore.AgentSwitchFallbackNotice
	if e.model.Terminal() || !e.active {
		c.ops.mu.Unlock()
		if fallback && c.notice != nil {
			c.notice("Authored brief timed out; using previous-session pointers.")
		}
		return
	}
	if e.model.Phase() != couchcore.AgentSwitchWaitingBrief && e.model.Phase() != couchcore.AgentSwitchWaitingSettled {
		c.ops.mu.Unlock()
		return
	}
	send := e.model.Phase() == couchcore.AgentSwitchWaitingBrief && !e.briefSent
	if send {
		e.briefSent = true
	}
	e.observing = true
	a := e.admission
	a.Content = e.model.Content()
	observedModel := e.model
	c.ops.mu.Unlock()
	if fallback && c.notice != nil {
		c.notice("Authored brief timed out; using previous-session pointers.")
	}
	if send {
		var err error
		if c.requestBrief == nil {
			err = fmt.Errorf("outgoing agent request transport unavailable")
		} else {
			err = c.requestBrief(ctx, a, key.id)
		}
		if err != nil {
			c.ops.mu.Lock()
			if current := c.entries[key]; current == e {
				e.observing = false
				c.cancelLocked(key, e, "could not request authored brief: "+err.Error())
			}
			c.ops.mu.Unlock()
			return
		}
	}
	var event couchcore.AgentSwitchEvent
	if c.observe != nil {
		event = c.observe(ctx, a)
	} else {
		event = couchcore.AgentSwitchEvent{Source: a.Source}
	}
	event.Kind, event.RequestID, event.At = couchcore.AgentSwitchObserve, key.id, c.now()
	c.ops.mu.Lock()
	if c.entries[key] != e {
		c.ops.mu.Unlock()
		return
	}
	e.observing = false
	if ctx.Err() != nil {
		c.cancelLocked(key, e, "Couch stopped before switch completion")
		c.ops.mu.Unlock()
		return
	}
	if e.model != observedModel || !e.active {
		c.ops.mu.Unlock()
		return
	}
	next, effect, _ = couchcore.AdvanceAgentSwitch(e.model, event)
	e.model = next
	c.projectLocked(key, e)
	content := e.model.Content()
	c.ops.mu.Unlock()
	if effect != couchcore.AgentSwitchEnqueue {
		return
	}
	if ctx.Err() != nil {
		c.cancel(key.id, "Couch stopped before switch completion")
		return
	}
	if !c.canExecute(key.id, a.Source) {
		return
	}
	if c.enqueue == nil {
		c.finished(key, nil, fmt.Errorf("switch queue unavailable"))
		return
	}
	err := c.enqueue(a, content, func() {}, func(value any, err error) { c.finished(key, value, err) })
	if err != nil {
		c.finished(key, nil, err)
	}
}

func (c *switchCoordinator) finished(key receiptKey, value any, err error) {
	c.ops.mu.Lock()
	defer c.ops.mu.Unlock()
	e := c.entries[key]
	if e == nil || e.model.Terminal() || e.model.Phase() != couchcore.AgentSwitchSwitching {
		return
	}
	result, ok := value.(couchcore.SwitchAgentResult)
	kind := couchcore.AgentSwitchRegistered
	if err != nil || !ok || result.Outcome != couchcore.SwitchStarted {
		kind = couchcore.AgentSwitchFailed
	}
	next, _, _ := couchcore.AdvanceAgentSwitch(e.model, couchcore.AgentSwitchEvent{Kind: kind, RequestID: key.id, At: c.now()})
	e.model = next
	if result.Orientation != nil {
		copy := *result.Orientation
		e.delivery = &copy
	}
	c.projectLocked(key, e)
	receipt := c.ops.receipts[key]
	if ok {
		receipt.Warning = result.Warning
		if result.Record.Thread.Tag != "" {
			receipt.Tag = string(result.Record.Thread.Tag)
		}
	}
	if err != nil {
		receipt.Detail = err.Error()
	}
	if kind == couchcore.AgentSwitchFailed && ok {
		receipt.Diagnostic = string(result.Outcome)
	}
	c.ops.receipts[key] = receipt
	if e.model.Phase() == couchcore.AgentSwitchDeliveryPending && e.delivery == nil {
		c.deliverLocked(key, e, orientation.DeliveryState{Phase: orientation.DeliveryIndeterminate}, fmt.Errorf("registered target has no delivery receipt"))
	}
	if early := e.early; early != nil {
		e.early = nil
		if e.delivery != nil && early.request == *e.delivery {
			c.deliverLocked(key, e, early.state, early.err)
		}
	}
}

func (c *switchCoordinator) delivery(address couchcore.ThreadAddress, request orientation.Request, state orientation.DeliveryState, err error) {
	if request.Validate() != nil || (!state.Terminal() && err == nil) {
		return
	}
	c.ops.mu.Lock()
	defer c.ops.mu.Unlock()
	for key, e := range c.entries {
		if e.admission.Address != address || e.model.Terminal() || e.model.Content().Body != request.Body || e.model.Agent() != request.Agent {
			continue
		}
		if e.model.Phase() == couchcore.AgentSwitchSwitching {
			e.early = &switchDelivery{request: request, state: state, err: err}
			continue
		}
		if e.delivery != nil && *e.delivery == request {
			c.deliverLocked(key, e, state, err)
		}
	}
}
func (c *switchCoordinator) deliverLocked(key receiptKey, e *switchEntry, state orientation.DeliveryState, err error) {
	kind := couchcore.AgentSwitchDeliveryUnconfirmed
	if state.Phase == orientation.DeliverySubmitted && err == nil {
		kind = couchcore.AgentSwitchDeliverySubmitted
	}
	next, _, _ := couchcore.AdvanceAgentSwitch(e.model, couchcore.AgentSwitchEvent{Kind: kind, RequestID: key.id, At: c.now()})
	e.model = next
	c.projectLocked(key, e)
	if err != nil {
		receipt := c.ops.receipts[key]
		receipt.Detail = err.Error()
		c.ops.receipts[key] = receipt
	}
}
func (c *switchCoordinator) run(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.ops.mu.Lock()
			for key, e := range c.entries {
				c.cancelLocked(key, e, "Couch stopped before switch completion")
			}
			c.ops.mu.Unlock()
			return
		case <-ticker.C:
		case <-c.wake:
		}
		c.step(ctx)
	}
}
