package couchcmd

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func (s *messageService) installNoticeDelivery(console *couchtty.Console, startup *couchcore.StartupObservation) {
	if !console.NoticeDeliveryInstalled() {
		console.SetNoticeDelivery(newNoticeDelivery(console))
	}
	publish := console.Warn
	s.noticeOutput.Store(&publish)
	boot := map[string]bool{}
	if startup != nil {
		for root, hub := range startup.Hubs {
			if hub.Survived {
				boot[root] = true
			}
		}
	}
	if console.ReattachPassArmed() {
		boot[s.authority.hub] = true
	}
	s.mu.Lock()
	// A wrapper already admitted during service startup needs no pending witness.
	for _, identity := range s.workspaces {
		delete(boot, identity.WorktreeRoot)
	}
	s.noticeBoot = boot
	s.mu.Unlock()
	console.BindNoticeDelivery(s)
}

// publishSystemNotice is a bounded memory handoff in production. It is safe for
// registry callbacks; broker IO belongs to the delivery owner, never this call.
func (s *messageService) publishSystemNotice(n couchtty.Notice) {
	if publish := s.noticeOutput.Load(); publish != nil && *publish != nil {
		(*publish)(n)
	}
}

func (s *messageService) NoticeOwner(o couchcore.NoticeDeliveryOccurrence) couchcore.NoticeDeliveryOwner {
	return s.resolveNoticeOwner(o)
}

func (s *messageService) resolveNoticeOwner(o couchcore.NoticeDeliveryOccurrence) couchcore.NoticeDeliveryOwner {
	root := o.OwnerRoot
	var identity couchcore.WorkspaceIdentity
	known := false
	pendingIdentity := false
	s.mu.Lock()
	if connected := s.connected.Load(); connected != nil {
		for binding, live := range *connected {
			if _, committed := s.workspaces[binding]; live && !committed && (o.Thread == (couchcore.ThreadAddress{}) || bindingThread(binding) == o.Thread) {
				pendingIdentity = true
			}
		}
	}
	for binding, workspace := range s.workspaces {
		if o.Thread != (couchcore.ThreadAddress{}) && bindingThread(binding) == o.Thread || o.Thread == (couchcore.ThreadAddress{}) && o.SourceRoot != "" && workspace.WorktreeRoot == o.SourceRoot {
			identity, known = workspace, true
			break
		}
	}
	s.mu.Unlock()
	if known {
		root = s.authority.owner(identity)
	} else if o.SourceSlot != nil && s.authority.hubs != nil {
		if o.SourceSlot.Validate() != nil {
			return couchcore.NoticeDeliveryOwner{State: couchcore.NoticeOwnerUnavailable}
		}
		root = s.authority.hubs.snapshot().Resolve(*o.SourceSlot).Hub
	} else if root == "" && o.SourceRoot != "" && s.authority.hubs != nil && s.authority.hubs.snapshot().IsHub(o.SourceRoot) {
		root = o.SourceRoot
	} else if root == "" && o.Thread == (couchcore.ThreadAddress{}) && o.SourceRoot == "" {
		root = s.authority.hub // explicitly global diagnostic
	} else if root == "" && s.authority.hubs == nil {
		root = s.authority.hub // single-hub topology
	}
	hub, connected := s.hubBindingAt(root)
	if connected {
		registered := false
		for _, binding := range s.broker.Connected() {
			if binding == hub {
				registered = true
				break
			}
		}
		connected = registered
	}
	if !connected {
		s.mu.Lock()
		booting := s.noticeBoot[root]
		s.mu.Unlock()
		// Registry publication precedes EffectConnect's identity/broker commit.
		// That in-flight witness is unknown readiness, never proof of absence.
		if booting || pendingIdentity {
			return couchcore.NoticeDeliveryOwner{State: couchcore.NoticeOwnerBooting, Root: root}
		}
		// Availability is independent of automation generation. Retain the
		// resolved owner's epoch after disconnect so a queued occurrence can
		// distinguish owner loss from an intervening OFF/ON revocation.
		_, epoch := s.hubAutomationPermit(root)
		return couchcore.NoticeDeliveryOwner{State: couchcore.NoticeOwnerUnavailable, Epoch: epoch, Root: root}
	}
	s.mu.Lock()
	delete(s.noticeBoot, root) // actual connection ends this startup witness
	s.mu.Unlock()
	on, epoch := s.hubAutomationPermit(root)
	state := couchcore.NoticeOwnerDisabled
	if on {
		state = couchcore.NoticeOwnerEnabled
	}
	return couchcore.NoticeDeliveryOwner{State: state, Binding: lifecycleBindingKey(hub), Epoch: epoch, Root: root}
}

// DeliverNotice preserves the exact recipient and automation generation frozen
// by the reducer. Ownership moves serialize through the existing effect gate.
func (s *messageService) DeliverNotice(ctx context.Context, effect couchcore.NoticeDeliveryEffect) couchcore.NoticeDeliveryResult {
	release, err := holdHubEffect(ctx, s.authority.hubs)
	if err != nil {
		return couchcore.NoticeDeliveryRefused
	}
	defer release()
	current := s.NoticeOwner(effect.Occurrence)
	if current != effect.Owner || current.State != couchcore.NoticeOwnerEnabled || ctx.Err() != nil {
		return couchcore.NoticeDeliveryRefused
	}
	var hub couchmessage.Binding
	for _, binding := range s.broker.Connected() {
		if lifecycleBindingKey(binding) == effect.Owner.Binding {
			hub = binding
			break
		}
	}
	if hub.Slot == "" {
		return couchcore.NoticeDeliveryRefused
	}
	permit := s.captureHubInputPermit(hub)
	if permit.epoch != effect.Owner.Epoch || !permit.current(s) {
		return couchcore.NoticeDeliveryRefused
	}
	body := notifyosc.Sanitize([]byte(effect.Occurrence.Body))
	var receipt couchmessage.Receipt
	if coordinator := s.supervision.Load(); coordinator != nil {
		receipt, err = coordinator.sendGenerated(ctx, hub, hub, effect.AdmissionID, body, false, permit)
	} else {
		receipt, err = s.sendAutomaticSystem(ctx, hub, hub, effect.AdmissionID, body, permit)
	}
	if err == nil {
		return couchcore.NoticeDeliveryAdmitted
	}
	if receipt.Message.ID != "" {
		return couchcore.NoticeDeliveryUncertain
	}
	if _, known, _ := s.broker.ReceiptSnapshot(effect.AdmissionID); known {
		return couchcore.NoticeDeliveryUncertain
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return couchcore.NoticeDeliveryUncertain
	}
	return couchcore.NoticeDeliveryRefused
}

func (s *messageService) NoticeReceipt(id string) (bool, bool) {
	receipt, known, delivering := s.broker.ReceiptSnapshot(id)
	return known, known && receipt.Status.Terminal() && !delivering
}
