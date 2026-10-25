package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// Admission uses the exact binding's verified workspace and canonical target;
// current-world workspace inspection remains on the owned operation queue.
func (s *messageService) authorizeParkAdmission(ctx context.Context, caller couchmessage.Binding, target couchcore.HubSlot) error {
	if s.authority.hubs == nil {
		return nil
	}
	origin, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return fmt.Errorf("recovery caller inventory is unavailable; refresh the switcher inventory, inspect the row and retry: %w", err)
	}
	return authorizeHubMutation(origin, target, s.authority.hubs.snapshot())
}

func (s *messageService) authenticateRecoveryCaller(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request) error {
	current, err := s.switchCaller(ctx, r)
	if err != nil {
		return err
	}
	if err := s.authority.process(current); err != nil {
		return err
	}
	if current != caller {
		return fmt.Errorf("recovery confirmer binding generation changed")
	}
	_, err = s.liveCaller(ctx, r)
	return err
}

func (s *messageService) handleHubSlotOperation(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request) couchmessage.Response {
	if r.Op == "recovery-confirmation" {
		if s.slotOps.reference == nil {
			return couchmessage.Response{Code: "unsupported", Error: "recovery confirmation unavailable"}
		}
		s.slotOps.mu.Lock()
		s.slotOps.sweepLocked()
		receipt, held := s.slotOps.receipts[receiptKey{caller.Scope, caller.Tag, r.ID}]
		s.slotOps.mu.Unlock()
		if held {
			if receipt.Op != r.Op || receipt.RecoveryConfirmation != r.RecoveryConfirmation {
				return couchmessage.Response{Code: "id-conflict", Error: "receipt ID already names another operation"}
			}
			r.Target = receipt.Target
		} else {
			var err error
			r.Target, err = s.slotOps.reference(couchcore.WithRecoveryConfirmer(ctx, recoveryActor(caller)), r.RecoveryConfirmation)
			if err != nil {
				return couchmessage.Response{Code: "refused", Error: err.Error()}
			}
		}
	}
	if couchcore.IsBenchOperation(r.Op) {
		// No hub configuration means no .hub claim to prove (pair#847).
		if s.authority.hubs == nil {
			return couchmessage.Response{Code: "refused", Error: r.Op + " needs a hub configuration: start Couch with the bench hub as an explicit hub"}
		}
		if _, err := slotOperationOptions(r); err != nil {
			return couchmessage.Response{Code: "refused", Error: err.Error()}
		}
	}
	if (s.authority.hubs == nil && r.Op != "recovery-confirmation" && r.Op != couchcore.OpRecoverPark) || r.Op == "operation-status" {
		return s.slotOps.handle(ctx, caller, r)
	}
	// One check, two sources (pair#726). The queued guard proves authority on
	// fresh discovery in prepare and again at the effect gate. Admission is a
	// cheap gate on warm sources (the caller's connected workspace, the
	// remembered target slot), so a stale memo can only admit a job that then
	// refuses before any effect.
	guard := func(ctx context.Context) error {
		return s.checkHubSlotOperation(ctx, caller, r, hubSlotSources{
			origin: func(ctx context.Context) (couchcore.WorkspaceIdentity, error) {
				root, err := s.authority.thread(ctx, caller)
				if err != nil {
					return couchcore.WorkspaceIdentity{}, err
				}
				return s.authority.workspace(ctx, root)
			},
			target: func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
				if s.authority.hubTarget == nil {
					return couchcore.HubSlot{}, fmt.Errorf("target slot resolver unavailable")
				}
				return s.authority.hubTarget(ctx, ref)
			},
			phase: func(_ string, err error) error { return err },
		})
	}
	admission := func(ctx context.Context) error {
		return s.checkHubSlotOperation(ctx, caller, r, hubSlotSources{
			origin: func(ctx context.Context) (couchcore.WorkspaceIdentity, error) {
				return s.admissionCallerWorkspace(ctx, caller)
			},
			target: s.warmHubTarget,
			phase:  admissionPhase,
		})
	}
	if r.Op == couchcore.OpRecoverPark {
		admission = func(ctx context.Context) error {
			return admissionPhase("caller", s.authenticateRecoveryCaller(ctx, caller, r))
		}
	}
	err := admission(ctx)
	if err != nil {
		return hubAdmissionError(err)
	}
	if ctx.Err() != nil {
		return admissionRefusal("target", ctx.Err())
	}
	return s.slotOps.handle(ctx, caller, r, guard)
}

