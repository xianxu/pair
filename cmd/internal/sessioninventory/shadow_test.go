package sessioninventory_test

import (
	"bufio"
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

func TestNoWholeInventoryShadowInInteractiveConsumers(t *testing.T) {
	t.Parallel()
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	allowed := map[string]bool{
		"cmd/internal/sessioninventory/runcli.go:runCLIOptionsWithRenderers:InventoryWithRuntime":     true,
		"cmd/internal/sessioninventory/pair_inventory.go:RecoverPairBindings:NativeEventsWithRuntime": true,
		// A v1 launch can exist only across an in-place binary upgrade. Keep its
		// compatibility adapter explicit; all newly-created launches are v2.
	}
	seen := map[string]bool{}
	var violations []string
	err := filepath.WalkDir(filepath.Join(repoRoot, "cmd"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := calledName(call.Fun)
				if name != "InventoryWithRuntime" && name != "NativeEventsWithRuntime" {
					return true
				}
				key := filepath.ToSlash(relative) + ":" + function.Name.Name + ":" + name
				if allowed[key] {
					seen[key] = true
				} else {
					violations = append(violations, key)
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for key := range allowed {
		if !seen[key] {
			violations = append(violations, "stale allowlist entry "+key)
		}
	}
	activity := regexp.MustCompile(`session-inventory[^\n]*--activity`)
	for _, dir := range []string{"nvim", "bin"} {
		err := filepath.WalkDir(filepath.Join(repoRoot, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if extension := filepath.Ext(path); extension != ".lua" && extension != ".sh" {
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for line := 1; scanner.Scan(); line++ {
				if activity.MatchString(scanner.Text()) {
					relative, _ := filepath.Rel(repoRoot, path)
					violations = append(violations, fmt.Sprintf("%s:%d direct activity subprocess", filepath.ToSlash(relative), line))
				}
			}
			return scanner.Err()
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	if len(violations) != 0 {
		t.Fatalf("whole-inventory shadow paths: %s", strings.Join(violations, "; "))
	}
}

func calledName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	default:
		return ""
	}
}
