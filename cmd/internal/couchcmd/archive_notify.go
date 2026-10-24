package couchcmd

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

// archiveWarningOwner is the single routing seam for global backup warnings.
// Until source ownership is modeled (#694), only the primary hub owns them.
func (s *messageService) archiveWarningOwner() (couchmessage.Binding, bool) {
	owner, ok := s.hubBinding()
	return owner, ok && s.isConnected(owner)
}

// notifyArchiveWarning delivers an already-reserved warning once. Status,
// attention and desktop output are independent; an uncertain desktop write
// must not cause a retry. The caller retains the returned body in the pass log.
func (s *messageService) notifyArchiveWarning(ctx context.Context, body string, notice func(string)) (string, error) {
	body = notifyosc.Sanitize([]byte(body))
	if err := ctx.Err(); err != nil {
		return body, err
	}
	owner, available := s.archiveWarningOwner()
	label := ""
	title := "History backup"
	if available {
		s.mu.Lock()
		workspace := s.workspaces[owner]
		s.mu.Unlock()
		label = owner.Slot
		if workspace.WorktreeRoot != "" {
			label = filepath.Base(workspace.WorktreeRoot)
		}
		if workspace.Address != nil {
			label = *workspace.Address
		}
		label = truncateBytes(notifyosc.Sanitize([]byte(label)), 120)
		if hubs := s.authority.hubs; hubs != nil && len(hubs.snapshot().Hubs()) > 1 {
			title = "[" + label + "] " + title
		}
	}
	title = truncateBytes(notifyosc.Sanitize([]byte(title)), notifyosc.MaxTitleBytes-len("…"))
	var failures []error
	id, err := newRequestID()
	if err != nil {
		failures = append(failures, err)
	}
	if err := ctx.Err(); err != nil {
		return body, err
	}
	// Revalidate immediately before effects. Never choose another hub after
	// this point: a raised item names the original exact thread even if it exits.
	available = available && s.isConnected(owner)
	if !available {
		const suffix = "; primary hub unavailable; full report in continuous.log"
		body = truncateBytes(body, notifyosc.MaxMessageBytes-len(suffix)-len("…")) + suffix
		title = "History backup"
	}
	if id != "" {
		thread := couchcore.ThreadAddress{}
		if available {
			thread = bindingThread(owner)
		} else {
			label = "Couch"
		}
		s.observeAttention(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: couchcore.AttentionItem{
			ID: "notify-" + id, Thread: thread, Label: label, Title: "History backup", Body: body,
		}})
	}
	if err := ctx.Err(); err != nil {
		return body, errors.Join(append(failures, err)...)
	}
	if notice != nil {
		notice(body)
	}
	if err := ctx.Err(); err != nil {
		return body, errors.Join(append(failures, err)...)
	}
	if write := s.desktopNotify.Load(); write != nil && *write != nil {
		if err := (*write)(ctx, title, body); err != nil {
			failures = append(failures, err)
		}
	} else {
		failures = append(failures, errors.New("history backup desktop notification unavailable"))
	}
	return body, errors.Join(failures...)
}
