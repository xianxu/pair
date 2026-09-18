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
	// The store's archive guard. M1 took the occupancy rule out of DecideResume
	// and M3 took it out of archive: whether a process is running is a fact
	// about the world, and a decoded record cannot supply one.
	//
	// The first two keys were written from MEMORY -- the identifiers the author
	// had just changed -- and matched none of the four sites the boundary review
	// then found still stating the claim (#256 M3 BR, I3). The rest are DATA: a
	// paragraph-window sweep of the homes for the claim's VOCABULARY (live,
	// occupied, hosting, "same guard" near archive), run at the commit that
	// retired it. Paragraph, not line: three of the four sites spanned comment
	// lines, which a line-oriented grep never sees.
	"archivableRecord's occupancy":      "#256 M3: archive's occupancy question moved to ArchivableState over the classification",
	"occupiedIncarnation":               "#256 M3 folded it into hasOccupiedIncarnation; archive no longer asks it",
	"still LIVE or mid-park":            "#256 M3: the store refuses only an open park or start claim; hosting is ArchivableState's",
	"refuses an occupied thread":        "#256 M3: the admission rule refuses a hosted thread, not the store's guard",
	"to prove the thread is not live":   "#256 M3: archivableRecord proves nothing about liveness; the classification does",
	"proves a thread is not live":       "#256 M3: archivableRecord proves nothing about liveness; the classification does",
	"refuses a live/unknown helper":     "#256 M3: the store refuses an open park or start claim; the classification refuses a hosted thread",
	"runs the same guard":               "#256 M3: Couch.ArchiveThread runs ArchivableState AND the record guard; they are two, not one",
	"live one in the store":             "#256 M3: the store no longer refuses a live thread; Couch.ArchiveThread's admission does",
	"never archive-eligible":            "#256 M3: ArchivableState permits an unreadable row; archiving one moves its bytes and never stops its session",
	"checks occupancy before quiescing": "#256 M3: archive checks the continuation's actors and the classification; 'occupancy' named the retired rule",
	// The close's BR-34/BR-38: two referents M1 and M2 retired, still restated in
	// four atlas paragraphs after every per-milestone sweep. Found by the same
	// paragraph-window vocabulary sweep (occupied/occupancy near resume/park;
	// verified-park-as-handle; durable live proof) run over EVERY home at close.
	"exactly matches one observed TTY owner":    "#256 M1: live is positive evidence couch hosts the process, a union with no match required",
	"durable proven-live":                       "#256 M1: live is couch's own hosting or an OS-vouched process, never the record alone",
	"occupied-incarnation refusal is unchanged": "#256 M1 deleted DecideResume's occupancy refusal; it contradicted the detached classification",
	"exact verified resume handle":              "#256 M2: the ledger, not the park receipt, is the resume authority for parked",
	"no occupied incarnation":                   "#256 M2: a parked thread may carry a dead launcher's incarnation or a driverless claim",
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
