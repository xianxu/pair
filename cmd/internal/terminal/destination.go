package terminal

import (
	"errors"
	"fmt"
)

// ErrNoDestination is the presenter's answer when it holds no endpoint to
// deliver an input event to.
//
// It is a ROUTING answer, not a terminal failure, and the difference is the
// whole point. The compositor's panel is a legitimate no-destination state:
// Panel sets selected to nil and leaves Admitted empty, which is the healthy
// shape of a console showing its own screen. A caller that cannot tell this
// from a failed write tears down terminal ownership over a question it merely
// asked at the wrong moment -- which is what took couch down in pair#265.
//
// Physical failure keeps its own separate channel: Presenter.fail latches the
// View into Failed and closes Failed(). Nothing here touches that.
var ErrNoDestination = errors.New("terminal: no destination for input")

// noDestination is the ONE constructor, so the refusal sites cannot describe
// the same condition different ways. The View travels with the error because
// "which endpoint, in which state" is what makes the condition diagnosable when
// a caller does choose to surface it.
func noDestination(reason string, v View) error {
	return fmt.Errorf("%w: %s (state=%d selected=%q admitted=%q)",
		ErrNoDestination, reason, v.State, v.Selected, v.Admitted)
}

// IsRoutingAnswer reports whether err is the presenter saying it had nowhere to
// deliver an event, as opposed to the terminal being lost.
//
// This is the CLASSIFICATION, and it is deliberately a function rather than a
// second `errors.Is` at each caller. pair#265 fixed this class five times, each
// time by widening one `errors.Is` at one site, and each time the next member
// of the set was free to keep exiting couch. The set below is the enumeration;
// a consumer asks this question and gets every member at once.
//
// Membership, each declared rather than discovered:
//
//   - ErrNoDestination — routing. No endpoint is admitted; the panel is the
//     ordinary case.
//   - ErrInputEnded — routing. The child's input is closed, so there is nothing
//     to deliver to. It is NOT an ownership failure: the PTY read loop ends as
//     soon as the agent exits, while a console learns of that exit
//     asynchronously, so an ordinary keystroke lands in the gap. couch already
//     treats the same state as survivable when it resizes children.
//   - ErrBackpressure — NOT routing, deliberately. A full queue is a capacity
//     answer, and whether to drop input under load is its own decision with its
//     own operating envelope. Left fatal, and recorded as such in pair#265.
//   - "presenter unavailable" / "presenter released" — NOT routing. The
//     presenter is closing or has failed; ownership really is gone.
//   - a *WriteFailure — NOT routing. The parent terminal rejected bytes.
//
// TestInputErrorSetIsDeclared pins this list against the constructors the
// package actually has, so a new sentinel cannot join the set unclassified.
func IsRoutingAnswer(err error) bool {
	return errors.Is(err, ErrNoDestination) || errors.Is(err, ErrInputEnded)
}
