package couchcore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// RelaunchOutcome describes the observed state after a relaunch.
//
// Consumers switch on this rather than parsing an error, because failure
// states differ by which recovery works -- and telling an operator to
// press Enter on a thread whose park transaction is still open is the
// unnavigable-refusal class (pair#181 M3).
type RelaunchOutcome string

const (
	// Relaunched: a new Pair process on the same address, same conversation.
	Relaunched RelaunchOutcome = "relaunched"
	// RefusedBeforePark: admission or park begin refused without changing the
	// thread. The thread is exactly as it was.
	RefusedBeforePark RelaunchOutcome = "refused-before-park"
	// ParkIncomplete: the settled worker left an open transaction. Publication
	// and child termination may still be uncertain; recover-park owns resolution.
	ParkIncomplete RelaunchOutcome = "park-incomplete"
	// ParkInProgress: the caller stopped waiting while owned work is still active.
	ParkInProgress RelaunchOutcome = "park-in-progress"
	// ParkRolledBack: an unpublished park was closed, retaining its original owner.
	ParkRolledBack RelaunchOutcome = "park-rolled-back"
	// ParkStateUnknown: the current state could not be established safely.
	ParkStateUnknown RelaunchOutcome = "park-state-unknown"
	// ParkedNotResumed: the park succeeded and the resume did not. An ordinary
	// parked thread: listed, and Enter resumes it.
	// The VALUE avoids the `parked-` prefix deliberately: it is the artifact
	// vocabulary's token for parked-state filenames, and a constant that starts
	// with it reads to the artifact guard as a filename this file constructs.
	// Renaming the value beats requesting an exemption (pair#181's
	// agent-unsupported precedent).
	ParkedNotResumed RelaunchOutcome = "park-ok-resume-failed"
)

// ClassifyRelaunchParkOutcome uses current durable facts, never the park error.
// Active work takes precedence because its final mutation can follow the snapshot.
func ClassifyRelaunchParkOutcome(original, current ThreadRecord, active bool) RelaunchOutcome {
	if active {
		return ParkInProgress
	}
	if current.Park != nil && !current.Park.Closed {
		return ParkIncomplete
	}
	if current.Park == nil && current.VerifiedPark != nil &&
		!reflect.DeepEqual(original.VerifiedPark, current.VerifiedPark) && !hasOccupiedIncarnation(current) {
		return ParkedNotResumed
	}
	if reflect.DeepEqual(original, current) {
		return RefusedBeforePark
	}
	if current.Park == nil && reflect.DeepEqual(original.Incarnations, current.Incarnations) {
		for _, closed := range current.ParkHistory {
			if closed.Closed && closed.Tombstoned && closed.BaseRevision == original.Revision {
				for _, incarnation := range original.Incarnations {
					if closed.Identity.PID == incarnation.PID && closed.Identity.ProcessIdentity == incarnation.Identity {
						return ParkRolledBack
					}
				}
			}
		}
	}
	return ParkStateUnknown
}

// RelaunchResult reports what happened, including what could not be finished.
//
// Following ArchiveResult (pair#181 M3): an operation that mutated says so in
// its result rather than delivering a partial success on the error channel,
// where every consumer reads it as total failure.
type RelaunchResult struct {
	Outcome RelaunchOutcome
	Record  ActorRecord
	Handle  Handle
}

// ReceiptCode is the outcome a caller's receipt carries when the relaunch did
// not complete (pair#421): settled, active and uncertain states need different
// recovery, so they must not collapse into free text.
func (r RelaunchResult) ReceiptCode() string {
	if r.Outcome == Relaunched {
		return ""
	}
	return string(r.Outcome)
}

// Started reports the resumed child for console adoption. Only a completed
// relaunch has one: every other outcome either never parked or never resumed,
// and handing back a half-finished relaunch as a startable child would attach
// the console to a process that is not there.
func (r RelaunchResult) Started() (StartResult, bool) {
	if r.Outcome != Relaunched {
		return StartResult{}, false
	}
	return StartResult{Record: r.Record, Handle: r.Handle}, true
}

