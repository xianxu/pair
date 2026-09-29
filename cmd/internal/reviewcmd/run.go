package reviewcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Runtime is the IO/process boundary for the review helpers. The fs primitives
// (ReadFile/WriteFile/WriteAtomic/Remove/FileSize) come from an embedded osfs.FS
// on the OSRuntime; git/nvim-classify/zellij-spawn/session inventory are the domain seams.
type Runtime interface {
	ReadIdentityFile(context.Context, string, string, int) (string, error)
	GitContext(context.Context, int, string, ...string) (string, error)
	CanonicalDir(string) (string, error)
	RegularFileWithin(string, string) error
	ReadFile(path string) (string, error)
	WriteFile(path, data string) error
	WriteAtomic(path, data string) error // for review-target-<tag>.json (nvim Alt+c re-reads it)
	Remove(path string)
	FileSize(path string) (int64, bool)

	ProcessAlive(pid string) bool
	Kill(pid string)

	// AbsFile returns file as a logical absolute path (target's `cd dir && pwd`),
	// leaving it unchanged when its directory doesn't exist.
	AbsFile(file string) string
	// LogicalDir returns file's directory as a logical absolute path (open's
	// `cd dir && pwd`).
	LogicalDir(file string) string
	// PhysicalDir returns file's directory as a physical (symlink-resolved) path
	// (readiness's `pwd -P`), or "" when it can't be resolved.
	PhysicalDir(file string) string

	// Git runs `git -C dir <args…>` and returns stdout (untrimmed) + error.
	Git(dir string, args ...string) (string, error)
	// Classify runs the pure nvim/review/readiness.lua classifier via
	// `nvim --headless` (the single source of the 4-case decision).
	Classify(readinessLua string, f ReadinessFacts) (string, error)
	// SpawnReviewPane opens the floating nvim review pane (zellij run …).
	SpawnReviewPane(cwd, lua, absFile, nvimPidFile string) error
	// EstablishedSessionID returns inventory authority without compatibility or
	// live-process fallback.
	EstablishedSessionID(dataDir, scopeKey, tag, agent string) (string, sessioninventory.BindingStatus)
}

// ── target ────────────────────────────────────────────────────────────────

type TargetOptions struct {
	File, Status                 string
	Tag, Agent                   string
	DataDir, ScopeKey, SessionID string
}

func RunTarget(opts TargetOptions, rt Runtime, stdout, stderr io.Writer) int {
	if opts.Status != "proposed" && opts.Status != "ready" {
		fmt.Fprintf(stderr, "pair-review-target: status must be proposed|ready\n")
		return 2
	}
	if opts.DataDir == "" {
		fmt.Fprintf(stderr, "pair-review-target: PAIR_DATA_DIR not set\n")
		return 1
	}
	tag := orDefault(opts.Tag, "default")
	paths, err := artifactpath.ResolveScoped(opts.DataDir, tag)
	if err != nil {
		fmt.Fprintf(stderr, "pair-review-target: resolve artifact namespace: %v\n", err)
		return 1
	}
	agent := orDefault(opts.Agent, "claude")
	sid := resolveTargetSession(rt, opts.DataDir, opts.ScopeKey, tag, agent, opts.SessionID)
	file := rt.AbsFile(opts.File)

	out := paths.ReviewTarget()
	_ = rt.WriteAtomic(out, targetJSON(file, opts.Status, sid))
	fmt.Fprintf(stdout, "review target %s: %s (session %s)\n", opts.Status, file, orDefault(sid, "none"))
	return 0
}

// ── definition ────────────────────────────────────────────────────────────

type DefinitionOptions struct {
	Context                      json.RawMessage
	RequestID, Term              string
	Definition                   string
	Tag, Agent                   string
	DataDir, ScopeKey, SessionID string
}

