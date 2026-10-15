package couchcore

import (
	"context"
	"errors"

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
	repo, err := c.resolveRepoIdentity(ctx, record.WorkingPath)
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
	view := *c
	store := *c.Threads
	store.readOnly = true
	view.Threads = &store
	backend, err := store.storeForAddressContext(ctx, record.Address)
	if err != nil {
		return LaunchProfile{}, err
	}
	if backend.slot != nil {
		selected, err := view.slotLaunchProfile(backend, *backend.slot, record.WorkingPath, "")
		if err != nil {
			return LaunchProfile{}, err
		}
		if err := ctx.Err(); err != nil {
			return LaunchProfile{}, err
		}
		return cloneLaunchProfile(selected.Profile), nil
	}
	selected, err := view.resolveOrdinaryStartResolution(ctx, StartArgs{Worktree: Worktree(record.StartingPath), Cwd: record.WorkingPath})
	if err != nil {
		return LaunchProfile{}, err
	}
	if err := ctx.Err(); err != nil {
		return LaunchProfile{}, err
	}
	return cloneLaunchProfile(selected.Profile), nil
}
