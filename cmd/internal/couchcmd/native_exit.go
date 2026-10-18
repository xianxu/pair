package couchcmd

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

type nativeExitRecord struct {
	Binding       couchmessage.Binding
	LaunchOrdinal uint64
}

// handleNativeExit admits the wrapper's report that the native child it owns
// exited on its own (pair#657), then hands one exited fact to the lifecycle
// notices. Attach exit, detach, reconnect and wrapper replacement never reach
// here; this checks the report names the caller's current child:
//   - the caller is the authenticated, admitted, current wrapper (switchCaller);
//   - the ID is derived from that exact binding and launch;
//   - the launch ordinal and agent identity match the caller's ready record,
//     which fences the child from before a SIGUSR2 restart (same binding);
//   - the session is still connected (rechecked under the reservation lock);
//   - a repeat of the same (binding, launch) posts nothing.
func (s *messageService) handleNativeExit(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	report, _ := couchmessage.ParseNativeExit(r.Body)
	if s.authority.child == nil {
		return couchmessage.Response{Code: "unsupported", Error: "native exit reports are not admitted"}
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if r.ID != couchmessage.NativeExitID(caller, report.LaunchOrdinal) {
		return couchmessage.Response{Code: "invalid-request", Error: "native exit ID does not match caller and launch"}
	}
	ordinal, identity, err := s.authority.child(ctx, caller)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if ordinal != report.LaunchOrdinal || identity != report.AgentIdentity {
		return couchmessage.Response{Code: "stale", Error: "native exit names a child other than the current launch"}
	}
	if err := ctx.Err(); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	s.exits.Lock()
	if !s.isConnected(caller) {
		s.exits.Unlock()
		return couchmessage.Response{Code: "refused", Error: "native exit caller disconnected"}
	}
	if previous, ok := s.exits.records[caller.Slot]; ok && previous.Binding == caller && previous.LaunchOrdinal == report.LaunchOrdinal {
		s.exits.Unlock()
		return couchmessage.Response{Code: "ok"}
	}
	if _, ok := s.exits.records[caller.Slot]; !ok && len(s.exits.records) >= couchmessage.MaxActors {
		s.exits.Unlock()
		return couchmessage.Response{Code: "refused", Error: "native exit occurrence capacity reached"}
	}
	if s.exits.records == nil {
		s.exits.records = map[string]nativeExitRecord{}
	}
	s.exits.records[caller.Slot] = nativeExitRecord{Binding: caller, LaunchOrdinal: report.LaunchOrdinal}
	s.exits.Unlock()
	s.mu.Lock()
	slot, known := s.workspaces[caller]
	s.mu.Unlock()
	if exited := s.exited.Load(); exited != nil && *exited != nil && known {
		(*exited)(caller, slot, report)
	}
	return couchmessage.Response{Code: "ok"}
}

// wireNativeExitNotices routes admitted native exits through the Console, so
// each takes its place in the Console's per-slot observation order, then to
// the lifecycle publisher as one exited fact for the caller's verified slot.
func wireNativeExitNotices(s *messageService, console *couchtty.Console) {
	exited := func(caller couchmessage.Binding, identity couchcore.WorkspaceIdentity, report couchmessage.NativeExit) {
		// The caller's admitted slot address names the slot; the verified
		// workspace identity supplies its checkout roots.
		target := couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: identity.Repo, PrimaryRoot: identity.PrimaryRoot, WorktreeRoot: identity.WorktreeRoot}}
		if identity.Slot != nil {
			target.Slot.Number = *identity.Slot
		}
		if ref, ok, err := couchcore.ParseWorkspaceReference(caller.Slot); ok && err == nil {
			target.Slot.Repo, target.Slot.Number = ref.Repo, ref.Number
		}
		console.ObserveNativeExit(couchcore.ThreadAddress{RepoScope: caller.Scope, Tag: couchcore.ThreadTag(caller.Tag)}, target,
			fmt.Sprintf("%d:%s", report.LaunchOrdinal, report.AgentIdentity), report.ExitCode)
	}
	s.exited.Store(&exited)
}
