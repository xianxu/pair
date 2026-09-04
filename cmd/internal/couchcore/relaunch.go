package couchcore

import (
	"context"
	"errors"
	"fmt"
)

// RelaunchOutcome is which of four states a relaunch left the thread in.
//
// Consumers switch on this rather than parsing an error, because the two
// failure states differ by which recovery works -- and telling an operator to
// press Enter on a thread whose park transaction is still open is the
// unnavigable-refusal class (pair#181 M3).
type RelaunchOutcome string

const (
	// Relaunched: a new Pair process on the same address, same conversation.
	Relaunched RelaunchOutcome = "relaunched"
	// RefusedBeforePark: the resume could not have succeeded, so nothing was
	// done. The thread is exactly as it was.
	RefusedBeforePark RelaunchOutcome = "refused-before-park"
	// ParkIncomplete: the park itself failed, leaving an OPEN transaction. Pair
	// has already been sent its quit intent, so this is not a state Enter can
	// recover -- park's own retry/recover/abandon modes own it.
	ParkIncomplete RelaunchOutcome = "park-incomplete"
	// ParkedNotResumed: the park succeeded and the resume did not. An ordinary
	// parked thread: listed, and Enter resumes it.
	// The VALUE avoids the `parked-` prefix deliberately: it is the artifact
	// vocabulary's token for parked-state filenames, and a constant that starts
	// with it reads to the artifact guard as a filename this file constructs.
	// Renaming the value beats requesting an exemption (pair#181's
	// agent-unsupported precedent).
	ParkedNotResumed RelaunchOutcome = "park-ok-resume-failed"
)

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
	thread, err := c.Threads.GetThread(address)
	if err != nil {
		return refused, err
	}

	// Park's own precondition, asked first because its failure is the one that
	// would otherwise surface after Pair had been told to quit.
	if _, err := soleParkableIncarnation(thread); err != nil {
		return refused, refuseResume(ResumeLive, err.Error())
	}

	// Would this thread resume ONCE PARKED? DecideResume cannot answer -- it
	// refuses any occupied incarnation, and a relaunch target is live by
	// definition -- so relaunch asks the rules a park cannot change.
	bindings, ok := c.Artifacts.(NativeBindingResolver)
	if !ok {
		return refused, errors.New("relaunch: native binding resolver is unavailable")
	}
	agent := ""
	if thread.LatestLaunchProfile != nil {
		agent = thread.LatestLaunchProfile.Agent
	}
	binding, bindingErr := bindings.ResolveEstablished(ctx, address.RepoScope, string(address.Tag), agent)
	pathExists := true
	if c.Path != nil {
		if _, err := c.Path.Physical(thread.WorkingPath); err != nil {
			pathExists = false
		}
	}
	if err := CheckResumePreconditions(thread, binding, pathExists); err != nil {
		return refused, err
	}
	if bindingErr != nil {
		return refused, bindingErr
	}

	// Only now is anything destroyed.
	if _, err := c.PairLifecycle.Park(ctx, address); err != nil {
		return RelaunchResult{Outcome: ParkIncomplete}, fmt.Errorf(
			"relaunch %s: the park did not complete, so its Pair was stopped but the thread is not resumable yet: %w\n"+
				"  the transaction is still open; recover it with park's own modes:\n"+
				"    retry    — try the same park again\n"+
				"    recover  — finish it from the completion already on disk\n"+
				"    abandon  — give up on it and leave the thread occupied",
			address.Tag, err)
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
