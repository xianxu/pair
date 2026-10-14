package launcher

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func psExecViolations(root string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, arg := range call.Args {
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err == nil && (value == "ps" || value == "/bin/ps" || value == "/usr/bin/ps") {
					violations = append(violations, fset.Position(call.Pos()).String())
				}
			}
			return true
		})
		return nil
	})
	return violations, err
}

func TestProductionDoesNotExecPS(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	violations, err := psExecViolations(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("production ps invocations must use procutil: %v", violations)
	}
}

func TestPSGuardDetectsDirectContextAbsoluteAndWrappedCalls(t *testing.T) {
	dir := t.TempDir()
	raw := `package fixture
func bad() { exec.Command("ps"); exec.CommandContext(ctx,"/bin/ps"); command(ctx,"ps"); runner.Run("/usr/bin/ps") }
`
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "allowed_test.go"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := psExecViolations(dir)
	if err != nil || len(got) != 4 {
		t.Fatalf("violations=%v err=%v", got, err)
	}
}
