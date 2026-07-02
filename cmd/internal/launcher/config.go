package launcher

import (
	"path/filepath"
	"strings"
)

// Per-tag config + agent-transcript path derivations, and the one config
// migration rule the shell launcher carries (#99 M1, ported from bin/pair-shell).
// These are pure path/decision helpers; the stat / jq-read / mv effects sit on
// the Runtime seam (M2).

// CanonicalConfigPath is where a launch writes config-<tag>-<agent>.json.
func CanonicalConfigPath(dataDir, tag, agent string) string {
	return filepath.Join(dataDir, "config-"+tag+"-"+agent+".json")
}

// LegacyCodexConfigPath is the pre-#67 doubled shape config-<tag>-codex-codex.json
// that older Codex sessions on disk still use.
func LegacyCodexConfigPath(dataDir, tag string) string {
	return filepath.Join(dataDir, "config-"+tag+"-codex-codex.json")
}

// ShouldMigrateLegacyCodex decides whether resolve_config_file should rename the
// legacy Codex config to the canonical name: only when the canonical file is
// absent, the agent is codex, the legacy file exists, and its JSON declares
// `"agent": "codex"`. This is a narrow, agent-checked path — never a glob
// resolver — so an unrelated stale file can't silently win.
func ShouldMigrateLegacyCodex(canonicalExists bool, agent string, legacyExists bool, legacyAgentField string) bool {
	return !canonicalExists && agent == "codex" && legacyExists && legacyAgentField == "codex"
}

// encodeCwd mirrors the shell's `tr ./ -`: both '.' and '/' become '-'. Used to
// key claude's per-project transcript directory off the cwd.
func encodeCwd(cwd string) string {
	return strings.Map(func(r rune) rune {
		if r == '.' || r == '/' {
			return '-'
		}
		return r
	}, cwd)
}

// ClaudeTranscriptPath is where claude stores a session's jsonl transcript:
// $HOME/.claude/projects/<encoded-cwd>/<sid>.jsonl.
func ClaudeTranscriptPath(home, cwd, sid string) string {
	return filepath.Join(home, ".claude", "projects", encodeCwd(cwd), sid+".jsonl")
}

// AgyConversationPath is where agy stores a conversation db:
// $HOME/.gemini/antigravity-cli/conversations/<sid>.db.
func AgyConversationPath(home, sid string) string {
	return filepath.Join(home, ".gemini", "antigravity-cli", "conversations", sid+".db")
}

// CodexSessionsDir is the directory the codex session probe globs for `*<sid>*`
// (the match itself is IO — a find over this dir on the Runtime seam).
func CodexSessionsDir(home string) string {
	return filepath.Join(home, ".codex", "sessions")
}
