package couchcore

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// PrepareSwitchRecovery commits admission before its receipt is acknowledged.
// The accepted profile is checked before the write; recovery bookkeeping does
// not grant authority to change the accepted launch parameters.
func (c *Couch) PrepareSwitchRecovery(ctx context.Context, request SwitchAgentRequest, recovery SwitchRecovery) (PreparedAgentSwitch, error) {
	ctx, release, err := c.hold(ctx, request.Address, "prepare-switch-recovery")
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	defer release()
	p, err := c.PrepareAgentSwitch(ctx, request.Address, request.Agent, &request.Argv)
	if err != nil {
		return p, err
	}
	if request.AcceptedFingerprint == "" || p.Fingerprint != request.AcceptedFingerprint {
		return p, ErrSwitchResolutionChanged
	}
	if old := p.record.SwitchRecovery; old != nil && old.RequestID == recovery.RequestID {
		return p, nil
	}
	if old := p.record.SwitchRecovery; old != nil && old.Phase != threadrecord.SwitchSucceeded && old.Phase != threadrecord.SwitchRefused {
		return p, errors.New("switch-agent: previous switch still needs recovery")
	}
	recovery.Source.Profile = threadrecord.LaunchProfile{Agent: p.record.LatestLaunchProfile.Agent, Argv: cloneArgv(p.record.LatestLaunchProfile.Argv)}
	recovery.Source.Binding = cloneSessionBinding(p.record.SessionBinding)
	if len(p.record.Incarnations) == 1 {
		inc := p.record.Incarnations[0]
		recovery.Source.Process = threadrecord.SwitchProcess{PID: inc.PID, Identity: inc.Identity}
	}
	if c.SwitchContext != nil {
		evidence, e := c.SwitchContext.Resolve(ctx, p.record)
		if e == nil {
			recovery.Source.NativeSession = evidence.SourceSession
		}
	}
	if recovery.Source.NativeSession == "" {
		if native, e := c.resumeEvidence(ctx, p.record); e == nil {
			recovery.Source.NativeSession = native.NativeID
		}
	}
	recovery.TargetProfile = threadrecord.LaunchProfile{Agent: p.Profile.Agent, Argv: cloneArgv(p.Profile.Argv)}
	recovery.UpdatedAt = c.Clock.Now()
	_, err = c.Threads.updateExistingThread(request.Address, p.record.Revision, func(r *ThreadRecord) error {
		r.SwitchRecovery = threadrecord.CloneSwitchRecovery(&recovery)
		return nil
	})
	if err != nil {
		return p, err
	}
	return c.PrepareAgentSwitch(ctx, request.Address, request.Agent, &request.Argv)
}

func (c *Couch) NoteSwitchState(ctx context.Context, address ThreadAddress, id string, phase AgentSwitchPhase, content SwitchContent, diagnostic string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r, err := c.Threads.GetThread(address)
	if err != nil {
		return err
	}
	_, err = c.Threads.updateExistingThread(address, r.Revision, func(next *ThreadRecord) error {
		saved := next.SwitchRecovery
		if saved == nil || saved.RequestID != id {
			return errors.New("switch recovery request changed")
		}
		if content != (SwitchContent{}) {
			if err := ValidateSwitchContent(content); err != nil {
				return err
			}
			saved.Content = threadrecord.SwitchContent{Kind: threadrecord.SwitchContentKind(content.Kind), Body: content.Body}
		}
		saved.Phase = threadrecord.SwitchPhase(phase)
		saved.Error = boundedSwitchError(diagnostic)
		saved.UpdatedAt = c.Clock.Now()
		return nil
	})
	return err
}
func boundedSwitchError(s string) string {
	if len(s) > 2048 {
		return "switch recovery failed; inspect the operation diagnostics"
	}
	return s
}

func (c *Couch) NoteSwitchDelivery(ctx context.Context, address ThreadAddress, id, attempt string, state orientation.DeliveryState, deliveryErr error) error {
	r, err := c.Threads.GetThread(address)
	if err != nil {
		return err
	}
	if r.SwitchRecovery == nil || r.SwitchRecovery.RequestID != id || r.SwitchRecovery.Target == nil || r.SwitchRecovery.Target.Claim.Nonce != attempt {
		return errors.New("switch delivery does not match retained target")
	}
	phase := AgentSwitchUnconfirmed
	diagnostic := "handoff delivery was not confirmed"
	if state.Phase == orientation.DeliverySubmitted {
		phase = AgentSwitchSucceeded
		diagnostic = ""
	} else if deliveryErr != nil {
		diagnostic = deliveryErr.Error()
	}
	return c.NoteSwitchState(ctx, address, id, phase, SwitchContent{}, diagnostic)
}

