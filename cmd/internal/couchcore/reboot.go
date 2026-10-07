package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// RebootTarget names what reboot replaces: a slot by its host checkout (:1+),
// or an ordinary thread by its address (:0). Agent optionally overrides the
// fresh conversation's agent.
type RebootTarget struct {
	Path    string
	Address ThreadAddress
	Agent   string
}

// RebootResult is what reboot did, including what it deliberately did not do.
type RebootResult struct {
	Start StartResult
	// Archived is the retired record's address, when there was one to retire.
	Archived ThreadAddress
	// ArchiveOnly: the record was retired and nothing was started, for Reason.
	ArchiveOnly bool
	Reason      string
	// SessionNotStopped: the record could not be read, so its session was left
	// alone rather than stopped on the strength of a record couch could not
	// identify (ArchiveThread's rule).
	SessionNotStopped bool
}

// Started makes a reboot that launched a fresh agent adoptable like any start.
func (r RebootResult) Started() (StartResult, bool) { return r.Start, r.Start.Handle != nil }

// Warning is the operator-facing note, empty when there is nothing to say:
// archive's own wording, because the retirement is archive's.
func (r RebootResult) Warning() string {
	return ArchiveResult{Record: ThreadRecord{Address: r.Archived}, SessionNotStopped: r.SessionNotStopped}.Warning()
}

// Reboot archives the old conversation's record with its evidence and starts
// a fresh agent in the same slot or path, with a new tag. It never touches the
// worktree itself (pair#367 owns shaping a slot's workspace).
//
// ORDER (ARCH-ORDER). Everything that can fail without touching the world
// runs before the first irreversible effect, which is quiesce: classify, read
// the record, decide, and resolve the fresh launch profile. So a profile that
// cannot resolve refuses with the old agent still running -- reboot never stops
// an agent and then starts nothing. Then prepareRetirement (archive's own
// admission and quiesce), then one store journal that retires the old record
// and publishes the fresh claimed one (:0 ReplaceThreadExpected, :1+
// replaceSlotCurrent), then the launch. A launch failure rolls back only the
// new claim: the old record stays archived and the path is free, so a retry
// starts fresh rather than archiving twice.
func (c *Couch) Reboot(ctx context.Context, t RebootTarget) (RebootResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.Threads == nil || c.Artifacts == nil {
		return RebootResult{}, errors.New("reboot requires a thread store and an artifact controller")
	}
	if (t.Path == "") == (t.Address == ThreadAddress{}) {
		return RebootResult{}, errors.New("reboot needs exactly one of a slot path or a thread address")
	}
	if t.Path != "" {
		return c.rebootSlot(ctx, t)
	}
	return c.rebootPrimary(ctx, t)
}

// rebootAdmit is step 1: the classification, through the same admission rule
// the switcher's offer reads.
func (c *Couch) rebootAdmit(ctx context.Context, address ThreadAddress) error {
	state, reason, err := c.classifyForAction(ctx, address)
	if err != nil {
		return fmt.Errorf("reboot %s: its state could not be classified; inspect and retry: %w", address.Tag, err)
	}
	if !RebootableState(state, reason) {
		return fmt.Errorf("reboot %s: %s", address.Tag, archiveRefusal(state, reason))
	}
	return nil
}

func (c *Couch) directoryPresent(path string) bool {
	if path == "" || c.Path == nil {
		return false
	}
	_, err := c.Path.Physical(path)
	return err == nil
}

