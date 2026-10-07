package couchcore

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"strings"
	"time"

	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/panebirth"
)

type trackedThreadLaunch struct {
	Background     bool
	Context        context.Context
	Thread         ThreadRecord
	Nonce          string
	Args           StartArgs
	StartedAt      time.Time
	ProfileRaw     string
	UseRepoDefault bool
	Resume         bool
	Fresh          bool
	Orientation    *orientation.Request
	// Warm marks a REATTACH: the agent is alive behind a client-less zellij
	// session and Pair only has to attach to it.
	Warm bool
	// Begun is when the caller began the operation (before its claim), so
	// the step timings cover the claim too; zero starts them at the launch.
	Begun time.Time
}

// launchTrackedThread is the single post-claim launch path for both a newly
// allocated thread and an exact cold resume.
func (c *Couch) launchTrackedThread(in trackedThreadLaunch) (ActorRecord, Handle, error) {
	ctx := in.Context
	if ctx == nil {
		ctx = context.Background()
	}
	steps := newLaunchSteps(in.Begun)
	if err := freshNonceReachesPair(in); err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(in.Thread, in.Nonce))
	}
	thread := in.Thread
	if c.Slots != nil {
		cwd, err := c.Path.Physical(in.Args.WorkingDir())
		var tree Worktree
		if err == nil {
			tree, err = c.ResolveTree(cwd)
		}
		if err == nil {
			_, err = RelativeFamilyPath(string(tree), cwd)
		}
		if err == nil {
			scope, scopeErr := launcher.ResolveRepoScope(string(tree))
			if scopeErr != nil {
				err = scopeErr
			} else if scope.Key != thread.Address.RepoScope {
				err = errors.New("launch checkout does not match thread scope")
			}
		}
		if err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, in.Nonce))
		}
		in.Args.Cwd = cwd
		in.Args.Worktree = tree
	}
	steps.mark("checkout")

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
	background := ""
	if in.Background {
		background = "1"
	}
	env = append(env, "PAIR_RETENTION_BACKGROUND="+background)
	if in.Resume || in.Fresh {
		env = append(env, "COUCH_THREAD_RESUME=1")
	}
	if in.Orientation != nil {
		err := in.Orientation.Validate()
		var raw []byte
		if err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, in.Nonce))
		}
		var profile launcher.TrustedLaunchProfile
		if err := json.Unmarshal([]byte(in.ProfileRaw), &profile); err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, in.Nonce))
		}
		profile.Orientation = in.Orientation
		raw, err = json.Marshal(profile)
		if err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, in.Nonce))
		}
		in.ProfileRaw = string(raw)
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
	case in.Fresh:
		shape = StartFreshExisting
	case in.Resume && in.Warm:
		shape = StartWarmReattach
	case in.Resume:
		shape = StartColdResume
	}
	binding, err := c.sessionBindingForLaunch(ctx, thread, in.Nonce, in.Warm)
	if err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, in.Nonce))
	}
	bound, err := c.Threads.AdvanceStart(thread.Address, thread.Revision, StartEvent{Kind: StartSessionBound, Nonce: in.Nonce, Binding: &binding})
	if err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, in.Nonce))
	}
	thread = bound
	disposition := "create"
	if in.Warm {
		disposition = "attach"
	}
	intent, err := json.Marshal(launcher.CouchSessionIntent{Scope: thread.Address.RepoScope, Tag: string(thread.Address.Tag), Name: binding.Name, Nonce: in.Nonce, Disposition: disposition})
	if err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, in.Nonce))
	}
	argv = append([]string{argv[0], launcher.CouchSessionFlag}, argv[1:]...)
	env = append(env, launcher.CouchSessionIntentEnv+"="+string(intent))
	steps.mark("prepare")
	h, err := c.Runner.StartBlocked(ctx, in.Args.WorkingDir(), argv, env, 10*time.Second)
	steps.mark("spawn")
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
	// A cold resume starts a new zellij server, so its registration must not
	// ask zellij anything until the thread's pane has been born (#287).
	// Everything here runs while the helper is still blocked: Pair has not yet
	// run, so no birth of THIS thread can be in flight, and whatever changes
	// from here on is this launch's doing.
	var birth *PaneMarks
	if shape == StartColdResume {
		var err error
		if birth, err = c.coldResumeBirthBaseline(thread.Address); err != nil {
			return ActorRecord{}, h, c.failTrackedPreAckStart(thread, in.Nonce, h, err)
		}
	}
	steps.mark("record+baseline")
	if err := h.Acknowledge(); err != nil {
		cause := fmt.Errorf("acknowledge blocked helper %+v: %w", thread.Address, err)
		return ActorRecord{}, h, c.failTrackedPostAckStart(shape, thread, in.Nonce, h, cause)
	}
	if err := ctx.Err(); err != nil {
		return ActorRecord{}, h, c.failTrackedPostAckStart(shape, thread, in.Nonce, h, err)
	}
	registrationTimeout := pairRegistrationTimeout
	if (in.Resume || in.Fresh) && c.resumeRegistrationTimeout > 0 {
		registrationTimeout = c.resumeRegistrationTimeout
	}
	steps.mark("ack")
	registrationContext, cancelRegistration := context.WithTimeout(ctx, registrationTimeout)
	if in.Fresh {
		err = c.awaitFreshRegistration(registrationContext, thread.Address, in.Args.Stack, in.Nonce)
		if err == nil {
			err = c.awaitResumeRegistration(registrationContext, thread.Address, nil)
		}
	} else if in.Resume {
		err = c.awaitResumeRegistration(registrationContext, thread.Address, birth)
	} else {
		err = c.awaitThreadRegistration(registrationContext, thread.Address)
	}
	cancelRegistration()
	steps.mark("registration")
	if err != nil {
		cause := fmt.Errorf("await Pair registration %+v: %w [steps: %s]%s", thread.Address, err, steps,
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
	if err := c.mutateRegistry(func(reg Registry) (Registry, error) {
		pruned, _ := c.withoutDead(reg)
		return pruned.Insert(record), nil
	}); err != nil {
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
	return c.observeSessionPresenceContext(context.Background(), address)
}

func (c *Couch) observeSessionPresenceContext(ctx context.Context, address ThreadAddress) (SessionPresence, error) {
	binding, err := c.recoverySession(ctx, address)
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
	_, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return fmt.Sprintf(" (waited %s; no session observer, so whether Pair started is unknown)", budget)
	}
	// LIVENESS is the discriminator, and it is the only one. `PairSession`
	// reports a missing index entry as an ERROR rather than Present=false --
	// which is the "Pair never got as far as recording a name" case, the most
	// diagnostic one there is. Branching on the error first would have swallowed
	// it into "could not observe" and said nothing useful.
	binding, observeErr := c.recoverySession(context.Background(), address)
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

// awaitResumeRegistration polls until the thread's Pair session is live. A
// non-nil birth is a cold resume's pre-launch sidecar baseline: no session
// probe runs until the pane has been born against it. The probe is
// `list-sessions`, which connects to every socket, and at this cadence it
// killed every cold launch it overlapped (probes/zellijbirthrace: 10/10).
// A warm reattach passes nil, because its session is already live.
func (c *Couch) awaitResumeRegistration(ctx context.Context, address ThreadAddress, birth *PaneMarks) error {
	_, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return errors.New("exact Pair session observer is unavailable")
	}
	wait := &registrationWaitError{phase: "ownership poll", last: "none"}
	start := time.Now()
	if birth != nil {
		err := c.awaitPaneBirth(ctx, address, *birth)
		wait.birth = time.Since(start)
		if err != nil {
			wait.phase, wait.err = "pane-birth wait", err
			return wait
		}
	}
	polling := time.Now()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		binding, err := c.recoverySession(ctx, address)
		switch {
		case err == nil && binding.Present:
			return nil
		case err != nil && ctx.Err() != nil:
			// Cut off by the deadline itself: no observation to report.
			wait.ownership, wait.err = time.Since(polling), ctx.Err()
			return wait
		}
		wait.polls++
		switch {
		case err != nil:
			wait.last = err.Error()
		default:
			wait.last = "not owned"
			if binding.Owner != nil {
				wait.last += " (" + sessionOwnerWord(binding.Owner.State) + prefixed("; ", binding.Owner.Diagnostic) + ")"
			}
		}
		select {
		case <-ctx.Done():
			wait.ownership, wait.err = time.Since(polling), ctx.Err()
			return wait
		case <-ticker.C:
		}
	}
}

