// Package artifactpath owns Pair's composite repo-scope/thread-tag artifact
// namespace. It deliberately imports only the standard library so every Pair
// command can consume the same validated paths without depending on launcher.
package artifactpath

import (
	"fmt"
	"path/filepath"
	"regexp"
)

var componentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Address identifies one Pair work thread's artifact namespace.
type Address struct {
	DataDir   string
	RepoScope string
	Tag       string
}

// Paths is a validated composite artifact namespace.
type Paths struct {
	scopeDir string
	tag      string
}

// ScopePaths is the validated, tag-independent portion of one selected repo
// scope. Per-path launch defaults and session-name bindings live here.
type ScopePaths struct {
	scopeDir string
}

// Binding is one exact artifact path exported to non-Go consumers.
type Binding struct {
	Name string
	Path string
}

// ResolveSelectedScope validates a scope directory already selected by Pair.
func ResolveSelectedScope(scopeDir string) (ScopePaths, error) {
	if !filepath.IsAbs(scopeDir) {
		return ScopePaths{}, fmt.Errorf("pair scope directory must be absolute")
	}
	return ScopePaths{scopeDir: filepath.Clean(scopeDir)}, nil
}

func (p ScopePaths) SessionBindings() string {
	return filepath.Join(p.scopeDir, "session-names.jsonl")
}

func (p ScopePaths) AgentDefault(agent string) (string, error) {
	if err := validateComponent("artifact component", agent); err != nil {
		return "", err
	}
	return filepath.Join(p.scopeDir, "agent-default-"+agent+".json"), nil
}

// Resolve validates an address before deriving any artifact path.
func Resolve(address Address) (Paths, error) {
	dataDir, err := validateScope(address.DataDir, address.RepoScope)
	if err != nil {
		return Paths{}, err
	}
	if err := validateComponent("pair tag", address.Tag); err != nil {
		return Paths{}, err
	}
	return Paths{
		scopeDir: filepath.Join(dataDir, "repos", address.RepoScope),
		tag:      address.Tag,
	}, nil
}

// ResolveScoped validates an already-selected scope directory carried by a
// child process. This is the consumption-side counterpart to Resolve: callers
// must not append repos/<scope> a second time.
func ResolveScoped(scopeDir, tag string) (Paths, error) {
	if !filepath.IsAbs(scopeDir) {
		return Paths{}, fmt.Errorf("pair scope directory must be absolute")
	}
	if err := validateComponent("pair tag", tag); err != nil {
		return Paths{}, err
	}
	return Paths{scopeDir: filepath.Clean(scopeDir), tag: tag}, nil
}

// ResolveScopeDir validates the tag-independent half of the namespace.
func ResolveScopeDir(dataDir, repoScope string) (string, error) {
	clean, err := validateScope(dataDir, repoScope)
	if err != nil {
		return "", err
	}
	return filepath.Join(clean, "repos", repoScope), nil
}

func validateScope(dataDir, repoScope string) (string, error) {
	if !filepath.IsAbs(dataDir) {
		return "", fmt.Errorf("pair data directory must be absolute")
	}
	if err := validateComponent("repo scope", repoScope); err != nil {
		return "", err
	}
	return filepath.Clean(dataDir), nil
}

func validateComponent(kind, value string) error {
	if !componentPattern.MatchString(value) {
		return fmt.Errorf("invalid %s %q", kind, value)
	}
	return nil
}

func (p Paths) ScopeDir() string { return p.scopeDir }
func (p Paths) Tag() string      { return p.tag }

func (p Paths) tagged(prefix, suffix string) string {
	return filepath.Join(p.ScopeDir(), prefix+p.tag+suffix)
}

func (p Paths) taggedComponent(prefix, component, suffix string) (string, error) {
	if err := validateComponent("artifact component", component); err != nil {
		return "", err
	}
	return p.tagged(prefix, "-"+component+suffix), nil
}

func mustPath(path string, err error) string {
	if err != nil {
		panic(err)
	}
	return path
}

