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
