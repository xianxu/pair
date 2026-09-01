package couchcmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// readme returns the repo's README, located relative to this package rather
// than to the developer's checkout.
func readme(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	return string(raw)
}

func repoDocument(t *testing.T, path ...string) string {
	t.Helper()
	parts := append([]string{"..", "..", ".."}, path...)
	raw, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Join(path...), err)
	}
	return string(raw)
}

// M3 moves the ordinary switcher to proof-bearing actionable inventory while
// retaining ThreadInventory for diagnostics. Guard source and current-state
// summaries together so later edits cannot collapse those two views.
func TestM3DocsMatchActionableSwitcherInventoryProvider(t *testing.T) {
	if source := repoDocument(t, "cmd", "internal", "couchcmd", "run.go"); !strings.Contains(source, "console.SetActionableProvider") || !strings.Contains(source, "c.ActionableThreadInventoryContext") {
		t.Fatal("M3 contract expects the switcher to consume context-bearing actionable inventory")
	}

	checks := []struct {
		path []string
		want string
	}{
		{[]string{"atlas", "couch.md"}, "ordinary switcher reads the\nactionable projection"},
		{[]string{"workshop", "projects", "couch.md"}, "hierarchical switcher the reachable Console UI over the\nproof-bearing actionable projection"},
		{[]string{"README.md"}, "Unsupported or ambiguous lifecycle\nrecords stay available through `couch --list` / `couch --show` diagnostics"},
	}
	for _, check := range checks {
		if doc := repoDocument(t, check.path...); !strings.Contains(doc, check.want) {
			t.Errorf("%s does not declare the current M3 consumer stage %q", filepath.Join(check.path...), check.want)
		}
	}
}

// The panel renderer and README consume the same key inventory. Adding a key
// to the UI therefore makes this test fail until its operator documentation
// has a home in README, instead of relying on a boundary reviewer to remember.
func TestREADMEDocumentsEveryPanelControl(t *testing.T) {
	doc := readme(t)
	for _, control := range couchtty.MenuControls() {
		if !strings.Contains(doc, control.Keys) {
			t.Errorf("README does not document panel key %q (%s)", control.Keys, control.Action)
		}
	}
}

func TestREADMERootEscapeMatchesReducerWhenNoActorIsLive(t *testing.T) {
	const documented = "with no live actor, the switcher stays open and reports why"
	if doc := readme(t); !strings.Contains(doc, documented) {
		t.Fatalf("README does not document root Escape semantics %q", documented)
	}
	state := couchtty.NewMenuState([]couchcore.ActionableThreadSummary{{
		Address: couchcore.ThreadAddress{RepoScope: "repo", Tag: "couch-parked"},
		State:   couchcore.ThreadParked,
	}}, couchcore.ThreadAddress{})
	got, effects := couchtty.ReduceMenu(state, couchtty.MenuEvent{
		Kind: couchtty.MenuEventKey, Key: couchtty.PanelKey{Kind: couchtty.KeyEscape},
	})
	if len(effects) != 0 || got.Notice.Text != "no live thread can receive focus" || len(got.Frames) != 1 {
		t.Fatalf("root Escape behavior drifted from README: state=%+v effects=%+v", got, effects)
	}
}

func TestREADMEDocumentsOnlyThePublicProjection(t *testing.T) {
	doc := readme(t)
	for _, command := range []string{"couch [<repo>]", "couch --list", "couch --show <ref>"} {
		if !strings.Contains(doc, command) {
			t.Errorf("README does not document %q", command)
		}
	}
	for _, op := range couchcore.Operations() {
		if op.Presentation != couchcore.PresentationList && op.Presentation != couchcore.PresentationShow && strings.Contains(doc, "couch "+op.Name) {
			t.Errorf("README exposes non-public operation %q", op.Name)
		}
	}
	for _, option := range []string{"--no-console", "--agent="} {
		if strings.Contains(doc, option) {
			t.Errorf("README exposes removed public option %q", option)
		}
	}
}

// Flags that change what couch does with the operator's terminal must be
// documented, and so must the fact that couch CLAIMS a key from every child --
// an operator whose ctrl-space stops reaching their editor needs an explanation
// they can find.
func TestREADMEDocumentsTheOperatorFacingSurface(t *testing.T) {
	doc := readme(t)
	for _, want := range []string{
		"ctrl-space", // the key couch takes from every child
		"default: .", // the path default, which is how "home" is chosen
		"reserves the bottom row",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("README does not mention %q", want)
		}
	}
}

func TestREADMEDocumentsM3ThreadSemantics(t *testing.T) {
	doc := readme(t)
	for _, want := range []string{
		"`{repository scope, opaque tag}`",
		"current directory",
		"exact opaque tag wins",
		"human name",
		"canonical working path",
		"ambiguous match",
		"empty string",
		"Standalone Pair neither reads nor mutates Couch's ThreadStore",
		"not Pair addresses",
		"session bindings and tag history",
		"diagnostic view",
		"`{repository scope}/{opaque tag}`",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("README does not document M3 behavior %q", want)
		}
	}
}

// The atlas owns the complete typed operation vocabulary; public README help
// intentionally owns only the presentation projection.
func TestAtlasDocumentsEveryTypedOperation(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "atlas", "couch.md"))
	if err != nil {
		t.Fatalf("read atlas/couch.md: %v", err)
	}
	doc := string(raw)
	for _, op := range couchcore.Operations() {
		if !strings.Contains(doc, op.Name) {
			t.Errorf("atlas/couch.md does not document typed operation `%s`", op.Name)
		}
	}
}

func TestOperationPresentationDocs(t *testing.T) {
	readme := readme(t)
	atlas := repoDocument(t, "atlas", "couch.md")
	for _, op := range couchcore.Operations() {
		switch op.Presentation {
		case couchcore.PresentationList:
			if !strings.Contains(readme, "couch --list") {
				t.Error("list presentation has no README home")
			}
		case couchcore.PresentationShow:
			if !strings.Contains(readme, "couch --show") {
				t.Error("show presentation has no README home")
			}
		case couchcore.PresentationInternal:
			if !strings.Contains(atlas, "couch --internal "+op.Name) {
				t.Errorf("internal operation %q has no atlas protocol home", op.Name)
			}
		case couchcore.PresentationTUI:
			if !strings.Contains(atlas, op.Name) {
				t.Errorf("TUI operation %q has no atlas home", op.Name)
			}
		default:
			t.Errorf("operation %q has unknown presentation", op.Name)
		}
		if op.Presentation != couchcore.PresentationList && op.Presentation != couchcore.PresentationShow && strings.Contains(readme, "couch "+op.Name) {
			t.Errorf("non-public operation %q appears as a README command", op.Name)
		}
	}
}

func TestNoCurrentSourcesAdvertiseObsoleteCouchArgv(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	paths := []string{
		"README.md", "atlas/couch.md",
		"workshop/issues/000153-couch-managed-worktree-lifecycle.md",
		"probes/zellijpark/main.go",
	}
	err := filepath.Walk(filepath.Join(root, "cmd"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			paths = append(paths, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	obsolete := regexp.MustCompile(`couch (start|list|show|publish-description|resume|park|leave|stop|name|describe)(?:\s|["` + "`" + `])`)
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		for lineNumber, line := range strings.Split(string(raw), "\n") {
			if match := obsolete.FindString(line); match != "" && !strings.Contains(line, "obsolete-argv-rejection") {
				t.Errorf("%s:%d advertises obsolete argv %q", rel, lineNumber+1, match)
			}
		}
	}
}
