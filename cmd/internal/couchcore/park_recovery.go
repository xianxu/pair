package couchcore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// ParkRecoveryDecision is a permitted effect, not a claim that it occurred.
type ParkRecoveryDecision string

const (
	parkRecoverySettled  ParkRecoveryDecision = "already-settled"
	parkRecoveryRollback ParkRecoveryDecision = "rollback-unpublished"
	parkRecoveryDrive    ParkRecoveryDecision = "drive-existing"
	parkRecoveryRetire   ParkRecoveryDecision = "retire-interrupted"
	parkRecoveryRefuse   ParkRecoveryDecision = "refuse"
)

type ParkRecoveryFacts struct {
	Open        bool
	ExactOwner  bool
	Owner       Liveness
	Terminal    SessionState
	Unpublished bool
}

// DecideParkRecovery keeps unknown observations distinct from positive absence.
func DecideParkRecovery(in ParkRecoveryFacts) ParkRecoveryDecision {
	if !in.Open {
		return parkRecoverySettled
	}
	if !in.ExactOwner {
		return parkRecoveryRefuse
	}
	if in.Unpublished {
		return parkRecoveryRollback
	}
	if in.Owner == Unknown {
		return parkRecoveryRefuse
	}
	if in.Terminal == SessionPresent {
		return parkRecoveryDrive
	}
	if in.Terminal == SessionAbsent && in.Owner == Dead {
		return parkRecoveryRetire
	}
	return parkRecoveryRefuse
}

func parkRecoveryOwnerMatches(r ThreadRecord) bool {
	if r.Park == nil || len(r.Incarnations) != 1 {
		return false
	}
	inc := r.Incarnations[0]
	return inc.Start == nil && (inc.State == IncarnationLive || inc.State == IncarnationUnknown) && inc.PID == r.Park.Identity.PID && inc.Identity == r.Park.Identity.ProcessIdentity
}

// ParkRecoveryResult describes bookkeeping recovery and its next explicit action.
// It deliberately has no Started method: recovery never launches anything.
type ParkRecoveryResult struct {
	Outcome         string
	OperatorOutcome string
	NextCommand     string
	Diagnostic      string
	Thread          ThreadRecord
}

func (r ParkRecoveryResult) ReceiptCode() string { return r.Outcome }
func (r ParkRecoveryResult) Warning() string {
	detail := r.Diagnostic
	if r.NextCommand != "" {
		if detail != "" {
			detail += "\n"
		}
		detail += r.OperatorOutcome + "; next: " + r.NextCommand
	}
	return detail
}
func (r ParkRecoveryResult) ReceiptTag() string { return string(r.Thread.Address.Tag) }

func (c *PairLifecycleController) parkActive(address ThreadAddress) bool {
	if c == nil {
		return false
	}
	c.workerOnce.Do(func() {
		if c.worker == nil {
			c.worker = newParkWorker(LifecycleParallelism)
		}
	})
	c.worker.mu.Lock()
	defer c.worker.mu.Unlock()
	_, active := c.worker.active[address]
	return active
}

// RecoverParkExpected retains the admitted transaction identity and lifecycle
// gate; callers retain confirmation and exact-target admission.
func (c *Couch) RecoverParkExpected(ctx context.Context, address ThreadAddress, nonce string) (ParkRecoveryResult, error) {
	return c.recoverParkExpected(ctx, address, nonce, true)
}

func (c *Couch) recoverParkExpected(ctx context.Context, address ThreadAddress, nonce string, exact bool) (result ParkRecoveryResult, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.Threads == nil || c.PairLifecycle == nil {
		return result, errors.New("park recovery is unavailable")
	}
	if err := c.PairLifecycle.validate(); err != nil {
		return result, err
	}
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%w; next: %s", retErr, c.ParkRecoveryAdvice(ctx, address))
		}
	}()
	ctx, release, err := c.hold(ctx, address, "recover-park")
	if err != nil {
		return result, err
	}
	defer release()
	current, err := c.Threads.recoveryReadThread(ctx, address)
	if err != nil {
		return result, err
	}
	result.Thread = current
	if current.Park != nil {
		if exact && current.Park.Identity.Nonce != nonce {
			return result, errors.New("park changed since recovery was selected; inspect and select Recover park again")
		}
		expected := current.Park.Identity.Nonce
		future, submitErr := c.PairLifecycle.submitFuture(ctx, address, expected, func(workCtx context.Context) (ParkResult, error) {
			return c.PairLifecycle.recoverInterruptedPark(workCtx, address, expected)
		})
		if submitErr != nil {
			return result, submitErr
		}
		recovered, recoverErr := future.Await(ctx)
		if recovered.Thread.Address == address {
			result.Thread = recovered.Thread
		}
		if recoverErr != nil {
			return result, recoverErr
		}
		current = result.Thread
		if current.Park != nil {
			return result, errors.New("park is still unresolved")
		}
		result.Outcome = "park-recovered"
	} else {
		result.Outcome = "park-already-settled"
	}
	if current.VerifiedPark == nil && len(current.ParkHistory) > 0 && current.ParkHistory[len(current.ParkHistory)-1].Tombstoned {
		result.Diagnostic = "Interrupted cleanup was retired; scrollback preservation may be incomplete. Conversation and history were retained."
	}
	result.OperatorOutcome, result.NextCommand, err = c.ParkRecoveryNextAction(ctx, current)
	if err != nil {
		return result, fmt.Errorf("%w; bookkeeping is resolved but its next action is unconfirmed; inspect the thread before retrying", err)
	}
	return result, nil
}

