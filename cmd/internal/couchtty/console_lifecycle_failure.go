package couchtty

import (
	"context"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

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
	completed.value, completed.err = completed.attempt.FinishContext(c.lifetime, completed.value, completed.err)
	if warning := completed.attempt.DiagnosticWarning(); warning != "" {
		c.mu.Lock()
		label, _ := c.threadRowLocked(completed.origin.Address)
		c.mu.Unlock()
		c.Warn(Notice{Class: couchcore.NoticeLifecycleAttemptDiagnostic, Kind: "status", Thread: completed.origin.Address, SourceRoot: completed.origin.RowKey.SlotPath, Label: label, Body: warning})
	}
	return completed
}

// lifecycleCompletionWarningSource projects explicit successful-result warning
// facts. Generic Warning text also carries normal continuation and next-command
// feedback, so its wording grants no attention obligation. ParkResult cleanup
// is deliberately absent: its queue-owned attempt already publishes that fact.
func lifecycleCompletionWarningSource(value any, origin couchcore.ThreadAddress) (couchcore.ThreadAddress, bool) {
	thread, actionable := origin, false
	selectSource := func(address couchcore.ThreadAddress) {
		actionable = true
		if address != (couchcore.ThreadAddress{}) {
			thread = address
		}
	}
	if child, ok := value.(couchcore.StartedChild); ok {
		started, _ := child.Started()
		// Borrowed attachment preserves retained state during an ordinary
		// restart; it is inspectable feedback, not an operator obligation.
		// Clear only that typed field on a copy so configuration-publication
		// failures remain actionable without classifying warning text.
		started.Record.AttachmentNotice = ""
		if started.Warning() != "" {
			selectSource(started.Record.Thread)
		}
	}
	switch result := value.(type) {
	case couchcore.ArchiveResult:
		if result.SessionNotStopped {
			selectSource(result.Record.Address)
		}
	case couchcore.RebootResult:
		if result.SessionNotStopped {
			selectSource(result.Archived)
		}
	case couchcore.ParkRecoveryResult:
		if result.Diagnostic != "" {
			selectSource(result.Thread.Address)
		}
	case couchcore.SwitchAgentResult:
		if result.CleanupWarning != "" {
			selectSource(result.Record.Thread)
		}
	}
	return thread, actionable
}

// OperationLifetime is the joined console lifetime for remote terminal work.
func (c *Console) OperationLifetime() context.Context { return c.lifetime }

// publishOperationCompletion runs on an existing bounded worker, after child
// adoption. Publication and UI reduction have separate once owners: Stop may
// abandon an already-published result without repeating a caller or disk write.
func (c *Console) publishOperationCompletion(completed operationCompletion) operationCompletion {
	if completed.published == nil {
		completed.published = new(atomic.Bool)
	}
	if !completed.published.CompareAndSwap(false, true) {
		return completed
	}
	completed = c.finishLifecycleAttempt(completed)
	if completed.remote != nil && completed.remote.finished != nil {
		completed.remote.finished(completed.value, completed.err)
	}
	return completed
}
