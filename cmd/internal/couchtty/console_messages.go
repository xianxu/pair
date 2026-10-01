package couchtty

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func (c *Console) SetMessageBroker(broker *couchmessage.Broker) {
	c.mu.Lock()
	c.messageBroker = broker
	c.mu.Unlock()
}

// SubscribeMessageLifecycle hands the message service the Console's pane
// state. Panes attached before the service started (the startup pane, the
// reattach pass) are replayed under the same lock that installs the mailbox,
// so no attach or exit can fall between them (#365).
func (c *Console) SubscribeMessageLifecycle(m *couchmessage.PaneMailbox) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messagePanes = m
	for _, p := range c.panes {
		c.postMessagePaneLocked(p.thread)
	}
}

// postMessagePaneLocked publishes the thread's newest pane ("" when none): an
// exiting pane whose input had ended may leave behind a newer one for the
// same thread.
func (c *Console) postMessagePaneLocked(thread couchcore.ThreadAddress) {
	if c.messagePanes == nil {
		return
	}
	var current couchmessage.PaneHandle
	for _, id := range c.order {
		if p := c.panes[id]; p != nil && p.thread == thread {
			current = p.messageHandle
		}
	}
	c.messagePanes.Post(couchmessage.ThreadKey{Scope: thread.RepoScope, Tag: string(thread.Tag)}, current)
}

func messagePaneLive(p *pane) bool {
	if p == nil || p.child == nil || p.thread.RepoScope == "" || p.thread.Tag == "" || p.tree == "" || p.process.PID <= 0 || p.process.Identity == "" || p.child.Endpoint().InputEnded() {
		return false
	}
	select {
	case <-p.child.Exited():
		return false
	default:
		return true
	}
}

// MessageBinding verifies only the Console-owned client evidence. The caller
// must separately prove the workspace and wrapper incarnation before registering.
func (c *Console) MessageBinding(ctx context.Context, binding couchmessage.Binding) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.lifetime.Err(); err != nil {
		return "", err
	}
	var found *pane
	for _, p := range c.panes {
		if p.thread.RepoScope == binding.Scope && string(p.thread.Tag) == binding.Tag && messagePaneLive(p) {
			if found != nil {
				return "", errors.New("multiple live client bindings for message thread")
			}
			found = p
		}
	}
	if found == nil {
		return "", errors.New("message thread has no live committed Couch pane")
	}
	return string(found.tree), nil
}