func (c *Couch) rebootPrimary(ctx context.Context, t RebootTarget) (RebootResult, error) {
	address := t.Address
	if err := validateThreadAddress(address); err != nil {
		return RebootResult{}, err
	}
	ctx, release, err := c.hold(ctx, address, "reboot")
	if err != nil {
		return RebootResult{}, err
	}
	defer release()
	facts := RebootFacts{Record: RebootRecordReadable}
	old, readErr := c.Threads.GetThread(address)
	switch {
	case errors.Is(readErr, ErrThreadNotFound):
		facts.Record = RebootRecordNone
	case readErr != nil:
		// Unreadable is still retirable (ArchivableState permits it), but it
		// cannot be classified, so it is never grounds to stop a session.
		facts.Record = RebootRecordUnreadable
	default:
		if err := c.rebootAdmit(ctx, address); err != nil {
			return RebootResult{}, err
		}
		facts.DirectoryPresent = c.directoryPresent(old.StartingPath)
	}
	plan, reason := DecideReboot(facts)
	if plan == RebootRefuse {
		return RebootResult{}, fmt.Errorf("reboot %s: %s", address.Tag, reason)
	}
	// Step 2, preflight: the fresh profile resolves before anything is
	// stopped. Enrollment is an idempotent family write and touches no session.
	var resolution StartResolution
	if plan == RebootArchiveAndStart {
		var err error
		resolution, err = c.resolveStartResolution(ctx, StartArgs{Cwd: old.StartingPath, Stack: t.Agent, Action: StartOpen})
		if err != nil {
			return RebootResult{}, fmt.Errorf("reboot %s: the fresh agent cannot be resolved, so nothing was stopped: %w", address.Tag, err)
		}
		if resolution.Target.Kind == ThreadTargetSlot {
			return RebootResult{}, fmt.Errorf("reboot %s: its path now resolves to a slot; reboot the slot instead", address.Tag)
		}
		if err := launcher.ValidateFreshAgentArgs(resolution.Profile.Agent, resolution.Profile.Argv); err != nil {
			return RebootResult{}, fmt.Errorf("reboot %s: the fresh agent cannot be resolved, so nothing was stopped: %w", address.Tag, err)
		}
		if err := c.enrollPrimaryResolution(ctx, resolution); err != nil {
			return RebootResult{}, err
		}
	}
	// Step 3: archive's admission and quiesce.
	r, err := c.prepareRetirement(ctx, address)
	if err != nil {
		return RebootResult{}, err
	}
	result := retiredResult(address, r)
	if r.RolledBack {
		// The record held only an unfinished start and is already gone: there
		// is no old record left to replace, so this is an ordinary fresh :0
		// start at its path, through the same guard every start passes.
		if plan, reason = DecideReboot(RebootFacts{Record: RebootRecordRolledBack, DirectoryPresent: facts.DirectoryPresent}); plan != RebootStartOnly {
			return result, fmt.Errorf("reboot %s: %s", address.Tag, reason)
		}
		rows, err := c.ActionableThreadInventoryContext(ctx, nil)
		if err != nil {
			return result, err
		}
		actor, handle, err := c.spawnResolved(ctx, resolution, rows)
		result.Start = StartResult{Record: actor, Handle: handle}
		return result, err
	}
	if plan == RebootArchiveOnly {
		if err := c.Threads.ArchiveThreadExpected(address, r.Revision); err != nil {
			return RebootResult{}, err
		}
		result.ArchiveOnly, result.Reason = true, reason
		return result, nil
	}
	// Step 4: one journal retires the old record and publishes the fresh one.
	scope, err := launcher.ResolveRepoScope(string(resolution.Worktree))
	if err != nil {
		return RebootResult{}, err
	}
	profile := LaunchProfileResolution{Profile: cloneLaunchProfile(resolution.Profile), AgentSource: resolution.AgentSource, ArgvSource: resolution.ArgvSource}
	record, nonce, err := c.claimFreshRecord(ctx, freshClaimInput{
		ScopeKey: scope.Key, Cwd: resolution.CanonicalPath, RepoIdentity: resolution.RepoIdentity,
		TagPrefix: filepath.Base(string(resolution.Worktree)), Profile: profile, Store: c.Threads,
		Commit: func(next ThreadRecord) error {
			return c.Threads.ReplaceThreadExpected(address, r.Revision, next)
		},
	})
	if err != nil {
		return RebootResult{}, err
	}
	// Step 5: launch. A failure rolls back only the new claim.
	actor, handle, err := c.launchClaimedThread(claimedLaunch{
		Context: ctx, Thread: record, Nonce: nonce, StartedAt: c.Clock.Now(), Profile: profile,
		Args: StartArgs{Worktree: resolution.Worktree, Cwd: resolution.CanonicalPath, Issue: resolution.Issue},
	})
	result.Start = StartResult{Record: actor, Handle: handle}
	return result, err
}

