package couchcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// hubRuntime owns one immutable domain snapshot. It exists only for explicit
// directory launches; nil retains the legacy implicit-cwd path.
type hubRuntime struct {
	mu    sync.RWMutex
	state couchcore.HubOwnership
}

func (h *hubRuntime) snapshot() couchcore.HubOwnership {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.state
}
func prepareExplicitHubs(ctx context.Context, c *couchcore.Couch, paths []string) (*hubRuntime, error) {
	resolver, ok := c.Slots.(couchcore.SlotWorkspaceResolver)
	if !ok {
		return nil, fmt.Errorf("hub workspace resolver unavailable")
	}
	defs := make([]couchcore.HubDefinition, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		root, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			return nil, err
		}
		id, err := resolver.ResolveWorkspace(ctx, root)
		if err != nil {
			return nil, fmt.Errorf("hub %s: %w", path, err)
		}
		if id.Slot == nil || id.WorktreeRoot != root {
			return nil, fmt.Errorf("hub %s must name a primary or numbered slot directory", path)
		}
		doc, err := couchcore.ParseHubClaims(nil)
		if len(paths) > 1 {
			doc, err = ReadHubClaimFile(root)
		}
		if err != nil {
			return nil, err
		}
		defs = append(defs, couchcore.HubDefinition{Root: root, Workspace: id.FleetRoot, Slot: couchcore.HubSlot{Repository: id.PrimaryRoot, Number: *id.Slot}, Claims: doc})
	}
	state, err := couchcore.NewHubOwnership(defs)
	if err != nil {
		return nil, err
	}
	return &hubRuntime{state: state}, nil
}
func hubWorkspaceSlot(id couchcore.WorkspaceIdentity) (couchcore.HubSlot, bool) {
	if id.Slot == nil {
		return couchcore.HubSlot{}, false
	}
	slot := couchcore.HubSlot{Repository: id.PrimaryRoot, Number: *id.Slot}
	return slot, slot.Validate() == nil
}
func (h *hubRuntime) owner(id couchcore.WorkspaceIdentity) string {
	state := h.snapshot()
	if state.IsHub(id.WorktreeRoot) {
		return id.WorktreeRoot
	}
	slot, ok := hubWorkspaceSlot(id)
	if !ok {
		return ""
	}
	return state.Resolve(slot).Hub
}
func (h *hubRuntime) target(name string) (string, error) {
	var matches []string
	for _, hub := range h.snapshot().Hubs() {
		address := filepath.Base(hub.Slot.Repository) + ":" + strconv.Itoa(hub.Slot.Number)
		if name == hub.Root || name == address || name == filepath.Base(hub.Root) {
			matches = append(matches, hub.Root)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("target hub %q is absent or ambiguous; use its exact hub slot address", name)
	}
	return matches[0], nil
}

// Extra hubs join the existing serial operation queue and background adoption
// path, preserving the first hub's focus and the single reattach pass.
func startExplicitHubs(console *couchtty.Console, hubs *hubRuntime) {
	for _, hub := range hubs.snapshot().Hubs()[1:] {
		root := hub.Root
		finish := func(_ any, err error) {
			if err != nil {
				console.Notify(couchtty.Notice{Kind: "status", Body: fmt.Sprintf("hub %s failed: %v; open that hub in the switcher to retry", root, err)})
			}
		}
		err := console.EnqueueRemoteOperation("hub-start:"+root, "start-hub", func(ctx context.Context) (couchcore.OperationCall, error) {
			return couchcore.OperationCall{Name: "start-hub", Args: map[string]string{"path": root}}, ctx.Err()
		}, func() {}, finish)
		if err != nil {
			finish(nil, err)
		}
	}
}

func (s *messageService) sameHubBindings(from, to couchmessage.Binding) bool {
	s.mu.Lock()
	a, aok := s.workspaces[from]
	b, bok := s.workspaces[to]
	s.mu.Unlock()
	if !aok || !bok {
		return false
	}
	owner := s.authority.owner(a)
	return owner != "" && owner == s.authority.owner(b)
}
