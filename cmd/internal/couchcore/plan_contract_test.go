package couchcore

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type issue149ConceptRequirement struct {
	name string
	path string
	kind string
}

const issue149M5DeclarationDigest = "748b33313dc565abbbfd5db00e690892441c4fb408bd42a8ed194d06a75c8b1f"

const (
	issue149M5Base = "6a714336"
	issue149M5Head = "c434016"
)

// issue149M5GoSources is the exhaustive set of Go sources touched by M5. Every
// declaration in these files receives a disposition: a pair:m5-concept marker
// makes it architectural, and an absent marker explicitly means implementation
// detail. The plan inventory is derived only from the marked declarations.
var issue149M5GoSources = []string{
	"cmd/internal/adapt/adapt.go", "cmd/internal/adapt/adapt_test.go",
	"cmd/internal/agentcmd/restart.go",
	"cmd/internal/artifactpath/coverage_test.go", "cmd/internal/artifactpath/cross_scope_integration_test.go",
	"cmd/internal/artifactpath/manifest.go", "cmd/internal/artifactpath/paths.go", "cmd/internal/artifactpath/paths_test.go",
	"cmd/internal/changelogcmd/changelogcmd.go", "cmd/internal/changelogcmd/run_test.go",
	"cmd/internal/clipcmd/clipcmd.go", "cmd/internal/clipcmd/clipcmd_test.go", "cmd/internal/clipcmd/run.go",
	"cmd/internal/contextcmd/contextcmd.go", "cmd/internal/contextcmd/contextcmd_test.go", "cmd/internal/contextcmd/panejson_kdl_test.go",
	"cmd/internal/continuationcmd/continuationcmd.go",
	"cmd/internal/continuationcmd/draft.go",
	"cmd/internal/couchcmd/readme_test.go",
	"cmd/internal/couchcore/artifactcollision_test.go", "cmd/internal/couchcore/couch.go", "cmd/internal/couchcore/couch_test.go", "cmd/internal/couchcore/launchhelper_test.go", "cmd/internal/couchcore/migration.go",
	"cmd/internal/couchcore/migration_test.go", "cmd/internal/couchcore/plan_contract_test.go",
	"cmd/internal/couchcore/storejournal.go", "cmd/internal/couchcore/threadmetadata.go",
	"cmd/internal/couchcore/threadmetadata_test.go", "cmd/internal/couchcore/threadstore.go",
	"cmd/internal/draftroute/route.go",
	"cmd/internal/dispatcher/dispatcher.go", "cmd/internal/dispatcher/dispatcher_test.go",
	"cmd/internal/launcher/agent_defaults.go", "cmd/internal/launcher/agentargs.go", "cmd/internal/launcher/args.go", "cmd/internal/launcher/args_test.go",
	"cmd/internal/ctxmeter/ctxmeter.go", "cmd/internal/ctxmeter/ctxmeter_test.go",
	"cmd/internal/launcher/config.go", "cmd/internal/launcher/config_test.go", "cmd/internal/launcher/createflow.go",
	"cmd/internal/launcher/createflow_test.go", "cmd/internal/launcher/history.go", "cmd/internal/launcher/layoutflow.go",
	"cmd/internal/launcher/ledger.go", "cmd/internal/launcher/ledger_test.go",
	"cmd/internal/launcher/legacy_live.go", "cmd/internal/launcher/lifecycle.go", "cmd/internal/launcher/lifecycle_test.go",
	"cmd/internal/launcher/markers.go", "cmd/internal/launcher/markers_test.go",
	"cmd/internal/launcher/migrate.go", "cmd/internal/launcher/osruntime.go", "cmd/internal/launcher/osruntime_test.go",
	"cmd/internal/launcher/pick.go", "cmd/internal/launcher/pick_test.go", "cmd/internal/launcher/readiness.go",
	"cmd/internal/launcher/rename.go", "cmd/internal/launcher/rename_test.go",
	"cmd/internal/launcher/restart.go", "cmd/internal/launcher/restart_test.go", "cmd/internal/launcher/runcli.go",
	"cmd/internal/launcher/runtime.go", "cmd/internal/launcher/scoped_paths.go", "cmd/internal/launcher/session.go",
	"cmd/internal/launcher/session_index.go",
	"cmd/internal/launcher/thread_claim.go", "cmd/internal/launcher/thread_claim_test.go",
	"cmd/internal/opener/opener.go", "cmd/internal/opener/opener_test.go", "cmd/internal/opener/run.go",
	"cmd/internal/opener/run_test.go", "cmd/internal/opener/runcli.go", "cmd/internal/opener/runtime.go",
	"cmd/internal/reviewcmd/reviewcmd_test.go", "cmd/internal/reviewcmd/run.go", "cmd/internal/reviewcmd/run_test.go", "cmd/internal/reviewcmd/runcli.go", "cmd/internal/reviewcmd/runtime.go",
	"cmd/internal/pairlog/runcli.go", "cmd/internal/pairlog/runcli_test.go", "cmd/internal/pairlog/store.go", "cmd/internal/pairlog/store_test.go",
	"cmd/internal/runtimebundle/embed_test.go", "cmd/internal/runtimebundlegen/clean_source_test.go",
	"cmd/internal/scrollbackcmd/render_test.go", "cmd/internal/scrollbackcmd/scrollbackcmd.go",
	"cmd/internal/scrollbackcmd/scrollbackcmd_test.go", "cmd/internal/scrollbackcmd/timestamps_test.go",
	"cmd/internal/sessioninventory/activity.go", "cmd/internal/sessioninventory/activity_test.go", "cmd/internal/sessioninventory/activitycli.go", "cmd/internal/sessioninventory/activitycli_test.go",
	"cmd/internal/sessioninventory/conformance.go", "cmd/internal/sessioninventory/conformance_live_test.go", "cmd/internal/sessioninventory/conformance_test.go",
	"cmd/internal/sessioninventory/concept_contract_test.go",
	"cmd/internal/sessioninventory/binding.go", "cmd/internal/sessioninventory/binding_test.go",
	"cmd/internal/sessioninventory/diagnostic.go", "cmd/internal/sessioninventory/diagnostic_test.go",
	"cmd/internal/sessioninventory/event.go", "cmd/internal/sessioninventory/event_test.go",
	"cmd/internal/sessioninventory/events.go", "cmd/internal/sessioninventory/events_test.go",
	"cmd/internal/sessioninventory/forest_projection.go", "cmd/internal/sessioninventory/forest_projection_test.go",
	"cmd/internal/sessioninventory/inventory.go",
	"cmd/internal/sessioninventory/model.go", "cmd/internal/sessioninventory/model_test.go",
	"cmd/internal/sessioninventory/offline.go", "cmd/internal/sessioninventory/offline_test.go",
	"cmd/internal/sessioninventory/order.go", "cmd/internal/sessioninventory/order_test.go",
	"cmd/internal/sessioninventory/pair_inventory.go", "cmd/internal/sessioninventory/pair_inventory_test.go",
	"cmd/internal/sessioninventory/pairfacts.go", "cmd/internal/sessioninventory/pairfacts_test.go",
	"cmd/internal/sessioninventory/query.go", "cmd/internal/sessioninventory/query_test.go",
	"cmd/internal/sessioninventory/render.go", "cmd/internal/sessioninventory/render_test.go",
	"cmd/internal/sessioninventory/round.go", "cmd/internal/sessioninventory/round_test.go",
	"cmd/internal/sessioninventory/runcli.go", "cmd/internal/sessioninventory/runcli_failure_test.go", "cmd/internal/sessioninventory/runcli_test.go",
	"cmd/internal/sessioninventory/runtime.go", "cmd/internal/sessioninventory/runtime_os.go", "cmd/internal/sessioninventory/runtime_os_test.go",
	"cmd/internal/sessioninventory/scan.go", "cmd/internal/sessioninventory/scan_agy.go", "cmd/internal/sessioninventory/scan_agy_test.go",
	"cmd/internal/sessioninventory/scan_claude.go", "cmd/internal/sessioninventory/scan_claude_test.go",
	"cmd/internal/sessioninventory/scan_codex.go", "cmd/internal/sessioninventory/scan_codex_test.go",
	"cmd/internal/sessioninventory/scan_fuzz_test.go", "cmd/internal/sessioninventory/scan_helpers.go",
	"cmd/internal/sessioninventory/scan_muse.go", "cmd/internal/sessioninventory/scan_muse_test.go",
	"cmd/internal/sessioninventory/scan_test.go", "cmd/internal/sessioninventory/scanner_fixture_test.go",
	"cmd/internal/sessioninventory/shadow_test.go", "cmd/internal/sessioninventory/usage.go", "cmd/internal/sessioninventory/usage_test.go",
	"cmd/internal/sessioninventorytest/fake_runtime.go", "cmd/internal/sessioninventorytest/fake_runtime_test.go",
	"cmd/internal/sessionledger/record.go", "cmd/internal/sessionledger/record_test.go",
	"cmd/internal/sessionledger/store.go", "cmd/internal/sessionledger/store_subprocess_test.go", "cmd/internal/sessionledger/store_test.go", "cmd/internal/sessionledger/store_unix.go",
	"cmd/internal/sessionwatch/lifecycle.go", "cmd/internal/sessionwatch/lifecycle_test.go",
	"cmd/internal/sessionwatch/run.go", "cmd/internal/sessionwatch/run_test.go", "cmd/internal/sessionwatch/runcli.go", "cmd/internal/sessionwatch/runcli_test.go",
	"cmd/internal/sessionwatch/runtime.go", "cmd/internal/sessionwatch/sessionwatch.go", "cmd/internal/sessionwatch/sessionwatch_test.go",
	"cmd/internal/slugcmd/slug.go", "cmd/internal/slugcmd/slugcmd.go", "cmd/internal/slugcmd/slugcmd_test.go", "cmd/internal/titlepoller/run.go", "cmd/internal/titlepoller/run_test.go",
	"cmd/internal/strictjson/decode.go", "cmd/internal/threadrecord/record.go",
	"cmd/internal/titlepoller/runtime.go",
	"cmd/internal/workbenchshortcut/shortcut.go", "cmd/internal/wrapcmd/agent_restart_test.go", "cmd/internal/wrapcmd/wrap.go",
	"cmd/pair-go/changelog_seam_test.go", "cmd/pair-go/helper_equivalence_test.go", "cmd/pair-go/main.go", "cmd/pair-go/main_test.go",
}

