package launcher

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
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

// Validate proves the composite scope/tag boundary before a caller performs
// artifact IO. Existing constructors remain pure so legacy readers can derive
// paths, while new transaction/helper boundaries can fail closed first.
func (p ScopedPaths) Validate() error {
	_, err := artifactpath.Resolve(p.address())
	return err
}

func (p ScopedPaths) address() artifactpath.Address {
	return artifactpath.Address{DataDir: p.GlobalDataDir, RepoScope: p.Scope.Key, Tag: p.Tag}
}

func (p ScopedPaths) resolved() artifactpath.Paths {
	paths, err := artifactpath.Resolve(p.address())
	if err != nil {
		panic(err)
	}
	return paths
}

func (p ScopedPaths) ScopeDir() string {
	dir, err := artifactpath.ResolveScopeDir(p.GlobalDataDir, p.Scope.Key)
	if err != nil {
		panic(err)
	}
	return dir
}

func (p ScopedPaths) Meta() string { return p.resolved().Meta() }

func (p ScopedPaths) Ledger() string {
	return p.resolved().Ledger()
}

func (p ScopedPaths) LifecycleJournal() string { return p.resolved().LifecycleJournal() }

func (p ScopedPaths) Draft() string { return p.resolved().Draft() }

func (p ScopedPaths) Log() string { return p.resolved().Log() }

func (p ScopedPaths) QueueDir() string { return p.resolved().QueueDir() }

func (p ScopedPaths) Agent() string { return p.resolved().Agent() }

func (p ScopedPaths) AgentPID() string { return p.resolved().AgentPID() }

func (p ScopedPaths) AgentOutput() string {
	return p.resolved().AgentOutput()
}

func (p ScopedPaths) AgentPicks() string {
	return p.resolved().AgentPicks()
}

func (p ScopedPaths) AdaptLog() string {
	return p.resolved().AdaptLog()
}

func (p ScopedPaths) OuterTTY() string { return p.resolved().OuterTTY() }

func (p ScopedPaths) NvimDraftPID() string {
	return p.resolved().NvimPID("draft")
}

func (p ScopedPaths) NvimScrollbackPID() string {
	return p.resolved().NvimPID("scrollback")
}

func (p ScopedPaths) Config(agent string) string {
	return p.resolved().Config(agent)
}

func (p ScopedPaths) LegacyCodexConfig() string {
	return p.resolved().LegacyCodexConfig()
}

func (p ScopedPaths) AgentDefault(agent string) string {
	return AgentDefaultPath(p.ScopeDir(), agent)
}

func (p ScopedPaths) AgentReady(agent string) string {
	return p.resolved().AgentReady(agentDefaultPathComponent(agent))
}

func (p ScopedPaths) Pane(agent string) string {
	return p.resolved().Pane(agent)
}

func (p ScopedPaths) ScrollbackRaw(agent string) string {
	return p.resolved().ScrollbackRaw(agent)
}

func (p ScopedPaths) ScrollbackANSI(agent string) string {
	return p.resolved().ScrollbackANSI(agent)
}

func (p ScopedPaths) ScrollbackEvents(agent string) string {
	return p.resolved().ScrollbackEvents(agent)
}

func (p ScopedPaths) ScrollbackViewport(agent string) string {
	return p.resolved().ScrollbackViewport(agent)
}

func (p ScopedPaths) Changelog(agent string) string {
	return p.resolved().Changelog(agent)
}

func (p ScopedPaths) AgentDraft(agent string) string {
	return p.resolved().AgentDraft(agent)
}

func (p ScopedPaths) ThreadClaim() string {
	return p.resolved().ThreadClaim()
}

func (p ScopedPaths) SessionBindings() string {
	return p.resolved().SessionBindings()
}

// OwnsTagArtifact recognizes every tag-bearing filename in a Pair-owned scope,
// including families introduced by non-Go consumers. Pair filenames delimit a
// tag with '-' on the left and end, '.', or '-' on the right. This boundary
// rule covers future families without a second hand-maintained prefix enum;
// conservative false positives are safe because allocation retries a random
// opaque tag.
func OwnsTagArtifact(name, tag string) bool {
	if tag == "" {
		return false
	}
	for offset := 0; offset <= len(name)-len(tag); {
		relative := strings.Index(name[offset:], tag)
		if relative < 0 {
			return false
		}
		start := offset + relative
		end := start + len(tag)
		left := start == 0 || name[start-1] == '-'
		right := end == len(name) || name[end] == '.' || name[end] == '-'
		if left && right {
			return true
		}
		offset = start + 1
	}
	return false
}
