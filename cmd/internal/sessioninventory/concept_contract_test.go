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
	"unicode"
)

type conceptContract struct {
	Name       string
	Kind       string
	Path       string
	Status     string
	Introduced string
}

var conceptNamePattern = regexp.MustCompile("`([^`]+)`")

var issue155ConceptDirectories = []string{
	"cmd/internal/commitoutcome", "cmd/internal/sessioninventory", "cmd/internal/sessioninventorytest",
	"cmd/internal/sessionledger", "cmd/internal/sessionwatch", "cmd/internal/pairlog",
}

// issue155DetailTypes exhaustively disposes exported support shapes that are
// owned by #155 but are implementation detail rather than Core Concepts.
// The contract below rejects every new unmarked exported type until it is
// either promoted to the plan table or deliberately added here.
var issue155DetailTypes = map[string][]string{
	"cmd/internal/sessioninventory/binding.go":           {"BindingStatus", "EvidenceKind", "CandidateOutcome", "BindingInput", "NodeBinding"},
	"cmd/internal/sessioninventory/conformance.go":       {"ConformanceStatus", "ConformanceAgent", "ConformanceReport"},
	"cmd/internal/sessioninventory/event.go":             {"NativeEventKind", "EventDisposition"},
	"cmd/internal/sessioninventory/forest_projection.go": {"ForestProjection"},
	"cmd/internal/sessioninventory/model.go":             {"Agent", "Role", "TimeSource", "NativeTime", "ArtifactKind", "Artifact", "Fact", "Node", "Forest", "DiagnosticCode", "Severity"},
	"cmd/internal/sessioninventory/offline.go":           {"OfflineRecoveryInput"},
	"cmd/internal/sessioninventory/pairfacts.go":         {"PairLogParseResult", "PairLogEntry"},
	"cmd/internal/sessioninventory/query.go":             {"SessionQuery"},
	"cmd/internal/sessioninventory/render.go":            {"RenderFormat"},
	"cmd/internal/sessioninventory/round.go":             {"NativeEventFact"},
	"cmd/internal/sessioninventory/runtime.go":           {"StorageRoot", "FileEntry", "ListingIssuesError", "SQLiteResult", "ScanResult", "Scanner", "ScannerFunc"},
	"cmd/internal/sessioninventorytest/fake_runtime.go":  {"Operation"},
	"cmd/internal/sessionledger/record.go":               {"RecordKind", "NativeWatermark", "Record", "Owner", "ParseResult", "Current"},
	"cmd/internal/sessionledger/store.go":                {"AppendOutcome", "AppendOutcomeError", "Unlocker", "AppendFile", "Runtime"},
	"cmd/internal/sessionledger/store_unix.go":           {"OSRuntime"},
	"cmd/internal/sessionwatch/lifecycle.go":             {"LedgerAppender", "PrepareLaunchInput", "PreparedLaunch", "ConfigWriter"},
	"cmd/internal/sessionwatch/run.go":                   {"Options", "Runtime"},
	"cmd/internal/sessionwatch/runtime.go":               {"OSRuntime"},
	"cmd/internal/sessionwatch/sessionwatch.go":          {"ConfigPayload", "ObserveInput"},
	"cmd/internal/pairlog/store.go":                      {"File", "Runtime", "OSRuntime"},
}

func TestEveryCoreConceptIntroductionMatchesDeclarations(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	planPath := filepath.Join(root, "workshop", "plans", "000155-agent-session-tree-inventory-plan.md")
	introductions := readPlanIntroductions(t, planPath)
	want := []string{"M1", "M2", "final"}
	if strings.Join(introductions, ",") != strings.Join(want, ",") {
		t.Fatalf("Core Concepts introduction stages = %v, want exhaustive allowed stages %v", introductions, want)
	}
	for _, introduced := range introductions {
		t.Run(introduced, func(t *testing.T) { assertConceptContract(t, root, introduced, issue155ConceptDirectories) })
	}
}

func readPlanIntroductions(t *testing.T, planPath string) []string {
	t.Helper()
	raw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "|------") || strings.Contains(line, "| Name |") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) >= 6 && conceptNamePattern.MatchString(fields[1]) && conceptNamePattern.MatchString(fields[2]) {
			seen[strings.TrimSpace(fields[4])] = true
		}
	}
	result := make([]string, 0, len(seen))
	for _, candidate := range []string{"M1", "M2", "final"} {
		if seen[candidate] {
			result = append(result, candidate)
			delete(seen, candidate)
		}
	}
	for candidate := range seen {
		result = append(result, candidate)
	}
	return result
}

