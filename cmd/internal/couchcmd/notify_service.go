package couchcmd

import (
	"context"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
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
	label := filepath.Base(workspace.WorktreeRoot)
	if workspace.Address != nil {
		label = *workspace.Address
	}
	title := r.Title
	if hubs := s.authority.hubs; hubs != nil && len(hubs.snapshot().Hubs()) > 1 {
		name := truncateBytes(notifyosc.Sanitize([]byte(label)), 120)
		title = "[" + name + "] " + notifyosc.Sanitize([]byte(title))
		if len(title) > notifyosc.MaxTitleBytes {
			title = truncateBytes(title, notifyosc.MaxTitleBytes-3)
		}
	}
	// The list does not depend on the terminal showing the escape (pair#673):
	// a focused Ghostty tab drops it.
	if id, err := newRequestID(); err == nil {
		if s.noticeOutput.Load() != nil {
			s.publishSystemNotice(couchtty.Notice{Class: couchcore.NoticeHubRequested, Kind: "notify", Thread: bindingThread(caller), Label: label, Body: r.Title + ": " + r.Body})
		} else if couchcore.RouteNotice(couchcore.NoticeRouteInput{Class: couchcore.NoticeHubRequested}).Bell {
			s.observeAttention(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: couchcore.AttentionItem{
				ID: "notify-" + id, Thread: bindingThread(caller), Label: label, Title: r.Title, Body: r.Body}})
		}
	}
	if err := (*write)(ctx, title, r.Body); err != nil {
		return couchmessage.Response{Code: "uncertain", Error: err.Error()}
	}
	return couchmessage.Response{Code: "ok"}
}
