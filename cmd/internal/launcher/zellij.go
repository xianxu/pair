package launcher

import (
	"bytes"
	"os/exec"
	"sort"
	"strings"
)

// ZellijSource reads zellij session state through the zellij CLI.
type ZellijSource struct {
	Path string
}

func (s ZellijSource) Snapshot() ([]Session, error) {
	short, err := s.run("list-sessions", "--short")
	if err != nil {
		short = nil
	}
	raw, err := s.run("list-sessions", "--no-formatting")
	if err != nil {
		raw = nil
	}
	exited := exitedSessions(string(raw))
	var out []Session
	for _, name := range lines(string(short)) {
		if !strings.HasPrefix(name, "pair-") {
			continue
		}
		state := SessionDetached
		if exited[name] {
			state = SessionExited
		} else if s.clientCount(name) > 0 {
			state = SessionAttached
		}
		out = append(out, Session{Name: name, State: state})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s ZellijSource) clientCount(session string) int {
	out, err := s.run("--session", session, "action", "list-clients")
	if err != nil {
		return 0
	}
	lines := lines(string(out))
	if len(lines) <= 1 {
		return 0
	}
	return len(lines) - 1
}

func (s ZellijSource) run(args ...string) ([]byte, error) {
	path := s.Path
	if path == "" {
		path = "zellij"
	}
	cmd := exec.Command(path, args...)
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