func (p Paths) Meta() string             { return filepath.Join(p.ScopeDir(), "meta.json") }
func (p Paths) Draft() string            { return p.tagged("draft-", ".md") }
func (p Paths) Ledger() string           { return p.tagged("ledger-", ".jsonl") }
func (p Paths) Log() string              { return p.tagged("log-", ".md") }
func (p Paths) QueueDir() string         { return p.tagged("queue-", "") }
func (p Paths) Agent() string            { return p.tagged("agent-", "") }
func (p Paths) AgentPID() string         { return p.tagged("agent-pid-", "") }
func (p Paths) AgentOutput() string      { return p.tagged("agent-output-", "") }
func (p Paths) AgentPicks() string       { return p.tagged("agent-picks-", "") }
func (p Paths) OuterTTY() string         { return p.tagged("outer-tty-", "") }
func (p Paths) Parked() string           { return p.tagged("parked-", "") }
func (p Paths) AdaptLog() string         { return p.tagged("adapt-", ".jsonl") }
func (p Paths) ImageCapture() string     { return p.tagged("image-capture-", "") }
func (p Paths) ImageCaptureDone() string { return p.tagged("image-capture-", ".done") }
func (p Paths) Continuation() string     { return p.tagged("continuation-", ".md") }
func (p Paths) WorkbenchLayout() string {
	return p.tagged("workbench-layout-", "")
}
func (p Paths) LayoutMode() string      { return p.tagged("layout-mode-", "") }
func (p Paths) Restart() string         { return p.tagged("restart-", "") }
func (p Paths) DraftPane() string       { return p.tagged("draft-pane-", ".json") }
func (p Paths) SessionBindings() string { return filepath.Join(p.ScopeDir(), "session-names.jsonl") }
func (p Paths) ThreadClaim() string     { return p.tagged("thread-claim-", ".json") }
func (p Paths) Quote() string           { return p.tagged("quote-", "") }
func (p Paths) Slug() string            { return p.tagged("slug-", "") }
func (p Paths) SlugProposed() string    { return p.tagged("slug-proposed-", "") }
func (p Paths) TitlePID() string        { return p.tagged("title-pid-", "") }
func (p Paths) PairWrapPID() string     { return p.tagged("pair-wrap-pid-", "") }
func (p Paths) WrapEvents() string      { return p.tagged("wrap-events-", ".jsonl") }
func (p Paths) ScrollbackPending() string {
	return p.tagged("scrollback-pending-", ".md")
}
func (p Paths) LastLeftPane() string     { return p.tagged("last-left-pane-", "") }
func (p Paths) LastTerminalPane() string { return p.tagged("last-terminal-pane-", "") }
func (p Paths) TerminalPanes() string    { return p.tagged("terminal-panes-", "") }
func (p Paths) ZellijActions() string    { return p.tagged("zellij-actions-", ".jsonl") }
func (p Paths) ReviewOpen() string       { return p.tagged("review-", ".open") }
func (p Paths) ReviewMode() string       { return p.tagged("review-", ".mode") }
func (p Paths) ReviewTarget() string     { return p.tagged("review-target-", ".json") }
func (p Paths) ReviewContext() string    { return p.tagged("review-context-", ".md") }
func (p Paths) ReviewHandoff() string    { return p.tagged("review-handoff-", ".json") }
func (p Paths) ReviewLanded() string     { return p.tagged("review-landed-", ".json") }
func (p Paths) ReviewDefinitionRequest() string {
	return p.tagged("review-definition-request-", ".json")
}
func (p Paths) ReviewDefinitionResult() string {
	return p.tagged("review-definition-result-", ".json")
}
func (p Paths) CodexFilterKKP() string { return p.tagged("codex-filter-kkp-", "") }

