package artifactpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deletedVocabulary is the identifier set removed by a deletion milestone,
// paired with the docs that must stop describing it as live.
//
// The docs sweep kept being driven by recall of which files felt relevant.
// pair#170 M4 swept atlas/ thoroughly and forgot README.md, and the issue log
// then claimed a make target had been deleted when it had not -- two findings
// in the same family, both from the same cause: no enumeration. Deletion is
// proved by ABSENCE (ARCH-PURPOSE), and absence has to be checked mechanically
// or it is just a claim.
//
// Add a row when a milestone removes user-visible vocabulary. Removing a row is
// fine once the concept has been gone long enough that no reader will look for
// it; the point is the sweep, not permanent archaeology.
var deletedVocabulary = []struct {
	term  string
	why   string
	files []string
}{
	{
		term:  "sdlc fleet policy",
		why:   "fleet-policy admission deleted by pair#170 M4",
		files: []string{"README.md"},
	},
	{
		term:  "provision-worktree",
		why:   "the typed policy refusal went with admission (pair#170 M4)",
		files: []string{"README.md"},
	},
	{
		term:  "StartGrantStore",
		why:   "the start-grant capability table was deleted by pair#170 M4",
		files: []string{"README.md", "atlas/couch.md"},
	},
	{
		term:  "test-couch-policy-live",
		why:   "the policy conformance target went with admission (pair#170 M4)",
		files: []string{"README.md", "Makefile", "Makefile.local", ".github/workflows"},
	},
}

// Prose may still NAME a deleted thing to say it was deleted. What must not
// survive is a sentence describing it as current, so the check allows a mention
// on a line that marks it as history.
var retiredMarkers = []string{
	"pair#170", "Pair #170", "deleted", "went with", "used to", "no longer", "removed",
}

func TestDeletedVocabularyIsNotDescribedAsLive(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, entry := range deletedVocabulary {
		for _, target := range entry.files {
			for _, path := range docFiles(t, filepath.Join(repoRoot, target)) {
				raw, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				relative, _ := filepath.Rel(repoRoot, path)
				for number, line := range strings.Split(string(raw), "\n") {
					if !strings.Contains(line, entry.term) || marksAsRetired(line) {
						continue
					}
					t.Errorf("%s:%d describes %q as live (%s):\n  %s\n"+
						"Either update the sentence or mark it as history.",
						relative, number+1, entry.term, entry.why, strings.TrimSpace(line))
				}
			}
		}
	}
}

func marksAsRetired(line string) bool {
	for _, marker := range retiredMarkers {
		if strings.Contains(line, marker) {
			return true
		}
	}
	return false
}

// docFiles resolves a target that may be a file or a directory, so an entry can
// name ".github/workflows" without listing every workflow in it.
func docFiles(t *testing.T, target string) []string {
	t.Helper()
	info, err := os.Stat(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		return []string{target}
	}
	var out []string
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			out = append(out, filepath.Join(target, entry.Name()))
		}
	}
	return out
}
