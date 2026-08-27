package artifactpath

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/runtimebundlegen"
)

var productionRoots = []string{"cmd", "bin", "nvim", "zellij", "doctor"}

func constructorAuthorityViolations(classifications []SourceClassification) []string {
	var violations []string
	for _, classification := range classifications {
		if classification.Kind == Constructor && !strings.HasPrefix(classification.Path, "cmd/internal/artifactpath/") {
			violations = append(violations, classification.Path+": constructor authority belongs to cmd/internal/artifactpath")
		}
	}
	sort.Strings(violations)
	return violations
}

func TestManifestCoversRequiredArtifactFamilies(t *testing.T) {
	t.Parallel()

	required := []string{
		"draft", "ledger", "log", "queue", "config", "native-session",
		"pane", "agent", "agent-ready", "agent-pid", "outer-tty",
		"parked", "parked-scrollback", "adapt", "image-capture",
		"continuation", "layout", "restart", "picker", "session-binding",
	}
	seen := make(map[string]bool, len(Families))
	for _, family := range Families {
		if family.Name == "" || family.Token == "" {
			t.Fatalf("invalid artifact family: %+v", family)
		}
		if seen[family.Name] {
			t.Fatalf("duplicate artifact family %q", family.Name)
		}
		seen[family.Name] = true
	}
	for _, name := range required {
		if !seen[name] {
			t.Errorf("artifact manifest omits required family %q", name)
		}
	}
}

func TestResolvedBindingCatalogIsUniqueAndReferencesKnownFamilies(t *testing.T) {
	t.Parallel()

	knownFamilies := make(map[string]bool, len(Families))
	for _, family := range Families {
		knownFamilies[family.Name] = true
	}
	seenNames := make(map[string]bool, len(ResolvedBindings))
	seenWitnesses := make(map[string]bool, len(ResolvedBindings))
	for _, binding := range ResolvedBindings {
		if binding.Name == "" || binding.Family == "" || binding.Resolver == "" || binding.Member == "" {
			t.Fatalf("incomplete resolved binding: %+v", binding)
		}
		if seenNames[binding.Name] {
			t.Fatalf("duplicate resolved binding name %q", binding.Name)
		}
		seenNames[binding.Name] = true
		if !knownFamilies[binding.Family] {
			t.Fatalf("binding %q references unknown family %q", binding.Name, binding.Family)
		}
		witness := binding.Family + "\x00" + binding.Resolver + "\x00" + binding.Member
		if seenWitnesses[witness] {
			t.Fatalf("duplicate resolved binding witness for %s.%s family %s", binding.Resolver, binding.Member, binding.Family)
		}
		seenWitnesses[witness] = true
	}
}

func TestProductionArtifactReferencesAreExactlyClassified(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	familyByName := make(map[string]Family, len(Families))
	for _, family := range Families {
		familyByName[family.Name] = family
	}

	classified := make(map[string]SourceClassification, len(SourceClassifications))
	missing := constructorAuthorityViolations(SourceClassifications)
	for _, classification := range SourceClassifications {
		if _, exists := classified[classification.Path]; exists {
			t.Fatalf("duplicate source classification %q", classification.Path)
		}
		if classification.Kind != Constructor && classification.Kind != ResolvedConsumer && classification.Kind != VocabularyConsumer && classification.Kind != GeneratedMirror {
			t.Fatalf("invalid classification kind for %q: %q", classification.Path, classification.Kind)
		}
		for _, name := range classification.Families {
			if _, ok := familyByName[name]; !ok {
				t.Fatalf("%s classifies unknown family %q", classification.Path, name)
			}
		}
		seenBindings := map[string]bool{}
		for _, name := range classification.BindingNames {
			if seenBindings[name] {
				t.Fatalf("%s repeats resolved binding %q", classification.Path, name)
			}
			seenBindings[name] = true
		}
		seenVocabulary := map[string]bool{}
		for _, allowance := range classification.Vocabulary {
			key := vocabularyKey(allowance)
			if seenVocabulary[key] {
				t.Fatalf("%s repeats vocabulary allowance %+v", classification.Path, allowance)
			}
			seenVocabulary[key] = true
		}
		if classification.Kind != GeneratedMirror {
			if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(classification.Path))); err != nil {
				t.Fatalf("classified source %q is absent: %v", classification.Path, err)
			}
		}
		classified[classification.Path] = classification
	}

	generatedRoot := filepath.Join(t.TempDir(), "runtime")
	if _, err := runtimebundlegen.Generate(runtimebundlegen.GenerateOptions{RepoRoot: repoRoot, OutRoot: generatedRoot}); err != nil {
		t.Fatalf("generate runtime mirror from tracked inputs: %v", err)
	}
	seenGenerated := map[string]bool{}
	if err := filepath.WalkDir(generatedRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(generatedRoot, path)
		if err != nil {
			return err
		}
		logical := filepath.ToSlash(filepath.Join("cmd/internal/runtimebundle/assets/runtime", rel))
		classification, ok := classified[logical]
		if !ok || classification.Kind != GeneratedMirror {
			missing = append(missing, logical+": generated mirror is not exactly classified")
		}
		seenGenerated[logical] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for path, classification := range classified {
		if classification.Kind == GeneratedMirror && !seenGenerated[path] {
			missing = append(missing, path+": generated-mirror classification has no generated output")
		}
	}

	referenceViolations, err := artifactReferenceViolations(repoRoot, productionRoots, classified)
	if err != nil {
		t.Fatal(err)
	}
	missing = append(missing, referenceViolations...)
	inventoryViolations, err := productionSourceInventoryViolations(repoRoot, productionRoots, SourceClassifications, NonArtifactSources)
	if err != nil {
		t.Fatal(err)
	}
	missing = append(missing, inventoryViolations...)
	constructorViolations, err := artifactConstructorViolations(repoRoot, productionRoots, SourceClassifications)
	if err != nil {
		t.Fatal(err)
	}
	missing = append(missing, constructorViolations...)
	if len(missing) != 0 {
		sort.Strings(missing)
		t.Fatalf("unclassified artifact references:\n%s", strings.Join(missing, "\n"))
	}
}

