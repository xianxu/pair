package couchcmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// Scan the whole package so a warning in a new production file cannot escape
// classification. Read only local sources; policy and origin live elsewhere.
func TestNoticeClassInventory(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range noticeClassFindings(name, source) {
			t.Error(finding)
		}
	}
}

func noticeClassFindings(name string, source []byte) []string {
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, name, source, 0)
	if err != nil {
		return []string{err.Error()}
	}
	var findings []string
	scan := func(node ast.Node, function string) {
		ast.Inspect(node, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			method := selector.Sel.Name
			report := func(message string) {
				findings = append(findings, fmt.Sprintf("%s: %s", positions.Position(call.Pos()), message))
			}
			if noticeClassAddsAttention(selector) {
				if !noticeClassAuditedProjection(name, function) {
					report("direct attention addition bypasses the audited routing projection")
				}
				return true
			}
			if method == "setWarning" {
				report("producer uses unclassified setWarning; publish a typed Notice")
				return true
			}
			if method != "Warn" && method != "publishWarningLocked" {
				return true
			}
			if len(call.Args) != 1 {
				report("warning must carry one classified Notice")
				return true
			}
			if literal, ok := call.Args[0].(*ast.CompositeLit); ok {
				classified := false
				for _, element := range literal.Elts {
					field, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := field.Key.(*ast.Ident)
					if !ok || key.Name != "Class" {
						continue
					}
					value, ok := field.Value.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					pkg, ok := value.X.(*ast.Ident)
					classified = ok && pkg.Name == "couchcore" && strings.HasPrefix(value.Sel.Name, "Notice")
				}
				if !classified {
					report("warning Notice lacks an explicit typed Class")
				}
				return true
			}
			if arg, ok := call.Args[0].(*ast.Ident); ok && noticeClassAuditedForward(name, function, method, arg.Name) {
				return true
			}
			report("computed warning requires an audited producer/forwarder")
			return true
		})
	}
	for _, decl := range file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok {
			name := function.Name.Name
			if function.Recv != nil && len(function.Recv.List) == 1 {
				receiver := function.Recv.List[0].Type
				if star, ok := receiver.(*ast.StarExpr); ok {
					receiver = star.X
				}
				if id, ok := receiver.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			scan(function.Body, name)
		} else {
			scan(decl, "")
		}
	}
	return findings
}

// Recognize only the attention list receiver, not unrelated Apply reducers.
func noticeClassAddsAttention(selector *ast.SelectorExpr) bool {
	if selector.Sel.Name == "ObserveAttention" || selector.Sel.Name == "observeAttention" {
		return true
	}
	if selector.Sel.Name != "Apply" {
		return false
	}
	list, ok := selector.X.(*ast.SelectorExpr)
	if !ok || list.Sel.Name != "list" {
		return false
	}
	attention, ok := list.X.(*ast.SelectorExpr)
	return ok && attention.Sel.Name == "attn"
}

func noticeClassAuditedProjection(file, function string) bool {
	switch file + "/" + function {
	case "attention.go/waitWatcher.publishQuestions", "message_service.go/messageService.publishBlocked", "notify_service.go/messageService.handleNotify":
		return true
	}
	return false
}

// These are the existing audited computed paths, not a producer policy table.
// Every other producer must show its Class at the warning call.
func noticeClassAuditedForward(file, function, method, arg string) bool {
	if file == "run.go" && function == "beginConsole" && method == "Warn" && arg == "notice" {
		return true
	}
	if file == "hub_move.go" && function == "messageService.installHubMoves" && method == "Warn" && arg == "warning" {
		return true
	}
	return false
}

func TestNoticeClassInventoryRejectsNewUnclassifiedProducers(t *testing.T) {
	cases := []struct {
		name, source string
		want         int
	}{
		{"new_attention.go", `package couchcmd; func newProducer(){console.ObserveAttention(fact)}`, 1},
		{"new_service_attention.go", `package couchcmd; func newProducer(){service.observeAttention(fact)}`, 1},
		{"new_list_attention.go", `package couchcmd; func newProducer(){c.attn.list=c.attn.list.Apply(fact,now)}`, 1},
		{"new_unrelated_apply.go", `package couchcmd; func newProducer(){state.Apply(event)}`, 0},
		{"new_attention_callback.go", `package couchcmd; func wire(){service.SetAttention(console.ObserveAttention)}`, 0},
		{"new_unclassified.go", `package couchcmd; func newProducer(){ console.Warn(couchtty.Notice{Body:"failure"}) }`, 1},
		{"new_locked.go", `package couchcmd; func newProducer(){ console.publishWarningLocked(couchtty.Notice{Body:"failure"}) }`, 1},
		{"new_helper.go", `package couchcmd; func newProducer(n couchtty.Notice){console.Warn(n)}`, 1},
		{"new_legacy.go", `package couchcmd; func newProducer(){console.setWarning("failure")}`, 1},
		{"new_typed.go", `package couchcmd; func newProducer(){console.Warn(couchtty.Notice{Class:couchcore.NoticeUnknown,Body:"failure"})}`, 0},
		{"new_callback.go", `package couchcmd; func wire(){publish:=console.Warn; use(publish)}`, 0},
		{"new_initializer.go", `package couchcmd; var producer=func(){console.Warn(couchtty.Notice{Body:"failure"})}`, 1},
		{"new_stringclass.go", `package couchcmd; func newProducer(){console.Warn(couchtty.Notice{Class:"child-exited",Body:"failure"})}`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := noticeClassFindings(tc.name, []byte(tc.source)); len(got) != tc.want {
				t.Fatalf("findings=%v want count %d", got, tc.want)
			}
		})
	}
}
