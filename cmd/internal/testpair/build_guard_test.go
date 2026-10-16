package testpair

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestPerTestBuildGuardFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, path, source string
		want               bool
	}{
		{"direct", "cmd/example/main_test.go", `package example; func test() { exec.Command("go", "build", "-o", out, "./cmd/pair-go") }`, true},
		{"shared resolver source", "cmd/internal/testpair/binary.go", `package testpair; func binary() { exec.Command("go", "build", "-o", out, "./cmd/pair-go") }`, true},
		{"normal production source", "cmd/example/main.go", `package example; func build() { exec.Command("go", "build", "-o", out, "./cmd/pair-go") }`, false},
		{"context", "cmd/example/main_test.go", `package example; func test() { exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", out, "./cmd/pair-go") }`, true},
		{"argument table", "cmd/couch/main_test.go", `package example; func test() { for _, command := range [][]string{{"build", "-o", pairBin, "./cmd/pair-go"}, {"test", "-c", "-o", fixtureBin, "./cmd/internal/couchcore"}} { exec.Command("go", command...) } }`, true},
		{"struct table", "cmd/example/main_test.go", `package example; func test() { for _, target := range []struct{name, pkg string}{{"pair", "./cmd/pair-go"}, {"helper", "./cmd/pair-launch-helper"}} { exec.CommandContext(ctx, "go", "build", "-o", target.name, target.pkg) } }`, true},
		{"local variable", "cmd/example/main_test.go", `package example; func test() { pkg := "./cmd/pair-go"; exec.Command("go", "build", "-o", out, pkg) }`, true},
		{"package dot", "cmd/pair-go/main_test.go", `package main; func test() { exec.Command("go", "build", "-o", out, ".") }`, true},
		{"package parameter", "cmd/pair-go/main_test.go", `package main; func buildCommand(out, pkg string) { exec.Command("go", "build", "-o", out, pkg) }`, true},
		{"implicit package", "cmd/pair-go/main_test.go", `package main; func test() { exec.Command("go", "build", "-o", out) }`, true},
		{"shell", "tests/new-test.sh", "go build -o \"$bin\" ./cmd/pair-go\n", true},
		{"shell continuation", "tests/new-test.sh", "go build \\\n  -o \"$bin\" \\\n  ./cmd/pair-go\n", true},
		{"python argv", "tests/native-terminal-ci.py", `subprocess.run(["go", "build", "-o", candidate, "./cmd/pair-go"], check=True)`, true},
		{"python multiline", "tests/new-test.py", "subprocess.check_call([\n 'go', 'build', '-o', candidate,\n './cmd/pair-go',\n])", true},
		{"authorized helper", "tests/lib/pair-test-binary.sh", "go build -o \"$bin\" ./cmd/pair-go\n", false},
		{"couch", "cmd/couch/main_test.go", `package main; func test() { exec.Command("go", "build", "-o", out, "./cmd/couch") }`, false},
		{"launch helper", "cmd/example/main_test.go", `package main; func test() { for _, target := range []struct{name, pkg string}{{"helper", "./cmd/pair-launch-helper"}} { exec.CommandContext(ctx, "go", "build", "-o", target.name, target.pkg) } }`, false},
		{"probe variable", "cmd/pair-go/main_test.go", `package main; func test() { pkg := "./probes/messageidle"; exec.Command("go", "build", "-o", out, pkg) }`, false},
		{"test executable", "cmd/couch/main_test.go", `package main; func test() { for _, command := range [][]string{{"test", "-c", "-o", out, "./cmd/pair-go"}} { exec.Command("go", command...) } }`, false},
		{"unrelated table rows", "cmd/couch/main_test.go", `package main; func test() { for _, command := range [][]string{{"build", "-o", out, "./cmd/couch"}, {"test", "-c", "-o", out, "./cmd/pair-go"}} { exec.Command("go", command...) } }`, false},
		{"unrelated dot", "cmd/couch/main_test.go", `package main; func test() { exec.Command("go", "build", "-o", out, ".") }`, false},
		{"separate function facts", "cmd/example/main_test.go", `package main; func route() { use("./cmd/pair-go") }; func test() { exec.Command("go", "build", "-o", out, "./cmd/couch") }`, false},
		{"shell probe", "tests/new-test.sh", "go build -o \"$bin\" ./probes/messageidle\n", false},
		{"shell comment", "tests/new-test.sh", "# go build -o \"$bin\" ./cmd/pair-go\n", false},
		{"python helper", "tests/new-test.py", `subprocess.run(["go", "build", "-o", candidate, "./cmd/pair-launch-helper"], check=True)`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := perTestPairBuilds(tc.path, []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if (len(got) != 0) != tc.want {
				t.Fatalf("build findings = %v, want detected=%v", got, tc.want)
			}
		})
	}
}

