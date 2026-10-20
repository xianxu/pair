package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"path/filepath"
	"strconv"
	"time"
)

func ptrDrainGuard(f func(couchcore.OperationCall) error) *func(couchcore.OperationCall) error {
	return &f
}

func (s *messageService) installDrainService() error {
	d, err := newDrainCoordinator(s.lifetime, drainCoordinatorOptions{Broker: s.broker, Queue: sdlcDrainQueue{workspace: s.authority.hub}, Store: newDrainStore(s.authority.drainNamespace), Now: s.authority.now, Snapshot: s.drainSnapshot, Endpoint: func(b couchmessage.Binding) couchmessage.DrainEndpoint {
		return messageEndpoint{service: s, binding: b, endpoint: s.authority.endpoint(b)}
	}, Publish: s.authority.drainPublish, Census: s.authority.drainCensus})
	if err != nil {
		return err
	}
	s.drains.Store(d)
	if h := s.authority.drainGate; h != nil {
		h.drainGuard.Store(ptrDrainGuard(func(call couchcore.OperationCall) error {
			b := d.view()
			control := drainControlPlaneCall(h.snapshot(), call)
			if s.authority.hubs == nil {
				control = (call.Name == "start" || call.Name == "start-hub" || call.Name == "resume") && call.Args["path"] == s.authority.hub && s.authority.hub != ""
			}
			if b.ID == "" || b.State == couchcore.DrainCancelled || control {
				return nil
			}
			return couchmessage.ErrDraining
		}))
		if s.authority.drainWrap != nil {
			s.authority.drainWrap(h)
		}
	}
	return nil
}

// Control-plane recovery is restricted to the immutable first command-line hub.
func drainControlPlaneCall(h couchcore.HubOwnership, call couchcore.OperationCall) bool {
	if call.Name != "resume" && call.Name != "start" && call.Name != "start-hub" {
		return false
	}
	if call.Args["path"] == h.Main() && h.Main() != "" {
		return true
	}
	defs := h.Hubs()
	if len(defs) == 0 {
		return false
	}
	ref := filepath.Base(defs[0].Slot.Repository) + ":" + strconv.Itoa(defs[0].Slot.Number)
	return call.Args["ref"] == ref
}

func (s *messageService) authorizeDrainSteward(ctx context.Context, caller couchmessage.Binding) error {
	id, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return err
	}
	main := s.authority.hub
	if s.authority.hubs != nil {
		main = s.authority.hubs.snapshot().Main()
	}
	if main == "" || id.WorktreeRoot != main {
		return errors.New("only the first command-line hub steward may start or cancel a drain; ask the steward")
	}
	_, err = s.authority.current(ctx, caller)
	return err
}

func (s *messageService) handleDrain(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	d := s.drains.Load()
	if r.Drain.Action == "status" {
		b := couchcore.DrainBoard{State: couchcore.DrainCancelled, Detail: "no active drain"}
		if d != nil {
			b = d.view()
		}
		return couchmessage.Response{Code: "ok", Drain: &b}
	}
	if d == nil {
		return couchmessage.Response{Code: "unsupported", Error: "fleet drain unavailable; restart Couch on a drain-capable build"}
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if r.Drain.Action != "ack" {
		if err = s.authorizeDrainSteward(ctx, caller); err != nil {
			return couchmessage.Response{Code: "refused", Error: err.Error()}
		}
	}
	var b couchcore.DrainBoard
	switch r.Drain.Action {
	case "ack":
		m, ok := s.drainSnapshot()[caller.Slot]
		if !ok || m.Binding != caller {
			return couchmessage.Response{Code: "refused", Error: "drain acknowledgement requires the current admitted incarnation"}
		}
		b, err = d.ack(ctx, r.Drain.ID, caller.Slot, m.Identity())
	case "start", "cancel":
		gate := s.authority.drainGate
		if gate == nil {
			gate = s.authority.hubs
		}
		release, e := holdHubEffect(ctx, gate)
		if e != nil {
			return couchmessage.Response{Code: "refused", Error: e.Error()}
		}
		defer release()
		if err = s.authorizeDrainSteward(ctx, caller); err == nil {
			if r.Drain.Action == "start" {
				b, err = d.start(ctx, r.ID, r.Drain.Reason, r.Drain.Fast, time.Duration(r.Drain.TimeoutMillis)*time.Millisecond, caller)
			} else {
				b, err = d.cancel(ctx)
			}
		}
	}
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error(), Drain: &b}
	}
	return couchmessage.Response{Code: "ok", Drain: &b}
}

