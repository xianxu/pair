// Package artifactpath owns Pair's composite repo-scope/thread-tag artifact
// namespace. It deliberately imports only the standard library so every Pair
// command can consume the same validated paths without depending on launcher.
package artifactpath

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var componentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Address identifies one Pair work thread's artifact namespace.
// pair:m5-concept pure
type Address struct {
	DataDir   string
	RepoScope string
	Tag       string
}

// Paths is a validated composite artifact namespace.
// pair:m5-concept pure
type Paths struct {
	scopeDir string
	tag      string
}

// ScopePaths is the validated, tag-independent portion of one selected repo
// scope. Per-path launch defaults and session-name bindings live here.
// pair:m5-concept pure
type ScopePaths struct {
	scopeDir string
}

// PairCachePaths owns the compatibility restart/quit marker namespace. These
// markers are session-scoped rather than work-thread artifacts, but their
// filename vocabulary still has one constructor authority.
// pair:m5-concept pure
type PairCachePaths struct {
	dir string
}

// LifecyclePaths is one stable park-transaction namespace. Its directory and
// lock inode persist for the transaction lifetime; numbered request,
// completion, and trigger records are immutable attempt artifacts.
// pair:m5-concept pure
type LifecyclePaths struct {
	dir string
}

// LegacyRootPaths and LegacyPaths are the read/import-only authority for the
// pre-composite flat data directory. New writes use Paths; compatibility code
// must name old locations through these types instead of reopening filename
// construction throughout launcher.
// pair:m5-concept pure
type LegacyRootPaths struct {
	dataDir string
}

// pair:m5-concept pure
type LegacyPaths struct {
	root LegacyRootPaths
	tag  string
}

func ResolvePairCache(home string) (PairCachePaths, error) {
	if !filepath.IsAbs(home) {
		return PairCachePaths{}, fmt.Errorf("home directory must be absolute")
	}
	return PairCachePaths{dir: filepath.Join(filepath.Clean(home), ".cache", "pair")}, nil
}

func (p PairCachePaths) Restart(session string) (string, error) {
	if err := validateCacheSession(session); err != nil {
		return "", err
	}
	return filepath.Join(p.dir, "restart-"+session), nil
}

func (p PairCachePaths) Quit(session string) (string, error) {
	if err := validateCacheSession(session); err != nil {
		return "", err
	}
	return filepath.Join(p.dir, "quit-"+session), nil
}

// Binding is one exact artifact path exported to non-Go consumers.
// pair:m5-concept pure
type Binding struct {
	Name string
	Path string
}

// pair:m5-concept pure
type ScrollbackArtifactSet struct {
	Raw      string
	Events   string
	ANSI     string
	Viewport string
	OpenLock string
}

// pair:m5-concept pure
type ChangelogArtifactSet struct {
	Log         string
	Anchor      string
	Cleaned     string
	OpenLock    string
	DistillLock string
	Status      string
	Ready       string
}

// pair:m5-concept pure
type ParkedScrollbackArtifactSet struct {
	Base   string
	Raw    string
	Events string
}

func ResolveLegacyRoot(dataDir string) (LegacyRootPaths, error) {
	if !filepath.IsAbs(dataDir) {
		return LegacyRootPaths{}, fmt.Errorf("pair data directory must be absolute")
	}
	return LegacyRootPaths{dataDir: filepath.Clean(dataDir)}, nil
}

func ResolveLegacyFlat(dataDir, tag string) (LegacyPaths, error) {
	root, err := ResolveLegacyRoot(dataDir)
	if err != nil {
		return LegacyPaths{}, err
	}
	return root.ForTag(tag)
}

func (p LegacyRootPaths) ForTag(tag string) (LegacyPaths, error) {
	if err := validateComponent("pair tag", tag); err != nil {
		return LegacyPaths{}, err
	}
	return LegacyPaths{root: p, tag: tag}, nil
}

func (p LegacyRootPaths) SessionBindings() string {
	return filepath.Join(p.dataDir, "session-names.jsonl")
}

func (p LegacyRootPaths) HistoryGlobs() []string {
	return []string{
		filepath.Join(p.dataDir, "draft-*.md"),
		filepath.Join(p.dataDir, "log-*.md"),
		filepath.Join(p.dataDir, "ledger-*.jsonl"),
	}
}

