package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// ResumeRoute describes the retained target shape. Warm attachment is observed
// before routing; every native launch uses the shared prepared recovery gate.
type ResumeRoute uint8

const (
	ResumeRouteSlot         ResumeRoute = iota + 1 // slot current, or a positively listed survivor
	ResumeRouteContinuation                        // retained running continuation
	ResumeRouteRecover                             // retained recovery context
	ResumeRouteThread                              // ordinary retained thread
)

// ResumeRouteInput carries retained target shape. Positive listing is observed
// before execution and always takes precedence over continuation bookkeeping.
type ResumeRouteInput struct {
	Slot, HasRecord bool
	Continuation    checkpoint.Phase // "" when the record has none
}

// ChooseResumeRoute preserves the target's slot and continuation distinctions.
// Only a slot without a record can reconstruct a positively listed survivor;
// retained targets keep their existing continuation guards at native admission.
func ChooseResumeRoute(in ResumeRouteInput) ResumeRoute {
	if in.HasRecord {
		switch in.Continuation {
		case checkpoint.Running:
			return ResumeRouteContinuation
		case checkpoint.Pending:
			return ResumeRouteRecover
		}
	}
	if in.Slot {
		return ResumeRouteSlot
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

// ResumeTarget attaches a positively listed terminal or prepares an explicit
// native resume. Consumers read StartedChild across its result wrappers.
func (c *Couch) ResumeTarget(ctx context.Context, target ResumeTarget) (any, error) {
	return c.ResumeTargetWith(ctx, target, ResumeOptions{})
}

// ResumeTargetWith keeps warm-only attachment separate from the prepared native
// executor. A positive listing never asks for workspace repair or a new agent.
func (c *Couch) ResumeTargetWith(ctx context.Context, target ResumeTarget, opts ResumeOptions) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.Threads == nil {
		return nil, errors.New("resume requires a thread store")
	}
	if (target.Path == "") == (target.Address == ThreadAddress{}) {
		return nil, errors.New("resume needs exactly one of a slot path or a thread address")
	}
	result, err := c.resumeRouted(ctx, target, opts)
	if opts.WarmOnly {
		return result, err
	}
	return result, withRebootAdvice(err)
}

func (c *Couch) resumeRouted(ctx context.Context, target ResumeTarget, opts ResumeOptions) (any, error) {
	_, record, err := c.recoveryTarget(ctx, target)
	in := ResumeRouteInput{Slot: target.Path != "", HasRecord: err == nil}
	if err != nil && !errors.Is(err, ErrThreadNotFound) {
		return nil, err
	}
	if in.HasRecord && record.Continuation != nil {
		in.Continuation = record.Continuation.Phase
	}
	route := ChooseResumeRoute(in)
	if route == ResumeRouteSlot && !in.HasRecord {
		if opts.WarmOnly {
			return nil, refuseResume(ResumeNotDetached, "no retained thread is available for automatic attachment")
		}
		return c.resumeListedSlotSurvivor(ctx, target.Path)
	}
	if !in.HasRecord {
		return nil, refuseResume(ResumeNoSurvivor, "no retained conversation is available; choose Start fresh")
	}
	session, listed, listErr := c.listedTerminal(ctx, record)
	if listErr != nil && !errors.Is(listErr, ErrPairSessionBindingAbsent) {
		return nil, fmt.Errorf("%w: %v", refuseResume(ResumeUnknown, "terminal listing is unavailable; retry"), listErr)
	}
	if listed {
		actor, handle, err := c.attachListedTerminal(ctx, record, session, opts)
		if err != nil && handle == nil {
			return nil, err
		}
		return StartResult{Record: actor, Handle: handle}, err
	}
	if opts.WarmOnly {
		return nil, refuseResume(ResumeNotDetached, "the exact terminal is not listed live")
	}
	if record.Park != nil {
		return nil, refuseResume(ResumeParking, "a park transaction is unfinished; "+c.ParkRecoveryAdvice(ctx, record.Address))
	}
	// A retained compaction request owns its separate exact-attempt executor.
	// Resume never converts it into generic native recovery or another stop set.
	switch route {
	case ResumeRouteContinuation:
		return c.Continue(ctx, record.Address, record.Continuation.ID)
	case ResumeRouteRecover:
		return c.RecoverThread(ctx, record.Address, "")
	}
	if startClaimed(record) {
		transaction, err := CurrentStartTransaction(record)
		evidence := ThreadEvidence{}
		if err == nil && c.Proc != nil {
			evidence.StartOwner = observeExactProcess(c.Proc, transaction.Owner)
		}
		if c.Clock != nil {
			evidence.ObservedAt = c.Clock.Now()
		}
		if !DecideStaleStart(record, evidence) {
			return nil, refuseResume(ResumeStarting, "source is still starting; wait for its bounded launch")
		}
	}
	binding, err := c.resumeEvidence(ctx, record)
	if err != nil {
		return nil, err
	}
	if _, err := DecideResume(ResumeEligibilityInput{Thread: record, Binding: binding, WorkingPathExists: c.workingPathExists(record)}); err != nil {
		return nil, err
	}
	// Every actual native launch crosses the shared prepared recovery boundary.
	return c.RecoverSelection(ctx, RecoverySelection{Target: target, Kind: RecoveryProven})
}

// resumeListedSlotSurvivor reconstructs a lost pointer only to attach a proven
// existing terminal. Catalog discovery and observation never run Ensure.
func (c *Couch) resumeListedSlotSurvivor(ctx context.Context, path string) (StartResult, error) {
	target, _, err := c.resolveSlotInput(ctx, path)
	if err != nil {
		return StartResult{}, err
	}
	if target == nil || target.Kind != ThreadTargetSlot {
		return StartResult{}, refuseResume(ResumeNoSurvivor, "no numbered slot conversation is available; choose Start fresh")
	}
	slot := target.Slot
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return StartResult{}, err
	}
	verified := false
	for _, candidate := range repository.Slots {
		if candidate.Identity == slot && candidate.Verified && candidate.Err == nil {
			verified = true
			break
		}
	}
	if !verified {
		return StartResult{}, refuseResume(ResumePathMissing, "slot checkout identity is not verified; inspect it before attachment")
	}
	local := newSlotThreadStore(c.Namespace, slot)
	local.coordinator = c.Threads.coordinator
	observed, err := local.observeSlotCurrentContext(ctx)
	if err != nil {
		return StartResult{}, err
	}
	if observed.Unsupported {
		return StartResult{}, observed.Err
	}
	if observed.Record != nil {
		return StartResult{}, errors.New("slot current changed; retry resume")
	}
	sessions, err := c.ObserveSlotSessions(ctx, slot)
	if err != nil {
		return StartResult{}, fmt.Errorf("%w: %v", refuseResume(ResumeUnknown, "slot ownership is unavailable; retry"), err)
	}
	var survivors []ThreadRecord
	var names []string
	unproven := false
	for _, candidate := range sessions.Candidates {
		if candidate.Presence != SessionPresent {
			continue
		}
		var next ThreadRecord
		if candidate.Record != nil {
			next = cloneThreadRecord(*candidate.Record)
		} else {
			family, err := c.slotFamily(ctx, slot, false)
			if err != nil {
				return StartResult{}, err
			}
			cwd, err := ValidateFamilyPath(slot.WorktreeRoot, family.RelativeStart)
			if err != nil {
				return StartResult{}, err
			}
			profile, err := c.slotLaunchProfile(ctx, local, slot, cwd, "")
			if err != nil {
				return StartResult{}, err
			}
			next = ThreadRecord{SchemaVersion: ThreadSchemaVersion, Address: candidate.Address, StartingPath: cwd, WorkingPath: cwd, CreatedAt: c.Clock.Now(), Revision: 1, LatestLaunchProfile: &profile.Profile}
			binding, err := c.resumeEvidence(ctx, next)
			if err != nil || !coldResumeAuthorized(binding) {
				unproven = true
				continue
			}
		}
		name, listed, err := c.listedTerminal(ctx, next)
		if err != nil {
			return StartResult{}, err
		}
		if !listed {
			return StartResult{}, refuseResume(ResumeUnknown, "survivor listing changed; retry")
		}
		survivors = append(survivors, next)
		names = append(names, name)
	}
	if len(survivors) == 0 {
		if unproven {
			return StartResult{}, refuseResume(ResumeSurvivorUnproven, "a terminal survives but its agent is unproved; inspect it before resuming")
		}
		return StartResult{}, refuseResume(ResumeNoSurvivor, "no positively listed conversation could be proved; choose Start fresh")
	}
	if len(survivors) != 1 || unproven {
		return StartResult{}, refuseResume(ResumeSurvivorsAmbiguous, "several terminal owners survive; inspect them before resuming")
	}
	next, name := survivors[0], names[0]
	var others []ProcessIdentity
	for _, candidate := range sessions.Candidates {
		if candidate.Address == next.Address {
			continue
		}
		if candidate.Presence == SessionPresent {
			return StartResult{}, refuseResume(ResumeSurvivorsAmbiguous, "another terminal owner survives")
		}
		others = append(others, candidate.Processes...)
	}
	adoption, err := c.NewAdoptionAuthority(next.Address, others)
	if err != nil {
		return StartResult{}, refuseResume(ResumeUnknown, "another slot owner is not proved stopped")
	}
	if err := c.confirmListedTerminal(ctx, next, name); err != nil {
		return StartResult{}, err
	}
	if err := local.replaceSlotCurrent(observed, next, adoption); err != nil {
		return StartResult{}, err
	}
	actor, handle, err := c.attachListedTerminal(ctx, next, name, ResumeOptions{})
	return StartResult{Record: actor, Handle: handle}, err
}
