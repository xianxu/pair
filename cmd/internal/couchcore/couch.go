package couchcore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Couch is the composition root: every seam in one place, every operation a
// method on it. The terminal UI and (later) the advisor's tools are both
// clients of these methods -- never of two separate implementations.
type Couch struct {
	Namespace        CouchNamespace
	Runner           Runner
	Path             PathOps
	Git              GitRunner
	Proc             ProcOps
	Store            Store
	Clock            Clock
	IDs              IDGen
	Threads          *ThreadStore
	StartGrants      *StartGrantStore[StartResolution]
	Entropy          io.Reader
	Artifacts        ThreadArtifactController
	PairLifecycle    *PairLifecycleController
	RootAgent        string
	RepoAgentDefault func(repoRoot, agent string) (LaunchProfile, bool, error)

	reg                       Registry
	names                     NamingTable
	postAckQuiesceTimeout     time.Duration
	postAckRetryDelay         time.Duration
	resumeRegistrationTimeout time.Duration
	sleep                     func(time.Duration)
}

// ReconcileActiveParks performs an explicit active-park reconciliation pass.
// It may query external Pair/Zellij state, so constructors must never call it;
// interactive composition uses the context-bound RecoverActiveParks worker.
func (c *Couch) ReconcileActiveParks(ctx context.Context) error {
	if c == nil || c.PairLifecycle == nil {
		return errors.New("Couch Pair lifecycle controller is unavailable")
	}
	return c.PairLifecycle.ReconcileActive(ctx)
}

// RecoverActiveParks performs the blocking half of restart recovery serially.
// The composition root runs it in one context-bound worker after construction,
// so external Pair/Zellij teardown can never delay startup or fan out.
func (c *Couch) RecoverActiveParks(ctx context.Context) error {
	if c == nil || c.PairLifecycle == nil {
		return errors.New("Couch Pair lifecycle controller is unavailable")
	}
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return err
	}
	var result error
	for _, record := range snapshot.Records {
		if record.Park == nil {
			continue
		}
		_, recoverErr := c.PairLifecycle.Recover(ctx, record.Address)
		result = errors.Join(result, recoverErr)
		if ctx.Err() != nil {
			break
		}
	}
	return result
}

func New(namespace CouchNamespace, r Runner, p PathOps, g GitRunner, proc ProcOps, s Store, c Clock, ids IDGen, entropy io.Reader, artifacts ThreadArtifactController) (*Couch, error) {
	if namespace.Dir() == "" {
		return nil, fmt.Errorf("new couch: empty namespace")
	}
	if s.Dir() != namespace.Dir() {
		return nil, fmt.Errorf("store directory %q does not match couch namespace %q", s.Dir(), namespace.Dir())
	}
	if entropy == nil {
		entropy = rand.Reader
	}
	if artifacts == nil {
		return nil, fmt.Errorf("new couch: nil artifact collision checker")
	}
	reg, names, err := s.Load()
	if err != nil {
		return nil, err
	}
	threads := NewThreadStore(namespace)
	if err := threads.CutoverLegacyActors(reg.Records()); err != nil {
		return nil, fmt.Errorf("cut over legacy actors: %w", err)
	}
	if err := threads.MigrateLegacyRecords(names); err != nil {
		return nil, fmt.Errorf("migrate legacy thread metadata: %w", err)
	}
	result := &Couch{
		Namespace: namespace,
		Runner:    r, Path: p, Git: g, Proc: proc, Store: s, Clock: c, IDs: ids,
		Threads: threads, Entropy: entropy,
		StartGrants: NewStartGrantStore[StartResolution](c, entropy, cloneStartResolution),
		Artifacts:   artifacts,
		reg:         reg, names: names,
		postAckQuiesceTimeout:     500 * time.Millisecond,
		postAckRetryDelay:         100 * time.Millisecond,
		resumeRegistrationTimeout: 5 * time.Second,
		sleep:                     time.Sleep,
	}
	if err := result.reconcileInterruptedStarts(); err != nil {
		return nil, fmt.Errorf("reconcile interrupted starts: %w", err)
	}
	if environment, ok := artifacts.(PairLifecycleEnvironment); ok {
		result.PairLifecycle = &PairLifecycleController{
			Threads: result.Threads, DataDir: environment.PairLifecycleDataDir(),
			Lifecycle: environment.PairLifecycleIO(), Sessions: environment,
			Proc: result.Proc, Clock: result.Clock,
			Nonce:             func() (string, error) { return allocateParkNonce(result.Entropy) },
			CompletionTimeout: 15 * time.Second,
			PollInterval:      25 * time.Millisecond,
		}
	}
	return result, nil
}

