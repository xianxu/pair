package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrDraining = errors.New("Couch is draining; ordinary delivery is paused until cancellation")

// HoldDrain closes admission under the same lock used for mailbox reservation.
func (b *Broker) HoldDrain(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id == "" || b.drainGeneration != "" && b.drainGeneration != id {
		return ErrDraining
	}
	if b.drainGeneration == "" {
		b.drainGeneration = id
		b.drainCancelling = false
		b.drainControls = make(map[string]string)
		b.drainRevoked = make(map[string]bool)
	}
	return nil
}

// eligibleDrainHead is the one selection policy, including reserve revalidation.
func (b *Broker) eligibleDrainHead(a *SlotActor) (Message, bool) {
	m, ok := a.state.QueueHead()
	if b.drainGeneration != "" && (b.drainCancelling || b.drainControls[m.ID] != b.drainGeneration) {
		return Message{}, false
	}
	return m, ok
}

func (b *Broker) isDrainControl(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.drainControls[id] != ""
}

// SendDrain is internal-only; ordinary OriginSystem messages cannot use the lane.
func (b *Broker) SendDrain(ctx context.Context, generation string, from Binding, id string, to Binding, body string, expectedInstance ...string) (Receipt, error) {
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	instance := ""
	if len(expectedInstance) > 1 {
		return Receipt{}, errors.New("drain requires exactly one expected wrapper instance")
	}
	if len(expectedInstance) == 1 {
		instance = expectedInstance[0]
		if err := (DriveObservation{Instance: instance}).Validate(); err != nil {
			return Receipt{}, err
		}
	}

	if !validMessageID(id) {
		return Receipt{}, errors.New("invalid control ID")
	}
	if len(body) > 4096 {
		return Receipt{}, errors.New("drain notice exceeds body bound")
	}
	if err := ValidateBody(body); err != nil {
		return Receipt{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if generation == "" || b.drainGeneration != generation || b.drainCancelling {
		return Receipt{}, ErrDraining
	}
	if !b.caller(from) || !b.caller(to) {
		return Receipt{}, ErrUnavailable
	}
	if old, ok := b.receiptLocked(id); ok {
		if b.drainControls[id] != generation || old.Message.From != from || old.Message.To != to || old.Message.Body != body || old.Message.DrainInstance != instance {
			return Receipt{}, errors.New("conflicting drain control ID")
		}
		return old, nil
	}
	for control := range b.drainControls {
		if old, ok := b.receiptLocked(control); ok && old.Message.To == to && !old.Status.Terminal() {
			return Receipt{}, errors.New("recipient already holds a drain control")
		}
	}
	if len(b.drainControls) >= MaxActors {
		return Receipt{}, errors.New("drain control capacity reached")
	}
	now := b.now()
	m := Message{DrainInstance: instance, Origin: OriginSystem, ID: id, From: from, To: to, Body: body, AcceptedAt: now, Deadline: now.Add(PasteTimeout)}
	a := b.actors[to]
	if err := b.apply(a, Event{Kind: EnqueueDrainControl, Binding: to, Message: m, At: now}); err != nil {
		return Receipt{}, err
	}
	b.drainControls[id] = generation
	b.destinations[id] = Route{Target: to.Slot}
	a.wakeLocked()
	r, _ := b.receiptLocked(id)
	return r, nil
}

// ReleaseDrain follows queue-owner release and control reconciliation.
func (b *Broker) ReleaseDrain(generation string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.drainGeneration == "" {
		return nil
	}
	if b.drainGeneration != generation {
		return errors.New("another drain owns admission")
	}
	for id := range b.drainControls {
		if b.drainCancelling && !b.drainRevoked[id] {
			return errors.New("drain cancellation is not confirmed")
		}
		if r, ok := b.receiptLocked(id); !ok || !drainKnownTerminal(r.Status) {
			return errors.New("drain control is not reconciled")
		}
	}
	b.drainGeneration = ""
	b.drainCancelling = false
	b.drainControls = nil
	b.drainRevoked = nil
	for _, a := range b.actors {
		a.wakeLocked()
	}
	return nil
}

// DrainPending observes already handed work, distinct from the held peer FIFO.
func (b *Broker) DrainPending() map[Binding]string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[Binding]string)
	for binding, a := range b.actors {
		if m, ok := a.state.Pending(); ok && b.drainControls[m.ID] == "" {
			out[binding] = "pre-existing delivery is still active"
		}
	}
	return out
}

