package termcmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// definedSomewhere reports whether a symbol is declared in any of the paths the
// table still claims are live.
func definedSomewhere(root string, paths map[string]bool, symbol string) bool {
	for path := range paths {
		names, err := declaredNames(filepath.Join(root, path))
		if err != nil {
			continue
		}
		if names[symbol] {
			return true
		}
	}
	return false
}

// resolvePlan finds a plan whether it is ACTIVE or ARCHIVED.
//
// Every guard that reads a plan file needs this, and the reason is a dated
// hazard rather than a nicety: `sdlc close` MOVES plans to
// `workshop/history/plans/` (39 are there already), so a guard that names the
// active path hard-fails `make test` for the whole repo at the next close --
// including the close of the very issue whose plan it reads. This repo already
// solved it once, in `couchtty/core_concepts_contract_test.go`; re-implementing
// the guard without re-implementing the resolution is how the case got dropped
// (BR-61).
func resolvePlan(root, name string) (string, error) {
	active := filepath.Join(root, "workshop", "plans", name)
	if _, err := os.Stat(active); err == nil {
		return active, nil
	}
	var archived string
	_ = filepath.WalkDir(filepath.Join(root, "workshop", "history"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == name {
			archived = path
		}
		return nil
	})
	if archived == "" {
		return "", fmt.Errorf("plan %s is in neither workshop/plans nor workshop/history", name)
	}
	return archived, nil
}

// goIdentifier is what a Go symbol can look like; anything else in a backtick is
// prose, an escape sequence, or a zellij action name.
var goIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// declaredNames is every top-level identifier a Go file DECLARES: consts, vars,
// types, funcs and methods. Not what it references.
func declaredNames(path string) (map[string]bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			names[d.Name.Name] = true
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch sp := spec.(type) {
				case *ast.TypeSpec:
					names[sp.Name.Name] = true
					// Struct FIELDS too: the integration table names entities
					// like `stripOwed`, which is a field on terminalMux and is
					// exactly as much a declared thing as a func.
					if st, ok := sp.Type.(*ast.StructType); ok && st.Fields != nil {
						for _, f := range st.Fields.List {
							for _, n := range f.Names {
								names[n.Name] = true
							}
						}
					}
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						names[n.Name] = true
					}
				}
			}
		}
	}
	return names, nil
}

// A `planned — Mx` ROW MUST NOT SURVIVE A TICKED Mx.
//
// This is the sixth finding in the `plan-table-drift` family, so it is fixed as
// a rule rather than by editing the rows again. The mechanism that let the table
// rot is specific and worth naming: couchtty's Core-concepts contract SKIPS any
// row whose status contains "planned" (core_concepts_contract_test.go), which is
// correct while the milestone is ahead of you and silently wrong the moment it
// lands. So a table could carry `planned — M3` through M3's close with nothing
// objecting, and this one did.
//
// The check is deliberately cheap and text-level: it needs no package internals,
// which is why it can live next to the code the table describes rather than
// forcing couchtty's machinery to be copied (BR-52's lesson, applied to a test).
func TestNoPlannedRowSurvivesItsTickedMilestone(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	// BOTH homes, for the same reason resolvePlan reads both: `sdlc close`
	// ARCHIVES an issue to workshop/history/issues, and a guard that reads only
	// the active directory goes silent at exactly the moment the work ships.
	// Closing would then HIDE this failure rather than fix it -- and it did:
	// the Critical this test catches was found by a reviewer, not by the guard,
	// because the guard's own coverage ended at the close.
	var issues []string
	for _, dir := range [][]string{
		{"workshop", "issues"},
		{"workshop", "history", "issues"},
	} {
		found, err := filepath.Glob(filepath.Join(append([]string{root}, append(dir, "*.md")...)...))
		if err != nil {
			t.Fatal(err)
		}
		issues = append(issues, found...)
	}
	if len(issues) == 0 {
		t.Fatal("no issue files found; the guard would pass vacuously")
	}

	ticked := regexp.MustCompile(`(?m)^\s*-\s*\[x\]\s*(M\d+[a-z]?)\b`)
	checked := 0
	for _, issue := range issues {
		raw, err := os.ReadFile(issue)
		if err != nil {
			t.Fatal(err)
		}
		done := ticked.FindAllStringSubmatch(string(raw), -1)
		if len(done) == 0 {
			continue
		}
		// The plan sits beside the issue, same stem plus -plan.
		stem := strings.TrimSuffix(filepath.Base(issue), ".md")
		plan, err := resolvePlan(root, stem+"-plan.md")
		if err != nil {
			continue // simple work has no durable plan; that is allowed
		}
		planRaw, err := os.ReadFile(plan)
		if err != nil {
			continue
		}
		checked++
		for _, m := range done {
			milestone := m[1]
			stale := regexp.MustCompile(`planned\s*—\s*` + milestone + `\b`)
			for i, line := range strings.Split(string(planRaw), "\n") {
				if !strings.HasPrefix(strings.TrimSpace(line), "|") || !stale.MatchString(line) {
					continue
				}
				t.Errorf("%s:%d still says `planned — %s` after %s was ticked in %s:\n  %s",
					plan, i+1, milestone, milestone, filepath.Base(issue), strings.TrimSpace(line))
			}
		}
	}
	if checked == 0 {
		t.Fatal("no issue with a ticked milestone had a durable plan; the guard proved nothing")
	}
}

