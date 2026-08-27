package couchcore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	projectMilestoneRow = regexp.MustCompile(`^- \[([ xX])\].*\[([a-z]+#[0-9]+ M[^]]+)\]`)
	projectDetailHeader = regexp.MustCompile(`^### ([a-z]+#[0-9]+ M[^ ]+)`)
)

func TestUncheckedProjectMilestoneHasNoClosedMetadata(t *testing.T) {
	root := filepath.Join("..", "..", "..", "workshop", "projects")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		for _, problem := range projectMilestoneStateProblems(path) {
			t.Errorf("%s: %s", path, problem)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestProjectMilestoneStateProblemsRejectsPrematureClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project.md")
	raw := "- [ ] work [pair#9 M1]\n\n### pair#9 M1 — work\n\n**closed:** 2026-08-26\n"
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	problems := projectMilestoneStateProblems(path)
	if len(problems) != 1 || !strings.Contains(problems[0], "pair#9 M1") {
		t.Fatalf("problems = %v", problems)
	}
}

func projectMilestoneStateProblems(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return []string{err.Error()}
	}
	defer f.Close()
	checked := map[string]bool{}
	var current string
	var closed = map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if match := projectMilestoneRow.FindStringSubmatch(line); match != nil {
			checked[match[2]] = strings.TrimSpace(match[1]) != ""
		}
		if match := projectDetailHeader.FindStringSubmatch(line); match != nil {
			current = match[1]
			continue
		}
		if strings.HasPrefix(line, "### ") {
			current = ""
		}
		if current != "" && strings.HasPrefix(line, "**closed:**") && strings.TrimSpace(strings.TrimPrefix(line, "**closed:**")) != "" {
			closed[current] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return []string{err.Error()}
	}
	var problems []string
	for ref := range closed {
		if isChecked, exists := checked[ref]; exists && !isChecked {
			problems = append(problems, fmt.Sprintf("unchecked milestone %s carries closed metadata", ref))
		}
	}
	return problems
}
