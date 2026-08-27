// Package transcript resolves an agent's on-disk session transcript path and
// the session id pair recorded for it. Single source for both pair-slug and
// pair-context (ARCH-DRY) — extracted from cmd/pair-slug/main.go.
package transcript

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

const codexSessionMetaLineLimit = 1 << 20

// ClaudePathEncoder mirrors nvim's `cwd:gsub('[./]', '-')` for the
// ~/.claude/projects/<encoded-cwd>/ directory name.
var ClaudePathEncoder = strings.NewReplacer(".", "-", "/", "-")

var codexRolloutRE = regexp.MustCompile(`^(.*/\.codex/sessions/.*/rollout-.*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl)$`)

// CodexSessionIDFromPath extracts the native session id from a Codex rollout
// transcript path. It returns "" for non-Codex or malformed paths.
func CodexSessionIDFromPath(path string) string {
	m := codexRolloutRE.FindStringSubmatch(path)
	if len(m) < 3 {
		return ""
	}
	return m[2]
}

// CodexRootSessionID authorizes a root Codex rollout from its path and first
// JSONL event. Filename UUIDs identify candidates; session_meta establishes
// whether the candidate is the operator's resumable root session.
func CodexRootSessionID(path string, firstEvent []byte) string {
	sid := CodexSessionIDFromPath(path)
	if sid == "" {
		return ""
	}
	var event struct {
		Type    string `json:"type"`
		Payload struct {
			ID             string          `json:"id"`
			ParentThreadID *string         `json:"parent_thread_id"`
			Source         json.RawMessage `json:"source"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(firstEvent, &event); err != nil || event.Type != "session_meta" || event.Payload.ID != sid || event.Payload.ParentThreadID != nil {
		return ""
	}
	var source string
	if err := json.Unmarshal(event.Payload.Source, &source); err != nil || (source != "cli" && source != "exec") {
		return ""
	}
	return sid
}

// ReadCodexRootSessionID reads one bounded, newline-terminated JSONL event and
// delegates the semantic decision to CodexRootSessionID. It fails closed when
// the rollout is incomplete, oversized, unreadable, or not a root session.
func ReadCodexRootSessionID(path string) string {
	line, err := ReadFirstEvent(path)
	if err != nil {
		return ""
	}
	return CodexRootSessionID(path, line)
}

// ReadFirstEvent returns one bounded, newline-terminated JSONL event.
func ReadFirstEvent(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	line, err := bufio.NewReader(io.LimitReader(f, codexSessionMetaLineLimit+1)).ReadBytes('\n')
	if err != nil || len(line) > codexSessionMetaLineLimit {
		if err == nil {
			err = io.ErrShortBuffer
		}
		return nil, err
	}
	return line, nil
}

// SessionID reads the session id pair recorded for (tag, agent) in
// config-<tag>-<agent>.json (written by the launcher / pair session-watch).
func SessionID(dataDir, tag, agent, home string) string {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return ""
	}
	configPath, err := paths.ConfigChecked(agent)
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var c struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(b, &c) != nil {
		return ""
	}
	if agent == "codex" {
		path := Resolve(agent, c.SessionID, "", home)
		if path == "" || ReadCodexRootSessionID(path) != c.SessionID {
			return ""
		}
	}
	return c.SessionID
}

// Resolve returns the on-disk transcript path for (agent, sid), or "" if it
// can't be located. cwd is only needed for claude (project-dir encoding).
func Resolve(agent, sid, cwd, home string) string {
	switch agent {
	case "codex":
		matches, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "rollout-*"+sid+"*.jsonl"))
		if len(matches) > 0 {
			return matches[0]
		}
		return ""
	case "agy":
		return filepath.Join(home, ".gemini", "antigravity-cli", "brain", sid, ".system_generated", "logs", "transcript.jsonl")
	case "muse":
		matches, _ := filepath.Glob(filepath.Join(home, ".local", "share", "muse", "sessions", "*", "*", "*", sid, "session.jsonl"))
		for _, m := range matches {
			if !strings.Contains(m, string(filepath.Separator)+"subagent"+string(filepath.Separator)) {
				return m
			}
		}
		// Flat fallback: direct parent dir without date hierarchy (tests / future layout)
		candidate := filepath.Join(home, ".local", "share", "muse", "sessions", sid, "session.jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return ""
	default: // claude
		return filepath.Join(home, ".claude", "projects", ClaudePathEncoder.Replace(cwd), sid+".jsonl")
	}
}
