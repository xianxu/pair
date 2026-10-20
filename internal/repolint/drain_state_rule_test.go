package repolint

import "go/ast"

// Drain phase and error belong to ReduceDrain. Recognize the coordinator by
// type, not receiver or method spelling, so a new helper cannot escape the rule.
// Copies of its state are included: patching a copy before assigning it back is
// the same authority bypass. Whole-state construction and reducer assignment
// remain legal.
func drainStateOverrides(file *ast.File) []ast.Node {
	if file.Name.Name != "couchcmd" {
		return nil
	}
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		aliases := map[*ast.Object]drainAliasKind{}
		// Use parser object identities so a nested variable that shadows the
		// receiver or a state alias does not acquire the outer binding's authority.
		// This is a bounded, conservative alias analysis, not general Go dataflow:
		// retain every recognizable origin across branches and reassignment.
		changed := true
		for changed {
			changed = false
			bind := func(name ast.Expr, kind drainAliasKind) {
				id, ok := ast.Unparen(name).(*ast.Ident)
				if ok && id.Obj != nil && kind&^aliases[id.Obj] != 0 {
					aliases[id.Obj] |= kind
					changed = true
				}
			}
			ast.Inspect(fn, func(node ast.Node) bool {
				switch n := node.(type) {
				case *ast.Field:
					kind := drainAliasType(n.Type)
					for _, name := range n.Names {
						bind(name, kind)
					}
				case *ast.ValueSpec:
					for i, name := range n.Names {
						bind(name, drainAliasType(n.Type))
						if i < len(n.Values) {
							bind(name, drainAliasExpr(n.Values[i], aliases))
						}
					}
				case *ast.AssignStmt:
					for i, lhs := range n.Lhs {
						if i < len(n.Rhs) {
							bind(lhs, drainAliasExpr(n.Rhs[i], aliases))
						}
					}
				}
				return true
			})
		}
		owner := false
		for _, kind := range aliases {
			owner = owner || kind&drainCoordinatorAlias != 0
		}
		if !owner {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range assignment.Lhs {
				switch target := ast.Unparen(lhs).(type) {
				case *ast.SelectorExpr:
					if (target.Sel.Name == "Phase" || target.Sel.Name == "Error") && drainAliasExpr(target.X, aliases)&drainStateAlias != 0 {
						matches = append(matches, lhs)
					}
				case *ast.StarExpr:
					if drainAliasExpr(target.X, aliases)&drainFieldAlias != 0 {
						matches = append(matches, lhs)
					}
				}
			}
			return true
		})
	}
	return matches
}

type drainAliasKind uint8

const (
	drainCoordinatorAlias drainAliasKind = 1 << iota
	drainRecordAlias
	drainStateAlias
	drainFieldAlias
)

func drainAliasType(expr ast.Expr) drainAliasKind {
	if expr == nil {
		return 0
	}
	switch e := ast.Unparen(expr).(type) {
	case *ast.StarExpr:
		return drainAliasType(e.X)
	case *ast.Ident:
		if e.Name == "drainCoordinator" {
			return drainCoordinatorAlias
		}
	case *ast.SelectorExpr:
		if pkg, ok := ast.Unparen(e.X).(*ast.Ident); ok && pkg.Name == "couchcore" && e.Sel.Name == "DrainState" {
			return drainStateAlias
		}
	}
	return 0
}

func drainAliasExpr(expr ast.Expr, aliases map[*ast.Object]drainAliasKind) drainAliasKind {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return aliases[e.Obj]
	case *ast.UnaryExpr:
		return drainAliasExpr(e.X, aliases)
	case *ast.StarExpr:
		return drainAliasExpr(e.X, aliases)
	case *ast.CompositeLit:
		return drainAliasType(e.Type)
	case *ast.SelectorExpr:
		base := drainAliasExpr(e.X, aliases)
		switch {
		case e.Sel.Name == "record" && base&drainCoordinatorAlias != 0:
			return drainRecordAlias
		case e.Sel.Name == "State" && base&drainRecordAlias != 0:
			return drainStateAlias
		case (e.Sel.Name == "Phase" || e.Sel.Name == "Error") && base&drainStateAlias != 0:
			return drainFieldAlias
		}
	case *ast.CallExpr:
		switch callee := ast.Unparen(e.Fun).(type) {
		case *ast.Ident:
			if callee.Name == "new" && len(e.Args) == 1 {
				return drainAliasType(e.Args[0])
			}
		case *ast.SelectorExpr:
			if pkg, ok := ast.Unparen(callee.X).(*ast.Ident); ok && pkg.Name == "couchcore" && callee.Sel.Name == "ReduceDrain" {
				return drainStateAlias
			}
		}
	}
	return 0
}
