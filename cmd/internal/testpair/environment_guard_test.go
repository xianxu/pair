package testpair

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestPairEnvironmentGuardFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{"direct", `package fixture; import "os/exec"; func f(){ c:=exec.Command("pair", "version"); c.Run() }`, true},
		{"resolver alias", `package fixture; import x "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair"; func f(){ a:=tp.Binary(t); b:=a; c:=x.CommandContext(ctx,b,"title"); c.Env=os.Environ(); c.Run() }`, true},
		{"safe", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair"; func f(){ c:=exec.Command(tp.Binary(t)); c.Env=tp.Environment(t, t.TempDir()); c.Run() }`, false},
		{"ambient explicit", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){c:=exec.Command(tp.Binary(t));c.Env=tp.Environment(t,t.TempDir(),os.Environ()...);c.Run()}`, true},
		{"ambient explicit alias", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){e:=os.Environ();c:=exec.Command(tp.Binary(t));c.Env=tp.Environment(t,t.TempDir(),e...);c.Run()}`, true},
		{"initializes after spawn", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){c:=exec.Command(tp.Binary(t));c.Run();c.Env=tp.Environment(t,t.TempDir())}`, true},
		{"conditional only", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){c:=exec.Command(tp.Binary(t));if cond {c.Env=tp.Environment(t,t.TempDir())};c.Run()}`, true},
		{"context resolver", `package fixture; import "os/exec";import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){b:=tp.BinaryContext(t,ctx);c:=exec.Command(b);c.Run()}`, true},
		{"candidate selector", `package fixture; import "os/exec";func f(){b:=os.Getenv("PAIR_NATIVE_BINARY");c:=exec.Command(b);c.Run()}`, true},
		{"ambient helper return", `package fixture; import "os/exec";import tp "github.com/xianxu/pair/cmd/internal/testpair";func env()[]string{return os.Environ()};func f(){c:=exec.Command(tp.Binary(t));c.Env=tp.Environment(t,t.TempDir(),env()...);c.Run()}`, true},
		{"conditional alias", `package fixture; import "os/exec";import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){e:=os.Environ();if cond{e=tp.Environment(t,t.TempDir())};c:=exec.Command("pair");c.Env=e;c.Run()}`, true},
		{"later replacement", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair"; func f(){ c:=exec.Command(tp.Binary(t)); c.Env=tp.Environment(t,t.TempDir()); c.Env=os.Environ(); c.Run() }`, true},
		{"safe append", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair"; func f(){ env:=tp.Environment(t,t.TempDir()); env=append(env,"PAIR_TAG=x"); c:=exec.Command(tp.Binary(t)); c.Env=env; c.Run() }`, false},
		{"unsafe append", `package fixture; import "os/exec"; func f(){ env:=append(os.Environ(),"HOME=/tmp"); c:=exec.Command("pair-wrap"); c.Env=env; c.Run() }`, true},
		{"parameter forwarding", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair"; func f(){run(tp.Binary(t),tp.Environment(t,t.TempDir()))}; func run(name string, env []string){c:=exec.Command(name); c.Env=env;c.Run()}`, false},
		{"unsafe parameter forwarding", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair"; func f(){run(tp.Binary(t),os.Environ())}; func run(name string, env []string){c:=exec.Command(name); c.Env=env;c.Run()}`, true},
		{"safe and unsafe callers", `package fixture; import "os/exec"; import tp "github.com/xianxu/pair/cmd/internal/testpair"; func f(){run(tp.Binary(t),tp.Environment(t,t.TempDir()));run(tp.Binary(t),os.Environ())}; func run(name string, env []string){c:=exec.Command(name); c.Env=env;c.Run()}`, true},
		{"selfexec wrapper", `package wrapcmd; import "os/exec"; func f(){exe,_:=os.Executable();c:=exec.Command(exe,"-test.run=^TestWrapperHelper$");c.Run()};func TestWrapperHelper(t *testing.T){Run(nil,os.Stdin,os.Stdout,os.Stderr)}`, true},
		{"unrelated selfexec", `package fixture; import "os/exec"; func f(){exe,_:=os.Executable();c:=exec.Command(exe,"-test.run=^TestProcessHelper$");c.Run()};func TestProcessHelper(t *testing.T){os.Exit(0)}`, false},
		{"shell nested", `package fixture; import "os/exec";func f(){c:=exec.Command("sh","-c","exec /repo/bin/pair-wrap codex");c.Run()}`, true},
		{"build", `package fixture;import "os/exec";func f(){exec.Command("go","build","./cmd/pair-go").Run()}`, false},
		{"unrelated", `package fixture;import "os/exec";func f(){exec.Command("git","status").Run()}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pairEnvironmentFindings("cmd/fixture/x_test.go", []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if (len(got) > 0) != tc.want {
				t.Fatalf("findings=%v, want detected=%v", got, tc.want)
			}
		})
	}
}

func TestPairSubprocessesUseIsolatedEnvironment(t *testing.T) {
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
		rel = filepath.ToSlash(rel)
		if !strings.HasSuffix(rel, "_test.go") && !(strings.HasPrefix(rel, "tests/") && (strings.HasSuffix(rel, ".sh") || strings.HasSuffix(rel, ".py"))) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		findings, err := pairEnvironmentFindings(rel, body)
		if err != nil {
			return err
		}
		for _, finding := range findings {
			t.Errorf("%s: Pair/wrapper subprocess must assign testpair.Environment(t, disposableRoot, explicit...) to Env", finding)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// This regression guard follows ordinary local aliases and test helper calls.
// It is deliberately source based: it never executes the subprocess under audit.
// Unknown unrelated executables are outside its remit; known Pair origins and
// wrapper self-exec helpers remain tainted through parameters and shell strings.
// fixture marks an explicit opt-in to the stateful terminal fixture. Its
// executable may be Couch, but that invocation can launch Pair/wrapper children.
// This trait belongs to the assigned environment, never the whole test/file.
type environmentValue struct{ pair, safe, ambient, fixture bool }
type environmentFunction struct {
	decl   *ast.FuncDecl
	params []string
	inputs []environmentValue
	seen   []bool
	result environmentValue
}

func pairEnvironmentFindings(path string, source []byte) ([]string, error) {
	if !strings.HasSuffix(path, ".go") {
		return scriptPairEnvironmentFindings(path, source), nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		return nil, err
	}
	execNames := map[string]bool{}
	pairNames := map[string]bool{}
	for _, im := range file.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		name := filepath.Base(p)
		if im.Name != nil {
			name = im.Name.Name
		}
		if p == "os/exec" {
			execNames[name] = true
		}
		if p == "github.com/xianxu/pair/cmd/internal/testpair" {
			pairNames[name] = true
		}
	}
	funcs := map[string]*environmentFunction{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		f := &environmentFunction{decl: fn}
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					f.params = append(f.params, name.Name)
				}
			}
		}
		f.inputs = make([]environmentValue, len(f.params))
		f.seen = make([]bool, len(f.params))
		funcs[fn.Name.Name] = f
	}
	wrapperHelpers := map[string]bool{}
	for name, fn := range funcs {
		ast.Inspect(fn.decl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				if (file.Name.Name == "wrapcmd" && (fun.Name == "Run" || fun.Name == "run")) || (file.Name.Name == "main" && fun.Name == "run") {
					wrapperHelpers[name] = true
				}
			case *ast.SelectorExpr:
				if x, ok := fun.X.(*ast.Ident); ok && (x.Name == "wrapcmd" || x.Name == "titlepoller") && (fun.Sel.Name == "Run" || fun.Sel.Name == "RunCLI") {
					wrapperHelpers[name] = true
				}
			}
			return true
		})
	}
	var findings []string
	for round := 0; round < len(funcs)+2; round++ {
		changed := false
		findings = nil
		for _, fn := range funcs {
			returnSafeSeen, returnUnsafe, returnAmbient, returnFixture := false, false, false, false
			values := map[string]environmentValue{}
			for i, p := range fn.params {
				values[p] = fn.inputs[i]
			}
			commands := map[*ast.CallExpr]bool{}
			hasPairRecipe := false
			ast.Inspect(fn.decl.Body, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					v, _ := strconv.Unquote(lit.Value)
					hasPairRecipe = hasPairRecipe || strings.Contains(v, "exec pair wrap") || strings.Contains(v, "PAIR_REAL=")
				}
				return true
			})
			bindings := map[string]*ast.CallExpr{}
			envs := map[*ast.CallExpr]bool{}
			fixtureEnvs := map[*ast.CallExpr]bool{}
			spawnedUnsafe := map[*ast.CallExpr]bool{}
			creationAncestors := map[*ast.CallExpr][]ast.Node{}
			var ancestors []ast.Node
			aliases := map[string]string{}
			var value func(ast.Expr) environmentValue
			value = func(expr ast.Expr) environmentValue {
				switch e := expr.(type) {
				case *ast.Ident:
					return values[e.Name]
				case *ast.SelectorExpr:
					if e.Sel.Name == "Env" {
						if id, ok := e.X.(*ast.Ident); ok {
							if c := bindings[id.Name]; c != nil {
								return environmentValue{safe: envs[c], fixture: fixtureEnvs[c]}
							}
						}
					}
				case *ast.BasicLit:
					if e.Kind == token.STRING {
						s, _ := strconv.Unquote(e.Value)
						return environmentValue{pair: pairExecutableText(s), fixture: s == "PAIR_TEST_COUCH_TERMINAL_FIXTURE=1"}
					}
				case *ast.BinaryExpr:
					a, b := value(e.X), value(e.Y)
					return environmentValue{pair: a.pair || b.pair, ambient: a.ambient || b.ambient, fixture: a.fixture || b.fixture}
				case *ast.ParenExpr:
					return value(e.X)
				case *ast.CallExpr:
					if id, ok := e.Fun.(*ast.Ident); ok {
						if callee := funcs[id.Name]; callee != nil {
							return callee.result
						}
					}
					if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
						if sel.Sel.Name == "Getenv" && len(e.Args) == 1 {
							if lit, ok := e.Args[0].(*ast.BasicLit); ok {
								key, _ := strconv.Unquote(lit.Value)
								if key == "PAIR_NATIVE_BINARY" || key == "PAIR_TEST_BINARY" || key == "PAIR_BIN" {
									return environmentValue{pair: true}
								}
							}
						}
						if sel.Sel.Name == "LookPath" && len(e.Args) > 0 {
							return value(e.Args[0])
						}
						if sel.Sel.Name == "Environ" {
							return environmentValue{ambient: true}
						}
						if x, ok := sel.X.(*ast.Ident); ok && pairNames[x.Name] {
							switch sel.Sel.Name {
							case "Binary", "At", "BinaryContext", "AtContext":
								return environmentValue{pair: true}
							case "Environment", "EnvironmentAt", "EnvironmentFrom":
								ambient, fixture := false, false
								for _, arg := range e.Args {
									v := value(arg)
									ambient = ambient || v.ambient
									fixture = fixture || v.fixture
								}
								return environmentValue{safe: !ambient, ambient: ambient, fixture: fixture}
							}
						}
						if sel.Sel.Name == "Join" {
							var v environmentValue
							for _, a := range e.Args {
								v.pair = v.pair || value(a).pair
								v.fixture = v.fixture || value(a).fixture
							}
							return v
						}
					}
					if fun, ok := e.Fun.(*ast.Ident); ok && (fun.Name == "append" || fun.Name == "setEnv" || fun.Name == "setEnvValue") && len(e.Args) > 0 {
						v := value(e.Args[0])
						for _, arg := range e.Args[1:] {
							v.ambient = v.ambient || value(arg).ambient
							v.fixture = v.fixture || value(arg).fixture
						}
						if (fun.Name == "setEnv" || fun.Name == "setEnvValue") && len(e.Args) == 3 {
							v.fixture = v.fixture || terminalFixtureEntry(e.Args[1], e.Args[2])
						}
						v.safe = v.safe && !v.ambient
						return v
					}
				case *ast.KeyValueExpr:
					key, entry := value(e.Key), value(e.Value)
					return environmentValue{fixture: key.fixture || entry.fixture || terminalFixtureEntry(e.Key, e.Value)}
				case *ast.CompositeLit:
					var v environmentValue
					for _, a := range e.Elts {
						v.pair = v.pair || value(a).pair
						v.fixture = v.fixture || value(a).fixture
					}
					return v
				}
				return environmentValue{}
			}
			execCall := func(c *ast.CallExpr) (bool, []ast.Expr) {
				var name string
				switch f := c.Fun.(type) {
				case *ast.Ident:
					name = aliases[f.Name]
				case *ast.SelectorExpr:
					if x, ok := f.X.(*ast.Ident); ok && execNames[x.Name] {
						name = f.Sel.Name
					}
				}
				args := c.Args
				if name == "CommandContext" && len(args) > 0 {
					args = args[1:]
				}
				return name == "Command" || name == "CommandContext", args
			}
			ast.Inspect(fn.decl.Body, func(node ast.Node) bool {
				if node == nil {
					ancestors = ancestors[:len(ancestors)-1]
					return false
				}
				ancestors = append(ancestors, node)
				switch n := node.(type) {
				case *ast.AssignStmt:
					for i, lhs := range n.Lhs {
						if i >= len(n.Rhs) {
							continue
						}
						rhs := n.Rhs[i]
						if id, ok := lhs.(*ast.Ident); ok {
							next := value(rhs)
							if old, present := values[id.Name]; present {
								conditional := false
								for _, a := range ancestors {
									switch a.(type) {
									case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
										conditional = true
									}
								}
								if conditional {
									next.pair = next.pair || old.pair
									next.ambient = next.ambient || old.ambient
									next.fixture = next.fixture || old.fixture
									next.safe = next.safe && old.safe
								}
							}
							values[id.Name] = next
							if c, ok := rhs.(*ast.CallExpr); ok {
								if is, _ := execCall(c); is {
									bindings[id.Name] = c
									creationAncestors[c] = append([]ast.Node{}, ancestors...)
								}
							}
							if from, ok := rhs.(*ast.Ident); ok {
								if c := bindings[from.Name]; c != nil {
									bindings[id.Name] = c
								}
								if a := aliases[from.Name]; a != "" {
									aliases[id.Name] = a
								}
							}
							if s, ok := rhs.(*ast.SelectorExpr); ok {
								if x, ok := s.X.(*ast.Ident); ok && execNames[x.Name] {
									aliases[id.Name] = s.Sel.Name
								}
							}
						}
						if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Env" {
							if id, ok := sel.X.(*ast.Ident); ok {
								if c := bindings[id.Name]; c != nil {
									assigned := value(rhs)
									next := assigned.safe
									commands[c] = commands[c] || assigned.fixture
									fixtureEnvs[c] = assigned.fixture
									conditional := false
									for _, a := range ancestors {
										switch a.(type) {
										case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
											present := false
											for _, base := range creationAncestors[c] {
												present = present || base == a
											}
											conditional = conditional || !present
										}
									}
									if conditional {
										next = envs[c] && next
									}
									envs[c] = next
								}
							}
						}
					}
				case *ast.ReturnStmt:
					if len(n.Results) > 0 {
						v := value(n.Results[0])
						returnAmbient = returnAmbient || v.ambient
						returnFixture = returnFixture || v.fixture
						if id, ok := n.Results[0].(*ast.Ident); !ok || id.Name != "nil" {
							returnSafeSeen = returnSafeSeen || v.safe
							returnUnsafe = returnUnsafe || !v.safe
						}
						if v.pair && !fn.result.pair {
							fn.result.pair = true
							changed = true
						}
					}
				case *ast.RangeStmt:
					entry := value(n.X)
					for _, binding := range []ast.Expr{n.Key, n.Value} {
						if id, ok := binding.(*ast.Ident); ok {
							bound := values[id.Name]
							bound.fixture = bound.fixture || entry.fixture
							values[id.Name] = bound
						}
					}
				case *ast.ValueSpec:
					for i, id := range n.Names {
						if i < len(n.Values) {
							values[id.Name] = value(n.Values[i])
						}
					}
				case *ast.CallExpr:
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
						switch sel.Sel.Name {
						case "Run", "Start", "Output", "CombinedOutput":
							var command *ast.CallExpr
							if id, ok := sel.X.(*ast.Ident); ok {
								command = bindings[id.Name]
							} else if call, ok := sel.X.(*ast.CallExpr); ok {
								if is, _ := execCall(call); is {
									command = call
								}
							}
							if command != nil && !envs[command] {
								spawnedUnsafe[command] = true
							}
						}
					}
					if is, args := execCall(n); is {
						pair := false
						if len(args) > 0 {
							pair = value(args[0]).pair
						}
						for _, arg := range args {
							if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
								s, _ := strconv.Unquote(lit.Value)
								if strings.HasPrefix(s, "-test.run=") {
									for helper := range wrapperHelpers {
										if strings.Contains(s, helper) {
											pair = true
										}
									}
								}
								if len(args) > 0 {
									if first, ok := args[0].(*ast.BasicLit); ok {
										command, _ := strconv.Unquote(first.Value)
										if command == "sh" || command == "bash" || command == "/bin/sh" || command == "/bin/bash" {
											pair = pair || pairExecutableText(s)
										}
									}
								}
							}
						}
						if len(args) > 0 {
							if lit, ok := args[0].(*ast.BasicLit); ok {
								v, _ := strconv.Unquote(lit.Value)
								if (filepath.Base(v) == "sh" || filepath.Base(v) == "bash") && hasPairRecipe {
									pair = true
								}
							}
						}
						commands[n] = pair
					} else if id, ok := n.Fun.(*ast.Ident); ok {
						if callee := funcs[id.Name]; callee != nil {
							for i, arg := range n.Args {
								if i >= len(callee.inputs) {
									break
								}
								v := value(arg)
								old := callee.inputs[i]
								next := environmentValue{pair: old.pair || v.pair, safe: v.safe, ambient: old.ambient || v.ambient, fixture: old.fixture || v.fixture}
								if callee.seen[i] {
									next.safe = old.safe && v.safe
								}
								if !callee.seen[i] || old != next {
									callee.inputs[i] = next
									callee.seen[i] = true
									changed = true
								}
							}
						}
					}
				}
				return true
			})
			if returnFixture != fn.result.fixture {
				fn.result.fixture = returnFixture
				changed = true
			}
			if returnAmbient != fn.result.ambient {
				fn.result.ambient = returnAmbient
				changed = true
			}
			if next := returnSafeSeen && !returnUnsafe; next != fn.result.safe {
				fn.result.safe = next
				changed = true
			}
			for c, pair := range commands {
				if pair && (!envs[c] || spawnedUnsafe[c]) {
					findings = append(findings, fset.Position(c.Pos()).String())
				}
			}
		}
		if !changed {
			break
		}
	}
	return findings, nil
}

// Map/setEnv recipes split an explicit environment entry into key and value.
func terminalFixtureEntry(key, value ast.Expr) bool {
	k, kok := key.(*ast.BasicLit)
	v, vok := value.(*ast.BasicLit)
	if !kok || !vok || k.Kind != token.STRING || v.Kind != token.STRING {
		return false
	}
	name, _ := strconv.Unquote(k.Value)
	enabled, _ := strconv.Unquote(v.Value)
	return name == "PAIR_TEST_COUCH_TERMINAL_FIXTURE" && enabled == "1"
}

func pairExecutableText(text string) bool {
	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune(" \t\r\n;|&'\"()", r) }) {
		switch filepath.Base(word) {
		case "pair", "pair-go", "pair-wrap":
			return true
		}
		if strings.HasPrefix(filepath.Base(word), "pair-") && !strings.HasSuffix(word, ".sh") && !strings.HasSuffix(word, ".py") {
			return true
		}
	}
	return false
}

// Shell fixtures enter the same shared allowlist before any executable fixture
// statement, including embedded Python and editor programs that launch Pair.
// Merely mentioning or sourcing the bootstrap later cannot satisfy the guard.
const pairTestBootstrapSource = `. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"`

var scriptPairReference = regexp.MustCompile(`(?m)(?:^|[;&|{]\s*|\bexec\s+)["']?pair(?:-wrap|-go)?["']?[ \t]|bin/pair\b|PAIR_BIN|\$PAIR\b|pair-test-binary|exec pair|\./cmd/pair-go|\[binary,\s*['"]term['"]`)

// Shell entry files have a fixed two-line prelude; the bootstrap executes
// lines 3 onward in the clean child. Python recognizes the actual env used by
// each Pair spawn, not a source-wide mention of the helper. This remains a
// bounded regression guard for test recipes, not a shell/Python interpreter.
func scriptPairEnvironmentFindings(path string, source []byte) []string {
	if path == "tests/lib/pair-test-binary.sh" || path == "tests/lib/with-pair-test-binary.sh" {
		return nil
	} // canonical build owner, no Pair execution
	var executable []string
	lines := strings.Split(string(source), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			executable = append(executable, trimmed)
		}
	}
	body := strings.Join(executable, "\n")
	if strings.HasSuffix(path, ".py") {
		checker, err := filepath.Abs("../../../tests/lib/pair_test_environment_guard.py")
		if err != nil {
			return []string{path + ": cannot resolve Python guard"}
		}
		command := exec.Command("python3", "-I", checker)
		command.Env = []string{"PATH=" + os.Getenv("PATH")}
		command.Stdin = bytes.NewReader(source)
		output, err := command.Output()
		if err != nil {
			return []string{path + ": Python guard failed: " + err.Error()}
		}
		var rows []int
		if err := json.Unmarshal(output, &rows); err != nil {
			return []string{path + ": invalid Python guard result"}
		}
		var findings []string
		for _, row := range rows {
			findings = append(findings, path+":"+strconv.Itoa(row))
		}
		return findings
	}
	if !scriptPairReference.MatchString(body) {
		return nil
	}
	depth := strings.Count(filepath.ToSlash(filepath.Dir(path)), "/")
	expected := strings.Replace(pairTestBootstrapSource, "/lib/pair-test-bootstrap.sh", "/"+strings.Repeat("../", depth)+"lib/pair-test-bootstrap.sh", 1)
	if len(lines) > 1 && strings.HasPrefix(lines[0], "#!") && strings.TrimSpace(lines[1]) == expected {
		return nil
	}
	return []string{path + ":1"}
}

func TestPairEnvironmentGuardScriptFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, path, source string
		want               bool
	}{
		{"literal wrapper", "tests/nested/a.sh", `pair-wrap codex`, true},
		{"Python ambient with decoy", "tests/nested/a.py", `helper='lib/pair-test-env.sh';flag='--print0'
subprocess.Popen(['pair','term'],env=os.environ)`, true},
		{"Python late helper", "tests/a.py", `env=os.environ
subprocess.Popen([binary,'term'],env=env)
helper=path/'lib/pair-test-env.sh'
raw=subprocess.check_output(['bash',str(helper),'--print0',str(root)])
env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)`, true},
		{"Python ambient alias replacement", "tests/a.py", `helper=path/'lib/pair-test-env.sh'
raw=subprocess.check_output(['bash',str(helper),'--print0',str(root)])
env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
inherited=os.environ
env=inherited
subprocess.Popen([binary,'term'],env=env)`, true},
		{"Python branch assignment", "tests/a.py", `env=os.environ
if cond:
    helper=path/'lib/pair-test-env.sh'
    raw=subprocess.check_output(['/bin/sh',str(helper),'--print0',str(root)])
    env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
subprocess.Popen([binary,'term'],env=env)`, true},
		{"Python positional update plus keyword", "tests/a.py", `helper=path/'lib/pair-test-env.sh'
raw=subprocess.check_output(['/bin/sh',str(helper),'--print0',str(root)])
env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
inherited=os.environ
env.update(inherited,FIXTURE='yes')
subprocess.Popen([binary,'term'],env=env)`, true},
		{"Python loop binding", "tests/a.py", `helper=path/'lib/pair-test-env.sh'
raw=subprocess.check_output(['/bin/sh',str(helper),'--print0',str(root)])
env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
for env in [os.environ]:
    subprocess.Popen([binary,'term'],env=env)`, true},
		{"Python with binding", "tests/a.py", `helper=path/'lib/pair-test-env.sh'
raw=subprocess.check_output(['/bin/sh',str(helper),'--print0',str(root)])
env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
with source() as env:
    subprocess.Popen([binary,'term'],env=env)`, true},
		{"Python exception after mutation", "tests/a.py", `helper=path/'lib/pair-test-env.sh'
raw=subprocess.check_output(['/bin/sh',str(helper),'--print0',str(root)])
env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
try:
    env=os.environ
    raise ValueError()
except ValueError:
    subprocess.Popen([binary,'term'],env=env)`, true},
		{"Python finally after mutation", "tests/a.py", `helper=path/'lib/pair-test-env.sh'
raw=subprocess.check_output(['/bin/sh',str(helper),'--print0',str(root)])
env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
try:
    env=os.environ
    raise ValueError()
    env=dict(entry.decode().split('=',1) for entry in raw.split(b'\0') if entry)
finally:
    subprocess.Popen([binary,'term'],env=env)`, true},
		{"shell original", "tests/a.sh", `#!/bin/bash
 "$ROOT/bin/pair" title t codex s`, true},
		{"shell safe", "tests/a.sh", "#!/bin/bash\n" + pairTestBootstrapSource + "\n" + `"$ROOT/bin/pair" title t codex s`, false},
		{"shell reordered", "tests/a.sh", `"$ROOT/bin/pair" title t codex s` + "\n" + pairTestBootstrapSource, true},
		{"comment does not count", "tests/a.sh", "# " + "#!/bin/bash\n" + pairTestBootstrapSource + "\n" + `"$PAIR_BIN" title t codex s`, true},
		{"nested Python", "tests/a.sh", `python3 - <<'PY'
subprocess.run([str(root/'bin/pair'),'title'])
PY`, true},
		{"Python original", "tests/a.py", `subprocess.Popen([binary,'term'],env=os.environ)`, true},
		{"unrelated shell", "tests/a.sh", `git status`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := scriptPairEnvironmentFindings(tc.path, []byte(tc.source))
			if (len(got) > 0) != tc.want {
				t.Fatalf("findings=%v want detected=%v", got, tc.want)
			}
		})
	}
}

func TestPairEnvironmentBootstrapIntegration(t *testing.T) {
	command := exec.Command("bash", "../../../tests/lib/pair-test-bootstrap-test.sh")
	command.Env = Environment(t, t.TempDir())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap isolation: %v\n%s", err, output)
	}
}