func TestEveryIssueOwnedExportedTypeHasConceptDisposition(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	wantDetails := map[string]bool{}
	for path, names := range issue155DetailTypes {
		for _, name := range names {
			wantDetails[path+":"+name] = true
		}
	}
	seenDetails := map[string]bool{}
	for _, directory := range issue155ConceptDirectories {
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
			relativePath = filepath.ToSlash(relativePath)
			for _, declaration := range file.Decls {
				group, ok := declaration.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range group.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !unicode.IsUpper([]rune(typeSpec.Name.Name)[0]) {
						continue
					}
					doc := typeSpec.Doc
					if doc == nil {
						doc = group.Doc
					}
					key := relativePath + ":" + typeSpec.Name.Name
					if conceptMarker(doc) == nil && !hasDetailMarker(doc) && !wantDetails[key] {
						t.Errorf("%s: exported type %s has no Core Concept marker or detail disposition", relativePath, typeSpec.Name.Name)
					} else if wantDetails[key] {
						seenDetails[key] = true
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for key := range wantDetails {
		if !seenDetails[key] {
			t.Errorf("stale issue155DetailTypes disposition %s", key)
		}
	}
}

func hasDetailMarker(group *ast.CommentGroup) bool {
	if group == nil {
		return false
	}
	for _, comment := range group.List {
		if strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(comment.Text, "//")), "pair:155-detail ") {
			return true
		}
	}
	return false
}

func assertConceptContract(t *testing.T, root, milestone string, directories []string) {
	t.Helper()
	plan := readPlanConcepts(t, filepath.Join(root, "workshop", "plans", "000155-agent-session-tree-inventory-plan.md"), milestone)
	declarations := readConceptDeclarations(t, root, directories, milestone)
	if len(plan) != len(declarations) {
		t.Fatalf("%s concept count: plan=%d declarations=%d\nplan=%#v\ndeclarations=%#v", milestone, len(plan), len(declarations), plan, declarations)
	}
	for name, want := range declarations {
		got, ok := plan[name]
		if !ok {
			t.Errorf("marked %s concept %s is absent from Core Concepts", milestone, name)
			continue
		}
		if got != want {
			t.Errorf("%s concept %s = %#v, want declaration %#v", milestone, name, got, want)
		}
	}
}

func readPlanConcepts(t *testing.T, planPath, milestone string) map[string]conceptContract {
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
		if len(fields) < 6 || strings.TrimSpace(fields[4]) != milestone {
			continue
		}
		pathMatches := conceptNamePattern.FindStringSubmatch(fields[2])
		if len(pathMatches) != 2 {
			t.Fatalf("%s concept row has no path: %s", milestone, line)
		}
		for _, match := range conceptNamePattern.FindAllStringSubmatch(fields[1], -1) {
			concept := conceptContract{Name: match[1], Kind: kind, Path: pathMatches[1], Status: strings.TrimSpace(fields[3]), Introduced: milestone}
			if previous, exists := result[concept.Name]; exists {
				t.Fatalf("duplicate %s concept %s: %#v and %#v", milestone, concept.Name, previous, concept)
			}
			result[concept.Name] = concept
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func readConceptDeclarations(t *testing.T, root string, directories []string, milestone string) map[string]conceptContract {
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
				for _, concept := range markedConcepts(declaration, filepath.ToSlash(relativePath), milestone) {
					if previous, exists := result[concept.Name]; exists {
						t.Fatalf("duplicate marked %s concept %s: %#v and %#v", milestone, concept.Name, previous, concept)
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

func markedConcepts(declaration ast.Decl, path, milestone string) []conceptContract {
	if function, ok := declaration.(*ast.FuncDecl); ok {
		marker := conceptMarker(function.Doc)
		if len(marker) >= 3 && marker[2] == milestone {
			name := function.Name.Name
			if len(marker) >= 4 {
				name = strings.Join(marker[3:], " ")
			}
			return []conceptContract{{Name: name, Kind: marker[0], Path: path, Status: marker[1], Introduced: marker[2]}}
		}
		return nil
	}
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
		if len(marker) >= 3 && marker[2] == milestone {
			name := typeSpec.Name.Name
			if len(marker) >= 4 {
				name = strings.Join(marker[3:], " ")
			}
			result = append(result, conceptContract{Name: name, Kind: marker[0], Path: path, Status: marker[1], Introduced: marker[2]})
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
