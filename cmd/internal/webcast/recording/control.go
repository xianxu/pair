package recording

import (
	"errors"
	"fmt"
	"slices"
)

// Source attributes an interaction record (a click, a value, focus, a
// scroll, a selection, hover, the sampled pointer) to the viewer whose
// injected input caused it (#466). Empty is the operator, or the page's own
// script. The reducer refuses a By that isn't the current driver.
type Source struct {
	By string `json:"by,omitempty"`
}

func (s *Source) source() string { return s.By }

// attributed is every record that embeds Source.
type attributed interface{ source() string }

// Control ops (#466).
const (
	// CtlReset clears active and pending control in one operator action.
	CtlReset = "reset"
	// CtlRequest: a viewer asks to drive.
	CtlRequest = "request"
	// CtlGrant: the operator lets a pending viewer drive; a previous
	// driver loses control.
	CtlGrant = "grant"
	// CtlDeny: the operator turns a pending request down.
	CtlDeny = "deny"
	// CtlRevoke: the operator takes control back from the driver.
	CtlRevoke = "revoke"
	// CtlRelease: the driver gives control back, or a pending viewer
	// withdraws its request.
	CtlRelease = "release"
	// CtlLeave: the viewer's seat expired (no stream for the grace period);
	// it stops driving or asking.
	CtlLeave = "leave"
)

// Control is who drives the cast tab: one viewer, or nobody (the operator),
// and who has asked to. It is per recording, not per segment: it survives
// pause, resume and navigation, and ends at stop end.
type Control struct {
	Driver  *Author  `json:"driver,omitempty"`
	Pending []Author `json:"pending,omitempty"`
}

// Driving is the driver's viewer id, or "" for the operator.
func (c Control) Driving() string {
	if c.Driver == nil {
		return ""
	}
	return c.Driver.ID
}

func (c Control) pendingAt(id string) int {
	return slices.IndexFunc(c.Pending, func(a Author) bool { return a.ID == id })
}

// Equal compares by viewer ids, in order.
func (c Control) Equal(o Control) bool {
	if c.Driving() != o.Driving() || len(c.Pending) != len(o.Pending) {
		return false
	}
	for i := range c.Pending {
		if c.Pending[i].ID != o.Pending[i].ID {
			return false
		}
	}
	return true
}

// ErrControl: a control op that the current state doesn't allow, or that
// would change nothing.
var ErrControl = errors.New("recording: control op not allowed here")

// NextControl is the one control transition: the reducer checks every ctl
// record against it, and a controller produces its records with it. It
// never changes c.
func NextControl(c Control, op string, who Author) (Control, error) {
	if op == CtlReset {
		if who.ID != OperatorID {
			return c, fmt.Errorf("%w: reset requires operator attribution", ErrControl)
		}
		return Control{}, nil
	}
	if who.ID == "" {
		return c, fmt.Errorf("%w: no viewer", ErrControl)
	}
	driving := c.Driving() == who.ID
	at := c.pendingAt(who.ID)
	next := Control{Driver: c.Driver, Pending: slices.Clone(c.Pending)}
	dropPending := func() { next.Pending = slices.Delete(next.Pending, at, at+1) }
	switch op {
	case CtlRequest:
		if driving || at >= 0 {
			return c, fmt.Errorf("%w: %s already drives or asked", ErrControl, who.ID)
		}
		next.Pending = append(next.Pending, who)
	case CtlGrant:
		if at < 0 {
			return c, fmt.Errorf("%w: %s hasn't asked", ErrControl, who.ID)
		}
		dropPending()
		next.Driver = &who
	case CtlDeny:
		if at < 0 {
			return c, fmt.Errorf("%w: %s hasn't asked", ErrControl, who.ID)
		}
		dropPending()
	case CtlRevoke:
		if !driving {
			return c, fmt.Errorf("%w: %s isn't driving", ErrControl, who.ID)
		}
		next.Driver = nil
	case CtlRelease, CtlLeave:
		switch {
		case driving:
			next.Driver = nil
		case at >= 0:
			dropPending()
		default:
			return c, fmt.Errorf("%w: %s neither drives nor asked", ErrControl, who.ID)
		}
	default:
		return c, fmt.Errorf("%w: unknown op %q", ErrControl, op)
	}
	if len(next.Pending) == 0 {
		next.Pending = nil
	}
	return next, nil
}

