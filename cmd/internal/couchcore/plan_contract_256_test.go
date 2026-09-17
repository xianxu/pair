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

// The plan's Core-concepts tables are a DERIVED VIEW of the code. #256 M2 stated
// that rule three times in prose and the tables were wrong at every boundary
// anyway: a row for a function the same commit deleted, a row marking a live
// symbol `deleted`, a row marking an M2 symbol `M3`, and four production symbols
// with no row at all.
//
// A derived view is either machine-checked or it is prose. This is the check.
// It deliberately does NOT use the heavyweight ledger machinery in
// plan_contract_test.go — that pins frozen historical milestones by digest,
// whereas this has to run against a plan still being edited.
const issue256PlanPath = "workshop/plans/000256-lifecycle-transition-authority-plan.md"

type planConceptRow struct {
	name   string
	path   string
	status string
	landed string
	line   int
}

// issue256LandedMilestones are the milestones whose rows describe the tree as it
// IS. A row landing in a later milestone describes what the plan intends, so it
// is not a divergence yet. Closing M3 extends this list — and if that is
// forgotten, the check simply stops covering M3's rows rather than failing
// falsely, which is the safe direction for a guard nobody is watching.
var issue256LandedMilestones = map[string]bool{"M1": true, "M2": true}

// repoRootFrom walks up from the test's directory to the module root.
func repoRootFrom(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not find the module root above the test directory")
	return ""
}

// parseIssue256ConceptRows reads every row of the two Core-concepts tables.
// A row is any table line whose second cell is a Go path under cmd/.
func parseIssue256ConceptRows(t *testing.T, planText string) []planConceptRow {
	t.Helper()
	var rows []planConceptRow
	for i, line := range strings.Split(planText, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) < 3 {
			continue
		}
		clean := func(s string) string { return strings.TrimSpace(strings.ReplaceAll(s, "`", "")) }
		path := clean(cells[1])
		if !strings.HasPrefix(path, "cmd/") || !strings.HasSuffix(path, ".go") {
			continue
		}
		rows = append(rows, planConceptRow{
			name: clean(cells[0]), path: path, status: clean(cells[2]),
			// The milestone is the LAST cell in both tables, which differ in
			// width (Integration points carry an extra `Wraps` column).
			landed: clean(cells[len(cells)-1]), line: i + 1,
		})
	}
	return rows
}

// declaredIdentifiers collects every name a Go file declares: funcs, methods,
// types, consts, vars, and struct fields. Struct fields are included because the
// tables legitimately carry rows like `ThreadEvidence.StartOwner`.
func declaredIdentifiers(t *testing.T, path string) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			names[node.Name.Name] = true
		case *ast.TypeSpec:
			names[node.Name.Name] = true
		case *ast.ValueSpec:
			for _, ident := range node.Names {
				names[ident.Name] = true
			}
		case *ast.Field:
			for _, ident := range node.Names {
				names[ident.Name] = true
			}
		}
		return true
	})
	return names
}

// TestIssue256PlanTablesMatchTheTree runs the re-derivation the plan promises.
//
// For every row naming a Go path: a `new` or `modified` symbol must be declared
// there, and a `deleted` one must not. Rows whose name is not a plain identifier
// (a qualified method written with an ellipsis, a slash-separated pair) are
// split or skipped rather than guessed at — the check must be honest about what
// it can decide.
func TestIssue256PlanTablesMatchTheTree(t *testing.T) {
	root := repoRootFrom(t)
	planBytes, err := os.ReadFile(filepath.Join(root, issue256PlanPath))
	if err != nil {
		t.Skipf("the #256 plan has been archived; this check retires with it: %v", err)
	}
	rows := parseIssue256ConceptRows(t, string(planBytes))
	if len(rows) < 10 {
		t.Fatalf("parsed %d concept rows; the tables' shape changed and this check stopped checking", len(rows))
	}

	declared := map[string]map[string]bool{}
	checked := 0
	for _, row := range rows {
		if !issue256LandedMilestones[row.landed] {
			continue
		}
		full := filepath.Join(root, row.path)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("%s:%d: row %q names a path that does not exist: %s", issue256PlanPath, row.line, row.name, row.path)
			continue
		}
		if declared[row.path] == nil {
			declared[row.path] = declaredIdentifiers(t, full)
		}
		for _, name := range strings.Split(row.name, "/") {
			name = strings.TrimSpace(name)
			// A qualified name: check the last segment, which is the member the
			// row is really about (`ThreadEvidence.StartOwner`).
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				name = name[idx+1:]
			}
			if name == "" || strings.ContainsAny(name, " …()") {
				continue
			}
			checked++
			switch row.status {
			case "new", "modified":
				if !declared[row.path][name] {
					t.Errorf("%s:%d: %q is marked %s in %s but is not declared there",
						issue256PlanPath, row.line, name, row.status, row.path)
				}
			case "deleted":
				if declared[row.path][name] {
					t.Errorf("%s:%d: %q is marked deleted but is still declared in %s",
						issue256PlanPath, row.line, name, row.path)
				}
			}
		}
	}
	if checked < 10 {
		t.Fatalf("only %d symbols were decidable; the check has stopped checking", checked)
	}
}