// EnvironmentBindings exposes exact resolved paths to shell, Lua, and KDL
// consumers so they never reconstruct filenames from PAIR_DATA_DIR + PAIR_TAG.
func (p Paths) EnvironmentBindings(agent string) ([]Binding, error) {
	config, err := p.ConfigChecked(agent)
	if err != nil {
		return nil, err
	}
	pane, err := p.PaneChecked(agent)
	if err != nil {
		return nil, err
	}
	return []Binding{
		{Name: "PAIR_DRAFT_PATH", Path: p.Draft()},
		{Name: "PAIR_LOG_PATH", Path: p.Log()},
		{Name: "PAIR_LEDGER_PATH", Path: p.Ledger()},
		{Name: "PAIR_QUEUE_DIR", Path: p.QueueDir()},
		{Name: "PAIR_AGENT_PATH", Path: p.Agent()},
		{Name: "PAIR_AGENT_PID_PATH", Path: p.AgentPID()},
		{Name: "PAIR_AGENT_OUTPUT_PATH", Path: p.AgentOutput()},
		{Name: "PAIR_AGENT_PICKS_PATH", Path: p.AgentPicks()},
		{Name: "PAIR_AGENT_CONFIG_PATH", Path: config},
		{Name: "PAIR_AGENT_PANE_PATH", Path: pane},
		{Name: "PAIR_AGENT_READY_PATH", Path: p.AgentReady(agent)},
		{Name: "PAIR_ADAPT_LOG_PATH", Path: p.AdaptLog()},
		{Name: "PAIR_OUTER_TTY_PATH", Path: p.OuterTTY()},
		{Name: "PAIR_NVIM_DRAFT_PID_PATH", Path: p.NvimPID("draft")},
		{Name: "PAIR_NVIM_SCROLLBACK_PID_PATH", Path: p.NvimPID("scrollback")},
		{Name: "PAIR_PAIR_WRAP_PID_PATH", Path: p.PairWrapPID()},
		{Name: "PAIR_IMAGE_CAPTURE_PATH", Path: p.ImageCapture()},
		{Name: "PAIR_IMAGE_CAPTURE_DONE_PATH", Path: p.ImageCaptureDone()},
		{Name: "PAIR_QUOTE_PATH", Path: p.Quote()},
		{Name: "PAIR_SLUG_PATH", Path: p.Slug()},
		{Name: "PAIR_SLUG_PROPOSED_PATH", Path: p.SlugProposed()},
		{Name: "PAIR_SCROLLBACK_RAW_PATH", Path: p.ScrollbackRaw(agent)},
		{Name: "PAIR_SCROLLBACK_ANSI_PATH", Path: p.ScrollbackANSI(agent)},
		{Name: "PAIR_SCROLLBACK_EVENTS_PATH", Path: p.ScrollbackEvents(agent)},
		{Name: "PAIR_SCROLLBACK_VIEWPORT_PATH", Path: p.ScrollbackViewport(agent)},
		{Name: "PAIR_SCROLLBACK_PENDING_PATH", Path: p.ScrollbackPending()},
		{Name: "PAIR_CHANGELOG_PATH", Path: p.Changelog(agent)},
		{Name: "PAIR_DRAFT_PANE_PATH", Path: p.DraftPane()},
		{Name: "PAIR_LAYOUT_MODE_PATH", Path: p.LayoutMode()},
		{Name: "PAIR_WORKBENCH_LAYOUT_PATH", Path: p.WorkbenchLayout()},
		{Name: "PAIR_LAST_LEFT_PANE_PATH", Path: p.LastLeftPane()},
		{Name: "PAIR_LAST_TERMINAL_PANE_PATH", Path: p.LastTerminalPane()},
		{Name: "PAIR_TERMINAL_PANES_PATH", Path: p.TerminalPanes()},
		{Name: "PAIR_ZELLIJ_ACTIONS_PATH", Path: p.ZellijActions()},
		{Name: "PAIR_REVIEW_OPEN_PATH", Path: p.ReviewOpen()},
		{Name: "PAIR_REVIEW_MODE_PATH", Path: p.ReviewMode()},
		{Name: "PAIR_REVIEW_TARGET_PATH", Path: p.ReviewTarget()},
		{Name: "PAIR_REVIEW_CONTEXT_PATH", Path: p.ReviewContext()},
		{Name: "PAIR_REVIEW_HANDOFF_PATH", Path: p.ReviewHandoff()},
		{Name: "PAIR_REVIEW_LANDED_PATH", Path: p.ReviewLanded()},
		{Name: "PAIR_REVIEW_DEFINITION_REQUEST_PATH", Path: p.ReviewDefinitionRequest()},
		{Name: "PAIR_REVIEW_DEFINITION_RESULT_PATH", Path: p.ReviewDefinitionResult()},
		{Name: "PAIR_CODEX_FILTER_KKP_PATH", Path: p.CodexFilterKKP()},
	}, nil
}

