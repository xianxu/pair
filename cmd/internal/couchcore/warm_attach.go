package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// terminalObservationRefusal prevents a failed authority read from being
// retried by the independent helper-start retry policy.
type terminalObservationRefusal struct{ cause error }

func (e *terminalObservationRefusal) Error() string { return e.cause.Error() }
func (e *terminalObservationRefusal) Unwrap() error { return e.cause }

// attachListedTerminal owns a client helper, never the terminal's lifecycle.
// In particular, an independently owned start or park transaction is unchanged.
func (c *Couch) attachListedTerminal(ctx context.Context, thread ThreadRecord, session string, opts ResumeOptions) (ActorRecord, Handle, error) {
	queryCtx, cancel := zellijreply.AdmissionContext(ctx)
	defer cancel()
	return c.attachListedTerminalAdmitted(ctx, queryCtx, thread, session, opts)
}

func (c *Couch) attachListedTerminalAdmitted(ctx, queryCtx context.Context, thread ThreadRecord, session string, opts ResumeOptions) (result ActorRecord, handle Handle, resultErr error) {
	// Timing is a context-only observation, not retained lifecycle state. Avoid
	// timing allocations on ordinary Continue, and never report a cold claim.
	var steps *launchSteps
	if observe, _ := ctx.Value(launchTimingObserverKey{}).(launchTimingObserver); observe != nil {
		steps = newLaunchSteps(ctx, time.Time{})
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return result, handle, errors.Join(resultErr, err)
		}
		result, handle, resultErr = c.attachListedTerminalOnce(ctx, queryCtx, thread, session, opts, steps)
		if resultErr == nil {
			return result, handle, nil
		}
		// A committed client or an unquiesced helper cannot be repeated without
		// duplicating effects. Cancellation likewise owns cleanup, never retry.
		var observationRefusal *terminalObservationRefusal
		if errors.As(resultErr, &observationRefusal) || result.ID != "" || (handle != nil && handle.Alive()) || ctx.Err() != nil || queryCtx.Err() != nil {
			return result, handle, errors.Join(resultErr, ctx.Err())
		}
		if attempt == 2 {
			return result, handle, fmt.Errorf("attach to terminal %q failed after 2 attempts: %w", session, resultErr)
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			steps.mark("attach-retry-backoff")
			return result, handle, errors.Join(resultErr, ctx.Err())
		case <-timer.C:
		}
		steps.mark("attach-retry-backoff")
	}
	panic("unreachable attachment attempt")
}

