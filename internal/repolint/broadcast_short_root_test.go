package repolint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Pin the shared broadcast fixture allocator, not arbitrary temp directories.
// The helper's runtime regression proves path budget and owned cleanup.
func matchBroadcastShortRoot(file *ast.File) []ast.Node {
	if file.Name.Name != "broadcast" {
		return nil
	}
	var findings []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "shortDir" || fn.Body == nil {
			continue
		}
		valid := false
		for _, statement := range fn.Body.List {
			ret, ok := statement.(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				continue
			}
			call, ok := ret.Results[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			target, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || target.Sel.Name != "ShortRoot" {
				continue
			}
			qualifier, ok := target.X.(*ast.Ident)
			valid = ok && pairImportIs(file, qualifier.Name, "github.com/xianxu/pair/cmd/internal/testpair")
		}
		if !valid {
			findings = append(findings, fn)
		}
	}
	return findings
}

func TestBroadcastShortRootGuard(t *testing.T) {
	for _, tc := range []struct {
		body string
		want int
	}{
		{`func shortDir(t *testing.T)string{return fixture.ShortRoot(t)}`, 0},
		{`func shortDir(t *testing.T)string{dir,_:=os.MkdirTemp("","cb");return dir}`, 1},
		{`func shortDir(t *testing.T)string{return t.TempDir()}`, 1},
		{`func anotherDir(t *testing.T)string{return t.TempDir()}`, 0},
	} {
		source := `package broadcast;import fixture "github.com/xianxu/pair/cmd/internal/testpair";` + tc.body
		file, err := parser.ParseFile(token.NewFileSet(), "fixture_test.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(matchBroadcastShortRoot(file)); got != tc.want {
			t.Fatalf("%s: findings=%d want=%d", tc.body, got, tc.want)
		}
	}
}
