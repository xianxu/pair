package launcher

import (
	"fmt"
	"time"
)

// SessionState describes whether a zellij session blocks tag reuse.
type SessionState string

const (
	SessionAttached SessionState = "attached"
	SessionDetached SessionState = "detached"
	SessionExited   SessionState = "exited"
	// SessionLive is a session that has not exited, whose attach state was NOT
	// ASKED: a liveness snapshot reports it instead of guessing detached. It
	// exists because asking costs a list-clients per session -- about 250 ms
	// each against a real detached one (pair#228) -- and most callers only need
	// "not exited". A consumer that needs attached-versus-detached must take a
	// snapshot that asked, and refuses one that did not via RequireAttachState.
	SessionLive SessionState = "live"
)

// RequireAttachState is the one rule every reader of attached-versus-detached
// applies before reading it: a SessionLive row was never asked, so a reader
// that took it would see "not detached" and act on a fabrication -- skip the
// picker, list no detached row, refuse a resume. The callers take a snapshot
// that asked, so this fires only on drift; it is loud so drift is found.
//
// Readers today: DecideLaunch's bare branch, the picker (resolvePickWithPolicy),
// and couchcore's ProjectDetachedSessions.
func RequireAttachState(sessions []Session) error {
	for _, sess := range sessions {
		if sess.State == SessionLive {
			return fmt.Errorf("attach state is needed, but session %q was observed for liveness only", sess.Name)
		}
	}
	return nil
}

// Session is a zellij session row projected into launcher decision space.
type Session struct {
	Name     string
	Tag      string
	RepoName string
	Agent    string
	State    SessionState
}

// HistoricalTag is a recently touched Pair tag with no live zellij session.
// Picker metadata is populated by HistorySource.Scan; the decision path reads
// only Tag.
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
	// zellij session names. Empty preserves the legacy unscoped behavior.
	SessionNames map[string]string
}

// ListRow is one `pair list`/`ls` row: a Pair session with its resolved
// agent and reuse state, plus the live client count (0 for detached/exited) so
// the pure formatter can render "attached (N clients)" (#99 M5a).
type ListRow struct {
	Session string
	Agent   string
	State   SessionState
	Clients int
}
