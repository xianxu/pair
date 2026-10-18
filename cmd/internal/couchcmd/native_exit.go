package couchcmd

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
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
	if exited := s.exited.Load(); exited != nil && *exited != nil {
		(*exited)(caller, report)
	}
	return couchmessage.Response{Code: "ok"}
}
