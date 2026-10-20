package couchcore

import (
	"context"
	"errors"
)

// finalizeFailedRecoveryLaunch releases only a completed owned recovery attempt.
// The caller still holds the thread operation lease. No cleanup or retry occurs:
// unknown native evidence or a failed CAS leaves the exact claim occupied.
func (c *Couch) finalizeFailedRecoveryLaunch(ctx context.Context, thread ThreadRecord, nonce string, h Handle) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	transaction, err := CurrentStartTransaction(thread)
	if err != nil {
		return false, err
	}
	if h == nil || transaction.Nonce != nonce || transaction.Helper == nil || *transaction.Helper != (ProcessIdentity{PID: h.PID(), Identity: h.Identity()}) {
		return false, errors.New("failed recovery helper identity is unconfirmed")
	}
	select {
	case <-h.Exited():
		_ = h.Wait()
	default:
		return false, nil
	}
	// Exited is only the direct child. A surviving owned sidecar may still be
	// creating the terminal, so the whole group must have stopped too.
	if h.Alive() {
		return false, nil
	}
	binding, err := observeRecordSession(withoutSessionOwnerPass(ctx), c.Artifacts, thread)
	if err != nil {
		return false, err
	}
	if binding.Name != recordSessionName(thread) || binding.Name == "" {
		return false, errors.New("failed recovery terminal identity is unconfirmed")
	}
	observation := StartObservation{Helper: Dead, Session: PresenceAbsent, Registration: RegistrationAbsent}
	if binding.Present {
		observation.Session, observation.Registration = PresencePresent, RegistrationUnknown
	}
	decision, err := ReconcileStart(thread, observation)
	if err != nil || decision.Action != StartRollback {
		return false, err
	}
	if h.Alive() {
		return false, errors.New("failed recovery process group changed during observation")
	}
	_, err = c.Threads.updateExistingThreadContext(ctx, thread.Address, thread.Revision, func(next *ThreadRecord) error {
		advanced, err := AdvanceStartTransaction(*next, StartEvent{Kind: StartFailedConfirmed, Nonce: nonce})
		if err == nil {
			*next = advanced
		}
		return err
	})
	return err == nil, err
}