func (c *PairLifecycleController) recoverInterruptedPark(ctx context.Context, address ThreadAddress, nonce string) (ParkResult, error) {
	current, err := c.Threads.recoveryReadThread(ctx, address)
	if err != nil {
		return ParkResult{}, err
	}
	result := ParkResult{Thread: current}
	if current.Park == nil {
		return result, nil
	}
	if current.Park.Identity.Nonce != nonce {
		return result, errors.New("park transaction changed")
	}
	if !parkRecoveryOwnerMatches(current) {
		return result, errors.New("park ownership is replaced or ambiguous; inspect before recovering")
	}
	binding, err := observeRecordSession(ctx, c.Sessions, current)
	if err != nil {
		return result, err
	}
	if binding.Name == "" {
		return result, errors.New("exact terminal identity is unavailable")
	}
	// Completion reconciliation can share a lock held by the live helper's
	// cleanup. Refuse this case before entering it: recovery is not permission
	// to wait for an absent terminal's live/unknown owner to disappear.
	if !binding.Present {
		owner := observeExactProcess(c.Proc, ProcessIdentity{PID: current.Park.Identity.PID, Identity: current.Park.Identity.ProcessIdentity})
		if owner != Dead {
			if owner == Live {
				return result, errors.New("park is still finishing; owner is live; inspect before retrying recovery")
			}
			return result, fmt.Errorf("park owner is %s; inspect before retrying recovery", owner)
		}
	}
	result, completed, err := c.reconcileAttempts(ctx, result, current, binding.Name)
	if completed {
		return result, err
	}
	// A matching failure completion is diagnostic evidence, not success. Its
	// persisted failure can still be retired when exact death/absence are proved.
	if err != nil && !(errors.Is(err, errParkCleanupFailed) && result.Thread.Park != nil) {
		return result, err
	}
	current = result.Thread
	if !parkRecoveryOwnerMatches(current) {
		return result, errors.New("park owner changed during reconciliation")
	}
	owner := observeExactProcess(c.Proc, ProcessIdentity{PID: current.Park.Identity.PID, Identity: current.Park.Identity.ProcessIdentity})
	terminal := SessionAbsent
	if binding.Present {
		terminal = SessionPresent
	}
	decision := DecideParkRecovery(ParkRecoveryFacts{Open: true, ExactOwner: true, Owner: owner, Terminal: terminal})
	switch decision {
	case parkRecoveryDrive:
		return c.recover(ctx, address)
	case parkRecoveryRetire:
		if err := ctx.Err(); err != nil {
			return result, err
		}
		// Repeat terminal observation at the mutation boundary; a failed probe
		// cannot be interpreted as absence, even after an earlier positive absence.
		fresh, probeErr := observeRecordSession(ctx, c.Sessions, current)
		if probeErr != nil {
			return result, probeErr
		}
		if fresh.Present {
			return result, errors.New("terminal became live; inspect before retrying recovery")
		}
		retired, retireErr := c.Threads.RetireInterruptedPark(ctx, address, current.Revision, current.Park.Identity, c.Clock.Now())
		if retireErr == nil {
			result.Thread = retired
		}
		return result, retireErr
	default:
		return result, fmt.Errorf("park owner is %s or terminal state is unresolved; inspect before recovering", owner)
	}
}

// rollbackUnpublishedPark is called only by the initiating worker while it
// still owns the address, and only with positive no-publication authority.
func (c *PairLifecycleController) rollbackUnpublishedPark(result ParkResult, current ThreadRecord, cause error) (ParkResult, error) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rolled, err := c.Threads.rollbackUnpublishedPark(cleanupCtx, current)
	if err == nil {
		result.Thread = rolled
		return result, &parkRolledBackError{cause: cause, nonce: current.Park.Identity.Nonce}
	}
	return result, errors.Join(cause, err)
}

// parkRolledBackError marks a failure the controller itself proved harmless:
// the transaction was never published and its rollback committed. It is the
// only positive no-publication proof a full-quit leave accepts before minting
// a fresh park (#827). It adds no text, so refusals read as before.
type parkRolledBackError struct {
	cause error
	nonce string
}

func (e *parkRolledBackError) Error() string {
	if e.cause == nil {
		return "unpublished park rolled back"
	}
	return e.cause.Error()
}
func (e *parkRolledBackError) Unwrap() error { return e.cause }

var errParkCleanupFailed = errors.New("Pair cleanup failed")

// knownUnpublished distinguishes a definite failure from an uncertain rename.
func knownUnpublished(err error) bool {
	return pairlifecycle.PublicationOutcomeOf(err) == pairlifecycle.NotCommitted
}
