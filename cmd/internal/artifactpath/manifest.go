package artifactpath

// Family is one checked artifact filename family. Token is the stable literal
// a source-coverage test uses to find constructors and exact-path consumers.
// pair:m5-concept pure
type Family struct {
	Name  string
	Token string
}

// SourceKind records why a production source may mention an artifact token.
// pair:m5-concept pure
type SourceKind string

const (
	Constructor        SourceKind = "constructor"
	ResolvedConsumer   SourceKind = "resolved-consumer"
	VocabularyConsumer SourceKind = "vocabulary-consumer"
	GeneratedMirror    SourceKind = "generated-mirror"
)

// pair:m5-concept pure
type VocabularyContext string

const (
	GoStructTagVocabulary    VocabularyContext = "go-struct-tag"
	GoCallArgumentVocabulary VocabularyContext = "go-call-argument"
	GoCaseValueVocabulary    VocabularyContext = "go-case-value"
	GoComparisonVocabulary   VocabularyContext = "go-comparison"
	ExactLineVocabulary      VocabularyContext = "exact-line"
)

// VocabularyAllowance names one exact non-path use of an artifact-family token.
// Count makes the manifest exhaustive rather than an open-ended allowlist.
// pair:m5-concept pure
type VocabularyAllowance struct {
	Family   string
	Value    string
	Context  VocabularyContext
	Use      string
	Argument int
	Count    int
}

// ResolvedBinding is one canonical positive derivation witness. Resolver is an
// exported artifactpath function call; Member is the family-specific method
// consumed from its returned value.
// pair:m5-concept pure
type ResolvedBinding struct {
	Name     string
	Family   string
	Resolver string
	Member   string
}

// pair:m5-concept pure
var ResolvedBindings = append([]ResolvedBinding{
	{Name: "cache-restart", Family: "restart", Resolver: "ResolvePairCache", Member: "Restart"},
	{Name: "direct-draft-command", Family: "draft", Resolver: "CommandReferencesDraftArtifact"},
	{Name: "direct-draft-history-tag", Family: "draft", Resolver: "TagFromHistorySidecar"},
	{Name: "direct-ledger-history", Family: "ledger", Resolver: "IsLedgerHistorySidecar"},
	{Name: "direct-ledger-history-tag", Family: "ledger", Resolver: "TagFromHistorySidecar"},
	{Name: "direct-log-history", Family: "log", Resolver: "IsLogHistorySidecar"},
	{Name: "direct-log-history-tag", Family: "log", Resolver: "TagFromHistorySidecar"},
	{Name: "direct-config-recognition", Family: "config", Resolver: "IsConfigSidecar"},
	{Name: "direct-config-sidecar", Family: "config", Resolver: "TagAgentFromConfigSidecar"},
	{Name: "legacy-pane", Family: "pane", Resolver: "ResolveLegacyRoot", Member: "PanePrefix"},
	{Name: "legacy-queue", Family: "queue", Resolver: "ResolveLegacyFlat", Member: "QueueDir"},
	{Name: "legacy-history-draft", Family: "draft", Resolver: "ResolveLegacyRoot", Member: "HistoryGlobs"},
	{Name: "legacy-history-ledger", Family: "ledger", Resolver: "ResolveLegacyRoot", Member: "HistoryGlobs"},
	{Name: "legacy-history-log", Family: "log", Resolver: "ResolveLegacyRoot", Member: "HistoryGlobs"},
	{Name: "legacy-session-binding", Family: "session-binding", Resolver: "ResolveLegacyRoot", Member: "SessionBindings"},
	{Name: "scoped-adapt", Family: "adapt", Resolver: "ResolveScoped", Member: "AdaptLog"},
	{Name: "scoped-agent", Family: "agent", Resolver: "ResolveScoped", Member: "Agent"},
	{Name: "scoped-agent-output", Family: "agent", Resolver: "ResolveScoped", Member: "AgentOutput"},
	{Name: "scoped-agent-pid", Family: "agent-pid", Resolver: "ResolveScoped", Member: "AgentPID"},
	{Name: "scoped-agent-ready", Family: "agent-ready", Resolver: "ResolveScoped", Member: "AgentReadyChecked"},
	{Name: "scoped-agent-ready-path", Family: "agent-ready", Resolver: "ResolveScoped", Member: "AgentReady"},
	{Name: "scoped-changelog", Family: "changelog", Resolver: "ResolveScoped", Member: "ChangelogSessionChecked"},
	{Name: "scoped-changelog-artifacts", Family: "changelog", Resolver: "ResolveScoped", Member: "ChangelogArtifacts"},
	{Name: "scoped-config", Family: "config", Resolver: "ResolveScoped", Member: "ConfigChecked"},
	{Name: "scoped-config-glob", Family: "config", Resolver: "ResolveScoped", Member: "ConfigGlob"},
	{Name: "scoped-draft", Family: "draft", Resolver: "ResolveScoped", Member: "Draft"},
	{Name: "scoped-image-capture", Family: "image-capture", Resolver: "ResolveScoped", Member: "ImageCapture"},
	{Name: "scoped-ledger", Family: "ledger", Resolver: "ResolveScoped", Member: "Ledger"},
	{Name: "scoped-log", Family: "log", Resolver: "ResolveScoped", Member: "Log"},
	{Name: "scoped-nvim-pid", Family: "nvim-pid", Resolver: "ResolveScoped", Member: "NvimPID"},
	{Name: "scoped-outer-tty", Family: "outer-tty", Resolver: "ResolveScoped", Member: "OuterTTY"},
	{Name: "scoped-pair-wrap-pid", Family: "pair-wrap-pid", Resolver: "ResolveScoped", Member: "PairWrapPID"},
	{Name: "scoped-pane", Family: "pane", Resolver: "ResolveScoped", Member: "PaneChecked"},
	{Name: "scoped-pane-glob", Family: "pane", Resolver: "ResolveScoped", Member: "PaneGlob"},
	{Name: "scoped-parked", Family: "parked", Resolver: "ResolveScoped", Member: "Parked"},
	{Name: "scoped-parked-scrollback", Family: "parked-scrollback", Resolver: "ResolveScoped", Member: "ParkedScrollbackArtifacts"},
	{Name: "scoped-queue", Family: "queue", Resolver: "ResolveScoped", Member: "QueueDir"},
	{Name: "scoped-quote", Family: "quote", Resolver: "ResolveScoped", Member: "Quote"},
	{Name: "scoped-layout", Family: "layout", Resolver: "ResolveScoped", Member: "WorkbenchLayout"},
	{Name: "scoped-last-left-pane", Family: "last-left-pane", Resolver: "ResolveScoped", Member: "LastLeftPane"},
	{Name: "scoped-last-terminal-pane", Family: "last-terminal-pane", Resolver: "ResolveScoped", Member: "LastTerminalPane"},
	{Name: "scoped-picker", Family: "picker", Resolver: "ResolveScoped", Member: "DraftPane"},
	{Name: "scoped-review", Family: "review", Resolver: "ResolveScoped", Member: "ReviewTarget"},
	{Name: "scoped-scrollback", Family: "scrollback", Resolver: "ResolveScoped", Member: "ScrollbackArtifacts"},
	{Name: "scoped-scrollback-pending", Family: "scrollback-pending", Resolver: "ResolveScoped", Member: "ScrollbackPending"},
	{Name: "scoped-slug", Family: "slug", Resolver: "ResolveScoped", Member: "Slug"},
	{Name: "scoped-title-pid", Family: "title-pid", Resolver: "ResolveScoped", Member: "TitlePID"},
	{Name: "scoped-terminal-panes", Family: "terminal-panes", Resolver: "ResolveScoped", Member: "TerminalPanes"},
	{Name: "scoped-wrap-events", Family: "wrap-events", Resolver: "ResolveScoped", Member: "WrapEvents"},
	{Name: "composite-scope-session-binding", Family: "session-binding", Resolver: "Resolve", Member: "ScopeDir"},
	{Name: "composite-lifecycle-request", Family: "lifecycle-request", Resolver: "Resolve", Member: "Lifecycle"},
	{Name: "composite-lifecycle-completion", Family: "lifecycle-completion", Resolver: "Resolve", Member: "Lifecycle"},
	{Name: "composite-lifecycle-native-session", Family: "native-session", Resolver: "Resolve", Member: "Lifecycle"},
	{Name: "selected-nvim-pid", Family: "nvim-pid", Resolver: "ResolveSelectedScope", Member: "NvimPIDGlob"},
	{Name: "selected-agent-default", Family: "agent-default", Resolver: "ResolveSelectedScope", Member: "AgentDefault"},
	{Name: "selected-history-draft", Family: "draft", Resolver: "ResolveSelectedScope", Member: "HistoryGlobs"},
	{Name: "selected-history-ledger", Family: "ledger", Resolver: "ResolveSelectedScope", Member: "HistoryGlobs"},
	{Name: "selected-history-log", Family: "log", Resolver: "ResolveSelectedScope", Member: "HistoryGlobs"},
	{Name: "selected-session-binding", Family: "session-binding", Resolver: "ResolveSelectedScope", Member: "SessionBindings"},
	{Name: "selected-session-inventory-catalog", Family: "session-inventory-catalog", Resolver: "ResolveSelectedScope", Member: "SessionInventoryCatalog"},
	{Name: "scoped-session-inventory-catalog", Family: "session-inventory-catalog", Resolver: "ResolveScoped", Member: "SessionInventoryCatalog"},
}, generatedResolvedBindings()...)

