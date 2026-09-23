package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func (c *Couch) resolveManagedStart(ctx context.Context, args StartArgs) (StartResolution, error) {
	action := args.Action
	if action == "" {
		action = StartOpen
	}
	if action != StartOpen && action != StartCreate && action != StartFresh {
		return StartResolution{}, fmt.Errorf("unknown start action %q", action)
	}
	original := args.WorkingDir()
	if original == "" {
		original = "."
	}
	target, id, err := c.resolveSlotInput(ctx, original)
	if err != nil {
		return StartResolution{}, err
	}
	if _, explicit, _ := ParseWorkspaceReference(original); explicit && action == StartCreate {
		action = StartOpen
	}
	if id.Kind != "primary" && id.Kind != "slot" {
		return c.resolveOrdinaryStartResolution(ctx, args)
	}
	primary := id.PrimaryRoot
	if action == StartCreate {
		repository, e := c.Slots.Discover(ctx, primary)
		if e != nil {
			return StartResolution{}, e
		}
		snapshot, e := c.Threads.PreviewSnapshot()
		if e != nil {
			return StartResolution{}, e
		}
		occupied := len(repository.Slots) > 0
		scopes := slotRepositoryScopes(repository)
		for _, record := range snapshot.Records {
			if scopes[record.Address.RepoScope] {
				occupied = true
			}
		}
		for _, address := range snapshot.Unreadable {
			if scopes[address.RepoScope] {
				return StartResolution{}, fmt.Errorf("repository has an unreadable conversation; recover it before creating another slot")
			}
		}
		if occupied {
			allocation, e := SelectNewSlot(repository.Slots)
			if e != nil {
				return StartResolution{}, e
			}
			slot := conventionalSlot(primary, allocation.Number)
			slot.RepoIdentity = repository.Identity.RepoIdentity
			selected := ThreadTarget{Kind: ThreadTargetSlot, Slot: slot}
			target = &selected
		} else {
			target = nil
		}
	}
	path := primary
	if target != nil {
		path = target.Slot.WorktreeRoot
	}
	var resolution StartResolution
	if target == nil && (strings.ContainsAny(original, `/\`) || original == ".") {
		resolution, err = c.resolveOrdinaryStartResolution(ctx, args)
	} else {
		resolution, err = c.resolveStartProfile(args, path, Worktree(path), id.RepoIdentity, primary)
	}
	if err != nil {
		return StartResolution{}, err
	}
	resolution.OriginalInput, resolution.Action = original, action
	if target != nil {
		resolution.Target = *target
	} else if action == StartFresh {
		return StartResolution{}, errors.New("fresh action requires an existing numbered slot")
	}
	resolution.Fingerprint = fingerprintStartResolution(resolution)
	return resolution, nil
}

func slotRepositoryScopes(repository SlotRepository) map[string]bool {
	scopes := map[string]bool{}
	paths := []string{repository.Identity.PrimaryRoot}
	for _, slot := range repository.Slots {
		paths = append(paths, slot.Identity.WorktreeRoot)
	}
	for _, path := range paths {
		scope, err := launcher.ResolveRepoScope(path)
		if err == nil {
			scopes[scope.Key] = true
		}
	}
	return scopes
}
func (c *Couch) checkSlotCreation(ctx context.Context, repository SlotRepository) error {
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return err
	}
	scopes := slotRepositoryScopes(repository)
	blockers := []string{}
	for _, row := range rows {
		belongs := scopes[row.Address.RepoScope] || (row.Target.Kind == ThreadTargetSlot && row.Target.Slot.RepoIdentity == repository.Identity.RepoIdentity)
		if !belongs {
			continue
		}
		if row.State == ThreadParked {
			blockers = append(blockers, row.Label())
		}
		if row.State == ThreadUnusable && (row.Reason == ReasonUnreadable || row.Reason == ReasonUnknown) {
			return fmt.Errorf("repository ownership needs attention at %s before creating another slot", row.Label())
		}
	}
	if len(blockers) > 0 {
		return fmt.Errorf("resume or resolve parked work before creating another slot: %s", strings.Join(blockers, ", "))
	}
	return nil
}

func (c *Couch) spawnManagedResolution(ctx context.Context, resolution StartResolution) (StartResult, error) {
	if resolution.Target.Kind != ThreadTargetSlot {
		return StartResult{}, errors.New("managed launch requires a slot target")
	}
	slot := resolution.Target.Slot
	switch resolution.Action {
	case StartFresh:
		return c.StartFreshSlot(ctx, slot.WorktreeRoot, resolution.RequestedAgent)
	case StartOpen:
		return c.OpenSlot(ctx, slot.WorktreeRoot, resolution.RequestedAgent)
	case StartCreate:
	default:
		return StartResult{}, fmt.Errorf("invalid slot start action %q", resolution.Action)
	}
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return StartResult{}, err
	}
	for _, candidate := range repository.Slots {
		if candidate.Identity.Number == slot.Number {
			return StartResult{}, ErrStartResolutionChanged
		}
	}
	if err := c.checkSlotCreation(ctx, repository); err != nil {
		return StartResult{}, err
	}
	if c.Workspaces == nil {
		return StartResult{}, errors.New("workspace readiness unavailable")
	}
	result, err := c.Workspaces.Ensure(ctx, ProvisionRequest{Path: slot.PrimaryRoot, Slot: slot.Number, Progress: c.WorkspaceProgress})
	if err != nil {
		return StartResult{}, err
	}
	if result.SchemaVersion != 1 || filepath.Clean(result.Path) != slot.WorktreeRoot || result.Address != (WorkspaceReference{Repo: slot.Repo, Number: slot.Number}).String() {
		return StartResult{}, errors.New("created workspace does not match accepted slot")
	}
	// A competing creator may have won after preview. Do not adopt its result as
	// our new slot or silently increment the number.
	if result.Disposition != "created" {
		return StartResult{}, fmt.Errorf("slot appeared during creation; open %s explicitly", result.Address)
	}
	repository, err = c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return StartResult{}, err
	}
	if err = c.Threads.EnrollSlotRepository(ctx, repository); err != nil {
		return StartResult{}, err
	}
	if err = c.checkSlotCreation(ctx, repository); err != nil {
		return StartResult{}, err
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return StartResult{}, err
	}
	record, handle, err := c.spawnResolved(ctx, resolution, rows)
	return StartResult{Record: record, Handle: handle}, err
}

// Recheck profile, physical identity and repository parked work after setup;
// the accepted number is fixed even though discovery now sees the new directory.
func (c *Couch) revalidateCreatedSlot(ctx context.Context, accepted StartResolution) error {
	slot := accepted.Target.Slot
	identity, err := c.slotWorkspace(ctx, slot.WorktreeRoot)
	if err != nil {
		return err
	}
	currentSlot, err := SlotIdentityFromWorkspace(identity)
	if err != nil {
		return err
	}
	if currentSlot != slot {
		return ErrStartResolutionChanged
	}
	current, err := c.resolveStartProfile(StartArgs{Stack: accepted.RequestedAgent, Issue: accepted.Issue}, slot.WorktreeRoot, Worktree(slot.WorktreeRoot), slot.RepoIdentity, slot.PrimaryRoot)
	if err != nil {
		return err
	}
	current.Action, current.OriginalInput, current.Target = accepted.Action, accepted.OriginalInput, accepted.Target
	if fingerprintStartResolution(current) != accepted.Fingerprint {
		return ErrStartResolutionChanged
	}
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return err
	}
	return c.checkSlotCreation(ctx, repository)
}