func TestNoPerTestBuildsOfFullPair(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "third_party", "node_modules", "workshop", ".agents", ".claude", "construct":
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !guardSource(filepath.ToSlash(rel)) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		findings, err := perTestPairBuilds(filepath.ToSlash(rel), body)
		if err != nil {
			return err
		}
		for _, finding := range findings {
			t.Errorf("%s: use testpair.Binary/At or tests/lib/pair-test-binary.sh instead of a per-test full Pair build", finding)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func guardSource(path string) bool {
	return strings.HasSuffix(path, "_test.go") ||
		(filepath.ToSlash(filepath.Dir(path)) == "cmd/internal/testpair" && strings.HasSuffix(path, ".go")) ||
		strings.HasSuffix(path, ".sh") || strings.HasSuffix(path, ".py")
}

// This is a regression guard for ordinary test build recipes, not a Go/shell/
// Python interpreter. Go facts stay within one function; local argv variables,
// range tables and string parameters cover the recipes that previously linked
// Pair for each test. The shared shell builder is the sole test-build owner;
// the development Makefile's build target is intentionally outside this scope.
func perTestPairBuilds(path string, source []byte) ([]string, error) {
	if !guardSource(path) || path == "tests/lib/pair-test-binary.sh" {
		return nil, nil
	}
	if strings.HasSuffix(path, ".go") {
		return goPairBuilds(path, source)
	}
	var findings []string
	record := func(offset int, argv []string) {
		if fullPairBuild(path, argv) {
			findings = append(findings, fmt.Sprintf("%s:%d", path, 1+strings.Count(string(source[:offset]), "\n")))
		}
	}
	if strings.HasSuffix(path, ".py") {
		// Python CI uses subprocess argv lists. Retain unknown entries such as
		// candidate so a -o value cannot swallow the following package target.
		for _, match := range pythonBuildArgv.FindAllSubmatchIndex(source, -1) {
			var argv []string
			for _, arg := range strings.Split(string(source[match[2]:match[3]]), ",") {
				arg = strings.TrimSpace(arg)
				if arg != "" {
					argv = append(argv, unquoteBuildArg(arg))
				}
			}
			record(match[0], argv)
		}
	} else {
		// A shell continuation is whitespace within one logical command.
		logical := strings.ReplaceAll(string(source), "\\\n", "  ")
		for _, match := range shellBuildCommand.FindAllStringIndex(logical, -1) {
			lineStart := strings.LastIndex(logical[:match[0]], "\n") + 1
			if strings.HasPrefix(strings.TrimSpace(logical[lineStart:]), "#") {
				continue
			}
			var argv []string
			for _, arg := range shellBuildWord.FindAllString(logical[match[0]:match[1]], -1) {
				if strings.HasPrefix(arg, "#") {
					break
				}
				argv = append(argv, unquoteBuildArg(arg))
			}
			record(match[0], argv)
		}
	}
	return findings, nil
}

var (
	pythonBuildArgv   = regexp.MustCompile(`(?s)subprocess\.(?:run|call|check_call|check_output|Popen)\s*\(\s*\[([^\]]*)\]`)
	shellBuildCommand = regexp.MustCompile(`\bgo[ \t]+build\b[^\n;&|)]*`)
	shellBuildWord    = regexp.MustCompile(`(?:"[^"]*"|'[^']*'|[^\s]+)`)
)

func unquoteBuildArg(arg string) string {
	if len(arg) >= 2 && (arg[0] == '\'' || arg[0] == '"') && arg[len(arg)-1] == arg[0] {
		return arg[1 : len(arg)-1]
	}
	return arg
}

func fullPairBuild(path string, argv []string) bool {
	if len(argv) < 2 || filepath.Base(argv[0]) != "go" || argv[1] != "build" {
		return false
	}
	localPair := filepath.ToSlash(filepath.Dir(path)) == "cmd/pair-go"
	var targets []string
	for i := 2; i < len(argv); i++ {
		arg := argv[i]
		if strings.HasPrefix(arg, "-") {
			// These Go build flags consume the next argument. Boolean flags
			// (-trimpath, -race, etc.) and -flag=value consume no extra entry.
			switch arg {
			case "-o", "-tags", "-ldflags", "-gcflags", "-asmflags", "-mod", "-modfile", "-overlay", "-p", "-pkgdir", "-toolexec", "-buildmode", "-compiler", "-coverpkg", "-installsuffix":
				i++
			}
			continue
		}
		targets = append(targets, arg)
	}
	if localPair && len(targets) == 0 {
		return true
	}
	for _, target := range targets {
		target = strings.TrimSuffix(target, "/")
		if target == "./cmd/pair-go" || strings.HasSuffix(target, "/cmd/pair-go") || (localPair && (target == "." || target == "?")) {
			return true
		}
	}
	return false
}

func goPairBuilds(path string, source []byte) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		return nil, err
	}
	var findings []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		facts := buildFacts{vars: map[string][]ast.Expr{}, fields: map[string][]ast.Expr{}}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.AssignStmt:
				for i, lhs := range n.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && i < len(n.Rhs) {
						facts.vars[id.Name] = append(facts.vars[id.Name], n.Rhs[i])
					}
				}
			case *ast.ValueSpec:
				for i, id := range n.Names {
					if i < len(n.Values) {
						facts.vars[id.Name] = append(facts.vars[id.Name], n.Values[i])
					}
				}
			case *ast.RangeStmt:
				if id, ok := n.Value.(*ast.Ident); ok {
					facts.vars[id.Name] = append(facts.vars[id.Name], n.X)
				}
			case *ast.CompositeLit:
				facts.structFields(n)
			}
			return true
		})
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext") {
				return true
			}
			args := call.Args
			if sel.Sel.Name == "CommandContext" && len(args) > 0 {
				args = args[1:]
			}
			for _, argv := range facts.argv(args, 0) {
				if fullPairBuild(path, argv) {
					findings = append(findings, fset.Position(call.Pos()).String())
					break
				}
			}
			return true
		})
	}
	return findings, nil
}

