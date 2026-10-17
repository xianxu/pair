package couchmessage

import "context"

// freezeBatch transfers one current FIFO prefix into the prepared head's
// ownership group. No endpoint IO occurs under the broker mutex.
func (b *Broker) freezeBatch(ctx context.Context, a *SlotActor, head Message, job *deliveryJob) ([]Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	pending, ok := a.state.Pending()
	if !ok || pending != head || b.actors[a.binding] != a || !b.caller(head.To) || !b.caller(head.From) || b.jobs[head.ID] != job {
		return nil, ErrUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := b.now()
	for _, m := range a.state.QueuedMessages() {
		if !b.caller(m.From) {
			_ = b.apply(a, Event{Kind: Cancel, Binding: a.binding, Message: Message{ID: m.ID}, Detail: "sender disconnected before delivery", At: now})
		}
	}
	queue := a.state.QueuedMessages()
	for i := range queue {
		queue[i].Deadline = now.Add(PasteTimeout)
	}
	group := SelectBatch(head, queue, a.batchMaxChars)
	if err := b.apply(a, Event{Kind: FreezeBatch, Binding: a.binding, Message: head, Messages: group, At: now}); err != nil {
		return nil, err
	}
	for _, member := range group {
		b.jobs[member.ID] = job
	}
	return group, nil
}
