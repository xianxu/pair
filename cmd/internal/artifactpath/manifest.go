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
var ResolvedBindings = []ResolvedBinding{
	{Name: "cache-restart", Family: "restart", Resolver: "ResolvePairCache", Member: "Restart"},
	{Name: "legacy-pane", Family: "pane", Resolver: "ResolveLegacyRoot", Member: "PanePrefix"},
	{Name: "legacy-queue", Family: "queue", Resolver: "ResolveLegacyFlat", Member: "QueueDir"},
	{Name: "scoped-adapt", Family: "adapt", Resolver: "ResolveScoped", Member: "AdaptLog"},
	{Name: "scoped-agent", Family: "agent", Resolver: "ResolveScoped", Member: "Agent"},
	{Name: "scoped-agent-output", Family: "agent", Resolver: "ResolveScoped", Member: "AgentOutput"},
	{Name: "scoped-agent-pid", Family: "agent-pid", Resolver: "ResolveScoped", Member: "AgentPID"},
	{Name: "scoped-agent-ready", Family: "agent-ready", Resolver: "ResolveScoped", Member: "AgentReadyChecked"},
	{Name: "scoped-changelog", Family: "changelog", Resolver: "ResolveScoped", Member: "ChangelogSessionChecked"},
	{Name: "scoped-config", Family: "config", Resolver: "ResolveScoped", Member: "ConfigChecked"},
	{Name: "scoped-config-glob", Family: "config", Resolver: "ResolveScoped", Member: "ConfigGlob"},
	{Name: "scoped-draft", Family: "draft", Resolver: "ResolveScoped", Member: "Draft"},
	{Name: "scoped-image-capture", Family: "image-capture", Resolver: "ResolveScoped", Member: "ImageCapture"},
	{Name: "scoped-ledger", Family: "ledger", Resolver: "ResolveScoped", Member: "Ledger"},
	{Name: "scoped-nvim-pid", Family: "nvim-pid", Resolver: "ResolveScoped", Member: "NvimPID"},
	{Name: "scoped-outer-tty", Family: "outer-tty", Resolver: "ResolveScoped", Member: "OuterTTY"},
	{Name: "scoped-pair-wrap-pid", Family: "pair-wrap-pid", Resolver: "ResolveScoped", Member: "PairWrapPID"},
	{Name: "scoped-pane", Family: "pane", Resolver: "ResolveScoped", Member: "PaneChecked"},
	{Name: "scoped-pane-glob", Family: "pane", Resolver: "ResolveScoped", Member: "PaneGlob"},
	{Name: "scoped-parked", Family: "parked", Resolver: "ResolveScoped", Member: "Parked"},
	{Name: "scoped-parked-scrollback", Family: "parked-scrollback", Resolver: "ResolveScoped", Member: "ParkedScrollbackArtifacts"},
	{Name: "scoped-queue", Family: "queue", Resolver: "ResolveScoped", Member: "QueueDir"},
	{Name: "scoped-scrollback", Family: "scrollback", Resolver: "ResolveScoped", Member: "ScrollbackArtifacts"},
	{Name: "scoped-scrollback-pending", Family: "scrollback-pending", Resolver: "ResolveScoped", Member: "ScrollbackPending"},
	{Name: "scoped-slug", Family: "slug", Resolver: "ResolveScoped", Member: "Slug"},
	{Name: "scoped-title-pid", Family: "title-pid", Resolver: "ResolveScoped", Member: "TitlePID"},
	{Name: "scoped-wrap-events", Family: "wrap-events", Resolver: "ResolveScoped", Member: "WrapEvents"},
	{Name: "selected-nvim-pid", Family: "nvim-pid", Resolver: "ResolveSelectedScope", Member: "NvimPIDGlob"},
	{Name: "selected-session-binding", Family: "session-binding", Resolver: "ResolveSelectedScope", Member: "SessionBindings"},
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
}