// issue149M5DeletedGoSources records files in the milestone diff whose deletion
// is their complete declaration disposition.
var issue149M5DeletedGoSources = []string{
	"cmd/internal/codexsid/codexsid.go",
	"cmd/internal/codexsid/codexsid_test.go",
	"cmd/internal/transcript/transcript.go",
	"cmd/internal/transcript/transcript_test.go",
	"cmd/internal/launcher/thread_index.go",
	"cmd/internal/launcher/thread_index_conformance_test.go",
	"cmd/internal/launcher/thread_index_test.go",
}

// issue149M5RetiredGoSources records M5 sources introduced after the diff
// baseline and retired later. They have a historical declaration disposition,
// but their create-then-delete lifecycle is correctly absent from the net diff.
var issue149M5RetiredGoSources = []string{
	"cmd/internal/couchcore/standalone.go",
	"cmd/internal/couchcore/standalone_test.go",
}

// issue149M5RevertedGoSources records M5 sources whose later edits restored
// their baseline content. Their declarations still belong to the historical
// concept inventory, while the files are absent from the current net diff.
var issue149M5RevertedGoSources = []string{}

// issue149M5RetiredConceptRequirements preserves the historical M5 concept
// disposition for declarations removed after that milestone. The plan remains
// the M5 record of truth; retired concepts are not active production symbols.
var issue149M5RetiredConceptRequirements = []issue149ConceptRequirement{
	{name: "LaunchNativeWithStandaloneRegistrar", path: "cmd/internal/launcher/runcli.go", kind: "integration"},
	{name: "RegisterStandalonePair", path: "cmd/internal/couchcore/standalone.go", kind: "integration"},
	{name: "StandaloneThreadRegistrar", path: "cmd/internal/launcher/runtime.go", kind: "pure"},
	{name: "StandaloneThreadRegistration", path: "cmd/internal/launcher/runtime.go", kind: "pure"},
	{name: "ThreadStore.UpsertStandalonePair", path: "cmd/internal/couchcore/standalone.go", kind: "integration"},
}

