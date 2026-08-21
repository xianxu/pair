package couchcore

// Message is one item in an actor's mailbox.
//
// Control marks the priority class -- stop, deadline, budget. It is what makes
// the capacity rule implementable: "drop the oldest non-control entry" needs a
// way to tell the classes apart.
type Message struct {
	Kind    string `json:"kind"`
	Control bool   `json:"control,omitempty"`
	Body    string `json:"body,omitempty"`
}

// Enqueue is the whole mailbox decision as a pure function, so it is testable
// without goroutines -- which is where this kind of code usually rots.
//
// Rules, in order:
//
//   - Collapse by Kind. A second "status" request replaces the first and moves
//     to the tail: five of them are one question, and answering it five times
//     is waste. The newest body wins because it is the current one.
//   - Over capacity, drop the OLDEST NON-CONTROL entry and return it, so the
//     caller can say what was lost.
//   - Never drop a Control message. Control carries stop and deadline; trading
//     a real obligation for a capacity number is the wrong way round. If
//     control alone exceeds capacity the queue grows and ok is false anyway.
//
// ok is false whenever the invariant was violated -- something was dropped, or
// capacity was exceeded. The issue's Spec asks for a full mailbox to be a loud
// bug signal rather than flow control, and a signature that cannot fail makes
// that impossible to honour.
func Enqueue(queue []Message, incoming Message, capacity int) (out []Message, dropped Message, ok bool) {
	out = make([]Message, 0, len(queue)+1)
	for _, m := range queue {
		if m.Kind == incoming.Kind {
			continue // collapsed: the incoming copy replaces it, at the tail
		}
		out = append(out, m)
	}
	out = append(out, incoming)

	if capacity <= 0 || len(out) <= capacity {
		return out, Message{}, true
	}

	for i, m := range out {
		if !m.Control {
			dropped = m
			out = append(out[:i:i], out[i+1:]...)
			return out, dropped, false
		}
	}
	// Control-only overflow: keep everything and report the violation.
	return out, Message{}, false
}