// retiredResult is what reboot reports about the retirement half, for both
// kinds: the record it retired (none when retirement only rolled back an
// unfinished start) and the session it deliberately did not stop. One
// construction site, so a :0 and a :1+ reboot cannot report it differently.
func retiredResult(address ThreadAddress, r retirement) RebootResult {
	result := RebootResult{SessionNotStopped: r.SessionNotStopped}
	if !r.RolledBack {
		result.Archived = address
	}
	return result
}

func (c *Couch) rebootSlot(ctx context.Context, t RebootTarget) (RebootResult, error) {
	// Selecting the slot reconciles its workspace first (pair#387): a missing
	// directory is re-created, and a failure refuses with its resource and
	// cause before anything is stopped. A live agent is a degraded hold here;
	// the post-stop pass inside startFreshSlot repairs what it held.
	local, slot, err := c.selectSlot(ctx, t.Path, true)
	if err != nil {
		return RebootResult{}, err
	}
	observed, err := local.observeSlotCurrent()
	if err != nil {
		return RebootResult{}, err
	}
	if observed.Unsupported {
		return RebootResult{}, observed.Err
	}
	facts := RebootFacts{Slot: true, Record: RebootRecordNone, DirectoryPresent: true}
	var address ThreadAddress
	switch {
	case observed.Record != nil:
		facts.Record, address = RebootRecordReadable, observed.Record.Address
		if err := c.rebootAdmit(ctx, address); err != nil {
			return RebootResult{}, err
		}
	case observed.Exists:
		// Unreadable: startFreshSlot moves its exact bytes to recovery/ in the
		// same journal, and refuses unless every managed session is absent, so
		// nothing is stopped on the strength of a record couch cannot read.
		facts.Record = RebootRecordUnreadable
	}
	// Held after the switch so the case bodies cannot shadow ctx. An
	// unreadable record has no address to hold; startFreshSlot refuses unless
	// every managed session is absent, so it stops nothing on that record.
	if address != (ThreadAddress{}) {
		var release func()
		ctx, release, err = c.hold(ctx, address, "reboot")
		if err != nil {
			return RebootResult{}, err
		}
		defer release()
	}
	plan, reason := DecideReboot(facts)
	if plan == RebootRefuse || plan == RebootArchiveOnly {
		// Archive-only cannot arise here: a slot record exists only inside a
		// present directory. Refusing keeps the impossible case loud.
		return RebootResult{}, fmt.Errorf("reboot %s: %s", t.Path, reason)
	}
	// Preflight: the fresh profile, before any quiesce.
	family, err := c.slotFamily(ctx, slot, false)
	if err != nil {
		return RebootResult{}, err
	}
	cwd, err := ValidateFamilyPath(slot.WorktreeRoot, family.RelativeStart)
	if err != nil {
		return RebootResult{}, err
	}
	profile, err := c.slotLaunchProfile(local, slot, cwd, t.Agent)
	if err == nil {
		err = launcher.ValidateFreshAgentArgs(profile.Profile.Agent, profile.Profile.Argv)
	}
	if err != nil {
		return RebootResult{}, fmt.Errorf("reboot %s: the fresh agent cannot be resolved, so nothing was stopped: %w", t.Path, err)
	}
	var result RebootResult
	if facts.Record == RebootRecordReadable {
		r, err := c.prepareRetirement(ctx, address)
		if err != nil {
			return RebootResult{}, err
		}
		// The archive half is replaceSlotCurrent's, inside startFreshSlot: it
		// re-observes the slot and refuses on any change since. A record that
		// turned unreadable after the preflight read it is filed without its
		// session being stopped, and the result says so, as a :0 reboot's does.
		result = retiredResult(address, r)
	}
	start, err := c.startFreshSlot(ctx, t.Path, t.Agent, false, nil, &profile)
	result.Start = start
	return result, err
}