func RunDefinition(opts DefinitionOptions, rt Runtime, stdout, stderr io.Writer) int {
	if len(opts.Context) > 0 {
		var fields map[string]string
		if err := json.Unmarshal(opts.Context, &fields); err != nil || len(fields) != 4 || fields["repo"] == "" || fields["branch"] == "" || fields["file"] == "" || fields["activation"] == "" {
			fmt.Fprintln(stderr, "pair-review-definition: context requires repo, branch, file and activation strings")
			return 2
		}
	}

	if opts.DataDir == "" {
		fmt.Fprintf(stderr, "pair-review-definition: PAIR_DATA_DIR not set\n")
		return 1
	}
	if strings.TrimSpace(opts.RequestID) == "" {
		fmt.Fprintf(stderr, "pair-review-definition: request id is required\n")
		return 2
	}
	if strings.TrimSpace(opts.Definition) == "" {
		fmt.Fprintf(stderr, "pair-review-definition: definition is required\n")
		return 2
	}
	tag := orDefault(opts.Tag, "default")
	paths, err := artifactpath.ResolveScoped(opts.DataDir, tag)
	if err != nil {
		fmt.Fprintf(stderr, "pair-review-definition: resolve artifact namespace: %v\n", err)
		return 1
	}
	agent := orDefault(opts.Agent, "claude")
	sid := resolveTargetSession(rt, opts.DataDir, opts.ScopeKey, tag, agent, opts.SessionID)
	out := paths.ReviewDefinitionResult()
	body, _ := json.Marshal(definitionDoc{RequestID: opts.RequestID, Term: opts.Term, Definition: opts.Definition, Session: sid, Context: opts.Context})
	if err := rt.WriteAtomic(out, string(body)); err != nil {
		fmt.Fprintf(stderr, "pair-review-definition: write %s: %v\n", out, err)
		return 1
	}
	fmt.Fprintf(stdout, "review definition %s: %s (session %s)\n", opts.RequestID, orDefault(opts.Term, "definition"), orDefault(sid, "none"))
	return 0
}

// resolveTargetSession implements the target seam's session priority:
// PAIR_SESSION_ID → established inventory root. Other binding states remain
// explicit absence and never fall through to config or live-process discovery.
func resolveTargetSession(rt Runtime, dataDir, scopeKey, tag, agent, envSID string) string {
	if envSID != "" {
		return envSID
	}
	if sid, status := rt.EstablishedSessionID(dataDir, scopeKey, tag, agent); status == sessioninventory.BindingEstablished {
		return sid
	}
	return ""
}

// ── open ──────────────────────────────────────────────────────────────────

type OpenOptions struct {
	File         string
	Tag, DataDir string
	PairHome     string
}

func RunOpen(opts OpenOptions, rt Runtime, stderr io.Writer) int {
	if opts.File == "" {
		fmt.Fprintf(stderr, "pair-review-open: needs a file argument\n")
		return 1
	}
	if _, ok := rt.FileSize(opts.File); !ok {
		fmt.Fprintf(stderr, "pair-review-open: %s not found\n", opts.File)
		return 1
	}
	if opts.DataDir == "" || opts.Tag == "" || opts.PairHome == "" {
		fmt.Fprintf(stderr, "pair-review-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_HOME\n")
		fmt.Fprintf(stderr, "  This is meant to run inside a pair session.\n")
		return 1
	}

	// A live pane owns its buffers and pending protocol state. Only its private
	// activation endpoint may retarget it; never replace it from this launcher.
	paths, err := artifactpath.ResolveScoped(opts.DataDir, opts.Tag)
	if err != nil {
		fmt.Fprintf(stderr, "pair-review-open: resolve artifact namespace: %v\n", err)
		return 1
	}
	state := paths.ReviewOpen()
	if content, err := rt.ReadFile(state); err == nil {
		if old := firstLine(content); old != "" && rt.ProcessAlive(old) {
			fmt.Fprintln(stderr, "pair-review-open: a live review pane exists; use Alt+C to activate it, or finish and close it first")
			return 1
		}
		rt.Remove(state)
	}

	dir := rt.LogicalDir(opts.File)
	abs := filepath.Join(dir, filepath.Base(opts.File))
	nvimPid := paths.NvimPID("review")
	if err := rt.SpawnReviewPane(dir, opts.PairHome+"/nvim/review.lua", abs, nvimPid); err != nil {
		fmt.Fprintf(stderr, "pair-review-open: %v\n", err)
		return 1
	}
	return 0
}

