package couchcmd

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

// archiveWarningOwner selects only the primary hub for global backup warnings.
func (s *messageService) archiveWarningOwner() (couchmessage.Binding, bool) {
	owner, ok := s.hubBinding()
	return owner, ok && s.isConnected(owner)
}

// notifyArchiveWarning publishes one already-reserved inventory warning. The
// typed delivery owner decides hub or fallback attention; this producer owns
// neither a desktop effect nor another hub admission. The returned body is
// retained in the archive pass log.
func (s *messageService) notifyArchiveWarning(ctx context.Context, body string, notice func(string)) (string, error) {
	body = notifyosc.Sanitize([]byte(body))
	if err := ctx.Err(); err != nil {
		return body, err
	}
	if _, available := s.archiveWarningOwner(); !available {
		const suffix = "; primary hub unavailable; full report in continuous.log"
		body = truncateBytes(body, notifyosc.MaxMessageBytes-len(suffix)-len("…")) + suffix
	}
	s.publishSystemNotice(couchtty.Notice{Class: couchcore.NoticeArchiveInventoryWarning, Kind: "status", Body: body})
	// Standalone scheduler fixtures retain their existing feed callback. A wired
	// Console already received the feed through publishSystemNotice.
	if publish := s.noticeOutput.Load(); (publish == nil || *publish == nil) && notice != nil {
		notice(body)
	}
	return body, nil
}