// hubSlotSources are where a hub slot check gets the caller's workspace and
// the target slot, and how it labels a failing step.
type hubSlotSources struct {
	origin func(context.Context) (couchcore.WorkspaceIdentity, error)
	target func(context.Context, string) (couchcore.HubSlot, error)
	phase  func(string, error) error
}

// checkHubSlotOperation is the one hub slot-operation check: a live caller,
// then the caller's and the target's hub must match.
func (s *messageService) checkHubSlotOperation(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request, src hubSlotSources) error {
	if r.Op == "recovery-confirmation" || r.Op == couchcore.OpRecoverPark {
		if err := s.authenticateRecoveryCaller(ctx, caller, r); err != nil {
			return src.phase("caller", err)
		}
	} else if _, err := s.liveCaller(ctx, r); err != nil {
		return src.phase("caller", err)
	}
	if s.authority.hubs == nil {
		if couchcore.IsBenchOperation(r.Op) {
			return src.phase("bench slot", errors.New("no hub configuration to prove a .hub claim"))
		}
		return nil
	}
	origin, err := src.origin(ctx)
	if err != nil {
		return src.phase("caller workspace", err)
	}
	target, err := src.target(ctx, r.Target)
	if err != nil {
		return src.phase("target", err)
	}
	state := s.authority.hubs.snapshot()
	if couchcore.IsBenchOperation(r.Op) {
		if err := state.Provisions(origin.WorktreeRoot, target); err != nil {
			return src.phase("bench slot", err)
		}
		return nil
	}
	return authorizeHubMutation(origin, target, state)
}

// slotAdmissionBudget covers the complete lifecycle request, starting before
// authentication and ending at enqueue (or a status reply). Nested admission
// phases inherit this deadline; queued execution gets its own queue context.
// Staying below the transport's 2 s exchange leaves time for a typed refusal
// instead of a reply the server drops (pair#726).
const slotAdmissionBudget = 1500 * time.Millisecond

// admissionPhase names the admission step that refused or ran out of time.
func admissionPhase(phase string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("admission ran out of its %s budget in the %s check; nothing was queued; retry", slotAdmissionBudget, phase)
	}
	return fmt.Errorf("%s: %w", phase, err)
}

func admissionRefusal(phase string, err error) couchmessage.Response {
	return couchmessage.Response{Code: "refused", Error: admissionPhase(phase, err).Error()}
}

// admissionCallerWorkspace reads only the exact connected registration.
// A context does not bound native resolver teardown, so admission must never
// fall back to workspace inspection when this cached proof is missing.
func (s *messageService) admissionCallerWorkspace(ctx context.Context, caller couchmessage.Binding) (couchcore.WorkspaceIdentity, error) {
	origin, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return couchcore.WorkspaceIdentity{}, fmt.Errorf("caller workspace unavailable; reconnect this Couch slot and retry; nothing was queued: %w", err)
	}
	return origin, nil
}

// Target resolutions run only in service-owned memo flights or queued proofs.
func (s *messageService) warmHubTarget(ctx context.Context, ref string) (couchcore.HubSlot, error) {
	if s.hubTargets == nil {
		return couchcore.HubSlot{}, errors.New("slot catalog unavailable; reconnect or restart Couch and retry; nothing was queued")
	}
	return s.hubTargets.Warm(ctx, ref)
}

func hubAdmissionError(err error) couchmessage.Response {
	if errors.Is(err, errSlotCatalogWarming) {
		return couchmessage.Response{Code: couchmessage.CodeWarming, RetryAfterMillis: 500, Error: err.Error()}
	}
	return couchmessage.Response{Code: "refused", Error: err.Error()}
}

func (s *messageService) hubReceiptFilter(ctx context.Context, identity couchcore.WorkspaceIdentity) func(couchmessage.OperationReceipt) bool {
	return func(receipt couchmessage.OperationReceipt) bool {
		if s.authority.hubs == nil {
			return true
		}
		target, err := s.warmHubTarget(ctx, receipt.Target)
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
