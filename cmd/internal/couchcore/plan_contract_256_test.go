package couchcore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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

// issue256OwnedFiles are the files #256 CREATED. Every package-scope symbol they
// declare is this issue's to account for, so the tree->rows direction has a
// domain that needs no git.
var issue256OwnedFiles = []string{
	"cmd/internal/couchcore/sessionevidence.go",
	"cmd/internal/couchcore/lifecycledebris.go",
}

// issue256ConceptSymbols are the symbols the tables are expected to carry. A
// helper that exists only to keep a function readable is detail, not a concept,
// and the plan says so; listing the concepts here is what stops this check from
// demanding a row for every unexported closure.
var issue256ConceptSymbols = map[string]bool{
	"SessionState": true, "SessionObservation": true, "ProjectSessionPresence": true,
	"SessionPresenceResolver": true, "indexSessionsByName": true, "uniquelyClaimed": true,
	"clearLifecycleDebris": true, "ErrThreadRolledBack": true,
}

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
var issue256LandedMilestones = map[string]bool{"M1": true, "M2": true, "M3": true}

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
	// THE OTHER DIRECTION. Rows->tree catches a row that lies; tree->rows
	// catches a symbol with no row at all, which is the half of C2 that let four
	// M2 symbols go unlisted and then let a fifth (`PreparedAgentSwitch.state`)
	// go unlisted in the very commit that added the check.
	//
	// The domain is the production symbols this milestone's own files declare at
	// package scope. It is deliberately narrow: an exhaustive git-diff sweep
	// needs git, which is unavailable in the pinned scratch trees the boundary
	// reviewer builds, and a check that silently skips there is a check that
	// stops checking exactly where it is being audited.
	listed := map[string]bool{}
	for _, row := range rows {
		for _, name := range strings.Split(row.name, "/") {
			name = strings.TrimSpace(name)
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				name = name[idx+1:]
			}
			listed[name] = true
		}
	}
	for _, owned := range issue256OwnedFiles {
		for name := range declaredIdentifiers(t, filepath.Join(root, owned)) {
			if !issue256ConceptSymbols[name] || listed[name] {
				continue
			}
			t.Errorf("%s declares %q, which #256 owns, and neither Core-concepts table has a row for it",
				owned, name)
		}
	}

	if checked < 10 {
		t.Fatalf("only %d symbols were decidable; the check has stopped checking", checked)
	}
}

// The plan's Core-concepts PROSE is held to the same rule as its tables
// (#256 close, BR-42): family `plan-code-divergence`, sixth occurrence, and the
// first five were fixed one sentence at a time.
//
// TestIssue256PlanTablesMatchTheTree machine-checks the table ROWS, so the
// divergence moved into the bullets beneath them, which nothing checked --
// seven stale claims in one section at the close. A bullet naming a consumer, a
// branch position or a file:line coordinate is a hand-maintained restatement of
// the model, a deferred consumer (ARCH-PURPOSE). So in this section:
//
//   - no `file.go:NNN` coordinates and no `:NNN` back-references -- they rot the
//     moment the file above them grows, and every one found here had;
//   - no branch positions ("row 4", "Rows 7-9") -- the plan deleted the branch
//     table they indexed, because it kept becoming instructions to undo a fix;
//   - every test the prose CITES must exist, because citing the test that
//     enumerates a fact is the sanctioned alternative, and a citation that has
//     rotted is the same restatement one level up.
//
// Consumer claims themselves cannot be checked mechanically; the rule for them is
// to cite the enumerating test instead of listing consumers, which the third
// check then keeps honest.
func TestIssue256CoreConceptsProseCitesTestsNotCoordinates(t *testing.T) {
	root := repoRootFrom(t)
	planBytes, err := os.ReadFile(filepath.Join(root, issue256PlanPath))
	if err != nil {
		t.Skipf("the #256 plan has been archived; this check retires with it: %v", err)
	}
	plan := string(planBytes)
	start := strings.Index(plan, "\n## Core concepts")
	end := strings.Index(plan, "\n## Milestones")
	if start < 0 || end < start {
		t.Fatal("could not find the Core concepts section; the plan's shape changed and this check stopped checking")
	}
	section := plan[start:end]
	firstLine := strings.Count(plan[:start], "\n") + 2

	coordinate := regexp.MustCompile("[A-Za-z0-9_/]+\\.go:\\d+|`:\\d+`")
	branchPosition := regexp.MustCompile(`(?i)\brows? \d`)
	// A claim that something is NOT covered cites nothing, so the citation
	// check below cannot see it rot -- and it rots faster than a positive one,
	// because the fix that adds the coverage never mentions the sentence. The
	// close's BR-44 was this exact shape, in the commit that added the bound.
	negativeCoverage := regexp.MustCompile(`(?i)nothing (yet )?(bounds|pins|covers|tests)|is not (yet )?(bounded|pinned|covered|tested)|side is not\b|known gap|no test (bounds|pins|covers)`)
	for i, line := range strings.Split(section, "\n") {
		// Table rows carry paths, not coordinates, and are checked elsewhere.
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		if m := coordinate.FindString(line); m != "" {
			t.Errorf("%s:%d: Core concepts cites the coordinate %q; cite the test that enumerates the fact instead",
				issue256PlanPath, firstLine+i, m)
		}
		if m := branchPosition.FindString(line); m != "" {
			t.Errorf("%s:%d: Core concepts indexes a branch position (%q) in a table the plan deleted",
				issue256PlanPath, firstLine+i, m)
		}
		if m := negativeCoverage.FindString(line); m != "" {
			t.Errorf("%s:%d: Core concepts asserts an ABSENCE of coverage (%q); name the test that would "+
				"cover it, or delete the claim -- nothing keeps a negative claim true",
				issue256PlanPath, firstLine+i, m)
		}
	}

	declared := map[string]bool{}
	for _, pkg := range []string{"couchcore", "couchtty", "couchcmd"} {
		dir := filepath.Join(root, "cmd", "internal", pkg)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			for name := range declaredIdentifiers(t, filepath.Join(dir, entry.Name())) {
				if strings.HasPrefix(name, "Test") {
					declared[name] = true
				}
			}
		}
	}
	cited := regexp.MustCompile("`(Test[A-Za-z0-9_]+)`")
	for _, match := range cited.FindAllStringSubmatch(section, -1) {
		if !declared[match[1]] {
			t.Errorf("Core concepts cites %s, which no couch test declares; a rotted citation is the restatement one level up", match[1])
		}
	}
}
