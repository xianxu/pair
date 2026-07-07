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

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// ContinuationDir is the repo-relative home for continuation instances (matches
// construct/datatype/continuation.md). Exported so the launcher's `pair continue`
// resolver shares one source for where continuations live (#99 M5b, ARCH-DRY).
const ContinuationDir = "workshop/continuation"

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
	fs.BoolVar(&a.noRestart, "no-restart", false, "don't restart the session after writing (escape hatch for a manual in-pane write)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Env inputs are read here (the non-injected outer seam) and threaded into
	// run() so the fold + restart logic stays testable with a fake env + seam.
	env := runEnv{
		pairTag:       os.Getenv("PAIR_TAG"),
		dataDir:       adapt.DataDir(),
		zellijSession: os.Getenv("ZELLIJ_SESSION_NAME"),
	}
	// Real restart seam: re-invoke `pair continue <slug>` on ourselves. Inherits
	// the env (PAIR_DEV/PAIR_TAG/ZELLIJ_SESSION_NAME ride through), so it re-enters
	// compaction (compactionDecision fires) under the SAME config, then the outer
	// reincarnation loop relaunches. The kill-session inside tears this process
	// down too — that's fine, the continuation is already written + pushed.
	restart := func(slug string) error {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		c := exec.Command(exe, "continue", slug)
		c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
		return c.Run()
	}

	if err := run(a, env, now, stdin, stdout, restart); err != nil {
		fmt.Fprintf(stderr, "pair-continuation: %v\n", err)
		return 1
	}
	return 0
}

type runArgs struct {
	repoRoot, slug, agent, sessionID, issuesCSV, branch, worktree, supersedes, bodyFile string
	noRestart                                                                           bool
}

// runEnv are the process-env inputs the writer's compaction behavior keys off.
// Populated in Run from the real environment; injected in tests.
type runEnv struct {
	pairTag, dataDir, zellijSession string
}

// run is the thin orchestration over the pure core: resolve inputs, write the
// file, then commit + push. Clock and stdin are injected so it's testable; git
// + fs are the real IO seam (the integration test drives the built binary
// against a real temp repo).
func run(a runArgs, env runEnv, now func() time.Time, stdin io.Reader, stdout io.Writer, restart func(string) error) error {
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
	// #105: in a compaction context, fold the operator's parked draft WIP into
	// NEXT ACTION *before* writing, so the persisted+committed doc carries it —
	// otherwise the restart's draft re-seed (createflow.go) discards it. Done
	// before the HasNextAction guard so folded WIP can round out a thin section.
	if !a.noRestart && InCompactionContext(env.pairTag, env.zellijSession) {
		draft := filepath.Join(env.dataDir, "draft-"+env.pairTag+".md")
		if raw, rerr := os.ReadFile(draft); rerr == nil {
			if wip := StripStickyComments(string(raw)); wip != "" {
				body = FoldDraftIntoNextAction(body, wip)
			}
		}
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

	dir := filepath.Join(root, ContinuationDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	existing, err := listMarkdown(dir)
	if err != nil {
		return err
	}
	name := AllocName(f.Slug, ts, existing)
	rel := filepath.ToSlash(filepath.Join(ContinuationDir, name))
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

	// #105: in a compaction context, the writer OWNS the restart — no agent step
	// to forget. Fires only after a successful write+commit (the doc is durable
	// first). --no-restart opts out (manual in-pane write). The seam kills the
	// session; the outer reincarnation loop relaunches fresh, seeded from the doc.
	if !a.noRestart && InCompactionContext(env.pairTag, env.zellijSession) {
		if err := restart(f.Slug); err != nil {
			fmt.Fprintf(os.Stderr, "pair-continuation: restart failed (continuation kept): %v\n", err)
		}
	}
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
