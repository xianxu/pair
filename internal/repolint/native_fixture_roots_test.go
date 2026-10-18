package repolint

import (
	"errors"
	"fmt"
	"github.com/xianxu/ariadne/pkg/golint"
	"github.com/xianxu/pair/internal/reporoot"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const nativeFixtureDir = "cmd/internal/couchcore/"

var nativeFixtureFiles = []string{"recovery_conformance_live_test.go", "park_lifecycle_live_test.go", "recovery_live_environment_test.go"}
var nativeFixtureRoots = []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "ZELLIJ_SOCKET_DIR", "ZELLIJ_CONFIG_DIR"}

func nativeRootSources() map[string][]byte {
	reads := ""
	for _, root := range nativeFixtureRoots {
		reads += fmt.Sprintf("os.Getenv(%q),", root)
	}
	return map[string][]byte{
		nativeFixtureFiles[0]: []byte("package fixture\nfunc TestRecoveryRealHelperAndSessionConformanceLive(){ setupRecoveryLiveEnvironment(t) }"),
		nativeFixtureFiles[1]: []byte("package fixture\nfunc newRealParkConformance(){ setupRecoveryLiveEnvironment(t) }\nfunc startParkLifecycleHelper(){ cmd.Env = recoveryLiveChildEnvironment(t) }"),
		nativeFixtureFiles[2]: []byte("package fixture\nfunc recoveryLiveChildEnvironment(){ return []string{" + reads + "} }"),
	}
}

func TestNativeFixtureRootRule(t *testing.T) {
	if got, err := nativeFixtureRootFindings(nativeRootSources()); err != nil || len(got) != 0 {
		t.Fatalf("clean sources: %v %v", got, err)
	}
	cases := []struct{ name, file, old, replacement string }{
		{"warm setup", nativeFixtureFiles[0], "setupRecoveryLiveEnvironment(t)", "omitted(t)"},
		{"bodyless entry", nativeFixtureFiles[0], "func TestRecoveryRealHelperAndSessionConformanceLive(){ setupRecoveryLiveEnvironment(t) }", "func TestRecoveryRealHelperAndSessionConformanceLive()"},
		{"missing entry", nativeFixtureFiles[0], "TestRecoveryRealHelperAndSessionConformanceLive", "renamedEntry"},
		{"park setup", nativeFixtureFiles[1], "setupRecoveryLiveEnvironment(t)", "omitted(t)"},
		{"nested handoff", nativeFixtureFiles[1], "cmd.Env = recoveryLiveChildEnvironment(t)", "cmd.Env = omitted(t)"},
	}
	for _, root := range nativeFixtureRoots {
		cases = append(cases, struct{ name, file, old, replacement string }{root, nativeFixtureFiles[2], fmt.Sprintf("os.Getenv(%q)", root), "omitted()"})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sources := nativeRootSources()
			sources[c.file] = []byte(strings.Replace(string(sources[c.file]), c.old, c.replacement, 1))
			got, err := nativeFixtureRootFindings(sources)
			if err != nil || len(got) != 1 || got[0].Path != nativeFixtureDir+c.file || got[0].Finding.Rule != "native-fixture-roots" || got[0].Finding.Line <= 0 || !strings.Contains(got[0].Finding.Message, "#636") {
				t.Fatalf("omission escaped named local lint: %v %v", got, err)
			}
		})
	}
	for _, file := range nativeFixtureFiles {
		t.Run("missing/"+file, func(t *testing.T) {
			sources := nativeRootSources()
			delete(sources, file)
			if _, err := nativeFixtureRootFindings(sources); err == nil {
				t.Fatal("missing protected source accepted")
			}
		})
		t.Run("malformed/"+file, func(t *testing.T) {
			sources := nativeRootSources()
			sources[file] = []byte("package ???")
			if _, err := nativeFixtureRootFindings(sources); err == nil {
				t.Fatal("malformed protected source accepted")
			}
		})
	}
}