// TagFromHistorySidecar recognizes only the three history-bearing families.
// It is shared by current-scope and legacy-flat scanners.
// pair:m5-concept pure
func TagFromHistorySidecar(name string) (string, bool) {
	for _, pattern := range []struct {
		prefix string
		suffix string
	}{
		{prefix: "draft-", suffix: ".md"},
		{prefix: "log-", suffix: ".md"},
		{prefix: "ledger-", suffix: ".jsonl"},
	} {
		if len(name) > len(pattern.prefix)+len(pattern.suffix) &&
			strings.HasPrefix(name, pattern.prefix) && strings.HasSuffix(name, pattern.suffix) {
			return strings.TrimSuffix(strings.TrimPrefix(name, pattern.prefix), pattern.suffix), true
		}
	}
	return "", false
}

// IsLedgerHistorySidecar distinguishes the ledger member of the history
// vocabulary without making scanners repeat its filename prefix.
func IsLedgerHistorySidecar(name string) bool {
	return strings.HasPrefix(name, "ledger-") && strings.HasSuffix(name, ".jsonl")
}

// IsLogHistorySidecar distinguishes the authored log member after
// TagFromHistorySidecar has validated the shared tag grammar.
func IsLogHistorySidecar(name string) bool {
	return strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md")
}

// IsConfigSidecar recognizes the config filename family before owner
// validation, so callers can distinguish an unsupported or malformed owner
// from an unrelated file.
func IsConfigSidecar(name string) bool {
	return strings.HasPrefix(name, "config-") && strings.HasSuffix(name, ".json")
}

// TagAgentFromConfigSidecar recognizes a validated config sidecar for one of
// the caller's supported agent names without duplicating the filename family.
func TagAgentFromConfigSidecar(name string, agents []string) (string, string, bool) {
	if !IsConfigSidecar(name) {
		return "", "", false
	}
	base := strings.TrimSuffix(strings.TrimPrefix(name, "config-"), ".json")
	for _, agent := range agents {
		if err := validateComponent("artifact component", agent); err != nil {
			continue
		}
		suffix := "-" + agent
		if !strings.HasSuffix(base, suffix) {
			continue
		}
		tag := strings.TrimSuffix(base, suffix)
		if validateComponent("pair tag", tag) == nil {
			return tag, agent, true
		}
	}
	return "", "", false
}

// CommandReferencesDraftArtifact recognizes the descriptive draft path token in
// an external pane command without making layout classifiers own that token.
func CommandReferencesDraftArtifact(command string) bool {
	return strings.Contains(command, "draft-")
}

func (p LegacyRootPaths) Entry(name string) (string, error) {
	if name == "" || filepath.Base(name) != name {
		return "", fmt.Errorf("invalid legacy artifact entry %q", name)
	}
	return filepath.Join(p.dataDir, name), nil
}

func (p LegacyPaths) QueueDir() string {
	return filepath.Join(p.root.dataDir, "queue-"+p.tag)
}

func (p LegacyPaths) PanePrefix() string { return "pane-" + p.tag + "-" }

func (p LegacyPaths) HistoryGlobs() []string { return p.root.HistoryGlobs() }