// ObserveAgentSwitchEvidence asks about each saved incarnation independently.
// A missing witness proves no launch was begun because witnesses precede effects.
func (c *Couch) ObserveAgentSwitchEvidence(ctx context.Context, address ThreadAddress, saved *SwitchRecovery) AgentSwitchEvidence {
	out := AgentSwitchEvidence{Now: c.Clock.Now()}
	if saved == nil {
		return out
	}
	out.Source = c.observeSwitchIdentity(ctx, address, saved.Source.Process, saved.Source.Binding)
	out.Target = c.observeSwitchWitness(ctx, address, saved.Target)
	out.Restoration = c.observeSwitchWitness(ctx, address, saved.Restoration)
	if out.Source == Unknown || out.Target == Unknown || out.Restoration == Unknown {
		out.Diagnostic = "exact source or target evidence is unavailable; recovery will recheck before resuming"
	}
	return out
}
func (c *Couch) observeSwitchWitness(ctx context.Context, address ThreadAddress, w *threadrecord.SwitchLaunchWitness) Liveness {
	if w == nil {
		return Dead
	}
	process := threadrecord.SwitchProcess{}
	if w.Helper != nil {
		process = threadrecord.SwitchProcess{PID: w.Helper.PID, Identity: w.Helper.Identity}
	}
	if w.Claim.SessionBinding == nil && w.Helper == nil {
		return Dead
	}
	return c.observeSwitchIdentity(ctx, address, process, w.Claim.SessionBinding)
}
func (c *Couch) observeSwitchIdentity(ctx context.Context, address ThreadAddress, p threadrecord.SwitchProcess, binding *couchidentity.SessionBinding) Liveness {
	process := Dead
	if p.PID > 0 {
		process = observeExactProcess(c.Proc, ProcessIdentity{PID: p.PID, Identity: p.Identity})
	}
	if process == Unknown {
		return Unknown
	}
	if binding == nil {
		if process == Live {
			return Live
		}
		return Unknown
	}
	observed, err := observeRecordSession(ctx, c.Artifacts, ThreadRecord{Address: address, SessionBinding: binding})
	if err != nil {
		return Unknown
	}
	if observed.Present {
		return Live
	}
	if process == Live {
		return Unknown
	}
	return Dead
}

func (c *Couch) recordSwitchAttempt(thread ThreadRecord, request SwitchAgentRequest, nonce string, owner ProcessIdentity, native string) (ThreadRecord, error) {
	return c.Threads.updateExistingThread(thread.Address, thread.Revision, func(next *ThreadRecord) error {
		r := next.SwitchRecovery
		if r == nil || r.RequestID != request.RequestID {
			return errors.New("switch-agent: retained admission changed")
		}
		if r.Target != nil {
			return errors.New("switch-agent: target already attempted; recover slot")
		}
		if native != "" {
			if r.Source.NativeSession != "" && r.Source.NativeSession != native {
				return errors.New("switch-agent: source native session changed")
			}
			r.Source.NativeSession = native
		}
		r.Content = threadrecord.SwitchContent{Kind: threadrecord.SwitchContentKind(request.Content.Kind), Body: request.Content.Body}
		r.Phase = threadrecord.SwitchSwitching
		r.UpdatedAt = c.Clock.Now()
		r.Error = ""
		r.Target = &threadrecord.SwitchLaunchWitness{Claim: threadrecord.StartClaim{Nonce: nonce, OwnerPID: owner.PID, OwnerIdentity: owner.Identity, LaunchProfile: &r.TargetProfile}}
		return nil
	})
}

