package launcher

import "testing"

func TestConfigPaths(t *testing.T) {
	if got := CanonicalConfigPath("/dd", "t", "claude"); got != "/dd/config-t-claude.json" {
		t.Errorf("canonical = %q", got)
	}
	if got := LegacyCodexConfigPath("/dd", "t"); got != "/dd/config-t-codex-codex.json" {
		t.Errorf("legacy = %q", got)
	}
}

func TestShouldMigrateLegacyCodex(t *testing.T) {
	// Migrate only when canonical absent, agent codex, legacy present + declares codex.
	if !ShouldMigrateLegacyCodex(false, "codex", true, "codex") {
		t.Error("should migrate the agent-checked legacy codex config")
	}
	// Never when the canonical already exists.
	if ShouldMigrateLegacyCodex(true, "codex", true, "codex") {
		t.Error("canonical present → no migration")
	}
	// Never for a non-codex agent.
	if ShouldMigrateLegacyCodex(false, "claude", true, "codex") {
		t.Error("non-codex agent → no migration")
	}
	// Never when the legacy file's declared agent isn't codex (guards against a
	// stale unrelated file silently winning).
	if ShouldMigrateLegacyCodex(false, "codex", true, "claude") {
		t.Error("legacy agent field mismatch → no migration")
	}
	if ShouldMigrateLegacyCodex(false, "codex", false, "") {
		t.Error("legacy absent → no migration")
	}
}

func TestEncodeCwd(t *testing.T) {
	// tr ./ - : both '.' and '/' become '-'.
	if got := encodeCwd("/Users/x/workspace/parley.nvim"); got != "-Users-x-workspace-parley-nvim" {
		t.Errorf("encodeCwd = %q", got)
	}
}

func TestAgentTranscriptPaths(t *testing.T) {
	if got := ClaudeTranscriptPath("/home", "/Users/x/repo", "sid1"); got != "/home/.claude/projects/-Users-x-repo/sid1.jsonl" {
		t.Errorf("claude transcript = %q", got)
	}
	if got := AgyConversationPath("/home", "sid1"); got != "/home/.gemini/antigravity-cli/conversations/sid1.db" {
		t.Errorf("agy conversation = %q", got)
	}
	if got := CodexSessionsDir("/home"); got != "/home/.codex/sessions" {
		t.Errorf("codex sessions dir = %q", got)
	}
}
