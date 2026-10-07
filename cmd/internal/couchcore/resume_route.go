package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// ResumeRoute is which existing executor resume hands a row to. Resume is one
// operator action -- "get the old conversation back by whatever path works" --
// over four executors that already know how to do each half; it composes them
// and re-implements none (ARCH-DRY).
type ResumeRoute uint8

const (
	ResumeRouteSlot         ResumeRoute = iota + 1 // OpenSlot: warm, cold, or adopt a still-running agent
	ResumeRouteContinuation                        // RetryContinuation(request.ID)
	ResumeRouteRecover                             // RecoverThread(address): survivor reattach or retained checkpoint
	ResumeRouteThread                              // ResumeContextWith: warm, then cold
)

// ResumeRouteInput is what ChooseResumeRoute reads. RecoveryRecover comes from
// FRESH evidence gathered by the shell (DecideRecovery over observeRecovery),
// never from a possibly stale row projection.
type ResumeRouteInput struct {
	Slot, HasRecord bool
	Continuation    checkpoint.Phase // "" when the record has none
	RecoveryRecover bool             // unusable ordinary record with a surviving session (DecideRecovery.Recover)
}

// ChooseResumeRoute is resume's routing as a pure decision.
//
// An in-flight continuation outranks everything: a failed or running request
// is retried where it stands (RetryContinuation), and a pending one belongs to
// RecoverThread, which owns retained checkpoints. Without one, a slot goes
// through OpenSlot -- which also adopts a running agent whose pointer couch
// lost -- and an ordinary record through RecoverThread when fresh evidence
// says a session survived an unusable record, else through ResumeContextWith.
func ChooseResumeRoute(in ResumeRouteInput) ResumeRoute {
	if in.HasRecord {
		switch in.Continuation {
		case checkpoint.Failed, checkpoint.Running:
			return ResumeRouteContinuation
		case checkpoint.Pending:
			return ResumeRouteRecover
		}
	}
	switch {
	case in.Slot:
		return ResumeRouteSlot
	case in.HasRecord && in.RecoveryRecover:
		return ResumeRouteRecover
	}
	return ResumeRouteThread
}

// ResumeRebootAdvice says, for every resume refusal, whether the conversation
// cannot come back -- so the operator is pointed at reboot -- or whether the
// refusal is transient and a retry is the answer. Every declared
// ResumeDiagnosticCode is a key; TestResumeRebootAdviceClassifiesEveryDeclaredCode
// derives the declared set from resume.go, so a new code cannot ship
// unclassified.
var ResumeRebootAdvice = map[ResumeDiagnosticCode]bool{
	ResumePathMissing:        true,
	ResumeProfileMissing:     true,
	ResumeProfileInvalid:     true,
	ResumeAgentUnsupported:   true,
	ResumeBindingUnbound:     true,
	ResumeBindingRootMissing: true,
	ResumeBindingAmbiguous:   true,
	ResumeTombstoned:         true,
	ResumeSessionGone:        true,
	ResumeNoSurvivor:         true,
	ResumeLive:               false,
	ResumeUnknown:            false,
	ResumeParking:            false,
	ResumeStarting:           false,
	ResumeBindingProvisional: false,
	ResumeNotDetached:        false,
	ResumeNotRunning:         false,
	ResumeSurvivorsAmbiguous: false,
	ResumeSurvivorUnproven:   false,
	ResumeOrphanedServer:     false,
}

// withRebootAdvice keeps errors.As working (%w) and appends the exit once, at
// the top of resume, so no executor below has to know about reboot.
func withRebootAdvice(err error) error {
	if err == nil || !ResumeRebootAdvice[ResumeDiagnosticOf(err)] {
		return err
	}
	return fmt.Errorf("%w; this conversation cannot come back -- Tab → reboot archives it and starts a fresh agent here", err)
}

