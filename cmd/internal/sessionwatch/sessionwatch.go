package sessionwatch

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	uuidRE    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	endUUIDRE = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
)

// AgentSpec describes one async session-file discovery contract.
type AgentSpec struct {
	Agent    string
	Home     string
	WatchDir string
}

// SessionID is the outcome of matching a candidate session file path.
type SessionID struct {
	Matched  bool
	NearMiss bool
	ID       string
	Path     string
}

// ConfigPayload is the restart config written as config-<tag>-<agent>.json.
type ConfigPayload struct {
	Agent     string   `json:"agent"`
	Args      []string `json:"args"`
	SessionID string   `json:"session_id"`
}

// SpecForAgent returns the async watcher contract for agents that need it.
func SpecForAgent(agent, home string) (AgentSpec, bool) {
	switch agent {
	case "codex":
		return AgentSpec{
			Agent:    agent,
			Home:     home,
			WatchDir: filepath.Join(home, ".codex", "sessions"),
		}, true
	case "agy":
		return AgentSpec{
			Agent:    agent,
			Home:     home,
			WatchDir: filepath.Join(home, ".gemini", "antigravity-cli", "conversations"),
		}, true
	default:
		return AgentSpec{}, false
	}
}

// Match checks whether path belongs to the agent's session-file shape and, if
// so, extracts the session id or reports a near miss.
func (s AgentSpec) Match(path string) SessionID {
	switch s.Agent {
	case "codex":
		prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
		clean := filepath.Clean(path)
		if !strings.HasPrefix(clean, prefix) {
			return SessionID{}
		}
		base := filepath.Base(clean)
		if !strings.HasPrefix(base, "rollout-") || !strings.HasSuffix(base, ".jsonl") {
			return SessionID{}
		}
		stem := strings.TrimSuffix(base, ".jsonl")
		if match := endUUIDRE.FindStringSubmatch(stem); len(match) == 2 {
			return SessionID{Matched: true, ID: match[1], Path: path}
		}
		return SessionID{Matched: true, NearMiss: true, Path: path}
	case "agy":
		prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
		clean := filepath.Clean(path)
		if !strings.HasPrefix(clean, prefix) {
			return SessionID{}
		}
		base := filepath.Base(clean)
		if !strings.HasSuffix(base, ".db") {
			return SessionID{}
		}
		id := strings.TrimSuffix(base, ".db")
		if uuidRE.MatchString(id) {
			return SessionID{Matched: true, ID: id, Path: path}
		}
		return SessionID{Matched: true, NearMiss: true, Path: path}
	default:
		return SessionID{}
	}
}

// StripResumeArgs removes resume bindings from args before they are persisted;
// the session_id field is the canonical store for that binding.
func StripResumeArgs(agent string, args []string) []string {
	stripped := make([]string, 0, len(args))
	i := 0
	if agent == "codex" && len(args) >= 2 && args[0] == "resume" {
		i = 2
	}
	for i < len(args) {
		if args[i] == "--resume" {
			i += 2
			continue
		}
		stripped = append(stripped, args[i])
		i++
	}
	return stripped
}

// ConfigJSON renders the restart config with structured JSON encoding.
func ConfigJSON(payload ConfigPayload) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
