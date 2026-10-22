package repolint

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

func TestDrainStateBeforePublicationRule(t *testing.T) {
	ready := "s.drainOps = ops; s.drainNotices = notices; s.drainExecutor = executor; "
	publish := "message.StartServer(ctx, path, handler); "
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"ready", ready + publish, 0},
		{"published first", publish + ready, 1},
		{"registry late", "s.drainNotices=notices;s.drainExecutor=executor;" + publish + "s.drainOps=ops;", 1},
		{"executor late", "s.drainOps=ops;s.drainNotices=notices;" + publish + "s.drainExecutor=executor;", 1},
		{"missing map", "s.drainOps=ops;s.drainExecutor=executor;" + publish, 1},
		{"nested late callback", ready + publish + "s.drainOps.refreshDrain=refresh;", 1},
		{"closure is not initialization", "_ = func(){" + ready + "};" + publish, 1},
		{"wrong receiver", "other:= &otherService{}; other.drainOps=ops;other.drainNotices=notices;other.drainExecutor=executor;" + publish, 1},
		{"no listener", ready, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package couchcmd\nimport message \"github.com/xianxu/pair/cmd/internal/couchmessage\"\nfunc newMessageService(){s:= &messageService{};" + tc.body + "}")
			repository := newFakeRepository()
			repository.index["cmd/internal/couchcmd/service.go"] = source
			findings, _, _, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || len(findings) != tc.want {
				t.Fatalf("findings=%v error=%v want%d", findings, err, tc.want)
			}
			for _, finding := range findings {
				if finding.Finding.Rule != "drain-state-before-listen" || !strings.Contains(finding.Finding.Message, "pair#722") {
					t.Fatalf("unexpected finding %+v", finding)
				}
			}
		})
	}
}

// This syntax guard covers the owned constructor's literal listener publication.
// Runtime race tests cover concurrent callers; it deliberately does not infer
// arbitrary helper/control-flow publication or prove values nonnil.
func drainStatePublishedBeforeReady(file *ast.File) []ast.Node {
	if file.Name.Name != "couchcmd" {
		return nil
	}
	var findings []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "newMessageService" || fn.Body == nil {
			continue
		}
		var owner *ast.Object
		var listener *ast.CallExpr
		var assignments []*ast.AssignStmt
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if _, ok := node.(*ast.FuncLit); ok {
				return false
			}
			if assignment, ok := node.(*ast.AssignStmt); ok {
				assignments = append(assignments, assignment)
				if owner == nil && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 {
					id, idOK := assignment.Lhs[0].(*ast.Ident)
					address, addressOK := ast.Unparen(assignment.Rhs[0]).(*ast.UnaryExpr)
					if idOK && addressOK && address.Op == token.AND {
						if value, ok := ast.Unparen(address.X).(*ast.CompositeLit); ok {
							if name, ok := value.Type.(*ast.Ident); ok && name.Name == "messageService" {
								owner = id.Obj
							}
						}
					}
				}
			}
			if call, ok := node.(*ast.CallExpr); ok && listener == nil && importedZellijReference(file, ast.Unparen(call.Fun), "github.com/xianxu/pair/cmd/internal/couchmessage", "StartServer") {
				listener = call
			}
			return true
		})
		if listener == nil {
			continue
		}
		ready := map[string]bool{"drainOps": false, "drainNotices": false, "drainExecutor": false}
		late := false
		for _, assignment := range assignments {
			for _, lhs := range assignment.Lhs {
				field := ""
				expr := ast.Unparen(lhs)
				for {
					selector, ok := expr.(*ast.SelectorExpr)
					if !ok {
						break
					}
					field = selector.Sel.Name
					expr = ast.Unparen(selector.X)
				}
				root, ok := expr.(*ast.Ident)
				if !ok || owner == nil || root.Obj != owner {
					continue
				}
				if _, dependency := ready[field]; dependency {
					if assignment.Pos() < listener.Pos() {
						ready[field] = true
					} else {
						late = true
					}
				}
			}
		}
		for _, initialized := range ready {
			if !initialized {
				late = true
			}
		}
		if late {
			findings = append(findings, listener)
		}
	}
	return findings
}
