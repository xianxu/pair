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

	"github.com/xianxu/pair/cmd/internal/zellijexec"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// zellijQueryTimeout bounds one zellij CLI query.
//
// Couch's inventory refresh runs Snapshot on its periodic worker, and a hung
// zellij would otherwise block that worker forever -- the switcher would keep
// rendering its last-good projection and never notice a thread again. A query
// that does not answer promptly is treated as no answer, which is the
// fail-closed direction: a row is not proved this round rather than proved
// wrong.
const zellijQueryTimeout = 5 * time.Second

// ErrZellijNoReply distinguishes an unanswered read from an observed empty
// inventory. A fresh call is new evidence; this boundary never retries.
var ErrZellijNoReply = errors.New("zellij query returned no reply")

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
// Cost, stated because it is the expensive form: one `list-sessions` run plus
// one `action list-clients` per non-exited pair session ON THE HOST, and
// list-clients costs about 250 ms against a real detached session (pair#228).
// A caller asking about particular sessions wants SnapshotSessionsContext; one
// that needs only "not exited" wants LivenessContext.
func (s ZellijSource) SnapshotContext(ctx context.Context) ([]Session, error) {
	return s.snapshot(ctx, func(string) bool { return true }, func(string) bool { return true })
}

// SnapshotSessionsContext classifies only the named sessions: one list-clients
// per named session that is live, however many sessions the host has. Names
// that are absent from zellij's listing are absent from the result.
func (s ZellijSource) SnapshotSessionsContext(ctx context.Context, names []string) ([]Session, error) {
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}
	in := func(name string) bool { return wanted[name] }
	return s.snapshot(ctx, in, in)
}

// LivenessContext lists every pair session without asking any of them for
// clients: non-exited sessions are SessionLive. One list-sessions run, and
// nothing that scales with the host's session count.
func (s ZellijSource) LivenessContext(ctx context.Context) ([]Session, error) {
	return s.snapshot(ctx, func(string) bool { return false }, func(string) bool { return true })
}

// snapshot is the one call pattern and parser behind all three forms. keep
// decides which pair sessions appear in the result; ask decides which of the
// kept, non-exited ones are asked for their clients -- a kept session not asked
// is SessionLive, because an attach state that was not asked must not be
// guessed.
func (s ZellijSource) snapshot(ctx context.Context, ask, keep func(name string) bool) ([]Session, error) {
	raw, err := s.listings(ctx)
	if err != nil {
		return nil, err
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
	if s.Listings != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return s.Listings.Listings()
	}
	return s.fullListing(ctx)
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
// timeout and no automatic retries, without listing unrelated sessions.
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

// ZellijQueryObservation retains bounded copies of a completed CLI query.
// Stdout and Stderr each keep at most 64 KiB; byte counts describe the original
// streams, which the production parser consumes without truncation. Err is the
// caller-visible result; ExitCode retains the subprocess status (-1 if absent).
// CallerDeadline is zero when absent; EffectiveDeadline includes the query cap.
type ZellijQueryObservation struct {
	Args                              []string
	Started, Finished                 time.Time
	CallerDeadline, EffectiveDeadline time.Time
	Duration                          time.Duration
	Stdout, Stderr                    []byte
	StdoutBytes, StderrBytes          int
	StdoutTruncated, StderrTruncated  bool
	ExitCode                          int
	Err                               error
}

type zellijQueryObserverKey struct{}

// WithZellijQueryObserver scopes optional query measurement to ctx. Callbacks
// run synchronously after each direct subprocess completes and must not block.
// Concurrent operations require a synchronized observer. No query is added or
// retried, and ordinary contexts retain the original execution and output path.
func WithZellijQueryObserver(ctx context.Context, observe func(ZellijQueryObservation)) context.Context {
	return context.WithValue(ctx, zellijQueryObserverKey{}, observe)
}

func (s ZellijSource) runContext(ctx context.Context, args ...string) (output []byte, resultErr error) {
	path := s.Path
	if path == "" {
		path = "zellij"
	}
	callerDeadline, _ := ctx.Deadline()
	ctx, cancel := context.WithTimeout(ctx, zellijQueryTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if observe, _ := ctx.Value(zellijQueryObserverKey{}).(func(ZellijQueryObservation)); observe != nil {
		effectiveDeadline, _ := ctx.Deadline()
		started := time.Now()
		defer func() {
			finished := time.Now()
			exitCode := -1
			if cmd.ProcessState != nil {
				exitCode = cmd.ProcessState.ExitCode()
			}
			const bound = 64 * 1024
			observe(ZellijQueryObservation{
				CallerDeadline: callerDeadline, EffectiveDeadline: effectiveDeadline,
				Args: append([]string(nil), args...), Started: started, Finished: finished,
				Duration: finished.Sub(started), ExitCode: exitCode, Err: resultErr,
				Stdout:      append([]byte(nil), stdout.Bytes()[:min(stdout.Len(), bound)]...),
				Stderr:      append([]byte(nil), stderr.Bytes()[:min(stderr.Len(), bound)]...),
				StdoutBytes: stdout.Len(), StderrBytes: stderr.Len(),
				StdoutTruncated: stdout.Len() > bound, StderrTruncated: stderr.Len() > bound,
			})
		}()
	}
	run := s.queryRun
	if run == nil {
		run = zellijexec.Run
	}
	if err := run(cmd); err != nil {
		if ctx.Err() != nil {
			err = errors.Join(err, ctx.Err())
		} else {
			// Zellij reports a successfully observed empty inventory with exit
			// 1 and this exact diagnostic. Other failures are not absence proof.
			// Only the direct process exit can be normalized. A joined collector
			// failure remains unknown even if it contains this same exit code.
			exitErr, pureExit := err.(*exec.ExitError)
			if len(args) > 0 && args[0] == "list-sessions" &&
				pureExit && exitErr.ExitCode() == 1 &&
				stdout.Len() == 0 && strings.TrimSpace(stderr.String()) == "No active zellij sessions found." {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("zellij %s: %w", strings.Join(args, " "), err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("zellij %s: %w", strings.Join(args, " "), err)
	}
	if len(bytes.TrimSpace(stdout.Bytes())) == 0 {
		return nil, fmt.Errorf("zellij %s: %w", strings.Join(args, " "), ErrZellijNoReply)
	}
	return stdout.Bytes(), nil
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