// Ctl is a control change: the op, the viewer it concerns, and the control
// state after it. Carrying the state makes the latest ctl alone enough for a
// late joiner; the reducer recomputes it, so it can't drift from the rules.
type Ctl struct {
	Header
	Op  string `json:"op"`
	Who Author `json:"who"`
	Control
}

func (*Ctl) Kind() string { return "ctl" }

func (r *Ctl) check() error {
	if r.Op == CtlReset {
		return need(r.Who.ID == OperatorID, "operator attribution")
	}
	return errors.Join(need(oneOf(r.Op, CtlRequest, CtlGrant, CtlDeny, CtlRevoke, CtlRelease, CtlLeave), "op"),
		need(r.Who.ID != "", "who.id"))
}

// applyCtl folds a ctl record into the state. Every ctl must state what its
// op gives from the previous one. The first ctl a reader sees is its
// baseline instead: a late joiner's catch-up starts at the latest ctl, with
// no history before it, so the first is taken as stated if it is consistent
// in itself. A reader of a whole recording (Validate) holds the first one to
// the rule too, from no control.
func (s *State) applyCtl(r *Ctl) error {
	if err := r.consistent(); err != nil {
		return err
	}
	if !s.ctlSeen && !s.whole {
		s.sketchControl(r)
		s.Control, s.ctlSeen = r.Control, true
		return nil
	}
	next, err := NextControl(s.Control, r.Op, r.Who)
	if err != nil {
		return err
	}
	if !next.Equal(r.Control) {
		return fmt.Errorf("ctl %s %s states driver %q pending %v; the op gives driver %q pending %v",
			r.Op, r.Who.ID, r.Driving(), ids(r.Pending), next.Driving(), ids(next.Pending))
	}
	s.sketchControl(r)
	s.Control, s.ctlSeen = r.Control, true
	return nil
}

// consistent: the stated control is one the op can leave behind: no viewer
// twice, and the viewer where the op puts it.
func (r *Ctl) consistent() error {
	if r.Op == CtlReset {
		if r.Who.ID != OperatorID || r.Driver != nil || len(r.Pending) != 0 {
			return fmt.Errorf("%w: reset must leave no driver or requests", ErrControl)
		}
		return nil
	}
	if r.Driver != nil && r.Driver.ID == "" {
		return fmt.Errorf("%w: a driver without an id", ErrControl)
	}
	seen := map[string]bool{r.Driving(): r.Driver != nil}
	for _, p := range r.Pending {
		if p.ID == "" || seen[p.ID] {
			return fmt.Errorf("%w: viewer %q twice, or without an id", ErrControl, p.ID)
		}
		seen[p.ID] = true
	}
	driving, asked := r.Driving() == r.Who.ID, r.pendingAt(r.Who.ID) >= 0
	var ok bool
	switch r.Op {
	case CtlRequest:
		ok = asked
	case CtlGrant:
		ok = driving
	default: // deny, revoke, release, leave
		ok = !driving && !asked
	}
	if !ok {
		return fmt.Errorf("%w: after %s, %s can't be where the ctl states", ErrControl, r.Op, r.Who.ID)
	}
	return nil
}

// checkSource refuses an attributed record whose viewer isn't driving.
func (s *State) checkSource(r Record) error {
	a, ok := r.(attributed)
	if !ok || a.source() == "" {
		return nil
	}
	if by := a.source(); by != s.Control.Driving() {
		return fmt.Errorf("by %q, but the driver is %q", by, s.Control.Driving())
	}
	return nil
}

func ids(as []Author) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.ID
	}
	return out
}

// sketchControl is called only after the entire control transition validates.
func (s *State) sketchControl(r *Ctl) {
	if r.Op == CtlReset {
		s.Sketches.interrupt(SketchInterruptReset)
	} else if s.Control.Driving() != r.Control.Driving() {
		s.Sketches.interrupt(SketchInterruptHandoff)
	}
}
