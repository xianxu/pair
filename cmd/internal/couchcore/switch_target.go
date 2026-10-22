package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// ResolveAgentSwitchTarget selects a retained thread by its verified workspace.
// The evidence-free projection is used only for address selection; lifecycle
// admission and proof of vacancy remain with PrepareAgentSwitch.
func (c *Couch) ResolveAgentSwitchTarget(ctx context.Context, target string) (ThreadAddress, WorkspaceIdentity, error) {
	if ctx == nil || c == nil || c.Threads == nil || c.Slots == nil {
		return ThreadAddress{}, WorkspaceIdentity{}, errors.New("switch-agent target services unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, err
	}
	ref, recognized, err := ParseWorkspaceReference(target)
	if err != nil || !recognized || ref.Repo == "" {
		return ThreadAddress{}, WorkspaceIdentity{}, errors.New("switch-agent requires an exact repo:N target")
	}
	// A named target needs only its own current identity. Whole-repository
	// discovery here repeats every peer slot's SDLC probes during admission
	// and again at authorization, exceeding the bounded broker exchange.
	fleetRoot := ""
	if current, contextErr := c.slotWorkspace(ctx, "."); contextErr == nil {
		fleetRoot = current.FleetRoot
	}
	if err := ctx.Err(); err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, err
	}
	primary, err := c.repositoryPrimary(ref.Repo, fleetRoot)
	if err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, err
	}
	path := primary
	if ref.Number != 0 {
		// Layout derives a location, never authority. Resolve and validate that
		// exact candidate below, including its Git-backed primary identity.
		path = NewSlotLayout(primary, "", ref.Number).Host()
	}
	identity, err := c.slotWorkspace(ctx, path)
	if err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, err
	}
	if err := identity.validate(); err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, fmt.Errorf("switch-agent target workspace: %w", err)
	}
	if identity.WorktreeRoot != path || identity.PrimaryRoot != primary || identity.Slot == nil || *identity.Slot != ref.Number || ref.Number == 0 && identity.Kind != "primary" {
		return ThreadAddress{}, WorkspaceIdentity{}, errors.New("switch-agent target workspace does not match the requested slot")
	}
	if ref.Number != 0 {
		if _, err := SlotIdentityFromWorkspace(identity); err != nil {
			return ThreadAddress{}, WorkspaceIdentity{}, fmt.Errorf("switch-agent target slot: %w", err)
		}
	}
	snapshot, err := c.Threads.PreviewSnapshotContext(ctx)
	if err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, err
	}
	address, err := selectAgentSwitchTarget(snapshot, ref.Number, path)
	if err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, err
	}
	if err := ctx.Err(); err != nil {
		return ThreadAddress{}, WorkspaceIdentity{}, err
	}
	return address, identity, nil
}

func selectAgentSwitchTarget(snapshot ThreadSnapshot, number int, path string) (ThreadAddress, error) {
	// A partial snapshot may retain a readable record alongside its backend
	// error. Projecting that record must never discard the selection refusal.
	for _, slot := range snapshot.Slots {
		if slot.Identity.WorktreeRoot == path && slot.Err != nil {
			return ThreadAddress{}, fmt.Errorf("switch-agent retained target is unreadable: %w", slot.Err)
		}
	}
	if number == 0 {
		// An unreadable primary record has no trustworthy StartingPath, so the
		// normal primary-row filter cannot decide whether it competes here.
		scope, err := launcher.ResolveRepoScope(path)
		if err != nil {
			return ThreadAddress{}, err
		}
		for _, unreadable := range snapshot.Unreadable {
			if unreadable.RepoScope == scope.Key {
				return ThreadAddress{}, fmt.Errorf("switch-agent retained target is unreadable: %s/%s", unreadable.RepoScope, unreadable.Tag)
			}
		}
	}
	row, err := SelectSlotRow(ProjectActionableThreads(FromSnapshot(snapshot, nil)), number, path)
	if err != nil {
		return ThreadAddress{}, err
	}
	if row.Reason == ReasonUnreadable {
		return ThreadAddress{}, errors.New("switch-agent retained target is unreadable")
	}
	if row.Address == (ThreadAddress{}) {
		return ThreadAddress{}, &SlotOperationError{Code: SlotOpNoThread, Detail: "no retained Couch thread stands for " + path}
	}
	return row.Address, nil
}