// ── readiness ───────────────────────────────────────────────────────────────

type ReadinessOptions struct {
	File                         string
	Prepare                      bool
	PairHome                     string
	Tag, Agent                   string
	DataDir, ScopeKey, SessionID string
}

// gitInfo holds the non-boolean git facts gathered alongside ReadinessFacts.
type gitInfo struct {
	abs, top, branch, scoped string
	identity                 ReviewIdentity
}

func RunReadiness(opts ReadinessOptions, rt Runtime, stdout, stderr io.Writer) int {
	if opts.File == "" {
		fmt.Fprintf(stderr, "usage: pair-review-readiness [--prepare] <file>\n")
		return 2
	}
	readinessLua := filepath.Join(opts.PairHome, "nvim", "review", "readiness.lua")
	dir := rt.PhysicalDir(opts.File)
	facts, gi := gatherGitFacts(rt, dir, opts.File)
	// Explicit selection may establish the document before the first round.
	if opts.Prepare && facts.IsTracked && gi.identity.Status == "missing" {
		facts.FileMatches = true
	}

	reviewCase, err := rt.Classify(readinessLua, facts)
	if err != nil || reviewCase == "" {
		fmt.Fprintf(stderr, "pair-review-readiness: classify failed (nvim/readiness.lua)\n")
		return 1
	}

	if opts.Prepare {
		if facts.OnReviewBranch && (gi.identity.Status == "invalid" || gi.identity.Status == "ambiguous") {
			fmt.Fprintf(stdout, "review not prepared: %s %s.\n", gi.identity.Status, gi.identity.Diagnostic)
			return 1
		}
		return prepare(opts, rt, stdout, reviewCase, facts, gi)
	}

	doc := readinessDoc{
		Case: reviewCase, IsGit: facts.IsGit, IsTracked: facts.IsTracked,
		Branch: gi.branch, OnReviewBranch: facts.OnReviewBranch,
		ScopedFile: gi.scoped, FileMatches: facts.FileMatches, IsClean: facts.IsClean,
	}
	b, _ := json.Marshal(doc)
	fmt.Fprintln(stdout, string(b))
	return 0
}

// gatherGitFacts runs the read-only git probes that feed the classifier.
func gatherGitFacts(rt Runtime, dir, file string) (ReadinessFacts, gitInfo) {
	var f ReadinessFacts
	var gi gitInfo
	if dir == "" {
		return f, gi
	}
	if _, err := rt.Git(dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return f, gi
	}
	f.IsGit = true
	gi.abs = filepath.Join(dir, filepath.Base(file))
	if top, err := rt.Git(dir, "rev-parse", "--show-toplevel"); err == nil {
		gi.top = strings.TrimSpace(top)
	}
	if gi.top != "" && strings.HasPrefix(gi.abs, gi.top+"/") {
		rel := strings.TrimPrefix(gi.abs, gi.top+"/")
		if _, err := rt.Git(gi.top, "ls-files", "--error-unmatch", "--", rel); err == nil {
			f.IsTracked = true
		}
	}
	if br, err := rt.Git(dir, "branch", "--show-current"); err == nil {
		gi.branch = strings.TrimSpace(br)
	}
	f.OnReviewBranch = strings.HasPrefix(gi.branch, "review/")
	if st, _ := rt.Git(dir, "status", "--porcelain"); strings.TrimSpace(st) == "" {
		f.IsClean = true
	}
	if f.OnReviewBranch {
		identity := resolveIdentity(rt, dir, "", "")
		gi.identity = identity
		if identity.Status == "resolved" {
			gi.scoped = identity.File
			f.FileMatches = filepath.Join(identity.Repo, identity.File) == gi.abs
		}
	}

	return f, gi
}

