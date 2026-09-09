package dispatcher

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// dispatchCaseNames collects every string literal that appears as a switch case
// in the two files that route commands: dispatcher.go's buffered switch, and
// cmd/pair-go/main.go, which routes the STREAMING commands (wrap, term, …)
// before they ever reach Dispatch. Both are routers; scanning only one reports
// half the surface as broken.
//
// Read from source rather than by calling Dispatch, because calling it EXECUTES
// the command: a routability check that launches `wrap` or `term` is not a test.
// Parsing is the only way to ask "is this name wired?" without running it.
func dispatchCaseNames(t *testing.T) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for _, path := range []string{"dispatcher.go", "../../pair-go/main.go"} {
		collectCaseNames(t, path, names)
	}
	if len(names) == 0 {
		t.Fatal("parsed no switch cases — the scan is broken, not the routing")
	}
	return names
}

func collectCaseNames(t *testing.T, path string, names map[string]bool) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(n ast.Node) bool {
		clause, ok := n.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, value := range clause.List {
			lit, ok := value.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			if name, err := strconv.Unquote(lit.Value); err == nil {
				names[name] = true
			}
		}
		return true
	})
}

// Every family declared implemented must actually be routable. A family whose
// case is renamed still passes `go test ./...` otherwise: the mutant fails only
// at RUNTIME, with "has no buffered route wired" (#216 BR-14).
func TestEveryImplementedFamilyIsRoutable(t *testing.T) {
	routed := dispatchCaseNames(t)
	for _, family := range Families() {
		if family.Status != "implemented" {
			continue
		}
		if !routed[family.Name] {
			t.Errorf("family %q is declared implemented but no Dispatch case routes it — "+
				"it would fail at runtime with %q", family.Name, "has no buffered route wired")
		}
	}
}

// The other half of the chain (#216 BR-14): the draft pane reaches these
// commands from LUA, so the two sides of that string contract are in different
// languages and nothing type-checks between them. This crosses the boundary by
// reading the argv the Lua actually builds and asserting Go declares and routes
// it. A rename on either side reddens here.
func TestDraftLuaSubcommandsAreDeclaredAndRoutable(t *testing.T) {
	source, err := os.ReadFile("../../../nvim/workbench_route.lua")
	if err != nil {
		t.Fatal(err)
	}
	// `return { pair_bin, 'layout', 'switch-terminal-tab', direction }` — the
	// quoted literals are the command name; the unquoted ones are variables.
	command := regexp.MustCompile(`return \{[^}]*?\}`)
	quoted := regexp.MustCompile(`'([a-z][a-z-]*)'`)
	var found []string
	for _, block := range command.FindAllString(string(source), -1) {
		var parts []string
		for _, m := range quoted.FindAllStringSubmatch(block, -1) {
			parts = append(parts, m[1])
		}
		if len(parts) >= 2 {
			found = append(found, strings.Join(parts[:2], " "))
		}
	}
	if len(found) == 0 {
		t.Fatal("found no Lua-built pair subcommands — the extraction is broken, not the Lua")
	}

	declared := map[string]bool{}
	for _, family := range Families() {
		declared[family.Name] = true
	}
	routed := dispatchCaseNames(t)
	for _, name := range found {
		if !declared[name] {
			t.Errorf("nvim/workbench_route.lua runs `pair %s`, which Families() does not declare", name)
		}
		if !routed[name] {
			t.Errorf("nvim/workbench_route.lua runs `pair %s`, which no Dispatch case routes", name)
		}
	}
}
