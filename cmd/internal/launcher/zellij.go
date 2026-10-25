package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// zellijQueryTimeout bounds one logical Zellij read across all its attempts.
//
// Couch's inventory refresh runs Snapshot on its periodic worker, and a hung
// zellij would otherwise block that worker forever -- the switcher would keep
// rendering its last-good projection and never notice a thread again. A query
// that does not answer promptly is treated as no answer, which is the
// fail-closed direction: a row is not proved this round rather than proved
// wrong.
const zellijQueryTimeout = zellijquery.Timeout

// ErrZellijNoReply distinguishes an unanswered read from an observed empty
// inventory. Eligible idempotent reads recover at most three attempts before
// returning this error; a later explicit call begins a new observation.
var ErrZellijNoReply = zellijquery.ErrNoReply

var errSessionNameTooLong = errors.New("session name too long")

// ZellijSource reads zellij session state through the zellij CLI.
type ZellijSource struct {
	Path string
	// queryRun is the synchronous process-ownership boundary. Nil uses the OS collector.
	queryRun func(*exec.Cmd) error
	// Listings, when set, answers the full list-sessions reply from a poller's
	// latest snapshot instead of running zellij (pair#618). Nil lists on every
	// read. Per-session queries (list-clients) always run.
	Listings *ListingPoller
}

func (s ZellijSource) Snapshot() ([]Session, error) {
	return s.SnapshotContext(context.Background())
}

// SnapshotContext is Snapshot with the caller's cancellation: every pair
// session, each classified attached / detached / exited.
//
// Cost: one logical `list-sessions` read plus one `action list-clients` read
// per non-exited pair session ON THE HOST. Each may use up to three attempts;
//
// list-clients costs about 250 ms against a real detached session (pair#228).
// A caller asking about particular sessions wants SnapshotSessionsContext; one
// that needs only "not exited" wants LivenessContext.
func (s ZellijSource) SnapshotContext(ctx context.Context) ([]Session, error) {
	return s.snapshot(ctx, func(string) bool { return true }, func(string) bool { return true }, nil)
}

// SnapshotSessionsContext classifies only the named sessions: one logical
// list-clients read (up to three attempts) per named live session. Names
// that are absent from zellij's listing are absent from the result.
func (s ZellijSource) SnapshotSessionsContext(ctx context.Context, names []string) ([]Session, error) {
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}
	in := func(name string) bool { return wanted[name] }
	return s.snapshot(ctx, in, in, nil)
}

// LivenessContext lists every pair session without asking any of them for
// clients: non-exited sessions are SessionLive. One logical list-sessions read
// (up to three attempts), and
// nothing that scales with the host's session count.
func (s ZellijSource) LivenessContext(ctx context.Context) ([]Session, error) {
	return s.snapshot(ctx, func(string) bool { return false }, func(string) bool { return true }, nil)
}

// LivenessEvidenceContext pairs the liveness projection with its judged listing.
func (s ZellijSource) LivenessEvidenceContext(ctx context.Context) ([]Session, ListingEvidence, error) {
	var evidence ListingEvidence
	sessions, err := s.snapshot(ctx, func(string) bool { return false }, func(string) bool { return true }, &evidence)
	return sessions, evidence, err
}

// snapshot is the one call pattern and parser behind all three forms. keep
// decides which pair sessions appear in the result; ask decides which of the
// kept, non-exited ones are asked for their clients -- a kept session not asked
// is SessionLive, because an attach state that was not asked must not be
// guessed.
func (s ZellijSource) snapshot(ctx context.Context, ask, keep func(name string) bool, evidence *ListingEvidence) ([]Session, error) {
	raw, at, err := s.listingsAt(ctx)
	if err != nil {
		return nil, err
	}
	if evidence != nil {
		*evidence = newListingEvidence(raw, at)
	}
	states, err := sessionListingStates(raw)
	if err != nil {
		return nil, err
	}
	var names []string
	for name := range states {
		if isPairSessionName(name) && keep(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var out []Session
	for _, name := range names {
		var state SessionState
		switch {
		case states[name]:
			state = SessionExited
		case !ask(name):
			state = SessionLive
		default:
			clients, err := s.ClientCountContext(ctx, name)
			if err != nil {
				return nil, err
			}
			state = SessionDetached
			if clients > 0 {
				state = SessionAttached
			}
		}
		out = append(out, Session{Name: name, State: state})
	}
	return out, nil
}

// listings is one coherent observation; independent short/full replies cannot
// prove agreement while unrelated sessions start or end between subprocesses.
func (s ZellijSource) listings(ctx context.Context) ([]byte, error) {
	raw, _, err := s.listingsAt(ctx)
	return raw, err
}

func (s ZellijSource) listingsAt(ctx context.Context) ([]byte, time.Time, error) {
	if s.Listings != nil {
		if err := ctx.Err(); err != nil {
			return nil, time.Time{}, err
		}
		return s.Listings.listingsAt()
	}
	raw, err := s.fullListing(ctx)
	return raw, time.Now(), err
}

func (s ZellijSource) fullListing(ctx context.Context) ([]byte, error) {
	raw, err := s.runContext(ctx, "list-sessions", "--no-formatting")
	if err != nil {
		return nil, err
	}
	if err := validateSessionListing(raw, false); err != nil {
		return nil, err
	}
	return raw, nil
}

// ClientCountContext observes the attached client count for one exact session.
// Header-only output proves zero clients; an empty successful reply remains
// unknown. It shares the inventory query boundary, including cancellation,
// timeout and bounded known-corrupt-reply recovery, without listing unrelated sessions.
func (s ZellijSource) ClientCountContext(ctx context.Context, session string) (int, error) {
	out, err := s.runContext(ctx, "--session", session, "action", "list-clients")
	if err != nil {
		return 0, err
	}
	clients, err := zellijreply.ParseClients(out)
	if err != nil {
		return 0, fmt.Errorf("observe session clients: %w", err)
	}
	return len(clients), nil
}

func (s ZellijSource) run(args ...string) ([]byte, error) {
	return s.runContext(context.Background(), args...)
}

// ZellijQueryObservation retains the shared seam's per-attempt evidence.
type ZellijQueryObservation = zellijquery.Observation

func WithZellijQueryObserver(ctx context.Context, observe func(ZellijQueryObservation)) context.Context {
	return zellijquery.WithObserver(ctx, observe)
}

func (s ZellijSource) runContext(ctx context.Context, args ...string) ([]byte, error) {
	result, err := (zellijquery.Source{Path: s.Path, Run: s.queryRun}).ReadContext(ctx, args...)
	// Preserve the launcher's historical refusal for an unanswered unsupported
	// request; screen-reading consumers use the shared source directly.
	if err == nil && result.ProcessErr == nil && zellijReadKind(args) == zellijUnknownReply && len(bytes.TrimSpace(result.Stdout)) == 0 {
		err = ErrZellijNoReply
	}
	if err != nil {
		return nil, zellijquery.Error(args, err)
	}
	return result.Stdout, nil
}

func lines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
