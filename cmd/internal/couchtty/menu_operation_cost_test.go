package couchtty

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// Each scanned element must hit the work counter before any branch can skip it.
// This makes an added nested scan visible to the N/2N execution regression.
func TestMenuOperationWorkCountersCoverEveryScan(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "menu_operations.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || (fn.Name.Name != "menuOperationRowsMeasured" && fn.Name.Name != "pruneMenuOperationsMeasured") {
			continue
		}
		found++
		scans := 0
		views := 0
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok && identNamed(call.Fun, "menuRows") {
				views++
			}
			return true
		})
		if views != 1 {
			t.Errorf("%s builds %d row views, want exactly one", fn.Name.Name, views)
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			var body *ast.BlockStmt
			switch loop := node.(type) {
			case *ast.RangeStmt:
				body = loop.Body
			case *ast.ForStmt:
				body = loop.Body
			}
			if body == nil {
				return true
			}
			scans++
			ast.Inspect(body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				allowed := identNamed(call.Fun, "menuRowKey") || identNamed(call.Fun, "menuOperationTarget")
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					allowed = identNamed(selector.X, "couchcore") && (selector.Sel.Name == "ThreadAddress" || selector.Sel.Name == "ThreadRowKey")
				}
				if !allowed {
					t.Errorf("%s scan calls an uncounted helper; account for its work explicitly", fn.Name.Name)
				}
				return true
			})
			if len(body.List) == 0 {
				t.Errorf("%s has uncounted scan", fn.Name.Name)
				return true
			}
			gate, ok := body.List[0].(*ast.IfStmt)
			if !ok || gate.Else != nil || len(gate.Body.List) != 1 {
				t.Errorf("%s scan must count before branching", fn.Name.Name)
				return true
			}
			cond, ok := gate.Cond.(*ast.BinaryExpr)
			if !ok || cond.Op != token.NEQ || !identNamed(cond.X, "work") || !identNamed(cond.Y, "nil") {
				t.Errorf("%s counter must be nil by default", fn.Name.Name)
				return true
			}
			inc, ok := gate.Body.List[0].(*ast.IncDecStmt)
			if !ok || inc.Tok != token.INC {
				t.Errorf("%s scan needs an increment", fn.Name.Name)
				return true
			}
			field, ok := inc.X.(*ast.SelectorExpr)
			if !ok || !identNamed(field.X, "work") || (field.Sel.Name != "Rows" && field.Sel.Name != "Jobs") {
				t.Errorf("%s scan increments wrong counter", fn.Name.Name)
			}
			return true
		})
		if scans != 2 {
			t.Errorf("%s scans=%d, want one row scan and one job scan", fn.Name.Name, scans)
		}
	}
	if found != 2 {
		t.Fatalf("found %d measured helpers; want projection and pruning", found)
	}
}

func identNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func TestMenuOperationProjectionAndPruningVisitCountsGrowLinearly(t *testing.T) {
	for _, n := range []int{32, 64} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			rows := make([]couchcore.ActionableThreadSummary, n)
			jobs := make(map[menuOperationTarget]MenuOperation, 3*n)
			for i := range rows {
				address := couchcore.ThreadAddress{RepoScope: "scope", Tag: couchcore.ThreadTag(fmt.Sprint(i))}
				rows[i] = couchcore.ActionableThreadSummary{Address: address}
				key := menuOperationTarget{RowKey: menuRowKey(rows[i])}
				jobs[key] = MenuOperation{Origin: MenuOperationOrigin{Attempt: uint64(i + 1)}, Phase: MenuOperationSucceeded, Detail: "older"}
				jobs[menuOperationTarget{StartPath: fmt.Sprintf("/alias/%d", i)}] = MenuOperation{Origin: MenuOperationOrigin{Attempt: uint64(n + i + 1)}, ResultAddress: address, Phase: MenuOperationFailed, Detail: "newest"}
				jobs[menuOperationTarget{StartPath: fmt.Sprintf("/removed/%d", i)}] = MenuOperation{Origin: MenuOperationOrigin{Attempt: uint64(2*n + i + 1)}, Phase: MenuOperationFailed, Detail: "missing"}
			}
			state := NewMenuState(rows, rows[0].Address)
			state.Operations = jobs
			projectionWork := menuOperationWork{}
			projected := menuOperationRowsMeasured(state, &projectionWork)
			pruneWork := menuOperationWork{}
			pruned := pruneMenuOperationsMeasured(state, &pruneWork)
			want := menuOperationWork{Rows: n, Jobs: 3 * n}
			if projectionWork != want || pruneWork != want {
				t.Fatalf("N=%d: projection=%+v prune=%+v want=%+v", n, projectionWork, pruneWork, want)
			}
			if len(projected) != n || len(pruned.Operations) != n {
				t.Fatalf("projection/pruning controls failed: projected=%d retained=%d want=%d", len(projected), len(pruned.Operations), n)
			}
			for _, row := range rows {
				key := menuRowKey(row)
				if projected[key].Detail != "newest" || pruned.Operations[menuOperationTarget{RowKey: key}].Detail != "newest" {
					t.Fatalf("alias result did not supersede older row result: %+v", key)
				}
			}
			if len(state.Operations) != 3*n {
				t.Fatal("pruning mutated input")
			}
			if len(menuOperationRows(state)) != len(projected) || len(pruneMenuOperations(state).Operations) != len(pruned.Operations) {
				t.Fatal("production nil-counter entry differs from measured entry")
			}
		})
	}
}

// The hot path has no observer installation, global counter or state field.
// Its only production entry points explicitly pass nil to the measured core.
func TestMenuOperationProductionCountersAreDisabled(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "menu_operations.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	wrappers := map[string]string{"menuOperationRows": "menuOperationRowsMeasured", "pruneMenuOperations": "pruneMenuOperationsMeasured"}
	found := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		helper, ok := wrappers[fn.Name.Name]
		if !ok {
			continue
		}
		found++
		if len(fn.Body.List) != 1 {
			t.Fatalf("%s must remain a plain nil-counter wrapper", fn.Name.Name)
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			t.Fatalf("%s must return measured core", fn.Name.Name)
		}
		call, ok := ret.Results[0].(*ast.CallExpr)
		if !ok || !identNamed(call.Fun, helper) || len(call.Args) != 2 || !identNamed(call.Args[0], "state") || !identNamed(call.Args[1], "nil") {
			t.Fatalf("%s production work counter must be nil", fn.Name.Name)
		}
	}
	if found != 2 {
		t.Fatalf("production wrappers=%d want2", found)
	}
}