// launchSteps times each step of a launch in memory (pair#367 smoke test: the
// operator asked that an earlier slow step never hide behind the one that
// timed out). It writes nothing; failures carry its text.
type launchSteps struct {
	last  time.Time
	steps []string
}

func newLaunchSteps(begun time.Time) *launchSteps {
	s := &launchSteps{last: time.Now()}
	if !begun.IsZero() {
		s.last = begun
		s.mark("claim")
	}
	return s
}

func (s *launchSteps) mark(step string) {
	now := time.Now()
	s.steps = append(s.steps, step+" "+now.Sub(s.last).Round(time.Millisecond).String())
	s.last = now
}

func (s *launchSteps) String() string { return strings.Join(s.steps, ", ") }

// registrationWaitError is a cold resume's registration timeout with its
// phases: which one consumed the budget (the pane-birth wait or the zellij
// ownership poll), each phase's elapsed time, how many ownership polls ran
// and what the last one said (pair#367 smoke test: a bare "context deadline
// exceeded" left the stalled phase unknown). It unwraps to the cause, so the
// deadline diagnosis still applies.
type registrationWaitError struct {
	err              error
	phase            string
	birth, ownership time.Duration
	polls            int
	last             string
}

func (e *registrationWaitError) Error() string {
	return fmt.Sprintf("%v [%s consumed the budget: pane birth %s, ownership poll %s, %d ownership polls; last ownership check: %s]",
		e.err, e.phase, e.birth.Round(time.Millisecond), e.ownership.Round(time.Millisecond), e.polls, e.last)
}

