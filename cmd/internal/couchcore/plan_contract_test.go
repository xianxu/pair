package couchcore

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type issue149ConceptRequirement struct {
	name string
	path string
}

func TestIssue149M5CoreConceptInventoryContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(findIssue149Plan(t, root))
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149ArtifactConceptRequirements(t, root)
	for _, problem := range issue149M5ConceptProblems(string(raw), requirements) {
		t.Error(problem)
	}
}

func TestIssue149M5CoreConceptInventoryRejectsRowMutation(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(findIssue149Plan(t, root))
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149ArtifactConceptRequirements(t, root)
	plan := string(raw)
	for _, requirement := range requirements {
		row := ""
		needle := ""
		for _, line := range issue149PureConceptRows(plan) {
			for _, candidate := range []string{"`" + requirement.name + "`", "`artifactpath." + requirement.name + "`"} {
				if !strings.Contains(line, candidate) {
					continue
				}
				row = line
				needle = candidate
				break
			}
			if row != "" {
				break
			}
		}
		mutated := strings.Replace(plan, needle, "`removed`", 1)
		if problems := issue149M5ConceptProblems(mutated, requirements); len(problems) == 0 {
			t.Fatalf("entity deletion escaped derived contract: %s", requirement.name)
		}
		for label, replacement := range map[string]string{
			"kind":   strings.Replace(plan, row, strings.Replace(row, "| pure", "| integration", 1), 1),
			"path":   strings.Replace(plan, row, strings.Replace(row, "`"+requirement.path+"`", "`wrong/path.go`", 1), 1),
			"status": strings.Replace(plan, row, strings.Replace(row, "M5", "M4", 1), 1),
		} {
			if problems := issue149M5ConceptProblems(replacement, requirements); len(problems) == 0 {
				t.Fatalf("%s mutation escaped derived contract for %s", label, requirement.name)
			}
		}
	}
}

func issue149M5ConceptProblems(plan string, requirements []issue149ConceptRequirement) []string {
	var problems []string
	lines := issue149PureConceptRows(plan)
	for _, requirement := range requirements {
		var matches []string
		for _, line := range lines {
			if strings.Contains(line, "`"+requirement.name+"`") || strings.Contains(line, "`artifactpath."+requirement.name+"`") {
				matches = append(matches, line)
			}
		}
		if len(matches) != 1 {
			problems = append(problems, "M5 artifact concept must appear in exactly one row: "+requirement.name)
			continue
		}
		row := matches[0]
		if !strings.Contains(strings.ToLower(row), "| pure") || !strings.Contains(row, "`"+requirement.path+"`") || !strings.Contains(row, "M5") {
			problems = append(problems, "M5 artifact concept has wrong kind/path/status: "+requirement.name)
		}
	}
	return problems
}

func issue149PureConceptRows(plan string) []string {
	var rows []string
	inTable := false
	for _, line := range strings.Split(plan, "\n") {
		if line == "| Entity | Kind | Lives in | Status |" {
			inTable = true
			continue
		}
		if inTable && line == "" {
			break
		}
		if inTable && strings.HasPrefix(line, "| `") {
			rows = append(rows, line)
		}
	}
	return rows
}

func issue149ArtifactConceptRequirements(t *testing.T, root string) []issue149ConceptRequirement {
	t.Helper()
	var requirements []issue149ConceptRequirement
	for _, rel := range []string{"cmd/internal/artifactpath/manifest.go", "cmd/internal/artifactpath/paths.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, rel), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE && gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				switch typed := spec.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(typed.Name.Name) {
						requirements = append(requirements, issue149ConceptRequirement{name: typed.Name.Name, path: rel})
					}
				case *ast.ValueSpec:
					for _, name := range typed.Names {
						if ast.IsExported(name.Name) {
							requirements = append(requirements, issue149ConceptRequirement{name: name.Name, path: rel})
						}
					}
				}
			}
		}
	}
	sort.Slice(requirements, func(i, j int) bool { return requirements[i].name < requirements[j].name })
	return requirements
}

