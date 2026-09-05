package couchtty

import (
	"fmt"
	"time"

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

// NoticeLifetime is how long a TRANSIENT notice stands on the status row.
//
// Long enough to read a sentence the operator did not expect, short enough that
// it is gone before they wonder whether it still applies.
const NoticeLifetime = 12 * time.Second

// FeedRow is what the status row shows, and when it stops being true.
//
// One method returns both, because a caller that asks for the body always needs
// to know when to look again -- and two methods walking the same queue is how
// the two answers drift apart.
type FeedRow struct {
	Body string
	// Expires is the zero time for a row that stands until something replaces
	// it: an empty row, or a control notice. It is the row's IDENTITY, so a
	// caller re-arming a timer every event loop can tell "the same deadline"
	// from "a new one" instead of pushing the deadline out forever.
	Expires time.Time
	// Standing is how much longer the row stands, measured on the FEED's clock.
	// A caller must arm its timer from this rather than from time.Until(Expires):
	// the two agree only while the feed's clock is the wall clock, and the point
	// of injecting a clock is that it is not.
	Standing time.Duration
}

// Feed is the bounded rolling status-row history. couchcore.Enqueue remains the
// single owner of collapse, capacity, and control-priority policy; Feed owns
// only the capacity, the Notice-to-Message key convention, and how long a
// transient notice stands.
//
// Nothing used to retire a notice at all -- no timer, no clear on keystroke, no
// expiry -- so a momentary refusal like "previous: nowhere to return to" sat on
// the row until some unrelated notice happened to replace it, reading as current
// state long after it stopped being about anything. The distinction the fix
// needs was already in the type: an exit is an OBLIGATION (Control) and stands
// until it is displaced, while a refusal is an EVENT about the keystroke just
// pressed and is meaningless a minute later.
type Feed struct {
	capacity int
	now      func() time.Time
	lifetime time.Duration
	queue    []couchcore.Message
	// expiry is keyed by Message.Kind, which IS Enqueue's collapse identity, so
	// a replaced notice inherits the slot rather than accumulating one.
	expiry map[string]time.Time
}

// NewFeed takes its clock AND its lifetime.
//
// Both, because they are tested at different levels and a seam that only moves
// one is a trap: pure expiry tests hand-advance a fake clock, while a console
// test has to let a REAL timer fire, so it needs a real clock and a short
// lifetime instead. Injecting only the clock lets a test advance fake time while
// the console waits on a twelve-second real timer -- the arming looks exercised
// and is not.
func NewFeed(capacity int, now func() time.Time, lifetime time.Duration) *Feed {
	if now == nil {
		now = time.Now
	}
	if lifetime <= 0 {
		lifetime = NoticeLifetime
	}
	return &Feed{capacity: capacity, now: now, lifetime: lifetime, expiry: map[string]time.Time{}}
}

// Push adds a notice and reports whether the capacity invariant held.
func (f *Feed) Push(n Notice) bool {
	if f == nil {
		return false
	}
	message := n.Message()
	next, _, ok := couchcore.Enqueue(f.queue, message, f.capacity)
	f.queue = next
	if f.expiry == nil {
		f.expiry = map[string]time.Time{}
	}
	if message.Control {
		delete(f.expiry, message.Kind)
	} else {
		f.expiry[message.Kind] = f.now().Add(f.lifetime)
	}
	// Drop expiries for messages Enqueue no longer retains, so the map cannot
	// outgrow the bounded queue it describes.
	retained := make(map[string]bool, len(f.queue))
	for _, m := range f.queue {
		retained[m.Kind] = true
	}
	for kind := range f.expiry {
		if !retained[kind] {
			delete(f.expiry, kind)
		}
	}
	return ok
}

// Row is the status-row projection: the newest notice STILL STANDING.
//
// Walking from the tail and skipping what has expired, rather than showing the
// newest unconditionally. An older transient is staler than the one that just
// expired, so it is never a better answer -- only a control notice survives to
// be found this way.
func (f *Feed) Row() FeedRow {
	if f == nil {
		return FeedRow{}
	}
	now := f.now()
	for i := len(f.queue) - 1; i >= 0; i-- {
		message := f.queue[i]
		expires, transient := f.expiry[message.Kind]
		if !transient {
			return FeedRow{Body: message.Body}
		}
		if now.Before(expires) {
			return FeedRow{Body: message.Body, Expires: expires, Standing: expires.Sub(now)}
		}
	}
	return FeedRow{}
}

// Messages returns an independent snapshot for audits and tests.
func (f *Feed) Messages() []couchcore.Message {
	if f == nil {
		return nil
	}
	return append([]couchcore.Message(nil), f.queue...)
}