func (c *Couch) attachListedTerminalOnce(ctx, queryCtx context.Context, thread ThreadRecord, session string, opts ResumeOptions, steps *launchSteps) (result ActorRecord, handle Handle, resultErr error) {

	args := StartArgs{Worktree: Worktree(thread.StartingPath), Cwd: thread.WorkingPath}
	if profile := thread.LatestLaunchProfile; profile != nil {
		args.Stack, args.ExtraArgs = profile.Agent, cloneArgv(profile.Argv)
	}
	if err := c.normalizeTrackedLaunchArgs(&args, thread.Address, true); err != nil {
		return ActorRecord{}, nil, err
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	intent, err := json.Marshal(launcher.CouchSessionIntent{Scope: thread.Address.RepoScope, Tag: string(thread.Address.Tag), Name: session, Nonce: nonce, Disposition: "attach"})
	if err != nil {
		return ActorRecord{}, nil, err
	}
	background := ""
	if opts.WarmOnly {
		background = "1"
	}
	lease, err := c.lifetimeLeaseEnv(thread)
	if err != nil {
		return ActorRecord{}, nil, fmt.Errorf("prepare lifetime lease: %w", err)
	}
	env := []string{"COUCH_TREE=" + string(args.Worktree), "COUCH_STORE_DIR=" + c.Namespace.Dir(), "COUCH_THREAD_SCOPE=" + thread.Address.RepoScope, "COUCH_THREAD_TAG=" + string(thread.Address.Tag), "COUCH_THREAD_RESUME=1", "PAIR_RETENTION_BACKGROUND=" + background, launcher.CouchSessionIntentEnv + "=" + string(intent), lease}
	if c.SlotEnvironment != nil {
		env = append(env, c.SlotEnvironment(ctx, thread)...)
	}
	dir := args.WorkingDir()
	if !c.workingPathExists(thread) {
		dir = string(filepath.Separator)
	}
	steps.mark("attach-prepare")
	if err := c.confirmListedTerminal(queryCtx, thread, session); err != nil {
		return ActorRecord{}, nil, err
	}
	finishRetention, err := c.beginResumeRetention(ctx, thread.Address, !opts.WarmOnly)
	steps.mark("attach-admission")
	if err != nil {
		return ActorRecord{}, nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, finishRetention(resultErr == nil))
		steps.mark("attach-cleanup+release")
	}()
	if err := queryCtx.Err(); err != nil {
		return ActorRecord{}, nil, &terminalObservationRefusal{err}
	}
	h, err := c.Runner.StartBlocked(ctx, dir, []string{"pair", launcher.CouchSessionFlag, "resume", string(thread.Address.Tag)}, env, 10*time.Second)
	steps.mark("attach-spawn")
	if err != nil {
		return ActorRecord{}, nil, err
	}
	// Until acknowledgement Cancel owns only the blocked helper. Afterwards use
	// the existing warm cleanup, which cannot terminate the borrowed session.
	abort := func(cause error, acknowledged bool) (ActorRecord, Handle, error) {
		var cleanup error
		if acknowledged {
			cleanup = c.quiescePostAckStart(thread.Address, h, StartWarmReattach)
		} else {
			cleanup = h.Cancel()
			if cleanup == nil {
				_ = h.Wait()
			}
		}
		return ActorRecord{}, h, errors.Join(cause, cleanup)
	}
	if err := ctx.Err(); err != nil {
		return abort(err, false)
	}
	if err := c.confirmListedTerminal(queryCtx, thread, session); err != nil {
		return abort(err, false)
	}
	ackErr := h.Acknowledge()
	steps.mark("attach-ack")
	if ackErr != nil {
		return abort(ackErr, true)
	}
	budget := pairRegistrationTimeout
	if c.resumeRegistrationTimeout > 0 {
		budget = c.resumeRegistrationTimeout
	}
	deadline := c.registrationDeadline
	if deadline == nil {
		deadline = context.WithTimeout
	}
	attachment, stopWatch := whileHelperAlive(ctx, h)
	attachment, cancel := deadline(attachment, budget)
	// A public Continue borrows the listed terminal; lifecycle registration
	// and wrapper ownership are neither prerequisites nor effects of attachment.
	queryDeadline, _ := queryCtx.Deadline()
	confirmation, stopConfirmation := context.WithDeadline(attachment, queryDeadline)
	err = c.confirmListedTerminal(confirmation, thread, session)
	stopConfirmation()
	cancel()
	exited := stopWatch()
	select {
	case <-h.Exited():
		exited = true
	default:
	}
	if exited {
		// Helper exit adds evidence; it must not erase a failed authority read
		// and let the independent helper-start retry admit a later observation.
		err = errors.Join(err, errors.New("Pair attachment helper exited before attachment completed"))
	}
	steps.mark("attach-live-check")
	if err != nil {
		return abort(fmt.Errorf("await Pair attachment %+v: %w", thread.Address, err), true)
	}
	if err := ctx.Err(); err != nil {
		return abort(err, true)
	}
	actor := ActorRecord{ID: c.IDs.NewID(), Thread: thread.Address, Args: args, StartedAt: c.Clock.Now(), PID: h.PID(), Identity: h.Identity(), Shape: StartWarmReattach}
	if err := c.mutateRegistry(func(reg Registry) (Registry, error) {
		pruned, _ := c.withoutDead(reg)
		return pruned.Insert(actor), nil
	}); err != nil {
		return abort(err, true)
	}
	c.regMu.Lock()
	if c.warmClients == nil {
		c.warmClients = make(map[ActorID]Handle)
	}
	c.warmClients[actor.ID] = h
	c.regMu.Unlock()
	actor.AttachmentNotice = c.attachmentClientNotice(ctx, thread, actor, h)
	steps.mark("attach-registry+repair")
	return actor, h, nil
}

