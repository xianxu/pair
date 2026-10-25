package repolint

import (
	"go/ast"

	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/golint"
	"github.com/xianxu/pair/internal/reporoot"
)

// #866: direct-syntax inventory, not whole-program dataflow analysis. It follows
// imports, literal executables, direct local executable variables and the named
// Source/adapter seams. Computed names, executor aliases and inter-file factories
// still require review; runtime stateful admission tests remain authoritative.
// Explicit execution classes, not file exemptions. Fixture/qualification
// sites deliberately exercise native Zellij without claiming production admission.
var zellijNativeFixtureOwners = map[string]string{
	"probes/zellijbirthrace/main.go:sessionState":       "native startup race reproducer",
	"probes/zellijbirthrace/main.go:hammer":             "native startup race reproducer",
	"probes/zellijbirthrace/main.go:teardown":           "native reproducer cleanup",
	"probes/zellijpark/main.go:main":                    "interactive native qualification",
	"probes/zellijpark/main.go:sessionState":            "native qualification observation",
	"probes/zellijpark/main.go:cleanup":                 "native qualification cleanup",
	"probes/zellijprobe/zellijprobe.go:Start":           "disposable native qualification",
	"probes/zellijprobe/zellijprobe.go:WaitUntilListed": "disposable native qualification",
	"probes/zellijprobe/zellijprobe.go:Action":          "disposable native qualification",
	"probes/zellijprobe/zellijprobe.go:Close":           "disposable native qualification cleanup",

	"cmd/probes/reattachcost/main.go:run":                                            "disposable native timing qualification",
	"cmd/probes/reattachcost/main.go:startSampler":                                   "disposable native timing qualification",
	"cmd/probes/reattachcost/main.go:attach":                                         "interactive timing qualification",
	"cmd/probes/reattachcost/main.go:allListed":                                      "disposable native timing qualification",
	"cmd/probes/reattachcost/main.go:cotenancy":                                      "diagnostic inventory qualification",
	"cmd/probes/couchnestedrows/main.go:runOuter":                                    "interactive native qualification",
	"cmd/probes/couchnestedrows/main.go:runProbe":                                    "disposable native qualification cleanup",
	"cmd/internal/pairlifecycletest/live_zellij.go:StartControlledZellijWithOptions": "owned native fixture construction",
	"cmd/internal/pairlifecycletest/live_zellij.go:close":                            "owned native fixture retirement",
	"cmd/internal/pairlifecycletest/live_zellij.go:AddOwnerWrapperPane":              "owned native fixture construction",
	"cmd/internal/testcouch/startup_evidence.go:runStartupQuery":                     "owned fleet diagnostics",
}