// artifactConstructorViolations prevents an already-classified Go consumer from
// hiding a selected-scope constructor. It examines syntax trees rather than
// lines, so multiline joins and concatenations cannot evade the guard.
// Compatibility-only restart markers are session-cache paths rather than
// composite thread artifacts.
func artifactConstructorViolations(repoRoot string, roots []string, classifications []SourceClassification) ([]string, error) {
	classified := make(map[string]SourceClassification, len(classifications))
	for _, classification := range classifications {
		classified[classification.Path] = classification
	}
	violations := map[string]bool{}
	for _, root := range roots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			classification, ok := classified[rel]
			if !ok || (classification.Kind != ResolvedConsumer && classification.Kind != VocabularyConsumer) || !productionSourceFile(rel, path) {
				return nil
			}
			if filepath.Ext(rel) != ".go" {
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				for _, violation := range nonGoClassificationViolations(rel, string(raw), classification) {
					violations[violation] = true
				}
				return nil
			}
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, path, nil, 0)
			if err != nil {
				return err
			}
			for _, violation := range goClassificationViolations(rel, fileSet, file, classification) {
				violations[violation] = true
			}
			if classification.Kind != ResolvedConsumer {
				return nil
			}
			ast.Inspect(file, func(node ast.Node) bool {
				expr, ok := node.(ast.Expr)
				if !ok {
					return true
				}
				values := []string{}
				if value, exact := constantString(expr); exact {
					values = append(values, value)
				}
				if value, assembled := assembledStringFragments(expr, classification); assembled {
					values = append(values, value)
				}
				for _, value := range values {
					for _, familyName := range classification.Families {
						if familyName == "restart" || familyName == "native-session" {
							continue
						}
						for _, family := range Families {
							if family.Name == familyName && strings.Contains(value, family.Token) && !classificationAllowsVocabulary(classification, familyName, value) {
								line := fileSet.Position(node.Pos()).Line
								violations[fmt.Sprintf("%s:%d: resolved consumer contains %s assembled construction %q", rel, line, familyName, value)] = true
							}
						}
					}
				}
				return true
			})
			for _, violation := range resolvedFunctionAssemblyViolations(rel, fileSet, file, classification) {
				violations[violation] = true
			}
			for _, violation := range resolvedDataflowAssemblyViolations(rel, fileSet, file, classification) {
				violations[violation] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(violations))
	for violation := range violations {
		out = append(out, violation)
	}
	sort.Strings(out)
	return out, nil
}

type stringFlow struct {
	classification SourceClassification
	summaries      map[string]string
	vars           map[*ast.Object]string
	builders       map[*ast.Object]string
	values         []string
	returns        []string
}

func resolvedDataflowAssemblyViolations(rel string, fileSet *token.FileSet, file *ast.File, classification SourceClassification) []string {
	summaries := map[string]string{}
	globals := map[*ast.Object]string{}
	maxIterations := len(file.Decls) + longestFamilyToken() + 1
	for range maxIterations {
		changed := false
		nextGlobals := packageStringValues(file, classification, summaries, globals)
		if !sameObjectStringMap(globals, nextGlobals) {
			globals = nextGlobals
			changed = true
		}
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			flow := newStringFlow(classification, summaries, globals)
			flow.block(function.Body)
			summary := strings.Join(flow.returns, "")
			if summaries[function.Name.Name] != summary {
				summaries[function.Name.Name] = summary
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	var violations []string
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		flow := newStringFlow(classification, summaries, globals)
		flow.block(function.Body)
		for _, value := range flow.values {
			for _, familyName := range classification.Families {
				if familyName == "restart" || familyName == "native-session" {
					continue
				}
				for _, family := range Families {
					if family.Name == familyName && strings.Contains(value, family.Token) {
						line := fileSet.Position(function.Pos()).Line
						violations = append(violations, fmt.Sprintf("%s:%d: resolved consumer function %s dataflow assembles %s vocabulary", rel, line, function.Name.Name, familyName))
					}
				}
			}
		}
	}
	return violations
}

func packageStringValues(file *ast.File, classification SourceClassification, summaries map[string]string, previous map[*ast.Object]string) map[*ast.Object]string {
	flow := newStringFlow(classification, summaries, previous)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST && gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, expression := range valueSpec.Values {
				if i < len(valueSpec.Names) && valueSpec.Names[i].Obj != nil {
					flow.vars[valueSpec.Names[i].Obj] = flow.expression(expression)
				}
			}
		}
	}
	return flow.vars
}

func sameObjectStringMap(left, right map[*ast.Object]string) bool {
	if len(left) != len(right) {
		return false
	}
	for object, value := range left {
		if right[object] != value {
			return false
		}
	}
	return true
}

func longestFamilyToken() int {
	longest := 0
	for _, family := range Families {
		if len(family.Token) > longest {
			longest = len(family.Token)
		}
	}
	return longest
}

func newStringFlow(classification SourceClassification, summaries map[string]string, initial map[*ast.Object]string) *stringFlow {
	flow := &stringFlow{
		classification: classification,
		summaries:      summaries,
		vars:           map[*ast.Object]string{},
		builders:       map[*ast.Object]string{},
	}
	for object, value := range initial {
		flow.vars[object] = value
	}
	return flow
}

func (f *stringFlow) block(block *ast.BlockStmt) {
	for _, statement := range block.List {
		f.statement(statement)
	}
}

func (f *stringFlow) statement(statement ast.Stmt) {
	switch typed := statement.(type) {
	case *ast.AssignStmt:
		for i, rhs := range typed.Rhs {
			value := f.expression(rhs)
			f.record(value)
			if i < len(typed.Lhs) {
				if ident, ok := typed.Lhs[i].(*ast.Ident); ok && ident.Obj != nil {
					f.vars[ident.Obj] = value
				}
			}
		}
	case *ast.DeclStmt:
		gen, ok := typed.Decl.(*ast.GenDecl)
		if !ok {
			return
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, valueExpr := range valueSpec.Values {
				value := f.expression(valueExpr)
				f.record(value)
				if i < len(valueSpec.Names) && valueSpec.Names[i].Obj != nil {
					f.vars[valueSpec.Names[i].Obj] = value
				}
			}
		}
	case *ast.ExprStmt:
		f.record(f.expression(typed.X))
	case *ast.ReturnStmt:
		for _, result := range typed.Results {
			value := f.expression(result)
			f.record(value)
			f.returns = append(f.returns, value)
		}
	case *ast.IfStmt:
		if typed.Init != nil {
			f.statement(typed.Init)
		}
		f.expression(typed.Cond)
		f.block(typed.Body)
		if typed.Else != nil {
			f.statement(typed.Else)
		}
	case *ast.BlockStmt:
		f.block(typed)
	case *ast.ForStmt:
		if typed.Init != nil {
			f.statement(typed.Init)
		}
		if typed.Cond != nil {
			f.expression(typed.Cond)
		}
		f.block(typed.Body)
		if typed.Post != nil {
			f.statement(typed.Post)
		}
	case *ast.RangeStmt:
		f.expression(typed.X)
		f.block(typed.Body)
	case *ast.SwitchStmt:
		if typed.Init != nil {
			f.statement(typed.Init)
		}
		if typed.Tag != nil {
			f.expression(typed.Tag)
		}
		for _, item := range typed.Body.List {
			clause, _ := item.(*ast.CaseClause)
			for _, child := range clause.Body {
				f.statement(child)
			}
		}
	}
}