func TestNativeFixtureRootReader(t *testing.T) {
	sources := nativeRootSources()
	reads := map[string]int{}
	got, err := readNativeFixtureRoots("repo", func(path string) ([]byte, error) {
		reads[path]++
		return sources[filepath.Base(path)], nil
	})
	if err != nil || len(got) != 0 || len(reads) != 3 {
		t.Fatalf("read boundary: %v %v %v", got, err, reads)
	}
	for path, count := range reads {
		if count != 1 || filepath.Dir(path) != filepath.Join("repo", filepath.FromSlash(nativeFixtureDir)) {
			t.Fatalf("unexpected source read: %q count=%d", path, count)
		}
	}
	sources[nativeFixtureFiles[0]] = []byte("package fixture")
	if findings, err := readNativeFixtureRoots("repo", func(path string) ([]byte, error) {
		return sources[filepath.Base(path)], nil
	}); err != nil || len(findings) != 1 {
		t.Fatalf("reader did not consume rule findings: %v %v", findings, err)
	}
	sentinel := errors.New("unreadable source")
	if _, err := readNativeFixtureRoots("repo", func(string) ([]byte, error) { return nil, sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("read failure lost: %v", err)
	}
}

// nativeFixtureRootFindings is deliberately a repository rule: its contract
// crosses three Pair-specific files, outside the single-file shared catalog.
func nativeFixtureRootFindings(sources map[string][]byte) ([]locatedFinding, error) {
	type function struct {
		node *ast.FuncDecl
		file string
		line int
	}
	functions := map[string]function{}
	for _, name := range nativeFixtureFiles {
		raw, ok := sources[name]
		if !ok {
			return nil, fmt.Errorf("native-fixture-roots: missing %s%s (#636)", nativeFixtureDir, name)
		}
		positions := token.NewFileSet()
		parsed, err := parser.ParseFile(positions, name, raw, 0)
		if err != nil {
			return nil, fmt.Errorf("native-fixture-roots: parse %s: %w", name, err)
		}
		for _, decl := range parsed.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				functions[name+":"+fn.Name.Name] = function{fn, name, positions.Position(fn.Pos()).Line}
			}
		}
	}
	var findings []locatedFinding
	require := func(file, name string, valid bool, message string) {
		if valid {
			return
		}
		line := functions[file+":"+name].line
		if line == 0 {
			line = 1
		}
		findings = append(findings, locatedFinding{nativeFixtureDir + file, golint.Finding{Line: line, Rule: "native-fixture-roots", Message: message + " (#636)"}})
	}
	inspect := func(file, name string, match func(ast.Node) bool) bool {
		fn := functions[file+":"+name].node
		found := false
		if fn != nil && fn.Body != nil {
			ast.Inspect(fn.Body, func(n ast.Node) bool { found = match(n) || found; return true })
		}
		return found
	}
	for _, entry := range []struct{ file, function string }{{nativeFixtureFiles[0], "TestRecoveryRealHelperAndSessionConformanceLive"}, {nativeFixtureFiles[1], "newRealParkConformance"}} {
		found := inspect(entry.file, entry.function, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return false
			}
			name, ok := call.Fun.(*ast.Ident)
			return ok && name.Name == "setupRecoveryLiveEnvironment"
		})
		require(entry.file, entry.function, found, entry.function+": restore explicit setupRecoveryLiveEnvironment call")
	}
	handoff := inspect(nativeFixtureFiles[1], "startParkLifecycleHelper", func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return false
		}
		field, ok := assign.Lhs[0].(*ast.SelectorExpr)
		if !ok || field.Sel.Name != "Env" {
			return false
		}
		command, ok := field.X.(*ast.Ident)
		if !ok || command.Name != "cmd" {
			return false
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return false
		}
		name, ok := call.Fun.(*ast.Ident)
		return ok && name.Name == "recoveryLiveChildEnvironment"
	})
	require(nativeFixtureFiles[1], "startParkLifecycleHelper", handoff, "startParkLifecycleHelper: restore explicit child cmd.Env handoff")
	for _, root := range nativeFixtureRoots {
		found := inspect(nativeFixtureFiles[2], "recoveryLiveChildEnvironment", func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return false
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Getenv" {
				return false
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return false
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return false
			}
			key, err := strconv.Unquote(literal.Value)
			return err == nil && key == root
		})
		require(nativeFixtureFiles[2], "recoveryLiveChildEnvironment", found, "recoveryLiveChildEnvironment: restore explicit "+root+" read")
	}
	return findings, nil
}

func readNativeFixtureRoots(root string, readFile func(string) ([]byte, error)) ([]locatedFinding, error) {
	sources := map[string][]byte{}
	for _, name := range nativeFixtureFiles {
		source, err := readFile(filepath.Join(root, filepath.FromSlash(nativeFixtureDir+name)))
		if err != nil {
			return nil, fmt.Errorf("native-fixture-roots: read %s: %w", name, err)
		}
		sources[name] = source
	}
	return nativeFixtureRootFindings(sources)
}

func TestNativeFixtureRootRealSources(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, name := range []string{"recovery_conformance_live_test.go", "park_lifecycle_live_test.go", "recovery_live_environment_test.go"} {
		raw, err := os.ReadFile(filepath.Join(root, nativeFixtureDir, name))
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = string(raw)
	}
	if violations, err := nativeFixtureRootFindings(rootSourceBytes(sources)); err != nil || len(violations) != 0 {
		t.Fatalf("live isolation source guard: %v", violations)
	}
	for _, mutation := range []struct{ name, file, old, replacement string }{
		{"warm setup", "recovery_conformance_live_test.go", "setupRecoveryLiveEnvironment(t)", "_ = t"},
		{"park setup", "park_lifecycle_live_test.go", "setupRecoveryLiveEnvironment(t)", "_ = t"},
		{"nested handoff", "park_lifecycle_live_test.go", "cmd.Env = recoveryLiveChildEnvironment(t,", "cmd.Env = omittedEnvironment(t,"},
		{"runtime root", "recovery_live_environment_test.go", `os.Getenv("XDG_RUNTIME_DIR")`, `os.Getenv("OMITTED_RUNTIME")`},
		{"socket root", "recovery_live_environment_test.go", `os.Getenv("ZELLIJ_SOCKET_DIR")`, `os.Getenv("OMITTED_SOCKET")`},
		{"config root", "recovery_live_environment_test.go", `os.Getenv("ZELLIJ_CONFIG_DIR")`, `os.Getenv("OMITTED_CONFIG")`},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			changed := map[string]string{}
			for name, source := range sources {
				changed[name] = source
			}
			source := changed[mutation.file]
			start := 0
			if mutation.file == "recovery_live_environment_test.go" {
				// Mutate the actual builder, excluding the mutation-case strings.
				start = strings.Index(source, "func recoveryLiveChildEnvironment(")
				if start < 0 {
					t.Fatal("child environment builder missing")
				}
			}
			changed[mutation.file] = source[:start] + strings.Replace(source[start:], mutation.old, mutation.replacement, 1)
			if changed[mutation.file] == sources[mutation.file] {
				t.Fatal("source mutation did not apply")
			}
			if violations, err := nativeFixtureRootFindings(rootSourceBytes(changed)); err != nil || len(violations) != 1 {
				t.Fatal("missing isolation boundary escaped source guard")
			}
		})
	}
}

func rootSourceBytes(sources map[string]string) map[string][]byte {
	result := map[string][]byte{}
	for name, source := range sources {
		result[name] = []byte(source)
	}
	return result
}
