package couchcmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// Every automatic broker ingress has one policy owner. A new direct send must
// first join that owner rather than silently bypassing hub automation.
func TestAutomaticSystemSendHasOnePolicyOwner(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	sites := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "SendSystem" {
					return true
				}
				sites++
				if name != "automation.go" || fn.Name.Name != "sendAutomaticSystem" {
					t.Errorf("unowned automatic input in %s:%s", name, fn.Name.Name)
				}
				if len(call.Args) != 6 {
					t.Errorf("automatic input lacks its required commit guard in %s:%s", name, fn.Name.Name)
					return true
				}
				guard, ok := call.Args[5].(*ast.CallExpr)
				if !ok {
					t.Errorf("automatic input bypasses its commit guard in %s:%s", name, fn.Name.Name)
					return true
				}
				owner, ok := guard.Fun.(*ast.SelectorExpr)
				if !ok || owner.Sel.Name != "automationAdmissionGuard" {
					t.Errorf("automatic input uses another commit policy in %s:%s", name, fn.Name.Name)
				}
				return true
			})
		}
	}
	if sites != 1 {
		t.Fatalf("automatic broker ingress count=%d want 1", sites)
	}
}