func generatedResolvedBindings() []ResolvedBinding {
	var out []ResolvedBinding
	addFamilies := func(prefix, resolver, member string, families []string) {
		for _, family := range families {
			out = append(out, ResolvedBinding{
				Name: prefix + "-" + family, Family: family, Resolver: resolver, Member: member,
			})
		}
	}
	environmentFamilies := []string{
		"adapt", "agent", "agent-pid", "agent-ready", "config", "draft",
		"image-capture", "ledger", "log", "nvim-pid", "outer-tty", "pair-wrap-pid",
		"pane", "queue", "quote", "scrollback", "slug",
	}
	renameFamilies := []string{
		"agent", "agent-pid", "config", "draft", "image-capture", "layout",
		"layout-mode", "ledger", "log", "nvim-pid", "outer-tty", "pair-wrap-pid",
		"pane", "queue", "quote", "scrollback", "title-pid",
	}
	addFamilies("environment", "ResolveScoped", "EnvironmentBindings", environmentFamilies)
	addFamilies("scoped-rename", "ResolveScoped", "RenameArtifacts", renameFamilies)
	addFamilies("legacy-rename", "ResolveLegacyFlat", "RenameArtifacts", renameFamilies)
	addFamilies("scoped-wrapper", "ResolveScoped", "", []string{"last-left-pane", "last-terminal-pane", "terminal-panes"})
	addFamilies("composite", "Resolve", "", []string{
		"adapt", "agent", "agent-pid", "agent-ready", "changelog", "config",
		"draft", "ledger", "log", "nvim-pid", "outer-tty", "pane", "queue",
		"scrollback", "session-binding", "thread-claim",
	})
	return out
}

// SourceClassification is deliberately exact: Path names one repository file
// and Families names every artifact family it constructs or consumes. New
// files cannot disappear behind directory or wildcard exemptions.
// pair:m5-concept pure
type SourceClassification struct {
	Path         string
	Kind         SourceKind
	Families     []string
	BindingNames []string
	Vocabulary   []VocabularyAllowance
}

