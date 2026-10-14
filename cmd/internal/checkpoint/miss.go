package checkpoint

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Miss is what a compaction that didn't finish leaves on its thread (#522):
// one visible line, never a blocking state. The request is gone; the line
// names the continuation, why it didn't finish, and whether the new
// conversation may still be running. The next start or compaction clears it.
type Miss struct {
	RequestID string    `json:"request_id"`
	Path      string    `json:"path"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
	// TargetMayExist: the new conversation's helper may be alive (its start
	// is unresolved, or stopping it failed). Relaunch parks it exactly; every
	// other start is refused by the thread's incarnation guard.
	TargetMayExist bool `json:"target_may_exist,omitempty"`
}

// MissOf is the line a request leaves when it ends without finishing. Its
// path is the repository's committed checkpoint, deliberately not Couch's
// private copy (which the launch prompt and PreviousSession name): the private
// copy is one per thread and the next compaction replaces it, while the
// committed file is what the operator can find, read and hand on.
func MissOf(r Request, reason string, at time.Time, targetMayExist bool) Miss {
	return Miss{RequestID: r.ID, Path: r.Checkpoint.SourcePath, Reason: boundedFailure(reason), At: at, TargetMayExist: targetMayExist}
}

func (m Miss) Validate() error {
	if !boundedIdentity(m.RequestID) || m.At.IsZero() {
		return errors.New("invalid continuation miss")
	}
	if len(m.Path) > 4096 || !utf8.ValidString(m.Path) || strings.ContainsRune(m.Path, 0) {
		return errors.New("invalid continuation miss path")
	}
	if m.Reason == "" || len(m.Reason) > MaxFailureBytes || !utf8.ValidString(m.Reason) || strings.ContainsRune(m.Reason, 0) {
		return errors.New("invalid continuation miss reason")
	}
	return nil
}

// Line is the one visible line: what happened, where the continuation is,
// and the one action for the slot's state. running says whether the slot has
// a live conversation now (the old one, when the miss came before park).
func (m Miss) Line(slot string, running bool) string {
	reason := strings.Join(strings.Fields(m.Reason), " ")
	head := fmt.Sprintf("%s: compaction didn't finish: %s; continuation at %s", slot, reason, m.Path)
	switch {
	case m.TargetMayExist:
		return head + "; the new conversation may be running. Relaunch the slot."
	case running:
		return head + ". Compact again when ready."
	default:
		return head + ". Resume the slot to return to the previous conversation."
	}
}

func boundedFailure(s string) string {
	if len(s) > MaxFailureBytes {
		s = s[:MaxFailureBytes]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	s = strings.ReplaceAll(s, "\x00", "")
	if s == "" {
		s = "unknown failure"
	}
	return s
}
