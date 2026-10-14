package couchcmd

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// handleSessionWrite admits only a current wrapper reporting its own owner slot.
func (s *messageService) handleSessionWrite(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	binding, err := s.switchCaller(ctx, r)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if binding.Slot != r.Target || !s.isConnected(binding) {
		return couchmessage.Response{Code: "refused", Error: "session-write requires the current caller's owner slot"}
	}
	if err := ctx.Err(); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if log := s.facts.Load(); log != nil {
		log.offer(supervisionFact{Event: "session_write", Subject: log.subject(binding), Incarnation: log.incarnation(binding)})
	}
	return couchmessage.Response{Code: "ok"}
}
