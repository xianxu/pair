package couchtty

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// Notice is one status-row event before mailbox policy is applied. Actor is
// part of its identity: two actors ringing are two obligations, while repeated
// rings from one actor are one newer obligation.
type Notice struct {
	Actor   couchcore.ActorID
	Kind    string
	Body    string
	Control bool
}

// Message gives couchcore.Enqueue the per-actor collapse key and priority.
func (n Notice) Message() couchcore.Message {
	kind := n.Kind
	if n.Actor != "" {
		kind += ":" + string(n.Actor)
	}
	return couchcore.Message{Kind: kind, Body: n.Body, Control: n.Control}
}

// ExitNotice is control priority: capacity pressure may discard an activity
// hint, never the fact that an actor ended and why its pane disappeared.
func ExitNotice(actor couchcore.ActorID, label string, code int) Notice {
	return Notice{
		Actor: actor, Kind: "exit", Control: true,
		Body: fmt.Sprintf("%s [%s] exited (%d)", label, actor, code),
	}
}

// BellNotice is deliberately keyed by actor rather than by the global kind
// "bell": repeated pages from one actor collapse, while two actors remain two
// obligations.
func BellNotice(actor couchcore.ActorID, label string) Notice {
	return Notice{Actor: actor, Kind: "bell", Body: label + " wants you"}
}

// Feed is the bounded rolling status-row history. couchcore.Enqueue remains the
// single owner of collapse, capacity, and control-priority policy; Feed owns
// only the capacity and Notice-to-Message key convention.
type Feed struct {
	capacity int
	queue    []couchcore.Message
}

func NewFeed(capacity int) *Feed { return &Feed{capacity: capacity} }

// Push adds a notice and reports whether the capacity invariant held.
func (f *Feed) Push(n Notice) bool {
	if f == nil {
		return false
	}
	next, _, ok := couchcore.Enqueue(f.queue, n.Message(), f.capacity)
	f.queue = next
	return ok
}

// Latest is the row-sized projection: the newest retained notice body.
func (f *Feed) Latest() string {
	if f == nil || len(f.queue) == 0 {
		return ""
	}
	return f.queue[len(f.queue)-1].Body
}

// Messages returns an independent snapshot for audits and tests.
func (f *Feed) Messages() []couchcore.Message {
	if f == nil {
		return nil
	}
	return append([]couchcore.Message(nil), f.queue...)
}