// ResolveTree turns an operator-supplied path into a canonical worktree.
func (c *Couch) ResolveTree(path string) (Worktree, error) { return Resolve(path, c.Git, c.Path) }

// Spawn brings up an agent on a tree.
//
// Order matters: the snapshot is persisted BEFORE the caller waits on the
// child. Couch owns the child for its lifetime, so if Save happened after Wait
// a second shell running `couch --list` would see an empty registry
// for the entire session -- which is most of the time.
func (c *Couch) Spawn(args StartArgs) (ActorRecord, Handle, error) {
	resolution, err := c.resolveStartResolution(context.Background(), args)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	return c.spawnResolved(context.Background(), resolution)
}

type PreparedStart struct {
	Token      StartGrantToken `json:"token"`
	Resolution StartResolution `json:"resolution"`
}

func (c *Couch) PrepareStart(ctx context.Context, args StartArgs) (PreparedStart, error) {
	resolution, err := c.resolveStartResolution(ctx, args)
	if err != nil {
		return PreparedStart{}, err
	}
	token, err := c.StartGrants.Issue(resolution)
	if err != nil {
		return PreparedStart{}, err
	}
	return PreparedStart{Token: token, Resolution: cloneStartResolution(resolution)}, nil
}

func (c *Couch) SpawnPrepared(ctx context.Context, token StartGrantToken) (record ActorRecord, handle Handle, err error) {
	accepted, err := c.StartGrants.Claim(token)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	defer func() { err = errors.Join(err, c.StartGrants.Finish(token)) }()
	current, err := c.resolveStartResolution(ctx, StartArgs{
		Worktree: accepted.Worktree, Cwd: accepted.CanonicalPath,
		Stack: accepted.RequestedAgent, Issue: accepted.Issue,
	})
	if err != nil {
		return ActorRecord{}, nil, err
	}
	if current.Fingerprint != accepted.Fingerprint {
		return ActorRecord{}, nil, ErrStartResolutionChanged
	}
	return c.spawnResolved(ctx, accepted)
}

func (c *Couch) resolveStartResolution(ctx context.Context, args StartArgs) (StartResolution, error) {
	if args.WorkingDir() == "" {
		return StartResolution{}, fmt.Errorf("spawn: no path given")
	}
	tree, err := c.ResolveTree(args.WorkingDir())
	if err != nil {
		return StartResolution{}, err
	}
	canonicalPath := NormalizePath(args.WorkingDir())
	if physical, physicalErr := c.Path.Physical(canonicalPath); physicalErr == nil {
		canonicalPath = physical
	}
	repoIdentity, err := c.resolveRepoIdentity(canonicalPath)
	if err != nil {
		return StartResolution{}, err
	}
	preference, found, err := c.Threads.GetPathLaunchPreference(repoIdentity, canonicalPath)
	if err != nil {
		return StartResolution{}, fmt.Errorf("read launch preference: %w", err)
	}
	var pathPreference *PathLaunchPreference
	if found {
		pathPreference = &preference
	}
	rootAgent := c.RootAgent
	if rootAgent == "" {
		rootAgent = "claude"
	}
	selected, err := ResolveLaunchProfile(LaunchProfileInputs{
		ExplicitAgent: args.Stack, Path: pathPreference, RootAgent: rootAgent,
	})
	if err != nil {
		return StartResolution{}, err
	}
	if !launcher.IsSupportedAgent(selected.Profile.Agent) {
		return StartResolution{}, fmt.Errorf("unsupported launch agent %q", selected.Profile.Agent)
	}
	var repoDefault *LaunchProfile
	if c.RepoAgentDefault != nil {
		value, ok, defaultErr := c.RepoAgentDefault(string(tree), selected.Profile.Agent)
		if defaultErr != nil {
			return StartResolution{}, fmt.Errorf("read %s repository default: %w", selected.Profile.Agent, defaultErr)
		}
		if ok {
			repoDefault = &value
		}
	}
	return ResolveStartResolution(StartResolutionInput{
		CanonicalPath: canonicalPath, Worktree: tree, Issue: args.Issue,
		LaunchProfileInputs: LaunchProfileInputs{
			ExplicitAgent: args.Stack, Path: pathPreference, RootAgent: rootAgent, RepoDefault: repoDefault,
		},
		RepoIdentity: repoIdentity,
	})
}

