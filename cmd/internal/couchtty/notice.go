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
