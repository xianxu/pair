package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// LeaveParkBudget bounds one confirmed full quit (#827): gate waits, queueing,
// drives and reconciliation all share it, and every retry uses the same
// absolute deadline. It is a scheduling budget, not a process-exit ceiling --
// a synchronous durable write still settles and is joined after it ends.
const LeaveParkBudget = 60 * time.Second

// leaveParkMaxDrives caps the drives one target gets in one full quit,
// including its first.
const leaveParkMaxDrives = 3

// leaveParkAction is which existing controller entry a drive uses.
type leaveParkAction uint8

const (
	// leaveParkFresh begins a new park transaction on a live thread.
	leaveParkFresh leaveParkAction = iota + 1
	// leaveParkRecover drives the thread's existing transaction, nonce kept.
	leaveParkRecover
)

// leaveParkEvidence is what one drive proved, read back from its result and
// the authoritative record.
type leaveParkEvidence struct {
	// Verified: the drive returned verified inactive history and no open park.
	Verified bool
	// Failed: the drive returned an error.
	Failed bool
	// OpenTransaction: the record still carries a park transaction.
	OpenTransaction bool
	// RolledBack: the controller proved the transaction unpublished and its
	// rollback committed (parkRolledBackError).
	RolledBack bool
	// SameSource: the record's sole live incarnation is still the one this
	// leave captured before its first drive.
	SameSource bool
	// BudgetLeft: the full-quit budget had not ended.
	BudgetLeft bool
}

type leaveParkDecision uint8

const (
	leaveParkSettled leaveParkDecision = iota + 1
	leaveParkRetryFresh
	leaveParkRetryRecover
	// leaveParkRefused needs the operator: the evidence permits no safe drive.
	leaveParkRefused
	// leaveParkExhausted: a retry would be safe but no drive or budget remains.
	leaveParkExhausted
)

// decideLeaveParkRetry is the whole retry policy of a full-quit target (#827).
// Verified success is absorbing. A failure retries only on positive evidence:
// an open transaction goes back through Recover under its own nonce, and a
// fresh park needs the controller's proof that nothing was published plus the
// same guarded incarnation. A drive that returned without error but without
// verified history (an unknown or closed transaction) is the operator's.
func decideLeaveParkRetry(ev leaveParkEvidence, drives int) leaveParkDecision {
	if ev.Verified {
		return leaveParkSettled
	}
	if !ev.Failed {
		return leaveParkRefused
	}
	if !ev.OpenTransaction && !(ev.RolledBack && ev.SameSource) {
		return leaveParkRefused
	}
	if drives >= leaveParkMaxDrives || !ev.BudgetLeft {
		return leaveParkExhausted
	}
	if ev.OpenTransaction {
		return leaveParkRetryRecover
	}
	return leaveParkRetryFresh
}

// LeaveUnresolved names one thread a full quit could not prove parked, with
// the reason and the operator's next action.
type LeaveUnresolved struct {
	Address ThreadAddress
	Reason  string
	Next    string
}

// leaveParkOne is leaveOne's LeavePark form (#827). It holds the thread for
// every drive it makes, joins each admitted park future before deciding, and
// retries only what decideLeaveParkRetry permits.
func (c *Couch) leaveParkOne(ctx context.Context, address ThreadAddress) (leaveOutcome, error) {
	if c.PairLifecycle == nil {
		return leaveNone, errors.New("Pair lifecycle controller is unavailable")
	}
	ctx, release, err := c.holdWait(ctx, address, "leave")
	if err != nil {
		return leaveNone, fmt.Errorf("leave couch: wait for %s: %w", address.Tag, err)
	}
	defer release()
	record, err := c.Threads.GetThread(address)
	if errors.Is(err, ErrThreadNotFound) {
		return leaveNone, nil // archived while Leave waited
	}
	if err != nil {
		return leaveNone, err
	}
	action := leaveParkRecover
	var source ProcessIdentity
	if record.Park == nil {
		record, err = c.adoptOwnedAttachment(ctx, record)
		if err != nil {
			return leaveNone, err
		}
		if !hasActiveIncarnation(record) {
			return leaveNone, nil
		}
		if len(record.Incarnations) != 1 || record.Incarnations[0].State != IncarnationLive {
			return leaveSkipped, nil
		}
		action = leaveParkFresh
		source = ProcessIdentity{PID: record.Incarnations[0].PID, Identity: record.Incarnations[0].Identity}
	}
	var failures []error
	for drives := 1; ; drives++ {
		if c.onLeaveParkDrive != nil {
			c.onLeaveParkDrive(address, action)
		}
		var result ParkResult
		var driveErr error
		if action == leaveParkFresh {
			result, driveErr = c.PairLifecycle.parkJoined(ctx, address, threadrecord.SleepFullQuit)
		} else {
			result, driveErr = c.PairLifecycle.recoverJoined(ctx, address)
		}
		after, readErr := c.Threads.GetThread(address)
		var rolledBack *parkRolledBackError
		ev := leaveParkEvidence{
			Verified:   driveErr == nil && result.Thread.VerifiedPark != nil && result.Thread.Park == nil,
			Failed:     driveErr != nil,
			RolledBack: errors.As(driveErr, &rolledBack),
			BudgetLeft: ctx.Err() == nil,
		}
		if readErr == nil {
			ev.OpenTransaction = after.Park != nil
			ev.SameSource = source.PID != 0 && len(after.Incarnations) == 1 &&
				after.Incarnations[0].State == IncarnationLive &&
				after.Incarnations[0].PID == source.PID && after.Incarnations[0].Identity == source.Identity
		}
		if driveErr != nil {
			failures = append(failures, driveErr)
		}
		switch decideLeaveParkRetry(ev, drives) {
		case leaveParkSettled:
			return leaveParked, nil
		case leaveParkRetryFresh:
			action = leaveParkFresh
			continue
		case leaveParkRetryRecover:
			action = leaveParkRecover
			continue
		}
		cause := errors.Join(append(failures, readErr)...)
		if cause == nil {
			cause = errors.New("did not produce verified inactive history")
		}
		// No recovery advice here: Leave names it once, as the target's Next.
		return leaveNone, fmt.Errorf("leave couch: park %s (%d drive(s)): %w", address.Tag, drives, cause)
	}
}