// resolveRepoIdentity returns the Git common directory for a working path.
//
// This is the value the fleet-policy provider used to supply as
// `repo_identity`, derived locally now that the provider is gone (pair#170 M4).
// It must be BYTE-IDENTICAL to what that provider produced, because it keys the
// operator's saved launch preferences: a different string silently orphans
// every remembered agent+argv, and couch would simply forget which agent you
// use in which tree.
//
// `rev-parse --git-common-dir` answers relative in a main checkout (".git") and
// absolute in a linked worktree, so the result is joined against the query
// directory before canonicalizing -- which is also what makes every worktree of
// one repository share an identity, as the name promises. Verified against
// ariadne's own derivation and against the operator's live preference files.
func (c *Couch) resolveRepoIdentity(workingPath string) (string, error) {
	if c.Git == nil {
		return "", errors.New("resolve repository identity: no git runner")
	}
	out, err := c.Git.Run(workingPath, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("resolve repository identity for %q: %w", workingPath, err)
	}
	common := strings.TrimSpace(out)
	if common == "" {
		return "", fmt.Errorf("resolve repository identity for %q: empty git common dir", workingPath)
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(workingPath, common)
	}
	if c.Path != nil {
		if physical, physicalErr := c.Path.Physical(common); physicalErr == nil {
			return physical, nil
		}
	}
	return common, nil
}

func (c *Couch) spawnResolved(ctx context.Context, resolution StartResolution) (ActorRecord, Handle, error) {
	args := StartArgs{
		Worktree: resolution.Worktree, Cwd: resolution.CanonicalPath,
		Stack: resolution.Profile.Agent, ExtraArgs: cloneArgv(resolution.Profile.Argv), Issue: resolution.Issue,
	}
	if err := c.PruneDead(); err != nil {
		return ActorRecord{}, nil, err
	}
	scope, err := launcher.ResolveRepoScope(string(resolution.Worktree))
	if err != nil {
		return ActorRecord{}, nil, err
	}
	startedAt := c.Clock.Now()
	thread, err := c.Threads.AllocateThreadTag(scope.Key, resolution.CanonicalPath, startedAt, c.Entropy, c.Artifacts)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	threadAddress := thread.Address
	profile := LaunchProfileResolution{Profile: cloneLaunchProfile(resolution.Profile), AgentSource: resolution.AgentSource, ArgvSource: resolution.ArgvSource}
	args.Stack = profile.Profile.Agent
	args.ExtraArgs = cloneArgv(profile.Profile.Argv)
	owner, err := c.Proc.Current()
	if err != nil {
		return ActorRecord{}, nil, errors.Join(fmt.Errorf("identify couch supervisor: %w", err), c.releaseClaimIfThreadAbsent(threadAddress))
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.releaseClaimIfThreadAbsent(threadAddress))
	}
	// One revision-checked write for what admission used to do around its
	// capacity decision: clear the reservation, append the creating
	// incarnation, and record the start claim.
	thread, err = c.Threads.CommitStartClaim(threadAddress, thread.Revision, resolution.RepoIdentity, startedAt, StartEvent{
		Kind:    StartClaimed,
		Nonce:   nonce,
		Owner:   SupervisorOwner{PID: owner.PID, Identity: owner.Identity},
		Profile: &profile.Profile,
	})
	if err != nil {
		return ActorRecord{}, nil, errors.Join(fmt.Errorf("record start transaction: %w", err), c.releaseClaimIfThreadAbsent(threadAddress))
	}

	// `pair resume <tag> --layout2` rather than a bare `pair`.
	//
	// The tag: with none, launcher.DecideLaunch returns ActionPick as soon as a
	// detached session exists (decision.go:47), which inside couch's own pty is
	// an fzf picker waiting on an operator who only asked to start. `resume`
	// takes the ForcedTag branch -- attach if live or detached, create
	// otherwise -- and skips the name prompt (help.go:15). The final opaque tag
	// was claimed in ThreadStore before admission, so each accepted start owns a
	// distinct durable Pair session even when several threads share one path.
	//
	// The layout: pinned to layout2 by operator decision 2026-08-22. couch owns
	// terminal switching now, so layout3's third pane -- pair's own user
	// terminal -- is the layer couch replaces. Provisional ("for now"), which is
	// why it is a literal here rather than a knob nobody has asked for.
	//
	// A correction worth keeping: an earlier version of this comment claimed
	// `resume` REFUSES a third argv element and that --layout2 was therefore
	// impossible. Only POSITIONALS are refused -- `ParseArgs` runs
	// `extractLayoutRequest` first (args.go:51), which strips layout flags
	// before the guard ever sees them, and `launchArgsAcceptLayout` admits them
	// for resume because its Command is "". Measured, not reasoned:
	// `resume mytag --layout2` parses to {tag, layout2}; `resume mytag stray`
	// is the thing that errors.
	//
	// This is a deliberate slice of #149, which makes the tag the space's
	// durable identity; #146 needs only that re-entry is deterministic.
	profileRaw, err := launcher.BuildCouchLaunchProfile(
		string(thread.Address.Tag), profile.Profile.Agent, profile.Profile.Argv,
		string(profile.AgentSource), string(profile.ArgvSource),
	)
	if err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
	}
	return c.launchTrackedThread(trackedThreadLaunch{
		Context: ctx,
		Thread:  thread, Nonce: nonce, Args: args, StartedAt: startedAt,
		ProfileRaw: profileRaw, UseRepoDefault: profile.ArgvSource == ArgvSourceRepoDefault,
	})
}

