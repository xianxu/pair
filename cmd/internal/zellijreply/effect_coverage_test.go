package zellijreply

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type effectBoundary struct{ token, exemption string }

// Enumerate logical effect entry points, not their OS implementations' internal
// steps. A dispatched mutation retains its existing lifetime. Every new sink in
// these owner files must be classified; getters and refusal diagnostics have
// separate explicit exemptions below.
var m2EffectManifest = map[string]effectBoundary{
	"layoutcmd/layoutcmd.go:FocusRightTerminalContext:RunZellijAction#1":     {token: "ctx"},
	"layoutcmd/layoutcmd.go:SwitchRightTerminalTabContext:RunZellijAction#1": {token: "ctx"},
	"layoutcmd/layoutcmd.go:newRightTerminalTabContext:RunZellijAction#1":    {token: "ctx"},
	"layoutcmd/layoutcmd.go:execZellijContext:Output#1":                      {exemption: "read-only query collector, not a mutation"},
	"layoutcmd/layoutcmd.go:RunZellijActionContext:RunNativeCommand#1":       {exemption: "issued Runtime mutation implementation retains its lifetime"},
	"layoutcmd/rightpane.go:restoreTilingContext:RunZellijAction#1":          {token: "ctx"},
	"layoutcmd/rightpane.go:restoreTilingContext:RunZellijAction#2":          {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:Write#1":                 {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:RunZellijAction#1":       {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:RunZellijAction#2":       {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:RunZellijAction#3":       {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:RunZellijAction#4":       {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:RunZellijAction#5":       {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:Clear#1":                 {token: "ctx"},
	"layoutcmd/rightpane.go:RunToggleFocusedContext:NudgeWrap#1":             {token: "ctx"},
	"termcmd/run.go:runDecisionContext:RecordLastLeftPaneID#1":               {token: "ctx"},
	"termcmd/run.go:runDecisionContext:RecordLastTerminalPaneID#1":           {token: "ctx"},
	"termcmd/run.go:runDecisionContext:RunZellijAction#1":                    {token: "ctx"},
	"termcmd/run.go:splitTerminalDownAdmitted:RunZellijActionQuiet#1":        {token: "ctx"},
	"termcmd/run.go:runShellWithExec:RegisterTerminalPane#1":                 {exemption: "startup local registration precedes native admission"},
	"termcmd/run.go:runShellOnHostObserved:newTab#1":                         {exemption: "startup local terminal tab, independent of Zellij observations"},
	"termcmd/run.go:pumpStdinContext:RunZellijAction#1":                      {exemption: "ordinary wheel input, independent of Zellij observations"},
	"termcmd/run.go:pumpStdinContext:RunZellijAction#2":                      {exemption: "ordinary wheel input, independent of Zellij observations"},
	"termcmd/run.go:pumpStdinContext:handle#1":                               {exemption: "ordinary decoded input dispatcher, not admitted effect"},
	"termcmd/run.go:pumpStdinContext:handle#2":                               {exemption: "ordinary decoded input dispatcher, not admitted effect"},
	"termcmd/run.go:handleTerminalChordContext:newTab#1":                     {exemption: "local mux tab action, independent of Zellij observations"},
	"termcmd/run.go:handleTerminalChordContext:closeActive#1":                {exemption: "local mux tab action, independent of Zellij observations"},
	"termcmd/run.go:handleTerminalChordContext:previousTab#1":                {exemption: "local mux tab action, independent of Zellij observations"},
	"termcmd/run.go:handleTerminalChordContext:nextTab#1":                    {exemption: "local mux tab action, independent of Zellij observations"},
	"termcmd/run.go:NudgeWrap:NudgeWrap#1":                                   {exemption: "issued Runtime mutation delegation retains its lifetime"},
	"termcmd/run.go:RecordLastLeftPaneID:Write#1":                            {exemption: "issued Runtime record-write implementation"},
	"termcmd/run.go:RecordLastTerminalPaneID:Write#1":                        {exemption: "issued Runtime record-write implementation"},
	"termcmd/run.go:RunZellijAction:RunZellijActionQuiet#1":                  {exemption: "issued Runtime mutation delegation retains its lifetime"},
	"termcmd/run.go:RunZellijActionQuiet:RunZellijAction#1":                  {exemption: "issued Runtime mutation delegation retains its lifetime"},
	"termcmd/run.go:runZellij:RunNativeCommand#1":                            {exemption: "issued Runtime mutation subprocess retains its lifetime"},
	"draftroute/route.go:RouteLuaToPaneContext:RunZellijAction#1":            {token: "ctx"},
	"draftroute/route.go:RouteLuaToPaneContext:RunZellijAction#2":            {token: "ctx"},
	"clipcmd/run.go:RunCopyOnSelect:ClipboardCopy#1":                         {exemption: "independent clipboard mirror must survive refused auto-quote"},
	"clipcmd/run.go:RunCopyOnSelect:SpawnDetached#1":                         {exemption: "pre-admission hook handoff; child owns fresh admission"},
	"clipcmd/run.go:pasteAdmitted:WriteFile#1":                               {token: "ctx"},
	"clipcmd/run.go:pasteAdmitted:FocusPaneContext#1":                        {token: "ctx"},
	"clipcmd/run.go:pasteAdmitted:WritePaneKeyContext#1":                     {token: "ctx"},
	"clipcmd/run.go:flashAdmitted:SetPaneColor#1":                            {token: "ctx"},
	"clipcmd/run.go:flashAdmitted:ResetPaneColorAfter#1":                     {exemption: "owed cleanup after attempted flash must survive cancellation"},
	"couchtty/console.go:Run:Run#1":                                          {exemption: "existing background operation queue lifecycle, not native admission"},
	"couchtty/console.go:dispatchInputCandidate:route#1":                     {exemption: "already-buffered prefix precedes native authority decision"},
	"couchtty/console.go:dispatchInputCandidate:route#2":                     {token: "ctx"},
	"couchtty/console.go:dispatchInputCandidate:route#3":                     {exemption: "ordinary input independent of restart-role observation"},
	"couchtty/console.go:dispatchInputCandidate:handle#1":                    {token: "admission"},
	"couchtty/console.go:dispatchInputCandidate:handle#2":                    {exemption: "non-observed handler path retains its existing lifecycle"},
}

var m2OwnerFiles = []string{
	"layoutcmd/layoutcmd.go", "layoutcmd/rightpane.go", "termcmd/run.go",
	"draftroute/route.go", "clipcmd/run.go", "couchtty/console.go",
}

var effectSinkNames = strings.Fields("RunZellijAction RunZellijActionQuiet RecordLastLeftPaneID RecordLastTerminalPaneID Write Clear NudgeWrap SetPaneColor ResetPaneColorAfter WriteFile FocusPaneContext WritePaneKeyContext SetNoteAnchor ClipboardCopy SpawnDetached RegisterTerminalPane newTab closeActive previousTab nextTab route handle Run RunNativeCommand Output CombinedOutput Start")

var effectReadMethods = strings.Fields("CurrentPaneID FullscreenStore CurrentTabJSON CurrentTabJSONContext ListPanesJSON ListPanesJSONContext LastLeftPaneID LastTerminalPaneID TerminalPaneIDs CachedDraftPaneID Read TryLock ShellCommand ClipboardPaste ListPanesContext ListClientsContext")
var effectDiagnosticMethods = strings.Fields("Log LogFresh LogFailure NotifyRefusal")

func containsName(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
func callName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.Ident:
		return fun.Name
	}
	return ""
}
func isRuntimeCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	recv, ok := sel.X.(*ast.Ident)
	return ok && (recv.Name == "rt" || recv.Name == "store")
}
func classifiedCall(call *ast.CallExpr) bool {
	n := callName(call)
	return containsName(effectSinkNames, n) || containsName(effectDiagnosticMethods, n) || isRuntimeCall(call)
}

