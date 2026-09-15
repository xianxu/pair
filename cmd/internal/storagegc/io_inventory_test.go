package storagegc

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

// This is a reference/call-edge guard, not a claim of whole-program coverage.
// The inventory records behavioral tests separately from artifact membership.
func TestManagedIOInventoryAnchors(t *testing.T) {
	root := filepath.Clean("../../..")
	body, err := os.ReadFile(filepath.Join(root, "atlas/storage-retention-io.md"))
	if err != nil {
		t.Fatal(err)
	}
	anchor := regexp.MustCompile("`((?:cmd|nvim)/[^`#]+)#([^`]+)`")
	refs := anchor.FindAllStringSubmatch(string(body), -1)
	if len(refs) == 0 {
		t.Fatal("inventory has no checked source/test anchors")
	}
	functions := map[string]map[string]*ast.FuncDecl{}
	files := map[string]string{}
	for _, ref := range refs {
		path, name := ref[1], ref[2]
		if filepath.IsAbs(path) || strings.Contains(path, "..") {
			t.Fatalf("unsafe reference %q", path)
		}
		source, ok := files[path]
		if !ok {
			raw, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				t.Fatalf("inventory %s: %v", path, err)
			}
			source = string(raw)
			files[path] = source
		}
		if filepath.Ext(path) != ".go" {
			if !strings.Contains(source, name) {
				t.Errorf("inventory anchor %s#%s missing", path, name)
			}
			continue
		}
		if functions[path] == nil {
			file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
			if err != nil {
				t.Fatal(err)
			}
			functions[path] = map[string]*ast.FuncDecl{}
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					functions[path][fn.Name.Name] = fn
				}
			}
		}
		if functions[path][name] == nil {
			t.Errorf("inventory function %s#%s missing", path, name)
		}
		if strings.HasSuffix(path, "_test.go") && !strings.HasPrefix(name, "Test") {
			t.Errorf("inventory evidence is not a named test: %s#%s", path, name)
		}
	}
	// Each inventory row must name an entry source, artifact, protection, clock
	// rule, and an actual test reference; an empty generic policy row cannot pass.
	rows := 0
	for _, line := range strings.Split(string(body), "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| Entrypoint") || strings.HasPrefix(line, "| ---") {
			continue
		}
		rows++
		columns := strings.Split(line, "|")
		if len(columns) != 7 {
			t.Errorf("inventory row needs five populated columns: %s", line)
			continue
		}
		for _, column := range columns[1:6] {
			if strings.TrimSpace(column) == "" {
				t.Errorf("empty inventory column: %s", line)
			}
		}
		if !strings.Contains(columns[5], "_test.go#Test") && !strings.Contains(columns[5], "_test.lua#") {
			t.Errorf("row lacks checked behavioral evidence: %s", line)
		}
	}
	if rows == 0 {
		t.Fatal("inventory has no managed entrypoint rows")
	}
	// Declared direct guard calls are checked in the production function AST,
	// so leaving a function name in place after removing its guard is caught.
	edge := regexp.MustCompile(`<!-- retention-call: ([^# ]+)#([^ ]+) -> ([^ ]+) -->`)
	for _, ref := range edge.FindAllStringSubmatch(string(body), -1) {
		fn := functions[ref[1]][ref[2]]
		if fn == nil {
			t.Errorf("call edge needs a documented source anchor: %s", ref[0])
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch callee := call.Fun.(type) {
			case *ast.Ident:
				found = found || callee.Name == ref[3]
			case *ast.SelectorExpr:
				found = found || callee.Sel.Name == ref[3]
			}
			return true
		})
		if !found {
			t.Errorf("managed IO guard call disappeared: %s", ref[0])
		}
	}
	if len(edge.FindAllString(string(body), -1)) == 0 {
		t.Fatal("inventory has no checked call edges")
	}
}
