package wrapcmd

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// peerDeliveryBatch retains immutable membership under the existing input owner.
// Readiness permits assembly only; the ordinary dispatcher checks input again.
type peerDeliveryBatch struct {
	phase   couchmessage.BatchPreparationPhase
	members []couchmessage.Message
}

func (d *peerDelivery) prepareBatch(head couchmessage.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.binding.Agent != "claude" {
		return errors.New("wrapper does not support peer batches")
	}
	if err := d.enqueueLocked(head); err != nil {
		return err
	}
	d.batch = &peerDeliveryBatch{phase: couchmessage.BatchPreparing}
	return nil
}
func (d *peerDelivery) batchReady() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.batch != nil && d.batch.phase == couchmessage.BatchReady && !d.current.Status.Terminal()
}
func (d *peerDelivery) finalizeBatch(id string, messages []couchmessage.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := couchmessage.ValidateBatch(messages, d.binding, id); err != nil {
		return err
	}
	if d.batch == nil || d.current.Message.ID != id || d.current.Status.Terminal() || d.released || d.exited {
		return errors.New("peer batch preparation unavailable")
	}
	head, original := messages[0], d.current.Message
	if !head.SameProvenance(original) || head.ID != original.ID || head.From != original.From || head.To != original.To || head.Body != original.Body || !head.Deadline.Equal(original.Deadline) || !head.AcceptedAt.Equal(original.AcceptedAt) {
		return errors.New("peer batch head changed")
	}
	for _, m := range messages[1:] {
		if _, ok := d.retainedLocked(m.ID); ok {
			return errors.New("delivery already committed")
		}
	}
	phase, err := couchmessage.AdvanceBatchPreparation(d.batch.phase, couchmessage.BatchFinalize)
	if err != nil {
		return err
	}
	d.batch.members = append([]couchmessage.Message(nil), messages...)
	d.batch.phase = phase
	d.signal()
	return nil
}

// membersLocked projects the shared outcome onto the original identities.
func (d *peerDelivery) membersLocked() []couchmessage.Message {
	if d.batch != nil && d.batch.phase == couchmessage.BatchFrozen {
		return d.batch.members
	}
	if d.current.Message.ID == "" {
		return nil
	}
	return []couchmessage.Message{d.current.Message}
}

// batchStatus snapshots identity, outcome and assembly permission atomically.
func (d *peerDelivery) batchStatus(id string) (couchmessage.Receipt, bool, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	r, ok := d.retainedLocked(id)
	ready := ok && d.current.Message.ID == id && d.batch != nil && d.batch.phase == couchmessage.BatchReady && !r.Status.Terminal()
	return r, ok, ready
}