// hasLocalWarmClient screens whether a borrowed-client probe could be needed.
// It grants no signal authority: detachBorrowedHelpers rechecks the local
// handle and retained incarnation before acting.
func (c *Couch) hasLocalWarmClient(address ThreadAddress) bool {
	for _, actor := range c.actorRegistry().Records() {
		if actor.Thread != address || actor.Shape != StartWarmReattach {
			continue
		}
		c.regMu.Lock()
		handle := c.warmClients[actor.ID]
		c.regMu.Unlock()
		if handle != nil && handle.PID() == actor.PID && handle.Identity() == actor.Identity {
			return true
		}
	}
	return false
}

// detachBorrowedHelpers ends only clients this Couch attached without adopting
// the durable incarnation. The local Handle plus matching registry PID/start
// proves ownership; a restored registry entry alone cannot authorize a signal.
func (c *Couch) detachBorrowedHelpers(ctx context.Context, record ThreadRecord) (bool, error) {
	handled := false
	for _, actor := range c.actorRegistry().Records() {
		if actor.Thread != record.Address || actor.Shape != StartWarmReattach {
			continue
		}
		c.regMu.Lock()
		handle := c.warmClients[actor.ID]
		c.regMu.Unlock()
		if handle == nil || handle.PID() != actor.PID || handle.Identity() != actor.Identity {
			continue
		}
		owns := false
		for _, inc := range record.Incarnations {
			if inc.PID == actor.PID && inc.Identity == actor.Identity {
				owns = true
			}
		}
		if owns {
			continue
		}
		handled = true
		if err := ctx.Err(); err != nil {
			return true, err
		}
		identity := ProcessIdentity{PID: actor.PID, Identity: actor.Identity}
		switch observeExactProcess(c.Proc, identity) {
		case Unknown:
			return true, errors.New("attached helper identity is unresolved")
		case Live:
			if err := c.Proc.SignalGroup(identity.PID, TermSignal); err != nil {
				return true, err
			}
			if err := c.awaitExactProcessExit(ctx, identity); err != nil {
				return true, err
			}
		}
		if err := c.Forget(actor.Args.Worktree, actor.ID); err != nil {
			return true, err
		}
	}
	return handled, nil
}

func (c *Couch) listedTerminal(ctx context.Context, thread ThreadRecord) (string, bool, error) {
	observer, ok := c.Artifacts.(interface {
		ListedPairSessionContext(context.Context, ThreadAddress, string) (PairSessionBinding, error)
	})
	if !ok {
		return "", false, &terminalObservationRefusal{errors.New("selected session listing unavailable")}
	}
	// Each failed observation is final. A later valid reply cannot authorize
	// this request after its first evidence was unknown.
	if err := ctx.Err(); err != nil {
		return "", false, &terminalObservationRefusal{err}
	}
	observed, err := observer.ListedPairSessionContext(ctx, thread.Address, recordSessionName(thread))
	if err != nil {
		return "", false, &terminalObservationRefusal{errors.Join(err, ctx.Err())}
	}
	if err := ctx.Err(); err != nil {
		return "", false, &terminalObservationRefusal{err}
	}
	return observed.Name, observed.Present && observed.Name != "", nil
}

func (c *Couch) confirmListedTerminal(ctx context.Context, thread ThreadRecord, session string) error {
	name, present, err := c.listedTerminal(ctx, thread)
	if err != nil {
		return err
	}
	if !present || name != session {
		return &terminalObservationRefusal{refuseResume(ResumeSessionGone, "the same terminal is no longer uniquely listed live")}
	}
	return nil
}