// Families is the exhaustive tag-bearing artifact vocabulary. Adding a new
// family requires adding its constructor and classifying every source consumer.
// pair:m5-concept pure
var Families = []Family{
	{Name: "agent-default", Token: "agent-default-"},
	{Name: "draft", Token: "draft-"},
	{Name: "ledger", Token: "ledger-"},
	{Name: "log", Token: "log-"},
	{Name: "queue", Token: "queue-"},
	{Name: "config", Token: "config-"},
	{Name: "native-session", Token: "session_id"},
	{Name: "pane", Token: "pane-"},
	{Name: "agent", Token: "agent-"},
	{Name: "agent-ready", Token: "agent-ready-"},
	{Name: "agent-pid", Token: "agent-pid-"},
	{Name: "outer-tty", Token: "outer-tty-"},
	{Name: "nvim-pid", Token: "nvim-pid-"},
	{Name: "parked", Token: "parked-"},
	{Name: "parked-scrollback", Token: "parked-scrollback-"},
	{Name: "adapt", Token: "adapt-"},
	{Name: "image-capture", Token: "image-capture-"},
	{Name: "continuation", Token: "continuation-"},
	{Name: "layout", Token: "workbench-layout-"},
	{Name: "layout-mode", Token: "layout-mode-"},
	{Name: "restart", Token: "restart-"},
	{Name: "picker", Token: "draft-pane-"},
	{Name: "session-binding", Token: "session-names.jsonl"},
	{Name: "session-inventory-catalog", Token: "session-inventory-catalog.json"},
	{Name: "scrollback", Token: "scrollback-"},
	{Name: "changelog", Token: "changelog-"},
	{Name: "thread-claim", Token: "thread-claim-"},
	{Name: "quote", Token: "quote-"},
	{Name: "slug", Token: "slug-"},
	{Name: "title-pid", Token: "title-pid-"},
	{Name: "pair-wrap-pid", Token: "pair-wrap-pid-"},
	{Name: "wrap-events", Token: "wrap-events-"},
	{Name: "scrollback-pending", Token: "scrollback-pending-"},
	{Name: "last-left-pane", Token: "last-left-pane-"},
	{Name: "last-terminal-pane", Token: "last-terminal-pane-"},
	{Name: "terminal-panes", Token: "terminal-panes-"},
	{Name: "zellij-actions", Token: "zellij-actions-"},
	{Name: "review", Token: "review-"},
	{Name: "codex-filter-kkp", Token: "codex-filter-kkp-"},
	{Name: "lifecycle", Token: "lifecycle-"},
	{Name: "lifecycle-lock", Token: "lifecycle.lock"},
	{Name: "lifecycle-request", Token: "quit-request-"},
	{Name: "lifecycle-completion", Token: "quit-completion-"},
	{Name: "lifecycle-trigger", Token: "quit-trigger-"},
}