// RecoverAgentSwitch reconciles a saved transaction. It never delivers content,
// and no cold launch is permitted while either saved terminal is unproven.
func (c *Couch) RecoverAgentSwitch(ctx context.Context, address ThreadAddress, id string) (result SwitchAgentResult, err error) {
	result.Outcome = SwitchRefused
	ctx, release, err := c.hold(ctx, address, "recover-agent-switch")
	if err != nil {
		return result, err
	}
	defer release()
	defer func() {
		if err != nil {
			_ = c.NoteSwitchState(context.Background(), address, id, AgentSwitchUnconfirmed, SwitchContent{}, err.Error())
		}
	}()
	r, err := c.Threads.GetThread(address)
	if err != nil {
		return result, err
	}
	s := r.SwitchRecovery
	if s == nil || s.RequestID != id {
		return result, errors.New("switch recovery request changed")
	}
	evidence := c.ObserveAgentSwitchEvidence(ctx, address, s)
	if evidence.Source == Unknown || evidence.Target == Unknown || evidence.Restoration == Unknown {
		return result, errors.New(evidence.Diagnostic)
	}
	live := 0
	for _, state := range []Liveness{evidence.Source, evidence.Target, evidence.Restoration} {
		if state == Live {
			live++
		}
	}
	if live > 1 {
		return result, errors.New("more than one saved switch session is live; inspect before recovery")
	}
	if evidence.Target == Live {
		return c.attachSwitchSurvivor(ctx, r, s.Target, threadrecord.SwitchSurvivorTarget)
	}
	if evidence.Restoration == Live {
		return c.attachSwitchSurvivor(ctx, r, s.Restoration, threadrecord.SwitchSurvivorRestoredSource)
	}
	if evidence.Source == Live {
		return c.attachSwitchSurvivor(ctx, r, nil, threadrecord.SwitchSurvivorSource)
	}
	// All exact identities were proved absent. No park retry may stop a survivor.
	if r.Park != nil {
		if c.PairLifecycle == nil {
			return result, errors.New("park recovery unavailable")
		}
		recovered, recoverErr := c.PairLifecycle.Recover(ctx, address)
		if recoverErr == nil {
			r = recovered.Thread
		} else {
			// Completion can be lost after actual source exit. Only the already proved
			// absence of both source and target authorizes retiring this orphaned park.
			current, e := c.Threads.GetThread(address)
			if e != nil {
				return result, e
			}
			again := c.ObserveAgentSwitchEvidence(ctx, address, current.SwitchRecovery)
			if again.Source != Dead || again.Target != Dead || again.Restoration != Dead {
				return result, recoverErr
			}
			if current.Park != nil {
				current, e = c.Threads.AbandonPark(address, current.Revision, current.Park.Identity)
				if e != nil {
					return result, e
				}
			}
			r = current
		}
	}
	if r.SwitchRecovery.Source.NativeSession == "" {
		return result, errors.New("saved old native session is unavailable; cannot resume a different conversation")
	}
	// Resolve before retiring or changing saved profile authority.
	exact, ok := c.Artifacts.(ExactNativeBindingResolver)
	if !ok {
		return result, errors.New("exact native resolver unavailable")
	}
	if _, err = exact.ResolveExact(ctx, address.RepoScope, string(address.Tag), s.Source.Profile.Agent, s.Source.NativeSession); err != nil {
		return result, err
	}
	if next, e := c.clearSwitchDeadStart(r); e != nil {
		return result, e
	} else {
		r = next
	}
	r, err = c.Threads.updateExistingThread(address, r.Revision, func(next *ThreadRecord) error {
		profile := LaunchProfile{Agent: next.SwitchRecovery.Source.Profile.Agent, Argv: cloneArgv(next.SwitchRecovery.Source.Profile.Argv)}
		next.LatestLaunchProfile = &profile
		next.SessionBinding = cloneSessionBinding(next.SwitchRecovery.Source.Binding)
		next.SwitchRecovery.Survivor = threadrecord.SwitchSurvivorNone
		next.SwitchRecovery.Error = ""
		return nil
	})
	if err != nil {
		return result, err
	}
	actor, handle, err := c.ResumeContextWith(ctx, address, ResumeOptions{NativeSession: s.Source.NativeSession, SwitchRequest: id})
	if err != nil {
		return result, err
	}
	return SwitchAgentResult{Outcome: SwitchStarted, Record: actor, Handle: handle}, nil
}

// clearSwitchDeadStart differs from generic debris only for this exact completed
// operation's claim: its original supervisor may still be alive after failure.
func (c *Couch) clearSwitchDeadStart(r ThreadRecord) (ThreadRecord, error) {
	if len(r.Incarnations) == 1 && r.Incarnations[0].Start != nil {
		inc := r.Incarnations[0]
		s := r.SwitchRecovery
		matches := s.Target != nil && s.Target.Claim.Nonce == inc.Start.Nonce || s.Restoration != nil && s.Restoration.Claim.Nonce == inc.Start.Nonce
		if !matches {
			return r, errors.New("unrelated start is still outstanding")
		}
		e := c.ObserveAgentSwitchEvidence(context.Background(), r.Address, s)
		if e.Target != Dead || e.Restoration != Dead {
			return r, errors.New("target absence no longer proved")
		}
		if err := c.Threads.DeleteStart(r.Address, r.Revision, inc.Start.Nonce); err != nil {
			return r, err
		}
		return c.Threads.GetThread(r.Address)
	}
	next, err := c.clearLifecycleDebris(r)
	if next != nil {
		r = *next
	}
	return r, err
}