func TestPairEnvironmentGuardIndirectTerminalFixture(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         int
	}{
		{"unsafe direct activation", `package fixture;import "os/exec";func f(){c:=exec.Command("couch");c.Env=append(os.Environ(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1");c.Run()}`, 1},
		{"unsafe alias", `package fixture;import "os/exec";func f(){e:=append(os.Environ(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1");alias:=e;c:=exec.Command("couch");c.Env=alias;c.Run()}`, 1},
		{"unsafe helper return", `package fixture;import "os/exec";func environment()[]string{return append(os.Environ(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1")};func f(){c:=exec.Command("couch");c.Env=environment();c.Run()}`, 1},
		{"unsafe helper parameter", `package fixture;import "os/exec";func launch(e []string){c:=exec.Command("couch");c.Env=e;c.Run()};func f(){launch(append(os.Environ(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1"))}`, 1},
		{"unsafe map entries", `package fixture;import "os/exec";func f(){values:=map[string]string{"PAIR_TEST_COUCH_TERMINAL_FIXTURE":"1"};e:=os.Environ();for k,v:=range values{e=append(e,k+"="+v)};c:=exec.Command("couch");c.Env=e;c.Run()}`, 1},
		{"safe helper alias", `package fixture;import "os/exec";import tp "github.com/xianxu/pair/cmd/internal/testpair";func environment()[]string{return append(tp.Environment(t,t.TempDir()),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1")};func f(){e:=environment();c:=exec.Command("couch");c.Env=e;c.Run()}`, 0},
		{"safe explicit entry", `package fixture;import "os/exec";import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){c:=exec.Command("couch");c.Env=tp.Environment(t,t.TempDir(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1");c.Run()}`, 0},
		{"ambient explicit entry", `package fixture;import "os/exec";import tp "github.com/xianxu/pair/cmd/internal/testpair";func f(){e:=append(os.Environ(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1");c:=exec.Command("couch");c.Env=tp.Environment(t,t.TempDir(),e...);c.Run()}`, 1},
		{"disabled fixture", `package fixture;import "os/exec";func f(){c:=exec.Command("couch","--version");c.Env=append(os.Environ(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=0");c.Run()}`, 0},
		{"ordinary couch", `package fixture;import "os/exec";func f(){c:=exec.Command("couch","--version");c.Env=os.Environ();c.Run()}`, 0},
		{"unrelated flag variable", `package fixture;import "os/exec";func f(){unused:=[]string{"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1"};_ = unused;c:=exec.Command("couch","--version");c.Env=os.Environ();c.Run()}`, 0},
		{"build and git unaffected", `package fixture;import "os/exec";func f(){exec.Command("go","build","./cmd/couch").Run();exec.Command("git","init").Run();c:=exec.Command("couch");c.Env=append(os.Environ(),"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1");c.Run()}`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pairEnvironmentFindings("cmd/fixture/indirect_test.go", []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.want {
				t.Fatalf("findings=%v want %d", got, tc.want)
			}
		})
	}
}