// SourceClassifications is checked by coverage_test.go against production Go,
// shell, Lua, and KDL sources.
// pair:m5-concept pure
var SourceClassifications = []SourceClassification{
	{Path: "cmd/internal/artifactpath/paths.go", Kind: Constructor, Families: []string{
		"adapt", "agent", "agent-default", "agent-pid", "agent-ready", "changelog", "config",
		"continuation", "draft", "image-capture", "layout", "layout-mode",
		"last-left-pane", "last-terminal-pane", "ledger", "log", "nvim-pid",
		"outer-tty", "pair-wrap-pid", "pane", "parked", "parked-scrollback",
		"picker", "queue", "quote", "restart", "review", "scrollback",
		"scrollback-pending", "session-binding", "session-inventory-catalog", "slug", "terminal-panes",
		"thread-claim", "title-pid", "wrap-events", "zellij-actions",
		"codex-filter-kkp", "lifecycle", "lifecycle-lock", "lifecycle-request",
		"lifecycle-completion", "lifecycle-trigger",
	}},
	{Path: "cmd/internal/artifactpath/manifest.go", Kind: Constructor, Families: familyNames()},
	{Path: "cmd/internal/adapt/adapt.go", Kind: ResolvedConsumer, Families: []string{"adapt"}, BindingNames: []string{"scoped-adapt"}},
	{Path: "cmd/internal/agentcmd/restart.go", Kind: ResolvedConsumer, Families: []string{"pair-wrap-pid"}, BindingNames: []string{"scoped-pair-wrap-pid"}},
	{Path: "cmd/internal/clipcmd/clipcmd.go", Kind: ResolvedConsumer, Families: []string{"quote"}, BindingNames: []string{"scoped-quote"}},
	{Path: "cmd/internal/sessioninventory/scan_claude.go", Kind: VocabularyConsumer, Families: []string{"agent"}, Vocabulary: []VocabularyAllowance{
		goComparisonVocabulary("agent", "agent-", "claudePathFact", 1),
	}},
	{Path: "cmd/internal/sessioninventory/event.go", Kind: VocabularyConsumer, Families: []string{"queue"}, Vocabulary: []VocabularyAllowance{
		goComparisonVocabulary("queue", "queue-operation", "normalizeClaudeEvent", 1),
		goCallVocabulary("queue", "queue-operation.enqueue", "function.nativeTextEvent", 2, 1),
	}},
	{Path: "cmd/internal/continuationcmd/continuationcmd.go", Kind: ResolvedConsumer, Families: []string{"draft"}, BindingNames: []string{"scoped-draft"}},
	{Path: "cmd/internal/draftroute/route.go", Kind: ResolvedConsumer, Families: []string{"picker"}, BindingNames: []string{"scoped-picker"}},
	{Path: "cmd/internal/launcher/agent_defaults.go", Kind: ResolvedConsumer, Families: []string{"agent-default"}, BindingNames: []string{"selected-agent-default"}},
	{Path: "cmd/internal/launcher/config.go", Kind: ResolvedConsumer, Families: []string{"config"}, BindingNames: []string{"scoped-config"}},
	{Path: "cmd/internal/launcher/createflow.go", Kind: ResolvedConsumer,
		Families: []string{
			"adapt", "agent", "agent-pid", "agent-ready", "config", "draft",
			"image-capture", "ledger", "log", "nvim-pid", "outer-tty", "pair-wrap-pid",
			"pane", "queue", "quote", "scrollback", "slug",
		},
		BindingNames: []string{
			"environment-adapt", "environment-agent", "environment-agent-pid", "environment-agent-ready",
			"environment-config", "environment-draft", "environment-image-capture", "environment-ledger",
			"environment-log", "environment-nvim-pid", "environment-outer-tty", "environment-pair-wrap-pid",
			"environment-pane", "environment-queue", "environment-quote", "environment-scrollback", "environment-slug",
		},
		Vocabulary: goCallVocabularyFamilies(continuationBootstrapPrompt, "fmt.Sprintf", 0,
			"draft", "log", "parked", "parked-scrollback", "queue", "scrollback"),
	},
	{Path: "cmd/internal/launcher/history.go", Kind: ResolvedConsumer,
		Families: []string{"draft", "ledger", "log", "queue"},
		BindingNames: []string{
			"direct-draft-history-tag", "direct-ledger-history", "direct-ledger-history-tag", "direct-log-history-tag",
			"legacy-history-draft", "legacy-history-ledger", "legacy-history-log", "legacy-queue",
			"scoped-ledger", "scoped-queue", "selected-history-draft", "selected-history-ledger", "selected-history-log",
		}},
	{Path: "cmd/internal/launcher/layoutflow.go", Kind: ResolvedConsumer,
		Families: []string{"draft", "layout"}, BindingNames: []string{"direct-draft-command", "scoped-layout"}},
	{Path: "cmd/internal/launcher/readiness.go", Kind: ResolvedConsumer, Families: []string{"agent-ready"}, BindingNames: []string{"scoped-agent-ready-path"}},
	{Path: "cmd/internal/launcher/rename.go", Kind: ResolvedConsumer,
		Families: []string{
			"agent", "agent-pid", "config", "draft", "image-capture", "layout", "layout-mode", "ledger", "log",
			"nvim-pid", "outer-tty", "pair-wrap-pid", "pane", "queue", "quote", "scrollback", "title-pid",
		},
		BindingNames: []string{
			"scoped-rename-agent", "scoped-rename-agent-pid", "scoped-rename-config", "scoped-rename-draft",
			"scoped-rename-image-capture", "scoped-rename-layout", "scoped-rename-layout-mode", "scoped-rename-ledger",
			"scoped-rename-log", "scoped-rename-nvim-pid", "scoped-rename-outer-tty", "scoped-rename-pair-wrap-pid",
			"scoped-rename-pane", "scoped-rename-queue", "scoped-rename-quote", "scoped-rename-scrollback",
			"scoped-rename-title-pid",
		}},
	{Path: "cmd/internal/launcher/scoped_paths.go", Kind: ResolvedConsumer,
		Families: []string{
			"adapt", "agent", "agent-pid", "agent-ready", "changelog", "config", "draft", "ledger", "log",
			"nvim-pid", "outer-tty", "pane", "queue", "scrollback", "session-binding", "thread-claim",
		},
		BindingNames: []string{
			"composite-adapt", "composite-agent", "composite-agent-pid", "composite-agent-ready", "composite-changelog",
			"composite-config", "composite-draft", "composite-ledger", "composite-log", "composite-nvim-pid",
			"composite-outer-tty", "composite-pane", "composite-queue", "composite-scrollback",
			"composite-session-binding", "composite-thread-claim",
		}},
	{Path: "cmd/internal/launcher/session_index.go", Kind: ResolvedConsumer,
		Families: []string{"session-binding"}, BindingNames: []string{"legacy-session-binding", "selected-session-binding"}},
	{Path: "cmd/internal/opener/run.go", Kind: ResolvedConsumer,
		Families:     []string{"changelog", "nvim-pid", "scrollback"},
		BindingNames: []string{"scoped-changelog-artifacts", "scoped-nvim-pid", "scoped-scrollback"},
		Vocabulary: append(
			goCallVocabularyValues("scrollback", "fmt.Fprintf", 1,
				"pair-scrollback-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n",
				"pair-scrollback-open: resolve artifact namespace: %v\n",
				"pair-scrollback-open: resolve scrollback artifact: %v\n",
				"pair-scrollback-open: no scrollback yet for %s/%s\n",
				"pair-scrollback-open: scrollback-render failed: %v\n"),
			goCallVocabularyValues("changelog", "fmt.Fprintf", 1,
				"pair-changelog-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n",
				"pair-changelog-open: resolve artifact namespace: %v\n",
				"pair-changelog-open: resolve changelog artifact: %v\n",
				"pair-changelog-open: resolve scrollback artifact: %v\n")...),
	},
	{Path: "cmd/internal/reviewcmd/run.go", Kind: ResolvedConsumer,
		Families: []string{"nvim-pid", "review"}, BindingNames: []string{"scoped-nvim-pid", "scoped-review"},
		Vocabulary: goCallVocabularyValues("review", "fmt.Fprintf", 1,
			"pair-review-target: status must be proposed|ready\n",
			"pair-review-target: PAIR_DATA_DIR not set\n",
			"pair-review-target: resolve artifact namespace: %v\n",
			"pair-review-definition: PAIR_DATA_DIR not set\n",
			"pair-review-definition: request id is required\n",
			"pair-review-definition: definition is required\n",
			"pair-review-definition: resolve artifact namespace: %v\n",
			"pair-review-definition: write %s: %v\n",
			"pair-review-open: needs a file argument\n",
			"pair-review-open: %s not found\n",
			"pair-review-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_HOME\n",
			"pair-review-open: resolve artifact namespace: %v\n",
			"pair-review-open: %v\n",
			"usage: pair-review-readiness [--prepare] <file>\n",
			"pair-review-readiness: classify failed (nvim/readiness.lua)\n"),
	},
	{Path: "cmd/internal/titlepoller/run.go", Kind: ResolvedConsumer,
		Families: []string{"draft", "title-pid"}, BindingNames: []string{"scoped-draft", "scoped-title-pid"}},
	{Path: "cmd/internal/workbenchshortcut/shortcut.go", Kind: ResolvedConsumer,
		Families:     []string{"last-left-pane", "last-terminal-pane", "terminal-panes"},
		BindingNames: []string{"scoped-wrapper-last-left-pane", "scoped-wrapper-last-terminal-pane", "scoped-wrapper-terminal-panes"}},
	{Path: "cmd/internal/continuationcmd/continuation.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("native-session", "session_id: %s\n", "fmt.Fprintf", 1, 1),
	}},
	{Path: "cmd/internal/launcher/compaction.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("native-session", "session_id=%s\n", "fmt.Fprintf", 1, 1),
	}},
	{Path: "cmd/internal/launcher/createlogic.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
	}},
	{Path: "cmd/internal/launcher/ledger.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
	}},
	{Path: "cmd/internal/launcher/markers.go", Kind: ResolvedConsumer, Families: []string{"lifecycle-request", "native-session"}, BindingNames: []string{"composite-lifecycle-request", "composite-lifecycle-native-session"}, Vocabulary: []VocabularyAllowance{
		goCaseVocabulary("native-session", "session_id", "parseRestartMarker", 1),
	}},
	{Path: "cmd/internal/couchcore/artifactcollision.go", Kind: ResolvedConsumer,
		Families: []string{"session-binding"}, BindingNames: []string{"composite-scope-session-binding"}},
	{Path: "cmd/internal/couchcore/park.go", Kind: ResolvedConsumer,
		Families: []string{"lifecycle-completion", "lifecycle-request"}, BindingNames: []string{"composite-lifecycle-completion", "composite-lifecycle-request"}},
	{Path: "cmd/internal/opener/opener.go", Kind: ResolvedConsumer, Families: []string{"changelog"}, BindingNames: []string{"scoped-changelog"}},
	{Path: "cmd/internal/sessionwatch/sessionwatch.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
	}},
	{Path: "cmd/internal/sessionledger/record.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
	}},
	{Path: "cmd/internal/titlepoller/runtime.go", Kind: ResolvedConsumer, Families: []string{"pane"}, BindingNames: []string{"scoped-pane-glob"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("pane", "--pane-id", "os/exec.Command", 5, 1),
	}},
	{Path: "cmd/internal/launcher/legacy_live.go", Kind: ResolvedConsumer, Families: []string{"pane"}, BindingNames: []string{"legacy-pane"}},
	{Path: "cmd/internal/launcher/lifecycle.go", Kind: ResolvedConsumer,
		Families:     []string{"adapt", "agent", "draft", "image-capture", "nvim-pid", "outer-tty", "pair-wrap-pid", "pane", "scrollback", "title-pid"},
		BindingNames: []string{"scoped-adapt", "scoped-agent", "scoped-draft", "scoped-image-capture", "scoped-nvim-pid", "scoped-outer-tty", "scoped-pair-wrap-pid", "scoped-pane", "scoped-scrollback", "scoped-title-pid"}},
	{Path: "cmd/internal/launcher/migrate.go", Kind: ResolvedConsumer,
		Families: []string{
			"agent", "agent-pid", "config", "draft", "image-capture", "layout", "layout-mode", "ledger", "log",
			"nvim-pid", "outer-tty", "pair-wrap-pid", "pane", "queue", "quote", "scrollback", "title-pid",
		},
		BindingNames: []string{
			"legacy-rename-agent", "legacy-rename-agent-pid", "legacy-rename-config", "legacy-rename-draft",
			"legacy-rename-image-capture", "legacy-rename-layout", "legacy-rename-layout-mode", "legacy-rename-ledger",
			"legacy-rename-log", "legacy-rename-nvim-pid", "legacy-rename-outer-tty", "legacy-rename-pair-wrap-pid",
			"legacy-rename-pane", "legacy-rename-queue", "legacy-rename-quote", "legacy-rename-scrollback",
			"legacy-rename-title-pid", "scoped-queue",
		}},
	{Path: "cmd/internal/launcher/osruntime.go", Kind: ResolvedConsumer,
		Families:     []string{"agent", "draft", "ledger", "nvim-pid", "outer-tty", "parked", "parked-scrollback", "restart", "scrollback", "session-binding", "session-inventory-catalog", "title-pid"},
		BindingNames: []string{"cache-restart", "scoped-agent", "scoped-draft", "scoped-ledger", "scoped-nvim-pid", "scoped-outer-tty", "scoped-parked", "scoped-parked-scrollback", "scoped-scrollback", "selected-nvim-pid", "selected-session-binding", "selected-session-inventory-catalog", "scoped-title-pid"},
		Vocabulary:   []VocabularyAllowance{goCallVocabulary("config", "--config-dir", "os/exec.Command", 1, 2)}},
	{Path: "cmd/internal/sessioninventory/query.go", Kind: ResolvedConsumer,
		Families: []string{"ledger"}, BindingNames: []string{"direct-ledger-history", "direct-ledger-history-tag"}},
	{Path: "cmd/internal/sessioninventory/runtime_os.go", Kind: ResolvedConsumer,
		Families: []string{"session-inventory-catalog"}, BindingNames: []string{"selected-session-inventory-catalog"}},
	{Path: "cmd/internal/opener/runtime.go", Kind: VocabularyConsumer, Families: []string{"scrollback"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("scrollback", "scrollback-render exit %d", "fmt.Sprintf", 0, 1),
	}},
	{Path: "cmd/internal/scrollbackcmd/scrollbackcmd.go", Kind: VocabularyConsumer, Families: []string{"scrollback"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("scrollback", "pair-scrollback-render", "flag.NewFlagSet", 0, 1),
		goCallVocabulary("scrollback", "usage: pair-scrollback-render [--plain] [--viewport F] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n", "fmt.Fprintf", 1, 1),
		goCallVocabulary("scrollback", "scrollback-render: %v\n", "fmt.Fprintf", 1, 1),
	}},
	{Path: "cmd/internal/sessionwatch/run.go", Kind: ResolvedConsumer,
		Families: []string{"agent-pid", "config", "ledger", "log", "session-inventory-catalog"}, BindingNames: []string{"scoped-agent-pid", "scoped-config", "scoped-ledger", "scoped-log", "scoped-session-inventory-catalog"},
		Vocabulary: []VocabularyAllowance{
			goCallVocabulary("native-session", "session_id=", "method.Log", 1, 1),
		}},
	{Path: "cmd/internal/sessionwatch/lifecycle.go", Kind: ResolvedConsumer,
		Families: []string{"ledger", "log"}, BindingNames: []string{"scoped-ledger", "scoped-log"}},
	{Path: "cmd/internal/sessionwatch/proof_migration_os.go", Kind: ResolvedConsumer,
		Families: []string{"ledger"}, BindingNames: []string{"selected-history-ledger"}},
	{Path: "cmd/internal/sessioninventory/pair_inventory.go", Kind: ResolvedConsumer,
		Families: []string{"config", "ledger", "log"}, BindingNames: []string{"direct-config-recognition", "direct-config-sidecar", "direct-ledger-history", "direct-ledger-history-tag", "direct-log-history", "direct-log-history-tag"},
		Vocabulary: []VocabularyAllowance{goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1)}},
	{Path: "cmd/internal/slugcmd/slugcmd.go", Kind: ResolvedConsumer,
		Families: []string{"slug"}, BindingNames: []string{"scoped-slug"},
		Vocabulary: []VocabularyAllowance{
			goCallVocabulary("slug", "slug-parse", "method.Log", 1, 3),
		}},
	{Path: "cmd/internal/wrapcmd/wrap.go", Kind: ResolvedConsumer,
		Families:     []string{"agent", "agent-pid", "agent-ready", "config", "image-capture", "outer-tty", "pair-wrap-pid", "wrap-events"},
		BindingNames: []string{"scoped-agent-output", "scoped-agent-pid", "scoped-agent-ready", "scoped-config", "scoped-image-capture", "scoped-outer-tty", "scoped-pair-wrap-pid", "scoped-wrap-events"},
		Vocabulary: []VocabularyAllowance{
			goCallVocabulary("scrollback", "--scrollback-log", "builtin.append", 1, 1),
			goComparisonVocabulary("scrollback", "--scrollback-log", "run", 1),
			goCallVocabulary("scrollback", "usage: pair-wrap [--scrollback-log <path>] <command> [args...]", "errors.New", 0, 1),
			goCallVocabulary("agent", "agent-restart-request", "method.traceWrap", 0, 1),
			goCallVocabulary("scrollback", "scrollback-write", "method.traceWrap", 0, 2),
		}},
	{Path: "nvim/review/record.lua", Kind: VocabularyConsumer, Families: []string{"review"}, Vocabulary: []VocabularyAllowance{
		exactLineVocabulary("review", "local OPEN = '```review-records'", 1),
	}},

	{Path: "cmd/internal/runtimebundle/assets/runtime/files/bin/lib/adapt-log.sh", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/bin/lib/dev-rebuild.sh", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/bin/pair-help", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/bin/pair-notify", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/doctor/README.md", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/doctor/SKILL.md", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/doctor/doctor.sh", Kind: GeneratedMirror, Families: []string{"adapt"}},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/doctor/emitter-health.sh", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/adapt.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/annotate.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/changelog.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/confirm_quit.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/doctor.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/draft_send.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/marker_codec.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/normalization.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/pairlog.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/pair_poke.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/submission.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/apply.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/define.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/definition_seam.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/docflow.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/gate.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/handoff.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/init.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/markers.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/menu.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/mode.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/poke_bodies.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/projection.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/readiness.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/reconcile.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/reconstruct.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/record.lua", Kind: GeneratedMirror, Families: []string{"review"}},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/resolve.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/seam.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/spinner.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/review/wrap.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/scrollback.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/slug.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/workbench_actions.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/workbench_route.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/zellij_trace.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/zellij/config.kdl", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-2.kdl", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-3.kdl", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/manifest.json", Kind: GeneratedMirror},
}

