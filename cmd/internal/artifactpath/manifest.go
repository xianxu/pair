package artifactpath

// Family is one checked artifact filename family. Token is the stable literal
// a source-coverage test uses to find constructors and exact-path consumers.
type Family struct {
	Name  string
	Token string
}

// SourceKind records why a production source may mention an artifact token.
type SourceKind string

const (
	Constructor      SourceKind = "constructor"
	ResolvedConsumer SourceKind = "resolved-consumer"
	GeneratedMirror  SourceKind = "generated-mirror"
)

// SourceClassification is deliberately exact: Path names one repository file
// and Families names every artifact family it constructs or consumes. New
// files cannot disappear behind directory or wildcard exemptions.
type SourceClassification struct {
	Path     string
	Kind     SourceKind
	Families []string
}

// Families is the exhaustive tag-bearing artifact vocabulary. Adding a new
// family requires adding its constructor and classifying every source consumer.
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
	{Path: "cmd/internal/continuationcmd/continuation.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/launcher/compaction.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/launcher/createlogic.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/launcher/ledger.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/launcher/markers.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/opener/opener.go", Kind: ResolvedConsumer, Families: []string{"changelog", "native-session"}},
	{Path: "cmd/internal/sessionwatch/sessionwatch.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/titlepoller/runtime.go", Kind: ResolvedConsumer, Families: []string{"pane"}},
	{Path: "cmd/internal/transcript/transcript.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/launcher/history.go", Kind: ResolvedConsumer, Families: []string{"draft", "ledger", "log", "queue"}},
	{Path: "cmd/internal/launcher/layoutflow.go", Kind: ResolvedConsumer, Families: []string{"draft"}},
	{Path: "cmd/internal/launcher/legacy_live.go", Kind: ResolvedConsumer, Families: []string{"pane"}},
	{Path: "cmd/internal/launcher/lifecycle.go", Kind: ResolvedConsumer, Families: []string{"draft", "scrollback"}},
	{Path: "cmd/internal/launcher/migrate.go", Kind: ResolvedConsumer, Families: []string{"queue"}},
	{Path: "cmd/internal/launcher/osruntime.go", Kind: ResolvedConsumer, Families: []string{"nvim-pid", "restart"}},
	{Path: "cmd/internal/opener/runtime.go", Kind: ResolvedConsumer, Families: []string{"scrollback"}},
	{Path: "cmd/internal/scrollbackcmd/scrollbackcmd.go", Kind: ResolvedConsumer, Families: []string{"scrollback"}},
	{Path: "cmd/internal/sessionwatch/run.go", Kind: ResolvedConsumer, Families: []string{"native-session"}},
	{Path: "cmd/internal/slugcmd/slugcmd.go", Kind: ResolvedConsumer, Families: []string{"slug"}},
	{Path: "cmd/internal/wrapcmd/wrap.go", Kind: ResolvedConsumer, Families: []string{"agent", "scrollback"}},
	{Path: "doctor/doctor.sh", Kind: ResolvedConsumer, Families: []string{"adapt"}},
	{Path: "nvim/review/record.lua", Kind: ResolvedConsumer, Families: []string{"review"}},

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

func familyNames() []string {
	names := make([]string, 0, len(Families))
	for _, family := range Families {
		names = append(names, family.Name)
	}
	return names
}