func TestPairEnvironmentGuardInstalledCouchFullSource(t *testing.T) {
	source, err := os.ReadFile("../../couch/main_test.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "cmd/couch/main_test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var start, end int
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "installedEnv" {
			start = fset.Position(fn.Pos()).Offset
			end = fset.Position(fn.End()).Offset
		}
	}
	if end == 0 {
		t.Fatal("installed Couch environment helper missing")
	}
	// The actual pre-a64b0c223 denylist, retained without requiring Git history.
	const oldHelper = `func installedEnv(t *testing.T, binDir string) []string {
 t.Helper()
 root,err:=filepath.EvalSymlinks(t.TempDir());if err!=nil{t.Fatal(err)}
 var env []string
 for _,entry:=range os.Environ(){key,_,_:=strings.Cut(entry,"=");if !strings.HasPrefix(key,"PAIR_")&&!strings.HasPrefix(key,"COUCH_")&&!strings.HasPrefix(key,"XDG_"){env=append(env,entry)}}
 return append(env,"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),"COUCH_ISOLATED_ROOT="+root,"HOME="+filepath.Join(root,"home"),"XDG_DATA_HOME="+filepath.Join(root,"data"),"PAIR_DATA_DIR="+filepath.Join(root,"data","pair"),"COUCH_STORE_DIR="+filepath.Join(root,"store"),"COUCH_IDENTITY_DIR="+filepath.Join(root,"identity"))
 }`
	old := append(append(append([]byte{}, source[:start]...), []byte(oldHelper)...), source[end:]...)
	for _, tc := range []struct {
		name   string
		source []byte
		want   int
	}{{"original denylist", old, 1}, {"shared allowlist", source, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pairEnvironmentFindings("cmd/couch/main_test.go", tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.want {
				t.Fatalf("findings=%v want %d (only the terminal-fixture entrypoint)", got, tc.want)
			}
		})
	}
}