func zellijNativeAdmissionBypasses(filename string, file *ast.File) []ast.Node {
	if strings.HasSuffix(filename, "_test.go") {
		return nil
	} // native fixture inputs
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, _ := decl.(*ast.FuncDecl)
		nativeNames := map[string]bool{}
		sourceNames := map[string]bool{}
		if fn != nil {
			for _, fields := range []*ast.FieldList{fn.Recv, fn.Type.Params} {
				if fields != nil {
					for _, field := range fields.List {
						typ := ast.Unparen(field.Type)
						if ptr, ok := typ.(*ast.StarExpr); ok {
							typ = ast.Unparen(ptr.X)
						}
						local, ok := typ.(*ast.Ident)
						if ok && file.Name.Name == "zellijquery" && local.Name == "Source" || importedZellijReference(file, typ, "github.com/xianxu/pair/cmd/internal/zellijquery", "Source") {
							for _, name := range field.Names {
								sourceNames[name.Name] = true
							}
						}
					}
				}
			}
		}
		printed := map[ast.Node]bool{}
		ast.Inspect(decl, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				for _, name := range []string{"Printf", "Println", "Print", "Fprintf", "Fprintln"} {
					if importedZellijReference(file, ast.Unparen(call.Fun), "fmt", name) {
						for _, arg := range call.Args {
							printed[arg] = true
						}
					}
				}
			}
			return true
		})
		ast.Inspect(decl, func(node ast.Node) bool {
			if a, ok := node.(*ast.AssignStmt); ok {
				for i, rhs := range a.Rhs {
					if filepath.Base(zellijString(rhs)) == "zellij" && i < len(a.Lhs) {
						if id, ok := a.Lhs[i].(*ast.Ident); ok {
							nativeNames[id.Name] = true
						}
					}
				}
			}
			if a, ok := node.(*ast.ValueSpec); ok {
				for i, rhs := range a.Values {
					if filepath.Base(zellijString(rhs)) == "zellij" && i < len(a.Names) {
						nativeNames[a.Names[i].Name] = true
					}
				}
			}
			return true
		})
		if fn != nil && filename == "cmd/internal/zellijquery/query.go" && fn.Name.Name == "readAttempt" && zellijReceiver(fn, "Source") && !sourceAttemptHasAdmission(fn) {
			matches = append(matches, fn)
		}
		// Discover callbacks owned by the actual imported namespace guard. A call in
		// the same function but outside that callback is still an unguarded escape.
		guarded := map[ast.Node]bool{}
		ast.Inspect(decl, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			qualified := importedZellijReference(file, ast.Unparen(call.Fun), "github.com/xianxu/pair/cmd/internal/zellijquery", "WithProbe") || importedZellijReference(file, ast.Unparen(call.Fun), "github.com/xianxu/pair/cmd/internal/zellijquery", "WithReadyProbe")
			if filename == "cmd/internal/zellijquery/command.go" {
				id, ok := ast.Unparen(call.Fun).(*ast.Ident)
				qualified = qualified || ok && id.Name == "WithProbe"
			}
			if filename == "cmd/internal/zellijquery/command.go" && fn != nil && fn.Name.Name == "RunNativeCommand" {
				id, ok := ast.Unparen(call.Fun).(*ast.Ident)
				qualified = qualified || ok && id.Name == "withNativeAdmission" && len(call.Args) == 4
			}
			if qualified && len(call.Args) >= 3 {
				if cb, ok := ast.Unparen(call.Args[len(call.Args)-1]).(*ast.FuncLit); ok {
					ast.Inspect(cb, func(n ast.Node) bool {
						if n != nil {
							guarded[n] = true
						}
						return true
					})
				}
			}
			return true
		})
		ast.Inspect(decl, func(node ast.Node) bool {
			if guarded[node] {
				return false
			}
			if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "Run" {
				if owner, ok := selector.X.(*ast.Ident); ok && sourceNames[owner.Name] {
					if !(fn != nil && filename == "cmd/internal/zellijquery/query.go" && fn.Name.Name == "readAttempt" && zellijReceiver(fn, "Source")) {
						matches = append(matches, selector)
					}
				}
			}
			if importedZellijReference(file, node, "github.com/xianxu/pair/cmd/internal/zellijexec", "Run") {
				permitted := fn != nil && (zellijNativeFixtureOwners[filename+":"+fn.Name.Name] != "" || filename == "cmd/internal/launcher/zellij_query_evidence.go" && fn.Name.Name == "queryZellijVersion" || filename == "cmd/internal/zellijquery/query.go" && fn.Name.Name == "readAttempt" && sourceAttemptHasAdmission(fn))
				if !permitted {
					matches = append(matches, node)
				}
			}
			if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				value, _ := strconv.Unquote(literal.Value)
				if !printed[literal] && directNativeShell.MatchString(value) {
					matches = append(matches, literal)
				}
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			offset := -1
			if importedZellijReference(file, ast.Unparen(call.Fun), "os/exec", "Command") || importedZellijReference(file, ast.Unparen(call.Fun), "syscall", "Exec") {
				offset = 0
			}
			if importedZellijReference(file, ast.Unparen(call.Fun), "os/exec", "CommandContext") {
				offset = 1
			}
			if offset < 0 || len(call.Args) <= offset || !isNativeExecutable(call.Args[offset], nativeNames) {
				return true
			}
			args := call.Args[offset+1:]
			if fn != nil && filename == "cmd/internal/zellijquery/query.go" && fn.Name.Name == "readAttempt" && zellijReceiver(fn, "Source") && sourceAttemptHasAdmission(fn) && len(args) == 1 {
				path, ok := call.Args[offset].(*ast.Ident)
				argv, argvOK := args[0].(*ast.Ident)
				if ok && argvOK && path.Name == "path" && argv.Name == "args" && call.Ellipsis.IsValid() {
					return true
				}
			}
			// --version does not open a native client or enumerate the namespace.
			if len(args) == 1 && zellijString(args[0]) == "--version" {
				return true
			}
			if fn != nil && zellijNativeFixtureOwners[filename+":"+fn.Name.Name] != "" {
				return true
			}
			// The creator's one interactive client has a different lifetime owner.
			// terminalLaunchArgs carries both create and ordinary attach. The latter is
			// the explicitly accepted uncovered class; never classify the whole file.
			if fn != nil && filename == "cmd/internal/launcher/session_handoff_os.go" && fn.Name.Name == "Start" && zellijReceiver(fn, "terminalOSAttempt") && len(args) == 1 {
				if a, ok := ast.Unparen(args[0]).(*ast.CallExpr); ok {
					if id, ok := ast.Unparen(a.Fun).(*ast.Ident); ok && id.Name == "terminalLaunchArgs" {
						return true
					}
				}
			}
			matches = append(matches, call)
			return true
		})
	}
	return matches
}