// failPostAckStart owns every error exit after the helper has executed the
// target and before Spawn transfers the handle to its caller. It first proves
// that exact handle is reaped; only then may durable state be reconciled. If
// quiescence cannot be proved, the creating/live record remains occupied.
func (c *Couch) failPostAckStart(address ThreadAddress, h Handle, cause error) error {
	cleanupErr := c.quiescePostAckStart(address, h)

	reconcileErr := c.reconcileInterruptedStarts()
	current, getErr := c.Threads.GetThread(address)
	var markErr error
	if getErr == nil {
		expected := ProcessIdentity{PID: h.PID(), Identity: h.Identity()}
		for _, incarnation := range current.Incarnations {
			if incarnation.PID == expected.PID && incarnation.Identity == expected.Identity && incarnation.State == IncarnationLive {
				_, markErr = c.Threads.MarkIncarnationUnknown(address, expected)
				break
			}
		}
	} else if !errors.Is(getErr, ErrThreadNotFound) {
		markErr = getErr
	}
	return errors.Join(cause, cleanupErr, reconcileErr, markErr)
}

// AbortStarted is the corresponding failure half after Spawn HAS transferred
// a StartResult to the Console but owner-local terminal attachment did not
// commit. The supplied record and handle must still identify the exact actor
// registered by this Couch; otherwise cleanup could kill an unrelated reused
// PID or a handle forged by another caller.
func (c *Couch) AbortStarted(start StartResult, cause error) error {
	if c == nil || start.Handle == nil {
		return errors.Join(cause, errors.New("abort started: start handle is unavailable"))
	}
	if err := validateThreadAddress(start.Record.Thread); err != nil {
		return errors.Join(cause, fmt.Errorf("abort started: invalid thread address: %w", err))
	}
	if start.Record.ID == "" || start.Record.PID <= 0 || start.Record.Identity == "" ||
		start.Handle.PID() != start.Record.PID || start.Handle.Identity() != start.Record.Identity {
		return errors.Join(cause, errors.New("abort started: record/handle process identity mismatch"))
	}
	registered := false
	for _, record := range c.reg.Records() {
		if record.ID != start.Record.ID {
			continue
		}
		if record.Thread != start.Record.Thread || record.Args.Worktree != start.Record.Args.Worktree ||
			record.PID != start.Record.PID || record.Identity != start.Record.Identity {
			return errors.Join(cause, errors.New("abort started: registered actor identity mismatch"))
		}
		registered = true
		break
	}
	if !registered {
		return errors.Join(cause, errors.New("abort started: actor is not registered by this Couch"))
	}

	cleanupErr := c.failPostAckStart(start.Record.Thread, start.Handle, cause)
	c.reg = c.reg.RemoveActor(start.Record.Args.Worktree, start.Record.ID)
	return errors.Join(cleanupErr, c.Store.Save(c.reg, c.names))
}