func (s *messageService) drainSnapshot() map[string]drainLiveMember {
	out := map[string]drainLiveMember{}
	live := s.liveness.Load()
	connected := s.connected.Load()
	if live == nil || connected == nil {
		return out
	}
	for slot, l := range *live {
		if !(*connected)[l.Binding] {
			continue
		}
		o := couchcore.DrainObservation{Connected: true, Submission: l.Submission}
		if l.Settled != nil && l.Picker != nil && l.BackgroundWork != nil {
			o.Known = true
			o.Idle = *l.Settled
			o.Picker = *l.Picker
			o.Background = *l.BackgroundWork
		}
		out[slot] = drainLiveMember{Binding: l.Binding, Session: strconv.FormatUint(uint64(l.Session), 10), Observation: o}
	}
	return out
}

// Cached live inventory includes unsupported/unregistered wrappers, which must
// never disappear from the board merely because messaging cannot reach them.
func drainCensusRows(inventory couchtty.ResourceInventorySnapshot, s *messageService) []couchcore.DrainRow {
	if !inventory.InventoryKnown || s == nil {
		return []couchcore.DrainRow{{Workspace: "fleet", State: couchcore.DrainStuck, Reason: "live inventory unknown", Recovery: "inspect status"}}
	}
	connected := s.drainSnapshot()
	seen := map[couchcore.ThreadAddress]bool{}
	for _, m := range connected {
		seen[couchcore.ThreadAddress{RepoScope: m.Binding.Scope, Tag: couchcore.ThreadTag(m.Binding.Tag)}] = true
	}
	var rows []couchcore.DrainRow
	hosted := map[couchcore.ThreadAddress]bool{}
	for _, h := range inventory.Hosted {
		hosted[h.Address] = true
	}
	for _, r := range inventory.Rows {
		if r.State != couchcore.ThreadLive && !hosted[r.Address] || seen[r.Address] {
			continue
		}
		delete(hosted, r.Address)
		name := r.Address.RepoScope + "/" + string(r.Address.Tag)
		if r.Target.Kind == couchcore.ThreadTargetSlot {
			name = fmt.Sprintf("%s:%d", r.Target.Slot.Repo, r.Target.Slot.Number)
		}
		rows = append(rows, couchcore.DrainRow{Workspace: name, State: couchcore.DrainStuck, Reason: "live wrapper is not admitted; drain capability unknown", Recovery: "inspect status"})
	}
	for address := range hosted {
		if !seen[address] {
			rows = append(rows, couchcore.DrainRow{Workspace: address.RepoScope + "/" + string(address.Tag), State: couchcore.DrainStuck, Reason: "hosted client has no admitted wrapper", Recovery: "inspect status"})
		}
	}
	return rows
}

func (e messageEndpoint) drainEndpoint(ctx context.Context) (couchmessage.DrainEndpoint, error) {
	if err := e.check(ctx); err != nil {
		return nil, err
	}
	d, ok := e.endpoint.(couchmessage.DrainEndpoint)
	if !ok {
		return nil, errors.New("wrapper drain capability unavailable; relaunch needed")
	}
	return d, nil
}
func (e messageEndpoint) ObserveDrain(ctx context.Context) (couchmessage.DrainObservation, error) {
	d, err := e.drainEndpoint(ctx)
	if err != nil {
		return couchmessage.DrainObservation{}, err
	}
	return d.ObserveDrain(ctx)
}
func (e messageEndpoint) InterruptDrain(ctx context.Context, id string, o couchmessage.DrainObservation) (couchmessage.DrainInterruptResult, error) {
	d, err := e.drainEndpoint(ctx)
	if err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	return d.InterruptDrain(ctx, id, o)
}
func (e messageEndpoint) DrainInterruptStatus(ctx context.Context, id string) (couchmessage.DrainInterruptResult, error) {
	d, err := e.drainEndpoint(ctx)
	if err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	return d.DrainInterruptStatus(ctx, id)
}
func (e messageEndpoint) RevokeDrain(ctx context.Context, id string) (couchmessage.DrainRevokeResult, error) {
	d, err := e.drainEndpoint(ctx)
	if err != nil {
		return couchmessage.DrainRevokeResult{}, err
	}
	return d.RevokeDrain(ctx, id)
}
