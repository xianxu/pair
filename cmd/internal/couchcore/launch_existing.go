package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type trackedThreadLaunch struct {
	Context        context.Context
	Thread         ThreadRecord
	Nonce          string
	Args           StartArgs
	StartedAt      time.Time
	ProfileRaw     string
	UseRepoDefault bool
	Resume         bool
	// Warm marks a REATTACH: the agent is alive behind a client-less zellij
	// session and Pair only has to attach to it.
	Warm bool
}

// launchTrackedThread is the single post-claim launch path for both a newly
// allocated thread and an exact verified-park resume.
func (c *Couch) launchTrackedThread(in trackedThreadLaunch) (ActorRecord, Handle, error) {
	ctx := in.Context
	if ctx == nil {
		ctx = context.Background()
	}
	thread := in.Thread
	// A warm reattach sends NEITHER a layout flag NOR a trusted resume profile,
	// and both omissions are the fix rather than an oversight (#179).
	//
	// The profile carries ResumeRequired, which Pair honours only at a CREATE
	// boundary (`createflow.go:238`) -- it exists to force `--resume <native
	// id>` and skip a config picker, the two things a cold resume needs. A live
	// session decides ATTACH, so sending that authority made Pair refuse the
	// one path that would have worked. Without it, couch's child does exactly
	// what the operator's own `pair resume <tag>` does, which is the behaviour
	// they already rely on.
	//
	// The layout flag is dropped for the same reason: a running session already
	// has its layout, and asking for a different one sends Pair down a conflict
	// path that offers to DELETE the live session -- destroying the agent this
	// exists to preserve. Since #198 couch has a layout of its own, so this
	// omission is load-bearing rather than incidental: `c.Layout` must NOT
	// reach a warm argv.
	argv := []string{"pair", "resume", string(thread.Address.Tag), c.Layout.Flag()}
	if in.Warm {
		argv = []string{"pair", "resume", string(thread.Address.Tag)}
	}
	env := []string{
		"COUCH_TREE=" + string(in.Args.Worktree),
		"COUCH_STORE_DIR=" + c.Namespace.Dir(),
		"COUCH_THREAD_SCOPE=" + thread.Address.RepoScope,
		"COUCH_THREAD_TAG=" + string(thread.Address.Tag),
	}
	if in.Resume {
		env = append(env, "COUCH_THREAD_RESUME=1")
	}
	if !in.Warm {
		env = append(env,
			launcher.CouchLaunchProfileEnv+"="+strings.TrimSpace(in.ProfileRaw),
			"PAIR_USE_REPO_DEFAULT=",
		)
	}
	if in.UseRepoDefault && !in.Warm {
		env[len(env)-1] = "PAIR_USE_REPO_DEFAULT=1"
	}
	shape := StartSpawn
	switch {
	case in.Resume && in.Warm:
		shape = StartWarmReattach
	case in.Resume:
		shape = StartColdResume
	}
	h, err := c.Runner.StartBlocked(ctx, in.Args.WorkingDir(), argv, env, 10*time.Second)
	if err != nil {
		return ActorRecord{}, nil, errors.Join(
			fmt.Errorf("spawn %s: %w", in.Args.Worktree, err),
			c.rollbackTrackedStart(thread, in.Nonce),
		)
	}
	if err := ctx.Err(); err != nil {
		return ActorRecord{}, h, c.failTrackedPreAckStart(thread, in.Nonce, h, err)
	}
	recorded, err := c.Threads.AdvanceStart(thread.Address, thread.Revision, StartEvent{
		Kind:   StartHelperRecorded,
		Nonce:  in.Nonce,
		Helper: ProcessIdentity{PID: h.PID(), Identity: h.Identity()},
	})
	if err != nil {
		cancelErr := h.Cancel()
		if cancelErr == nil {
			_ = h.Wait()
		}
		var rollbackErr error
		if cancelErr == nil && !h.Alive() {
			rollbackErr = c.rollbackTrackedStart(thread, in.Nonce)
		}
		return ActorRecord{}, h, errors.Join(fmt.Errorf("record blocked helper %+v: %w", thread.Address, err), cancelErr, rollbackErr)
	}
	thread = recorded
	if err := ctx.Err(); err != nil {
		return ActorRecord{}, h, c.failTrackedPreAckStart(thread, in.Nonce, h, err)
	}
	if err := h.Acknowledge(); err != nil {
		cause := fmt.Errorf("acknowledge blocked helper %+v: %w", thread.Address, err)
		return ActorRecord{}, h, c.failTrackedPostAckStart(shape, thread, in.Nonce, h, cause)
	}
	if err := ctx.Err(); err != nil {
		return ActorRecord{}, h, c.failTrackedPostAckStart(shape, thread, in.Nonce, h, err)
	}
	registrationTimeout := pairRegistrationTimeout
	if in.Resume && c.resumeRegistrationTimeout > 0 {
		registrationTimeout = c.resumeRegistrationTimeout
	}
	registrationContext, cancelRegistration := context.WithTimeout(ctx, registrationTimeout)
	if in.Resume {
		err = c.awaitResumeRegistration(registrationContext, thread.Address)
	} else {
		err = c.awaitThreadRegistration(registrationContext, thread.Address)
	}
	cancelRegistration()
	if err != nil {
		cause := fmt.Errorf("await Pair registration %+v: %w%s", thread.Address, err,
			c.diagnoseRegistrationFailure(err, thread.Address, registrationTimeout))
		return ActorRecord{}, h, c.failTrackedPostAckStart(shape, thread, in.Nonce, h, cause)
	}
	// The layout witness rides this transaction, and only at a cold boundary:
	// `in.Warm` chose no layout (see the argv above), so it records none -- nil,
	// not an empty Layout -- and the existing witness survives.
	var registeredLayout *Layout
	if !in.Warm {
		chosen := c.Layout
		registeredLayout = &chosen
	}
	registeredThread, err := c.Threads.AdvanceStart(thread.Address, thread.Revision, StartEvent{
		Kind: StartRegistered, Nonce: in.Nonce, Layout: registeredLayout,
	})
	if err != nil {
		cause := fmt.Errorf("promote registered thread %+v: %w", thread.Address, err)
		return ActorRecord{}, h, c.failTrackedPostAckStart(shape, thread, in.Nonce, h, cause)
	}
	thread = registeredThread

	record := ActorRecord{
		ID: c.IDs.NewID(), Thread: thread.Address, Args: in.Args,
		StartedAt: in.StartedAt, PID: h.PID(), Identity: h.Identity(),
		Shape: shape,
	}
	c.reg = c.reg.Insert(record)
	if err := c.Store.Save(c.reg, c.names); err != nil {
		c.reg = c.reg.RemoveActor(in.Args.Worktree, record.ID)
		return record, h, c.failPostAckStart(thread.Address, h, shape, fmt.Errorf("persist registry: %w", err))
	}
	return record, h, nil
}