func (e *registrationWaitError) Unwrap() error { return e.err }

// sessionOwnerWord names an ownership verdict for a diagnostic.
func sessionOwnerWord(state launcher.SessionOwnerState) string {
	switch state {
	case launcher.SessionOwnerAbsent:
		return "absent"
	case launcher.SessionOwnerOwned:
		return "owned"
	case launcher.SessionOwnerForeign:
		return "foreign"
	case launcher.SessionOwnerOrphaned:
		return "orphaned"
	}
	return "unknown"
}

func prefixed(prefix, s string) string {
	if s == "" {
		return ""
	}
	return prefix + s
}

// coldResumeBirthBaseline captures the proposed terminal pane marks while the
// helper is blocked. Only an absent proposed terminal may proceed; unreadable
// or live ownership must not become a birth wait followed by destructive cleanup.
func (c *Couch) coldResumeBirthBaseline(address ThreadAddress) (*PaneMarks, error) {
	binding, err := c.recoverySession(context.Background(), address)
	switch {
	case err == nil && binding.Present:
		return nil, errors.New("proposed terminal is already live before cold launch")
	case err != nil && !errors.Is(err, ErrPairSessionBindingAbsent):
		return nil, fmt.Errorf("observe session before cold resume %+v: %w", address, err)
	}
	baseline, err := c.observePaneSidecars(address)
	if err != nil {
		return nil, fmt.Errorf("observe pane sidecars %+v: %w", address, err)
	}
	return &baseline, nil
}

func (c *Couch) observePaneSidecars(address ThreadAddress) (PaneMarks, error) {
	panes, ok := c.Artifacts.(PaneBirthIO)
	if !ok {
		return nil, errors.New("pane birth observer is unavailable")
	}
	return panes.PaneSidecars(address)
}

// awaitPaneBirth polls the thread's pane sidecars, which costs a glob and a
// stat and asks zellij nothing, until one has been born since baseline.
//
// A failed observation is "not yet", never birth. It also doesn't end the wait
// (panebirth.Await owns that rule): a transient stat error after a good create
// would otherwise fail registration, and the cold-resume cleanup would then
// delete the session this launch just made. The last error rides the
// deadline's error, for the diagnosis. The grace is the registration
// deadline itself, and Await reports either one passing as
// DeadlineExceeded, which diagnoseRegistrationFailure keys on.
func (c *Couch) awaitPaneBirth(ctx context.Context, address ThreadAddress, baseline PaneMarks) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return errors.New("pane birth wait has no registration deadline")
	}
	return panebirth.Await(ctx, panebirth.WallClock{}, 10*time.Millisecond, time.Until(deadline), func() (bool, error) {
		now, err := c.observePaneSidecars(address)
		if err != nil {
			return false, fmt.Errorf("observe pane sidecars: %w", err)
		}
		return baseline.BornIn(now), nil
	})
}