// NonArtifactSources completes the production-source inventory. Entries here
// contain no Pair artifact constructor or filename vocabulary; moving one into
// the artifact namespace requires an explicit SourceClassification.
// pair:m5-concept pure
var NonArtifactSources = []string{
	"bin/lib/adapt-log.sh",
	"bin/lib/dev-rebuild.sh",
	"bin/pair-dev",
	"bin/pair-help",
	"bin/pair-notify",
	"cmd/couch/main.go",
	"cmd/internal/ansi/ansi.go",
	"cmd/internal/changelogcmd/changelogcmd.go",
	"cmd/internal/changelogcmd/distill.go",
	"cmd/internal/changelogcmd/prompt.go",
	"cmd/internal/clipcmd/run.go",
	"cmd/internal/clipcmd/runcli.go",
	"cmd/internal/clipcmd/runtime.go",
	"cmd/internal/commitoutcome/outcome.go",
	"cmd/internal/continuationcmd/draft.go",
	"cmd/internal/contextcmd/contextcmd.go",
	"cmd/internal/continuationcmd/git.go",
	"cmd/internal/couchcmd/cli.go",
	"cmd/internal/couchcmd/errors.go",
	"cmd/internal/couchcmd/run.go",
	"cmd/internal/couchcore/actionableinventory.go",
	"cmd/internal/couchcore/actor.go",
	"cmd/internal/couchcore/actorid.go",
	"cmd/internal/couchcore/admission.go",
	"cmd/internal/couchcore/artifactcollision_fake.go",
	"cmd/internal/couchcore/clock.go",
	"cmd/internal/couchcore/couch.go",
	"cmd/internal/couchcore/git.go",
	"cmd/internal/couchcore/git_fake.go",
	"cmd/internal/couchcore/launchhelper.go",
	"cmd/internal/couchcore/launch_existing.go",
	"cmd/internal/couchcore/launchprofile.go",
	"cmd/internal/couchcore/mailbox.go",
	"cmd/internal/couchcore/migration.go",
	"cmd/internal/couchcore/namespace.go",
	"cmd/internal/couchcore/naming.go",
	"cmd/internal/couchcore/operationdispatch.go",
	"cmd/internal/couchcore/ops.go",
	"cmd/internal/couchcore/parktransaction.go",
	"cmd/internal/couchcore/parkworker.go",
	"cmd/internal/couchcore/path.go",
	"cmd/internal/couchcore/pathops.go",
	"cmd/internal/couchcore/policyresolver.go",
	"cmd/internal/couchcore/policyresolver_exec.go",
	"cmd/internal/couchcore/policyresolver_fake.go",
	"cmd/internal/couchcore/procops.go",
	"cmd/internal/couchcore/ptyrunner.go",
	"cmd/internal/couchcore/registry.go",
	"cmd/internal/couchcore/resume.go",
	"cmd/internal/couchcore/runner.go",
	"cmd/internal/couchcore/runner_fake.go",
	"cmd/internal/couchcore/startargs.go",
	"cmd/internal/couchcore/startgrant.go",
	"cmd/internal/couchcore/startresolution.go",
	"cmd/internal/couchcore/starttransaction.go",
	"cmd/internal/couchcore/store.go",
	"cmd/internal/couchcore/storejournal.go",
	"cmd/internal/couchcore/storelock.go",
	"cmd/internal/couchcore/storelock_unix.go",
	"cmd/internal/couchcore/strings.go",
	"cmd/internal/couchcore/supervisorlease.go",
	"cmd/internal/couchcore/supervisorlease_unix.go",
	"cmd/internal/couchcore/thread.go",
	"cmd/internal/couchcore/threadinventory.go",
	"cmd/internal/couchcore/threadmetadata.go",
	"cmd/internal/couchcore/threadmetadata_model.go",
	"cmd/internal/couchcore/threadstore.go",
	"cmd/internal/couchcore/threadtag.go",
	"cmd/internal/couchcore/worktree.go",
	"cmd/internal/couchtty/attention.go",
	"cmd/internal/couchtty/console.go",
	"cmd/internal/couchtty/console_completion.go",
	"cmd/internal/couchtty/console_menu.go",
	"cmd/internal/couchtty/focus.go",
	"cmd/internal/couchtty/keys.go",
	"cmd/internal/couchtty/menu.go",
	"cmd/internal/couchtty/menu_async.go",
	"cmd/internal/couchtty/menu_completion.go",
	"cmd/internal/couchtty/menu_render.go",
	"cmd/internal/couchtty/menu_refresh.go",
	"cmd/internal/couchtty/notice.go",
	"cmd/internal/couchtty/operation_queue.go",
	"cmd/internal/couchtty/panelkeys.go",
	"cmd/internal/couchtty/reserve.go",
	"cmd/internal/ctxmeter/ctxmeter.go",
	"cmd/internal/dispatcher/dispatcher.go",
	"cmd/internal/entrypoint/alias.go",
	"cmd/internal/entrypoint/asset_root.go",
	"cmd/internal/entrypoint/mode.go",
	"cmd/internal/hostty/control.go",
	"cmd/internal/hostty/fake.go",
	"cmd/internal/hostty/host.go",
	"cmd/internal/hostty/os.go",
	"cmd/internal/keyhelp/catalog.go",
	"cmd/internal/keyhelp/keyhelp.go",
	"cmd/internal/keyhelp/parse.go",
	"cmd/internal/keyhelp/render.go",
	"cmd/internal/notifycmd/run.go",
	"cmd/internal/notifyosc/notification.go",
	"cmd/internal/keyhelp/sections.go",
	"cmd/internal/keyhelp/sources.go",
	"cmd/internal/keyscmd/keyscmd.go",
	"cmd/internal/launcher/agentargs.go",
	"cmd/internal/launcher/args.go",
	"cmd/internal/launcher/continuation.go",
	"cmd/internal/launcher/datadir.go",
	"cmd/internal/launcher/decision.go",
	"cmd/internal/launcher/format.go",
	"cmd/internal/launcher/help.go",
	"cmd/internal/launcher/launch_args_policy.go",
	"cmd/internal/launcher/layout.go",
	"cmd/internal/launcher/lifecycle_os.go",
	"cmd/internal/launcher/list.go",
	"cmd/internal/launcher/pathenv.go",
	"cmd/internal/launcher/pick.go",
	"cmd/internal/launcher/restart.go",
	"cmd/internal/launcher/run.go",
	"cmd/internal/launcher/runcli.go",
	"cmd/internal/launcher/runtime.go",
	"cmd/internal/launcher/scope.go",
	"cmd/internal/launcher/session.go",
	"cmd/internal/launcher/session_quiescence.go",
	"cmd/internal/launcher/tag.go",
	"cmd/internal/launcher/thread_claim.go",
	"cmd/internal/launcher/zellij.go",
	"cmd/internal/launcher/zellijparse.go",
	"cmd/internal/layoutcmd/layoutcmd.go",
	"cmd/internal/layoutcmd/resizeplan.go",
	"cmd/internal/model/model.go",
	"cmd/internal/opener/runcli.go",
	"cmd/internal/pairlog/runcli.go",
	"cmd/internal/pairlog/store.go",
	"cmd/internal/pairlifecycle/model.go",
	"cmd/internal/pairlifecycle/cleanup.go",
	"cmd/internal/pairlifecycle/store.go",
	"cmd/internal/pairlifecycle/store_unix.go",
	"cmd/internal/pairlifecycletest/fake.go",
	"cmd/internal/pairlifecycletest/live_zellij.go",
	"cmd/internal/osfs/osfs.go",
	"cmd/internal/procutil/identity_darwin.go",
	"cmd/internal/procutil/identity_linux.go",
	"cmd/internal/procutil/identity_other.go",
	"cmd/internal/procutil/procutil.go",
	"cmd/internal/ptychild/child.go",
	"cmd/internal/ptychild/fake.go",
	"cmd/internal/ptychild/replay.go",
	"cmd/internal/ptychild/ring.go",
	"cmd/internal/ptychild/screen.go",
	"cmd/internal/readiness/record.go",
	"cmd/internal/reviewcmd/reviewcmd.go",
	"cmd/internal/reviewcmd/runcli.go",
	"cmd/internal/reviewcmd/runtime.go",
	"cmd/internal/runtimebundle/cleanup.go",
	"cmd/internal/runtimebundle/embed.go",
	"cmd/internal/runtimebundle/generatecmd/main.go",
	"cmd/internal/runtimebundle/manifest.go",
	"cmd/internal/runtimebundle/manifestmodel/manifest.go",
	"cmd/internal/runtimebundle/plan.go",
	"cmd/internal/runtimebundle/store.go",
	"cmd/internal/runtimebundlegen/generate.go",
	"cmd/internal/scribecmd/scribecmd.go",
	"cmd/internal/sessioninventory/conformance.go",
	"cmd/internal/sessioninventory/activity.go",
	"cmd/internal/sessioninventory/activitycli.go",
	"cmd/internal/sessioninventory/events.go",
	"cmd/internal/sessioninventory/binding.go",
	"cmd/internal/sessioninventory/catalog.go",
	"cmd/internal/sessioninventory/catalog_store.go",
	"cmd/internal/sessioninventory/catalog_store_unix.go",
	"cmd/internal/sessioninventory/catalog_publication.go",
	"cmd/internal/sessioninventory/diagnostic.go",
	"cmd/internal/sessioninventory/pairfacts.go",
	"cmd/internal/sessioninventory/proof_migration.go",
	"cmd/internal/sessioninventory/forest_projection.go",
	"cmd/internal/sessioninventory/filemeta.go",
	"cmd/internal/sessioninventory/filemeta_darwin.go",
	"cmd/internal/sessioninventory/filemeta_linux.go",
	"cmd/internal/sessioninventory/filemeta_other.go",
	"cmd/internal/sessioninventory/inventory.go",
	"cmd/internal/sessioninventory/incremental_inventory.go",
	"cmd/internal/sessioninventory/jsonl_incremental.go",
	"cmd/internal/sessioninventory/model.go",
	"cmd/internal/sessioninventory/offline.go",
	"cmd/internal/sessioninventory/order.go",
	"cmd/internal/sessioninventory/provider_contract.go",
	"cmd/internal/sessioninventory/render.go",
	"cmd/internal/sessioninventory/reconcile.go",
	"cmd/internal/sessioninventory/round.go",
	"cmd/internal/sessioninventory/runcli.go",
	"cmd/internal/sessioninventory/runtime.go",
	"cmd/internal/sessioninventory/scan.go",
	"cmd/internal/sessioninventory/scan_agy.go",
	"cmd/internal/sessioninventory/scan_codex.go",
	"cmd/internal/sessioninventory/scan_helpers.go",
	"cmd/internal/sessioninventory/scan_muse.go",
	"cmd/internal/sessioninventory/scanner_state.go",
	"cmd/internal/sessioninventory/target.go",
	"cmd/internal/sessioninventory/usage.go",
	"cmd/internal/sessioninventorytest/fake_runtime.go",
	"cmd/internal/sessionledger/store.go",
	"cmd/internal/sessionledger/store_unix.go",
	"cmd/internal/sessionwatch/runcli.go",
	"cmd/internal/sessionwatch/runtime.go",
	"cmd/internal/slugcmd/slug.go",
	"cmd/internal/strictjson/decode.go",
	"cmd/internal/termcmd/rename.go",
	"cmd/internal/termcmd/rename_input.go",
	"cmd/internal/termcmd/run.go",
	"cmd/internal/textwidth/textwidth.go",
	"cmd/internal/threadrecord/lifecycle.go",
	"cmd/internal/threadrecord/record.go",
	"cmd/internal/titlepoller/runcli.go",
	"cmd/internal/titlepoller/titlepoller.go",
	"cmd/internal/workbenchshortcut/generatecmd/main.go",
	"cmd/internal/workbenchshortcut/render_lua.go",
	"cmd/internal/wrapcmd/composer_recognizers.go",
	"cmd/internal/wrapcmd/harness_tty.go",
	"cmd/internal/wrapcmd/notification_lifecycle.go",
	"cmd/internal/wrapcmd/notification_rewriter.go",
	"cmd/internal/wrapcmd/terminal_model.go",
	"cmd/internal/zellijpane/zellijpane.go",
	"cmd/pair-go/main.go",
	"cmd/pair-launch-helper/main.go",
	"cmd/probes/couchstartrecovery/main.go",
	"doctor/doctor.sh",
	"doctor/emitter-health.sh",
	"nvim/adapt.lua",
	"nvim/annotate.lua",
	"nvim/changelog.lua",
	"nvim/confirm_quit.lua",
	"nvim/doctor.lua",
	"nvim/draft_send.lua",
	"nvim/init.lua",
	"nvim/marker_codec.lua",
	"nvim/normalization.lua",
	"nvim/pairlog.lua",
	"nvim/pair_poke.lua",
	"nvim/submission.lua",
	"nvim/review.lua",
	"nvim/review/apply.lua",
	"nvim/review/define.lua",
	"nvim/review/definition_seam.lua",
	"nvim/review/docflow.lua",
	"nvim/review/gate.lua",
	"nvim/review/handoff.lua",
	"nvim/review/init.lua",
	"nvim/review/markers.lua",
	"nvim/review/menu.lua",
	"nvim/review/mode.lua",
	"nvim/review/poke_bodies.lua",
	"nvim/review/projection.lua",
	"nvim/review/readiness.lua",
	"nvim/review/reconcile.lua",
	"nvim/review/reconstruct.lua",
	"nvim/review/resolve.lua",
	"nvim/review/seam.lua",
	"nvim/review/spinner.lua",
	"nvim/review/wrap.lua",
	"nvim/scrollback.lua",
	"nvim/slug.lua",
	"nvim/workbench_actions.lua",
	"nvim/workbench_route.lua",
	"nvim/zellij_trace.lua",
	"zellij/config.kdl",
	"zellij/layouts/main-2.kdl",
	"zellij/layouts/main-3.kdl",
}

