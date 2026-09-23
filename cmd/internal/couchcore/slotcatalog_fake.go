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
	Calls        []string
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
	return r, nil
}
