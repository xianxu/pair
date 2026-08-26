package couchcore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIssue149CurrentCoreConceptKinds(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path := findIssue149Plan(t, root)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := map[string]string{
		"CouchNamespace":    "integration",
		"PolicyResult":      "pure",
		"AdmissionDecision": "pure",
		"PolicyTable":       "pure",
		"ThreadAddress":     "pure",
		"Operation":         "integration",
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