func isAdmissionEffect(call *ast.CallExpr, file *ast.File) bool {
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || fun.Sel.Name != "Effect" {
		return false
	}
	qualifier, ok := fun.X.(*ast.Ident)
	if !ok {
		return false
	}
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil || path != "github.com/xianxu/pair/cmd/internal/zellijreply" {
			continue
		}
		name := "zellijreply"
		if imported.Name != nil {
			name = imported.Name.Name
		}
		return qualifier.Name == name
	}
	return false
}

func inspectEffectCoverage(files map[string]*ast.File, manifest map[string]effectBoundary) []string {
	var failures []string
	seen := map[string]bool{}
	for path, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			parents := map[ast.Node]ast.Node{}
			var stack []ast.Node
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if n == nil {
					stack = stack[:len(stack)-1]
					return false
				}
				if len(stack) > 0 {
					parents[n] = stack[len(stack)-1]
				}
				stack = append(stack, n)
				return true
			})
			ordinals := map[string]int{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if selected, ok := n.(*ast.SelectorExpr); ok && containsName(effectSinkNames, selected.Sel.Name) {
					receiver, runtimeReceiver := selected.X.(*ast.Ident)
					knownMethod := runtimeReceiver && (receiver.Name == "rt" || receiver.Name == "store") || selected.Sel.Name == "SetNoteAnchor"
					call, called := parents[selected].(*ast.CallExpr)
					if knownMethod && (!called || call.Fun != selected) {
						failures = append(failures, fmt.Sprintf("unclassified bound effect reference %s:%s:%s", path, fn.Name.Name, selected.Sel.Name))
					}
				}
				call, ok := n.(*ast.CallExpr)
				if !ok || !classifiedCall(call) {
					return true
				}
				name := callName(call)
				if containsName(effectReadMethods, name) || containsName(effectDiagnosticMethods, name) {
					return true
				}
				ordinals[name]++
				key := fmt.Sprintf("%s:%s:%s#%d", path, fn.Name.Name, name, ordinals[name])
				seen[key] = true
				rule, ok := manifest[key]
				if !ok {
					failures = append(failures, "unclassified "+key)
					return true
				}
				var closure *ast.FuncLit
				for p := parents[call]; p != nil; p = parents[p] {
					if f, ok := p.(*ast.FuncLit); ok {
						closure = f
						break
					}
				}
				var guard *ast.CallExpr
				if closure != nil {
					guard, _ = parents[closure].(*ast.CallExpr)
					if guard != nil && (!isAdmissionEffect(guard, file) || len(guard.Args) != 2 || guard.Args[1] != closure) {
						guard = nil
					}
				}
				if rule.exemption != "" {
					if guard != nil {
						failures = append(failures, "exempt effect unexpectedly guarded "+key)
					}
					return true
				}
				if guard == nil {
					failures = append(failures, "unguarded "+key)
					return true
				}
				ctx, ok := guard.Args[0].(*ast.Ident)
				if !ok || ctx.Name != rule.token {
					failures = append(failures, "wrong admission token "+key)
				}
				calls := 0
				ast.Inspect(closure.Body, func(part ast.Node) bool {
					switch x := part.(type) {
					case *ast.ForStmt, *ast.RangeStmt:
						failures = append(failures, "loop inside effect "+key)
					case *ast.FuncLit:
						failures = append(failures, "nested closure inside effect "+key)
					case *ast.CallExpr:
						if classifiedCall(x) {
							calls++
						}
					}
					return true
				})
				if calls != 1 {
					failures = append(failures, fmt.Sprintf("effect contains %d boundary calls %s", calls, key))
				}
				return true
			})
		}
	}
	for key := range manifest {
		if !seen[key] {
			failures = append(failures, "stale manifest "+key)
		}
	}
	return failures
}