func (f *stringFlow) expression(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return ""
		}
		value, err := strconv.Unquote(typed.Value)
		if err != nil || classificationAllowsAnyVocabulary(f.classification, value) {
			return ""
		}
		return value
	case *ast.Ident:
		return f.vars[typed.Obj]
	case *ast.ParenExpr:
		return f.expression(typed.X)
	case *ast.BinaryExpr:
		return f.expression(typed.X) + f.expression(typed.Y)
	case *ast.CompositeLit:
		var out strings.Builder
		for _, element := range typed.Elts {
			if keyValue, ok := element.(*ast.KeyValueExpr); ok {
				out.WriteString(f.expression(keyValue.Value))
			} else if value, ok := element.(ast.Expr); ok {
				out.WriteString(f.expression(value))
			}
		}
		return out.String()
	case *ast.CallExpr:
		if selector, ok := typed.Fun.(*ast.SelectorExpr); ok {
			if owner, ok := selector.X.(*ast.Ident); ok && owner.Obj != nil {
				switch selector.Sel.Name {
				case "WriteString":
					for _, arg := range typed.Args {
						f.builders[owner.Obj] += f.expression(arg)
					}
					return ""
				case "String":
					if value, exists := f.builders[owner.Obj]; exists {
						return value
					}
				}
			}
		}
		if function, ok := typed.Fun.(*ast.Ident); ok && f.summaries[function.Name] != "" {
			return f.summaries[function.Name]
		}
		var out strings.Builder
		for _, arg := range typed.Args {
			out.WriteString(f.expression(arg))
		}
		return out.String()
	case *ast.UnaryExpr:
		return f.expression(typed.X)
	case *ast.IndexExpr:
		return f.expression(typed.X) + f.expression(typed.Index)
	default:
		return ""
	}
}

func (f *stringFlow) record(value string) {
	if value != "" {
		f.values = append(f.values, value)
	}
}

func resolvedFunctionAssemblyViolations(rel string, fileSet *token.FileSet, file *ast.File, classification SourceClassification) []string {
	var violations []string
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		var fragments []string
		ast.Inspect(function.Body, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err == nil && !classificationAllowsAnyVocabulary(classification, value) {
				fragments = append(fragments, value)
			}
			return true
		})
		assembled := strings.Join(fragments, "")
		for _, familyName := range classification.Families {
			if familyName == "restart" || familyName == "native-session" {
				continue
			}
			for _, family := range Families {
				if family.Name == familyName && strings.Contains(assembled, family.Token) {
					line := fileSet.Position(function.Pos()).Line
					violations = append(violations, fmt.Sprintf("%s:%d: resolved consumer function %s assembles %s vocabulary across runtime fragments", rel, line, function.Name.Name, familyName))
				}
			}
		}
	}
	return violations
}

const artifactpathImportPath = "github.com/xianxu/pair/cmd/internal/artifactpath"

func goClassificationViolations(rel string, fileSet *token.FileSet, file *ast.File, classification SourceClassification) []string {
	var violations []string
	if classification.Kind == ResolvedConsumer {
		violations = append(violations, resolvedBindingViolations(rel, file, classification)...)
	}
	if classification.Kind == VocabularyConsumer && len(classification.BindingNames) != 0 {
		violations = append(violations, rel+": vocabulary consumer declares resolved bindings")
	}
	if classification.Kind == VocabularyConsumer {
		if alias, imported := importAliases(file)[artifactpathImportPath]; imported {
			violations = append(violations, fmt.Sprintf("%s: vocabulary consumer imports artifactpath as %s", rel, alias))
		}
	}
	violations = append(violations, goVocabularyViolations(rel, fileSet, file, classification)...)
	return violations
}

func resolvedBindingViolations(rel string, file *ast.File, classification SourceClassification) []string {
	definitions := make(map[string]ResolvedBinding, len(ResolvedBindings))
	for _, binding := range ResolvedBindings {
		definitions[binding.Name] = binding
	}
	claimedFamilies := stringSet(classification.Families)
	bindingsByFamily := make(map[string][]ResolvedBinding)
	var violations []string
	for _, name := range classification.BindingNames {
		binding, ok := definitions[name]
		if !ok {
			violations = append(violations, rel+": unknown resolved binding "+name)
			continue
		}
		if !claimedFamilies[binding.Family] {
			violations = append(violations, fmt.Sprintf("%s: binding %s witnesses unclaimed family %s", rel, name, binding.Family))
			continue
		}
		bindingsByFamily[binding.Family] = append(bindingsByFamily[binding.Family], binding)
	}
	for _, family := range classification.Families {
		if len(bindingsByFamily[family]) == 0 {
			violations = append(violations, fmt.Sprintf("%s: resolved family %s has no positive binding", rel, family))
		}
	}

	aliases := importAliases(file)
	artifactAlias := aliases[artifactpathImportPath]
	if artifactAlias == "" {
		if len(classification.BindingNames) != 0 {
			violations = append(violations, rel+": resolved bindings require the artifactpath import")
		}
		return violations
	}
	resultsByResolver := resolverResults(file, artifactAlias)
	parents := astParents(file)
	for _, name := range classification.BindingNames {
		binding, ok := definitions[name]
		if !ok || !claimedFamilies[binding.Family] {
			continue
		}
		results := resultsByResolver[binding.Resolver]
		if len(results) == 0 {
			violations = append(violations, fmt.Sprintf("%s: binding %s resolver %s is not called", rel, name, binding.Resolver))
			continue
		}
		if binding.Member != "" && !resultMemberConsumed(file, parents, results, binding.Member) {
			violations = append(violations, fmt.Sprintf("%s: binding %s member %s is not consumed", rel, name, binding.Member))
		}
	}
	return violations
}

