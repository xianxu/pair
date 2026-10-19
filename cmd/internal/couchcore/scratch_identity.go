package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

// ScratchSlotIdentity names the slot's disposable temp storage. RepoIdentity is
// retained repository authority, so equal repository basenames cannot share it.
type ScratchSlotIdentity struct {
	Repo, RepoIdentity string
	Number             int
}

// ErrScratchAuthorityMissing permits a verified first-workspace fallback;
// malformed or ambiguous retained authority must never take that fallback.
var ErrScratchAuthorityMissing = errors.New("scratch slot has no retained repository authority")

// ScratchSlot classifies an already-retained launch record without consulting
// Git or requiring its checkout to remain on disk. It uses enrollment/family
// authority and the record's exact checkout scope, never the child's working
// directory (which can be / during a warm attach to a missing checkout).
func (s *ThreadStore) ScratchSlot(ctx context.Context, record ThreadRecord) (ScratchSlotIdentity, error) {
	if s == nil {
		return ScratchSlotIdentity{}, errors.New("scratch slot requires thread store")
	}
	if err := ValidateThreadRecord(record); err != nil {
		return ScratchSlotIdentity{}, err
	}
	root := s.familyRootStore()
	view := *root
	view.readOnly = true
	manifest, err := view.slotRepositoryManifestContext(ctx)
	if err != nil {
		return ScratchSlotIdentity{}, err
	}
	roots := make(map[string]string, len(manifest.SlotRepositoryIdentities)+len(manifest.RepositoryFamilies))
	for root, identity := range manifest.SlotRepositoryIdentities {
		roots[root] = identity
	}
	for _, family := range manifest.RepositoryFamilies {
		roots[family.PrimaryRoot] = family.RepoIdentity
	}
	var result ScratchSlotIdentity
	found := false
	for root, identity := range roots {
		checkout, number := root, 0
		if slot, ok := slotForContainedPath(root, record.StartingPath, identity); ok {
			checkout, number = slot.WorktreeRoot, slot.Number
		}
		_, belongs, err := RecordCheckoutMembership(record, identity, checkout)
		if err != nil {
			return ScratchSlotIdentity{}, err
		}
		if !belongs {
			continue
		}
		candidate := ScratchSlotIdentity{Repo: filepath.Base(root), RepoIdentity: identity, Number: number}
		if !workspaceRepoName(candidate.Repo) {
			return ScratchSlotIdentity{}, errors.New("scratch slot repository name is invalid")
		}
		if found && result != candidate {
			return ScratchSlotIdentity{}, errors.New("scratch slot has ambiguous retained repository authority")
		}
		result, found = candidate, true
	}
	if !found {
		return ScratchSlotIdentity{}, fmt.Errorf("%w for %s:%s", ErrScratchAuthorityMissing, record.Address.RepoScope, record.Address.Tag)
	}
	return result, nil
}
