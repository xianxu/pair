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

// M1 introduces the proof-bearing inventory authority, but the shipped flat
// panel remains on raw diagnostic inventory until M3 migrates the consumer.
// Guard the source and every durable current-state summary together so a staged
// migration cannot be described as already delivered (BR-4).
func TestM1DocsMatchTransitionalPanelInventoryProvider(t *testing.T) {
	if source := repoDocument(t, "cmd", "internal", "couchcmd", "run.go"); !strings.Contains(source, "return c.ThreadInventory()") {
		t.Fatal("M1 contract expects the transitional panel to consume raw ThreadInventory")
	}

	checks := []struct {
		path []string
		want string
	}{
		{[]string{"atlas", "couch.md"}, "transitional flat panel remains wired to raw `ThreadInventory`"},
		{[]string{"workshop", "projects", "couch.md"}, "transitional flat panel remains wired to raw `ThreadInventory`"},
		{[]string{"workshop", "issues", "000151-hierarchical-thread-menu.md"}, "transitional flat panel remains wired to raw `ThreadInventory`"},
		{[]string{"workshop", "plans", "000151-hierarchical-thread-menu-plan.md"}, "M1 exposes the authority while M3 adopts it"},
		{[]string{"README.md"}, "current flat panel remains wired to Couch's raw diagnostic inventory through #151 M1"},
	}
	for _, check := range checks {
		if doc := repoDocument(t, check.path...); !strings.Contains(doc, check.want) {
			t.Errorf("%s does not declare the current M1 consumer stage %q", filepath.Join(check.path...), check.want)
		}
	}
}

// The panel renderer and README consume the same key inventory. Adding a key
// to the UI therefore makes this test fail until its operator documentation
// has a home in README, instead of relying on a boundary reviewer to remember.
func TestREADMEDocumentsEveryPanelControl(t *testing.T) {
	doc := readme(t)
	for _, control := range couchtty.PanelControls() {
		if !strings.Contains(doc, control.Keys) {
			t.Errorf("README does not document panel key %q (%s)", control.Keys, control.Action)
		}
	}
}

// Every operation couch declares must appear in the README.
//
// This is the README counterpart of the atlas identifier check (M2 BR-38): the
// atlas sweep caught stale identifiers, and the same class then recurred at the
// one documented surface the sweep did not cover. Enumerating from
// couchcore.Operations() means a NEW operation is documented by existing, not by
// somebody remembering.
// agentFacing names the operations an operator never types, so the README is
// not the place for them. Each must still be documented SOMEWHERE, which the
// test below enforces -- the first version of this exemption was a bare
// `continue` with a comment pointing at atlas/couch.md, which did not document
// it either (M2 BR-39). An exemption that names another home has to check that
// home.
var agentFacing = map[string]bool{
	"prepare-start": true,
	"switch":        true,
	"attach":        true,
}

func documentsCommand(doc, command string) bool {
	return regexp.MustCompile(regexp.QuoteMeta(command) + "(?:\\s|`|$)").MatchString(doc)
}

func TestREADMEDocumentsEveryOperation(t *testing.T) {
	doc := readme(t)
	for _, op := range couchcore.Operations() {
		if agentFacing[op.Name] {
			continue // checked against the atlas instead, below
		}
		if !documentsCommand(doc, "couch "+op.Name) {
			t.Errorf("README does not document `couch %s`", op.Name)
		}
	}
}

func TestDocumentsCommandDoesNotAcceptALongerOperationPrefix(t *testing.T) {
	if documentsCommand("run `couch stop-all` here", "couch stop") {
		t.Fatal("couch stop-all must not document couch stop")
	}
	if !documentsCommand("run `couch stop TAG` here", "couch stop") {
		t.Fatal("couch stop TAG should document couch stop")
	}
}

// Flags that change what couch does with the operator's terminal must be
// documented, and so must the fact that couch CLAIMS a key from every child --
// an operator whose ctrl-space stops reaching their editor needs an explanation
// they can find.
func TestREADMEDocumentsTheOperatorFacingSurface(t *testing.T) {
	doc := readme(t)
	for _, want := range []string{
		"--no-console", // the escape hatch
		"ctrl-space",   // the key couch takes from every child
		"default: .",   // the path default, which is how "home" is chosen
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

// Every FlagOnly argument is a bypass of something; each must be documented, or
// an operator meets a refusal with no stated way past it.
func TestREADMEDocumentsEveryBypassFlag(t *testing.T) {
	doc := readme(t)
	for _, op := range couchcore.Operations() {
		for _, a := range op.Args {
			if !a.FlagOnly {
				continue
			}
			if !strings.Contains(doc, "--"+a.Name) {
				t.Errorf("README does not document the `--%s` bypass on `couch %s`", a.Name, op.Name)
			}
		}
	}
}

// The other half of the exemption: every agent-facing operation must be
// documented in the atlas, since the README deliberately skips it.
func TestAtlasDocumentsEveryAgentFacingOperation(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "atlas", "couch.md"))
	if err != nil {
		t.Fatalf("read atlas/couch.md: %v", err)
	}
	doc := string(raw)
	for name := range agentFacing {
		if !strings.Contains(doc, name) {
			t.Errorf("atlas/couch.md does not document the agent-facing `%s`", name)
		}
	}
	// And the exemption list may not name an operation that no longer exists.
	declared := map[string]bool{}
	for _, op := range couchcore.Operations() {
		declared[op.Name] = true
	}
	for name := range agentFacing {
		if !declared[name] {
			t.Errorf("agentFacing exempts %q, which couch no longer declares", name)
		}
	}
}