func TestIssue149M5DeclarationDispositionSourceSetMatchesMilestoneDiff(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("source archive has no Git metadata; the checked-in disposition set remains the oracle")
	}
	catalog := map[string]bool{}
	for _, sources := range [][]string{issue149M5GoSources, issue149M5DeletedGoSources, issue149M5RetiredGoSources, issue149M5RevertedGoSources} {
		for _, rel := range sources {
			catalog[rel] = true
		}
	}
	for _, rel := range issue149M5BoundaryGoSources(t, root) {
		if !catalog[rel] {
			t.Errorf("M5 boundary source lacks a declaration disposition: %s", rel)
		}
	}
}

func TestIssue149M5DeclarationDispositionSetIsClosed(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if got := issue149M5SourceDeclarationDigest(t, root); got != issue149M5DeclarationDigest {
		t.Fatalf("M5 declaration set changed without an explicit concept/detail disposition: got %s, want %s", got, issue149M5DeclarationDigest)
	}
}

func TestIssue149M5CoreConceptInventoryContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(findPlanArtifact(t, root, "000149-couch-opaque-tags-and-a-human-naming-layer-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149M5ConceptRequirements(t, root)
	for _, problem := range issue149M5ConceptProblems(string(raw), requirements) {
		t.Error(problem)
	}
}

func TestIssue149M5UnmarkedExportedAuthorityFailsClosed(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "migration.go", "package couchcore\ntype ReviewAddedAuthority struct{}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149M5ConceptsForDecl(t, file.Name.Name, "cmd/internal/couchcore/migration.go", file.Decls[0], false)
	if len(requirements) != 1 || requirements[0].name != "ReviewAddedAuthority" || requirements[0].kind != "pure" {
		t.Fatalf("unmarked exported authority disposition = %+v, want one fail-closed pure concept", requirements)
	}
}

func TestIssue149M5CoreConceptInventoryRejectsRowMutation(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(findPlanArtifact(t, root, "000149-couch-opaque-tags-and-a-human-naming-layer-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149M5ConceptRequirements(t, root)
	plan := string(raw)
	for _, requirement := range requirements {
		row := ""
		needle := ""
		for _, line := range issue149ConceptRows(plan) {
			for _, candidate := range issue149RowConceptNames(line) {
				if candidate != requirement.name {
					continue
				}
				row = line
				short := strings.TrimPrefix(candidate, "artifactpath.")
				needle = "`" + short + "`"
				if strings.Contains(line, "`"+candidate+"`") {
					needle = "`" + candidate + "`"
				}
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
		kindMutation := strings.Replace(plan, row, strings.Replace(row, "| pure", "| integration", 1), 1)
		if requirement.kind == "integration" {
			kindMutation = strings.Replace(plan, "| Integration | Lives in | Status | Wraps |", "| Entity | Kind | Lives in | Status |", 1)
		}
		for label, replacement := range map[string]string{
			"kind":   kindMutation,
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
	lines := issue149ConceptRows(plan)
	required := map[string]bool{}
	for _, requirement := range requirements {
		required[requirement.name] = true
		var matches []string
		for _, line := range lines {
			for _, name := range issue149RowConceptNames(line) {
				if name == requirement.name {
					matches = append(matches, line)
					break
				}
			}
		}
		if len(matches) != 1 {
			problems = append(problems, "M5 artifact concept must appear in exactly one row: "+requirement.name)
			continue
		}
		row := matches[0]
		if issue149ConceptRowKind(plan, row) != requirement.kind || !strings.Contains(row, "`"+requirement.path+"`") || !strings.Contains(row, "M5") {
			problems = append(problems, "M5 artifact concept has wrong kind/path/status: "+requirement.name)
		}
	}
	for _, name := range issue149M5PlanConceptNames(lines) {
		if !required[name] {
			problems = append(problems, "M5 plan concept has no source declaration disposition: "+name)
		}
	}
	return problems
}

func issue149ConceptRows(plan string) []string {
	var rows []string
	inTable := false
	for _, line := range strings.Split(plan, "\n") {
		if line == "| Entity | Kind | Lives in | Status |" || line == "| Integration | Lives in | Status | Wraps |" {
			inTable = true
			continue
		}
		if inTable && line == "" {
			inTable = false
			continue
		}
		if inTable && strings.HasPrefix(line, "| `") {
			rows = append(rows, line)
		}
	}
	return rows
}

func issue149ConceptRowKind(plan, row string) string {
	position := strings.Index(plan, row)
	if position < 0 {
		return ""
	}
	before := plan[:position]
	core := strings.LastIndex(before, "| Entity | Kind | Lives in | Status |")
	integration := strings.LastIndex(before, "| Integration | Lives in | Status | Wraps |")
	if integration > core {
		return "integration"
	}
	fields := strings.Split(row, "|")
	if len(fields) > 2 && strings.HasPrefix(strings.ToLower(strings.TrimSpace(fields[2])), "pure") {
		return "pure"
	}
	return ""
}

func issue149M5PlanConceptNames(lines []string) []string {
	var names []string
	for _, line := range lines {
		if !strings.Contains(line, "M5") {
			continue
		}
		names = append(names, issue149RowConceptNames(line)...)
	}
	sort.Strings(names)
	return names
}

func issue149RowConceptNames(line string) []string {
	fields := strings.Split(line, "|")
	if len(fields) < 3 {
		return nil
	}
	entity := fields[1]
	var names []string
	for {
		start := strings.Index(entity, "`")
		if start < 0 {
			break
		}
		entity = entity[start+1:]
		end := strings.Index(entity, "`")
		if end < 0 {
			break
		}
		names = append(names, entity[:end])
		entity = entity[end+1:]
	}
	prefix := ""
	if len(names) > 0 && strings.HasPrefix(names[0], "artifactpath.") {
		prefix = "artifactpath."
	}
	for i := 1; i < len(names); i++ {
		if prefix != "" && !strings.Contains(names[i], ".") {
			names[i] = prefix + names[i]
		}
	}
	return names
}

func issue149M5ConceptRequirements(t *testing.T, root string) []issue149ConceptRequirement {
	t.Helper()
	closedSet := issue149M5SourceDeclarationDigest(t, root) == issue149M5DeclarationDigest
	var requirements []issue149ConceptRequirement
	for _, rel := range issue149M5BoundaryGoSources(t, root) {
		raw, exists := issue149M5SourceAtHead(t, root, rel)
		if !exists {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), rel, raw, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			requirements = append(requirements, issue149M5ConceptsForDecl(t, file.Name.Name, rel, decl, closedSet)...)
		}
	}
	requirements = append(requirements, issue149M5RetiredConceptRequirements...)
	sort.Slice(requirements, func(i, j int) bool { return requirements[i].name < requirements[j].name })
	return requirements
}

func issue149M5ConceptsForDecl(t *testing.T, packageName, rel string, decl ast.Decl, closedSet bool) []issue149ConceptRequirement {
	t.Helper()
	marker := func(doc *ast.CommentGroup) string {
		if doc == nil {
			return ""
		}
		for _, line := range doc.List {
			text := strings.TrimSpace(strings.TrimPrefix(line.Text, "//"))
			if strings.HasPrefix(text, "pair:m5-concept ") {
				return strings.TrimSpace(strings.TrimPrefix(text, "pair:m5-concept "))
			}
		}
		return ""
	}
	qualified := func(name string) string {
		if packageName == "artifactpath" {
			return "artifactpath." + name
		}
		return name
	}
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		kind := marker(typed.Doc)
		if kind == "" && !closedSet && typed.Recv == nil && ast.IsExported(typed.Name.Name) {
			kind = "pure"
		}
		if kind == "" {
			return nil
		}
		name := typed.Name.Name
		if typed.Recv != nil && len(typed.Recv.List) == 1 {
			receiver := typed.Recv.List[0].Type
			if pointer, ok := receiver.(*ast.StarExpr); ok {
				receiver = pointer.X
			}
			if ident, ok := receiver.(*ast.Ident); ok {
				name = ident.Name + "." + name
			}
		}
		return []issue149ConceptRequirement{{name: qualified(name), path: rel, kind: kind}}
	case *ast.GenDecl:
		var out []issue149ConceptRequirement
		for _, spec := range typed.Specs {
			switch item := spec.(type) {
			case *ast.TypeSpec:
				kind := marker(item.Doc)
				if kind == "" {
					kind = marker(typed.Doc)
				}
				if kind == "" && !closedSet && ast.IsExported(item.Name.Name) {
					kind = "pure"
				}
				if kind != "" {
					out = append(out, issue149ConceptRequirement{name: qualified(item.Name.Name), path: rel, kind: kind})
				}
			case *ast.ValueSpec:
				kind := marker(item.Doc)
				if kind == "" {
					kind = marker(typed.Doc)
				}
				for _, name := range item.Names {
					resolvedKind := kind
					if resolvedKind == "" && typed.Tok == token.VAR && !closedSet && ast.IsExported(name.Name) {
						resolvedKind = "pure"
					}
					if resolvedKind != "" {
						out = append(out, issue149ConceptRequirement{name: qualified(name.Name), path: rel, kind: resolvedKind})
					}
				}
			}
		}
		return out
	default:
		t.Fatalf("unclassified declaration kind %T in %s", decl, rel)
		return nil
	}
}

func issue149M5SourceDeclarationDigest(t *testing.T, root string) string {
	t.Helper()
	var keys []string
	retired := map[string]bool{}
	for _, rel := range issue149M5RetiredGoSources {
		retired[rel] = true
	}
	for _, rel := range issue149M5BoundaryGoSources(t, root) {
		raw, exists := issue149M5SourceAtHead(t, root, rel)
		if !exists {
			status := "deleted"
			if retired[rel] {
				status = "retired"
			}
			keys = append(keys, rel+"|"+status)
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), rel, raw, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				receiver := ""
				if typed.Recv != nil && len(typed.Recv.List) == 1 {
					receiver = issue149ReceiverName(typed.Recv.List[0].Type)
				}
				keys = append(keys, rel+"|func|"+receiver+"|"+typed.Name.Name)
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					switch item := spec.(type) {
					case *ast.TypeSpec:
						keys = append(keys, rel+"|"+typed.Tok.String()+"|"+item.Name.Name)
					case *ast.ValueSpec:
						for _, name := range item.Names {
							keys = append(keys, rel+"|"+typed.Tok.String()+"|"+name.Name)
						}
					}
				}
			}
		}
	}
	sort.Strings(keys)
	digest := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return fmt.Sprintf("%x", digest)
}

func issue149M5SourceAtHead(t *testing.T, root, rel string) ([]byte, bool) {
	t.Helper()
	command := exec.Command("git", "-C", root, "show", issue149M5Head+":"+rel)
	raw, err := command.Output()
	if err == nil {
		return raw, true
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, false
	}
	t.Fatal(err)
	return nil, false
}

func issue149M5BoundaryGoSources(t *testing.T, root string) []string {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("M5 boundary source derivation requires Git metadata")
	}
	command := exec.Command("git", "-C", root, "diff", "--name-only", issue149M5Base+".."+issue149M5Head, "--", "*.go")
	raw, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	sources := strings.Fields(string(raw))
	sort.Strings(sources)
	return sources
}

func issue149ReceiverName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return "*" + issue149ReceiverName(typed.X)
	case *ast.IndexExpr:
		return issue149ReceiverName(typed.X) + "[]"
	case *ast.IndexListExpr:
		return issue149ReceiverName(typed.X) + "[]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func TestIssue149CurrentCoreConceptKinds(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path := findPlanArtifact(t, root, "000149-couch-opaque-tags-and-a-human-naming-layer-plan.md")
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

func findPlanArtifact(t *testing.T, root, name string) string {
	t.Helper()
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

func TestIssue152DeliveredCoreConceptsResolveToGoDeclarations(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	planPath := findPlanArtifact(t, root, "000152-couch-verified-park-resume-plan.md")
	raw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	const heading = "#### Delivered Core Concepts (authoritative)"
	section := strings.SplitN(string(raw), heading, 2)
	if len(section) != 2 {
		t.Fatalf("plan has no %q section", heading)
	}
	body := strings.SplitN(section[1], "\n#### ", 2)[0]
	resolved := 0
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 6 {
			t.Fatalf("malformed delivered concept row %q", line)
		}
		name := strings.Trim(strings.TrimSpace(cells[1]), "`")
		kind := strings.TrimSpace(cells[2])
		path := strings.Trim(strings.TrimSpace(cells[3]), "`")
		status := strings.TrimSpace(cells[4])
		if kind != "PURE" && kind != "INTEGRATION" {
			t.Errorf("%s has invalid kind %q", name, kind)
		}
		if status != "new" && status != "modified" && status != "deleted" {
			t.Errorf("%s has invalid status %q", name, status)
		}
		if err := requireGoDeclaration(filepath.Join(root, path), name); err != nil {
			t.Errorf("%s at %s: %v", name, path, err)
		}
		resolved++
	}
	if resolved != 21 {
		t.Fatalf("resolved %d delivered concepts, want 21", resolved)
	}
}

func requireGoDeclaration(path, qualifiedName string) error {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return err
	}
	receiver, name, isMethod := strings.Cut(qualifiedName, ".")
	if !isMethod {
		name = receiver
	}
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.GenDecl:
			if isMethod {
				continue
			}
			for _, spec := range value.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == name {
					return nil
				}
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, declared := range valueSpec.Names {
						if declared.Name == name {
							return nil
						}
					}
				}
			}
		case *ast.FuncDecl:
			if value.Name.Name != name {
				continue
			}
			if !isMethod && value.Recv == nil {
				return nil
			}
			if isMethod && receiverName(value.Recv) == receiver {
				return nil
			}
		}
	}
	return fmt.Errorf("Go declaration not found")
}

func receiverName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) != 1 {
		return ""
	}
	expression := fields.List[0].Type
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}
