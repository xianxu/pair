package termcmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A `planned — Mx` ROW MUST NOT SURVIVE A TICKED Mx.
//
// This is the sixth finding in the `plan-table-drift` family, so it is fixed as
// a rule rather than by editing the rows again. The mechanism that let the table
// rot is specific and worth naming: couchtty's Core-concepts contract SKIPS any
// row whose status contains "planned" (core_concepts_contract_test.go), which is
// correct while the milestone is ahead of you and silently wrong the moment it
// lands. So a table could carry `planned — M3` through M3's close with nothing
// objecting, and this one did.
//
// The check is deliberately cheap and text-level: it needs no package internals,
// which is why it can live next to the code the table describes rather than
// forcing couchtty's machinery to be copied (BR-52's lesson, applied to a test).
func TestNoPlannedRowSurvivesItsTickedMilestone(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	issues, err := filepath.Glob(filepath.Join(root, "workshop", "issues", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Fatal("no issue files found; the guard would pass vacuously")
	}

	ticked := regexp.MustCompile(`(?m)^\s*-\s*\[x\]\s*(M\d+[a-z]?)\b`)
	checked := 0
	for _, issue := range issues {
		raw, err := os.ReadFile(issue)
		if err != nil {
			t.Fatal(err)
		}
		done := ticked.FindAllStringSubmatch(string(raw), -1)
		if len(done) == 0 {
			continue
		}
		// The plan sits beside the issue, same stem plus -plan.
		stem := strings.TrimSuffix(filepath.Base(issue), ".md")
		plan := filepath.Join(root, "workshop", "plans", stem+"-plan.md")
		planRaw, err := os.ReadFile(plan)
		if err != nil {
			continue // simple work has no durable plan; that is allowed
		}
		checked++
		for _, m := range done {
			milestone := m[1]
			stale := regexp.MustCompile(`planned\s*—\s*` + milestone + `\b`)
			for i, line := range strings.Split(string(planRaw), "\n") {
				if !strings.HasPrefix(strings.TrimSpace(line), "|") || !stale.MatchString(line) {
					continue
				}
				t.Errorf("%s:%d still says `planned — %s` after %s was ticked in %s:\n  %s",
					plan, i+1, milestone, milestone, filepath.Base(issue), strings.TrimSpace(line))
			}
		}
	}
	if checked == 0 {
		t.Fatal("no issue with a ticked milestone had a durable plan; the guard proved nothing")
	}
}
