package couchcore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type slotCurrentObservation struct {
	HostInfo        os.FileInfo
	EnvironmentInfo os.FileInfo
	Raw             []byte
	Exists          bool
	Record          *ThreadRecord
	Unsupported     bool
	Err             error
}

func (s *ThreadStore) readSlotCurrentLocked() (slotCurrentObservation, error) {
	var out slotCurrentObservation
	var physicalErr error
	out.HostInfo, physicalErr = os.Stat(s.slot.WorktreeRoot)
	if physicalErr != nil {
		return out, physicalErr
	}
	out.EnvironmentInfo, physicalErr = os.Stat(s.slot.EnvironmentRoot)
	if physicalErr != nil {
		return out, physicalErr
	}
	raw, err := s.readRetentionFile(filepath.Join(s.root, "thread.json"))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Raw = raw
	out.Exists = true
	var envelope struct {
		SchemaVersion int           `json:"schema_version"`
		Address       ThreadAddress `json:"address"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		out.Err = err
		return out, nil
	}
	if envelope.SchemaVersion != 0 && envelope.SchemaVersion != ThreadSchemaVersion {
		out.Unsupported = true
		out.Err = fmt.Errorf("unsupported slot record version %d; use a compatible Couch binary", envelope.SchemaVersion)
		return out, nil
	}
	record, err := s.decodeThreadRaw(envelope.Address, raw)
	if err != nil {
		out.Err = err
		return out, nil
	}
	out.Record = &record
	return out, nil
}
func (s *ThreadStore) observeSlotCurrent() (out slotCurrentObservation, err error) {
	if !s.layout.Local {
		return out, errors.New("slot observation requires local store")
	}
	err = s.withLock(func() error { var err error; out, err = s.readSlotCurrentLocked(); return err })
	return
}

// replaceSlotCurrent keeps retained evidence and publication in one existing
// local journal. Exact observed bytes serialize fresh against resume and fresh.
func (s *ThreadStore) replaceSlotCurrent(old slotCurrentObservation, next ThreadRecord) error {
	if !s.layout.Local {
		return errors.New("slot replacement requires local store")
	}
	if old.Unsupported {
		return old.Err
	}
	if err := ValidateThreadRecord(next); err != nil {
		return err
	}
	if err := s.validateLocalOrigin(next); err != nil {
		return err
	}
	return s.withLock(func() error {
		current, err := s.readSlotCurrentLocked()
		if err != nil {
			return err
		}
		if old.HostInfo == nil || old.EnvironmentInfo == nil || !os.SameFile(old.HostInfo, current.HostInfo) || !os.SameFile(old.EnvironmentInfo, current.EnvironmentInfo) {
			return errors.New("slot physical directory changed during recovery")
		}
		if current.Exists != old.Exists || !bytes.Equal(current.Raw, old.Raw) {
			return errors.New("slot current changed during recovery; inspect and retry")
		}
		if current.Unsupported {
			return current.Err
		}
		var entries []storeJournalEntry
		if old.Record != nil {
			path := s.archivePath(old.Record.Address)
			if err := provisionSafePath(path); err != nil {
				return err
			}
			var archiveBefore, graceBefore *[]byte
			if prior, err := s.readRetentionFile(path); err == nil {
				archiveBefore = &prior
				backup, err := s.slotRecoveryBackupLocked(prior)
				if err != nil {
					return err
				}
				if backup != nil {
					entries = append(entries, *backup)
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if prior, err := s.readRetentionFile(s.archiveGracePath(old.Record.Address)); err == nil {
				graceBefore = &prior
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			grace, err := s.archiveGraceBytes(old.Record.Address, old.Raw)
			if err != nil {
				return err
			}
			entries = append(entries, storeJournalEntry{Path: relativeStorePath(s.root, path), Expected: archiveBefore, After: &old.Raw}, storeJournalEntry{Path: relativeStorePath(s.root, s.archiveGracePath(old.Record.Address)), Expected: graceBefore, After: &grace})
		} else if old.Exists {
			entry, err := s.slotRecoveryBackupLocked(old.Raw)
			if err != nil {
				return err
			}
			if entry != nil {
				entries = append(entries, *entry)
			}
		}
		raw, err := json.MarshalIndent(next, "", "  ")
		if err != nil {
			return err
		}
		raw = append(raw, '\n')
		var expected *[]byte
		if old.Exists {
			expected = &old.Raw
		}
		entries = append(entries, storeJournalEntry{Path: "thread.json", Expected: expected, After: &raw})
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: entries})
	})
}
func (s *ThreadStore) slotRecoveryBackupLocked(raw []byte) (*storeJournalEntry, error) {
	root := filepath.Join(s.root, "recovery")
	if err := checkSlotDirectory(root); err != nil {
		return nil, err
	}
	files, err := os.ReadDir(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	name := fmt.Sprintf("%x.json", digest)
	total := int64(0)
	exists := false
	for _, file := range files {
		path := filepath.Join(root, file.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("unknown recovery evidence at %s; export or remove it before recovery", path)
		}
		total += info.Size()
		if file.Name() == name {
			got, err := s.readRetentionFile(path)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(got, raw) {
				return nil, errors.New("recovery digest evidence disagrees")
			}
			exists = true
		}
	}
	if exists {
		return nil, nil
	}
	if len(files) >= 16 || total+int64(len(raw)) > 64<<20 {
		return nil, fmt.Errorf("recovery evidence limit reached at %s; export and remove retained recovery files before retrying", root)
	}
	return &storeJournalEntry{Path: filepath.Join("recovery", name), After: &raw}, nil
}

// slotAgentNow is the slot's agent evidence for reconcile: any session left in
// the slot's scope (a record-less survivor included) is a running agent, and a
// failed observation is unknown. Reconcile never removes a checkout unless this
// says none.
func (c *Couch) slotAgentNow(ctx context.Context, slot SlotIdentity) EvidenceAgent {
	observation, err := c.ObserveSlotSessions(ctx, slot)
	switch {
	case err != nil:
		return AgentUnusableUnknown
	case observation.Absent:
		return AgentNone
	}
	return AgentLive
}

func (c *Couch) selectedSlot(ctx context.Context, path string) (*ThreadStore, SlotIdentity, error) {
	return c.selectSlot(ctx, path, false)
}

// selectSlot is selectedSlot. toleratesHolds is reboot's first pass: a slot
// whose repair is held only because an agent may be working there (a hold, not
// a failure) is still selected, so reboot can stop that agent; its post-stop
// pass (startFreshSlot) then repairs what the hold deferred. Any other blocking
// outcome refuses as usual. This keeps the hold's advice ("reboot the slot to
// repair it") an action that can succeed.
func (c *Couch) selectSlot(ctx context.Context, path string, toleratesHolds bool) (*ThreadStore, SlotIdentity, error) {
	if ctx == nil || c == nil || c.Slots == nil || c.Threads == nil {
		return nil, SlotIdentity{}, errors.New("slot services unavailable")
	}
	identity, resolveErr := c.slotWorkspace(ctx, path)
	var slot SlotIdentity
	var err error
	if resolveErr == nil {
		slot, err = SlotIdentityFromWorkspace(identity)
		if err != nil {
			return nil, slot, err
		}
	} else {
		var ok bool
		slot, ok = conventionalSlotFromPath(path)
		if !ok {
			return nil, slot, resolveErr
		}
	}
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return nil, slot, err
	}
	find := func(repo SlotRepository) (SlotCandidate, bool) {
		for _, candidate := range repo.Slots {
			if candidate.Identity.WorktreeRoot == slot.WorktreeRoot && candidate.Identity.Number == slot.Number {
				return candidate, true
			}
		}
		return SlotCandidate{}, false
	}
	candidate, found := find(repository)
	if !found {
		return nil, slot, errors.New("slot directory is not an existing conventional candidate")
	}
	// Every open, resume, reboot and fresh start converges the slot's
	// workspace first (pair#387): verifyHost proves identity only, not setup
	// or dependency clones, so a verified candidate is reconciled too. On a
	// healthy slot this is one observation and an empty plan.
	// A candidate that is not yet a verified checkout is provisioned below,
	// which needs its repository family; a verified one keeps the family it
	// has (reserving an empty start would contradict a recorded one).
	if !candidate.Verified || candidate.Err != nil {
		if _, err := c.Threads.ReserveRepositoryFamily(ctx, repository, RepositoryFamily{RepoIdentity: repository.Identity.RepoIdentity, PrimaryRoot: slot.PrimaryRoot}); err != nil {
			return nil, slot, err
		}
	}
	if c.Workspaces == nil {
		return nil, slot, errors.New("slot reconcile needs workspace readiness")
	}
	identified := candidate.Identity
	agentNow := func(ctx context.Context) EvidenceAgent { return c.slotAgentNow(ctx, identified) }
	result, err := c.Workspaces.Ensure(ctx, ProvisionRequest{Path: slot.PrimaryRoot, Slot: slot.Number, Progress: c.WorkspaceProgress,
		Agent: agentNow(ctx), AgentNow: agentNow})
	held := false
	var blocked *SlotReconcileError
	switch {
	case err != nil && toleratesHolds && errors.As(err, &blocked) && blocked.Failure.Class == FailureHold && blocked.Failure.Cause == StopReasonAgentLive:
		held = true
		if c.WorkspaceProgress != nil {
			fmt.Fprintln(c.WorkspaceProgress, err.Error())
		}
	case err != nil:
		return nil, slot, err
	case result.Warning != "" && c.WorkspaceProgress != nil:
		fmt.Fprintln(c.WorkspaceProgress, result.Warning) // degraded: the slot is usable
	}
	repository, err = c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return nil, slot, err
	}
	candidate, found = find(repository)
	if !found || (!held && (!candidate.Verified || candidate.Err != nil)) {
		return nil, slot, errors.New("slot host is not verified; inspect workspace before opening")
	}
	if resolveErr == nil && candidate.Identity != slot {
		return nil, slot, errors.New("slot identity changed during discovery")
	}
	slot = candidate.Identity
	if err := c.Threads.EnrollSlotRepository(ctx, repository); err != nil {
		return nil, slot, err
	}
	local := newSlotThreadStore(c.Namespace, slot)
	local.coordinator = c.Threads.coordinator
	return local, slot, nil
}

func (c *Couch) StartFreshSlot(ctx context.Context, path, agent string) (StartResult, error) {
	return c.startFreshSlot(ctx, path, agent, false, nil, nil)
}

// startFreshSlot archives the slot's current record (if any) and starts a fresh
// conversation in one slot journal. profile, when set, is a launch profile the
// caller already resolved before an irreversible step (reboot resolves it
// before quiescing), so it is not re-resolved here; it is distinct from
// accepted, which also triggers revalidateCreatedSlot.
func (c *Couch) startFreshSlot(ctx context.Context, path, agent string, requireEmpty bool, accepted *StartResolution, preflight *LaunchProfileResolution) (StartResult, error) {
	local, slot, err := c.selectedSlot(ctx, path)
	if err != nil {
		return StartResult{}, err
	}
	family, err := c.slotFamily(ctx, slot, true)
	if err != nil {
		return StartResult{}, err
	}
	cwd, err := ValidateFamilyPath(slot.WorktreeRoot, family.RelativeStart)
	if err != nil {
		return StartResult{}, err
	}
	old, err := local.observeSlotCurrent()
	if err != nil {
		return StartResult{}, err
	}
	if old.Unsupported {
		return StartResult{}, old.Err
	}
	if requireEmpty && old.Exists {
		return StartResult{}, ErrStartResolutionChanged
	}
	observation, err := c.ObserveSlotSessions(ctx, slot)
	if err != nil {
		return StartResult{}, err
	}
	if !observation.Absent {
		return StartResult{}, errors.New("slot has a live or unresolved owner; park the running conversation before starting fresh")
	}
	var profile LaunchProfileResolution
	if preflight != nil {
		profile = LaunchProfileResolution{Profile: cloneLaunchProfile(preflight.Profile), AgentSource: preflight.AgentSource, ArgvSource: preflight.ArgvSource}
	} else if accepted != nil {
		profile = LaunchProfileResolution{
			Profile:     cloneLaunchProfile(accepted.Profile),
			AgentSource: accepted.AgentSource,
			ArgvSource:  accepted.ArgvSource,
		}
	} else {
		profile, err = c.slotLaunchProfile(local, slot, cwd, agent)
		if err != nil {
			return StartResult{}, err
		}
	}
	if err := launcher.ValidateFreshAgentArgs(profile.Profile.Agent, profile.Profile.Argv); err != nil {
		return StartResult{}, err
	}
	scope, err := launcher.ResolveRepoScope(slot.WorktreeRoot)
	if err != nil {
		return StartResult{}, err
	}
	used := map[ThreadAddress]bool{}
	for _, candidate := range observation.Candidates {
		used[candidate.Address] = true
	}
	record, nonce, err := c.claimFreshRecord(ctx, freshClaimInput{
		ScopeKey: scope.Key, Cwd: cwd, RepoIdentity: slot.RepoIdentity, TagPrefix: filepath.Base(slot.PrimaryRoot),
		Profile: profile, Used: used, Store: local,
		Commit: func(record ThreadRecord) error { return local.replaceSlotCurrent(old, record) },
	})
	if err != nil {
		return StartResult{}, err
	}
	actor, handle, err := c.launchClaimedThread(claimedLaunch{
		Context: ctx, Thread: record, Nonce: nonce, StartedAt: c.Clock.Now(), Profile: profile,
		Args: StartArgs{Worktree: Worktree(slot.WorktreeRoot), Cwd: cwd},
		AfterPrepare: func() error {
			if err := c.verifyOtherSlotOwnersAbsent(ctx, slot, record.Address); err != nil {
				return err
			}
			if accepted != nil {
				return c.revalidateCreatedSlot(ctx, *accepted)
			}
			return nil
		},
	})
	return StartResult{Record: actor, Handle: handle}, err
}

func (c *Couch) slotLaunchProfile(local *ThreadStore, slot SlotIdentity, cwd, agent string) (LaunchProfileResolution, error) {
	preference, found, err := local.GetPathLaunchPreference(slot.RepoIdentity, cwd)
	if err != nil {
		return LaunchProfileResolution{}, err
	}
	var saved *PathLaunchPreference
	if found {
		saved = &preference
	}
	root := c.RootAgent
	if root == "" {
		root = "claude"
	}
	inputs := LaunchProfileInputs{ExplicitAgent: agent, Path: saved, RootAgent: root}
	selected, err := ResolveLaunchProfile(inputs)
	if err != nil {
		return selected, err
	}
	if !launcher.IsSupportedAgent(selected.Profile.Agent) {
		return selected, errors.New("unsupported slot launch agent")
	}
	if c.RepoAgentDefault != nil {
		scope, scopeErr := launcher.ResolveRepoScope(slot.WorktreeRoot)
		if scopeErr != nil {
			return selected, scopeErr
		}
		value, ok, err := c.repoLaunchDefault(cwd, slot.PrimaryRoot, selected.Profile.Agent, slot.RepoIdentity, scope.Key)
		if err != nil {
			return selected, err
		}
		if ok {
			inputs.RepoDefault = &value
		}
	}
	return ResolveLaunchProfile(inputs)
}

// OpenSlot resumes the current conversation, or reconstructs exactly one
// independently proved survivor. Missing metadata never requests a fresh agent.
// An agent passed here is the operator's own choice.
func (c *Couch) OpenSlot(ctx context.Context, path, agent string) (StartResult, error) {
	return c.openSlot(ctx, path, agent, false)
}

// openSlot is OpenSlot with the agent's provenance. agentGuessed means the
// agent was inferred (resume reads the slot's launch profile), not chosen: it
// may then adopt only a survivor whose proof checks the agent -- the native
// ledger binds per agent -- and never a record-less DETACHED survivor, whose
// proof (DetachedSessions) echoes the agent it is asked about and so cannot
// tell a right guess from a wrong one.
func (c *Couch) openSlot(ctx context.Context, path, agent string, agentGuessed bool) (StartResult, error) {
	local, slot, err := c.selectedSlot(ctx, path)
	if err != nil {
		return StartResult{}, err
	}
	observed, err := local.observeSlotCurrent()
	if err != nil {
		return StartResult{}, err
	}
	if observed.Unsupported {
		return StartResult{}, observed.Err
	}
	record := observed.Record
	if record != nil {
		if err := c.verifyOtherSlotOwnersAbsent(ctx, slot, record.Address); err != nil {
			return StartResult{}, err
		}
	}
	if record == nil {
		sessions, err := c.ObserveSlotSessions(ctx, slot)
		if err != nil {
			return StartResult{}, err
		}
		var survivors []ThreadRecord
		unprovenWarm := false
		for _, candidate := range sessions.Candidates {
			recovered := candidate.Record
			if recovered == nil {
				if agent == "" {
					continue
				}
				if agentGuessed && candidate.Presence == SessionPresent {
					unprovenWarm = true
					continue
				}
				family, err := c.slotFamily(ctx, slot, false)
				if err != nil {
					return StartResult{}, err
				}
				cwd, err := ValidateFamilyPath(slot.WorktreeRoot, family.RelativeStart)
				if err != nil {
					return StartResult{}, err
				}
				recovered = &ThreadRecord{SchemaVersion: ThreadSchemaVersion, Address: candidate.Address, StartingPath: cwd, WorkingPath: cwd, CreatedAt: c.Clock.Now(), Revision: 1, LatestLaunchProfile: &LaunchProfile{Agent: agent, Argv: []string{}}}
			}
			next := cloneThreadRecord(*recovered)
			next.Incarnations = nil
			next.Park = nil
			next.Continuation = nil
			next.Reservation = false
			next.Revision = 1
			if next.LatestLaunchProfile == nil {
				if agent == "" {
					continue
				}
				next.LatestLaunchProfile = &LaunchProfile{Agent: agent, Argv: []string{}}
			}
			proven := false
			if candidate.Presence == SessionPresent {
				resolver, ok := c.Artifacts.(DetachedSessionResolver)
				if !ok {
					return StartResult{}, errors.New("detached survivor observer unavailable")
				}
				proof, err := resolver.DetachedSessions(ctx, []DetachedCandidate{{Address: next.Address, Agent: next.LatestLaunchProfile.Agent, SessionName: recordSessionName(next)}})
				if err != nil {
					return StartResult{}, err
				}
				proven = detachedResumeProofMatches(next, proof)
				for _, actor := range c.actorRegistry().Records() {
					if actor.Thread == next.Address && actor.Args.WorkingDir() == next.WorkingPath && observeExactProcess(c.Proc, ProcessIdentity{PID: actor.PID, Identity: actor.Identity}) == Live {
						next.Incarnations = []ThreadIncarnation{{PID: actor.PID, Identity: actor.Identity, State: IncarnationLive, StartedAt: actor.StartedAt, RepoIdentity: slot.RepoIdentity, LaunchProfile: next.LatestLaunchProfile}}
						proven = true
					}
				}
			} else {
				resolver, ok := c.Artifacts.(NativeBindingResolver)
				if !ok {
					return StartResult{}, errors.New("native survivor observer unavailable")
				}
				binding, err := resolver.ResolveEstablished(ctx, next.Address.RepoScope, string(next.Address.Tag), next.LatestLaunchProfile.Agent)
				if err != nil && !isBindingDiagnostic(ResumeDiagnosticOf(err)) {
					return StartResult{}, err
				}
				proven = err == nil && coldResumeAuthorized(binding)
			}
			if proven {
				survivors = append(survivors, next)
			}
		}
		switch len(survivors) {
		case 0:
			if unprovenWarm {
				return StartResult{}, refuseResume(ResumeSurvivorUnproven, "slot current is unavailable and a managed session survives, but couch cannot prove which agent runs in it; attach to it with zellij or stop it, then resume again")
			}
			return StartResult{}, refuseResume(ResumeNoSurvivor, "slot current is unavailable and no running conversation could be proved its own")
		case 1:
		default:
			return StartResult{}, refuseResume(ResumeSurvivorsAmbiguous, fmt.Sprintf("slot current is unavailable and %d running conversations could each be its own; stop all but one managed session, then resume again", len(survivors)))
		}
		for _, candidate := range sessions.Candidates {
			active := candidate.Presence == SessionPresent
			for _, process := range candidate.Processes {
				active = active || observeExactProcess(c.Proc, process) != Dead
			}
			if active && candidate.Address != survivors[0].Address {
				return StartResult{}, errors.New("another managed slot owner survives; resolve all owners before reconstruction")
			}
			if candidate.Address == survivors[0].Address && len(survivors[0].Incarnations) == 0 {
				for _, process := range candidate.Processes {
					if observeExactProcess(c.Proc, process) != Dead {
						return StartResult{}, errors.New("survivor helper is not proved stopped or hosted by this Couch")
					}
				}
			}
		}
		if err := local.replaceSlotCurrent(observed, survivors[0]); err != nil {
			return StartResult{}, err
		}
		record = &survivors[0]
	}
	for _, actor := range c.actorRegistry().Records() {
		if actor.Thread == record.Address && actor.Args.WorkingDir() == record.WorkingPath && observeExactProcess(c.Proc, ProcessIdentity{PID: actor.PID, Identity: actor.Identity}) == Live {
			return StartResult{Record: actor}, nil
		}
	}
	actor, handle, err := c.ResumeContextWith(ctx, record.Address, ResumeOptions{})
	if err != nil {
		return StartResult{Record: actor, Handle: handle}, fmt.Errorf("open slot: %w", err)
	}
	return StartResult{Record: actor, Handle: handle}, nil
}

func (c *Couch) verifyOtherSlotOwnersAbsent(ctx context.Context, slot SlotIdentity, current ThreadAddress) error {
	observation, err := c.ObserveSlotSessions(ctx, slot)
	if err != nil {
		return err
	}
	for _, candidate := range observation.Candidates {
		if candidate.Address == current {
			continue
		}
		if candidate.Presence != SessionAbsent {
			return errors.New("previous slot session changed during readiness; inspect before retry")
		}
		for _, process := range candidate.Processes {
			if observeExactProcess(c.Proc, process) != Dead {
				return errors.New("previous slot process changed during readiness; inspect before retry")
			}
		}
	}
	return ctx.Err()
}

// conventionalSlotFromPath recognizes only an absolute conventional host path.
// It supplies a discovery location, never proof that the host belongs to Git.
func conventionalSlotFromPath(path string) (SlotIdentity, bool) {
	primary, number, ok := ParseSlotPath(path)
	if !ok {
		return SlotIdentity{}, false
	}
	slot := conventionalSlot(primary, number)
	return slot, slot.WorktreeRoot == path && slot.Validate() == nil
}

func (c *Couch) slotFamily(ctx context.Context, slot SlotIdentity, reserve bool) (RepositoryFamily, error) {
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return RepositoryFamily{}, err
	}
	requested := RepositoryFamily{RepoIdentity: repository.Identity.RepoIdentity, PrimaryRoot: slot.PrimaryRoot}
	if reserve {
		return c.Threads.ReserveRepositoryFamily(ctx, repository, requested)
	}
	return c.Threads.PreviewRepositoryFamily(ctx, repository, requested)
}
