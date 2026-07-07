package launcher

import "path/filepath"

// ScopedPaths derives every tag-scoped sidecar path underneath one repo scope
// directory. It is pure; callers decide when to use legacy flat fallbacks.
type ScopedPaths struct {
	GlobalDataDir string
	Scope         RepoScope
	Tag           string
}

func NewScopedPaths(globalDataDir string, scope RepoScope, tag string) ScopedPaths {
	return ScopedPaths{GlobalDataDir: globalDataDir, Scope: scope, Tag: tag}
}

func (p ScopedPaths) ScopeDir() string {
	return filepath.Join(p.GlobalDataDir, "repos", p.Scope.Key)
}

func (p ScopedPaths) Meta() string { return filepath.Join(p.ScopeDir(), "meta.json") }

func (p ScopedPaths) Ledger() string {
	return filepath.Join(p.ScopeDir(), "ledger-"+p.Tag+".jsonl")
}

func (p ScopedPaths) Draft() string { return filepath.Join(p.ScopeDir(), "draft-"+p.Tag+".md") }

func (p ScopedPaths) Log() string { return filepath.Join(p.ScopeDir(), "log-"+p.Tag+".md") }

func (p ScopedPaths) QueueDir() string { return filepath.Join(p.ScopeDir(), "queue-"+p.Tag) }

func (p ScopedPaths) Agent() string { return filepath.Join(p.ScopeDir(), "agent-"+p.Tag) }

func (p ScopedPaths) AgentPID() string { return filepath.Join(p.ScopeDir(), "agent-pid-"+p.Tag) }

func (p ScopedPaths) AgentOutput() string {
	return filepath.Join(p.ScopeDir(), "agent-output-"+p.Tag)
}

func (p ScopedPaths) AgentPicks() string {
	return filepath.Join(p.ScopeDir(), "agent-picks-"+p.Tag)
}

func (p ScopedPaths) AdaptLog() string {
	return filepath.Join(p.ScopeDir(), "adapt-"+p.Tag+".jsonl")
}

func (p ScopedPaths) OuterTTY() string { return filepath.Join(p.ScopeDir(), "outer-tty-"+p.Tag) }

func (p ScopedPaths) NvimDraftPID() string {
	return filepath.Join(p.ScopeDir(), "nvim-pid-"+p.Tag+"-draft")
}

func (p ScopedPaths) NvimScrollbackPID() string {
	return filepath.Join(p.ScopeDir(), "nvim-pid-"+p.Tag+"-scrollback")
}

func (p ScopedPaths) Config(agent string) string {
	return filepath.Join(p.ScopeDir(), "config-"+p.Tag+"-"+agent+".json")
}

func (p ScopedPaths) LegacyCodexConfig() string {
	return filepath.Join(p.ScopeDir(), "config-"+p.Tag+"-codex-codex.json")
}

func (p ScopedPaths) Pane(agent string) string {
	return filepath.Join(p.ScopeDir(), "pane-"+p.Tag+"-"+agent+".json")
}

func (p ScopedPaths) ScrollbackRaw(agent string) string {
	return filepath.Join(p.ScopeDir(), "scrollback-"+p.Tag+"-"+agent+".raw")
}

func (p ScopedPaths) ScrollbackANSI(agent string) string {
	return filepath.Join(p.ScopeDir(), "scrollback-"+p.Tag+"-"+agent+".ansi")
}

func (p ScopedPaths) ScrollbackEvents(agent string) string {
	return filepath.Join(p.ScopeDir(), "scrollback-"+p.Tag+"-"+agent+".events.jsonl")
}

func (p ScopedPaths) ScrollbackViewport(agent string) string {
	return filepath.Join(p.ScopeDir(), "scrollback-"+p.Tag+"-"+agent+".viewport")
}

func (p ScopedPaths) Changelog(agent string) string {
	return filepath.Join(p.ScopeDir(), "changelog-"+p.Tag+"-"+agent+".md")
}

func (p ScopedPaths) AgentDraft(agent string) string {
	return filepath.Join(p.ScopeDir(), "draft-"+p.Tag+"-"+agent+".md")
}
