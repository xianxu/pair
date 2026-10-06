package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// FailureClass says what an operator can do about a resource that did not
// converge (pair#387).
type FailureClass string

const (
	// FailureRetryable: something else is running or the run was cut short;
	// running it again can succeed.
	FailureRetryable FailureClass = "retryable"
	// FailureHandoff: reconcile cannot fix it; the repository's :0 agent can.
	FailureHandoff FailureClass = "handoff"
	// FailureUnknown: the resource could not be observed.
	FailureUnknown FailureClass = "unknown"
	// FailureHold: reconcile may not act now (a running or unknown agent, saved
	// work full); the cause names what clears it.
	FailureHold FailureClass = "hold"
)

// AllFailureClasses is the failure vocabulary, for derived tests.
func AllFailureClasses() []FailureClass {
	return []FailureClass{FailureRetryable, FailureHandoff, FailureUnknown, FailureHold}
}

// ReconcileFailure is one resource that did not converge, with its cause
// verbatim.
type ReconcileFailure struct {
	Resource SlotResourceID
	Class    FailureClass
	Cause    string
}

// weaveSetupActive is weave's own line for a setup already running in the
// environment (ariadne staging/setup.go, ErrSetupInUse).
const weaveSetupActive = "environment setup is active in"

// ClassifyConvergeError classifies a failed converge step. Only a recognized
// transient condition is retryable; everything else is a hand-off, with the
// cause shown verbatim (weave's last "Error:" line when it printed one,
// matched anywhere in a line because weave may prefix the owner).
func ClassifyConvergeError(step PlannedStep, err error) ReconcileFailure {
	f := ReconcileFailure{Resource: step.Resource, Class: FailureHandoff, Cause: errorCause(err)}
	switch {
	case errors.Is(err, errAgentAppeared):
		f.Class, f.Cause = FailureHold, StopReasonAgentLive
	case errors.Is(err, errAgentUnobserved):
		f.Class, f.Cause = FailureHold, StopReasonAgentUnknown
	case errors.Is(err, errCheckoutRecovered):
		f.Class = FailureRetryable // the next run observes the recovered checkout
	case errors.Is(err, errSavedWorkFull):
		f.Class, f.Cause = FailureHold, StopReasonSavedWorkFull
	case errors.Is(err, errSetupRunning), errors.Is(err, ErrHostCreationBusy),
		errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled),
		strings.Contains(err.Error(), weaveSetupActive):
		f.Class = FailureRetryable
	}
	return f
}

// errorCause is the operator-facing cause: weave's last "Error:" line when
// present, else the error text.
func errorCause(err error) string {
	text := err.Error()
	cause := ""
	for _, line := range strings.Split(text, "\n") {
		if i := strings.Index(line, "Error:"); i >= 0 {
			cause = strings.TrimSpace(line[i:])
		}
	}
	if cause != "" {
		return cause
	}
	return strings.TrimSpace(text)
}

// stopFailure reads a plan stop as a failure.
func stopFailure(s PlanStop) ReconcileFailure {
	f := ReconcileFailure{Resource: s.Resource, Cause: s.Reason}
	switch s.Class {
	case StopUnknown:
		f.Class = FailureUnknown
	case StopRetryable:
		f.Class = FailureRetryable
	case StopHold:
		f.Class = FailureHold
	default:
		f.Class = FailureHandoff
	}
	return f
}

// ReconcileAdvice is the one operator text for a resource that did not
// converge: it names the resource and the cause, says to run it again only
// when that can help, and otherwise hands the slot to the repository's :0
// agent. Never "fix it", never a bare "retry".
func ReconcileAdvice(address, repo string, f ReconcileFailure) string {
	switch f.Class {
	case FailureRetryable:
		return fmt.Sprintf("slot %s: %s did not converge: %s; run it again when that finishes", address, f.Resource, f.Cause)
	case FailureUnknown:
		return fmt.Sprintf("slot %s: %s could not be observed: %s; nothing depending on it was changed", address, f.Resource, f.Cause)
	case FailureHold:
		switch f.Cause {
		case StopReasonAgentLive:
			return fmt.Sprintf("slot %s: %s needs repair, but an agent is working in the slot; reboot the slot to repair it", address, f.Resource)
		case StopReasonAgentUnknown:
			// Reboot cannot act on an agent it cannot observe either, so the
			// only action that can succeed is looking again.
			return fmt.Sprintf("slot %s: %s needs repair, but the slot's agent could not be observed; look again later (couch --show %s)", address, f.Resource, address)
		case StopReasonSavedWorkFull:
			return fmt.Sprintf("slot %s: %s needs repair, but its saved work is full; restore or remove old entries (couch --show %s lists the slot)", address, f.Resource, address)
		}
		return fmt.Sprintf("slot %s: %s is on hold: %s", address, f.Resource, f.Cause)
	}
	return fmt.Sprintf("slot %s: %s did not converge: %s. Reconcile cannot fix this; ask the %s:0 agent to investigate (couch --show %s lists every resource)", address, f.Resource, f.Cause, repo, address)
}