func TestM2EffectsUseExistingAdmissionToken(t *testing.T) {
	files := map[string]*ast.File{}
	for _, path := range m2OwnerFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", path), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files[path] = file
	}
	for _, failure := range inspectEffectCoverage(files, m2EffectManifest) {
		t.Error(failure)
	}
}

func TestEffectCoverageRejectsMissingAndWeakenedGuards(t *testing.T) {
	const sink = "rt.RunZellijAction()"
	for _, tc := range []struct {
		name, body, token, exemption string
		valid                        bool
	}{
		{"guarded", "return zellijreply.Effect(ctx,func()error{return " + sink + "})", "ctx", "", true},
		{"raw", "return " + sink, "ctx", "", false},
		{"new token", "return zellijreply.Effect(context.Background(),func()error{return " + sink + "})", "ctx", "", false},
		{"wrong helper", "return other.Effect(ctx,func()error{return " + sink + "})", "ctx", "", false},
		{"stale entry", "return nil", "ctx", "", false},
		{"loop", "return zellijreply.Effect(ctx,func()error{for i:=0;i<2;i++{_=" + sink + "};return nil})", "ctx", "", false},
		{"cleanup exemption", "return " + sink, "", "owed cleanup", true},
		{"double effect", "return zellijreply.Effect(ctx,func()error{_=" + sink + ";return rt.Clear()})", "ctx", "", false},
		{"unclassified runtime mutation", "return rt.NewMutation()", "ctx", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", "package fixture\nimport \"github.com/xianxu/pair/cmd/internal/zellijreply\"\nfunc run()error{"+tc.body+"}", 0)
			if err != nil {
				t.Fatal(err)
			}
			manifest := map[string]effectBoundary{"fixture.go:run:RunZellijAction#1": {token: tc.token, exemption: tc.exemption}}
			failures := inspectEffectCoverage(map[string]*ast.File{"fixture.go": file}, manifest)
			if (len(failures) == 0) != tc.valid {
				t.Fatalf("coverage failures=%v, valid=%v", failures, tc.valid)
			}
		})
	}
}

func TestEffectCoverageRejectsUnclassifiedSinkReferences(t *testing.T) {
	for _, body := range []string{
		"send:=rt.RunZellijAction;_=send()",
		"_=zellijquery.RunNativeCommand()",
	} {
		file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", "package fixture\nfunc unlisted(){"+body+"}", 0)
		if err != nil {
			t.Fatal(err)
		}
		if failures := inspectEffectCoverage(map[string]*ast.File{"fixture.go": file}, map[string]effectBoundary{}); len(failures) == 0 {
			t.Fatalf("unclassified effect escaped coverage: %s", body)
		}
	}
}