func isNativeExecutable(expr ast.Expr, names map[string]bool) bool {
	if filepath.Base(zellijString(expr)) == "zellij" {
		return true
	}
	id, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && names[id.Name]
}

// A source runner injection must occur after the same top-level acquisition,
// error return and deferred release as the real native runner. This deliberately
// pins the small production seam, rather than trusting a Run field or comment.
func sourceAttemptHasAdmission(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	acquired, refuses, release, runner := false, false, false, false
	var releasePosition, firstHook token.Pos
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "Run" {
			if id, ok := selector.X.(*ast.Ident); ok && id.Name == "s" && (firstHook == 0 || selector.Pos() < firstHook) {
				firstHook = selector.Pos()
			}
		}
		return true
	})
	for _, stmt := range fn.Body.List {
		if assignment, ok := stmt.(*ast.AssignStmt); ok {
			for _, rhs := range assignment.Rhs {
				if call, ok := rhs.(*ast.CallExpr); ok {
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "acquire" && len(call.Args) == 2 {
						acquired = true
					}
				}
				if sel, ok := rhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Run" {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == "s" {
						runner = true
						if !acquired || !refuses || !release {
							return false
						}
					}
				}
			}
		}
		if acquired && !runner {
			if conditional, ok := stmt.(*ast.IfStmt); ok {
				if check, ok := conditional.Cond.(*ast.BinaryExpr); ok && check.Op == token.NEQ {
					if id, ok := check.X.(*ast.Ident); ok && id.Name == "err" {
						for _, body := range conditional.Body.List {
							if _, ok := body.(*ast.ReturnStmt); ok {
								refuses = true
							}
						}
					}
				}
			}
		}
		if acquired && !runner {
			if deferred, ok := stmt.(*ast.DeferStmt); ok {
				ast.Inspect(deferred, func(node ast.Node) bool {
					if call, ok := node.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Close" {
							if id, ok := sel.X.(*ast.Ident); ok && id.Name == "lease" {
								release = true
								releasePosition = deferred.End()
							}
						}
					}
					return true
				})
			}
		}
	}
	// Default and explicit recovery must share this one acquisition variable.
	probe, explicit := false, false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok {
			probe = probe || id.Name == "AcquireProbe"
			explicit = explicit || id.Name == "AcquireExplicitCommand"
		}
		return true
	})
	return acquired && refuses && release && runner && probe && explicit && firstHook > releasePosition
}

var directNativeShell = regexp.MustCompile(`(?m)(?:^\s*|[;&|]\s*|["']\s*)(?:exec\s+)?(?:[\w./-]+/)?zellij["']?\s+(?:action\b|run\b|edit\b|list-sessions\b|kill-session\b|delete-session\b|attach\b|--server\b|--config\b|--layout\b|--session\b|-s\b|-c\b|-l\b|["']?\$)`)
var luaNativeExecutor = regexp.MustCompile(`(?:vim\.fn\.(?:system|jobstart|termopen)|vim\.system)\s*\(`)
var luaNativeVariable = regexp.MustCompile(`(?:local\s+)?(\w+)\s*=\s*\{\s*["']zellij["']`)
var luaNativeLiteral = regexp.MustCompile(`^\s*(?:\{\s*)?["'](?:[^"']*/)?zellij(?:["']|\s)`)

