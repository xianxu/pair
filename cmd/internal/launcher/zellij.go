package launcher

import (
	"bytes"
	"context"
	"os/exec"
	"sort"
	"strings"
	"time"
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

// ZellijSource reads zellij session state through the zellij CLI.
type ZellijSource struct {
	Path string
}

func (s ZellijSource) Snapshot() ([]Session, error) {
	return s.SnapshotContext(context.Background())
}

// SnapshotContext is Snapshot with the caller's cancellation.
//
// Cost, stated because Couch's refresh depends on it: two `list-sessions` runs
// plus one `action list-clients` per non-exited session ON THE HOST -- N is
// every pair session, not just the ones the caller cares about.
func (s ZellijSource) SnapshotContext(ctx context.Context) ([]Session, error) {
	short, err := s.runContext(ctx, "list-sessions", "--short")
	if err != nil {
		short = nil
	}
	raw, err := s.runContext(ctx, "list-sessions", "--no-formatting")
	if err != nil {
		raw = nil
	}
	exited := exitedSessions(string(raw))
	var out []Session
	for _, name := range lines(string(short)) {
		if !isPairSessionName(name) {
			continue
		}
		state := SessionDetached
		if exited[name] {
			state = SessionExited
		} else if s.clientCountContext(ctx, name) > 0 {
			state = SessionAttached
		}
		out = append(out, Session{Name: name, State: state})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s ZellijSource) clientCount(session string) int {
	return s.clientCountContext(context.Background(), session)
}

func (s ZellijSource) clientCountContext(ctx context.Context, session string) int {
	out, err := s.runContext(ctx, "--session", session, "action", "list-clients")
	if err != nil {
		return 0
	}
	return parseClientCount(string(out)) // one parser for both call sites (ARCH-DRY)
}

func (s ZellijSource) run(args ...string) ([]byte, error) {
	return s.runContext(context.Background(), args...)
}

func (s ZellijSource) runContext(ctx context.Context, args ...string) ([]byte, error) {
	path := s.Path
	if path == "" {
		path = "zellij"
	}
	ctx, cancel := context.WithTimeout(ctx, zellijQueryTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func exitedSessions(raw string) map[string]bool {
	out := map[string]bool{}
	for _, line := range lines(raw) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if strings.Contains(line, "EXITED") {
			out[fields[0]] = true
		}
	}
	return out
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
