package launcher

import "testing"

func TestScopedPaths(t *testing.T) {
	scope, err := ResolveRepoScope("/Users/x/workspace/pair")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}
	paths := NewScopedPaths("/data", scope, "work")
	scopeDir := "/data/repos/" + scope.Key

	checks := map[string]string{
		"ScopeDir":          scopeDir,
		"Meta":              scopeDir + "/meta.json",
		"Ledger":            scopeDir + "/ledger-work.jsonl",
		"Draft":             scopeDir + "/draft-work.md",
		"Log":               scopeDir + "/log-work.md",
		"QueueDir":          scopeDir + "/queue-work",
		"Agent":             scopeDir + "/agent-work",
		"AgentPID":          scopeDir + "/agent-pid-work",
		"AgentOutput":       scopeDir + "/agent-output-work",
		"AgentPicks":        scopeDir + "/agent-picks-work",
		"AdaptLog":          scopeDir + "/adapt-work.jsonl",
		"OuterTTY":          scopeDir + "/outer-tty-work",
		"NvimDraftPID":      scopeDir + "/nvim-pid-work-draft",
		"NvimScrollbackPID": scopeDir + "/nvim-pid-work-scrollback",
	}

	got := map[string]string{
		"ScopeDir":          paths.ScopeDir(),
		"Meta":              paths.Meta(),
		"Ledger":            paths.Ledger(),
		"Draft":             paths.Draft(),
		"Log":               paths.Log(),
		"QueueDir":          paths.QueueDir(),
		"Agent":             paths.Agent(),
		"AgentPID":          paths.AgentPID(),
		"AgentOutput":       paths.AgentOutput(),
		"AgentPicks":        paths.AgentPicks(),
		"AdaptLog":          paths.AdaptLog(),
		"OuterTTY":          paths.OuterTTY(),
		"NvimDraftPID":      paths.NvimDraftPID(),
		"NvimScrollbackPID": paths.NvimScrollbackPID(),
	}
	for name, want := range checks {
		if got[name] != want {
			t.Fatalf("%s = %q, want %q", name, got[name], want)
		}
	}
}

func TestScopedAgentPaths(t *testing.T) {
	scope, err := ResolveRepoScope("/Users/x/workspace/pair")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}
	paths := NewScopedPaths("/data", scope, "work")
	scopeDir := "/data/repos/" + scope.Key

	checks := map[string]string{
		"Config":             scopeDir + "/config-work-claude.json",
		"LegacyCodexConfig":  scopeDir + "/config-work-codex-codex.json",
		"Pane":               scopeDir + "/pane-work-claude.json",
		"ScrollbackRaw":      scopeDir + "/scrollback-work-claude.raw",
		"ScrollbackANSI":     scopeDir + "/scrollback-work-claude.ansi",
		"ScrollbackEvents":   scopeDir + "/scrollback-work-claude.events.jsonl",
		"ScrollbackViewport": scopeDir + "/scrollback-work-claude.viewport",
		"Changelog":          scopeDir + "/changelog-work-claude.md",
		"AgentDraft":         scopeDir + "/draft-work-claude.md",
	}

	got := map[string]string{
		"Config":             paths.Config("claude"),
		"LegacyCodexConfig":  paths.LegacyCodexConfig(),
		"Pane":               paths.Pane("claude"),
		"ScrollbackRaw":      paths.ScrollbackRaw("claude"),
		"ScrollbackANSI":     paths.ScrollbackANSI("claude"),
		"ScrollbackEvents":   paths.ScrollbackEvents("claude"),
		"ScrollbackViewport": paths.ScrollbackViewport("claude"),
		"Changelog":          paths.Changelog("claude"),
		"AgentDraft":         paths.AgentDraft("claude"),
	}
	for name, want := range checks {
		if got[name] != want {
			t.Fatalf("%s = %q, want %q", name, got[name], want)
		}
	}
}
