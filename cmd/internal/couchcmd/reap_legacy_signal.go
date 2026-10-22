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

// A legacy tree is a caller's frozen proposal, never a retained permission.
// Every individual signal returns through current supervisor ownership evidence.
func (s *messageService) handleLegacyOrphanSignal(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if r.Op != "reap-legacy-signal" {
		return couchmessage.Response{Code: "invalid-request", Error: "expected reap-legacy-signal"}
	}
	target := *r.LegacyTarget
	answer := func(status, detail string) couchmessage.Response {
		return couchmessage.Response{Code: "ok", LegacySignal: &couchmessage.LegacyOrphanSignalResult{Target: target, Status: status, Detail: detail}}
	}
	caller, origin, err := s.provenOrphanCaller(ctx, r)
	if err != nil {
		return answer("refused", err.Error())
	}
	bound := r.Scope != ""
	if bound && !s.authority.hubs.snapshot().IsHub(origin.WorktreeRoot) {
		return answer("refused", "only an authenticated hub may reap legacy orphans; ask that hub")
	}
	sampler := s.resources.Load()
	if sampler == nil {
		return answer("refused", "current ownership proof unavailable")
	}
	current, err := sampler.sweep(ctx)
	if err != nil {
		return answer("refused", fmt.Sprintf("current ownership proof unavailable: %v", err))
	}
	// Collection and native discovery never hold the exclusion locks. Moves and
	// sticky transitions cannot commit during the final decision and one effect.
	if hubs := s.authority.hubs; hubs != nil {
		hubs.mu.RLock()
		defer hubs.mu.RUnlock()
	}
	sampler.legacyGate.RLock()
	defer sampler.legacyGate.RUnlock()
	if sampler.provenanceIncomplete {
		return answer("refused", orphanLegacyRefusal)
	}
	if bound {
		latest, workspace, e := s.provenOrphanCaller(ctx, r)
		if e != nil {
			return answer("refused", e.Error())
		}
		if latest != caller || !reflect.DeepEqual(workspace, origin) || !s.authority.hubs.state.IsHub(workspace.WorktreeRoot) {
			return answer("refused", "caller binding, workspace or hub ownership changed")
		}
	}
	status, detail := couchcore.DecideLegacyOrphanSignal(target, current.proof, current.view)
	if status != "sent" {
		return answer(status, detail)
	}
	processes := s.provenOrphanProcesses
	if processes == nil {
		processes = launcher.OSProcessTable{}
	}
	if identity := processes.Identity(target.Root.PID); identity == "" || identity != target.Root.Identity {
		return answer("refused", "legacy root identity changed or unavailable")
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