func (c *Couch) attachSwitchSurvivor(ctx context.Context, r ThreadRecord, w *threadrecord.SwitchLaunchWitness, survivor threadrecord.SwitchSurvivor) (SwitchAgentResult, error) {
	result := SwitchAgentResult{Outcome: SwitchRefused}
	s := r.SwitchRecovery
	profile := s.Source.Profile
	binding := s.Source.Binding
	if w != nil {
		profile = *w.Claim.LaunchProfile
		binding = w.Claim.SessionBinding
		if survivor == threadrecord.SwitchSurvivorTarget {
			if c.FreshRegistration == nil {
				return result, errors.New("target registration observer unavailable")
			}
			registered, err := c.FreshRegistration(ctx, r.Address, profile.Agent, w.Claim.Nonce)
			if err != nil {
				return result, err
			}
			if !registered {
				return result, errors.New("saved target is alive but its registration is unconfirmed")
			}
		}
	}
	// A currently hosted exact helper is already attached. Returning no handle
	// makes repeated recovery a no-op for the console's child adoption path.
	for _, actor := range c.actorRegistry().Records() {
		if actor.Thread == r.Address && c.Liveness(actor) == Live && len(r.Incarnations) == 1 && actor.PID == r.Incarnations[0].PID && actor.Identity == r.Incarnations[0].Identity {
			err := c.selectSwitchSurvivor(r, survivor)
			return SwitchAgentResult{Outcome: SwitchStarted, Record: actor}, err
		}
	}
	// Do not steal another supervisor's live helper. A surviving detached session
	// with a dead helper can be adopted using existing WarmOnly resume machinery.
	if len(r.Incarnations) == 1 {
		inc := r.Incarnations[0]
		if observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) != Dead {
			return result, errors.New("survivor helper is still owned; cannot attach twice")
		}
		var err error
		if inc.Start != nil {
			if w == nil || inc.Start.Nonce != w.Claim.Nonce {
				return result, errors.New("survivor start identity changed")
			}
			r, err = c.Threads.AdvanceStart(r.Address, r.Revision, StartEvent{Kind: StartRegistered, Nonce: inc.Start.Nonce})
			if err != nil {
				return result, err
			}
			inc = r.Incarnations[0]
		}
		retire := c.Threads.RetireIncarnation
		if inc.State == IncarnationUnknown {
			retire = c.Threads.RetireUnprovenIncarnation
		}
		r, err = retire(r.Address, r.Revision, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}, r.LastActiveAt)
		if err != nil {
			return result, err
		}
	}
	if r.Park != nil {
		return result, errors.New("surviving source has an unfinished park; recovery evidence is not settled")
	}
	var err error
	r, err = c.Threads.updateExistingThread(r.Address, r.Revision, func(next *ThreadRecord) error {
		p := LaunchProfile{Agent: profile.Agent, Argv: cloneArgv(profile.Argv)}
		next.LatestLaunchProfile = &p
		next.SessionBinding = cloneSessionBinding(binding)
		return nil
	})
	if err != nil {
		return result, err
	}
	actor, handle, err := c.ResumeContextWith(ctx, r.Address, ResumeOptions{WarmOnly: true, SwitchRequest: s.RequestID})
	if err != nil {
		return result, err
	}
	current, err := c.Threads.GetThread(r.Address)
	if err == nil {
		err = c.selectSwitchSurvivor(current, survivor)
	}
	return SwitchAgentResult{Outcome: SwitchStarted, Record: actor, Handle: handle}, err
}
func (c *Couch) selectSwitchSurvivor(r ThreadRecord, survivor threadrecord.SwitchSurvivor) error {
	_, err := c.Threads.updateExistingThread(r.Address, r.Revision, func(next *ThreadRecord) error {
		s := next.SwitchRecovery
		s.Survivor = survivor
		s.Error = ""
		s.UpdatedAt = c.Clock.Now()
		if survivor != threadrecord.SwitchSurvivorTarget {
			s.Phase = threadrecord.SwitchRefused
		} else if s.Content.Kind == threadrecord.SwitchContentFresh {
			s.Phase = threadrecord.SwitchSucceeded
		} else if s.Phase != threadrecord.SwitchSucceeded {
			s.Phase = threadrecord.SwitchUnconfirmed
		}
		return nil
	})
	return err
}