func (c *Couch) awaitFreshRegistration(ctx context.Context, address ThreadAddress, agent, attempt string) error {
	if c.FreshRegistration == nil {
		return errors.New("fresh launch registration observer unavailable")
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		registered, err := c.FreshRegistration(ctx, address, agent, attempt)
		if err != nil {
			return err
		}
		if registered {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *Couch) sessionBindingForLaunch(ctx context.Context, thread ThreadRecord, nonce string, warm bool) (couchidentity.SessionBinding, error) {
	if warm {
		observed, err := c.recoverySession(ctx, thread.Address)
		if err != nil {
			return couchidentity.SessionBinding{}, err
		}
		if !observed.Present || observed.Name == "" {
			return couchidentity.SessionBinding{}, errors.New("managed attach session disappeared; resume it again")
		}
		if thread.SessionBinding != nil {
			if thread.SessionBinding.Name != observed.Name {
				return couchidentity.SessionBinding{}, errors.New("managed attach binding differs from observed session")
			}
			return *thread.SessionBinding, nil
		}
		return couchidentity.SessionBinding{Legacy: true, Name: observed.Name, ScopeKey: thread.Address.RepoScope, Tag: string(thread.Address.Tag), StartNonce: nonce}, nil
	}
	// A failed detached proof does not establish absence: an attached session
	// is still the conversation's running agent. Cold creation needs its own
	// positive absence observation before allocating another terminal.
	observed, observeErr := observeRecordSession(ctx, c.Artifacts, thread)
	if observeErr != nil && !errors.Is(observeErr, ErrPairSessionBindingAbsent) {
		return couchidentity.SessionBinding{}, fmt.Errorf("observe session before cold resume: %w", observeErr)
	}
	if observeErr == nil && observed.Present {
		return couchidentity.SessionBinding{}, errors.New("conversation terminal is still live; detach or park it before cold creation")
	}
	if errors.Is(observeErr, ErrPairSessionBindingAbsent) && (thread.SessionBinding != nil || thread.LatestLaunchProfile != nil) {
		return couchidentity.SessionBinding{}, observeErr
	}
	if c.Identities == nil {
		return couchidentity.SessionBinding{}, errors.New("Couch identity allocator is unavailable")
	}
	allocated, err := c.Identities.Allocate(ctx, couchidentity.AllocationRequest{Terminal: true})
	if err != nil {
		return couchidentity.SessionBinding{}, err
	}
	return couchidentity.SessionBinding{C: allocated.C, M: allocated.M, Name: allocated.SessionName, ScopeKey: thread.Address.RepoScope, Tag: string(thread.Address.Tag), StartNonce: nonce}, nil
}

// errFreshNonceUnreachable refuses a fresh launch whose registration nonce
// Pair could never learn.
var errFreshNonceUnreachable = errors.New("fresh launch nonce cannot reach Pair")

// freshNonceReachesPair is the rule a fresh launch's registration rests on: it
// waits for a ready file carrying in.Nonce (awaitFreshRegistration), and Pair
// takes a Couch nonce only from the orientation's attempt; otherwise it mints
// its own and the wait can only run out (pair#367 smoke test: a resume that
// restarted fresh stalled the full budget). So a fresh launch must carry an
// orientation whose attempt is its nonce, checked before any child starts.
func freshNonceReachesPair(in trackedThreadLaunch) error {
	if !in.Fresh {
		return nil
	}
	if in.Orientation == nil || in.Orientation.Attempt != in.Nonce {
		return fmt.Errorf("%w: a fresh launch must hand Pair its nonce through the orientation's attempt", errFreshNonceUnreachable)
	}
	return nil
}