// CancelDrainControl revokes only this admitted notice. Setting cancelling
// under the reservation lock stops all further control promotions before IO.
// Unknown outcomes retain both holds; retry queries the same wrapper and ID.
func (b *Broker) CancelDrainControl(parent context.Context, generation, id string) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	if generation == "" || b.drainGeneration != generation || b.drainControls[id] != generation {
		b.mu.Unlock()
		return errors.New("drain control does not belong to current generation")
	}
	b.drainCancelling = true
	if b.drainRevoked[id] {
		b.mu.Unlock()
		return nil
	}
	r, known := b.receiptLocked(id)
	if !known {
		b.mu.Unlock()
		return errors.New("drain control receipt unknown")
	}
	a := b.actors[r.Message.To]
	if a == nil || !b.caller(r.Message.To) {
		b.mu.Unlock()
		return errors.New("drain control wrapper identity unavailable")
	}
	if drainKnownTerminal(r.Status) {
		job := b.jobs[id]
		if job != nil {
			job.cancel()
		}
		b.mu.Unlock()
		return b.confirmDrainRevoked(ctx, generation, id, job)
	}

	pending, hasPending := a.state.Pending()
	// No reservation/pending job means the control never left our queue.
	queued := false
	for _, m := range a.state.QueuedMessages() {
		queued = queued || m.ID == id
	}
	if queued && a.reservation != id && (!hasPending || pending.ID != id) && b.jobs[id] == nil {
		err := b.apply(a, Event{Kind: Cancel, Binding: a.binding, Message: Message{ID: id}, Detail: "drain cancelled before reservation", At: b.now()})
		if err == nil {
			b.drainRevoked[id] = true
		}
		b.mu.Unlock()
		return err
	}
	endpoint, capable := a.endpoint.(DrainEndpoint)
	if !capable {
		b.mu.Unlock()
		return errors.New("wrapper drain revocation unsupported; relaunch needed")
	}
	b.mu.Unlock()
	outcome, err := endpoint.RevokeDrain(ctx, id)
	if err != nil {
		return fmt.Errorf("drain revocation uncertain: %w", err)
	}
	if outcome.ID != id {
		return errors.New("drain revocation returned a different ID")
	}
	if r.Message.DrainInstance != "" && outcome.Instance != r.Message.DrainInstance {
		return errors.New("drain revocation returned a different wrapper instance")
	}
	status, detail := Cancelled, "drain cancelled before wrapper commit"
	switch outcome.Outcome {
	case "not-committed":
	case "already-committed":
		retained := outcome.Receipt
		if retained == nil || !drainReceiptMatches(retained.Message, r.Message) || !drainKnownTerminal(retained.Status) {
			return errors.New("drain committed handoff has no known terminal receipt")
		}
		status, detail = retained.Status, retained.Detail
	default:
		return errors.New("drain cancellation unconfirmed")
	}
	b.mu.Lock()
	if b.drainGeneration != generation || b.drainControls[id] != generation || b.actors[a.binding] != a || !b.caller(a.binding) {
		b.mu.Unlock()
		return errors.New("drain identity changed during revocation")
	}
	job := b.jobs[id]
	if job != nil {
		job.cancel()
	}
	// Keep model dispositions fenced while the cancelled delivery joins cleanup.
	applyErr := func() error {
		if err := b.apply(a, Event{Kind: Cancel, Binding: a.binding, Message: Message{ID: id}, Detail: detail, At: b.now()}); err != nil {
			return err
		}
		if m, ok := a.state.Pending(); ok && m.ID == id {
			if status == Submitted {
				if err := b.apply(a, Event{Kind: DeliveryStarted, Binding: a.binding, Message: m, At: b.now()}); err != nil {
					return err
				}
			}
			if err := b.apply(a, Event{Kind: DeliveryFinished, Binding: a.binding, Message: m, Status: status, Detail: detail, At: b.now()}); err != nil {
				return err
			}
		}
		// Only explicit retained evidence may strengthen an Indeterminate receipt.
		final := r
		final.Status = status
		final.Detail = boundedReceiptDetail(detail)
		final.RetainUntil = b.now().Add(ReceiptLifetime)
		b.receipts[id] = final
		return nil
	}()
	b.mu.Unlock()
	if applyErr != nil {
		return applyErr
	}
	return b.confirmDrainRevoked(ctx, generation, id, job)
}
func (b *Broker) confirmDrainRevoked(ctx context.Context, generation, id string, job *deliveryJob) error {
	if job != nil {
		select {
		case <-job.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.drainGeneration != generation || b.drainControls[id] != generation {
		return errors.New("drain changed while delivery cleanup joined")
	}
	b.drainRevoked[id] = true
	return nil
}

func drainKnownTerminal(s Status) bool { return s.Terminal() && s != Indeterminate }
func drainReceiptMatches(a, b Message) bool {
	return a.ID == b.ID && a.Origin == b.Origin && a.From == b.From && a.To == b.To && a.Body == b.Body && a.DrainInstance == b.DrainInstance
}

// BeginDrainCancel fences even a generation which admitted no notices yet.
func (b *Broker) BeginDrainCancel(generation string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if generation == "" || b.drainGeneration != generation {
		return errors.New("another drain owns admission")
	}
	b.drainCancelling = true
	return nil
}