// quiescePostAckStart retains the live Handle on this call stack and retries
// every unproven cleanup class. It deliberately has no bounded "give up" path:
// returning would transfer an error without a supervisor, and the operation
// caller is allowed to discard handles on error. A persistent external failure
// therefore leaves the owning start operation blocked and retrying, not a
// workspace writer merely represented by an occupied durable record.
func (c *Couch) quiescePostAckStart(address ThreadAddress, h Handle) error {
	var firstErr error
	handleQuiet := false
	handleCleanup := newHandleCleanup(h)
	for {
		if !handleQuiet {
			if err := handleCleanup.attempt(c.postAckQuiesceTimeout); err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("quiesce post-ack child %d/%q: %w", h.PID(), h.Identity(), err)
				}
				c.sleepPostAckRetry()
				continue
			}
			handleQuiet = true
		}
		if err := c.Artifacts.Quiesce(address); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("quiesce durable Pair session %+v: %w", address, err)
			}
			c.sleepPostAckRetry()
			continue
		}
		return firstErr
	}
}

func (c *Couch) sleepPostAckRetry() {
	delay := c.postAckRetryDelay
	if delay <= 0 {
		delay = 100 * time.Millisecond
	}
	sleep := c.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	sleep(delay)
}

type handleCleanup struct {
	h            Handle
	waited       chan int
	directReaped bool
}

func newHandleCleanup(h Handle) *handleCleanup {
	waited := make(chan int, 1)
	go func() { waited <- h.Wait() }()
	return &handleCleanup{h: h, waited: waited}
}

func (q *handleCleanup) attempt(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	termErr := q.h.Signal(syscall.SIGTERM)
	select {
	case <-q.waited:
		q.directReaped = true
	case <-time.After(timeout):
	}
	// TERM may reap the direct Pair client while a TERM-resistant sidecar or
	// child remains in its owned process group. KILL is therefore unconditional:
	// this path is rollback, not graceful actor shutdown.
	killErr := q.h.Signal(os.Kill)
	if !q.directReaped {
		select {
		case <-q.waited:
			q.directReaped = true
		case <-time.After(timeout):
			return errors.Join(termErr, killErr, errors.New("timed out waiting for killed child"))
		}
	}
	deadline := time.Now().Add(timeout)
	for q.h.Alive() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if q.h.Alive() {
		return errors.Join(termErr, killErr, errors.New("owned process group remained alive after reap"))
	}
	return errors.Join(termErr, killErr)
}

func allocateStartNonce(entropy io.Reader) (string, error) {
	var random [8]byte
	if _, err := io.ReadFull(entropy, random[:]); err != nil {
		return "", fmt.Errorf("allocate start nonce: %w", err)
	}
	return "start-" + hex.EncodeToString(random[:]), nil
}

func allocateParkNonce(entropy io.Reader) (string, error) {
	var random [8]byte
	if _, err := io.ReadFull(entropy, random[:]); err != nil {
		return "", fmt.Errorf("allocate park nonce: %w", err)
	}
	return "park-" + hex.EncodeToString(random[:]), nil
}

func (c *Couch) rollbackUnforkedStart(thread ThreadRecord) error {
	deleteErr := c.Threads.DeleteUnstartedThread(thread.Address, thread.Revision)
	if deleteErr != nil {
		return deleteErr
	}
	return c.releaseClaimIfThreadAbsent(thread.Address)
}

func (c *Couch) rollbackTrackedStart(thread ThreadRecord, nonce string) error {
	deleteErr := c.Threads.DeleteStart(thread.Address, thread.Revision, nonce)
	if deleteErr != nil {
		return deleteErr
	}
	return c.releaseClaimIfThreadAbsent(thread.Address)
}