// First argument only: quoted prompt data and later arguments cannot classify an
// executable. This lightweight lexical scanner balances nested Lua expressions.
func firstLuaArgument(source string) string {
	depth := 0
	var quote byte
	for i := 0; i < len(source); i++ {
		c := source[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		switch c {
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			if depth == 0 {
				return source[:i]
			}
			depth--
		case ',':
			if depth == 0 {
				return source[:i]
			}
		}
	}
	return source
}

func zellijNativeSourceFindings(filename string, source []byte) []locatedFinding {
	var findings []locatedFinding
	add := func(offset int) {
		findings = append(findings, locatedFinding{filename, golint.Finding{Line: 1 + strings.Count(string(source[:offset]), "\n"), Rule: "zellij-native-admission", Message: "route bounded native argv through Pair internal zellij-command; preserve creator/interactive ownership (#866)"}})
	}
	text := string(source)
	// Preserve offsets when removing whole-line comments.
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") || strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") {
			lines[i] = strings.Repeat(" ", len(line))
		}
	}
	text = strings.Join(lines, "\n")
	if strings.HasSuffix(filename, ".lua") {
		variables := map[string]bool{}
		for _, m := range luaNativeVariable.FindAllStringSubmatch(text, -1) {
			variables[m[1]] = true
		}
		if filename == "nvim/zellij_trace.lua" {
			variables["argv"] = true
		}
		for _, span := range luaNativeExecutor.FindAllStringIndex(text, -1) {
			arg := strings.TrimSpace(firstLuaArgument(text[span[1]:]))
			if strings.HasPrefix(arg, "M.command(") {
				continue
			}
			if luaNativeLiteral.MatchString(arg) || variables[arg] {
				add(span[0])
			}
		}
		if filename == "nvim/review/restore_client.lua" {
			start := strings.Index(text, "function C:run(cmd,opts,callback)")
			if start >= 0 && !strings.HasPrefix(strings.TrimSpace(text[start+len("function C:run(cmd,opts,callback)"):]), "if cmd[1]=='zellij' then cmd=zellij_trace.command(cmd,self.opts.pair_bin()) end") {
				add(start)
			}
		}
	} else {
		for _, span := range directNativeShell.FindAllStringIndex(text, -1) {
			// One documented standalone false-miss reproducer, not a Pair observer.
			if filename == "scripts/repro/zellij-false-miss.sh" {
				start := span[0] + strings.Index(text[span[0]:span[1]], "zellij")
				native := text[start:]
				if strings.HasPrefix(native, `zellij --session "$target" action dump-layout`) || strings.HasPrefix(native, `zellij "$@" > "$out" 2> "$err"`) || strings.HasPrefix(native, `zellij --layout "$root/layout.kdl" attach --create-background "$target"`) {
					continue
				}
			}
			add(span[0])
		}
	}
	return findings
}

func TestTrackedZellijNativeSources(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	paths, err := (gitRepository{root: root}).TrackedPaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		// Scope production shell/layout/editor inputs; tests and standalone diagnostic
		// probes are explicit qualification inputs, not automatic Pair observers.
		if strings.HasPrefix(path, "tests/") || strings.HasPrefix(path, "workshop/") || strings.HasPrefix(path, ".agents/") || strings.HasPrefix(path, ".claude/") || strings.HasPrefix(path, "construct/") || strings.Contains(path, "/assets/") || strings.HasPrefix(path, "examples/") {
			continue // test/teaching inputs and byte-checked generated mirrors
		}
		if strings.Contains(path, "_test.") || strings.Contains(path, "/testdata/") {
			continue
		}
		if !strings.HasSuffix(path, ".lua") && !strings.HasSuffix(path, ".kdl") && !strings.HasSuffix(path, ".sh") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range zellijNativeSourceFindings(path, source) {
			t.Errorf("%s:%s", path, f.Finding.String())
		}
	}
}
