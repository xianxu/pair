package webcast

// Phase is where a cast Chrome is. A tagged value, not a running/live pair of
// booleans: there is no "ended but live".
type Phase uint8

const (
	// PhaseIdle: Chrome runs and nothing is cast (before go-live, or paused).
	PhaseIdle Phase = iota
	// PhaseLive: the active tab is cast.
	PhaseLive
	// PhaseEnded: Chrome is gone. Final.
	PhaseEnded
)

func (p Phase) String() string {
	switch p {
	case PhaseIdle:
		return "idle"
	case PhaseLive:
		return "live"
	case PhaseEnded:
		return "ended"
	}
	return "unknown"
}

// Tab is one browser tab. ID is opaque (Chrome's target id); compare it, don't
// parse it.
type Tab struct {
	ID, URL, Title string
}

// State is everything a Browser publishes. HasTab is false while no page is
// open (a windowless Chrome about to be closed).
type State struct {
	Phase  Phase
	Tab    Tab
	HasTab bool
}