func (c *Couch) awaitThreadRegistration(ctx context.Context, address ThreadAddress) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		evidence, err := c.Artifacts.Registration(address)
		if err != nil {
			return err
		}
		switch evidence {
		case RegistrationEstablished:
			return nil
		case RegistrationUnknown:
			return errors.New("Pair registration evidence is unknown")
		case RegistrationAbsent:
		default:
			return fmt.Errorf("invalid Pair registration evidence %q", evidence)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *Couch) reconcileInterruptedStarts() error {
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return err
	}
	for _, record := range snapshot.Records {
		transaction, err := CurrentStartTransaction(record)
		if err != nil {
			continue
		}
		observation := StartObservation{
			Owner:  observeExactProcess(c.Proc, transaction.Owner),
			Helper: Dead,
		}
		if transaction.Helper != nil {
			observation.Helper = observeExactProcess(c.Proc, *transaction.Helper)
		}
		registration, registrationErr := c.Artifacts.Registration(record.Address)
		if registrationErr != nil {
			return fmt.Errorf("read Pair registration for %+v: %w", record.Address, registrationErr)
		}
		observation.Registration = registration
		decision, err := ReconcileStart(record, observation)
		if err != nil {
			return err
		}
		switch decision.Action {
		case StartKeepOccupied:
			continue
		case StartRollback:
			if err := c.rollbackTrackedStart(record, decision.Nonce); err != nil {
				return err
			}
		case StartPromoteLive:
			if _, err := c.Threads.AdvanceStart(record.Address, record.Revision, StartEvent{Kind: StartRegistered, Nonce: decision.Nonce}); err != nil {
				return err
			}
		case StartPromoteUnknown:
			if _, err := c.Threads.AdvanceStart(record.Address, record.Revision, StartEvent{Kind: StartRecoveredUnknown, Nonce: decision.Nonce}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown start reconciliation action %q", decision.Action)
		}
	}
	return nil
}

func observeExactProcess(proc ProcOps, expected ProcessIdentity) Liveness {
	switch proc.Exists(expected.PID) {
	case Dead:
		return Dead
	case Unknown:
		return Unknown
	}
	identity, err := proc.Identity(expected.PID)
	if err != nil {
		return Unknown
	}
	if identity != expected.Identity {
		return Dead
	}
	return Live
}

func (c *Couch) releaseClaimIfThreadAbsent(address ThreadAddress) error {
	_, err := c.Threads.GetThread(address)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrThreadNotFound) {
		return err
	}
	return c.Artifacts.Release(address)
}

// Liveness recomputes an actor's state from the persisted {PID, Identity}.
//
// Three-valued on purpose. A PID recycled by an unrelated process reports Dead
// because the kernel start token differs; a probe that cannot answer reports
// Unknown, and callers that act destructively must treat Unknown as "leave it
// alone".
func (c *Couch) Liveness(a ActorRecord) Liveness {
	if a.PID == 0 || a.Identity == "" {
		return Dead // nothing was ever recorded to check against
	}
	switch c.Proc.Exists(a.PID) {
	case Dead:
		return Dead
	case Unknown:
		return Unknown
	}
	id, err := c.Proc.Identity(a.PID)
	if err != nil {
		// The process exists but we could not read its token. That is not
		// evidence of anything; refusing to guess is the safe answer.
		return Unknown
	}
	if id != a.Identity {
		return Dead // same PID, different process
	}
	return Live
}

// IsLive is the display-side convenience. Do not branch on it for anything
// destructive -- it folds Unknown into false.
func (c *Couch) IsLive(a ActorRecord) bool { return c.Liveness(a) == Live }

func (c *Couch) List() []ActorRecord {
	out := c.reg.Records()
	sort.Slice(out, func(i, j int) bool { return out[i].Args.Worktree < out[j].Args.Worktree })
	return out
}

func (c *Couch) Get(w Worktree) []ActorRecord { return c.reg.Get(w) }
func (c *Couch) Entry(w Worktree) NameEntry   { return c.names.Entry(w) }

func (c *Couch) SetName(w Worktree, name string) error {
	c.names = c.names.SetName(w, name)
	return c.Store.Save(c.reg, c.names)
}

func (c *Couch) SetDescription(w Worktree, desc string) error {
	c.names = c.names.SetDescription(w, desc)
	return c.Store.Save(c.reg, c.names)
}

