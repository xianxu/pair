package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// attachListedTerminal owns a client helper, never the terminal's lifecycle.
// In particular, an independently owned start or park transaction is unchanged.
func (c *Couch) attachListedTerminal(ctx context.Context, thread ThreadRecord, session string, opts ResumeOptions) (ActorRecord, Handle, error) {

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
	env := []string{"COUCH_TREE=" + string(args.Worktree), "COUCH_STORE_DIR=" + c.Namespace.Dir(), "COUCH_THREAD_SCOPE=" + thread.Address.RepoScope, "COUCH_THREAD_TAG=" + string(thread.Address.Tag), "COUCH_THREAD_RESUME=1", "PAIR_RETENTION_BACKGROUND=" + background, launcher.CouchSessionIntentEnv + "=" + string(intent)}
	dir := args.WorkingDir()
	if !c.workingPathExists(thread) {
		dir = string(filepath.Separator)
	}
	if err := c.confirmListedTerminal(ctx, thread, session); err != nil {
		return ActorRecord{}, nil, err
	}
	h, err := c.Runner.StartBlocked(ctx, dir, []string{"pair", launcher.CouchSessionFlag, "resume", string(thread.Address.Tag)}, env, 10*time.Second)
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
	if err := c.confirmListedTerminal(ctx, thread, session); err != nil {
		return abort(err, false)
	}
	if err := h.Acknowledge(); err != nil {
		return abort(err, true)
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
	err = c.confirmListedTerminal(attachment, thread, session)
	cancel()
	exited := stopWatch()
	select {
	case <-h.Exited():
		exited = true
	default:
	}
	if exited {
		err = errors.New("Pair attachment helper exited before attachment completed")
	}
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
	return actor, h, nil
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
		return "", false, errors.New("selected session listing unavailable")
	}
	observed, err := observer.ListedPairSessionContext(ctx, thread.Address, recordSessionName(thread))
	if err != nil {
		return "", false, err
	}
	return observed.Name, observed.Present && observed.Name != "", nil
}

func (c *Couch) confirmListedTerminal(ctx context.Context, thread ThreadRecord, session string) error {
	name, present, err := c.listedTerminal(ctx, thread)
	if err != nil {
		return err
	}
	if !present || name != session {
		return refuseResume(ResumeSessionGone, "the same terminal is no longer uniquely listed live")
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