// RenameArtifacts enumerates the pre-composite flat sidecars imported during
// migration. The filename set stays identical to the current scoped set while
// the legacy type keeps the old root explicit at the call site.
func (p LegacyPaths) RenameArtifacts(agents []string) []string {
	return (Paths{scopeDir: p.root.dataDir, tag: p.tag}).RenameArtifacts(agents)
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

func (p ScopePaths) SessionInventoryCatalog() string {
	return filepath.Join(p.scopeDir, "session-inventory-catalog.json")
}

func (p ScopePaths) HistoryGlobs() []string {
	return []string{
		filepath.Join(p.scopeDir, "draft-*.md"),
		filepath.Join(p.scopeDir, "log-*.md"),
		filepath.Join(p.scopeDir, "ledger-*.jsonl"),
	}
}

// NvimPIDGlob returns the selected-scope enumeration pattern for one nvim
// surface. Callers may glob this pattern, but must not reconstruct its filename
// family themselves.
func (p ScopePaths) NvimPIDGlob(kind string) (string, error) {
	if err := validateComponent("artifact component", kind); err != nil {
		return "", err
	}
	return filepath.Join(p.scopeDir, "nvim-pid-*-"+kind), nil
}

// TagFromNvimPID recognizes one selected-scope nvim pid sidecar returned by
// NvimPIDGlob. It rejects paths from another scope and invalid tag components.
func (p ScopePaths) TagFromNvimPID(path, kind string) (string, bool) {
	if err := validateComponent("artifact component", kind); err != nil {
		return "", false
	}
	clean := filepath.Clean(path)
	if filepath.Dir(clean) != p.scopeDir {
		return "", false
	}
	base := filepath.Base(clean)
	prefix, suffix := "nvim-pid-", "-"+kind
	if !strings.HasPrefix(base, prefix) || !strings.HasSuffix(base, suffix) {
		return "", false
	}
	tag := strings.TrimSuffix(strings.TrimPrefix(base, prefix), suffix)
	if err := validateComponent("pair tag", tag); err != nil {
		return "", false
	}
	return tag, true
}

// TagFromNvimEmbedArgv recognizes exact draft and scrollback paths under this
// selected scope. It is the pure parser used by the process-scan fallback when
// a pid sidecar is missing.
func (p ScopePaths) TagFromNvimEmbedArgv(argv string) string {
	if marker := filepath.Join(p.scopeDir, "draft-"); strings.Contains(argv, marker) {
		rest := firstArgvField(argv[strings.LastIndex(argv, marker)+len(marker):])
		tag := strings.TrimSuffix(rest, ".md")
		if validateComponent("pair tag", tag) == nil {
			return tag
		}
		return ""
	}
	if marker := filepath.Join(p.scopeDir, "scrollback-"); strings.Contains(argv, marker) {
		rest := firstArgvField(argv[strings.LastIndex(argv, marker)+len(marker):])
		rest = strings.TrimSuffix(rest, ".ansi")
		if i := strings.LastIndex(rest, "-"); i >= 0 {
			rest = rest[:i]
		}
		if validateComponent("pair tag", rest) == nil {
			return rest
		}
	}
	return ""
}

func firstArgvField(value string) string {
	if i := strings.IndexByte(value, ' '); i >= 0 {
		return value[:i]
	}
	return value
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

// validateCacheSession preserves Zellij's Unicode session names while keeping
// compatibility marker files confined to the Pair cache directory.
func validateCacheSession(value string) error {
	if value == "" || value == "." || value == ".." || filepath.Base(value) != value || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("invalid session name %q", value)
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

// Lifecycle derives one transaction namespace after validating its stable
// nonce. All lifecycle filenames are owned by this type.
func (p Paths) Lifecycle(nonce string) (LifecyclePaths, error) {
	if err := validateComponent("lifecycle nonce", nonce); err != nil {
		return LifecyclePaths{}, err
	}
	return LifecyclePaths{dir: filepath.Join(p.tagged("lifecycle-", ""), nonce)}, nil
}

func (p LifecyclePaths) Dir() string  { return p.dir }
func (p LifecyclePaths) Lock() string { return filepath.Join(p.dir, "lifecycle.lock") }

func (p LifecyclePaths) Request(attempt uint64) (string, error) {
	name, err := lifecycleAttemptName("quit-request-", attempt)
	if err != nil {
		return "", err
	}
	return filepath.Join(p.dir, name+".json"), nil
}

func (p LifecyclePaths) Completion(attempt uint64) (string, error) {
	name, err := lifecycleAttemptName("quit-completion-", attempt)
	if err != nil {
		return "", err
	}
	return filepath.Join(p.dir, name+".json"), nil
}

// CompletionKey is the path-independent immutable key carried in both wire
// records.
func (p LifecyclePaths) CompletionKey(attempt uint64) (string, error) {
	return lifecycleAttemptName("quit-completion-", attempt)
}

func (p LifecyclePaths) Trigger(session string, attempt uint64) (string, error) {
	if err := validateCacheSession(session); err != nil {
		return "", err
	}
	name, err := lifecycleAttemptName("quit-trigger-"+session+"-", attempt)
	if err != nil {
		return "", err
	}
	return filepath.Join(p.dir, name+".json"), nil
}

func lifecycleAttemptName(prefix string, attempt uint64) (string, error) {
	if attempt == 0 {
		return "", fmt.Errorf("lifecycle attempt must be positive")
	}
	return prefix + strconv.FormatUint(attempt, 10), nil
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
func (p Paths) SessionInventoryCatalog() string {
	return filepath.Join(p.ScopeDir(), "session-inventory-catalog.json")
}
func (p Paths) ThreadClaim() string  { return p.tagged("thread-claim-", ".json") }
func (p Paths) Quote() string        { return p.tagged("quote-", "") }
func (p Paths) Slug() string         { return p.tagged("slug-", "") }
func (p Paths) SlugProposed() string { return p.tagged("slug-proposed-", "") }
func (p Paths) TitlePID() string     { return p.tagged("title-pid-", "") }
func (p Paths) PairWrapPID() string  { return p.tagged("pair-wrap-pid-", "") }
func (p Paths) WrapEvents() string   { return p.tagged("wrap-events-", ".jsonl") }
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

// RenameArtifacts is the stable, exact sidecar inventory used by tag rename
// and legacy import. Callers supply the harness names because artifactpath is a
// dependency leaf and does not own the agent inventory.
func (p Paths) RenameArtifacts(agents []string) []string {
	out := []string{
		p.OuterTTY(), p.PairWrapPID(), p.TitlePID(), p.Agent(),
		p.AgentPID(), p.AgentOutput(), p.AgentPicks(), p.LayoutMode(),
		p.WorkbenchLayout(), p.QueueDir(), p.Quote(), p.ImageCapture(),
		p.ImageCaptureDone(), p.Draft(), p.Log(), p.Ledger(),
		p.NvimPID("draft"), p.NvimPID("scrollback"),
	}
	for _, agent := range agents {
		out = append(out,
			p.Config(agent), p.Pane(agent), p.ScrollbackANSI(agent),
			p.ScrollbackRaw(agent), p.ScrollbackViewport(agent),
			p.ScrollbackEvents(agent), p.AgentDraft(agent),
		)
	}
	return out
}

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
		{Name: "PAIR_CHANGELOG_READY_PATH", Path: p.ChangelogReady(agent)},
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

// AgentFromPane recognizes one pane sidecar from this exact work thread.
func (p Paths) AgentFromPane(path string) (string, bool) {
	clean := filepath.Clean(path)
	if filepath.Dir(clean) != p.ScopeDir() {
		return "", false
	}
	base := filepath.Base(clean)
	prefix := "pane-" + p.tag + "-"
	if !strings.HasPrefix(base, prefix) || !strings.HasSuffix(base, ".json") {
		return "", false
	}
	agent := strings.TrimSuffix(strings.TrimPrefix(base, prefix), ".json")
	if err := validateComponent("artifact component", agent); err != nil {
		return "", false
	}
	return agent, true
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

func (p Paths) ParkedScrollbackArtifacts(timestamp string) (ParkedScrollbackArtifactSet, error) {
	base, err := p.ParkedScrollbackChecked(timestamp)
	if err != nil {
		return ParkedScrollbackArtifactSet{}, err
	}
	return ParkedScrollbackArtifactSet{Base: base, Raw: base + ".raw", Events: base + ".events.jsonl"}, nil
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

func (p Paths) ScrollbackArtifacts(agent string) (ScrollbackArtifactSet, error) {
	if err := validateComponent("artifact component", agent); err != nil {
		return ScrollbackArtifactSet{}, err
	}
	return ScrollbackArtifactSet{
		Raw:      p.tagged("scrollback-", "-"+agent+".raw"),
		Events:   p.tagged("scrollback-", "-"+agent+".events.jsonl"),
		ANSI:     p.tagged("scrollback-", "-"+agent+".ansi"),
		Viewport: p.tagged("scrollback-", "-"+agent+".viewport"),
		OpenLock: p.tagged("scrollback-", "-"+agent+".openlock"),
	}, nil
}

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

func (p Paths) ChangelogReady(agent string) string {
	if err := validateComponent("artifact component", agent); err != nil {
		panic(err)
	}
	return p.tagged("changelog-", "-"+agent+".ready")
}

func (p Paths) ChangelogArtifacts(agent, sessionID string) (ChangelogArtifactSet, error) {
	base, err := p.ChangelogSessionChecked(agent, sessionID)
	if err != nil {
		return ChangelogArtifactSet{}, err
	}
	return ChangelogArtifactSet{
		Log:         base + ".md",
		Anchor:      base + ".anchor",
		Cleaned:     base + ".cleaned",
		OpenLock:    base + ".openlock",
		DistillLock: base + ".distill.lock",
		Status:      base + ".status",
		Ready:       p.ChangelogReady(agent),
	}, nil
}

func (p Paths) AgentDraft(agent string) string {
	return mustPath(p.taggedComponent("draft-", agent, ".md"))
}
