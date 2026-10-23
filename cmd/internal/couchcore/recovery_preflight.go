package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// prepareRecoveryLaunch checks existing launch prerequisites before retirement,
// then holds the source's existing retention registration through preservation
// and launch. The caller finishes it with the operation's actual success.
func (c *Couch) prepareRecoveryLaunch(ctx context.Context, record ThreadRecord, profile LaunchProfile) (string, func(bool) error, error) {
	if ctx == nil {
		return "", nil, errors.New("recovery launch preparation requires context")
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if c == nil || c.Threads == nil {
		return "", nil, errors.New("recovery launch preparation requires thread store")
	}
	if !c.workingPathExists(record) {
		return "", nil, errors.New("recovery requires its saved working directory")
	}
	if err := launcher.ValidateFreshAgentArgs(profile.Agent, profile.Argv); err != nil {
		return "", nil, err
	}
	if c.SwitchLaunchCheck != nil {
		if err := c.SwitchLaunchCheck(profile.Agent); err != nil {
			return "", nil, err
		}
	}
	repo, err := c.recoveryLaunchRepoIdentity(ctx, record)
	if err != nil {
		return "", nil, err
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	finish, err := c.beginResumeRetention(ctx, record.Address, true)
	if err != nil {
		return "", nil, err
	}
	return repo, finish, nil
}

// recoveryLaunchProfile preserves the conversation's saved configuration.
// Only explicit Fresh may use normal path/root/default precedence when that
// conversation no longer has a saved profile. Resolution is a preview read.
func (c *Couch) recoveryLaunchProfile(ctx context.Context, record ThreadRecord, kind RecoverySelectionKind) (LaunchProfile, error) {
	if ctx == nil {
		return LaunchProfile{}, errors.New("recovery launch profile requires context")
	}
	if err := ctx.Err(); err != nil {
		return LaunchProfile{}, err
	}
	if record.LatestLaunchProfile != nil {
		return cloneLaunchProfile(*record.LatestLaunchProfile), nil
	}
	if kind != RecoveryFresh {
		return LaunchProfile{}, errors.New("native recovery requires the saved agent configuration")
	}
	if c == nil || c.Threads == nil {
		return LaunchProfile{}, errors.New("recovery launch profile requires thread store")
	}
	return c.freshRecoveryLaunchProfile(ctx, record, "")
}

func (c *Couch) recoveryLaunchProfileForSelection(ctx context.Context, record ThreadRecord, selection RecoverySelection) (LaunchProfile, error) {
	if selection.Agent == "" {
		return c.recoveryLaunchProfile(ctx, record, selection.Kind)
	}
	if selection.Kind != RecoveryFresh {
		return LaunchProfile{}, errors.New("native resume cannot change its agent")
	}
	return c.freshRecoveryLaunchProfile(ctx, record, selection.Agent)
}

func (c *Couch) freshRecoveryLaunchProfile(ctx context.Context, record ThreadRecord, agent string) (LaunchProfile, error) {
	store := *c.Threads
	store.readOnly = true
	backend, err := store.storeForAddressContext(ctx, record.Address)
	if err != nil {
		return LaunchProfile{}, err
	}
	if backend.slot != nil {
		selected, err := c.slotLaunchProfile(ctx, backend, *backend.slot, record.WorkingPath, agent)
		if err != nil {
			return LaunchProfile{}, err
		}
		if err := ctx.Err(); err != nil {
			return LaunchProfile{}, err
		}
		return cloneLaunchProfile(selected.Profile), nil
	}
	selected, err := c.resolveOrdinaryStartResolutionWithStore(ctx, StartArgs{Worktree: Worktree(record.StartingPath), Cwd: record.WorkingPath, Stack: agent}, &store)
	if err != nil {
		return LaunchProfile{}, err
	}
	if err := ctx.Err(); err != nil {
		return LaunchProfile{}, err
	}
	return cloneLaunchProfile(selected.Profile), nil
}

// recoveryLaunchRepoIdentity may plan a numbered host repair from the primary's
// verified catalog identity. It never repairs or infers identity from a record
// alone; the actual checkout must match this common directory after repair.
func (c *Couch) recoveryLaunchRepoIdentity(ctx context.Context, record ThreadRecord) (string, error) {
	repo, hostErr := c.resolveRepoIdentity(ctx, record.WorkingPath)
	if hostErr == nil {
		return repo, nil
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	refuse := func(cause error) (string, error) {
		return "", errors.Join(hostErr, fmt.Errorf("verify numbered recovery repository: %w", cause))
	}
	if c.Slots == nil {
		return refuse(errors.New("slot catalog unavailable"))
	}
	view := *c.Threads
	view.readOnly = true
	backend, err := view.storeForAddressContext(ctx, record.Address)
	if err != nil {
		return refuse(err)
	}
	if !backend.layout.Local || backend.slot == nil {
		return refuse(errors.New("record does not name a numbered workspace"))
	}
	slot := *backend.slot
	if err := slot.Validate(); err != nil {
		return refuse(err)
	}
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return refuse(err)
	}
	if err := repository.Identity.validate(); err != nil {
		return refuse(err)
	}
	if repository.Identity.Kind != "primary" || repository.Identity.PrimaryRoot != slot.PrimaryRoot || repository.Identity.RepoIdentity != slot.RepoIdentity || repository.Identity.Repo != slot.Repo {
		return refuse(errors.New("primary repository identity differs from numbered workspace"))
	}
	found := false
	for _, candidate := range repository.Slots {
		if candidate.Identity == slot {
			found = true
			break
		}
	}
	if !found {
		return refuse(errors.New("numbered workspace is absent from verified primary catalog"))
	}
	if _, belongs, err := RecordCheckoutMembership(record, slot.RepoIdentity, slot.WorktreeRoot); err != nil || !belongs {
		return refuse(errors.New("record does not belong to numbered workspace"))
	}
	relative, err := RelativeFamilyPath(slot.WorktreeRoot, record.WorkingPath)
	if err != nil {
		return refuse(err)
	}
	if _, err := ValidateFamilyPath(slot.WorktreeRoot, relative); err != nil {
		return refuse(err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return slot.RepoIdentity, nil
}