func (p Paths) Config(agent string) string { return mustPath(p.ConfigChecked(agent)) }
func (p Paths) ConfigGlob() string         { return p.tagged("config-", "-*.json") }
func (p Paths) ConfigChecked(agent string) (string, error) {
	return p.taggedComponent("config-", agent, ".json")
}

func (p Paths) LegacyCodexConfig() string {
	return p.tagged("config-", "-codex-codex.json")
}

func (p Paths) AgentReady(agent string) string { return mustPath(p.AgentReadyChecked(agent)) }
func (p Paths) AgentReadyChecked(agent string) (string, error) {
	return p.taggedComponent("agent-ready-", agent, ".json")
}

func (p Paths) Pane(agent string) string { return mustPath(p.PaneChecked(agent)) }
func (p Paths) PaneGlob() string         { return p.tagged("pane-", "-*.json") }
func (p Paths) PaneChecked(agent string) (string, error) {
	return p.taggedComponent("pane-", agent, ".json")
}

func (p Paths) NvimPID(kind string) string { return mustPath(p.NvimPIDChecked(kind)) }
func (p Paths) NvimPIDChecked(kind string) (string, error) {
	return p.taggedComponent("nvim-pid-", kind, "")
}

func (p Paths) ParkedScrollback(timestamp string) string {
	return mustPath(p.ParkedScrollbackChecked(timestamp))
}
func (p Paths) ParkedScrollbackChecked(timestamp string) (string, error) {
	return p.taggedComponent("parked-scrollback-", timestamp, "")
}

func (p Paths) Scrollback(agent, suffix string) string {
	return mustPath(p.ScrollbackChecked(agent, suffix))
}
func (p Paths) ScrollbackPrefix() string { return p.tagged("scrollback-", "-") }
func (p Paths) ScrollbackChecked(agent, suffix string) (string, error) {
	if err := validateComponent("artifact component", agent); err != nil {
		return "", err
	}
	if err := validateComponent("artifact component", suffix); err != nil {
		return "", err
	}
	return p.tagged("scrollback-", "-"+agent+"."+suffix), nil
}

func (p Paths) ScrollbackRaw(agent string) string  { return p.Scrollback(agent, "raw") }
func (p Paths) ScrollbackANSI(agent string) string { return p.Scrollback(agent, "ansi") }
func (p Paths) ScrollbackEvents(agent string) string {
	if err := validateComponent("artifact component", agent); err != nil {
		panic(err)
	}
	return p.tagged("scrollback-", "-"+agent+".events.jsonl")
}
func (p Paths) ScrollbackViewport(agent string) string { return p.Scrollback(agent, "viewport") }

func (p Paths) Changelog(agent string) string {
	return p.ChangelogSession(agent, "") + ".md"
}

func (p Paths) ChangelogSession(agent, sessionID string) string {
	return mustPath(p.ChangelogSessionChecked(agent, sessionID))
}

func (p Paths) ChangelogSessionChecked(agent, sessionID string) (string, error) {
	if err := validateComponent("artifact component", agent); err != nil {
		return "", err
	}
	suffix := "-" + agent
	if sessionID != "" {
		if err := validateComponent("artifact component", sessionID); err != nil {
			return "", err
		}
		suffix += "-" + sessionID
	}
	return p.tagged("changelog-", suffix), nil
}

func (p Paths) AgentDraft(agent string) string {
	return mustPath(p.taggedComponent("draft-", agent, ".md"))
}
