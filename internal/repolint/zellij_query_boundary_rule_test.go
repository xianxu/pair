package repolint

import (
	"go/ast"
	"strconv"
	"strings"

	"github.com/xianxu/ariadne/pkg/golint"
)

// Bind filename-sensitive production/diagnostic ownership here without
// exempting entire fixture or consumer packages.
func pairCatalogFor(filename string) []golint.Rule {
	catalog := append([]golint.Rule(nil), pairCatalog...)
	for i := range catalog {
		if catalog[i].Name == "test-fifo-empty-handoff" && strings.HasSuffix(filename, "_test.go") {
			catalog[i].Match = syntaxMatch(emptyFIFOHandoffs)
		}
		if catalog[i].Name == "slash-command-submit" {
			catalog[i].Match = func(file *ast.File, _ *golint.CheckContext) []ast.Node { return slashCommandSubmits(filename, file) }
		}
		if catalog[i].Name == "zellij-query-boundary" {
			catalog[i].Match = func(file *ast.File, _ *golint.CheckContext) []ast.Node {
				return zellijQueryBoundaryBypasses(filename, file)
			}
		}
	}
	return catalog
}

func zellijQueryBoundaryBypasses(filename string, file *ast.File) []ast.Node {
	if strings.HasSuffix(filename, "_test.go") || strings.HasPrefix(filename, "cmd/probes/") || strings.HasPrefix(filename, "probes/") || strings.HasPrefix(filename, "cmd/repro/") {
		return nil
	}
	var matches []ast.Node
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && rawZellijCollectorOwner(filename, fn) {
			continue
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			if importedZellijReference(file, node, "github.com/xianxu/pair/cmd/internal/zellijexec", "Run") {
				matches = append(matches, node)
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			offset := 0
			switch {
			case importedZellijReference(file, ast.Unparen(call.Fun), "os/exec", "Command"):
			case importedZellijReference(file, ast.Unparen(call.Fun), "os/exec", "CommandContext"):
				offset = 1
			default:
				return true
			}
			if len(call.Args) > offset && zellijString(call.Args[offset]) == "zellij" && literalZellijRead(call.Args[offset+1:]) {
				matches = append(matches, call)
			}
			return true
		})
	}
	return matches
}

func rawZellijCollectorOwner(filename string, fn *ast.FuncDecl) bool {
	switch filename {
	case "cmd/internal/zellijquery/query.go":
		return fn.Name.Name == "readAttempt" && zellijReceiver(fn, "Source")
	case "cmd/internal/launcher/zellij_query_evidence.go":
		return fn.Name.Name == "queryZellijVersion" && fn.Recv == nil
	case "cmd/internal/testcouch/startup_evidence.go":
		return fn.Name.Name == "runStartupQuery" && zellijReceiver(fn, "Fleet")
	}
	return false
}

func zellijReceiver(fn *ast.FuncDecl, want string) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	receiver := ast.Unparen(fn.Recv.List[0].Type)
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = ast.Unparen(pointer.X)
	}
	name, ok := receiver.(*ast.Ident)
	return ok && name.Name == want
}

// Import identity plus parser object resolution follows aliases, rejects local
// shadows and handles unresolved dot-import names without type checking.
func importedZellijReference(file *ast.File, node ast.Node, path, member string) bool {
	for _, spec := range file.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil || imported != path {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." {
			id, ok := node.(*ast.Ident)
			if ok && id.Name == member && id.Obj == nil {
				return true
			}
			continue
		}
		sel, ok := node.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != member {
			continue
		}
		id, ok := ast.Unparen(sel.X).(*ast.Ident)
		if ok && id.Name == name && id.Obj == nil {
			return true
		}
	}
	return false
}

func zellijString(expr ast.Expr) string {
	literal, ok := ast.Unparen(expr).(*ast.BasicLit)
	if !ok {
		return ""
	}
	value, _ := strconv.Unquote(literal.Value)
	return value
}

// Recognize the command position, never an arbitrary later argument (a write
// may itself contain "list-panes"). Dynamic sessions are safe to skip by argv
// position; dynamically assembled executables/argv require human review.
func literalZellijRead(args []ast.Expr) bool {
	if len(args) >= 2 && zellijString(args[0]) == "--session" {
		args = args[2:]
	}
	if len(args) == 0 {
		return false
	}
	if zellijString(args[0]) == "list-sessions" {
		return true
	}
	if len(args) < 2 || zellijString(args[0]) != "action" {
		return false
	}
	switch zellijString(args[1]) {
	case "list-clients", "list-panes", "current-tab-info", "query-tab-names", "dump-layout", "dump-screen":
		return true
	}
	return false
}