func familyNames() []string {
	names := make([]string, 0, len(Families))
	for _, family := range Families {
		names = append(names, family.Name)
	}
	return names
}

func goStructVocabulary(family, value, field string, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: GoStructTagVocabulary, Use: field, Count: count}
}

func goCallVocabulary(family, value, callee string, argument, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: GoCallArgumentVocabulary, Use: callee, Argument: argument, Count: count}
}

func goCallVocabularyValues(family, callee string, argument int, values ...string) []VocabularyAllowance {
	out := make([]VocabularyAllowance, 0, len(values))
	for _, value := range values {
		out = append(out, goCallVocabulary(family, value, callee, argument, 1))
	}
	return out
}

func goCallVocabularyFamilies(value, callee string, argument int, families ...string) []VocabularyAllowance {
	out := make([]VocabularyAllowance, 0, len(families))
	for _, family := range families {
		out = append(out, goCallVocabulary(family, value, callee, argument, 1))
	}
	return out
}

const continuationBootstrapPrompt = `Continue Pair tag %s with %s.

The previous driver was %s. No continuation doc was found.

First reconstruct the current work state from this tag's persisted Pair files:
- draft-%s.md
- log-%s.md
- queue-%s/
- parked-%s and parked-scrollback-%s-*.raw/events.jsonl if present

Create a continuation-quality summary from the available local state before making code changes. Preserve the tag identity; do not create a sibling tag.
`

func goCaseVocabulary(family, value, function string, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: GoCaseValueVocabulary, Use: function, Count: count}
}

func goComparisonVocabulary(family, value, function string, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: GoComparisonVocabulary, Use: function, Count: count}
}

func exactLineVocabulary(family, value string, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: ExactLineVocabulary, Count: count}
}