// ResumeTarget names what resume acts on: a slot by its host checkout (:1+),
// or an ordinary thread by its address (:0). Exactly one is set.
type ResumeTarget struct {
	Path    string        // slot host checkout (:1+)
	Address ThreadAddress // ordinary (:0)
}

// ResumeTarget is the resume operation: read the target, gather the one piece
// of fresh evidence routing needs, hand the row to the executor
// ChooseResumeRoute names, and -- only at this top -- say whether a refusal
// means the conversation cannot come back, so the operator is pointed at
// reboot.
//
// The result is whatever the executor returns (StartResult from OpenSlot and
// ResumeContextWith, ContinuationResult from RetryContinuation and
// RecoverThread); every consumer of a ResultStart operation already reads both.
func (c *Couch) ResumeTarget(ctx context.Context, t ResumeTarget) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.Threads == nil {
		return nil, errors.New("resume requires a thread store")
	}
	if (t.Path == "") == (t.Address == ThreadAddress{}) {
		return nil, errors.New("resume needs exactly one of a slot path or a thread address")
	}
	result, err := c.resumeRouted(ctx, t)
	return result, withRebootAdvice(err)
}

func (c *Couch) resumeRouted(ctx context.Context, t ResumeTarget) (any, error) {
	in := ResumeRouteInput{Slot: t.Path != ""}
	var record ThreadRecord
	var local *ThreadStore
	var slot SlotIdentity
	if in.Slot {
		var err error
		local, slot, err = c.selectedSlot(ctx, t.Path)
		if err != nil {
			return nil, err
		}
		observed, err := local.observeSlotCurrent()
		if err != nil {
			return nil, err
		}
		if observed.Unsupported {
			return nil, observed.Err
		}
		if observed.Record != nil {
			in.HasRecord, record = true, *observed.Record
		}
	} else {
		var err error
		record, err = c.Threads.GetThread(t.Address)
		if err != nil {
			return nil, err
		}
		in.HasRecord = true
	}
	if in.HasRecord && record.Continuation != nil {
		in.Continuation = record.Continuation.Phase
	}
	// A surviving session behind an unusable ordinary record is RecoverThread's
	// to reattach. Asked from FRESH evidence: the row the operator pressed may
	// have been drawn before the agent came back, or after it left.
	if !in.Slot && (record.Continuation == nil || record.Continuation.Phase == checkpoint.Complete) {
		state, _, err := c.classifyForAction(ctx, record.Address)
		if err != nil {
			return nil, err
		}
		if state == ThreadUnusable {
			evidence, err := c.observeRecovery(ctx, record)
			if err != nil {
				return nil, err
			}
			in.RecoveryRecover = DecideRecovery(evidence).Recover
		}
	}
	switch ChooseResumeRoute(in) {
	case ResumeRouteSlot:
		agent := ""
		if !in.HasRecord {
			// Adopting a record-less survivor needs an agent to prove it
			// against; the operator only pressed resume, so it is GUESSED from
			// the slot's own launch profile (path preference, else the
			// repository default, else the root agent), and openSlot adopts on
			// a guess only where the proof checks the agent.
			family, err := c.slotFamily(ctx, slot, false)
			if err != nil {
				return nil, err
			}
			cwd, err := ValidateFamilyPath(slot.WorktreeRoot, family.RelativeStart)
			if err != nil {
				return nil, err
			}
			profile, err := c.slotLaunchProfile(local, slot, cwd, "")
			if err != nil {
				return nil, err
			}
			agent = profile.Profile.Agent
		}
		return c.openSlot(ctx, t.Path, agent, true)
	case ResumeRouteContinuation:
		return c.RetryContinuation(ctx, record.Address, record.Continuation.ID)
	case ResumeRouteRecover:
		return c.RecoverThread(ctx, record.Address, "")
	default:
		actor, handle, err := c.ResumeContextWith(ctx, record.Address, ResumeOptions{})
		if err != nil {
			return nil, err
		}
		return StartResult{Record: actor, Handle: handle}, nil
	}
}
