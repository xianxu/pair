package couchcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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

// Every operation couch declares must appear in the README.
//
// This is the README counterpart of the atlas identifier check (M2 BR-38): the
// atlas sweep caught stale identifiers, and the same class then recurred at the
// one documented surface the sweep did not cover. Enumerating from
// couchcore.Operations() means a NEW operation is documented by existing, not by
// somebody remembering.
func TestREADMEDocumentsEveryOperation(t *testing.T) {
	doc := readme(t)
	for _, op := range couchcore.Operations() {
		// publish-description is agent-facing, not operator-facing: it is run
		// by a session inside its own tree, so it belongs in atlas/couch.md
		// rather than in the operator's README.
		if op.Name == "publish-description" {
			continue
		}
		if !strings.Contains(doc, "couch "+op.Name) {
			t.Errorf("README does not document `couch %s`", op.Name)
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
