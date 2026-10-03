package couchcore

import (
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
}

// withRebootAdvice keeps errors.As working (%w) and appends the exit once, at
// the top of resume, so no executor below has to know about reboot.
func withRebootAdvice(err error) error {
	if err == nil || !ResumeRebootAdvice[ResumeDiagnosticOf(err)] {
		return err
	}
	return fmt.Errorf("%w; this conversation cannot come back -- Tab → reboot archives it and starts a fresh agent here", err)
}