// SourceClassifications is checked by coverage_test.go against production Go,
// shell, Lua, and KDL sources.
// pair:m5-concept pure
var SourceClassifications = []SourceClassification{
	{Path: "cmd/internal/artifactpath/paths.go", Kind: Constructor, Families: []string{
		"adapt", "agent", "agent-pid", "agent-ready", "changelog", "config",
		"continuation", "draft", "image-capture", "layout", "layout-mode",
		"last-left-pane", "last-terminal-pane", "ledger", "log", "nvim-pid",
		"outer-tty", "pair-wrap-pid", "pane", "parked", "parked-scrollback",
		"picker", "queue", "quote", "restart", "review", "scrollback",
		"scrollback-pending", "session-binding", "slug", "terminal-panes",
		"thread-claim", "title-pid", "wrap-events", "zellij-actions",
		"codex-filter-kkp",
	}},
	{Path: "cmd/internal/artifactpath/manifest.go", Kind: Constructor, Families: familyNames()},
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
	{Path: "cmd/internal/launcher/markers.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goCaseVocabulary("native-session", "session_id", "parseRestartMarker", 1),
	}},
	{Path: "cmd/internal/opener/opener.go", Kind: ResolvedConsumer, Families: []string{"changelog"}, BindingNames: []string{"scoped-changelog"}, Vocabulary: []VocabularyAllowance{
		goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
	}},
	{Path: "cmd/internal/sessionwatch/sessionwatch.go", Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: []VocabularyAllowance{
		goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
	}},
	{Path: "cmd/internal/titlepoller/runtime.go", Kind: ResolvedConsumer, Families: []string{"pane"}, BindingNames: []string{"scoped-pane-glob"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("pane", "--pane-id", "os/exec.Command", 5, 1),
	}},
	{Path: "cmd/internal/transcript/transcript.go", Kind: ResolvedConsumer, Families: []string{"config"}, BindingNames: []string{"scoped-config"}, Vocabulary: []VocabularyAllowance{
		goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
	}},
	{Path: "cmd/internal/launcher/legacy_live.go", Kind: ResolvedConsumer, Families: []string{"pane"}, BindingNames: []string{"legacy-pane"}},
	{Path: "cmd/internal/launcher/lifecycle.go", Kind: ResolvedConsumer,
		Families:     []string{"adapt", "agent", "draft", "image-capture", "outer-tty", "pair-wrap-pid", "pane", "scrollback"},
		BindingNames: []string{"scoped-adapt", "scoped-agent", "scoped-draft", "scoped-image-capture", "scoped-outer-tty", "scoped-pair-wrap-pid", "scoped-pane", "scoped-scrollback"}},
	{Path: "cmd/internal/launcher/migrate.go", Kind: ResolvedConsumer, Families: []string{"queue"}, BindingNames: []string{"legacy-queue", "scoped-queue"}},
	{Path: "cmd/internal/launcher/osruntime.go", Kind: ResolvedConsumer,
		Families:     []string{"agent", "agent-pid", "config", "draft", "ledger", "nvim-pid", "outer-tty", "parked", "parked-scrollback", "restart", "scrollback", "session-binding", "title-pid"},
		BindingNames: []string{"cache-restart", "scoped-agent", "scoped-agent-pid", "scoped-config-glob", "scoped-draft", "scoped-ledger", "scoped-nvim-pid", "scoped-outer-tty", "scoped-parked", "scoped-parked-scrollback", "scoped-scrollback", "selected-nvim-pid", "selected-session-binding", "scoped-title-pid"},
		Vocabulary:   []VocabularyAllowance{goCallVocabulary("config", "--config-dir", "os/exec.Command", 1, 2)}},
	{Path: "cmd/internal/opener/runtime.go", Kind: VocabularyConsumer, Families: []string{"scrollback"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("scrollback", "scrollback-render exit %d", "fmt.Sprintf", 0, 1),
	}},
	{Path: "cmd/internal/scrollbackcmd/scrollbackcmd.go", Kind: VocabularyConsumer, Families: []string{"scrollback"}, Vocabulary: []VocabularyAllowance{
		goCallVocabulary("scrollback", "pair-scrollback-render", "flag.NewFlagSet", 0, 1),
		goCallVocabulary("scrollback", "usage: pair-scrollback-render [--plain] [--viewport F] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n", "fmt.Fprintf", 1, 1),
		goCallVocabulary("scrollback", "scrollback-render: %v\n", "fmt.Fprintf", 1, 1),
	}},
	{Path: "cmd/internal/sessionwatch/run.go", Kind: ResolvedConsumer,
		Families: []string{"agent-pid", "config", "ledger"}, BindingNames: []string{"scoped-agent-pid", "scoped-config", "scoped-ledger"},
		Vocabulary: []VocabularyAllowance{
			goStructVocabulary("native-session", `json:"session_id"`, "SessionID", 1),
			goCallVocabulary("native-session", "session_id=", "method.Log", 1, 1),
		}},
	{Path: "cmd/internal/slugcmd/slugcmd.go", Kind: ResolvedConsumer,
		Families: []string{"agent-pid", "slug"}, BindingNames: []string{"scoped-agent-pid", "scoped-slug"},
		Vocabulary: []VocabularyAllowance{
			goCallVocabulary("config", "no session_id in config-%s-%s.json", "function.logf", 0, 1),
			goCallVocabulary("native-session", "no session_id in config-%s-%s.json", "function.logf", 0, 1),
			goCallVocabulary("slug", "slug-parse", "method.Log", 1, 4),
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
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/doctor.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/marker_codec.lua", Kind: GeneratedMirror},
	{Path: "cmd/internal/runtimebundle/assets/runtime/files/nvim/pair_poke.lua", Kind: GeneratedMirror},
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
	"cmd/internal/adapt/adapt.go",
	"cmd/internal/agentcmd/restart.go",
	"cmd/internal/ansi/ansi.go",
	"cmd/internal/changelogcmd/changelogcmd.go",
	"cmd/internal/changelogcmd/distill.go",
	"cmd/internal/changelogcmd/prompt.go",
	"cmd/internal/clipcmd/clipcmd.go",
	"cmd/internal/clipcmd/run.go",
	"cmd/internal/clipcmd/runcli.go",
	"cmd/internal/clipcmd/runtime.go",
	"cmd/internal/codexsid/codexsid.go",
	"cmd/internal/contextcmd/contextcmd.go",
	"cmd/internal/continuationcmd/continuationcmd.go",
	"cmd/internal/continuationcmd/draft.go",
	"cmd/internal/continuationcmd/git.go",
	"cmd/internal/couchcmd/errors.go",
	"cmd/internal/couchcmd/run.go",
	"cmd/internal/couchcore/actor.go",
	"cmd/internal/couchcore/actorid.go",
	"cmd/internal/couchcore/admission.go",
	"cmd/internal/couchcore/artifactcollision.go",
	"cmd/internal/couchcore/artifactcollision_fake.go",
	"cmd/internal/couchcore/clock.go",
	"cmd/internal/couchcore/couch.go",
	"cmd/internal/couchcore/git.go",
	"cmd/internal/couchcore/git_fake.go",
	"cmd/internal/couchcore/launchhelper.go",
	"cmd/internal/couchcore/launchprofile.go",
	"cmd/internal/couchcore/mailbox.go",
	"cmd/internal/couchcore/migration.go",
	"cmd/internal/couchcore/namespace.go",
	"cmd/internal/couchcore/naming.go",
	"cmd/internal/couchcore/operationdispatch.go",
	"cmd/internal/couchcore/ops.go",
	"cmd/internal/couchcore/path.go",
	"cmd/internal/couchcore/pathops.go",
	"cmd/internal/couchcore/policyresolver.go",
	"cmd/internal/couchcore/policyresolver_exec.go",
	"cmd/internal/couchcore/policyresolver_fake.go",
	"cmd/internal/couchcore/procops.go",
	"cmd/internal/couchcore/ptyrunner.go",
	"cmd/internal/couchcore/registry.go",
	"cmd/internal/couchcore/runner.go",
	"cmd/internal/couchcore/runner_fake.go",
	"cmd/internal/couchcore/standalone.go",
	"cmd/internal/couchcore/startargs.go",
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
	"cmd/internal/couchtty/console.go",
	"cmd/internal/couchtty/focus.go",
	"cmd/internal/couchtty/keys.go",
	"cmd/internal/couchtty/notice.go",
	"cmd/internal/couchtty/panel.go",
	"cmd/internal/couchtty/panelkeys.go",
	"cmd/internal/couchtty/reserve.go",
	"cmd/internal/ctxmeter/ctxmeter.go",
	"cmd/internal/dispatcher/dispatcher.go",
	"cmd/internal/draftroute/route.go",
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
	"cmd/internal/keyhelp/sections.go",
	"cmd/internal/keyhelp/sources.go",
	"cmd/internal/keyscmd/keyscmd.go",
	"cmd/internal/launcher/agent_defaults.go",
	"cmd/internal/launcher/agentargs.go",
	"cmd/internal/launcher/args.go",
	"cmd/internal/launcher/config.go",
	"cmd/internal/launcher/continuation.go",
	"cmd/internal/launcher/createflow.go",
	"cmd/internal/launcher/datadir.go",
	"cmd/internal/launcher/decision.go",
	"cmd/internal/launcher/format.go",
	"cmd/internal/launcher/help.go",
	"cmd/internal/launcher/history.go",
	"cmd/internal/launcher/launch_args_policy.go",
	"cmd/internal/launcher/layout.go",
	"cmd/internal/launcher/layoutflow.go",
	"cmd/internal/launcher/list.go",
	"cmd/internal/launcher/pathenv.go",
	"cmd/internal/launcher/pick.go",
	"cmd/internal/launcher/readiness.go",
	"cmd/internal/launcher/rename.go",
	"cmd/internal/launcher/restart.go",
	"cmd/internal/launcher/run.go",
	"cmd/internal/launcher/runcli.go",
	"cmd/internal/launcher/runtime.go",
	"cmd/internal/launcher/scope.go",
	"cmd/internal/launcher/scoped_paths.go",
	"cmd/internal/launcher/session.go",
	"cmd/internal/launcher/session_index.go",
	"cmd/internal/launcher/session_quiescence.go",
	"cmd/internal/launcher/tag.go",
	"cmd/internal/launcher/thread_claim.go",
	"cmd/internal/launcher/thread_index.go",
	"cmd/internal/launcher/zellij.go",
	"cmd/internal/launcher/zellijparse.go",
	"cmd/internal/layoutcmd/layoutcmd.go",
	"cmd/internal/layoutcmd/resizeplan.go",
	"cmd/internal/model/model.go",
	"cmd/internal/opener/run.go",
	"cmd/internal/opener/runcli.go",
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
	"cmd/internal/reviewcmd/run.go",
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
	"cmd/internal/sessionwatch/runcli.go",
	"cmd/internal/sessionwatch/runtime.go",
	"cmd/internal/slugcmd/slug.go",
	"cmd/internal/strictjson/decode.go",
	"cmd/internal/termcmd/rename.go",
	"cmd/internal/termcmd/rename_input.go",
	"cmd/internal/termcmd/run.go",
	"cmd/internal/textwidth/textwidth.go",
	"cmd/internal/threadrecord/record.go",
	"cmd/internal/titlepoller/run.go",
	"cmd/internal/titlepoller/runcli.go",
	"cmd/internal/titlepoller/titlepoller.go",
	"cmd/internal/workbenchshortcut/generatecmd/main.go",
	"cmd/internal/workbenchshortcut/render_lua.go",
	"cmd/internal/workbenchshortcut/shortcut.go",
	"cmd/internal/wrapcmd/composer_recognizers.go",
	"cmd/internal/wrapcmd/harness_tty.go",
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
	"nvim/doctor.lua",
	"nvim/init.lua",
	"nvim/marker_codec.lua",
	"nvim/pair_poke.lua",
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

func goCaseVocabulary(family, value, function string, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: GoCaseValueVocabulary, Use: function, Count: count}
}

func goComparisonVocabulary(family, value, function string, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: GoComparisonVocabulary, Use: function, Count: count}
}

func exactLineVocabulary(family, value string, count int) VocabularyAllowance {
	return VocabularyAllowance{Family: family, Value: value, Context: ExactLineVocabulary, Count: count}
}
