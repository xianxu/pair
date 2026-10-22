package couchcmd

import (
	"context"
	"fmt"
	"reflect"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Unlike a slot mutation, reaping a former incarnation is reserved to its
// owning hub. A worker with the same owner does not acquire that authority.
func authorizeProvenOrphan(caller couchcore.WorkspaceIdentity, target couchcore.HubSlot, state couchcore.HubOwnership) error {
	owner := state.Resolve(target)
	if !state.IsHub(caller.WorktreeRoot) || owner.Hub == "" || len(owner.Conflicts) > 0 || owner.Hub != caller.WorktreeRoot {
		return fmt.Errorf("slot belongs to hub %s; ask that hub to reap its orphans", owner.Hub)
	}
	return nil
}

// provenOrphanCaller authenticates bound credentials without interpreting an
// invalid or stale binding as the unbound operator boundary.
func (s *messageService) provenOrphanCaller(ctx context.Context, r couchmessage.Request) (couchmessage.Binding, couchcore.WorkspaceIdentity, error) {
	if r.Scope == "" && r.Tag == "" && r.Session == "" && r.Nonce == "" {
		return couchmessage.Binding{}, couchcore.WorkspaceIdentity{}, nil
	}
	if r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || s.broker == nil || s.authority.hubs == nil {
		return couchmessage.Binding{}, couchcore.WorkspaceIdentity{}, fmt.Errorf("complete current owning hub credentials required")
	}
	b, err := s.switchCaller(ctx, r)
	if err != nil {
		return b, couchcore.WorkspaceIdentity{}, err
	}
	connected := s.connected.Load()
	if connected == nil || !(*connected)[b] {
		return b, couchcore.WorkspaceIdentity{}, fmt.Errorf("caller is disconnected")
	}
	if s.authority.process == nil {
		return b, couchcore.WorkspaceIdentity{}, fmt.Errorf("caller process proof unavailable")
	}
	if err = s.authority.process(b); err != nil {
		return b, couchcore.WorkspaceIdentity{}, err
	}
	origin, err := s.connectedWorkspace(ctx, b)
	return b, origin, err
}

func (s *messageService) handleProvenOrphanSignal(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if r.Op != "reap-orphan-signal" {
		return couchmessage.Response{Code: "invalid-request", Error: "expected reap-orphan-signal"}
	}
	target := *r.OrphanTarget
	answer := func(status, detail string) couchmessage.Response {
		return couchmessage.Response{Code: "ok", OrphanSignal: &couchmessage.ProvenOrphanSignalResult{Target: target, Status: status, Detail: detail}}
	}
	caller, origin, err := s.provenOrphanCaller(ctx, r)
	if err != nil {
		return answer("refused", err.Error())
	}
	bound := r.Scope != ""
	slot, ok := hubWorkspaceSlot(target.Workspace)
	if !ok {
		return answer("refused", "target canonical slot unavailable")
	}
	if bound {
		if err = authorizeProvenOrphan(origin, slot, s.authority.hubs.snapshot()); err != nil {
			return answer("refused", err.Error())
		}
	}
	sampler := s.resources.Load()
	if sampler == nil {
		return answer("refused", "current ownership proof unavailable")
	}
	proof, err := sampler.proveReap(ctx)
	if err != nil {
		return answer("refused", fmt.Sprintf("current ownership proof unavailable: %v", err))
	}
	// A move's write lock cannot commit between the final ownership decision and
	// this one effect. Native discovery and the worker exchange happen above.
	if hubs := s.authority.hubs; hubs != nil {
		hubs.mu.RLock()
		defer hubs.mu.RUnlock()
	}
	if bound {
		latest, workspace, e := s.provenOrphanCaller(ctx, r)
		if e != nil {
			return answer("refused", e.Error())
		}
		if latest != caller || !reflect.DeepEqual(workspace, origin) {
			return answer("refused", "caller binding or workspace changed")
		}
		if e = authorizeProvenOrphan(workspace, slot, s.authority.hubs.state); e != nil {
			return answer("refused", e.Error())
		}
	}
	status, detail := couchcore.DecideProvenOrphanSignal(target, proof)
	if status != "sent" {
		return answer(status, detail)
	}
	processes := s.provenOrphanProcesses
	if processes == nil {
		processes = launcher.OSProcessTable{}
	}
	identity := processes.Identity(target.Key.PID)
	if identity == "" {
		return answer("refused", "final process identity unavailable")
	}
	if identity != target.Key.Identity {
		return answer("skipped", "original process generation changed")
	}
	if err = ctx.Err(); err != nil {
		return answer("failed", err.Error())
	}
	signal := syscall.SIGTERM
	if r.OrphanSignal == "KILL" {
		signal = syscall.SIGKILL
	}
	if err = processes.Signal(target.Key.PID, signal); err != nil {
		return answer("failed", fmt.Sprintf("signal result uncertain: %v", err))
	}
	return answer("sent", "")
}
