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
		if classification.Kind != Constructor && classification.Kind != ResolvedConsumer && classification.Kind != GeneratedMirror {
			t.Fatalf("invalid classification kind for %q: %q", classification.Path, classification.Kind)
		}
		for _, name := range classification.Families {
			if _, ok := familyByName[name]; !ok {
				t.Fatalf("%s classifies unknown family %q", classification.Path, name)
			}
		}
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(classification.Path))); err != nil {
			t.Fatalf("classified source %q is absent: %v", classification.Path, err)
		}
		classified[classification.Path] = classification
	}

	generatedRoot := filepath.Join(repoRoot, "cmd/internal/runtimebundle/assets/runtime")
	if err := filepath.WalkDir(generatedRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		classification, ok := classified[filepath.ToSlash(rel)]
		if !ok || classification.Kind != GeneratedMirror {
			missing = append(missing, filepath.ToSlash(rel)+": generated mirror is not exactly classified")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	referenceViolations, err := artifactReferenceViolations(repoRoot, productionRoots, classified)
	if err != nil {
		t.Fatal(err)
	}
	missing = append(missing, referenceViolations...)
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
			if !ok || classification.Kind != ResolvedConsumer || filepath.Ext(rel) != ".go" || !productionSourceFile(rel, path) {
				return nil
			}
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				var expression ast.Expr
				joinCall := false
				switch value := node.(type) {
				case *ast.CallExpr:
					selector, ok := value.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					owner, ownerOK := selector.X.(*ast.Ident)
					if !ownerOK || owner.Name != "filepath" || selector.Sel.Name != "Join" {
						return true
					}
					expression = value
					joinCall = true
				case *ast.BinaryExpr:
					if value.Op != token.ADD {
						return true
					}
					expression = value
				default:
					return true
				}
				for _, familyName := range classification.Families {
					if familyName == "restart" {
						continue
					}
					for _, family := range Families {
						if family.Name == familyName && expressionConstructsFamily(expression, family.Token, joinCall) {
							line := fileSet.Position(node.Pos()).Line
							violations[fmt.Sprintf("%s:%d: resolved consumer constructs %s", rel, line, familyName)] = true
						}
					}
				}
				return true
			})
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

func expressionConstructsFamily(expression ast.Expr, familyToken string, joinCall bool) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		if joinCall && strings.Contains(value, familyToken) ||
			strings.HasSuffix(value, familyToken) || strings.Contains(value, "/"+familyToken) {
			found = true
		}
		return !found
	})
	return found
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
	classifications := make([]SourceClassification, 0, 2)
	for _, rel := range []string{
		"cmd/pair-go/mutation.go",
		"cmd/internal/launcher/mutation.go",
	} {
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "package mutation\nimport \"path/filepath\"\nfunc bad(scope string) string { return filepath.Join(scope, \"nvim-pid-*-draft\") }\n"
		if strings.Contains(rel, "pair-go") {
			body = "package mutation\nfunc bad(tag string) string { path := \"nvim-pid-\" + tag + \"-draft\"; return path }\n"
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		classifications = append(classifications, SourceClassification{
			Path: rel, Kind: ResolvedConsumer, Families: []string{"nvim-pid"},
		})
	}

	violations, err := artifactConstructorViolations(repoRoot, []string{"cmd"}, classifications)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("false resolved-consumer classifications escaped constructor guard: %v", violations)
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
