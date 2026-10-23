package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// hubMoveCall is an owner-local capability, never populated from wire JSON.
// dispatchHubOperation's effect gate orders it against agent mutations.
type hubMoveCall struct {
	service *messageService
	caller  couchmessage.Binding
	move    couchcore.HubMove
	persist bool
}
type hubMoveService struct {
	enqueue func(string, string, func(context.Context) (couchcore.OperationCall, error), func(), func(any, error)) error
	publish func(couchcore.HubOwnership)
	files   hubClaimFiles
}

func (s *messageService) installHubMoves(console *couchtty.Console, couches ...*couchcore.Couch) {
	if s.authority.hubs == nil {
		return
	}
	enqueue := console.EnqueueRemoteOperation
	if len(couches) > 0 && couches[0] != nil {
		enqueue = newLifecycleRemoteQueue(console, couches[0]).Queue
	}
	s.hubMoves.Store(&hubMoveService{enqueue: enqueue, publish: console.SetHubOwnership})
	console.SetHubOwnership(s.authority.hubs.snapshot())
	for _, warning := range hubConflictWarnings(s.authority.hubs.snapshot()) {
		console.Notify(couchtty.Notice{Kind: "status", Body: warning})
	}
}
func (s *messageService) handleHubMove(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	ctx, cancel := context.WithTimeout(ctx, slotAdmissionBudget)
	defer cancel()
	refuse := func(err error) couchmessage.Response {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		return refuse(err)
	}
	service := s.hubMoves.Load()
	if service == nil || s.slotOps == nil || s.authority.hubs == nil || s.authority.hubTarget == nil {
		return refuse(errors.New("hub moves unavailable; start Couch with explicit hub directories"))
	}
	caller, err := s.switchCaller(ctx, r)
	if ctx.Err() != nil {
		return admissionRefusal("caller", ctx.Err())
	}
	if err != nil {
		return refuse(err)
	}
	identity, err := s.connectedWorkspace(ctx, caller)
	if ctx.Err() != nil {
		return admissionRefusal("caller workspace", ctx.Err())
	}
	if err != nil {
		return refuse(err)
	}
	if !s.authority.isHub(identity) {
		return refuse(errors.New("only a verified hub may move slots; ask your hub"))
	}
	// A confirmed earlier effect may have moved the slot out of this caller's
	// group. Authenticate the caller, then answer its retained exact intent
	// before evaluating authority for a new effect.
	s.slotOps.mu.Lock()
	s.slotOps.sweepLocked()
	previous, held := s.slotOps.receipts[receiptKey{caller.Scope, caller.Tag, r.ID}]
	s.slotOps.mu.Unlock()
	if held {
		_, err := couchmessage.ApplyReceiptEvent(previous, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptAdmit, ID: r.ID, Op: r.Op, Target: r.Target, ToHub: r.ToHub, Persist: r.Persist})
		if err != nil {
			return couchmessage.Response{Code: "id-conflict", Error: err.Error()}
		}
		return couchmessage.Response{Code: "accepted", Operation: &previous}
	}
	target, err := s.authority.hubs.target(r.ToHub)
	if err != nil {
		return refuse(err)
	}
	state := s.authority.hubs.snapshot()
	move := couchcore.HubMove{Caller: identity.WorktreeRoot, Target: target, Revision: state.Revision()}
	// Admission names the slots from warm sources inside its budget
	// (pair#726); the queued job re-resolves each one fresh and refuses if any
	// moved, so the move's authority is still current before its effect.
	names := strings.Split(r.Target, ",")
	for _, name := range names {
		slot, err := s.warmHubTarget(ctx, name)
		if errors.Is(err, errSlotCatalogWarming) {
			return hubAdmissionError(admissionPhase("target", err))
		}
		if ctx.Err() != nil {
			return admissionRefusal("target", ctx.Err())
		}
		if err != nil {
			return refuse(admissionPhase("target", err))
		}
		move.Slots = append(move.Slots, slot)
	}
	if _, err := state.Apply(move); err != nil {
		return refuse(err)
	}
	payload := hubMoveCall{service: s, caller: caller, move: move, persist: r.Persist}
	return s.slotOps.admit(ctx, caller, r, func(key string, started func(), _ couchcore.OperationProgress, finished func(any, error)) error {
		return service.enqueue(key, "move-slot", func(ctx context.Context) (couchcore.OperationCall, error) {
			call := couchcore.OperationCall{Name: "move-slot", TypedPayload: payload, FailureTarget: &couchcore.LifecycleFailureTarget{Target: r.Target}}
			if _, err := s.authority.current(ctx, caller); err != nil {
				return call, err
			}
			if !s.isConnected(caller) {
				return call, errors.New("hub disconnected; retry from its current session")
			}
			for i, name := range names {
				fresh, err := s.authority.hubTarget(ctx, name)
				if err != nil {
					return call, err
				}
				if fresh != move.Slots[i] {
					return call, fmt.Errorf("slot %s resolved differently since admission; nothing was moved; retry", name)
				}
			}
			return call, nil
		}, started, finished)
	}, "hub-move")
}
func executeHubMove(call couchcore.OperationCall) (any, error) {
	payload, ok := call.TypedPayload.(hubMoveCall)
	if !ok || payload.service == nil {
		return nil, errors.New("move-slot requires the authenticated console capability")
	}
	s := payload.service
	service := s.hubMoves.Load()
	if service == nil {
		return nil, errors.New("hub moves unavailable")
	}
	h := s.authority.hubs
	before := h.snapshot()
	state, err := h.applyMove(call.Context, payload.move, payload.persist, service.files, func() error {
		if !s.isConnected(payload.caller) {
			return errors.New("hub disconnected; retry from its current session")
		}
		_, err := s.authority.current(call.Context, payload.caller)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, slot := range payload.move.Slots {
		if before.Resolve(slot).Hub != state.Resolve(slot).Hub {
			if publisher := s.lifecycle.Load(); publisher != nil {
				publisher.ownershipChanged(payload.move.Target)
			}
			break
		}
	}
	if service.publish != nil {
		service.publish(state)
	}
	return nil, nil
}
func (h *hubRuntime) applyMove(ctx context.Context, move couchcore.HubMove, persist bool, files hubClaimFiles, verify func() error) (couchcore.HubOwnership, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	fail := func(err error) (couchcore.HubOwnership, error) { return h.state, err }
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := verify(); err != nil {
		return fail(err)
	}
	next, err := h.state.Apply(move)
	if err != nil {
		return fail(err)
	}
	if persist {
		defs := h.state.Hubs()
		for i := range defs {
			doc, err := ReadHubClaimFile(defs[i].Root)
			if err != nil {
				return fail(err)
			}
			defs[i].Claims = doc
		}
		edits, err := couchcore.PlanHubClaimEdits(defs, move.Slots, move.Target)
		if err != nil {
			return fail(err)
		}
		if err := verify(); err != nil {
			return fail(err)
		}
		// Replanning visible bytes can omit an earlier rename whose directory
		// sync failed. Retain every unchanged snapshot as a confirmation-only
		// edit, so no persistent success bypasses durability reconciliation.
		confirm := append([]couchcore.HubClaimEdit(nil), edits...)
		for _, def := range defs {
			changed := false
			for _, edit := range edits {
				if edit.Root == def.Root {
					changed = true
					break
				}
			}
			if !changed {
				raw := def.Claims.Bytes()
				confirm = append(confirm, couchcore.HubClaimEdit{Root: def.Root, Before: raw, After: raw})
			}
		}
		if err := files.Apply(ctx, confirm); err != nil {
			return fail(err)
		}
		for i := range defs {
			for _, edit := range edits {
				if edit.Root == defs[i].Root {
					defs[i].Claims, err = couchcore.ParseHubClaims(edit.After)
					if err != nil {
						return fail(err)
					}
				}
			}
		}
		next, err = next.WithClaims(defs, next.Revision())
		if err != nil {
			return fail(fmt.Errorf("claims saved but projection failed: %w", err))
		}
	}
	postCheck := ctx.Err()
	if postCheck == nil {
		postCheck = verify()
	}
	if postCheck != nil {
		if persist {
			return fail(fmt.Errorf("claims saved but runtime ownership unchanged: %w; inspect claims and retry", postCheck))
		}
		return fail(postCheck)
	}
	h.state = next
	return next, nil
}