func importAliases(file *ast.File) map[string]string {
	aliases := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		aliases[path] = name
	}
	return aliases
}

func resolverResults(file *ast.File, artifactAlias string) map[string]map[*ast.Object]bool {
	results := map[string]map[*ast.Object]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok {
				continue
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			owner, ownerOK := selectorOwner(selector)
			if !ok || !ownerOK || i >= len(assign.Lhs) {
				continue
			}
			result, ok := assign.Lhs[i].(*ast.Ident)
			if !ok || result.Name == "_" || result.Obj == nil {
				continue
			}
			resolver := ""
			if owner == artifactAlias {
				resolver = selector.Sel.Name
			} else {
				ownerIdent, _ := selector.X.(*ast.Ident)
				for candidate, objects := range results {
					if ownerIdent != nil && objects[ownerIdent.Obj] {
						resolver = candidate
						break
					}
				}
			}
			if resolver == "" {
				continue
			}
			if results[resolver] == nil {
				results[resolver] = map[*ast.Object]bool{}
			}
			results[resolver][result.Obj] = true
		}
		return true
	})
	return results
}

func selectorOwner(selector *ast.SelectorExpr) (string, bool) {
	if selector == nil {
		return "", false
	}
	owner, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	return owner.Name, true
}

func resultMemberConsumed(file *ast.File, parents map[ast.Node]ast.Node, results map[*ast.Object]bool, member string) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return !found
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return !found
		}
		owner, ownerOK := selector.X.(*ast.Ident)
		if ownerOK && owner.Obj != nil && results[owner.Obj] && selector.Sel.Name == member && callResultConsumed(parents, call) {
			found = true
		}
		return !found
	})
	return found
}