// Forget drops an actor from the registry, freeing its tree.
func (c *Couch) Forget(w Worktree, id ActorID) error {
	c.reg = c.reg.RemoveActor(w, id)
	return c.Store.Save(c.reg, c.names)
}

// knownTrees is every tree couch knows about: those with actors and those with
// only a label. Both are addressable -- a parked tree is exactly the thread an
// operator loses track of.
func (c *Couch) knownTrees() []Worktree {
	seen := map[string]bool{}
	var out []Worktree
	add := func(w Worktree) {
		if w != "" && !seen[w.Key()] {
			seen[w.Key()] = true
			out = append(out, w)
		}
	}
	for _, r := range c.reg.Records() {
		add(r.Args.Worktree)
	}
	for _, e := range c.names.All() {
		add(e.Tree)
	}
	return out
}

// LookupTrees resolves a fuzzy human reference to every tree it could mean.
//
// It matches the repo basename rendered as an unnamed tree's fallback label,
// the operator's name and typed description, AND the agent's own published
// line. All four answer "what is this thread called", so all four derive from
// one lookup -- displaying a label while making it unsearchable delivers half
// the behaviour.
func (c *Couch) LookupTrees(ref string) []Worktree {
	needle := strings.ToLower(strings.TrimSpace(ref))
	if needle == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []Worktree
	for _, w := range c.names.Lookup(ref) {
		if !seen[w.Key()] {
			seen[w.Key()] = true
			out = append(out, w)
		}
	}
	for _, w := range c.knownTrees() {
		if seen[w.Key()] {
			continue
		}
		if strings.Contains(strings.ToLower(w.Repo()), needle) ||
			strings.Contains(strings.ToLower(c.Describe(w)), needle) {
			seen[w.Key()] = true
			out = append(out, w)
		}
	}
	return out
}

// ResolveRef turns a human reference into the actors it names.
//
// A reference may be an ActorID, a label (operator name, operator description,
// or the agent's published line), or a path. ActorID also distinguishes legacy
// co-tenants retained during migration.
func (c *Couch) ResolveRef(ref string) ([]ActorRecord, []Worktree, error) {
	trimmed := strings.TrimSpace(ref)

	for _, r := range c.reg.Records() {
		if string(r.ID) == trimmed {
			return []ActorRecord{r}, []Worktree{r.Args.Worktree}, nil
		}
	}

	trees := c.LookupTrees(trimmed)
	if len(trees) == 0 {
		if w, err := c.ResolveTree(trimmed); err == nil {
			trees = []Worktree{w}
		}
	}
	if len(trees) == 0 {
		return nil, nil, fmt.Errorf("no actor or tree matches %q", ref)
	}
	var out []ActorRecord
	for _, t := range trees {
		out = append(out, c.reg.Get(t)...)
	}
	return out, trees, nil
}

// Views decorates records with the state that is computed rather than stored.
func (c *Couch) Views(recs []ActorRecord) []ActorView {
	out := make([]ActorView, 0, len(recs))
	for _, r := range recs {
		e := c.names.Entry(r.Args.Worktree)
		out = append(out, ActorView{
			Record: r,
			Live:   c.Liveness(r) == Live,
			State:  c.Liveness(r),
			Name:   e.Name,
			Desc:   c.Describe(r.Args.Worktree),
		})
	}
	return out
}

// Describe returns the agent-supplied one-liner, preferring the sidecar the
// live agent writes and falling back to the last value couch stored.
//
// This is not the published-status artifact returning: it is a one-line LABEL,
// and a stale label still finds the right tree. Labels tolerate staleness;
// state does not.
func (c *Couch) Describe(w Worktree) string {
	if s, err := c.Store.ReadDescription(w); err == nil && s != "" {
		return s
	}
	return c.names.Entry(w).Description
}

// treeFor resolves a ref to exactly one tree, erroring on ambiguity rather
// than guessing -- fuzzy in, exact out.
func (c *Couch) treeFor(ref string) (Worktree, error) {
	if trees := c.LookupTrees(ref); len(trees) == 1 {
		return trees[0], nil
	} else if len(trees) > 1 {
		return "", fmt.Errorf("%q matches %d trees; be specific", ref, len(trees))
	}
	return c.ResolveTree(ref)
}

