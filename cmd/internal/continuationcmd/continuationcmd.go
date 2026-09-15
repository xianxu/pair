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
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
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
		pairTag:         os.Getenv("PAIR_TAG"),
		dataDir:         adapt.DataDir(),
		zellijSession:   os.Getenv("ZELLIJ_SESSION_NAME"),
		pairSessionName: os.Getenv("PAIR_SESSION_NAME"),
	}
	// Reinvoke the launcher's exact-checkpoint ingress with the committed digest.
	// Hosted continuation is accepted durably by Couch; standalone continuation
	// is handed to the existing outer launcher before the source is stopped.

	restart := func(path, digest string) error {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return newContinueRestartCmd(exe, path, digest, stdin, stdout, stderr).Run()
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
	pairTag, dataDir, zellijSession, pairSessionName string
}

// newContinueRestartCmd builds the `pair continue --checkpoint <absolute-path>` command the writer runs
// to trigger the compaction restart. It sets PAIR_FAKE_IN_ZELLIJ=1 for a specific
// reason (found by #105's live smoke): the writer has ALREADY confirmed the
// compaction context via InCompactionContext (the ZELLIJ_SESSION_NAME tag-match,
// which needs no process introspection), but the child `pair continue` re-derives
// "am I in a pane?" through a process-ancestry walk (the launcher's InZellijPane),
// and the agent's command sandbox blocks process introspection (EPERM). Without
// the fake, that walk fails under a sandboxed agent shell and the restart misfires
// (compactionDecision goes false → it tries to launch instead of compact). This is
// the actual "restart stopped working" root cause. PAIR_FAKE_IN_ZELLIJ fakes ONLY
// the ancestry half; `pair continue`'s own ZELLIJ_SESSION_NAME tag-match still runs,
// so it can't compact the wrong session.
func newContinueRestartCmd(exe, path, digest string, stdin io.Reader, stdout, stderr io.Writer) *exec.Cmd {
	c := exec.Command(exe, "continue", "--checkpoint", path)
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	c.Env = append(os.Environ(), "PAIR_FAKE_IN_ZELLIJ=1", checkpoint.DigestEnv+"="+digest)
	return c
}

// run is the thin orchestration over the pure core: resolve inputs, write the
// file, then commit + push. Clock and stdin are injected so it's testable; git
// + fs are the real IO seam (the integration test drives the built binary
// against a real temp repo).
func run(a runArgs, env runEnv, now func() time.Time, stdin io.Reader, stdout io.Writer, restart func(string, string) error) error {
	root := a.repoRoot
	if root == "" {
		out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return fmt.Errorf("resolve repo root: %w", err)
		}
		root = strings.TrimSpace(string(out))
	}

	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve absolute repo root: %w", err)
	}

	body, err := readBody(a.bodyFile, stdin)
	if err != nil {
		return err
	}
	// #105: in a compaction context, fold the operator's parked draft WIP into
	// NEXT ACTION *before* writing, so the persisted+committed doc carries it —
	// otherwise the restart's draft re-seed (createflow.go) discards it. Done
	// before the HasNextAction guard so folded WIP can round out a thin section.
	if !a.noRestart && InCompactionContext(env.pairTag, env.zellijSession, env.pairSessionName) {
		paths, pathErr := artifactpath.ResolveScoped(env.dataDir, env.pairTag)
		if pathErr != nil {
			return fmt.Errorf("resolve draft artifact: %w", pathErr)
		}
		draft := paths.Draft()
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
	snapshot, err := checkpoint.New(abs, Assemble(RenderFrontmatter(f), body))
	if err != nil {
		return err
	}
	if err := os.WriteFile(abs, []byte(snapshot.Body), 0o644); err != nil {
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
	// first). --no-restart opts out. The selected owner executes the restart
	// from this exact committed snapshot; a failed request preserves the commit.
	if !a.noRestart && InCompactionContext(env.pairTag, env.zellijSession, env.pairSessionName) {
		if err := restart(abs, snapshot.Digest); err != nil {
			return fmt.Errorf("restart failed; checkpoint kept at %s: %w", abs, err)
		}
	}
	return nil
}

func readBody(bodyFile string, stdin io.Reader) (string, error) {
	if bodyFile == "" {
		return "", fmt.Errorf("-body-file is required")
	}
	reader := stdin
	if bodyFile != "-" {
		f, err := os.Open(bodyFile)
		if err != nil {
			return "", err
		}
		defer f.Close()
		reader = f
	}
	raw, err := io.ReadAll(io.LimitReader(reader, checkpoint.MaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > checkpoint.MaxBytes {
		return "", fmt.Errorf("continuation body exceeds %d bytes", checkpoint.MaxBytes)
	}
	return string(raw), nil
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