// Relaunch replaces a thread's Pair process with the current binary, keeping the
// agent conversation.
//
// It is park-then-resume, and the ORDER OF THE CHECKS is the design. Park is
// destructive and resume can refuse, so a relaunch that parks and then discovers
// the resume cannot run has traded a working session for a cold one. Every
// refusal a check can see is therefore raised BEFORE the park, and the two that
// cannot be seen early are named in the plan rather than hoped over:
// the native binding is validated against artifacts the agent is still writing,
// so established-before does not imply established-after; and park's own
// soleParkableIncarnation is not one of resume's rules, so relaunch asks it
// here rather than discovering it after the quit intent has gone out.
func (c *Couch) Relaunch(ctx context.Context, address ThreadAddress) (RelaunchResult, error) {
	refused := RelaunchResult{Outcome: RefusedBeforePark}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateThreadAddress(address); err != nil {
		return refused, err
	}
	if c == nil || c.Threads == nil || c.PairLifecycle == nil {
		return refused, errors.New("relaunch requires a thread store and a Pair lifecycle controller")
	}
	if err := ctx.Err(); err != nil {
		return refused, err
	}
	ctx, release, err := c.hold(ctx, address, "relaunch")
	if err != nil {
		return refused, err
	}
	defer release()
	thread, err := c.Threads.GetThread(address)
	if err != nil {
		return refused, err
	}

	sourceThread := thread
	thread, _, err = c.ownedAttachmentFacts(thread)
	if err != nil {
		return refused, err
	}
	if err := continuationGuard(thread); err != nil {
		return refused, err
	}
	// Park's own precondition, asked first because its failure is the one that
	// would otherwise surface after Pair had been told to quit.
	//
	// Its two failures are opposite states and get opposite messages.
	// soleParkableIncarnation fails for NO live incarnation as well as for two,
	// and mapping both to ResumeLive told a parked row "resume-live" -- a code
	// naming the state it is not in, with a message about park's internals and
	// no next action. Parked rows reach this: the switcher's Alt+n relaunches
	// the highlighted row, and highlighted rows are often parked.
	if !hasOccupiedIncarnation(thread) {
		return refused, refuseResume(ResumeNotRunning,
			fmt.Sprintf("%s is not running, so there is no Pair to restart; Enter resumes it", address.Tag))
	}
	if _, err := soleParkableIncarnation(thread); err != nil {
		return refused, refuseResume(ResumeLive, err.Error())
	}

	// Would this thread resume ONCE PARKED? DecideResume answers a different
	// question -- whether THIS thread is resumable now, from a surviving session
	// or a resolvable conversation -- and a relaunch target is live by
	// definition, so relaunch asks the rules a park cannot change instead.
	// Relaunch is always a COLD resume -- it parks first -- so it needs the
	// binding, unlike a warm reattach.
	pathExists := c.workingPathExists(thread)
	binding, bindingErr := c.resumeEvidence(ctx, thread)
	// A resolver IO failure is not a verdict about the binding. ResolveEstablished
	// returns a ZERO resolution on a real error, which bindingResumeDiagnostic
	// reads as `unbound` -- so the operator was told the binding is not
	// established when the truth is that couch could not look. Raise the real
	// cause first; only a typed ResumeRefusal is a verdict.
	if bindingErr != nil && ResumeDiagnosticOf(bindingErr) == "" {
		return refused, fmt.Errorf("relaunch %s: could not resolve its native session binding: %w", address.Tag, bindingErr)
	}
	if err := CheckResumePreconditions(thread, binding, pathExists); err != nil {
		return refused, withRebootHint(err)
	}

	thread, err = c.adoptOwnedAttachment(ctx, sourceThread)
	if err != nil {
		return refused, err
	}
	// Only now is anything destroyed.
	if _, err := c.PairLifecycle.ParkExpected(WithParkCompletionWait(ctx, c.PairLifecycle.relaunchWait()), address, thread.Revision); err != nil {
		// Await may return on caller cancellation while the owned worker is still
		// settling. Read activity both before and after the durable snapshot so a
		// stale snapshot cannot masquerade as the worker's final state.
		active := c.PairLifecycle.parkActive(address)
		var current ThreadRecord
		var observeErr error
		outcome := ParkInProgress
		if !active {
			observationCtx, done := context.WithTimeout(context.WithoutCancel(ctx), relaunchObservationTimeout)
			current, observeErr = c.Threads.recoveryReadThread(observationCtx, address)
			done()
			active = c.PairLifecycle.parkActive(address)
			outcome = ParkStateUnknown
			if observeErr == nil {
				outcome = ClassifyRelaunchParkOutcome(thread, current, active)
			} else if active {
				outcome = ParkInProgress
			}
		}
		description := "the current park state could not be established"
		next := c.ParkRecoveryAdvice(ctx, address)
		switch outcome {
		case RefusedBeforePark:
			description = "the park did not begin; the thread is unchanged"
			next = "retry relaunch after resolving the error"
		case ParkRolledBack:
			description = "the unpublished park was rolled back; the original Pair was retained"
			next = "retry relaunch after resolving the error"
		case ParkInProgress:
			description = "the caller stopped waiting; park work is still active and its outcome is unsettled"
			next = "after the active work settles, " + next
		case ParkIncomplete:
			description = "the park transaction is still open; child shutdown is not confirmed"
		case ParkedNotResumed:
			description = "the park is durably complete; the thread is parked and was not resumed"
			_, action, adviceErr := c.ParkRecoveryNextAction(ctx, current)
			if adviceErr == nil {
				next = action
			} else {
				next = "inspect the retained thread before resuming: " + adviceErr.Error()
			}
		}
		return RelaunchResult{Outcome: outcome}, fmt.Errorf("relaunch %s: %s: %w\n  next: %s", address.Tag, description, errors.Join(err, observeErr), next)
	}

	record, handle, err := c.ResumeContext(ctx, address)
	if err != nil {
		return RelaunchResult{Outcome: ParkedNotResumed}, fmt.Errorf(
			"relaunch %s: it is parked and the restart did not take: %w\n"+
				"  the work is not lost — the thread is listed as parked, and Enter resumes it",
			address.Tag, err)
	}
	return RelaunchResult{Outcome: Relaunched, Record: record, Handle: handle}, nil
}

// RelaunchParkWait bounds relaunch's wait for its agent to quit (pair#618). A
// park's default 15 s stranded the first call under load; past this bound the
// explicit recovery choice still stands.
const RelaunchParkWait = 60 * time.Second

// relaunchObservationTimeout bounds relaunch's post-failure read of the park
// state. It was once the park commit threshold; #827 raised that threshold
// without extending this diagnostic read.
const relaunchObservationTimeout = time.Second

// withRebootHint names reboot on the refusals only a fresh session gets past
// (pair#456): a conversation never bound, or one a pre-hello-v2 wrapper left
// provisional, has nothing a relaunch can resume.
func withRebootHint(err error) error {
	var refusal *ResumeRefusal
	if !errors.As(err, &refusal) || (refusal.Code != ResumeBindingUnbound && refusal.Code != ResumeBindingProvisional) {
		return err
	}
	hinted := *refusal
	hinted.Diagnostic += "; or reboot it into a fresh session, discarding this conversation (Tab → reboot, or couch --reboot <repo:N> --confirm)"
	return &hinted
}
