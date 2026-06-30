package launcher

// SessionState describes whether a zellij session blocks tag reuse.
type SessionState string

const (
	SessionAttached SessionState = "attached"
	SessionDetached SessionState = "detached"
	SessionExited   SessionState = "exited"
)

// Session is a zellij session row projected into launcher decision space.
type Session struct {
	Name  string
	State SessionState
}

// HistoricalTag is a recently touched Pair tag with no live zellij session.
type HistoricalTag struct {
	Tag string
}

// SessionSnapshot is the pure input to launcher decision-making.
type SessionSnapshot struct {
	BaseTag    string
	Sessions   []Session
	Historical []HistoricalTag
}
