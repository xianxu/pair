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

// The door this guard closes: `updateExistingThread` took an arbitrary mutation
// callback, so any caller could write any lifecycle field and the store's CAS,
// immutable-field checks and final validation would all pass it. Those protect a
// COHERENT record; none of them requires an AUTHORIZED transition, which is
// #256's whole subject.
//
// TWO rules, because either alone leaves the door open:
//
//  1. No exported ThreadStore method takes a record mutator. Unexporting
//     updateExistingThread is what stops other packages, and this is the rule
//     rather than the name, so re-exporting it under any spelling fails here.
//  2. Inside this package, the mutator door is reachable only from a
//     *ThreadStore method. That is derived, not a file list: the three leaking
//     production sites were all in package `couchcore` and two of them in the
//     same directory as the store, so a package-scoped or sibling-package guard
//     (terminal/no_destination_contract_test.go's shape) could not see them.
//     The authority boundary is the RECEIVER, so that is what is checked.
//
// Test files are out of scope on purpose: a test builds fixtures, and the
// fixture it needs is often a record production can no longer write -- which is
// exactly what the store's guards must be exercised against.
func TestArbitraryLifecycleMutationHasNoDoor(t *testing.T) {
	root := repoRootFrom(t)
	dir := filepath.Join(root, "cmd", "internal", "couchcore")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	mutatorParam := func(field *ast.Field) bool {
		fn, ok := field.Type.(*ast.FuncType)
		if !ok || fn.Params == nil || len(fn.Params.List) != 1 {
			return false
		}
		star, ok := fn.Params.List[0].Type.(*ast.StarExpr)
		if !ok {
			return false
		}
		ident, ok := star.X.(*ast.Ident)
		return ok && ident.Name == "ThreadRecord"
	}
	receiverIsThreadStore := func(fn *ast.FuncDecl) bool {
		if fn.Recv == nil || len(fn.Recv.List) != 1 {
			return false
		}
		star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			return false
		}
		ident, ok := star.X.(*ast.Ident)
		return ok && ident.Name == "ThreadStore"
	}

	checkedFiles, doors := 0, 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		checkedFiles++
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			store := receiverIsThreadStore(fn)
			if store && fn.Name.IsExported() && fn.Type.Params != nil {
				for _, param := range fn.Type.Params.List {
					if mutatorParam(param) {
						t.Errorf("%s: exported ThreadStore.%s takes a record mutator; "+
							"an exported arbitrary-mutation door is the thing this issue closes",
							name, fn.Name.Name)
					}
				}
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "updateExistingThread" {
					return true
				}
				doors++
				if !store {
					t.Errorf("%s: %s passes an arbitrary mutation to the store; "+
						"lifecycle changes go through a NAMED transition, whose name is where "+
						"the caller's authority is recorded", name, fn.Name.Name)
				}
				return true
			})
		}
	}
	if checkedFiles < 20 {
		t.Fatalf("walked only %d production files; the directory layout changed and this guard stopped guarding", checkedFiles)
	}
	if doors == 0 {
		t.Fatal("no call to the mutator door was found at all; it was renamed and this guard now proves nothing")
	}
}
