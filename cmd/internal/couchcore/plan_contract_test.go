package couchcore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var issue149M5ConceptRows = []string{
	"| `MigrateLegacyRecord` | pure | `cmd/internal/couchcore/migration.go` | new in M5 |",
	"| `artifactpath.Address` / `Paths` / `ScopePaths` / `Binding` | pure | `cmd/internal/artifactpath/paths.go` | new in M5 |",
	"| `artifactpath.PairCachePaths` | pure | `cmd/internal/artifactpath/paths.go` | added in M5 boundary disposition |",
	"| `artifactpath.ScrollbackArtifactSet` / `ParkedScrollbackArtifactSet` / `ChangelogArtifactSet` | pure | `cmd/internal/artifactpath/paths.go` | added in M5 boundary disposition |",
	"| `artifactpath.Family` / `SourceKind` / `SourceClassification` / `NonArtifactSources` | pure declarations | `cmd/internal/artifactpath/manifest.go` | new in M5; exhaustive source inventory added in boundary disposition |",
	"| `artifactpath.VocabularyContext` / `VocabularyAllowance` | pure declarations | `cmd/internal/artifactpath/manifest.go` | added in M5 boundary disposition |",
	"| `artifactpath.ResolvedBinding` / `ResolvedBindings` | pure declarations | `cmd/internal/artifactpath/manifest.go` | added in M5 boundary disposition |",
	"| `artifactpath.LegacyRootPaths` / `LegacyPaths` / `TagFromHistorySidecar` | pure | `cmd/internal/artifactpath/paths.go` | added in M5 boundary disposition |",
	"| `DecodeSessionNameIndex` | pure | `cmd/internal/launcher/session_index.go` | added in M5 boundary disposition |",
	"| `StandaloneThreadRegistration` / `StandaloneThreadRegistrar` | pure seam types | `cmd/internal/launcher/runtime.go` | new in M5 |",
	"| `ThreadStore.MigrateLegacyRecords` | `cmd/internal/couchcore/migration.go` | new in M5 | one locked journal transaction over cutover records and manifest completion |",
	"| `LaunchNativeWithStandaloneRegistrar` / `RegisterStandalonePair` | `cmd/internal/launcher/runcli.go`, `cmd/internal/couchcore/standalone.go`, `cmd/pair-go/main.go` | new in M5 | composition-root injection of direct Pair registration without reversing the launcher→Couch package boundary |",
	"| `ThreadStore.UpsertStandalonePair` | `cmd/internal/couchcore/standalone.go` | new in M5 | locked/revisioned direct-Pair incarnation publication with metadata preservation |",
	"| `OSRuntime.ReadSessionNameIndex` | `cmd/internal/launcher/osruntime.go` | modified in M5 and its boundary disposition | strict merge of legacy-global and selected-scope durable bindings; missing files mean empty, malformed/unreadable files fail closed |",
	"| `readSessionNameIndexes` | `cmd/internal/launcher/session_index.go` | added in M5 boundary disposition | one injected-IO overlap reader used by runtime, address claim, and quiescence |",
}

func TestIssue149M5CoreConceptInventoryContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(findIssue149Plan(t, root))
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range issue149M5ConceptProblems(string(raw)) {
		t.Error(problem)
	}
}

func TestIssue149M5CoreConceptInventoryRejectsRowMutation(t *testing.T) {
	complete := strings.Join(issue149M5ConceptRows, "\n")
	for _, row := range issue149M5ConceptRows {
		if problems := issue149M5ConceptProblems(strings.Replace(complete, row, "", 1)); len(problems) == 0 {
			t.Fatalf("row deletion escaped contract: %s", row)
		}
		mutated := strings.Replace(row, " | ", " | wrong-", 1)
		if problems := issue149M5ConceptProblems(strings.Replace(complete, row, mutated, 1)); len(problems) == 0 {
			t.Fatalf("row field mutation escaped contract: %s", row)
		}
	}
}

func issue149M5ConceptProblems(plan string) []string {
	var problems []string
	for _, row := range issue149M5ConceptRows {
		if strings.Count(plan, row) != 1 {
			problems = append(problems, "M5 Core concepts row must appear exactly once: "+row)
		}
	}
	return problems
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
