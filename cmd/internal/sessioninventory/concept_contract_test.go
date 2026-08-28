package sessioninventory

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type conceptContract struct {
	Name       string
	Kind       string
	Path       string
	Status     string
	Introduced string
}

var conceptNamePattern = regexp.MustCompile("`([^`]+)`")

func TestM1CoreConceptTableMatchesDeclarations(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	plan := readM1PlanConcepts(t, filepath.Join(root, "workshop", "plans", "000155-agent-session-tree-inventory-plan.md"))
	declarations := readM1ConceptDeclarations(t, root, []string{"cmd/internal/sessioninventory", "cmd/internal/sessioninventorytest"})
	if len(plan) != len(declarations) {
		t.Fatalf("M1 concept count: plan=%d declarations=%d\nplan=%#v\ndeclarations=%#v", len(plan), len(declarations), plan, declarations)
	}
	for name, want := range declarations {
		got, ok := plan[name]
		if !ok {
			t.Errorf("marked M1 concept %s is absent from Core Concepts", name)
			continue
		}
		if got != want {
			t.Errorf("M1 concept %s = %#v, want declaration %#v", name, got, want)
		}
	}
}

func readM1PlanConcepts(t *testing.T, planPath string) map[string]conceptContract {
	t.Helper()
	file, err := os.Open(planPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	result := map[string]conceptContract{}
	kind := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "### Pure Entities":
			kind = "pure"
		case "### Integration Points":
			kind = "integration"
		}
		if kind == "" || !strings.HasPrefix(line, "|") || strings.Contains(line, "|------") || strings.Contains(line, "| Name |") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 6 || strings.TrimSpace(fields[4]) != "M1" {
			continue
		}
		pathMatches := conceptNamePattern.FindStringSubmatch(fields[2])
		if len(pathMatches) != 2 {
			t.Fatalf("M1 concept row has no path: %s", line)
		}
		for _, match := range conceptNamePattern.FindAllStringSubmatch(fields[1], -1) {
			concept := conceptContract{Name: match[1], Kind: kind, Path: pathMatches[1], Status: strings.TrimSpace(fields[3]), Introduced: "M1"}
			if previous, exists := result[concept.Name]; exists {
				t.Fatalf("duplicate M1 concept %s: %#v and %#v", concept.Name, previous, concept)
			}
			result[concept.Name] = concept
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func readM1ConceptDeclarations(t *testing.T, root string, directories []string) map[string]conceptContract {
	t.Helper()
	result := map[string]conceptContract{}
	for _, directory := range directories {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return walkErr
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			relativePath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			for _, declaration := range file.Decls {
				for _, concept := range markedM1Concepts(declaration, filepath.ToSlash(relativePath)) {
					if previous, exists := result[concept.Name]; exists {
						t.Fatalf("duplicate marked M1 concept %s: %#v and %#v", concept.Name, previous, concept)
					}
					result[concept.Name] = concept
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func markedM1Concepts(declaration ast.Decl, path string) []conceptContract {
	group, ok := declaration.(*ast.GenDecl)
	if !ok {
		return nil
	}
	var result []conceptContract
	for _, spec := range group.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		marker := conceptMarker(typeSpec.Doc)
		if marker == nil {
			marker = conceptMarker(group.Doc)
		}
		if len(marker) == 3 && marker[2] == "M1" {
			result = append(result, conceptContract{Name: typeSpec.Name.Name, Kind: marker[0], Path: path, Status: marker[1], Introduced: marker[2]})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func conceptMarker(group *ast.CommentGroup) []string {
	if group == nil {
		return nil
	}
	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if strings.HasPrefix(text, "pair:155-concept ") {
			return strings.Fields(strings.TrimPrefix(text, "pair:155-concept "))
		}
	}
	return nil
}
