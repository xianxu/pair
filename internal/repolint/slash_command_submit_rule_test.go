package repolint

import (
	"go/ast"
	"strings"
)

// This syntax guard recognizes literal KeySequence/[]KeyStep values and literal
// slash KeyText followed by KeySubmit, allowing only KeyWait between them. It
// does not follow variables, type aliases, assembled text, or mutated sequences.
// Tests deliberately retain access to the public submit compatibility contract.
func slashCommandSubmits(filename string, file *ast.File) []ast.Node {
	if !strings.HasSuffix(filename, ".go") || strings.HasSuffix(filename, "_test.go") {
		return nil
	}
	var matches []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		sequence, ok := node.(*ast.CompositeLit)
		if !ok || sequence.Type == nil || !slashCommandSequenceType(file, sequence.Type) {
			return true
		}
		slash := false
		for _, element := range sequence.Elts {
			step, ok := element.(*ast.CompositeLit)
			if !ok {
				slash = false
				continue
			}
			var kind ast.Expr
			var text string
			for _, field := range step.Elts {
				entry, ok := field.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := entry.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Kind":
					kind = entry.Value
				case "Text":
					text = zellijString(entry.Value)
				}
			}
			switch {
			case slashCommandReference(file, kind, "KeyText"):
				slash = strings.HasPrefix(text, "/")
			case slashCommandReference(file, kind, "KeyWait"):
				// Waiting preserves the authored command.
			case slashCommandReference(file, kind, "KeySubmit"):
				if slash {
					matches = append(matches, step)
				}
				slash = false
			default:
				slash = false
			}
		}
		return true
	})
	return matches
}

func slashCommandSequenceType(file *ast.File, expr ast.Expr) bool {
	if slice, ok := ast.Unparen(expr).(*ast.ArrayType); ok {
		return slice.Len == nil && slashCommandReference(file, slice.Elt, "KeyStep")
	}
	return slashCommandReference(file, expr, "KeySequence")
}

func slashCommandReference(file *ast.File, expr ast.Expr, name string) bool {
	if expr == nil {
		return false
	}
	expr = ast.Unparen(expr)
	if id, ok := expr.(*ast.Ident); ok && file.Name.Name == "agentdrive" && id.Name == name {
		return true
	}
	return importedZellijReference(file, expr, "github.com/xianxu/pair/cmd/internal/agentdrive", name)
}
