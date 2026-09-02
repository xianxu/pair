package couchtty

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type conceptContractRow struct {
	kind, name, status string
	paths              []string
	symbols            []string
}

var backtickField = regexp.MustCompile("`([^`]+)`")

// conceptInventory is the typed boundary for what the plan's Core concepts
// table must enumerate. The table supplies paths and lifecycle status; this
// inventory makes omission and addition visible instead of trusting whatever
// rows happen to remain in the prose.
var conceptInventory = []struct{ kind, name string }{
	{"PURE", "`Ring`"},
	{"PURE", "`StripQueries` + query deny-list"},
	{"PURE", "`Screen`"},
	{"PURE", "`updateMouseMode`"},
	{"PURE", "`Focus`"},
	{"PURE", "`PanelModel` / `Filter` / `SelectTree` / target join"},
	{"PURE", "`PanelKey` / `DecodePanelKeys`"},
	{"PURE", "`StatusModel` / `RenderStatusRow`"},
	{"PURE", "`Interceptor`"},
	{"PURE", "`Reserve` / `Release` / `PaintRow`"},
	{"PURE", "`ResetRegion` / `SaveCursor` / `RestoreCursor` / `ClearLine` / `HomeAndClear` / `LeaveAltScreen` / `ShowCursor` / `SetRegion` / `MoveTo`"},
	{"PURE", "`Notice` / `Feed`"},
	{"INTEGRATION", "`ptychild.Child`"},
	{"INTEGRATION", "`couchcore.TerminalHandle`"},
	{"INTEGRATION", "`couchcore.PtyRunner`"},
	{"INTEGRATION", "`FakeRunner` terminal double"},
	{"INTEGRATION", "`hostty.Host`"},
	{"INTEGRATION", "`hostty.TerminationHost`"},
	{"INTEGRATION", "`hostty.OSHost` / `hostty.FakeHost`"},
	{"INTEGRATION", "`couchtty.Console`"},
	{"INTEGRATION", "`runShell` host half"},
	{"INTEGRATION", "`termcmd.terminalTab`"},
	{"INTEGRATION", "`termcmd.restoreTerminal`"},
	{"INTEGRATION", "`consoleRunner` / `consoleRunnerFor`"},
	{"INTEGRATION", "`TestTerminalConformance_LifecyclePredicates`"},
}

// TestCoreConceptsContract turns pair#146's repeatedly drifting architecture
// table into an executable contract. Rows due through M3 must name real symbols
// at real paths; deleted symbols must be absent; PURE sources may not import IO
// seams and must have direct unit coverage. Future rows are explicit and skipped.
func TestCoreConceptsContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	plan := findConceptPlan(t, root)
	rows := parseConceptRows(t, plan)
	if len(rows) == 0 {
		t.Fatal("Core concepts contract has no rows")
	}
	assertConceptInventory(t, rows)
	for _, row := range rows {
		row := row
		t.Run(row.kind+"/"+row.name, func(t *testing.T) {
			// #151 supersedes #146's temporary flat-panel authority. Keep the
			// historical row in the inventory, but invert its current contract:
			// neither its source nor its owning type may return unnoticed.
			if row.name == "`PanelModel` / `Filter` / `SelectTree` / target join" {
				assertRetiredPanelAuthority(t, root)
				return
			}
			if strings.Contains(strings.ToLower(row.status), "planned") {
				return
			}
			deleted := strings.Contains(strings.ToLower(row.status), "deleted")
			if len(row.symbols) == 0 {
				t.Fatal("row has no backticked Go symbol")
			}
			paths := resolveConceptPaths(root, row.paths)
			var source strings.Builder
			for _, path := range paths {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read declared path %s: %v", path, err)
				}
				source.Write(raw)
				if row.kind == "PURE" && !deleted {
					assertPureSource(t, path)
				}
			}
			for _, qualified := range row.symbols {
				symbol := qualified[strings.LastIndex(qualified, ".")+1:]
				present := regexp.MustCompile(`\b` + regexp.QuoteMeta(symbol) + `\b`).MatchString(source.String())
				if deleted && present {
					t.Errorf("deleted symbol %s still exists at %v", qualified, row.paths)
				}
				if !deleted && !present {
					t.Errorf("symbol %s is absent from declared path(s) %v", qualified, row.paths)
				}
			}
			if row.kind == "PURE" && !deleted {
				assertDirectTest(t, paths, row.symbols)
			}
		})
	}
}

func assertRetiredPanelAuthority(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, "cmd/internal/couchtty/panel.go")); !os.IsNotExist(err) {
		t.Fatalf("retired panel.go exists or cannot be classified: %v", err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "cmd/internal/couchtty/*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if regexp.MustCompile(`\b(type\s+PanelModel|func\s+Filter|func\s+SelectTree)\b`).Match(raw) {
			t.Errorf("retired flat-panel authority remains in %s", path)
		}
	}
}

