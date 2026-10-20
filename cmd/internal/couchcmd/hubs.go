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

// hubRuntime owns a hub domain snapshot and serializes its process effects.
// Explicit launches populate ownership; implicit launches use only the effect
// gate and retain legacy cwd authority separately.
type hubRuntime struct {
	effectOnce sync.Once
	effects    chan struct{}
	mu         sync.RWMutex
	state      couchcore.HubOwnership
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
	return hubOwner(h.snapshot(), id)
}
func hubOwner(state couchcore.HubOwnership, id couchcore.WorkspaceIdentity) string {
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

// Extra hubs join the existing operation queue and background adoption
// path, preserving the first hub's focus and the single reattach pass.
func startExplicitHubs(console *couchtty.Console, hubs *hubRuntime, couches ...*couchcore.Couch) {
	console.SetHubOwnership(hubs.snapshot())
	enqueue := console.EnqueueRemoteOperation
	if len(couches) > 0 && couches[0] != nil {
		enqueue = newLifecycleRemoteQueue(console, couches[0]).Queue
	}
	queueExplicitHubs(hubs.snapshot().Hubs()[1:], enqueue, func(root string, err error) {
		console.Notify(couchtty.Notice{Kind: "status", Body: fmt.Sprintf("hub failed; open it in the switcher to retry: %s: %v", root, err)})
	})
}

// Only enqueue the next hub after the prior one has finished adoption. Queue
// worker scheduling cannot reorder the operator's explicit launch sequence.
func queueExplicitHubs(hubs []couchcore.HubDefinition, enqueue func(string, string, func(context.Context) (couchcore.OperationCall, error), func(), func(any, error)) error, report func(string, error)) {
	var next func(int)
	next = func(index int) {
		if index >= len(hubs) {
			return
		}
		root := hubs[index].Root
		finish := func(_ any, err error) {
			if err != nil {
				report(root, err)
			}
			next(index + 1)
		}
		err := enqueue("hub-start:"+root, "start-hub", func(ctx context.Context) (couchcore.OperationCall, error) {
			return couchcore.OperationCall{Name: "start-hub", Args: map[string]string{"path": root}}, ctx.Err()
		}, func() {}, finish)
		if err != nil {
			finish(nil, err)
		}
	}
	next(0)
}

func (s *messageService) sameHubBindings(from, to couchmessage.Binding) bool {
	s.mu.Lock()
	a, aok := s.workspaces[from]
	b, bok := s.workspaces[to]
	s.mu.Unlock()
	if !aok || !bok {
		return false
	}
	state := s.authority.hubs.snapshot()
	owner := hubOwner(state, a)
	return owner != "" && owner == hubOwner(state, b)
}

// Equal-specificity conflicts are warned once at startup, including broad
// claims for future slots. Different specificity is an intentional override.
func hubConflictWarnings(state couchcore.HubOwnership) []string {
	seen := map[couchcore.HubSlot]string{}
	selfOwned := map[couchcore.HubSlot]bool{}
	for _, hub := range state.Hubs() {
		selfOwned[hub.Slot] = true
	}
	var warnings []string
	for _, hub := range state.Hubs() {
		for _, claim := range hub.Claims.Claims() {
			key := couchcore.HubSlot{Repository: filepath.Join(hub.Workspace, claim.Repository), Number: claim.Slot}
			if selfOwned[key] {
				continue
			}
			if first := seen[key]; first != "" && first != hub.Root {
				if len(warnings) < couchcore.MaxHubSlots {
					name := key.Repository
					if key.Number >= 0 {
						name += ":" + strconv.Itoa(key.Number)
					}
					warnings = append(warnings, fmt.Sprintf("hub claim conflict for %s: %s wins before %s; edit .hub claims to change ownership", name, first, hub.Root))
				}
			} else {
				seen[key] = hub.Root
			}
		}
	}
	return warnings
}

// The existing queue has several workers. Multi-hub ownership changes and
// process mutations share this cancellable gate so ownership cannot change
// halfway through an effect. The prepared caller guard runs inside it.
func dispatchHubOperation(h *hubRuntime, call couchcore.OperationCall, dispatch couchcore.OperationExecutor) (any, error) {
	if h == nil {
		return dispatch(call)
	}
	op, ok := Resolve(call.Name)
	if !ok && call.Name != "start-hub" || ok && op.Effect != couchcore.EffectProcess && call.Name != "move-slot" {
		return dispatch(call)
	}
	ctx := call.Context
	if ctx == nil {
		ctx = context.Background()
	}
	release, err := holdHubEffect(ctx, h)
	if err != nil {
		return nil, err
	}
	defer release()
	payload, err := unwrapHubMutation(ctx, call.TypedPayload)
	if err != nil {
		return nil, err
	}
	call.TypedPayload = payload
	return dispatch(call)
}

// holdHubEffect also fences bounded system-message admission against an
// ownership move. It never holds the ownership snapshot mutex across IO.
func holdHubEffect(ctx context.Context, h *hubRuntime) (func(), error) {
	if h == nil {
		return func() {}, ctx.Err()
	}
	h.effectOnce.Do(func() { h.effects = make(chan struct{}, 1) })
	select {
	case h.effects <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		<-h.effects
		return nil, err
	}
	return func() { <-h.effects }, nil
}
