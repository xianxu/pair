package couchtty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// This inventory is deliberately tied to producer call sites, not warning text.
// It makes a new unclassified warning in these producers fail at its source.
func TestNoticeTTYProducerInventory(t *testing.T) {
	files := map[string]map[string]int{
		"console_broadcast.go":         {"NoticeCastStartFailed": 2, "NoticeCastEndedUnexpectedly": 1},
		"console_webcast.go":           {"NoticeChromeStartFailed": 1, "NoticeChromeEndedUnexpectedly": 1, "NoticeChromeRecovery": 1, "NoticeCastDiagnostic": 1},
		"console_castcodes.go":         {"NoticeCastCodeLockout": 1},
		"cast_control.go":              {"NoticeOperatorCapabilityFailed": 1},
		"console_drive.go":             {"NoticeOperatorDriveFailed": 1},
		"console_continuation.go":      {"NoticeContinuationScannerFailed": 1, "NoticeContinuationMissed": 2, "NoticeContinuationFailed": 1},
		"console_lifecycle_failure.go": {"NoticeLifecycleAttemptDiagnostic": 1},
		"trace.go":                     {"NoticeEventTraceFailed": 1},
	}
	for path, want := range files {
		t.Run(path, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]int{}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if selector.Sel.Name == "setWarning" {
					t.Errorf("unclassified setWarning call")
				}
				if selector.Sel.Name != "Warn" && selector.Sel.Name != "publishWarningLocked" {
					return true
				}
				if len(call.Args) != 1 {
					t.Fatal("warning must carry a Notice")
				}
				literal, ok := call.Args[0].(*ast.CompositeLit)
				if !ok {
					t.Fatal("warning must explicitly classify its Notice")
				}
				fields := map[string]ast.Expr{}
				for _, element := range literal.Elts {
					if kv, ok := element.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok {
							fields[key.Name] = kv.Value
						}
					}
				}
				class, ok := fields["Class"].(*ast.SelectorExpr)
				if !ok {
					t.Error("warning lacks typed Class")
					return true
				}
				got[class.Sel.Name]++
				switch path {
				case "console_broadcast.go", "console_webcast.go", "console_castcodes.go", "cast_control.go", "console_drive.go":
					origin, ok := fields["Origin"].(*ast.SelectorExpr)
					if !ok || origin.Sel.Name != "NoticeOperatorOrigin" {
						t.Errorf("%s lacks explicit operator origin", class.Sel.Name)
					}
				}
				return true
			})
			for class, count := range want {
				if got[class] != count {
					t.Errorf("%s count = %d, want %d", class, got[class], count)
				}
			}
			for class := range got {
				if _, ok := want[class]; !ok {
					t.Errorf("unreviewed class %s", class)
				}
			}
		})
	}
}