func callResultConsumed(parents map[ast.Node]ast.Node, call *ast.CallExpr) bool {
	switch parent := parents[call].(type) {
	case *ast.ExprStmt:
		return false
	case *ast.AssignStmt:
		if len(parent.Rhs) == len(parent.Lhs) {
			for i, rhs := range parent.Rhs {
				if rhs == call {
					ident, blank := parent.Lhs[i].(*ast.Ident)
					return !blank || ident.Name != "_"
				}
			}
		}
		for _, lhs := range parent.Lhs {
			if ident, ok := lhs.(*ast.Ident); !ok || ident.Name != "_" {
				return true
			}
		}
		return false
	case *ast.ValueSpec:
		for _, name := range parent.Names {
			if name.Name != "_" {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func goVocabularyViolations(rel string, fileSet *token.FileSet, file *ast.File, classification SourceClassification) []string {
	allowances := make(map[string]VocabularyAllowance)
	seen := make(map[string]int)
	relevantFamilies := stringSet(classification.Families)
	parents := astParents(file)
	var violations []string
	for _, allowance := range classification.Vocabulary {
		if allowance.Context != GoStructTagVocabulary && allowance.Context != GoCallArgumentVocabulary &&
			allowance.Context != GoCaseValueVocabulary && allowance.Context != GoComparisonVocabulary || allowance.Count <= 0 {
			violations = append(violations, fmt.Sprintf("%s: invalid Go vocabulary allowance %+v", rel, allowance))
			continue
		}
		allowances[vocabularyKey(allowance)] = allowance
		relevantFamilies[allowance.Family] = true
	}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		for _, family := range Families {
			if !relevantFamilies[family.Name] || !strings.Contains(value, family.Token) {
				continue
			}
			context, use, argument, ok := vocabularyUseForLiteral(file, parents, literal)
			if !ok {
				line := fileSet.Position(literal.Pos()).Line
				violations = append(violations, fmt.Sprintf("%s:%d: %s vocabulary %q is not at a closed use site", rel, line, family.Name, value))
				continue
			}
			candidate := VocabularyAllowance{Family: family.Name, Value: value, Context: context, Use: use, Argument: argument}
			key := vocabularyKey(candidate)
			if _, ok := allowances[key]; !ok {
				line := fileSet.Position(literal.Pos()).Line
				violations = append(violations, fmt.Sprintf("%s:%d: unlisted %s vocabulary %q at %s %s", rel, line, family.Name, value, context, use))
				continue
			}
			seen[key]++
		}
		return true
	})
	violations = append(violations, vocabularyAllowanceCoverage(rel, classification, allowances, seen)...)
	return violations
}

var permittedVocabularyCallees = map[string]bool{
	"builtin.append":   true,
	"errors.New":       true,
	"flag.NewFlagSet":  true,
	"fmt.Fprintf":      true,
	"fmt.Sprintf":      true,
	"function.logf":    true,
	"method.Log":       true,
	"method.traceWrap": true,
	"os/exec.Command":  true,
}

func astParents(root ast.Node) map[ast.Node]ast.Node {
	parents := map[ast.Node]ast.Node{}
	var stack []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) != 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func vocabularyUseForLiteral(file *ast.File, parents map[ast.Node]ast.Node, literal *ast.BasicLit) (VocabularyContext, string, int, bool) {
	aliases := importAliases(file)
	aliasPaths := map[string]string{}
	for path, alias := range aliases {
		aliasPaths[alias] = path
	}
	for node := ast.Node(literal); node != nil; node = parents[node] {
		switch parent := parents[node].(type) {
		case *ast.Field:
			if parent.Tag == literal && len(parent.Names) != 0 {
				return GoStructTagVocabulary, parent.Names[0].Name, 0, true
			}
		case *ast.CallExpr:
			for argument, expr := range parent.Args {
				if literal.Pos() < expr.Pos() || literal.End() > expr.End() {
					continue
				}
				callee := canonicalCallee(parent.Fun, aliasPaths)
				if permittedVocabularyCallees[callee] {
					return GoCallArgumentVocabulary, callee, argument, true
				}
				return "", "", 0, false
			}
		case *ast.CaseClause:
			for _, expr := range parent.List {
				if literal.Pos() >= expr.Pos() && literal.End() <= expr.End() {
					return GoCaseValueVocabulary, enclosingFunction(parents, parent), 0, true
				}
			}
		case *ast.BinaryExpr:
			if parent.Op == token.EQL || parent.Op == token.NEQ {
				return GoComparisonVocabulary, enclosingFunction(parents, parent), 0, true
			}
		}
	}
	return "", "", 0, false
}

func canonicalCallee(expr ast.Expr, aliasPaths map[string]string) string {
	switch callee := expr.(type) {
	case *ast.Ident:
		if callee.Name == "append" {
			return "builtin.append"
		}
		return "function." + callee.Name
	case *ast.SelectorExpr:
		owner, ok := callee.X.(*ast.Ident)
		if ok {
			if path := aliasPaths[owner.Name]; path != "" {
				return path + "." + callee.Sel.Name
			}
		}
		return "method." + callee.Sel.Name
	default:
		return ""
	}
}

func enclosingFunction(parents map[ast.Node]ast.Node, node ast.Node) string {
	for current := node; current != nil; current = parents[current] {
		if function, ok := current.(*ast.FuncDecl); ok {
			return function.Name.Name
		}
	}
	return ""
}

func nonGoClassificationViolations(rel, body string, classification SourceClassification) []string {
	allowances := make(map[string]VocabularyAllowance)
	seen := make(map[string]int)
	relevantFamilies := stringSet(classification.Families)
	var violations []string
	for _, allowance := range classification.Vocabulary {
		if allowance.Context != ExactLineVocabulary || allowance.Count <= 0 {
			violations = append(violations, fmt.Sprintf("%s: invalid exact-line vocabulary allowance %+v", rel, allowance))
			continue
		}
		allowances[vocabularyKey(allowance)] = allowance
		relevantFamilies[allowance.Family] = true
	}
	for lineNumber, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		for _, family := range Families {
			if !relevantFamilies[family.Name] || !artifactTokenMentioned(trimmed, family.Token) {
				continue
			}
			key := vocabularyKey(VocabularyAllowance{Family: family.Name, Value: trimmed, Context: ExactLineVocabulary})
			if _, ok := allowances[key]; !ok {
				violations = append(violations, fmt.Sprintf("%s:%d: unlisted %s vocabulary line", rel, lineNumber+1, family.Name))
				continue
			}
			seen[key]++
		}
	}
	violations = append(violations, vocabularyAllowanceCoverage(rel, classification, allowances, seen)...)
	return violations
}

func vocabularyAllowanceCoverage(rel string, classification SourceClassification, allowances map[string]VocabularyAllowance, seen map[string]int) []string {
	var violations []string
	allowedFamilies := map[string]bool{}
	for key, allowance := range allowances {
		allowedFamilies[allowance.Family] = true
		if seen[key] != allowance.Count {
			violations = append(violations, fmt.Sprintf("%s: vocabulary %s %q count = %d, want %d", rel, allowance.Family, allowance.Value, seen[key], allowance.Count))
		}
	}
	if classification.Kind == VocabularyConsumer {
		claimed := stringSet(classification.Families)
		for family := range claimed {
			if !allowedFamilies[family] {
				violations = append(violations, fmt.Sprintf("%s: vocabulary family %s has no allowance", rel, family))
			}
		}
		for family := range allowedFamilies {
			if !claimed[family] {
				violations = append(violations, fmt.Sprintf("%s: vocabulary allowance family %s is unclaimed", rel, family))
			}
		}
	}
	return violations
}

func classificationAllowsVocabulary(classification SourceClassification, family, value string) bool {
	for _, allowance := range classification.Vocabulary {
		if allowance.Family == family && allowance.Value == value {
			return true
		}
	}
	return false
}

func vocabularyKey(allowance VocabularyAllowance) string {
	return allowance.Family + "\x00" + string(allowance.Context) + "\x00" + allowance.Value + "\x00" + allowance.Use + "\x00" + strconv.Itoa(allowance.Argument)
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func TestProductionSourceFileRecognizesExtensionlessShebang(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pair-notify")
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\necho outer-tty-work\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !productionSourceFile("bin/pair-notify", path) {
		t.Fatal("extensionless shebang source escaped production coverage")
	}
}

func TestArtifactConstructorAuthorityRejectsProductionCommandMutations(t *testing.T) {
	t.Parallel()

	for _, rel := range []string{
		"cmd/pair-go/main.go",
		"cmd/internal/launcher/mutation.go",
	} {
		t.Run(rel, func(t *testing.T) {
			repoRoot := t.TempDir()
			path := filepath.Join(repoRoot, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("package mutation\nvar artifact = \"draft-\"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			references, err := artifactReferenceViolations(repoRoot, []string{"cmd"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(references) != 1 || !strings.Contains(references[0], rel) {
				t.Fatalf("production mutation escaped complete source scan: %v", references)
			}

			classifications := []SourceClassification{{
				Path: rel, Kind: Constructor, Families: []string{"draft"},
			}}
			violations := constructorAuthorityViolations(classifications)
			if len(violations) != 1 || !strings.Contains(violations[0], rel) {
				t.Fatalf("external constructor mutation escaped authority guard: %v", violations)
			}
		})
	}
}

func TestResolvedConsumerClassificationCannotHideConstructorMutations(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	mutations := map[string]string{
		"cmd/pair-go/concat.go":                  "package mutation\nfunc bad(tag string) string { return \"draft-\" + tag + \".md\" }\n",
		"cmd/pair-go/format.go":                  "package mutation\nimport \"fmt\"\nfunc bad(tag string) string { return fmt.Sprintf(\"draft-%s.md\", tag) }\n",
		"cmd/pair-go/join.go":                    "package mutation\nimport \"strings\"\nfunc bad(tag string) string { return strings.Join([]string{\"draft-\", tag, \".md\"}, \"\") }\n",
		"cmd/internal/launcher/builder.go":       "package mutation\nfunc bad(b interface{ WriteString(string) }, tag string) { b.WriteString(\"draft-\") }\n",
		"cmd/internal/launcher/replace.go":       "package mutation\nimport \"strings\"\nfunc bad(tag string) string { return strings.ReplaceAll(\"draft-{tag}.md\", \"{tag}\", tag) }\n",
		"cmd/internal/launcher/helper.go":        "package mutation\nfunc bad(tag string) string { return artifactName(\"draft-\", tag) }\n",
		"cmd/internal/launcher/filepath_join.go": "package mutation\nimport \"path/filepath\"\nfunc bad(scope string) string { return filepath.Join(scope, \"draft-*.md\") }\n",
	}
	classifications := make([]SourceClassification, 0, len(mutations))
	for rel, body := range mutations {
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		classifications = append(classifications, SourceClassification{
			Path: rel, Kind: ResolvedConsumer, Families: []string{"draft"},
		})
	}

	violations, err := artifactConstructorViolations(repoRoot, []string{"cmd"}, classifications)
	if err != nil {
		t.Fatal(err)
	}
	for rel := range mutations {
		found := false
		for _, violation := range violations {
			if strings.HasPrefix(violation, rel+":") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("false resolved-consumer classification %s escaped constructor guard: %v", rel, violations)
		}
	}
}

func TestResolvedConsumerRequiresFamilyCorrelatedBinding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		body          string
		families      []string
		bindingNames  []string
		wantViolation bool
	}{
		{
			name:          "split token without binding",
			body:          "package mutation\nfunc bad(tag string) string { return \"dra\" + \"ft-\" + tag + \".md\" }\n",
			families:      []string{"draft"},
			wantViolation: true,
		},
		{
			name: "wrong family binding",
			body: "package mutation\nimport \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"func bad(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.ScrollbackPending() }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-scrollback-pending"},
			wantViolation: true,
		},
		{
			name: "resolver reference is not a call",
			body: "package mutation\nimport \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"func bad() { _ = artifactpath.ResolveScoped }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "resolver call without family member",
			body: "package mutation\nimport \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"func bad(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.ScrollbackPending() }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "discarded family member call",
			body: "package mutation\nimport \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"func bad(root, tag string) { paths, _ := artifactpath.ResolveScoped(root, tag); paths.Draft() }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "blank-assigned family member call",
			body: "package mutation\nimport \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"func bad(root, tag string) { paths, _ := artifactpath.ResolveScoped(root, tag); _ = paths.Draft() }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "shadowed resolver result",
			body: "package mutation\nimport \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"type fake struct{}\nfunc (fake) Draft() string { return \"notes.md\" }\n" +
				"func bad(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); _ = paths; return func(paths fake) string { return paths.Draft() }(fake{}) }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "valid witness beside split-token constructor",
			body: "package mutation\nimport (\"path/filepath\"; \"github.com/xianxu/pair/cmd/internal/artifactpath\")\n" +
				"func good(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.Draft() }\n" +
				"func bad(root, tag string) string { return filepath.Join(root, \"dra\" + \"ft-\" + tag + \".md\") }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "valid witness beside runtime-assembled constructor",
			body: "package mutation\nimport (\"path/filepath\"; \"strings\"; \"github.com/xianxu/pair/cmd/internal/artifactpath\")\n" +
				"func good(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.Draft() }\n" +
				"func bad(root, tag string) string { prefix := strings.Join([]string{\"dra\", \"ft-\"}, \"\"); return filepath.Join(root, prefix + tag + \".md\") }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "valid witness beside builder-assembled constructor",
			body: "package mutation\nimport (\"path/filepath\"; \"strings\"; \"github.com/xianxu/pair/cmd/internal/artifactpath\")\n" +
				"func good(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.Draft() }\n" +
				"func bad(root, tag string) string { var b strings.Builder; b.WriteString(\"dra\"); b.WriteString(\"ft-\"); return filepath.Join(root, b.String() + tag + \".md\") }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "valid witness beside reversed-definition constructor",
			body: "package mutation\nimport (\"path/filepath\"; \"github.com/xianxu/pair/cmd/internal/artifactpath\")\n" +
				"func good(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.Draft() }\n" +
				"func bad(root, tag string) string { tail := \"ft-\"; head := \"dra\"; return filepath.Join(root, head + tail + tag + \".md\") }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "valid witness beside cross-helper constructor",
			body: "package mutation\nimport (\"path/filepath\"; \"github.com/xianxu/pair/cmd/internal/artifactpath\")\n" +
				"func good(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.Draft() }\n" +
				"func head() string { return \"dra\" }\nfunc tail() string { return \"ft-\" }\n" +
				"func bad(root, tag string) string { return filepath.Join(root, head() + tail() + tag + \".md\") }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "valid witness beside package-constant constructor",
			body: "package mutation\nimport (\"path/filepath\"; \"github.com/xianxu/pair/cmd/internal/artifactpath\")\n" +
				"const draftTail = \"ft-\"\nconst draftHead = \"dra\"\nconst draftPrefix = draftHead + draftTail\n" +
				"func good(root, tag string) string { paths, _ := artifactpath.ResolveScoped(root, tag); return paths.Draft() }\n" +
				"func bad(root, tag string) string { return filepath.Join(root, draftPrefix + tag + \".md\") }\n",
			families:      []string{"draft"},
			bindingNames:  []string{"scoped-draft"},
			wantViolation: true,
		},
		{
			name: "resolver and family member",
			body: "package mutation\nimport ap \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"func good(root, tag string) string { paths, _ := ap.ResolveScoped(root, tag); return paths.Draft() }\n",
			families:     []string{"draft"},
			bindingNames: []string{"scoped-draft"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			rel := "cmd/pair-go/mutation.go"
			path := filepath.Join(repoRoot, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			violations, err := artifactConstructorViolations(repoRoot, []string{"cmd"}, []SourceClassification{{
				Path: rel, Kind: ResolvedConsumer, Families: tc.families, BindingNames: tc.bindingNames,
			}})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(violations) != 0; got != tc.wantViolation {
				t.Fatalf("violation = %v, want %v: %v", got, tc.wantViolation, violations)
			}
		})
	}
}

func TestProductionSourceInventoryIsIndependentOfArtifactTokens(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for rel, body := range map[string]string{
		"cmd/pair-go/plain.go": "package mutation\nfunc okay() {}\n",
		"cmd/pair-go/split.go": "package mutation\nfunc bad(tag string) string { return \"dra\" + \"ft-\" + tag + \".md\" }\n",
	} {
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	violations, err := productionSourceInventoryViolations(repoRoot, []string{"cmd"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("unlisted production sources = %v, want both plain and split-token files", violations)
	}

	violations, err = productionSourceInventoryViolations(repoRoot, []string{"cmd"}, nil, []string{"cmd/pair-go/plain.go", "cmd/pair-go/split.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], "split.go") {
		t.Fatalf("split-token source escaped the non-artifact contract: %v", violations)
	}
}

func TestVocabularyConsumerContract(t *testing.T) {
	t.Parallel()

	allowSessionID := VocabularyAllowance{
		Family: "native-session", Value: `json:"session_id"`, Context: GoStructTagVocabulary, Use: "ID", Count: 1,
	}
	cases := []struct {
		name          string
		body          string
		allowances    []VocabularyAllowance
		wantViolation bool
	}{
		{
			name:       "exact vocabulary only",
			body:       "package mutation\ntype record struct { ID string `json:\"session_id\"` }\n",
			allowances: []VocabularyAllowance{allowSessionID},
		},
		{
			name: "unrelated caller supplied path IO",
			body: "package mutation\nimport \"os\"\n" +
				"type record struct { ID string `json:\"session_id\"` }\n" +
				"func read(path string) { _, _ = os.ReadFile(path) }\n",
			allowances: []VocabularyAllowance{allowSessionID},
		},
		{
			name: "artifactpath import requires resolved classification",
			body: "package mutation\nimport _ \"github.com/xianxu/pair/cmd/internal/artifactpath\"\n" +
				"type record struct { ID string `json:\"session_id\"` }\n",
			allowances:    []VocabularyAllowance{allowSessionID},
			wantViolation: true,
		},
		{
			name:          "unlisted vocabulary",
			body:          "package mutation\nconst field = \"session_id\"\n",
			wantViolation: true,
		},
		{
			name: "path construction is ineligible",
			body: "package mutation\nimport \"path/filepath\"\n" +
				"func bad(root string) string { return filepath.Join(root, \"session_id\") }\n",
			allowances: []VocabularyAllowance{{
				Family: "native-session", Value: "session_id", Context: GoCallArgumentVocabulary,
				Use: "path/filepath.Join", Argument: 1, Count: 1,
			}},
			wantViolation: true,
		},
		{
			name: "file read is ineligible",
			body: "package mutation\nimport \"os\"\n" +
				"func bad() { _, _ = os.ReadFile(\"session_id\") }\n",
			allowances: []VocabularyAllowance{{
				Family: "native-session", Value: "session_id", Context: GoCallArgumentVocabulary,
				Use: "os.ReadFile", Argument: 0, Count: 1,
			}},
			wantViolation: true,
		},
		{
			name: "local laundering is ineligible",
			body: "package mutation\nimport \"os\"\n" +
				"func bad() { field := \"session_id\"; _, _ = os.ReadFile(field) }\n",
			allowances: []VocabularyAllowance{{
				Family: "native-session", Value: "session_id", Context: GoCallArgumentVocabulary,
				Use: "os.ReadFile", Argument: 0, Count: 1,
			}},
			wantViolation: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			rel := "cmd/pair-go/vocabulary.go"
			path := filepath.Join(repoRoot, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			violations, err := artifactConstructorViolations(repoRoot, []string{"cmd"}, []SourceClassification{{
				Path: rel, Kind: VocabularyConsumer, Families: []string{"native-session"}, Vocabulary: tc.allowances,
			}})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(violations) != 0; got != tc.wantViolation {
				t.Fatalf("violation = %v, want %v: %v", got, tc.wantViolation, violations)
			}
		})
	}
}

func artifactReferenceViolations(repoRoot string, roots []string, classified map[string]SourceClassification) ([]string, error) {
	var violations []string
	for _, root := range roots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if !productionSourceFile(rel, path) {
				return nil
			}
			found, err := artifactFamiliesInFile(path)
			if err != nil {
				return err
			}
			if len(found) == 0 {
				return nil
			}
			classification, ok := classified[rel]
			if !ok {
				violations = append(violations, fmt.Sprintf("%s: %s", rel, strings.Join(found, ", ")))
				return nil
			}
			declared := make(map[string]bool, len(classification.Families))
			for _, name := range classification.Families {
				declared[name] = true
			}
			for _, allowance := range classification.Vocabulary {
				declared[allowance.Family] = true
			}
			for _, name := range found {
				if !declared[name] {
					violations = append(violations, fmt.Sprintf("%s: missing family %s", rel, name))
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(violations)
	return violations, nil
}

// productionSourceInventoryViolations makes source participation independent
// of artifact-token discovery. Every production source is either an explicit
// artifact classification or an explicit non-artifact source; a newly added
// file cannot inherit an implicit "irrelevant" default.
func productionSourceInventoryViolations(repoRoot string, roots []string, classifications []SourceClassification, nonArtifactSources []string) ([]string, error) {
	declared := make(map[string]string, len(classifications)+len(nonArtifactSources))
	var violations []string
	for _, classification := range classifications {
		declared[classification.Path] = "artifact classification"
	}
	for _, rel := range nonArtifactSources {
		if prior := declared[rel]; prior != "" {
			violations = append(violations, fmt.Sprintf("%s: both %s and non-artifact source", rel, prior))
			continue
		}
		declared[rel] = "non-artifact source"
	}

	seen := map[string]bool{}
	for _, root := range roots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if !productionSourceFile(rel, path) {
				return nil
			}
			seen[rel] = true
			if declared[rel] == "" {
				violations = append(violations, rel+": production source is absent from the exhaustive inventory")
			} else if declared[rel] == "non-artifact source" {
				sourceViolations, err := nonArtifactSourceViolations(rel, path)
				if err != nil {
					return err
				}
				violations = append(violations, sourceViolations...)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, rel := range nonArtifactSources {
		if !seen[rel] {
			violations = append(violations, rel+": non-artifact source inventory entry is absent")
		}
	}
	sort.Strings(violations)
	return violations, nil
}

func nonArtifactSourceViolations(rel, path string) ([]string, error) {
	if filepath.Ext(rel) != ".go" {
		families, err := artifactFamiliesInFile(path)
		if err != nil {
			return nil, err
		}
		if len(families) == 0 {
			return nil, nil
		}
		return []string{fmt.Sprintf("%s: non-artifact source contains %s vocabulary", rel, strings.Join(families, ", "))}, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		value, ok := constantString(expr)
		if !ok {
			return true
		}
		for _, family := range Families {
			if strings.HasPrefix(value, family.Token) || artifactTokenMentioned(value, family.Token) {
				seen[family.Name] = true
			}
		}
		return true
	})
	if len(seen) == 0 {
		return nil, nil
	}
	families := make([]string, 0, len(seen))
	for family := range seen {
		families = append(families, family)
	}
	sort.Strings(families)
	return []string{fmt.Sprintf("%s: non-artifact source contains constant %s vocabulary", rel, strings.Join(families, ", "))}, nil
}

func constantString(expr ast.Expr) (string, bool) {
	switch value := expr.(type) {
	case *ast.BasicLit:
		if value.Kind != token.STRING {
			return "", false
		}
		unquoted, err := strconv.Unquote(value.Value)
		return unquoted, err == nil
	case *ast.ParenExpr:
		return constantString(value.X)
	case *ast.BinaryExpr:
		if value.Op != token.ADD {
			return "", false
		}
		left, leftOK := constantString(value.X)
		right, rightOK := constantString(value.Y)
		if !leftOK || !rightOK {
			return "", false
		}
		return left + right, true
	default:
		return "", false
	}
}

// assembledStringFragments conservatively models runtime string constructors:
// every literal fragment under one call expression is concatenated in source
// order. This covers joins, formatters, replacers, builders passed through a
// helper, and filepath calls without maintaining a parallel constructor list.
func assembledStringFragments(expr ast.Expr, classification SourceClassification) (string, bool) {
	if _, ok := expr.(*ast.CallExpr); !ok {
		return "", false
	}
	var fragments []string
	ast.Inspect(expr, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err == nil && !classificationAllowsAnyVocabulary(classification, value) {
			fragments = append(fragments, value)
		}
		return true
	})
	if len(fragments) < 2 {
		return "", false
	}
	return strings.Join(fragments, ""), true
}

func classificationAllowsAnyVocabulary(classification SourceClassification, value string) bool {
	for _, allowance := range classification.Vocabulary {
		if allowance.Value == value {
			return true
		}
	}
	return false
}

func TestProductionRootsCoverTopLevelCommandPackages(t *testing.T) {
	t.Parallel()

	if len(productionRoots) == 0 || productionRoots[0] != "cmd" {
		t.Fatalf("production roots must cover all cmd packages, got %v", productionRoots)
	}
}

func TestNonGoConsumersUseExactPathBindings(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, tc := range []struct {
		path      string
		required  string
		forbidden string
	}{
		{path: "bin/pair-notify", required: "PAIR_OUTER_TTY_PATH", forbidden: "outer-tty-$tag"},
		{path: "nvim/init.lua", required: "PAIR_IMAGE_CAPTURE_DONE_PATH", forbidden: "cap_path .. '.done'"},
		{path: "nvim/init.lua", required: "PAIR_CHANGELOG_READY_PATH", forbidden: "base .. '.ready'"},
		{path: "cmd/internal/opener/run.go", required: "ChangelogArtifacts", forbidden: "base+\".anchor\""},
		{path: "cmd/internal/opener/run.go", required: "ScrollbackArtifacts", forbidden: "sb+\".events.jsonl\""},
	} {
		raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(tc.path)))
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		if !strings.Contains(body, tc.required) {
			t.Errorf("%s does not consume exact binding %s", tc.path, tc.required)
		}
		if strings.Contains(body, tc.forbidden) {
			t.Errorf("%s still derives a selected path with %q", tc.path, tc.forbidden)
		}
	}
}

func TestArtifactDocsUseBindingsForCurrentConsumers(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, tc := range []struct {
		path      string
		required  string
		forbidden []string
	}{
		{
			path:      "atlas/go-migration-inventory.md",
			required:  "Artifact filename shapes in this table are descriptive vocabulary only.",
			forbidden: []string{"writes `slug-proposed-<tag>`", "writes `review-target-<tag>.json`"},
		},
		{
			path:      "atlas/architecture.md",
			required:  "The continuation paragraph's filename shapes are descriptive only.",
			forbidden: []string{"it appends to `draft-<tag>.md`", "watches `slug-proposed-<tag>`", "to `slug-proposed-<tag>`. Gates"},
		},
		{
			path:     "atlas/session-identity.md",
			required: "Those names are descriptive storage vocabulary, not construction instructions.",
		},
		{
			path:      "atlas/review-workbench.md",
			required:  "writes exact `$PAIR_REVIEW_TARGET_PATH`",
			forbidden: []string{"writes `review-target-<tag>.json`"},
		},
	} {
		raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(tc.path)))
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		if !strings.Contains(body, tc.required) {
			t.Errorf("%s does not state exact-binding authority", tc.path)
		}
		for _, forbidden := range tc.forbidden {
			if strings.Contains(body, forbidden) {
				t.Errorf("%s still publishes current construction phrase %q", tc.path, forbidden)
			}
		}
	}
}

func TestNoCompanionSuffixConstructionOutsideArtifactpath(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	derived := regexp.MustCompile(`(?:\+|\.\.)\s*["']\.(?:raw|events\.jsonl|ansi|viewport|openlock|anchor|cleaned|distill\.lock|status|ready|done)["']`)
	var violations []string
	for _, root := range productionRoots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "cmd/internal/artifactpath/") || !productionSourceFile(rel, path) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for lineNumber, line := range strings.Split(string(raw), "\n") {
				if derived.MatchString(line) {
					violations = append(violations, fmt.Sprintf("%s:%d: %s", rel, lineNumber+1, strings.TrimSpace(line)))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(violations) != 0 {
		sort.Strings(violations)
		t.Fatalf("companion paths constructed outside artifactpath:\n%s", strings.Join(violations, "\n"))
	}
}

func productionSource(path string) bool {
	if strings.Contains(path, "/testdata/") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_test.lua") || strings.HasSuffix(path, "_test.sh") {
		return false
	}
	switch filepath.Ext(path) {
	case ".go", ".sh", ".lua", ".kdl":
		return true
	default:
		return false
	}
}

func productionSourceFile(rel, path string) bool {
	if productionSource(rel) {
		return true
	}
	if filepath.Ext(rel) != "" {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var header [2]byte
	n, _ := f.Read(header[:])
	return n == len(header) && string(header[:]) == "#!"
}

func artifactFamiliesInFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "#") {
			continue
		}
		for _, marker := range []string{"//", "--", "#"} {
			if index := strings.Index(line, marker); index >= 0 {
				line = line[:index]
			}
		}
		for _, family := range Families {
			if artifactTokenMentioned(line, family.Token) {
				seen[family.Name] = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func artifactTokenMentioned(line, token string) bool {
	for _, boundary := range []string{`"`, `'`, "`", "/"} {
		if strings.Contains(line, boundary+token) {
			return true
		}
	}
	return false
}
