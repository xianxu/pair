package couchcore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Couch owns repository arguments. Pair's standalone defaults may not become
// an implicit input to any Couch launch, preview, recovery or switch.
func TestCouchPreferencesNeverReadPairDefaults(t *testing.T) {
	for _, member := range []string{"AgentDefaultPath", "ParseAgentDefault", "ReadAgentDefault", "RepoAgentDefault", "ScopedLaunchDataDir"} {
		source := "package injected; func f() { x." + member + "() }"
		if len(pairPreferenceDependencies(t, []byte(source))) == 0 {
			t.Fatalf("dependency guard missed seeded %s call", member)
		}
	}
	for _, dir := range []string{".", "../couchcmd", "../couchtty"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, dependency := range pairPreferenceDependencies(t, source) {
				t.Errorf("%s reads Pair configuration via %s; Couch repository arguments must be explicit", path, dependency)
			}
		}
	}
}

func pairPreferenceDependencies(t *testing.T, source []byte) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "source.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var dependencies []string
	ast.Inspect(file, func(node ast.Node) bool {
		if selected, ok := node.(*ast.SelectorExpr); ok {
			switch selected.Sel.Name {
			case "AgentDefaultPath", "ParseAgentDefault", "ReadAgentDefault", "RepoAgentDefault", "ScopedLaunchDataDir":
				dependencies = append(dependencies, selected.Sel.Name)
			}
		}
		return true
	})
	return dependencies
}
