package launcher

import "time"

// SessionState describes whether a zellij session blocks tag reuse.
type SessionState string

const (
	SessionAttached SessionState = "attached"
	SessionDetached SessionState = "detached"
	SessionExited   SessionState = "exited"
)

// Session is a zellij session row projected into launcher decision space.
type Session struct {
	Name     string
	Tag      string
	RepoName string
	Agent    string
	State    SessionState
}

// HistoricalTag is a recently touched Pair tag with no live zellij session.
// MTime + QueueCount are populated by HistorySource.Scan (the decision path only
// reads Tag; the #99 M5a fzf pick-row build reads all three, purely).
type HistoricalTag struct {
	Tag            string
	MTime          time.Time // latest draft/log sidecar mtime (picker age grading)
	QueueCount     int       // queued prompts under queue-<tag>/ (picker badge)
	RepoName       string
	Agent          string
	LegacyUnscoped bool
}

// SessionSnapshot is the pure input to launcher decision-making.
type SessionSnapshot struct {
	BaseTag    string
	Sessions   []Session
	Historical []HistoricalTag
	// SessionNames optionally maps repo-local tags to already assigned public
	// zellij session names. Empty preserves the legacy pair-<tag> behavior.
	SessionNames map[string]string
}

// ListRow is one `pair list`/`ls` row: a pair-<tag> session with its resolved
// agent and reuse state, plus the live client count (0 for detached/exited) so
// the pure formatter can render "attached (N clients)" (#99 M5a).
type ListRow struct {
	Session string
	Agent   string
	State   SessionState
	Clients int
}
