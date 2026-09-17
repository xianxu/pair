package couchcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #256 retired several claims about how couch decides things. Each was corrected
// in prose at one boundary and restated somewhere else at the next: the rule
// "sweep every home of a changed referent" was written down three times and
// failed to land seven times across four review rounds, twice in the very commit
// that wrote the rule.
//
// Prose discipline has demonstrably not worked, so this is the check.
//
// SCOPE, stated because a guard that overclaims is worse than none: this catches
// the RECURRENCE of a specific retired claim, not novel staleness. A new sentence
// that is wrong in a new way still needs a reader. What it guarantees is that a
// claim this issue deliberately retired cannot quietly come back — which is
// exactly what kept happening.
var issue256RetiredClaims = map[string]string{
	"refuses any occupied incarnation": "#256 M1 deleted that refusal: the incarnation names a launcher that dies with couch",
	// NOT the bare phrase "verified parked thread": threadrecord's validator uses
	// it for the record FIELD, whose invariant is unchanged and still true. The
	// retired claim is the receipt as AUTHORITY, so the keys name that use.
	"live or verified parked":          "#256 M2: the park receipt is not switch/resume authority; the ledger is",
	"verified parked thread's actions": "#256 M2: the park receipt is not switch/resume authority; the ledger is",
	"resume a verified-parked":         "#256 M2: the park receipt is not resume authority; the ledger is",
	"exact verified-park row":          "#256 M2: the park receipt is not resume authority; the ledger is",
	"verified-park resume":             "#256 M2: the park receipt is not resume authority; the ledger is",
	"parked` has two producers":        "#256 M2: there are four, enumerated in everyThreadShape",
	"switchableWhenNothingRuns":        "#256 M2 round 2 deleted it: asking the session was a second re-derivation",
}

// A line that explicitly narrates the retirement is not a restatement of it.
func narratesRetirement(line string) bool {
	for _, marker := range []string{"no longer", "used to", "until #256", "retired", "deleted", "RESTATED", "was wrong", "stopped"} {
		if strings.Contains(line, marker) {
			return true
		}
	}
	return false
}

func TestRetiredClaimsAreNotRestatedAsCurrent(t *testing.T) {
	root := repoRootFrom(t)
	// The homes of a referent, as #256's own rule enumerates them: production
	// code comments, the atlas, and the README. Review sidecars, gate ledgers,
	// `## Revisions` history and workshop/history legitimately quote what was
	// retired, so they are not homes.
	var files []string
	for _, dir := range []string{"cmd", "atlas"} {
		err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			name := info.Name()
			if strings.HasSuffix(name, "_test.go") {
				return nil
			}
			if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".md") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	files = append(files, filepath.Join(root, "README.md"))
	if len(files) < 50 {
		t.Fatalf("walked only %d files; the search lost its scope and stopped checking", len(files))
	}

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, path)
		for number, line := range strings.Split(string(content), "\n") {
			if narratesRetirement(line) {
				continue
			}
			for claim, why := range issue256RetiredClaims {
				if strings.Contains(line, claim) {
					t.Errorf("%s:%d states a claim #256 retired -- %q\n\t%s\n\tline: %s",
						rel, number+1, claim, why, strings.TrimSpace(line))
				}
			}
		}
	}
}
