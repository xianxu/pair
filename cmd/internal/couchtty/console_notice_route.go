package couchtty

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// NoticeDeliveryBackend wraps only cached ownership and the existing guarded
// broker ingress. Console never invokes its IO methods while holding locks.
type NoticeDeliveryBackend interface {
	NoticeOwner(couchcore.NoticeDeliveryOccurrence) couchcore.NoticeDeliveryOwner
	// NoticeOwnerCurrent compares only the retained root's cached generation;
	// both ownership methods are memory-only and invoke no callbacks.
	NoticeOwnerCurrent(couchcore.NoticeDeliveryOwner) bool
	DeliverNotice(context.Context, couchcore.NoticeDeliveryEffect) couchcore.NoticeDeliveryResult
	NoticeReceipt(string) (known, terminal bool)
}

type NoticeDelivery interface {
	Publish(Notice)
	Bind(NoticeDeliveryBackend)
	Close()
}

func (c *Console) SetNoticeDelivery(delivery NoticeDelivery) {
	c.mu.Lock()
	c.noticeDelivery = delivery
	c.noticeRouter = delivery.Publish
	c.mu.Unlock()
}

func (c *Console) NoticeDeliveryInstalled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.noticeDelivery != nil
}

func (c *Console) BindNoticeDelivery(backend NoticeDeliveryBackend) {
	c.mu.Lock()
	delivery := c.noticeDelivery
	c.mu.Unlock()
	if delivery != nil {
		delivery.Bind(backend)
	}
}

func (c *Console) CloseNoticeDelivery() {
	c.mu.Lock()
	delivery := c.noticeDelivery
	c.mu.Unlock()
	if delivery != nil {
		delivery.Close()
	}
}

// SetNoticeRouter installs the lifetime owner's bounded memory handoff. The
// callback runs under the Console mutex: it must neither do IO nor re-enter
// Console. Production installs it before trace and startup warning producers.
func (c *Console) SetNoticeRouter(publish func(Notice)) {
	c.mu.Lock()
	c.noticeRouter = publish
	c.mu.Unlock()
}

// ApplyNoticeAttention projects a decided route without publishing feed again.
// Reusing an ID updates an aggregate rather than creating another bell item.
func (c *Console) ApplyNoticeAttention(n Notice, id string, updateOnly ...bool) {
	c.mu.Lock()
	if c.lifetime.Err() != nil || id == "" {
		c.mu.Unlock()
		return
	}
	c.noticeSequence++
	if item, ok := noticeAttention(n, c.noticeSequence); ok {
		item.ID = id
		c.attn.mu.Lock()
		found := false
		for _, previous := range c.attn.list.Items() {
			if previous.ID == id {
				found = true
				item.Since = previous.Since
				break
			}
		}
		if len(updateOnly) > 0 && updateOnly[0] && !found {
			c.attn.mu.Unlock()
			c.mu.Unlock()
			return
		}
		c.attn.list = c.attn.list.Dismiss(id)
		c.attn.list = c.attn.list.Apply(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: item}, c.now())
		c.layoutAttentionLocked(int(c.size.Cols), int(c.size.Rows)-c.statusRows+2)
		c.attn.mu.Unlock()
	}
	started := c.started
	c.mu.Unlock()
	if started {
		c.repaint()
	}
}
