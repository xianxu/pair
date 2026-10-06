package couchcore

import (
	"context"
	"fmt"
)

// SlotCatalogFake retains mutable repository snapshots and discovery failures.
// Tests may change its state between calls; returned slices are detached copies.
type SlotCatalogFake struct {
	Workspaces   map[string]WorkspaceIdentity
	Repositories map[string]SlotRepository
	Errors       map[string]error
	// Leftovers names slot numbers per repository known only from leftovers
	// (a stale registration or a resting branch): Discover returns them as
	// candidates wrapping ErrSlotNeedsReconcile, as OSSlotCatalog does.
	Leftovers map[string][]int
	Calls     []string
}

func (f *SlotCatalogFake) Discover(ctx context.Context, path string) (SlotRepository, error) {
	if ctx == nil {
		return SlotRepository{}, fmt.Errorf("slot discovery requires context")
	}
	if err := ctx.Err(); err != nil {
		return SlotRepository{}, err
	}
	f.Calls = append(f.Calls, path)
	if err := f.Errors[path]; err != nil {
		return SlotRepository{}, err
	}
	r, ok := f.Repositories[path]
	if !ok {
		return SlotRepository{}, fmt.Errorf("unknown repository %s", path)
	}
	r.Slots = append([]SlotCandidate(nil), r.Slots...)
	for _, n := range f.Leftovers[path] {
		slot := conventionalSlot(path, n)
		slot.RepoIdentity = r.Identity.RepoIdentity
		r.Slots = append(r.Slots, SlotCandidate{Identity: slot, Err: fmt.Errorf("%w: only leftovers remain", ErrSlotNeedsReconcile)})
	}
	return r, nil
}