// leaveParkUnresolved accounts for every target a full quit did not prove
// parked, in snapshot order (#827): failed, not started, skipped (an incarnation
// Couch cannot vouch for, so it never parks it) and unreadable. Anything but a
// verified park keeps Couch open. Each next action is read from the record as
// it is now -- an open transaction has its exact recovery -- with lookups run
// at the same fan-out bound so many unresolved threads do not queue serially.
func (c *Couch) leaveParkUnresolved(ctx context.Context, records []ThreadRecord, outcomes []leaveOutcome, errs []error, started []bool, unreadable []ThreadAddress) ([]LeaveUnresolved, []error) {
	var unresolved []LeaveUnresolved
	var failures []error
	for i, record := range records {
		switch {
		case !started[i]:
			reason := "not started before the full quit ended"
			if errs[i] != nil {
				reason = errs[i].Error()
			}
			unresolved = append(unresolved, LeaveUnresolved{Address: record.Address, Reason: reason})
		case errs[i] != nil:
			unresolved = append(unresolved, LeaveUnresolved{Address: record.Address, Reason: errs[i].Error()})
		case outcomes[i] == leaveSkipped:
			unresolved = append(unresolved, LeaveUnresolved{Address: record.Address, Reason: "not parked: its process state is unknown", Next: "inspect it, then quit again (Alt+x)"})
			failures = append(failures, fmt.Errorf("leave couch: %s: skipped, process state unknown", record.Address.Tag))
		}
	}
	adviceCtx := context.WithoutCancel(ctx)
	units := make(chan struct{}, LifecycleParallelism)
	var wg sync.WaitGroup
	for i := range unresolved {
		if unresolved[i].Next != "" {
			continue
		}
		wg.Add(1)
		units <- struct{}{}
		go func(u *LeaveUnresolved) {
			defer wg.Done()
			defer func() { <-units }()
			u.Next = "resolve the error, then quit again (Alt+x)"
			if record, err := c.Threads.GetThread(u.Address); err == nil && record.Park != nil {
				lookup, cancel := context.WithTimeout(adviceCtx, recoveryObservationTimeout)
				u.Next = c.ParkRecoveryAdvice(lookup, u.Address)
				cancel()
			}
		}(&unresolved[i])
	}
	wg.Wait()
	// An unreadable record may still hold a live agent: Couch cannot park what
	// it cannot read, so the full quit is not complete.
	for _, address := range unreadable {
		unresolved = append(unresolved, LeaveUnresolved{Address: address, Reason: "thread record is unreadable", Next: "inspect the slot's thread record, then quit again (Alt+x)"})
		failures = append(failures, fmt.Errorf("leave couch: %s: thread record is unreadable", address.Tag))
	}
	return unresolved, failures
}

// FormatLeaveParkSummary is the one complete, copyable report of a full quit
// that did not finish: every unresolved thread once, in snapshot order, with
// its next action. It never reads as a completed shutdown.
func FormatLeaveParkSummary(result LeaveResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Full quit incomplete: %d parked, %d unresolved; Couch stays open.", len(result.Parked), len(result.Unresolved))
	for _, u := range result.Unresolved {
		fmt.Fprintf(&b, "\n- %s: %s; next: %s", u.Address.Tag, strings.ReplaceAll(u.Reason, "\n", "; "), u.Next)
	}
	return b.String()
}

// LeaveParkIncompleteError is a full quit that could not prove every thread
// parked. Its text is the complete summary (FormatLeaveParkSummary), so every
// surface that shows the error -- the switcher notice and its copy key, the
// CLI -- carries every unresolved thread and its next action (#827).
type LeaveParkIncompleteError struct {
	Result LeaveResult
	Err    error
}

func (e *LeaveParkIncompleteError) Error() string { return FormatLeaveParkSummary(e.Result) }
func (e *LeaveParkIncompleteError) Unwrap() error { return e.Err }