// Severity says whether an outcome lets an agent work in the slot.
type Severity string

const (
	// SeverityBlocking: the checkout is not usable; the caller refuses.
	SeverityBlocking Severity = "blocking"
	// SeverityDegraded: the checkout exists and was set up once; the caller
	// proceeds and shows the advice as a warning.
	SeverityDegraded Severity = "degraded"
)

// OutcomeSeverity reads a reconcile's final observation for a caller: it is
// blocking exactly when no agent could work in the slot -- the host checkout
// is not present, or setup never completed (no valid marker; a remembered
// failure under a valid marker, or a compile running under one, still counts
// as set up). Anything else that did not converge (a resting branch checked
// out elsewhere, an upstream conflict, a dependency, a hold) is degraded: the
// caller proceeds and shows the advice. It is a property of the outcome, not
// of the resource that failed (TestReconcileOutcomeTable states it from the
// observation).
func OutcomeSeverity(o SlotObservation) Severity {
	host, _ := o.Get(ResourceHost)
	setup, _ := o.Get(ResourceSetup)
	if host.State != StatePresent {
		return SeverityBlocking
	}
	switch {
	case setup.State == StatePresent:
		return SeverityDegraded
	case setup.Sub == SubMarkerValid || setup.Sub == SubWithWarning:
		return SeverityDegraded
	}
	return SeverityBlocking
}

// SlotReconcileError is a reconcile whose outcome blocks the slot: its text is
// ReconcileAdvice.
type SlotReconcileError struct {
	Address, Repo string
	Failure       ReconcileFailure
	Result        ReconcileResult
	// Err is the underlying cause, when one exists (errors.Is sees it).
	Err error
}

func (e *SlotReconcileError) Error() string { return ReconcileAdvice(e.Address, e.Repo, e.Failure) }
func (e *SlotReconcileError) Unwrap() error { return e.Err }

// SlotOutcome reads a reconcile run for a caller: the blocking failure (nil
// when the slot is usable) and the degraded warnings to show. Every resource
// that did not converge is reported, as the error or as a warning.
func SlotOutcome(address, repo string, result ReconcileResult, runErr error) (*SlotReconcileError, []string) {
	var failures []ReconcileFailure
	var rerr *ReconcileError
	switch {
	case errors.As(runErr, &rerr):
		f := ClassifyConvergeError(rerr.Failure.Step, rerr.Failure.Err)
		if errors.Is(runErr, errNoProgress) {
			f.Cause = rerr.Failure.Step.String() + ": " + errNoProgress.Error()
		}
		failures = append(failures, f)
	case runErr != nil:
		f := ReconcileFailure{Resource: ResourceEnv, Class: FailureHandoff, Cause: errorCause(runErr)}
		if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) || errors.Is(runErr, ErrHostCreationBusy) {
			f.Class = FailureRetryable
		}
		// The run never observed the slot: nothing shows it usable.
		return &SlotReconcileError{Address: address, Repo: repo, Failure: f, Result: result, Err: runErr}, nil
	}
	for _, w := range result.Warnings {
		failures = append(failures, ClassifyConvergeError(w.Step, w.Err))
	}
	for _, s := range result.Plan.Stops {
		failures = append(failures, stopFailure(s))
	}
	if OutcomeSeverity(result.Observation) == SeverityBlocking {
		f := ReconcileFailure{Resource: ResourceHost, Class: FailureHandoff, Cause: "the slot's checkout is not usable"}
		if len(failures) > 0 {
			f = failures[0]
		}
		return &SlotReconcileError{Address: address, Repo: repo, Failure: f, Result: result, Err: runErr}, nil
	}
	warnings := make([]string, 0, len(failures))
	for _, f := range failures {
		warnings = append(warnings, ReconcileAdvice(address, repo, f))
	}
	return nil, warnings
}
