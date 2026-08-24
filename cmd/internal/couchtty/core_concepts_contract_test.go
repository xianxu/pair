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
	for _, row := range rows {
		row := row
		t.Run(row.kind+"/"+row.name, func(t *testing.T) {
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
