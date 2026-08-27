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
		"PAIR_DRAFT_PATH":          paths.Draft(),
		"PAIR_AGENT_CONFIG_PATH":   paths.Config("codex"),
		"PAIR_AGENT_PANE_PATH":     paths.Pane("codex"),
		"PAIR_SCROLLBACK_RAW_PATH": paths.ScrollbackRaw("codex"),
		"PAIR_ADAPT_LOG_PATH":      paths.AdaptLog(),
	} {
		if seen[name] != want {
			t.Errorf("%s = %q, want %q", name, seen[name], want)
		}
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