func TestConceptInventoryRejectsWholeRowDeletion(t *testing.T) {
	rows := make([]conceptContractRow, 0, len(conceptInventory)-1)
	for _, item := range conceptInventory {
		if item.name != "`PanelKey` / `DecodePanelKeys`" {
			rows = append(rows, conceptContractRow{kind: item.kind, name: item.name})
		}
	}
	if problems := conceptInventoryProblems(rows); !strings.Contains(strings.Join(problems, "\n"), "missing PURE row `PanelKey` / `DecodePanelKeys`") {
		t.Fatalf("whole-row deletion was not rejected: %v", problems)
	}
}

func assertConceptInventory(t *testing.T, rows []conceptContractRow) {
	t.Helper()
	for _, problem := range conceptInventoryProblems(rows) {
		t.Error(problem)
	}
}

func conceptInventoryProblems(rows []conceptContractRow) []string {
	expected := make(map[string]bool, len(conceptInventory))
	for _, item := range conceptInventory {
		expected[item.kind+"\x00"+item.name] = true
	}
	seen := make(map[string]bool, len(rows))
	var problems []string
	for _, row := range rows {
		key := row.kind + "\x00" + row.name
		if seen[key] {
			problems = append(problems, "duplicate "+row.kind+" row "+row.name)
		}
		seen[key] = true
		if !expected[key] {
			problems = append(problems, "unexpected "+row.kind+" row "+row.name)
		}
	}
	for _, item := range conceptInventory {
		if !seen[item.kind+"\x00"+item.name] {
			problems = append(problems, "missing "+item.kind+" row "+item.name)
		}
	}
	return problems
}

func findConceptPlan(t *testing.T, root string) string {
	t.Helper()
	name := "000146-couch-tty-switching-and-attach-plan.md"
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

func parseConceptRows(t *testing.T, path string) []conceptContractRow {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open plan: %v", err)
	}
	defer f.Close()
	var kind string
	var rows []conceptContractRow
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		switch line {
		case "### Pure entities":
			kind = "PURE"
			continue
		case "### Integration points":
			kind = "INTEGRATION"
			continue
		case "## Milestones":
			kind = ""
		}
		if kind == "" || !strings.HasPrefix(line, "|") || strings.Contains(line, "|---") || strings.HasPrefix(line, "| Name |") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 4 {
			t.Fatalf("malformed Core concepts row: %s", line)
		}
		name, lives, status := strings.TrimSpace(fields[1]), strings.TrimSpace(fields[2]), strings.TrimSpace(fields[3])
		rows = append(rows, conceptContractRow{
			kind: kind, name: name, status: status,
			symbols: captures(name), paths: captures(lives),
		})
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scan plan: %v", err)
	}
	return rows
}

func captures(s string) []string {
	var out []string
	for _, match := range backtickField.FindAllStringSubmatch(s, -1) {
		out = append(out, match[1])
	}
	return out
}

func resolveConceptPaths(root string, declared []string) []string {
	paths := make([]string, 0, len(declared))
	for _, path := range declared {
		if !strings.Contains(path, "/") && len(paths) > 0 {
			path = filepath.Join(filepath.Dir(paths[0]), path)
		} else {
			path = filepath.Join(root, path)
		}
		paths = append(paths, path)
	}
	return paths
}

func assertPureSource(t *testing.T, path string) {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	banned := map[string]bool{
		"io": true, "os": true, "os/exec": true, "net": true, "syscall": true,
		"github.com/creack/pty": true, "golang.org/x/term": true,
	}
	for _, imp := range f.Imports {
		name := strings.Trim(imp.Path.Value, `"`)
		if banned[name] || strings.HasPrefix(name, "net/") {
			t.Errorf("PURE row source %s imports IO seam %s", path, name)
		}
	}
}

func assertDirectTest(t *testing.T, paths []string, symbols []string) {
	t.Helper()
	for _, source := range paths {
		matches, _ := filepath.Glob(filepath.Join(filepath.Dir(source), "*_test.go"))
		for _, testPath := range matches {
			raw, err := os.ReadFile(testPath)
			if err != nil {
				continue
			}
			for _, qualified := range symbols {
				symbol := qualified[strings.LastIndex(qualified, ".")+1:]
				if regexp.MustCompile(`\b` + regexp.QuoteMeta(symbol) + `\b`).Match(raw) {
					return
				}
			}
		}
	}
	t.Errorf("PURE row symbols %v have no direct test beside %v", symbols, paths)
}