// Values are alternatives, while each inner slice retains argv order. In
// particular, rows of a [][]string table must never blend "build couch" and
// "test -c pair-go" into a spurious "build pair-go" finding.
type buildFacts struct {
	vars   map[string][]ast.Expr
	fields map[string][]ast.Expr
}

func (f buildFacts) structFields(lit *ast.CompositeLit) {
	array, ok := lit.Type.(*ast.ArrayType)
	if !ok {
		return
	}
	st, ok := array.Elt.(*ast.StructType)
	if !ok {
		return
	}
	var names []string
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	for _, element := range lit.Elts {
		if row, ok := element.(*ast.CompositeLit); ok {
			for i, value := range row.Elts {
				if kv, ok := value.(*ast.KeyValueExpr); ok {
					if key, ok := kv.Key.(*ast.Ident); ok {
						f.fields[key.Name] = append(f.fields[key.Name], kv.Value)
					}
				} else if i < len(names) {
					f.fields[names[i]] = append(f.fields[names[i]], value)
				}
			}
		}
	}
}

func (f buildFacts) argv(args []ast.Expr, depth int) [][]string {
	rows := [][]string{{}}
	for _, arg := range args {
		var next [][]string
		for _, prefix := range rows {
			for _, value := range f.values(arg, depth+1) {
				next = append(next, append(append([]string{}, prefix...), value...))
			}
		}
		rows = next
	}
	return rows
}

func (f buildFacts) values(expr ast.Expr, depth int) [][]string {
	if depth > 12 { // Cyclic assignments are unresolved, never recursively walked.
		return [][]string{{"?"}}
	}
	var alternatives []ast.Expr
	switch n := expr.(type) {
	case *ast.BasicLit:
		if n.Kind == token.STRING {
			value, _ := strconv.Unquote(n.Value)
			return [][]string{{value}}
		}
	case *ast.Ident:
		alternatives = f.vars[n.Name]
	case *ast.SelectorExpr:
		alternatives = f.fields[n.Sel.Name]
	case *ast.CompositeLit:
		if len(n.Elts) > 0 {
			if _, nested := n.Elts[0].(*ast.CompositeLit); nested {
				alternatives = n.Elts
			} else {
				return f.argv(n.Elts, depth+1)
			}
		}
	}
	var rows [][]string
	for _, alternative := range alternatives {
		rows = append(rows, f.values(alternative, depth+1)...)
	}
	if len(rows) == 0 {
		return [][]string{{"?"}}
	}
	return rows
}