func TestIssue149CurrentCoreConceptKinds(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path := findIssue149Plan(t, root)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := map[string]string{
		"CouchNamespace":      "integration",
		"PolicyResult":        "pure",
		"AdmissionDecision":   "pure",
		"ThreadAddress":       "pure",
		"StartTransaction":    "pure",
		"ThreadMetadataPatch": "pure",
		"ThreadSummary":       "pure",
		"Operation":           "pure",
		"threadrecord.Record": "pure",
		"strictjson.Decode":   "pure",
	}
	seen := map[string]bool{}
	inTable := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "| Entity | Kind | Lives in | Status |" {
			inTable = true
			continue
		}
		if inTable && line == "" {
			break
		}
		if !inTable {
			continue
		}
		if !strings.HasPrefix(line, "| `") || strings.Contains(line, "| Entity |") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 5 {
			continue
		}
		name, kind := strings.TrimSpace(fields[1]), strings.ToLower(strings.TrimSpace(fields[2]))
		for symbol, expected := range want {
			if strings.Contains(name, "`"+symbol+"`") {
				seen[symbol] = true
				if kind != expected {
					t.Errorf("%s kind = %q, want %q", symbol, kind, expected)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for symbol := range want {
		if !seen[symbol] {
			t.Errorf("core concept row missing current %s", symbol)
		}
	}
}

func findIssue149Plan(t *testing.T, root string) string {
	t.Helper()
	name := "000149-couch-opaque-tags-and-a-human-naming-layer-plan.md"
	active := filepath.Join(root, "workshop", "plans", name)
	if _, err := os.Stat(active); err == nil {
		return active
	}
	var found string
	_ = filepath.WalkDir(filepath.Join(root, "workshop", "history"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == name {
			found = path
		}
		return nil
	})
	if found == "" {
		t.Fatalf("find %s in active or archived plans", name)
	}
	return found
}

func TestOpaqueIdentityCommentDoesNotReintroducePathDerivedContract(t *testing.T) {
	raw, err := os.ReadFile("couch.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, obsolete := range []string{"derives from the TREE", "same tree always resumes"} {
		if strings.Contains(string(raw), obsolete) {
			t.Errorf("obsolete path-derived identity comment returned: %q", obsolete)
		}
	}
}

func TestIssue149PureCoreTestsStayAtPureBoundary(t *testing.T) {
	for _, name := range []string{
		"thread_test.go", "starttransaction_test.go", "admission_test.go",
		"threadmetadata_model_test.go", "ops_declarations_test.go",
		filepath.Join("..", "threadrecord", "record_test.go"),
		filepath.Join("..", "strictjson", "decode_test.go"),
	} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"testCouchNamespace(", "t.TempDir(", "NewFake", "NewThreadStore(", "newTestThreadStore(", "os.", "exec."} {
			if strings.Contains(string(raw), forbidden) {
				t.Errorf("PURE direct test %s crosses integration boundary with %q", name, forbidden)
			}
		}
	}
}

func TestIssue149BlockedRunnersDelegateToOneHandshakeAuthority(t *testing.T) {
	want := map[string]string{
		"runner.go":    "return startBlockedChild(startExecChild, r.LaunchHelper, dir, argv, env, timeout)",
		"ptyrunner.go": "return startBlockedChild(r.start, r.LaunchHelper, dir, argv, env, timeout)",
	}
	for name, delegation := range want {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !strings.Contains(text, delegation) {
			t.Errorf("%s no longer delegates StartBlocked to shared authority", name)
		}
		for _, parallelAuthority := range []string{"os.Pipe(", "newAcknowledgedHandle("} {
			if strings.Contains(text, parallelAuthority) {
				t.Errorf("%s reintroduced blocked-start protocol %q", name, parallelAuthority)
			}
		}
	}
}
