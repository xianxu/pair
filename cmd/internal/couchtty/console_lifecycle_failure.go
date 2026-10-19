package couchtty

import "github.com/xianxu/pair/cmd/internal/couchcore"

// SetOperationAttemptFactory wires queue-owned diagnostics independently of the
// dispatcher so preparation and adoption share one terminal owner.
func (c *Console) SetOperationAttemptFactory(factory func(couchcore.OperationCall) (*couchcore.OperationAttempt, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.beginAttempt = factory
}

func (c *Console) beginLifecycleAttempt(op string, args map[string]string, address couchcore.ThreadAddress, keys ...couchcore.ThreadRowKey) (*couchcore.OperationAttempt, error) {
	c.mu.Lock()
	factory := c.beginAttempt
	var target *couchcore.LifecycleFailureTarget
	for _, row := range menuRows(c.menu) {
		matched := address != (couchcore.ThreadAddress{}) && row.Address == address
		if len(keys) > 0 && keys[0] != (couchcore.ThreadRowKey{}) {
			key := keys[0]
			matched = row.RowKey == key || key.Kind == couchcore.ThreadTargetSlot && row.Target.Kind == couchcore.ThreadTargetSlot && row.Target.Slot.WorktreeRoot == key.SlotPath
		}
		if matched {
			frozen := couchcore.LifecycleFailureTargetForRow(row)
			target = &frozen
			break
		}
	}
	c.mu.Unlock()
	if factory == nil {
		return nil, nil
	}
	return factory(couchcore.OperationCall{Name: op, Args: args, FailureTarget: target})
}

func (c *Console) finishLifecycleAttempt(completed operationCompletion) operationCompletion {
	// Socket operations retain terminal ownership until their readiness callback.
	if completed.attempt == nil || completed.remote != nil {
		return completed
	}
	completed.value, completed.err = completed.attempt.Finish(completed.value, completed.err)
	if warning := completed.attempt.DiagnosticWarning(); warning != "" {
		c.Notify(Notice{Kind: "status", Body: warning})
	}
	return completed
}
