package couchcmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