// ownedAttachmentFacts is preparation only. It projects the locally owned
// client's identity after applying the same authority screen as retirement.
func (c *Couch) ownedAttachmentFacts(record ThreadRecord) (ThreadRecord, Handle, error) {
	var found ActorRecord
	var handle Handle
	for _, actor := range c.actorRegistry().Records() {
		if actor.Thread != record.Address || actor.Shape != StartWarmReattach {
			continue
		}
		c.regMu.Lock()
		candidate := c.warmClients[actor.ID]
		c.regMu.Unlock()
		if candidate == nil || candidate.PID() != actor.PID || candidate.Identity() != actor.Identity || !candidate.Alive() {
			continue
		}
		if handle != nil {
			return record, nil, errors.New("several locally attached helpers require explicit resolution")
		}
		found, handle = actor, candidate
	}
	if handle == nil {
		return record, nil, nil
	}
	for _, inc := range record.Incarnations {
		if inc.PID == found.PID && inc.Identity == found.Identity {
			return record, nil, nil
		}
	}
	if observeExactProcess(c.Proc, ProcessIdentity{PID: found.PID, Identity: found.Identity}) != Live {
		return record, nil, errors.New("attached helper process identity is unresolved")
	}
	if _, err := c.inspectLifecycleDebris(record); err != nil {
		return record, nil, err
	}
	effective := cloneThreadRecord(record)
	effective.Park = nil
	effective.Incarnations = []ThreadIncarnation{{PID: found.PID, Identity: found.Identity, State: IncarnationLive, StartedAt: found.StartedAt, LaunchProfile: record.LatestLaunchProfile}}
	return effective, handle, nil
}

// adoptOwnedAttachment is called only by an explicitly requested lifecycle
// operation under its existing hold. It adopts an existing helper through the
// existing bookkeeping transitions; no child or session is created here.
func (c *Couch) adoptOwnedAttachment(ctx context.Context, record ThreadRecord) (ThreadRecord, error) {
	effective, handle, err := c.ownedAttachmentFacts(record)
	if err != nil || handle == nil {
		return record, err
	}
	observed, err := observeRecordSession(ctx, c.Artifacts, record)
	if err != nil {
		return record, err
	}
	session := observed.Name
	if !observed.Present || session == "" {
		return record, errors.New("attached terminal ownership is unavailable")
	}
	if err := c.confirmListedTerminal(ctx, record, session); err != nil {
		return record, err
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return record, err
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return record, err
	}
	if cleared, err := c.clearLifecycleDebris(record); err != nil {
		return record, err
	} else if cleared != nil {
		record = *cleared
	}
	currentSession, err := observeRecordSession(ctx, c.Artifacts, record)
	if err != nil {
		return record, err
	}
	if !currentSession.Present || currentSession.Name != session {
		return record, errors.New("attached terminal changed before adoption")
	}
	if err := c.confirmListedTerminal(ctx, record, session); err != nil {
		return record, err
	}
	if !handle.Alive() || observeExactProcess(c.Proc, ProcessIdentity{PID: handle.PID(), Identity: handle.Identity()}) != Live {
		return record, errors.New("attached helper exited before adoption")
	}
	binding, err := c.sessionBindingForLaunch(ctx, record, nonce, true)
	if err != nil {
		return record, err
	}
	claimed, err := c.Threads.CommitStartClaim(record.Address, record.Revision, "", effective.Incarnations[0].StartedAt, StartEvent{Kind: StartClaimed, Shape: StartWarmReattach, Nonce: nonce, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: record.LatestLaunchProfile})
	if err != nil {
		return record, err
	}
	advance := func(event StartEvent) error {
		next, err := c.Threads.AdvanceStart(claimed.Address, claimed.Revision, event)
		if err == nil {
			claimed = next
		}
		return err
	}
	for _, event := range []StartEvent{
		{Kind: StartSessionBound, Nonce: nonce, Binding: &binding},
		{Kind: StartHelperRecorded, Nonce: nonce, Helper: ProcessIdentity{PID: handle.PID(), Identity: handle.Identity()}},
		{Kind: StartRegistered, Shape: StartWarmReattach, Nonce: nonce},
	} {
		if err := advance(event); err != nil {
			// This nonce belongs solely to this bookkeeping attempt; the already
			// attached helper remains alive even when the record commit fails.
			return claimed, errors.Join(err, c.rollbackTrackedStart(claimed, nonce))
		}
	}
	c.regMu.Lock()
	for id, local := range c.warmClients {
		if local == handle {
			delete(c.warmClients, id)
		}
	}
	c.regMu.Unlock()
	return claimed, nil
}
