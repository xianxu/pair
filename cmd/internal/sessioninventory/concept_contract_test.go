package sessioninventory

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
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

// issue155DetailTypes exhaustively disposes support shapes that are
// owned by #155 but are implementation detail rather than Core Concepts.
// The contract below rejects every new unmarked type until it is
// either promoted to the plan table or deliberately added here.
var issue155DetailTypes = map[string][]string{
	"cmd/internal/pairlog/runcli.go":                     {"persistFunc"},
	"cmd/internal/pairlog/store.go":                      {"File", "Runtime", "OSRuntime", "fileLock"},
	"cmd/internal/sessioninventory/binding.go":           {"BindingStatus", "EvidenceKind", "CandidateOutcome", "BindingInput", "NodeBinding", "bindingWork"},
	"cmd/internal/sessioninventory/conformance.go":       {"ConformanceStatus", "ConformanceAgent", "ConformanceReport"},
	"cmd/internal/sessioninventory/event.go":             {"NativeEventKind", "EventDisposition", "textBlock"},
	"cmd/internal/sessioninventory/forest_projection.go": {"ForestProjection"},
	"cmd/internal/sessioninventory/model.go":             {"Agent", "Role", "TimeSource", "NativeTime", "ArtifactKind", "Artifact", "Fact", "Node", "Forest", "DiagnosticCode", "Severity", "factKey", "canonicalNode"},
	"cmd/internal/sessioninventory/offline.go":           {"OfflineRecoveryInput"},
	"cmd/internal/sessioninventory/pair_inventory.go":    {"pairOwner", "pairConfig"},
	"cmd/internal/sessioninventory/pairfacts.go":         {"PairLogParseResult", "PairLogEntry"},
	"cmd/internal/sessioninventory/query.go":             {"SessionQuery"},
	"cmd/internal/sessioninventory/render.go":            {"RenderFormat", "inventoryV1", "forestV1", "nodeV1", "diagnosticV1", "correlationV1", "evidenceV1", "ambiguityV1"},
	"cmd/internal/sessioninventory/round.go":             {"NativeEventFact", "normalizedTurn"},
	"cmd/internal/sessioninventory/runcli.go":            {"cliOptions", "cliRenderers"},
	"cmd/internal/sessioninventory/runtime.go":           {"StorageRoot", "FileEntry", "ListingIssuesError", "SQLiteResult", "ScanResult", "Scanner", "ScannerFunc"},
	"cmd/internal/sessioninventory/runtime_os.go":        {"boundedBuffer"},
	"cmd/internal/sessioninventorytest/fake_runtime.go":  {"Operation", "storedFile", "sqliteKey", "processState"},
	"cmd/internal/sessionledger/record.go":               {"RecordKind", "NativeWatermark", "Record", "Owner", "ParseResult", "Current", "wireRecord", "strictField", "nullableArgsField", "compatibilityWireRecord", "decodeNativeWatermark", "decodeWireRecord"},
	"cmd/internal/sessionledger/store.go":                {"AppendOutcome", "AppendOutcomeError", "Unlocker", "AppendFile", "Runtime"},
	"cmd/internal/sessionledger/store_unix.go":           {"OSRuntime", "osLock"},
	"cmd/internal/sessionwatch/lifecycle.go":             {"LedgerAppender", "PrepareLaunchInput", "PreparedLaunch", "ConfigWriter"},
	"cmd/internal/sessionwatch/sessionwatch.go":          {"ConfigPayload", "ObserveInput"},
}

func TestEveryCoreConceptIntroductionMatchesDeclarations(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	planPath := filepath.Join(root, "workshop", "plans", "000155-agent-session-tree-inventory-plan.md")
	sources := issue155OwnedSources(t, root, planPath)
	introductions := readPlanIntroductions(t, planPath)
	want := []string{"M1", "M2", "final"}
	if strings.Join(introductions, ",") != strings.Join(want, ",") {
		t.Fatalf("Core Concepts introduction stages = %v, want exhaustive allowed stages %v", introductions, want)
	}
	for _, introduced := range introductions {
		t.Run(introduced, func(t *testing.T) { assertConceptContract(t, root, introduced, sources) })
	}
}

func issue155OwnedSources(t *testing.T, root, planPath string) []string {
	t.Helper()
	command := exec.Command("git", "log", "--format=", "--name-only", "--diff-filter=A", "--grep=^#155", "4c454436..HEAD", "--", "*.go")
	command.Dir = root
	raw, err := command.Output()
	if err != nil {
		t.Fatalf("derive #155 source inventory: %v", err)
	}
	planRaw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	candidates := mergeIssue155SourceCandidates(string(raw), string(planRaw))
	result := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if _, err := os.Stat(filepath.Join(root, path)); err == nil {
			result = append(result, path)
		}
	}
	if len(result) == 0 {
		t.Fatal("derived #155 source inventory is empty")
	}
	return result
}