// TreeSummary is a worktree and whatever couch knows about it. A tree with no
// live actor is still a row: a named tree nobody is running is exactly the
// parked thread this project exists to stop losing.
type TreeSummary struct {
	Tree   Worktree    `json:"tree"`
	Name   string      `json:"name,omitempty"`
	Desc   string      `json:"description,omitempty"`
	Actors []ActorView `json:"actors,omitempty"`
}

// Live reports whether any actor on this tree is running.
func (s TreeSummary) Live() bool {
	for _, a := range s.Actors {
		if a.Live {
			return true
		}
	}
	return false
}

// Summarize groups actors by tree and folds in every tree that has a name or
// description but no actor, so parked threads stay visible.
func (c *Couch) Summarize(trees []Worktree) []TreeSummary {
	seen := map[string]*TreeSummary{}
	order := []string{}

	add := func(w Worktree) *TreeSummary {
		if s, ok := seen[w.Key()]; ok {
			return s
		}
		e := c.names.Entry(w)
		s := &TreeSummary{Tree: w, Name: e.Name, Desc: c.Describe(w)}
		seen[w.Key()] = s
		order = append(order, w.Key())
		return s
	}

	// A non-empty filter RESTRICTS the result. Folding in every registry
	// record regardless would make the argument additive only, so `show <ref>`
	// would print exactly what `list` prints.
	want := map[string]bool{}
	for _, w := range trees {
		want[w.Key()] = true
		add(w)
	}
	for _, r := range c.reg.Records() {
		if len(want) > 0 && !want[r.Args.Worktree.Key()] {
			continue
		}
		sum := add(r.Args.Worktree)
		sum.Actors = append(sum.Actors, c.Views([]ActorRecord{r})...)
	}
	if len(trees) == 0 {
		for _, e := range c.names.All() {
			add(e.Tree)
		}
	}

	out := make([]TreeSummary, 0, len(order))
	for _, k := range order {
		out = append(out, *seen[k])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tree < out[j].Tree })
	return out
}

// PruneDead removes records whose process is gone and persists the result.
//
// Liveness is recomputed rather than stored, so the registry accumulates
// records for children that have exited. Pruning is what keeps the
// one-agent-per-tree guard meaningful: without it the guard protects a tree
// against a process that no longer exists.
func (c *Couch) PruneDead() error {
	var dead []ActorRecord
	for _, r := range c.reg.Records() {
		// Only a KNOWN-dead record is pruned. Pruning on Unknown deletes a
		// live actor's registration whenever the probe fails, and then lets a
		// second agent onto its tree -- observed in smoke testing, where a
		// sandboxed probe destroyed a running session's record.
		if c.Liveness(r) == Dead {
			dead = append(dead, r)
		}
	}
	if len(dead) == 0 {
		return nil
	}
	for _, r := range dead {
		c.reg = c.reg.RemoveActor(r.Args.Worktree, r.ID)
	}
	return c.Store.Save(c.reg, c.names)
}

// Stop signals an actor's child and then forgets it.
//
// Order matters and is the opposite of what it was: forgetting first would
// free the tree while the agent kept running, so the next TUI start would
// be allowed and two agents would share one index lock and one branch --
// opening the exact hazard the registry exists to close.
//
// The identity token is re-checked immediately before signalling, because a
// stale record's PID may have been recycled by an unrelated process and
// SIGTERM to the wrong pid is not recoverable.
func (c *Couch) Stop(a ActorRecord) (signalled bool, err error) {
	// Signal on Live or Unknown: refusing to signal because we could not
	// confirm liveness would leave a running agent behind while freeing its
	// tree, which is the hazard Stop exists to close.
	if c.Liveness(a) != Dead {
		if err := c.Proc.Signal(a.PID, TermSignal); err != nil {
			return false, fmt.Errorf("stop %s: %w", a.ID, err)
		}
		signalled = true
	}
	return signalled, c.Forget(a.Args.Worktree, a.ID)
}

// PublishDescription is the agent-facing half of the description: a session
// running inside a tree writes its own one-liner, which Describe then prefers
// over anything the operator typed.
func (c *Couch) PublishDescription(w Worktree, text string) error {
	return c.Store.WriteDescription(w, text)
}
