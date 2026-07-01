// Package continuationcmd is the body of pair-continuation, shared by the
// bin/pair-continuation shim and the `pair continuation` dispatcher route. It
// reads the continuation body from stdin (--body-file -), so it runs on the
// streaming dispatch seam, not the buffered path.
package continuationcmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// continuationDir is the repo-relative home for continuation instances
// (matches construct/datatype/continuation.md).
const continuationDir = "workshop/continuation"

// Run parses flags from args and writes the continuation. now/stdin are injected
// (the clock-fake + stdin tests drive run() directly); real stdio is threaded in
// by the shim / streaming seam.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, now func() time.Time) int {
	fs := flag.NewFlagSet("pair-continuation", flag.ContinueOnError)
	fs.SetOutput(stderr)
	a := runArgs{}
	fs.StringVar(&a.repoRoot, "repo-root", "", "repo root (default: git rev-parse --show-toplevel)")
	fs.StringVar(&a.slug, "slug", "", "continuation slug (required)")
	fs.StringVar(&a.agent, "agent", "", "original agent, e.g. claude (required)")
	fs.StringVar(&a.sessionID, "session-id", "", "native session id (provenance only)")
	fs.StringVar(&a.issuesCSV, "issues", "", "comma-separated issue ids (required)")
	fs.StringVar(&a.branch, "branch", "", "git branch")
	fs.StringVar(&a.worktree, "worktree", "", "local worktree path (a hint, not portable)")
	fs.StringVar(&a.supersedes, "supersedes", "", "prior continuation slug")
	fs.StringVar(&a.bodyFile, "body-file", "", "file holding the continuation body; '-' = stdin (required)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := run(a, now, stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "pair-continuation: %v\n", err)
		return 1
	}
	return 0
}

type runArgs struct {
	repoRoot, slug, agent, sessionID, issuesCSV, branch, worktree, supersedes, bodyFile string
}

// run is the thin orchestration over the pure core: resolve inputs, write the
// file, then commit + push. Clock and stdin are injected so it's testable; git
// + fs are the real IO seam (the integration test drives the built binary
// against a real temp repo).
func run(a runArgs, now func() time.Time, stdin io.Reader, stdout io.Writer) error {
	root := a.repoRoot
	if root == "" {
		out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return fmt.Errorf("resolve repo root: %w", err)
		}
		root = strings.TrimSpace(string(out))
	}

	body, err := readBody(a.bodyFile, stdin)
	if err != nil {
		return err
	}
	// The writer is the structural guard (Spec Done-when): continuation.md
	// makes NEXT ACTION mandatory, so refuse a body that lacks it — or that has
	// only an empty heading with no actionable content (#52).
	if !HasNextAction(body) {
		return fmt.Errorf("continuation body must contain a non-empty '## NEXT ACTION' section")
	}

	ts := now()
	f := Fields{
		Slug: a.slug, Agent: a.agent, SessionID: a.sessionID, Created: ts,
		Supersedes: a.supersedes, Branch: a.branch, Worktree: a.worktree,
		Issues: splitCSV(a.issuesCSV),
	}
	if err := ValidateFields(f); err != nil {
		return err
	}

	dir := filepath.Join(root, continuationDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	existing, err := listMarkdown(dir)
	if err != nil {
		return err
	}
	name := AllocName(f.Slug, ts, existing)
	rel := filepath.ToSlash(filepath.Join(continuationDir, name))
	abs := filepath.Join(dir, name)
	if err := os.WriteFile(abs, []byte(Assemble(RenderFrontmatter(f), body)), 0o644); err != nil {
		return err
	}

	// Disaster-recovery: commit + push the instant it's written (record
	// artifact, not feature code) so the doc is durable and off-host.
	//   - the commit is PATH-SCOPED (`-- rel`) so an unrelated dirty index is
	//     never swept into the continuation commit (and unrelated staged work
	//     is left untouched);
	//   - push to origin/HEAD (the current branch — lands on main when that
	//     branch merges; avoids a fragile cross-branch push to main);
	//   - a push failure is non-fatal: a detached/offline park still keeps the
	//     local recovery commit.
	g := gitRunner{root: root}
	if out, err := g.run("add", rel); err != nil {
		return fmt.Errorf("git add: %v\n%s", err, out)
	}
	if out, err := g.run("commit", "-m", "continuation: "+f.Slug, "--", rel); err != nil {
		return fmt.Errorf("git commit: %v\n%s", err, out)
	}
	if out, err := g.run("push", "origin", "HEAD"); err != nil {
		fmt.Fprintf(os.Stderr, "pair-continuation: push failed (commit kept locally): %v\n%s\n", err, out)
	}
	fmt.Fprintln(stdout, abs)
	return nil
}

func readBody(bodyFile string, stdin io.Reader) (string, error) {
	switch bodyFile {
	case "":
		return "", fmt.Errorf("-body-file is required")
	case "-":
		b, err := io.ReadAll(stdin)
		return string(b), err
	default:
		b, err := os.ReadFile(bodyFile)
		return string(b), err
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func listMarkdown(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