// prepare performs the deterministic --prepare git effects + marks the target
// ready + prints the agent ack. Mirrors pair-review-readiness's --prepare block.
func prepare(opts ReadinessOptions, rt Runtime, stdout io.Writer, reviewCase string, facts ReadinessFacts, gi gitInfo) int {
	switch reviewCase {
	case "stop":
		fmt.Fprintf(stdout, "review not prepared: %s is not in a git repo; ask the operator how to proceed.\n", opts.File)
		return 1
	case "interact":
		fmt.Fprintf(stdout, "review not prepared: repo state needs operator choice for %s (branch %s, clean=%v).\n",
			gi.abs, orDefault(gi.branch, "detached"), facts.IsClean)
		return 1
	}

	reviewBranch := "review/" + slugify(gi.abs)
	action := reviewCase

	if reviewCase == "track" {
		if gi.top == "" {
			fmt.Fprintf(stdout, "review not prepared: cannot locate git root for %s.\n", opts.File)
			return 1
		}
		_, _ = rt.Git(gi.top, "add", "--", gi.abs)
		_, _ = rt.Git(gi.top, "commit", "-q", "-m", "review: track "+filepath.Base(gi.abs))
		rel := strings.TrimPrefix(gi.abs, gi.top+"/")
		if _, err := rt.Git(gi.top, "ls-files", "--error-unmatch", "--", rel); err != nil {
			fmt.Fprintf(stdout, "review not prepared: failed to track %s.\n", gi.abs)
			return 1
		}
		if st, _ := rt.Git(gi.top, "status", "--porcelain"); strings.TrimSpace(st) != "" {
			fmt.Fprintf(stdout, "review not prepared: tracked %s, but repo still has unrelated changes.\n", gi.abs)
			return 1
		}
		action = "tracked and started"
	}

	switch reviewCase {
	case "new", "track":
		if gi.top == "" {
			fmt.Fprintf(stdout, "review not prepared: cannot locate git root for %s.\n", opts.File)
			return 1
		}
		if _, err := rt.Git(gi.top, "show-ref", "--verify", "--quiet", "refs/heads/"+reviewBranch); err == nil {
			_, _ = rt.Git(gi.top, "checkout", "-q", reviewBranch)
			action = "resumed existing"
		} else {
			_, _ = rt.Git(gi.top, "checkout", "-q", "-b", reviewBranch)
			if reviewCase == "new" {
				action = "started"
			}
		}
	case "resume":
		reviewBranch = gi.branch
		action = "resumed"
	}

	// Verify checkout and tracked-file authority after every Git effect. A
	// successful explicit preparation is the only issuer of a first-open receipt.
	identity := resolveIdentity(rt, gi.top, "", "")
	if identity.Status == "missing" {
		rel, err := filepath.Rel(identity.Repo, gi.abs)
		if err == nil {
			identity = resolveIdentity(rt, gi.top, rel, identity.Head)
		}
	}
	if identity.Status != "resolved" || identity.Branch != reviewBranch || filepath.Join(identity.Repo, identity.File) != gi.abs {
		fmt.Fprintf(stdout, "review not prepared: cannot verify selected checkout: %s %s.\n", identity.Status, identity.Diagnostic)
		return 1
	}
	sid := resolveTargetSession(rt, opts.DataDir, opts.ScopeKey, orDefault(opts.Tag, "default"), orDefault(opts.Agent, "claude"), opts.SessionID)
	if opts.DataDir != "" {
		paths, err := artifactpath.ResolveScoped(opts.DataDir, orDefault(opts.Tag, "default"))
		if err != nil {
			fmt.Fprintf(stdout, "review not prepared: %v\n", err)
			return 1
		}
		body, _ := json.Marshal(targetDoc{File: gi.abs, Status: "ready", Session: sid, Identity: &identity})
		if err = rt.WriteAtomic(paths.ReviewTarget(), string(body)); err != nil {
			fmt.Fprintf(stdout, "review not prepared: cannot publish receipt: %v\n", err)
			return 1
		}
	}

	fmt.Fprintf(stdout, "review prepared: %s %s on %s. Do not load xx-fix for this ack; when asked to review this file, load the full xx-fix skill directly and follow its Pair review workbench protocol. Reply \"ready\".\n", action, gi.abs, reviewBranch)
	return 0
}

func orDefault(v, dflt string) string {
	if v == "" {
		return dflt
	}
	return v
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