func (c *Couch) failTrackedPreAckStart(thread ThreadRecord, nonce string, h BlockedHandle, cause error) error {
	cancelErr := h.Cancel()
	if cancelErr == nil {
		_ = h.Wait()
	}
	var rollbackErr error
	if cancelErr == nil && !h.Alive() {
		rollbackErr = c.rollbackTrackedStart(thread, nonce)
	}
	return errors.Join(cause, cancelErr, rollbackErr)
}

// failTrackedPostAckStart owns routes 1-4: a start that failed after its helper
// was acknowledged, while the record still holds a start CLAIM.
//
// What it may destroy depends on the start's shape, which is why the shape --
// not a warm boolean -- travels here (pair#230): a spawn takes the reconcile
// tail, a cold resume may end the session it created, and a warm reattach must
// end only its helper, because the session it attached to predates it and holds
// the agent the reattach exists to preserve.
func (c *Couch) failTrackedPostAckStart(shape StartShape, thread ThreadRecord, nonce string, h Handle, cause error) error {
	return errors.Join(cause, c.applyStartCleanup(shape, thread.Address, nonce, h, false))
}

// observeSessionPresence answers the decider's Presence input, and returns WHY
// when it cannot.
//
// An observer that is unavailable, or an error, is UNOBSERVED -- never absent.
// "We could not ask" must not be answered destructively. The error travels back
// rather than being swallowed: it is the operator's only account of why a
// thread was left occupied rather than tidied up, and the cold-resume tail this
// shell replaced did surface it.
func (c *Couch) observeSessionPresence(address ThreadAddress) (SessionPresence, error) {
	sessions, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return PresenceUnobserved, errors.New("exact Pair session observer is unavailable")
	}
	binding, err := sessions.PairSession(address)
	if err != nil {
		return PresenceUnobserved, err
	}
	if binding.Present {
		return PresencePresent, nil
	}
	return PresenceAbsent, nil
}

func (c *Couch) markResumeStartUnknown(thread ThreadRecord, nonce string) error {
	_, err := c.Threads.AdvanceStart(thread.Address, thread.Revision, StartEvent{
		Kind: StartRecoveredUnknown, Nonce: nonce,
	})
	return err
}

// diagnoseRegistrationFailure says WHICH half of startup did not finish, as a
// suffix on the bare deadline error.
//
// #215 cost hours because `await Pair registration {RepoScope:… Tag:…}: context
// deadline exceeded` names neither what was waited for nor whether Pair started
// at all -- the only visible clue was a set of orphaned zellij servers, and they
// only became meaningful after noticing they had no children.
//
// The discriminator is one observation couch already has and did not make: the
// zellij session either appeared or it did not. Taken AFTER the deadline, so it
// costs nothing on the happy path, and only for a timeout -- any other error
// already says what it is.
func (c *Couch) diagnoseRegistrationFailure(err error, address ThreadAddress, budget time.Duration) string {
	if !errors.Is(err, context.DeadlineExceeded) {
		return ""
	}
	sessions, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return fmt.Sprintf(" (waited %s; no session observer, so whether Pair started is unknown)", budget)
	}
	// LIVENESS is the discriminator, and it is the only one. `PairSession`
	// reports a missing index entry as an ERROR rather than Present=false --
	// which is the "Pair never got as far as recording a name" case, the most
	// diagnostic one there is. Branching on the error first would have swallowed
	// it into "could not observe" and said nothing useful.
	binding, observeErr := sessions.PairSession(address)
	if observeErr == nil && binding.Present {
		return fmt.Sprintf(" (waited %s; session %q IS live, so Pair STARTED and did not finish "+
			"registering -- look at Pair's startup, not the launch)", budget, binding.Name)
	}
	// An absent binding and an UNREADABLE one are different states, and only the
	// first supports the conclusion. Saying "Pair never started" because the index
	// could not be read states an inconclusive observation as a fact -- degrade
	// visibly instead (ARCH-SECURE).
	if observeErr != nil {
		return fmt.Sprintf(" (waited %s; could NOT determine whether Pair started: %v. "+
			"That is an unreadable observation, not a verdict -- check the session by hand "+
			"with `zellij list-sessions`)", budget, observeErr)
	}
	return fmt.Sprintf(" (waited %s; NO Pair session is live. Pair never started, or exited "+
		"before registering -- look at the launch, not registration)", budget)
}

func (c *Couch) awaitResumeRegistration(ctx context.Context, address ThreadAddress) error {
	sessions, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return errors.New("exact Pair session observer is unavailable")
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		binding, err := sessions.PairSession(address)
		if err == nil && binding.Present {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
