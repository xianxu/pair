package couchcmd

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

type scheduleWriterGuardResult struct {
	violations           []string
	owners, publications int
}

// This source dependency guard checks references, including method values and
// global initializers. StateOwner cannot be named outside its implementation
// and the scheduler owner. Within scheduler source, Update belongs to that owner.
func inspectScheduleWriters(path string, file *ast.File) scheduleWriterGuardResult {
	result := scheduleWriterGuardResult{}
	for _, decl := range file.Decls {
		receiver, name, ownerField := "", "", false
		fn, isFunc := decl.(*ast.FuncDecl)
		if isFunc {
			name = fn.Name.Name
			if fn.Recv != nil {
				typ := fn.Recv.List[0].Type
				if p, ok := typ.(*ast.StarExpr); ok {
					typ = p.X
				}
				if id, ok := typ.(*ast.Ident); ok {
					receiver = id.Name
				}
			}
		} else if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
			for _, spec := range gen.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok && typ.Name.Name == "scheduleOwner" {
					_, ownerField = typ.Type.(*ast.StructType)
				}
			}
		}
		reject := func(reason string) {
			result.violations = append(result.violations, fmt.Sprintf("%s: %s.%s: %s", path, receiver, name, reason))
		}
		if receiver == "StateStore" && (name == "Update" || name == "WithAuthority") {
			reject("legacy StateStore writer authority")
		}
		if receiver == "automaticScheduler" || receiver == "manualTaskRunner" || name == "runScheduleManual" || name == "ensureUncertainRuns" || name == "persistTaskHistoryFailure" {
			reject("legacy scheduler writer")
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				switch id.Name {
				case "StateOwner":
					implementation := file.Name.Name == "couchschedule" && filepath.Base(path) == "owner.go"
					if !implementation && !(file.Name.Name == "couchcmd" && (receiver == "scheduleOwner" || ownerField)) {
						reject("StateOwner capability outside owner scope")
					}
				case "publishSchedule":
					if file.Name.Name == "couchschedule" && !(receiver == "StateStore" && name == "updateHeld") && !(isFunc && fn.Name == id && name == "publishSchedule") {
						reject("raw ledger publication outside store implementation")
					}
				case "lockSchedule":
					if file.Name.Name == "couchschedule" && !(receiver == "StateStore" && name == "Own") && !(isFunc && fn.Name == id && name == "lockSchedule") {
						reject("namespace lock outside lifetime owner acquisition")
					}
				}
			}
			sel, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "PublishOccurrence", "FinalizeNotice":
				if file.Name.Name != "operationalrun" {
					result.publications++
					if receiver != "scheduleOwner" {
						reject("history mutation outside owner")
					}
				}
			case "Update":
				if file.Name.Name == "couchcmd" && strings.HasPrefix(filepath.Base(path), "schedule_") && receiver != "scheduleOwner" {
					reject("Update capability outside scheduler owner")
				}
			case "updateHeld":
				if receiver != "StateOwner" {
					reject("ledger mutation outside StateOwner")
				}
			case "Own":
				result.owners++
				if receiver != "scheduleOwner" {
					reject("writer acquisition outside scheduler owner")
				}
			}
			return true
		})
	}
	return result
}

func TestScheduleProductionHasOneWriter(t *testing.T) {
	checked, owners, publications := 0, 0, 0
	err := filepath.WalkDir(filepath.Join("../../..", "cmd"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		checked++
		result := inspectScheduleWriters(path, file)
		owners += result.owners
		publications += result.publications
		for _, violation := range result.violations {
			t.Error(violation)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 100 || owners == 0 || publications == 0 {
		t.Fatalf("writer guard lost anchors: files=%d owner acquisitions=%d publications=%d", checked, owners, publications)
	}
}

func TestScheduleWriterGuardRejectsBypasses(t *testing.T) {
	cases := map[string]string{
		"qualified alias":        `package couchcmd; type Leak = couchschedule.StateOwner`,
		"unqualified alias":      `package couchcmd; type Leak = StateOwner`,
		"escaped field alias":    `package couchcmd; func leak(o *scheduleOwner) { local:=o.renamed; f:=local.Update; _=f }`,
		"global method value":    `package couchcmd; var update=other.Update`,
		"global history value":   `package couchcmd; var publish=store.PublishOccurrence`,
		"global raw publication": `package couchschedule; var publish=publishSchedule`,
		"global raw lock":        `package couchschedule; var lock=lockSchedule`,
		"global held mutation":   `package couchschedule; var update=store.updateHeld`,
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "schedule_leak.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			if result := inspectScheduleWriters("schedule_leak.go", file); len(result.violations) == 0 {
				t.Fatal("writer guard accepted bypass")
			}
		})
	}
}

func TestScheduleWriterGuardAllowsUnrelatedUpdate(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "component.go", `package couchcmd; func refresh(c *component) { update:=c.Update; update() }`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result := inspectScheduleWriters("component.go", file); len(result.violations) != 0 {
		t.Fatal(result.violations)
	}
}

const scheduleConceptMapStart = "<!-- scheduler-concepts:active:start -->"
const scheduleConceptMapEnd = "<!-- scheduler-concepts:active:end -->"

