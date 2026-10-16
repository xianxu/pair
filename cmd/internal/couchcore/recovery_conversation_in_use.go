package couchcore

import (
	"context"
	"errors"
	"fmt"
)

// checkRecoveryConversationInUse is the one cross-slot check before a resume
// (pair#597): Couch's registry only, no process, lsof or ledger scan. It refuses
// only when another positively live slot records the same conversation; a dead
// or unknown slot, or an unreadable recording, never makes a collision.
func (c *Couch) checkRecoveryConversationInUse(ctx context.Context, record ThreadRecord, choice RecoveryChoice) error {
	if choice.NativeID == "" {
		return ctx.Err()
	}
	if c.Threads == nil {
		return errors.New("thread registry unavailable")
	}
	snapshot, err := c.Threads.PreviewSnapshotContext(ctx)
	if err != nil {
		return err
	}
	for _, other := range snapshot.Records {
		if other.Address == record.Address || RecordAgent(other) != choice.Agent || !recordPositivelyLive(c.Proc, other) {
			continue
		}
		if binding, _ := c.resumeEvidence(ctx, other); binding.NativeID == choice.NativeID || binding.RequestedNativeID == choice.NativeID {
			slot, err := c.recoverySlotAddress(ctx, other)
			if err != nil {
				slot = string(other.Address.Tag)
			}
			return fmt.Errorf("conversation %s is in use by %s", choice.NativeID, slot)
		}
	}
	return ctx.Err()
}

func recordPositivelyLive(proc ProcOps, record ThreadRecord) bool {
	for _, inc := range record.Incarnations {
		if observeExactProcessOrUnknown(proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) == Live {
			return true
		}
	}
	return false
}
