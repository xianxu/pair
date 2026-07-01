package continuationcmd

import (
	"os/exec"
	"strings"
)

// gitRunner is the thin IO seam over `git` (no git library — shell out, the
// same pattern as cmd/pair-slug). Scoped to a repo root via `-C`.
type gitRunner struct{ root string }

func (g gitRunner) run(args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", g.root}, args...)...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