// EVERY Core-concepts ROW OF THIS PLAN NAMES A SYMBOL THAT IS ACTUALLY THERE.
//
// Until now nothing read a single one of M3's rows. couchtty's contract test
// (`core_concepts_contract_test.go`) does check declared paths against declared
// symbols, but it filters #199's table down to `cmd/internal/couchtty/` — so
// the twelve rows describing `termcmd`, `hostty` and `ptychild` were unchecked,
// and one of them named `ResetSGR` in the wrong file (BR-58).
//
// This is deliberately NOT a copy of that machinery, which is 300 lines and
// package-private. It is the one claim the table makes that nothing else
// verifies: a `new`/`modified` row's symbols exist at its stated path, and a
// `deleted` row's symbols do not. Duplicating a harness to reuse a check is the
// mistake BR-52 was about.
func TestEveryCoreConceptRowNamesASymbolThatExists(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	plan, err := resolvePlan(root, "000199-pair-term-own-the-right-pane-tab-bar-plan.md")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(plan)
	if err != nil {
		t.Fatal(err)
	}

	backticked := regexp.MustCompile("`([^`]+)`")
	checked := 0
	// A row's declared symbols, so the BULLETS under the table can be checked
	// against the same claim the row made. The bullets say the same kind of
	// thing about the same symbols -- one of them went on naming
	// `Reservation.Reserve()` after it was deleted, and the pipe-rows-only guard
	// could not see it (BR-68).
	declaredAt := map[string]string{}
	deletedSymbols := map[string]bool{}
	livePaths := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) < 3 {
			continue
		}
		names, path, status := cells[0], strings.TrimSpace(cells[1]), strings.ToLower(cells[2])
		if !strings.HasSuffix(strings.Trim(path, "`"), ".go") {
			continue // not a Go entity row (layouts, headers, separators)
		}
		if strings.Contains(status, "planned") {
			continue // ahead of us; TestNoPlannedRowSurvivesItsTickedMilestone owns the stale case
		}
		file := filepath.Join(root, strings.Trim(path, "`"))
		declared, err := declaredNames(file)
		if err != nil {
			// A `deleted` row may name a file that is gone; that is the row
			// being right, not wrong.
			if strings.Contains(status, "deleted") {
				checked++
				continue
			}
			t.Errorf("row %q declares path %s, which does not exist", strings.TrimSpace(names), path)
			continue
		}
		for _, m := range backticked.FindAllStringSubmatch(names, -1) {
			for _, qualified := range strings.Split(m[1], " / ") {
				symbol := strings.TrimSpace(qualified)
				symbol = symbol[strings.LastIndex(symbol, ".")+1:]
				// Only actual Go identifiers. These tables also backtick things
				// that are not symbols at all -- `rename-pane` is a zellij
				// action, `\x1b[r` an escape -- and asserting those exist in a
				// .go file is the guard misreading its own input.
				if !goIdentifier.MatchString(symbol) {
					continue
				}
				checked++
				// DEFINED here, not merely MENTIONED here. A substring match
				// passes on a USE, which is how the wrong path survived: the row
				// declared `ResetSGR` at reserve.go, where ReserveAndPaint calls
				// it, while it is defined in control.go (BR-58). A table that
				// says where a thing LIVES must be checked against where it is
				// declared.
				present := declared[symbol]
				deleted := strings.Contains(status, "deleted")
				if deleted {
					deletedSymbols[symbol] = true
				} else {
					declaredAt[symbol] = path
					livePaths[strings.Trim(path, "`")] = true
				}
				if deleted && present {
					t.Errorf("row %q is marked deleted, but %s still exists in %s", strings.TrimSpace(names), symbol, path)
				}
				if !deleted && !present {
					t.Errorf("row %q declares %s at %s, where it is absent", strings.TrimSpace(names), symbol, path)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no Core-concepts row was checked; the guard proved nothing")
	}

	// THE PROSE UNDER THE TABLE MAKES THE SAME CLAIMS, so it is read the same
	// way. A bullet naming a symbol the table marked DELETED is the table
	// describing code that is gone -- which is what the guard is for, and what
	// reading only pipe rows could not catch.
	inSection := false
	for i, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## Core concepts"):
			inSection = true
			continue
		case inSection && strings.HasPrefix(trimmed, "## "):
			inSection = false
		}
		if !inSection || strings.HasPrefix(trimmed, "|") {
			continue
		}
		for _, m := range backticked.FindAllStringSubmatch(line, -1) {
			for _, qualified := range strings.Split(m[1], " / ") {
				symbol := strings.TrimSpace(qualified)
				symbol = strings.TrimSuffix(symbol, "()")
				symbol = symbol[strings.LastIndex(symbol, ".")+1:]
				if !goIdentifier.MatchString(symbol) || !deletedSymbols[symbol] {
					continue
				}
				// A row may mark a symbol deleted from ONE package while a
				// same-named one lives in another -- `ChildRows` and `Release`
				// left couchtty and are hostty.Reservation methods now. So
				// "deleted" here means "declared dead and defined in none of the
				// paths this table declares".
				if definedSomewhere(root, livePaths, symbol) {
					continue
				}
				t.Errorf("%s:%d — the prose still names %s, which this table marks deleted",
					filepath.Base(plan), i+1, symbol)
			}
		}
	}
}
