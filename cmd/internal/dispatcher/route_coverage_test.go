package dispatcher

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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

// The other half of the chain (#216 BR-14, generalised by BR-19): nvim reaches
// these commands from LUA, so the two sides of that string contract are in
// different languages and nothing type-checks between them.
//
// Scans EVERY production Lua file that builds a `pair` argv, not the one file
// the finding named — a crossing test that covers one of three boundaries is
// the same shape as the two tests that stopped on either side. The enumeration
// is `grep -n "pair_bin()\|/bin/pair'" nvim/*.lua`: init.lua, scrollback.lua and
// workbench_route.lua all construct one, in the same `{ <pairbin>, 'word', … }`
// shape.
func TestLuaBuiltPairSubcommandsAreDeclaredAndRoutable(t *testing.T) {
	// First element is the pair binary under one of its three local spellings;
	// the quoted words after it are the command name.
	argv := regexp.MustCompile(`\{\s*(?:pair|pair_bin|bin)(?:\(\))?\s*,\s*((?:'[a-z][a-z0-9-]*'\s*,\s*)+)`)
	quoted := regexp.MustCompile(`'([a-z][a-z0-9-]*)'`)

	declared := map[string]bool{}
	for _, family := range Families() {
		declared[family.Name] = true
	}
	routed := dispatchCaseNames(t)

	luaFiles, err := filepath.Glob("../../../nvim/*.lua")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, path := range luaFiles {
		if strings.HasSuffix(path, "_test.lua") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range argv.FindAllStringSubmatch(string(source), -1) {
			var words []string
			for _, w := range quoted.FindAllStringSubmatch(match[1], -1) {
				words = append(words, w[1])
			}
			// Families are one or two words ("title", "session-log append").
			// Prefer the longer reading, since "session-log" alone is not one.
			name := ""
			if len(words) >= 2 && declared[words[0]+" "+words[1]] {
				name = words[0] + " " + words[1]
			} else if len(words) >= 1 && declared[words[0]] {
				name = words[0]
			}
			if name == "" {
				t.Errorf("%s runs `pair %s`, which Families() does not declare",
					filepath.Base(path), strings.Join(words, " "))
				continue
			}
			found++
			if !routed[name] {
				t.Errorf("%s runs `pair %s`, which no router case handles", filepath.Base(path), name)
			}
		}
	}
	// The enumeration found six argv sites across three files; if a refactor
	// drops the scan to nothing, that must fail rather than pass vacuously.
	if found < 4 {
		t.Fatalf("extracted only %d Lua-built pair subcommands — the scan is broken, not the Lua", found)
	}
}
