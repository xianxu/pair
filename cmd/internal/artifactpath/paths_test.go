package artifactpath

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestResolveRejectsUnsafeCompositeAddresses(t *testing.T) {
	t.Parallel()

	valid := Address{DataDir: "/data", RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	for name, address := range map[string]Address{
		"relative data directory": {DataDir: "data", RepoScope: valid.RepoScope, Tag: valid.Tag},
		"traversing scope":        {DataDir: valid.DataDir, RepoScope: "../scope", Tag: valid.Tag},
		"traversing tag":          {DataDir: valid.DataDir, RepoScope: valid.RepoScope, Tag: "../tag"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Resolve(address); err == nil {
				t.Fatalf("Resolve(%+v) succeeded", address)
			}
		})
	}
}

func TestEveryArtifactPathStaysInsideCompositeScope(t *testing.T) {
	t.Parallel()

	paths, err := Resolve(Address{
		DataDir:   "/data",
		RepoScope: "0123456789abcdef",
		Tag:       "work",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantScope := "/data/repos/0123456789abcdef"
	got := map[string]string{
		"draft":             paths.Draft(),
		"ledger":            paths.Ledger(),
		"log":               paths.Log(),
		"queue":             paths.QueueDir(),
		"config":            paths.Config("codex"),
		"config glob":       paths.ConfigGlob(),
		"agent pid":         paths.AgentPID(),
		"agent ready":       paths.AgentReady("codex"),
		"pane":              paths.Pane("codex"),
		"pane glob":         paths.PaneGlob(),
		"outer tty":         paths.OuterTTY(),
		"parked":            paths.Parked(),
		"parked scrollback": paths.ParkedScrollback("20260826T120000"),
		"adapt":             paths.AdaptLog(),
		"image":             paths.ImageCapture(),
		"continuation":      paths.Continuation(),
		"layout":            paths.WorkbenchLayout(),
		"restart":           paths.Restart(),
		"picker":            paths.DraftPane(),
		"session binding":   paths.SessionBindings(),
		"agent output":      paths.AgentOutput(),
		"agent picks":       paths.AgentPicks(),
		"nvim pid":          paths.NvimPID("draft"),
		"scrollback raw":    paths.ScrollbackRaw("codex"),
		"scrollback events": paths.ScrollbackEvents("codex"),
		"changelog":         paths.Changelog("codex"),
		"changelog ready":   paths.ChangelogReady("codex"),
		"changelog session": paths.ChangelogSession("codex", "019eff64-6ceb-7e72-9d41-a735a97029ac"),
		"agent draft":       paths.AgentDraft("codex"),
		"thread claim":      paths.ThreadClaim(),
		"quote":             paths.Quote(),
		"slug":              paths.Slug(),
		"slug proposed":     paths.SlugProposed(),
		"title pid":         paths.TitlePID(),
		"pair wrap pid":     paths.PairWrapPID(),
		"wrap events":       paths.WrapEvents(),
		"pending":           paths.ScrollbackPending(),
		"last left":         paths.LastLeftPane(),
		"last terminal":     paths.LastTerminalPane(),
		"terminal panes":    paths.TerminalPanes(),
		"zellij actions":    paths.ZellijActions(),
		"review open":       paths.ReviewOpen(),
		"review target":     paths.ReviewTarget(),
		"codex filter":      paths.CodexFilterKKP(),
	}

	for family, path := range got {
		rel, err := filepath.Rel(wantScope, path)
		if err != nil || rel == ".." || filepath.IsAbs(rel) || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
			t.Errorf("%s path escaped scope: %q (rel %q, err %v)", family, path, rel, err)
		}
	}

	if got["draft"] != wantScope+"/draft-work.md" {
		t.Fatalf("Draft() = %q", got["draft"])
	}
	if got["parked scrollback"] != wantScope+"/parked-scrollback-work-20260826T120000" {
		t.Fatalf("ParkedScrollback() = %q", got["parked scrollback"])
	}
}

func TestResolveScopedKeepsSameTagInDistinctSelectedScopes(t *testing.T) {
	t.Parallel()

	first, err := ResolveScoped("/data/repos/aaaaaaaaaaaaaaaa", "work")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolveScoped("/data/repos/bbbbbbbbbbbbbbbb", "work")
	if err != nil {
		t.Fatal(err)
	}
	if first.Draft() == second.Draft() {
		t.Fatalf("same tag collided across scopes: %q", first.Draft())
	}
	if first.Draft() != "/data/repos/aaaaaaaaaaaaaaaa/draft-work.md" {
		t.Fatalf("first Draft() = %q", first.Draft())
	}
	if _, err := ResolveScoped("relative", "work"); err == nil {
		t.Fatal("relative selected scope accepted")
	}
}

func TestResolveSelectedScopeOwnsTagIndependentArtifacts(t *testing.T) {
	t.Parallel()

	scope, err := ResolveSelectedScope("/data/repos/aaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if got := scope.SessionBindings(); got != "/data/repos/aaaaaaaaaaaaaaaa/session-names.jsonl" {
		t.Fatalf("SessionBindings() = %q", got)
	}
	if got, err := scope.AgentDefault("codex"); err != nil || got != "/data/repos/aaaaaaaaaaaaaaaa/agent-default-codex.json" {
		t.Fatalf("AgentDefault() = %q, %v", got, err)
	}
	if _, err := scope.AgentDefault("../codex"); err == nil {
		t.Fatal("unsafe agent default component accepted")
	}
}

func TestLegacyFlatPathsAreValidatedByArtifactAuthority(t *testing.T) {
	legacy, err := ResolveLegacyFlat("/data", "work")
	if err != nil {
		t.Fatal(err)
	}
	if got := legacy.QueueDir(); got != "/data/queue-work" {
		t.Fatalf("QueueDir() = %q", got)
	}
	if got := legacy.PanePrefix(); got != "pane-work-" {
		t.Fatalf("PanePrefix() = %q", got)
	}
	if got := legacy.HistoryGlobs(); !reflect.DeepEqual(got, []string{"/data/draft-*.md", "/data/log-*.md", "/data/ledger-*.jsonl"}) {
		t.Fatalf("HistoryGlobs() = %#v", got)
	}
	if _, err := ResolveLegacyFlat("/data", "../escape"); err == nil {
		t.Fatal("ResolveLegacyFlat accepted an invalid tag")
	}
	root, err := ResolveLegacyRoot("/data")
	if err != nil || root.SessionBindings() != "/data/session-names.jsonl" {
		t.Fatalf("legacy SessionBindings = %q, %v", root.SessionBindings(), err)
	}
}

func TestLegacyAndCurrentRenameArtifactsShareOneShape(t *testing.T) {
	t.Parallel()

	legacy, err := ResolveLegacyFlat("/pair-data", "thread")
	if err != nil {
		t.Fatal(err)
	}
	current, err := ResolveScoped("/pair-data/scopes/repo", "thread")
	if err != nil {
		t.Fatal(err)
	}
	legacyPaths := legacy.RenameArtifacts([]string{"codex"})
	currentPaths := current.RenameArtifacts([]string{"codex"})
	if len(legacyPaths) == 0 || len(legacyPaths) != len(currentPaths) {
		t.Fatalf("rename artifact counts: legacy=%d current=%d", len(legacyPaths), len(currentPaths))
	}
	for i := range legacyPaths {
		if filepath.Base(legacyPaths[i]) != filepath.Base(currentPaths[i]) {
			t.Fatalf("artifact %d basename mismatch: legacy=%q current=%q", i, legacyPaths[i], currentPaths[i])
		}
		if !strings.HasPrefix(legacyPaths[i], "/pair-data/") {
			t.Fatalf("legacy artifact escaped flat root: %q", legacyPaths[i])
		}
		if !strings.HasPrefix(currentPaths[i], "/pair-data/scopes/repo/") {
			t.Fatalf("current artifact escaped scope root: %q", currentPaths[i])
		}
	}
}

func TestEnvironmentBindingsCarryExactResolvedPaths(t *testing.T) {
	t.Parallel()

	paths, err := ResolveScoped("/data/repos/scope", "work")
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := paths.EnvironmentBindings("codex")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, binding := range bindings {
		if _, duplicate := seen[binding.Name]; duplicate {
			t.Fatalf("duplicate environment binding %s", binding.Name)
		}
		if !strings.HasPrefix(binding.Path, paths.ScopeDir()+string(filepath.Separator)) {
			t.Fatalf("%s escaped scope: %q", binding.Name, binding.Path)
		}
		seen[binding.Name] = binding.Path
	}
	for name, want := range map[string]string{
		"PAIR_DRAFT_PATH":           paths.Draft(),
		"PAIR_AGENT_CONFIG_PATH":    paths.Config("codex"),
		"PAIR_AGENT_PANE_PATH":      paths.Pane("codex"),
		"PAIR_SCROLLBACK_RAW_PATH":  paths.ScrollbackRaw("codex"),
		"PAIR_ADAPT_LOG_PATH":       paths.AdaptLog(),
		"PAIR_CHANGELOG_READY_PATH": paths.ChangelogReady("codex"),
	} {
		if seen[name] != want {
			t.Errorf("%s = %q, want %q", name, seen[name], want)
		}
	}
}

func TestCompanionArtifactSetsResolveEverySibling(t *testing.T) {
	paths, err := ResolveScoped("/data/repos/scope", "work")
	if err != nil {
		t.Fatal(err)
	}
	scrollback, err := paths.ScrollbackArtifacts("codex")
	if err != nil {
		t.Fatal(err)
	}
	if scrollback.Raw != "/data/repos/scope/scrollback-work-codex.raw" ||
		scrollback.Events != "/data/repos/scope/scrollback-work-codex.events.jsonl" ||
		scrollback.ANSI != "/data/repos/scope/scrollback-work-codex.ansi" ||
		scrollback.Viewport != "/data/repos/scope/scrollback-work-codex.viewport" ||
		scrollback.OpenLock != "/data/repos/scope/scrollback-work-codex.openlock" {
		t.Fatalf("scrollback companions = %+v", scrollback)
	}
	changelog, err := paths.ChangelogArtifacts("codex", "sid")
	if err != nil {
		t.Fatal(err)
	}
	if changelog.Log != "/data/repos/scope/changelog-work-codex-sid.md" ||
		changelog.Anchor != "/data/repos/scope/changelog-work-codex-sid.anchor" ||
		changelog.Cleaned != "/data/repos/scope/changelog-work-codex-sid.cleaned" ||
		changelog.OpenLock != "/data/repos/scope/changelog-work-codex-sid.openlock" ||
		changelog.DistillLock != "/data/repos/scope/changelog-work-codex-sid.distill.lock" ||
		changelog.Status != "/data/repos/scope/changelog-work-codex-sid.status" ||
		changelog.Ready != "/data/repos/scope/changelog-work-codex.ready" {
		t.Fatalf("changelog companions = %+v", changelog)
	}
}

func TestParameterizedArtifactsRejectUnsafeComponents(t *testing.T) {
	t.Parallel()

	paths, err := Resolve(Address{DataDir: "/data", RepoScope: "scope", Tag: "work"})
	if err != nil {
		t.Fatal(err)
	}
	for name, resolve := range map[string]func() (string, error){
		"config agent":      func() (string, error) { return paths.ConfigChecked("../codex") },
		"pane agent":        func() (string, error) { return paths.PaneChecked("../codex") },
		"parked timestamp":  func() (string, error) { return paths.ParkedScrollbackChecked("../then") },
		"nvim pid kind":     func() (string, error) { return paths.NvimPIDChecked("../draft") },
		"scrollback suffix": func() (string, error) { return paths.ScrollbackChecked("codex", "../raw") },
	} {
		t.Run(name, func(t *testing.T) {
			if path, err := resolve(); err == nil {
				t.Fatalf("unsafe component produced %q", path)
			}
		})
	}
}

func TestScopePathsOwnNvimPIDEnumerationAndEmbedParsing(t *testing.T) {
	scope, err := ResolveSelectedScope("/data/repos/scope")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := scope.NvimPIDGlob("draft"); err != nil || got != "/data/repos/scope/nvim-pid-*-draft" {
		t.Fatalf("NvimPIDGlob() = %q, %v", got, err)
	}
	if _, err := scope.NvimPIDGlob("../draft"); err == nil {
		t.Fatal("NvimPIDGlob accepted unsafe kind")
	}
	if got, ok := scope.TagFromNvimPID("/data/repos/scope/nvim-pid-my-tag-draft", "draft"); !ok || got != "my-tag" {
		t.Fatalf("TagFromNvimPID() = %q, %v", got, ok)
	}
	if _, ok := scope.TagFromNvimPID("/other/nvim-pid-my-tag-draft", "draft"); ok {
		t.Fatal("TagFromNvimPID accepted path outside selected scope")
	}

	cases := map[string]string{
		"nvim --embed --headless /data/repos/scope/draft-work.md":             "work",
		"/usr/bin/nvim --embed /data/repos/scope/draft-my-tag.md --more":      "my-tag",
		"nvim --embed /data/repos/scope/scrollback-work-claude.ansi":          "work",
		"nvim --embed /data/repos/scope/scrollback-my-tag-codex.ansi":         "my-tag",
		"nvim --embed /some/other/file":                                       "",
		"nvim --embed /data/repos/scope/scrollback-solo-claude.ansi trailing": "solo",
	}
	for argv, want := range cases {
		if got := scope.TagFromNvimEmbedArgv(argv); got != want {
			t.Fatalf("TagFromNvimEmbedArgv(%q) = %q, want %q", argv, got, want)
		}
	}
}

func TestPathsOwnPaneSidecarParsing(t *testing.T) {
	paths, err := ResolveScoped("/data/repos/scope", "work")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := paths.AgentFromPane("/data/repos/scope/pane-work-codex.json"); !ok || got != "codex" {
		t.Fatalf("AgentFromPane() = %q, %v", got, ok)
	}
	for _, invalid := range []string{
		"/other/pane-work-codex.json",
		"/data/repos/scope/pane-other-codex.json",
		"/data/repos/scope/pane-work-../codex.json",
	} {
		if _, ok := paths.AgentFromPane(invalid); ok {
			t.Fatalf("AgentFromPane accepted %q", invalid)
		}
	}
}

func TestArtifactVocabularyOwnsExternalDraftCommandRecognition(t *testing.T) {
	if !CommandReferencesDraftArtifact("nvim /data/draft-work.md") {
		t.Fatal("draft artifact command was not recognized")
	}
	if CommandReferencesDraftArtifact("nvim /tmp/notes.md") {
		t.Fatal("unrelated command was recognized as a draft artifact")
	}
}

func TestHistoryAndConfigSidecarRecognition(t *testing.T) {
	t.Parallel()
	if tag, ok := TagFromHistorySidecar("log-work.md"); !ok || tag != "work" || !IsLogHistorySidecar("log-work.md") {
		t.Fatalf("log recognition = %q,%v", tag, ok)
	}
	if tag, agent, ok := TagAgentFromConfigSidecar("config-my-work-codex.json", []string{"claude", "codex"}); !ok || tag != "my-work" || agent != "codex" {
		t.Fatalf("config recognition = %q,%q,%v", tag, agent, ok)
	}
	for _, invalid := range []string{"config-work-other.json", "config-../work-codex.json", "log-.md"} {
		if tag, agent, ok := TagAgentFromConfigSidecar(invalid, []string{"codex"}); ok || tag != "" || agent != "" {
			t.Fatalf("accepted invalid config %q", invalid)
		}
	}
}

func TestPairCachePathsOwnRestartAndQuitMarkers(t *testing.T) {
	cache, err := ResolvePairCache("/tmp/home")
	if err != nil {
		t.Fatal(err)
	}
	got, err := cache.Restart("pair-dev")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/home/.cache/pair/restart-pair-dev"; got != want {
		t.Fatalf("Restart() = %q, want %q", got, want)
	}
	got, err = cache.Quit("pair-dev")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/home/.cache/pair/quit-pair-dev"; got != want {
		t.Fatalf("Quit() = %q, want %q", got, want)
	}
	got, err = cache.Restart("📁pair-work")
	if err != nil {
		t.Fatalf("Restart() rejected a valid Zellij session name: %v", err)
	}
	if want := "/tmp/home/.cache/pair/restart-📁pair-work"; got != want {
		t.Fatalf("Restart() = %q, want %q", got, want)
	}
	if _, err := ResolvePairCache("relative"); err == nil {
		t.Fatal("relative home accepted")
	}
	if _, err := cache.Restart("../escape"); err == nil {
		t.Fatal("invalid session escaped cache")
	}
}
