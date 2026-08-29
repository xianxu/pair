package launcher

import "github.com/xianxu/pair/cmd/internal/artifactpath"

// Per-tag config + agent-transcript path derivations, and the one config
// migration rule the shell launcher carries (#99 M1, ported from bin/pair-shell).
// These are pure path/decision helpers; the stat / jq-read / mv effects sit on
// the Runtime seam (M2).

// CanonicalConfigPath is where a launch writes config-<tag>-<agent>.json.
func CanonicalConfigPath(dataDir, tag, agent string) string {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return ""
	}
	path, _ := paths.ConfigChecked(agent)
	return path
}

// LegacyCodexConfigPath is the pre-#67 doubled shape config-<tag>-codex-codex.json
// that older Codex sessions on disk still use.
func LegacyCodexConfigPath(dataDir, tag string) string {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return ""
	}
	return paths.LegacyCodexConfig()
}

// ShouldMigrateLegacyCodex decides whether resolve_config_file should rename the
// legacy Codex config to the canonical name: only when the canonical file is
// absent, the agent is codex, the legacy file exists, and its JSON declares
// `"agent": "codex"`. This is a narrow, agent-checked path — never a glob
// resolver — so an unrelated stale file can't silently win.
func ShouldMigrateLegacyCodex(canonicalExists bool, agent string, legacyExists bool, legacyAgentField string) bool {
	return !canonicalExists && agent == "codex" && legacyExists && legacyAgentField == "codex"
}
