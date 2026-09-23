package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
)

// prepareTrackedWorkspace runs repeatable numbered-workspace readiness while
// the existing start claim reserves this process incarnation. Callers retain
// rollback and final conversation proof: compile may change external evidence.
func (c *Couch) prepareTrackedWorkspace(ctx context.Context, thread ThreadRecord, nonce string, warm bool) error {
	if warm {
		return nil
	}
	if ctx == nil {
		return errors.New("workspace readiness requires context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil || c.Threads == nil {
		return errors.New("workspace readiness requires thread store")
	}
	backend, err := c.Threads.storeForAddress(thread.Address)
	if err != nil {
		return err
	}
	if !backend.layout.Local {
		return nil
	}
	slot := backend.slot
	if slot == nil {
		return errors.New("local workspace has no slot identity")
	}
	if err := backend.validateLocalOrigin(thread); err != nil {
		return err
	}
	claim, err := exactStartIncarnation(&thread, nonce)
	if err != nil {
		return err
	}
	if claim.Start.LaunchProfile == nil || claim.PID != 0 || claim.Identity != "" || claim.RepoIdentity == "" {
		return errors.New("workspace readiness requires the unlaunched slot start claim and profile")
	}
	recheck := func() error {
		current, err := c.Threads.GetThread(thread.Address)
		if err != nil {
			return err
		}
		if current.Revision != thread.Revision {
			return &ThreadRevisionError{Address: thread.Address, Want: thread.Revision, Got: current.Revision}
		}
		currentClaim, err := exactStartIncarnation(&current, nonce)
		if err != nil {
			return err
		}
		if current.StartingPath != thread.StartingPath || current.WorkingPath != thread.WorkingPath || currentClaim.RepoIdentity != claim.RepoIdentity || currentClaim.PID != claim.PID || currentClaim.Identity != claim.Identity || !reflect.DeepEqual(currentClaim.Start, claim.Start) {
			return errors.New("slot start claim changed during workspace readiness")
		}
		return nil
	}
	if err := recheck(); err != nil {
		return err
	}
	if c.Workspaces == nil {
		return errors.New("numbered workspace readiness is unavailable")
	}
	result, err := c.Workspaces.Ensure(ctx, ProvisionRequest{Path: slot.PrimaryRoot, Slot: slot.Number, Progress: c.WorkspaceProgress})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if result.SchemaVersion != 1 || result.Address != fmt.Sprintf("%s:%d", slot.Repo, slot.Number) || result.Path != slot.WorktreeRoot {
		return errors.New("workspace readiness returned a different slot")
	}
	physical, err := filepath.EvalSymlinks(result.Path)
	if err != nil {
		return err
	}
	if physical != slot.WorktreeRoot {
		return errors.New("workspace readiness changed physical checkout identity")
	}
	repoIdentity, err := c.resolveRepoIdentity(ctx, result.Path)
	if err != nil {
		return err
	}
	// Filesystem-only routing reconstructs conventional locations, not the Git
	// common-dir identity (which may be outside primary/.git). The start claim
	// retains the identity proved before readiness; Ensure checks membership.
	if repoIdentity != claim.RepoIdentity {
		return errors.New("workspace readiness changed repository identity")
	}
	return recheck()
}
