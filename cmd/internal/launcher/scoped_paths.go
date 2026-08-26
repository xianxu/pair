package launcher

import (
	"path/filepath"
	"strings"
)

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

func (p ScopedPaths) AgentDefault(agent string) string {
	return AgentDefaultPath(p.ScopeDir(), agent)
}

func (p ScopedPaths) AgentReady(agent string) string {
	return AgentReadyPath(p.ScopeDir(), p.Tag, agent)
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

func (p ScopedPaths) ThreadClaim() string {
	return filepath.Join(p.ScopeDir(), "thread-claim-"+p.Tag+".json")
}

// OwnsTagArtifact is the single inventory for filenames whose identity is a
// Pair tag. Allocation and migration use it to prevent a new thread from
// adopting any durable sidecar that already belongs to another one.
func OwnsTagArtifact(name, tag string) bool {
	exact := map[string]bool{
		"ledger-" + tag + ".jsonl":        true,
		"draft-" + tag + ".md":            true,
		"log-" + tag + ".md":              true,
		"queue-" + tag:                    true,
		"agent-" + tag:                    true,
		"agent-pid-" + tag:                true,
		"agent-output-" + tag:             true,
		"agent-picks-" + tag:              true,
		"adapt-" + tag + ".jsonl":         true,
		"outer-tty-" + tag:                true,
		"nvim-pid-" + tag + "-draft":      true,
		"nvim-pid-" + tag + "-scrollback": true,
		"thread-claim-" + tag + ".json":   true,
	}
	if exact[name] {
		return true
	}
	for _, prefix := range []string{
		"config-" + tag + "-",
		"agent-ready-" + tag + "-",
		"pane-" + tag + "-",
		"scrollback-" + tag + "-",
		"changelog-" + tag + "-",
		"draft-" + tag + "-",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
