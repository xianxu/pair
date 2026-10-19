package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// borrowedClientReason screens only client-helper ownership. It never grants
// authority over an agent, wrapper, terminal server, or lifecycle transaction.
func borrowedClientReason(record ThreadRecord, proc ProcOps) string {
	if record.Reservation || record.Park != nil || record.VerifiedPark != nil || record.Continuation != nil && record.Continuation.Phase != checkpoint.Complete {
		return "an open lifecycle claim remains"
	}
	for _, inc := range record.Incarnations {
		if inc.Start != nil {
			return "an open start claim remains"
		}
	}
	if len(record.Incarnations) != 1 {
		return "no unique retained client helper is identified"
	}
	inc := record.Incarnations[0]
	if inc.PID <= 0 || inc.Identity == "" || inc.State != IncarnationLive && inc.State != IncarnationUnknown {
		return "the retained client helper identity is incomplete"
	}
	switch observeExactProcessOrUnknown(proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) {
	case Live:
		return "the retained client helper is still alive"
	case Unknown:
		return "the retained client helper could not be proved dead"
	}
	return ""
}

// RepairDeadClientHelper is a revision-checked replacement of one proven-dead
// Pair client helper. It preserves the producing launch, binding, native and
// terminal witnesses, and all park/history/claim fields. The new Handle proves
// local client ownership; neither an agent PID nor a wrapper PID is accepted.
// This single-record atomic publication deliberately refuses pending journals
// rather than replaying lifecycle work merely because Continue was requested.
func (s *ThreadStore) RepairDeadClientHelper(ctx context.Context, before ThreadRecord, client ActorRecord, owned Handle, proc ProcOps) error {
	if s == nil || s.readOnly {
		return errors.New("client repair requires a writable store")
	}
	view := *s
	view.readOnly = true
	backend, err := view.storeForAddressContext(ctx, before.Address)
	if err != nil {
		return err
	}
	target := *backend
	target.readOnly = false
	published := false
	err = target.withPreviewLockContext(ctx, func() error {
		raw, err := backend.readPayloadContext(ctx, backend.recordPath(before.Address))
		if err != nil {
			return err
		}
		current, err := target.decodeThreadRaw(before.Address, raw)
		if err != nil {
			return err
		}
		if current.Revision != before.Revision {
			return &ThreadRevisionError{Address: before.Address, Want: before.Revision, Got: current.Revision}
		}
		if reason := borrowedClientReason(current, proc); reason != "" {
			return errors.New(reason)
		}
		old := current.Incarnations[0]
		if len(before.Incarnations) != 1 || old.PID != before.Incarnations[0].PID || old.Identity != before.Incarnations[0].Identity {
			return errors.New("retained client identity changed before repair")
		}
		if client.Thread != before.Address || client.Shape != StartWarmReattach || owned == nil || owned.PID() != client.PID || owned.Identity() != client.Identity || client.PID <= 0 || client.Identity == "" {
			return errors.New("replacement is not the locally owned attachment client")
		}
		next := cloneThreadRecord(current)
		next.Incarnations[0].PID = client.PID
		next.Incarnations[0].Identity = client.Identity
		next.Incarnations[0].State = IncarnationLive
		next.Revision++
		if err := ValidateThreadRecord(next); err != nil {
			return err
		}
		if target.hooks.BeforeUpdate != nil {
			if err := target.hooks.BeforeUpdate(current, next); err != nil {
				return err
			}
		}
		encoded, err := encodeThreadRecord(next)
		if err != nil {
			return err
		}
		guard := func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if !owned.Alive() || owned.PID() != client.PID || owned.Identity() != client.Identity {
				return errors.New("local attachment client is no longer alive")
			}
			if observeExactProcessOrUnknown(proc, ProcessIdentity{PID: old.PID, Identity: old.Identity}) != Dead {
				return errors.New("retained client proof changed before repair")
			}
			return nil
		}
		if err := target.writeStoreAtomicLockedCheckedContext(ctx, target.recordPath(before.Address), append(encoded, '\n'), guard); err != nil {
			return fmt.Errorf("repair dead client helper: %w", err)
		}
		published = true
		return nil
	})
	if err == nil && !published {
		return errors.New("retained client store is no longer present")
	}
	return err
}

func (c *Couch) attachmentClientNotice(ctx context.Context, thread ThreadRecord, client ActorRecord, owned Handle) string {
	reason := borrowedClientReason(thread, c.Proc)
	if reason == "" {
		if err := c.Threads.RepairDeadClientHelper(ctx, thread, client, owned, c.Proc); err != nil {
			reason = err.Error()
		}
	}
	if reason == "" {
		return ""
	}
	return "BORROWED ATTACH: " + reason + "; retained lifecycle state was preserved."
}
