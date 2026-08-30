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
	Thread         ThreadRecord
	Nonce          string
	Args           StartArgs
	StartedAt      time.Time
	ProfileRaw     string
	UseRepoDefault bool
	Resume         bool
}

// launchTrackedThread is the single post-claim launch path for both a newly
// allocated thread and an exact verified-park resume.
func (c *Couch) launchTrackedThread(in trackedThreadLaunch) (ActorRecord, Handle, error) {
	thread := in.Thread
	argv := []string{"pair", "resume", string(thread.Address.Tag), "--layout2"}
	env := []string{
		"COUCH_TREE=" + string(in.Args.Worktree),
		"COUCH_STORE_DIR=" + c.Namespace.Dir(),
		"COUCH_THREAD_SCOPE=" + thread.Address.RepoScope,
		"COUCH_THREAD_TAG=" + string(thread.Address.Tag),
	}
	if in.Resume {
		env = append(env, "COUCH_THREAD_RESUME=1")
	}
	env = append(env,
		launcher.CouchLaunchProfileEnv+"="+strings.TrimSpace(in.ProfileRaw),
		"PAIR_USE_REPO_DEFAULT=",
	)
	if in.UseRepoDefault {
		env[len(env)-1] = "PAIR_USE_REPO_DEFAULT=1"
	}
	h, err := c.Runner.StartBlocked(in.Args.WorkingDir(), argv, env, 10*time.Second)
	if err != nil {
		return ActorRecord{}, nil, errors.Join(
			fmt.Errorf("spawn %s: %w", in.Args.Worktree, err),
			c.rollbackTrackedStart(thread, in.Nonce),
		)
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
	if err := h.Acknowledge(); err != nil {
		cause := fmt.Errorf("acknowledge blocked helper %+v: %w", thread.Address, err)
		return ActorRecord{}, h, c.failTrackedPostAckStart(in.Resume, thread, in.Nonce, h, cause)
	}
	registrationTimeout := 5 * time.Second
	if in.Resume && c.resumeRegistrationTimeout > 0 {
		registrationTimeout = c.resumeRegistrationTimeout
	}
	registrationContext, cancelRegistration := context.WithTimeout(context.Background(), registrationTimeout)
	if in.Resume {
		err = c.awaitResumeRegistration(registrationContext, thread.Address)
	} else {
		err = c.awaitThreadRegistration(registrationContext, thread.Address)
	}
	cancelRegistration()
	if err != nil {
		cause := fmt.Errorf("await Pair registration %+v: %w", thread.Address, err)
		return ActorRecord{}, h, c.failTrackedPostAckStart(in.Resume, thread, in.Nonce, h, cause)
	}
	registeredThread, err := c.Threads.AdvanceStart(thread.Address, thread.Revision, StartEvent{Kind: StartRegistered, Nonce: in.Nonce})
	if err != nil {
		cause := fmt.Errorf("promote registered thread %+v: %w", thread.Address, err)
		return ActorRecord{}, h, c.failTrackedPostAckStart(in.Resume, thread, in.Nonce, h, cause)
	}
	thread = registeredThread

	record := ActorRecord{
		ID: c.IDs.NewID(), Thread: thread.Address, Args: in.Args,
		StartedAt: in.StartedAt, PID: h.PID(), Identity: h.Identity(),
	}
	c.reg = c.reg.Insert(record)
	if err := c.Store.Save(c.reg, c.names); err != nil {
		c.reg = c.reg.RemoveActor(in.Args.Worktree, record.ID)
		return record, h, c.failPostAckStart(thread.Address, h, fmt.Errorf("persist registry: %w", err))
	}
	return record, h, nil
}

func (c *Couch) failTrackedPostAckStart(resume bool, thread ThreadRecord, nonce string, h Handle, cause error) error {
	if !resume {
		return c.failPostAckStart(thread.Address, h, cause)
	}
	cleanupErr := c.quiescePostAckStart(thread.Address, h)
	current, getErr := c.Threads.GetThread(thread.Address)
	if getErr != nil {
		return errors.Join(cause, cleanupErr, getErr)
	}
	sessions, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return errors.Join(cause, cleanupErr, c.markResumeStartUnknown(current, nonce), errors.New("exact Pair session observer is unavailable"))
	}
	binding, bindingErr := sessions.PairSession(thread.Address)
	if bindingErr == nil && !binding.Present && !h.Alive() {
		return errors.Join(cause, cleanupErr, c.rollbackTrackedStart(current, nonce))
	}
	return errors.Join(cause, cleanupErr, bindingErr, c.markResumeStartUnknown(current, nonce))
}

func (c *Couch) markResumeStartUnknown(thread ThreadRecord, nonce string) error {
	_, err := c.Threads.AdvanceStart(thread.Address, thread.Revision, StartEvent{
		Kind: StartRecoveredUnknown, Nonce: nonce,
	})
	return err
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