// Concept anchors name production declarations, not words appearing in comments
// or declarations that only exist in compatibility tests.
func inspectScheduleConceptMap(root, markdown string) (int, []string) {
	start, end := strings.Index(markdown, scheduleConceptMapStart), strings.Index(markdown, scheduleConceptMapEnd)
	if start < 0 || end < start {
		return 0, []string{"missing active scheduler concept map"}
	}
	anchor := regexp.MustCompile("`(cmd/[^`#]+\\.go)#([A-Za-z_][A-Za-z_0-9.]*)`")
	symbols := map[string]map[string]bool{}
	rows := 0
	var problems []string
	for _, line := range strings.Split(markdown[start+len(scheduleConceptMapStart):end], "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| Concept |") || strings.HasPrefix(line, "| ---") {
			continue
		}
		rows++
		refs := anchor.FindAllStringSubmatch(line, -1)
		if len(refs) == 0 {
			problems = append(problems, "concept row has no production declaration anchor: "+line)
		}
		for _, ref := range refs {
			path, name := ref[1], ref[2]
			if strings.Contains(path, "..") || strings.HasSuffix(path, "_test.go") {
				problems = append(problems, "non-production concept anchor: "+path)
				continue
			}
			if symbols[path] == nil {
				source, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, path), nil, 0)
				if err != nil {
					problems = append(problems, "missing concept source: "+path+": "+err.Error())
					continue
				}
				symbols[path] = map[string]bool{}
				for _, decl := range source.Decls {
					switch item := decl.(type) {
					case *ast.FuncDecl:
						key := item.Name.Name
						if item.Recv != nil {
							typ := item.Recv.List[0].Type
							if pointer, ok := typ.(*ast.StarExpr); ok {
								typ = pointer.X
							}
							if id, ok := typ.(*ast.Ident); ok {
								key = id.Name + "." + key
							}
						}
						symbols[path][key] = true
					case *ast.GenDecl:
						for _, spec := range item.Specs {
							if typ, ok := spec.(*ast.TypeSpec); ok {
								symbols[path][typ.Name.Name] = true
							}
						}
					}
				}
			}
			if !symbols[path][name] {
				problems = append(problems, "missing production declaration: "+path+"#"+name)
			}
		}
	}
	return rows, problems
}

func TestScheduleActiveConceptMapMatchesProduction(t *testing.T) {
	root := filepath.Clean("../../..")
	atlas, err := os.ReadFile(filepath.Join(root, "atlas/couch.md"))
	if err != nil {
		t.Fatal(err)
	}
	validate := func(body []byte) {
		t.Helper()
		rows, problems := inspectScheduleConceptMap(root, string(body))
		t.Logf("validated %d active concept rows against production declarations", rows)
		for _, problem := range problems {
			t.Error(problem)
		}
		if rows < 9 {
			t.Fatalf("active replacement map has %d rows; all nine original concepts must be reconciled", rows)
		}
	}
	validate(atlas)
	// The durable atlas outlives SDLC's archived issue/plan. Check the plan while
	// active, without reading archived plans or tying tests to a closed issue.
	plan, err := os.ReadFile(filepath.Join(root, "workshop/plans/000737-couch-scheduler-plan.md"))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	validate(plan)
	marked := func(body []byte) string {
		source := string(body)
		start, end := strings.Index(source, scheduleConceptMapStart), strings.Index(source, scheduleConceptMapEnd)
		if start < 0 || end < start {
			return ""
		}
		return source[start : end+len(scheduleConceptMapEnd)]
	}
	if marked(atlas) != marked(plan) {
		t.Error("active plan implementation snapshot differs from canonical atlas map")
	}
}

func TestScheduleConceptMapRejectsInvalidAnchors(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "cmd/fixture")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "owner.go"), []byte("package fixture\ntype StateOwner struct{}\nfunc (*StateOwner) Update() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "store.go"), []byte("package fixture\ntype StateStore struct{}\n// StateOwner is declared elsewhere.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "owner_test.go"), []byte("package fixture\ntype TestOnlyOwner struct{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	wrap := func(anchor string) string {
		return scheduleConceptMapStart + "\n| Owner | `" + anchor + "` | lifetime authority |\n" + scheduleConceptMapEnd
	}
	for _, target := range []string{"cmd/fixture/missing.go#StateOwner", "cmd/fixture/owner_test.go#TestOnlyOwner", "cmd/fixture/store.go#StateOwner", "cmd/fixture/owner.go#Other.Update"} {
		t.Run(target, func(t *testing.T) {
			if _, problems := inspectScheduleConceptMap(root, wrap(target)); len(problems) == 0 {
				t.Fatal("accepted invalid concept declaration")
			}
		})
	}
	if _, problems := inspectScheduleConceptMap(root, wrap("cmd/fixture/owner.go#StateOwner")+"\n"); len(problems) != 0 {
		t.Fatal(problems)
	}
	if _, problems := inspectScheduleConceptMap(root, wrap("cmd/fixture/owner.go#StateOwner.Update")); len(problems) != 0 {
		t.Fatal(problems)
	}
}
