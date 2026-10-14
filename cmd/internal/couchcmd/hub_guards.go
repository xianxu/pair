package couchcmd

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// The caller's existing role and incarnation checks remain at the handler.
// This extra restriction derives both owners from one current snapshot.
func authorizeHubMutation(caller couchcore.WorkspaceIdentity, target couchcore.HubSlot, state couchcore.HubOwnership) error {
	own := ""
	if state.IsHub(caller.WorktreeRoot) {
		own = caller.WorktreeRoot
	} else if slot, ok := hubWorkspaceSlot(caller); ok {
		own = state.Resolve(slot).Hub
	}
	targetOwner := state.Resolve(target).Hub
	if own == "" || own != targetOwner {
		return fmt.Errorf("slot belongs to hub %s; ask that hub to manage it", targetOwner)
	}
	return nil
}
func (s *messageService) authorizeHubBinding(ctx context.Context, caller, target couchmessage.Binding) error {
	if s.authority.hubs == nil {
		return nil
	}
	origin, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return err
	}
	dest, err := s.connectedWorkspace(ctx, target)
	if err != nil {
		return err
	}
	slot, ok := hubWorkspaceSlot(dest)
	if !ok {
		return fmt.Errorf("target slot identity unavailable")
	}
	return authorizeHubMutation(origin, slot, s.authority.hubs.snapshot())
}

func hubTargetResolver(c *couchcore.Couch) func(context.Context, string) (couchcore.HubSlot, error) {
	return func(ctx context.Context, target string) (couchcore.HubSlot, error) {
		path, recognized, err := c.WorkspaceReferencePath(ctx, target)
		if err != nil {
			return couchcore.HubSlot{}, err
		}
		if !recognized {
			return couchcore.HubSlot{}, fmt.Errorf("target must name repo:N")
		}
		resolver, ok := c.Slots.(couchcore.SlotWorkspaceResolver)
		if !ok {
			return couchcore.HubSlot{}, fmt.Errorf("target workspace resolver unavailable")
		}
		id, err := resolver.ResolveWorkspace(ctx, path)
		if err != nil {
			return couchcore.HubSlot{}, err
		}
		slot, ok := hubWorkspaceSlot(id)
		if !ok {
			return couchcore.HubSlot{}, fmt.Errorf("target slot identity unavailable")
		}
		return slot, nil
	}
}
func (s *messageService) handleHubSlotOperation(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request) couchmessage.Response {
	if s.authority.hubs == nil || r.Op == "operation-status" {
		return s.slotOps.handle(ctx, caller, r)
	}
	guard := func(ctx context.Context) error {
		if _, err := s.liveCaller(ctx, r); err != nil {
			return err
		}
		root, err := s.authority.thread(ctx, caller)
		if err != nil {
			return err
		}
		origin, err := s.authority.workspace(ctx, root)
		if err != nil {
			return err
		}
		if s.authority.hubTarget == nil {
			return fmt.Errorf("target slot resolver unavailable")
		}
		target, err := s.authority.hubTarget(ctx, r.Target)
		if err != nil {
			return err
		}
		return authorizeHubMutation(origin, target, s.authority.hubs.snapshot())
	}
	if err := guard(ctx); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	return s.slotOps.handle(ctx, caller, r, guard)
}

func (s *messageService) hubReceiptFilter(ctx context.Context, identity couchcore.WorkspaceIdentity) func(couchmessage.OperationReceipt) bool {
	return func(receipt couchmessage.OperationReceipt) bool {
		if s.authority.hubs == nil {
			return true
		}
		if s.authority.hubTarget == nil {
			return false
		}
		target, err := s.authority.hubTarget(ctx, receipt.Target)
		return err == nil && authorizeHubMutation(identity, target, s.authority.hubs.snapshot()) == nil
	}
}

// hubMutationPayload is a supervisor-local capability, never decoded from wire
// data. The effect gate unwraps it only after revalidating queued authority.
type hubMutationPayload struct {
	original any
	guard    func(context.Context) error
}

func unwrapHubMutation(ctx context.Context, payload any) (any, error) {
	if held, ok := payload.(hubMutationPayload); ok {
		if held.guard == nil {
			return nil, fmt.Errorf("mutation authority guard unavailable")
		}
		if err := held.guard(ctx); err != nil {
			return nil, err
		}
		return held.original, nil
	}
	return payload, nil
}
