// Package transcript resolves an agent's on-disk session transcript path and
// the session id pair recorded for it. Single source for both pair-slug and
// pair-context (ARCH-DRY) — extracted from cmd/pair-slug/main.go.
package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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

// SessionID reads the session id pair recorded for (tag, agent) in
// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch).
func SessionID(dataDir, tag, agent string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "config-"+tag+"-"+agent+".json"))
	if err != nil {
		return ""
	}
	var c struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(b, &c) != nil {
		return ""
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
