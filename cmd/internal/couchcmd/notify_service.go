package couchcmd

import (
	"context"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func (s *messageService) SetDesktopNotify(write func(context.Context, string, string) error) {
	s.desktopNotify.Store(&write)
}

func (s *messageService) handleNotify(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	workspace, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if !s.authority.isHub(workspace) {
		return couchmessage.Response{Code: "refused", Error: "only the live Couch hub may notify; send to the hub instead (couch --send-to hub --message TEXT)"}
	}
	write := s.desktopNotify.Load()
	if write == nil || *write == nil {
		return couchmessage.Response{Code: "unavailable", Error: "Couch notification output is unavailable"}
	}
	// Workspace lookup can yield while the wrapper is replaced. Revalidate
	// its recorded identity at admission, immediately before submission.
	if _, err := s.authority.current(ctx, caller); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if !s.isConnected(caller) {
		return couchmessage.Response{Code: "refused", Error: "notification caller disconnected"}
	}
	if err := ctx.Err(); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	title := r.Title
	if hubs := s.authority.hubs; hubs != nil && len(hubs.snapshot().Hubs()) > 1 {
		name := filepath.Base(workspace.WorktreeRoot)
		if workspace.Address != nil {
			name = *workspace.Address
		}
		name = truncateBytes(notifyosc.Sanitize([]byte(name)), 120)
		title = "[" + name + "] " + notifyosc.Sanitize([]byte(title))
		if len(title) > notifyosc.MaxTitleBytes {
			title = truncateBytes(title, notifyosc.MaxTitleBytes-3)
		}
	}
	if err := (*write)(ctx, title, r.Body); err != nil {
		return couchmessage.Response{Code: "uncertain", Error: err.Error()}
	}
	return couchmessage.Response{Code: "ok"}
}
