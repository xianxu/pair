package couchtty

import (
	"context"
	"errors"
	"sort"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// MessageThread is an immutable snapshot of a committed, input-capable Pair
// client. Process identifies the Pair client, not its separate agent wrapper.
type MessageThread struct {
	Scope, Tag, Tree string
	Process          couchcore.ProcessIdentity
}

func (c *Console) SetMessageBroker(broker *couchmessage.Broker) {
	c.mu.Lock()
	c.messageBroker = broker
	c.mu.Unlock()
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
func (c *Console) MessageThreads() []MessageThread {
	c.mu.Lock()
	defer c.mu.Unlock()
	var rows []MessageThread
	if c.lifetime.Err() != nil {
		return rows
	}
	for _, p := range c.panes {
		if messagePaneLive(p) {
			rows = append(rows, MessageThread{Scope: p.thread.RepoScope, Tag: string(p.thread.Tag), Tree: string(p.tree), Process: p.process})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Scope == rows[j].Scope {
			return rows[i].Tag < rows[j].Tag
		}
		return rows[i].Scope < rows[j].Scope
	})
	return rows
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
