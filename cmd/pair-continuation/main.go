package main

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

func main() {
	a := runArgs{}
	flag.StringVar(&a.repoRoot, "repo-root", "", "repo root (default: git rev-parse --show-toplevel)")
	flag.StringVar(&a.slug, "slug", "", "continuation slug (required)")
	flag.StringVar(&a.agent, "agent", "", "original agent, e.g. claude (required)")
	flag.StringVar(&a.sessionID, "session-id", "", "native session id (provenance only)")
	flag.StringVar(&a.issuesCSV, "issues", "", "comma-separated issue ids (required)")
	flag.StringVar(&a.branch, "branch", "", "git branch")
	flag.StringVar(&a.worktree, "worktree", "", "local worktree path (a hint, not portable)")
	flag.StringVar(&a.supersedes, "supersedes", "", "prior continuation slug")
	flag.StringVar(&a.bodyFile, "body-file", "", "file holding the continuation body; '-' = stdin (required)")
	flag.Parse()

	if err := run(a, time.Now, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "pair-continuation: %v\n", err)
		os.Exit(1)
	}
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

	// Disaster-recovery: commit + push straight to main the instant it's
	// written (record artifact, not feature code). A push failure must NOT
	// lose the local recovery doc — warn and keep the commit.
	g := gitRunner{root: root}
	if out, err := g.run("add", rel); err != nil {
		return fmt.Errorf("git add: %v\n%s", err, out)
	}
	if out, err := g.run("commit", "-m", "continuation: "+f.Slug); err != nil {
		return fmt.Errorf("git commit: %v\n%s", err, out)
	}
	if out, err := g.run("push"); err != nil {
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
