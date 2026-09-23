package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
)

// SlotWorkspaceResolver supplies explicit-operation SDLC context. Listing never
// invokes this external probe.
type SlotWorkspaceResolver interface {
	ResolveWorkspace(context.Context, string) (WorkspaceIdentity, error)
}

func (c *OSSlotCatalog) ResolveWorkspace(ctx context.Context, path string) (WorkspaceIdentity, error) {
	if c == nil || c.IO == nil || ctx == nil {
		return WorkspaceIdentity{}, fmt.Errorf("workspace resolver unavailable")
	}
	physical, err := filepath.EvalSymlinks(NormalizePath(path))
	if err != nil {
		return WorkspaceIdentity{}, err
	}
	return NewWorkspaceProvisioner(c.IO).identity(ctx, physical)
}
func (f *SlotCatalogFake) ResolveWorkspace(ctx context.Context, path string) (WorkspaceIdentity, error) {
	if err := ctx.Err(); err != nil {
		return WorkspaceIdentity{}, err
	}
	if err := f.Errors[path]; err != nil {
		return WorkspaceIdentity{}, err
	}
	if id, ok := f.Workspaces[path]; ok {
		return id, nil
	}
	if repo, ok := f.Repositories[path]; ok {
		return repo.Identity, nil
	}
	return WorkspaceIdentity{}, fmt.Errorf("unknown workspace %s", path)
}
func (c *Couch) slotWorkspace(ctx context.Context, path string) (WorkspaceIdentity, error) {
	resolver, ok := c.Slots.(SlotWorkspaceResolver)
	if !ok {
		return WorkspaceIdentity{}, fmt.Errorf("workspace context resolver unavailable")
	}
	return resolver.ResolveWorkspace(ctx, path)
}

// resolveSlotInput resolves references without publishing any local state. A
// nil target denotes a primary or ordinary path; identity retains its context.
func (c *Couch) resolveSlotInput(ctx context.Context, input string) (*ThreadTarget, WorkspaceIdentity, error) {
	ref, explicit, err := ParseWorkspaceReference(input)
	if err != nil {
		return nil, WorkspaceIdentity{}, err
	}
	path := input
	var id WorkspaceIdentity
	if explicit {
		id, err = c.slotWorkspace(ctx, ".")
		if err != nil {
			if ref.Repo == "" {
				return nil, id, err
			}
			primary, e := c.enrolledPrimary(ref.Repo)
			if e != nil {
				return nil, id, e
			}
			repository, e := c.Slots.Discover(ctx, primary)
			if e != nil {
				return nil, id, e
			}
			id = repository.Identity
		}
		if ref.Repo == "" && id.Kind == "dependency" {
			return nil, id, fmt.Errorf("bare workspace references are ambiguous from a dependency; use repo:N")
		}
		primary := id.PrimaryRoot
		if ref.Repo != "" {
			primary = filepath.Join(id.FleetRoot, ref.Repo)
		}
		repository, e := c.Slots.Discover(ctx, primary)
		if e != nil {
			return nil, id, e
		}
		id = repository.Identity
		if ref.Number == 0 {
			return nil, id, nil
		}
		for _, candidate := range repository.Slots {
			if candidate.Identity.Number == ref.Number {
				target := ThreadTarget{Kind: ThreadTargetSlot, Slot: candidate.Identity}
				return &target, id, nil
			}
		}
		return nil, id, fmt.Errorf("slot %s does not exist; create another slot explicitly", ref.String())
	}
	id, err = c.slotWorkspace(ctx, path)
	if err != nil {
		if conventional, ok := conventionalSlotFromPath(NormalizePath(path)); ok {
			repository, e := c.Slots.Discover(ctx, conventional.PrimaryRoot)
			if e != nil {
				return nil, id, e
			}
			for _, candidate := range repository.Slots {
				if candidate.Identity.WorktreeRoot == conventional.WorktreeRoot {
					target := ThreadTarget{Kind: ThreadTargetSlot, Slot: candidate.Identity}
					return &target, repository.Identity, nil
				}
			}
		}
	}
	if err != nil && workspaceRepoName(input) && input != "." {
		contextID, contextErr := c.slotWorkspace(ctx, ".")
		if contextErr == nil {
			path = filepath.Join(contextID.FleetRoot, input)
			id, err = c.slotWorkspace(ctx, path)
		} else if primary, e := c.enrolledPrimary(input); e == nil {
			id, err = c.slotWorkspace(ctx, primary)
		}
	}
	if err != nil {
		return nil, id, err
	}
	if id.Kind == "slot" {
		slot, e := SlotIdentityFromWorkspace(id)
		if e != nil {
			return nil, id, e
		}
		target := ThreadTarget{Kind: ThreadTargetSlot, Slot: slot}
		return &target, id, nil
	}
	return nil, id, nil
}

// enrolledPrimary resolves a human repository name from retained locations only.
// No lifecycle facts or directory-wide machine scan are involved.
func (c *Couch) enrolledPrimary(repo string) (string, error) {
	if c.Threads == nil {
		return "", fmt.Errorf("no enrolled repository named %q", repo)
	}
	view := *c.Threads
	view.readOnly = true
	var roots []string
	err := view.withPreviewLock(func() error {
		manifest, _, _, err := view.loadManifestLocked()
		roots = append(roots, manifest.SlotRepositories...)
		return err
	})
	if err != nil {
		return "", err
	}
	found := ""
	for _, root := range roots {
		if filepath.Base(root) == repo {
			if found != "" && found != root {
				return "", fmt.Errorf("repository %q is ambiguous; use an absolute path", repo)
			}
			found = root
		}
	}
	if found == "" {
		return "", fmt.Errorf("no enrolled repository named %q; open its absolute path first", repo)
	}
	return found, nil
}

// WorkspaceReferencePath resolves canonical references to physical workspace
// identity. Opaque thread tags remain the native reference resolver's concern.
func (c *Couch) WorkspaceReferencePath(ctx context.Context, ref string) (string, bool, error) {
	_, recognized, err := ParseWorkspaceReference(ref)
	if err != nil || !recognized {
		return "", recognized, err
	}
	if c.Slots == nil {
		return "", false, nil
	}
	target, identity, err := c.resolveSlotInput(ctx, ref)
	if err != nil {
		return "", true, err
	}
	if target != nil {
		return target.Slot.WorktreeRoot, true, nil
	}
	return identity.PrimaryRoot, true, nil
}
