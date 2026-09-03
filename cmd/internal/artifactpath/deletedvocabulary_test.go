package artifactpath

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// deletedVocabulary is the identifier set removed by a deletion milestone.
//
// The docs sweep kept being driven by recall of which files felt relevant.
// pair#170 M4 swept atlas/ thoroughly and forgot README.md, then the issue log
// claimed a make target had been deleted when it had not -- two findings in one
// family, both from the same cause: no enumeration. Deletion is proved by
// ABSENCE (ARCH-PURPOSE), and absence has to be checked mechanically or it is
// just a claim.
//
// There is deliberately NO per-row file list. The first version of this guard
// had one, and a hand-listed subset reintroduces exactly the recall step the
// guard exists to remove -- it omitted Go sources, so two live-voice admission
// comments survived the sweep that installed it. The scope is every tracked
// text file.
var deletedVocabulary = []struct {
	term string
	why  string
}{
	{term: "sdlc fleet policy", why: "fleet-policy admission deleted by pair#170 M4"},
	{term: "provision-worktree", why: "the typed policy refusal went with admission (pair#170 M4)"},
	{term: "StartGrantStore", why: "the start-grant capability table was deleted by pair#170 M4"},
	{term: "test-couch-policy-live", why: "the policy conformance target went with admission (pair#170 M4)"},
	{term: "normalized provider", why: "the fleet-policy provider was deleted by pair#170 M4"},
}

// Prose may still NAME a deleted thing in order to say it was deleted. What
// must not survive is a passage describing it as current.
var retiredMarkers = []string{
	"pair#170", "Pair #170", "#170", "deleted", "went with", "used to", "no longer",
	"removed", "deletion", "gone", "gained", "gets its", "before pair", "gitignore",
}

func TestDeletedVocabularyIsNotDescribedAsLive(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, path := range trackedTextFiles(t, repoRoot) {
		raw, err := os.ReadFile(filepath.Join(repoRoot, path))
		if err != nil {
			continue
		}
		// Paragraph, not line. Markdown prose WRAPS: the pre-fix README broke
		// "(`sdlc\nfleet policy`)" across a newline, so a line-oriented guard
		// could never fire on the very text it was written for -- measured, not
		// theorised. A blank-line-separated block is the smallest unit that
		// both survives wrapping and still scopes the retired-marker check.
		for number, paragraph := range paragraphs(string(raw)) {
			normalized := strings.Join(strings.Fields(paragraph.text), " ")
			if marksAsRetired(normalized) {
				continue
			}
			for _, entry := range deletedVocabulary {
				if !strings.Contains(normalized, entry.term) {
					continue
				}
				t.Errorf("%s:%d describes %q as live (%s):\n  %s\n"+
					"Either update the passage or mark it as history.",
					path, paragraph.line, entry.term, entry.why, truncate(normalized))
				break
			}
			_ = number
		}
	}
}

type paragraph struct {
	text string
	line int
}

func paragraphs(content string) []paragraph {
	var out []paragraph
	current := paragraph{line: 1}
	flush := func() {
		if strings.TrimSpace(current.text) != "" {
			out = append(out, current)
		}
	}
	for number, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			current = paragraph{line: number + 2}
			continue
		}
		if current.text == "" {
			current.line = number + 1
		}
		current.text += line + "\n"
	}
	flush()
	return out
}

func marksAsRetired(text string) bool {
	for _, marker := range retiredMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func truncate(text string) string {
	if len(text) <= 160 {
		return text
	}
	return text[:160] + "…"
}

// trackedTextFiles asks git what is tracked, so the set cannot drift from the
// repository the way a hand-maintained list does.
//
// workshop/ is excluded: it is the append-only working record -- issues, plans
// and review sidecars necessarily discuss deleted machinery in analytical
// voice, and a guard over it would flag the very notes that document the
// deletion.
func trackedTextFiles(t *testing.T, repoRoot string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", repoRoot, "ls-files").Output()
	if err != nil {
		t.Skipf("git ls-files unavailable: %v", err)
	}
	var files []string
	for _, path := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if path == "" || strings.HasPrefix(path, "workshop/") {
			continue
		}
		switch filepath.Ext(path) {
		case ".go", ".md", ".yml", ".yaml", ".sh", ".txt", ".json", ".cue", "":
			files = append(files, path)
		}
	}
	if len(files) == 0 {
		t.Fatal("found no tracked text files; the scan is broken, not the tree")
	}
	return files
}