func mergeIssue155SourceCandidates(added, plan string) []string {
	seen := map[string]bool{}
	for _, path := range strings.Fields(added) {
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			seen[filepath.ToSlash(path)] = true
		}
	}
	for _, line := range strings.Split(plan, "\n") {
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "|------") || strings.Contains(line, "| Name |") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 6 {
			continue
		}
		matches := conceptNamePattern.FindStringSubmatch(fields[2])
		if len(matches) == 2 && strings.HasSuffix(matches[1], ".go") {
			seen[filepath.ToSlash(matches[1])] = true
		}
	}
	result := make([]string, 0, len(seen))
	for path := range seen {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func TestIssueOwnedSourceCandidatesIncludeNewPackagesAndPlanSources(t *testing.T) {
	got := mergeIssue155SourceCandidates(
		"cmd/internal/newpkg/domain.go\ncmd/internal/newpkg/domain_test.go\n",
		"| `Existing` | `cmd/internal/existing/model.go` | new | M1 |\n",
	)
	want := []string{"cmd/internal/existing/model.go", "cmd/internal/newpkg/domain.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("sources=%v want=%v", got, want)
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

func TestEveryIssueOwnedTypeHasConceptDisposition(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	planPath := filepath.Join(root, "workshop", "plans", "000155-agent-session-tree-inventory-plan.md")
	sources := issue155OwnedSources(t, root, planPath)
	wantDetails := map[string]bool{}
	for path, names := range issue155DetailTypes {
		for _, name := range names {
			wantDetails[path+":"+name] = true
		}
	}
	seenDetails := map[string]bool{}
	for _, relativePath := range sources {
		path := filepath.Join(root, relativePath)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range group.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				doc := typeSpec.Doc
				if doc == nil {
					doc = group.Doc
				}
				key := relativePath + ":" + typeSpec.Name.Name
				marker := conceptMarker(doc)
				if problem := typeDispositionProblem(marker, wantDetails[key], typeSpec.Name.Name); problem != "" {
					t.Errorf("%s: %s", relativePath, problem)
				} else if wantDetails[key] {
					seenDetails[key] = true
				}
			}
		}
	}
	for key := range wantDetails {
		if !seenDetails[key] {
			t.Errorf("stale issue155DetailTypes disposition %s", key)
		}
	}
}

func typeDispositionProblem(marker []string, detailed bool, name string) string {
	if marker != nil && !validConceptMarker(marker) {
		return fmt.Sprintf("type %s has invalid Core Concept marker %q", name, marker)
	}
	if marker == nil && !detailed {
		return fmt.Sprintf("type %s has no Core Concept or detail disposition", name)
	}
	return ""
}

func TestTypeDispositionRejectsPrivateUnknownAndInlineBypassClasses(t *testing.T) {
	inlineDetail := conceptMarker(&ast.CommentGroup{List: []*ast.Comment{{Text: "// pair:155-detail parent"}}})
	if inlineDetail != nil {
		t.Fatalf("inline detail marker bypassed explicit inventory: %q", inlineDetail)
	}
	for _, test := range []struct {
		name     string
		marker   []string
		detailed bool
		wantBad  bool
	}{
		{name: "private domain", wantBad: true},
		{name: "unknown stage", marker: []string{"pure", "new", "M3"}, wantBad: true},
		{name: "malformed marker", marker: []string{"detail", "new", "M2"}, wantBad: true},
		{name: "explicit detail", detailed: true},
		{name: "known concept", marker: []string{"integration", "modified", "final"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotBad := typeDispositionProblem(test.marker, test.detailed, "privateShape") != ""
			if gotBad != test.wantBad {
				t.Fatalf("gotBad=%v want=%v", gotBad, test.wantBad)
			}
		})
	}
}

func validConceptMarker(marker []string) bool {
	if len(marker) < 3 || (marker[0] != "pure" && marker[0] != "integration") ||
		(marker[1] != "new" && marker[1] != "modified" && marker[1] != "deleted") {
		return false
	}
	return marker[2] == "M1" || marker[2] == "M2" || marker[2] == "final"
}

func assertConceptContract(t *testing.T, root, milestone string, sources []string) {
	t.Helper()
	plan := readPlanConcepts(t, filepath.Join(root, "workshop", "plans", "000155-agent-session-tree-inventory-plan.md"), milestone)
	declarations := readConceptDeclarations(t, root, sources, milestone)
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

func readConceptDeclarations(t *testing.T, root string, sources []string, milestone string) map[string]conceptContract {
	t.Helper()
	result := map[string]conceptContract{}
	for _, relativePath := range sources {
		path := filepath.Join(root, relativePath)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			for _, concept := range markedConcepts(declaration, filepath.ToSlash(relativePath), milestone) {
				if previous, exists := result[concept.Name]; exists {
					t.Fatalf("duplicate marked %s concept %s: %#v and %#v", milestone, concept.Name, previous, concept)
				}
				result[concept.Name] = concept
			}
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
