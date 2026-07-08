package reviewcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Runtime is the IO/process boundary for the review helpers. The fs primitives
// (ReadFile/WriteFile/WriteAtomic/Remove/FileSize) come from an embedded osfs.FS
// on the OSRuntime; git/nvim-classify/zellij-spawn/codex-sid are the domain seams.
type Runtime interface {
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
	// ResolveCodexSessionID walks the codex agent's process tree (codexsid).
	ResolveCodexSessionID(dataDir, tag string) string
}

// ── target ────────────────────────────────────────────────────────────────

type TargetOptions struct {
	File, Status       string
	Tag, Agent         string
	DataDir, SessionID string
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
	agent := orDefault(opts.Agent, "claude")
	sid := resolveTargetSession(rt, opts.DataDir, tag, agent, opts.SessionID)
	file := rt.AbsFile(opts.File)

	out := filepath.Join(opts.DataDir, "review-target-"+tag+".json")
	_ = rt.WriteAtomic(out, targetJSON(file, opts.Status, sid))
	fmt.Fprintf(stdout, "review target %s: %s (session %s)\n", opts.Status, file, orDefault(sid, "none"))
	return 0
}

// ── definition ────────────────────────────────────────────────────────────

type DefinitionOptions struct {
	RequestID, Term    string
	Definition         string
	Tag, Agent         string
	DataDir, SessionID string
}

func RunDefinition(opts DefinitionOptions, rt Runtime, stdout, stderr io.Writer) int {
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
	agent := orDefault(opts.Agent, "claude")
	sid := resolveTargetSession(rt, opts.DataDir, tag, agent, opts.SessionID)
	out := filepath.Join(opts.DataDir, "review-definition-result-"+tag+".json")
	_ = rt.WriteAtomic(out, definitionJSON(opts.RequestID, opts.Term, opts.Definition, sid))
	fmt.Fprintf(stdout, "review definition %s: %s (session %s)\n", opts.RequestID, orDefault(opts.Term, "definition"), orDefault(sid, "none"))
	return 0
}

// resolveTargetSession implements the target seam's session priority:
// PAIR_SESSION_ID → config session_id → (codex only) the live-rollout lsof walk.
func resolveTargetSession(rt Runtime, dataDir, tag, agent, envSID string) string {
	if envSID != "" {
		return envSID
	}
	if cfg, err := rt.ReadFile(filepath.Join(dataDir, "config-"+tag+"-"+agent+".json")); err == nil {
		if sid := sessionFromConfig(cfg); sid != "" {
			return sid
		}
	}
	if agent == "codex" {
		return rt.ResolveCodexSessionID(dataDir, tag)
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

	// Single review pane: replace any LIVE review (kill the old nvim → its
	// close_on_exit floating pane self-dismisses) before spawning the new one.
	state := filepath.Join(opts.DataDir, "review-"+opts.Tag+".open")
	if content, err := rt.ReadFile(state); err == nil {
		if old := firstLine(content); old != "" && rt.ProcessAlive(old) {
			rt.Kill(old)
		}
		rt.Remove(state)
	}

	dir := rt.LogicalDir(opts.File)
	abs := filepath.Join(dir, filepath.Base(opts.File))
	nvimPid := filepath.Join(opts.DataDir, "nvim-pid-"+opts.Tag+"-review")
	if err := rt.SpawnReviewPane(dir, opts.PairHome+"/nvim/review.lua", abs, nvimPid); err != nil {
		fmt.Fprintf(stderr, "pair-review-open: %v\n", err)
		return 1
	}
	return 0
}

// ── readiness ───────────────────────────────────────────────────────────────

type ReadinessOptions struct {
	File               string
	Prepare            bool
	PairHome           string
	Tag, Agent         string
	DataDir, SessionID string
}

// gitInfo holds the non-boolean git facts gathered alongside ReadinessFacts.
type gitInfo struct{ abs, top, branch, scoped string }

func RunReadiness(opts ReadinessOptions, rt Runtime, stdout, stderr io.Writer) int {
	if opts.File == "" {
		fmt.Fprintf(stderr, "usage: pair-review-readiness [--prepare] <file>\n")
		return 2
	}
	readinessLua := filepath.Join(opts.PairHome, "nvim", "review", "readiness.lua")
	dir := rt.PhysicalDir(opts.File)
	facts, gi := gatherGitFacts(rt, dir, opts.File)

	reviewCase, err := rt.Classify(readinessLua, facts)
	if err != nil || reviewCase == "" {
		fmt.Fprintf(stderr, "pair-review-readiness: classify failed (nvim/readiness.lua)\n")
		return 1
	}

	if opts.Prepare {
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
		if out, err := rt.Git(dir, "log", "-1", "--name-only", "--pretty=format:", "--grep=^review("); err == nil {
			for _, line := range strings.Split(out, "\n") {
				if strings.TrimSpace(line) != "" {
					gi.scoped = strings.TrimSpace(line)
					break
				}
			}
		}
		if gi.scoped != "" && gi.top != "" && filepath.Join(gi.top, gi.scoped) == gi.abs {
			f.FileMatches = true
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

	// Mark the target ready in-process (the shell shelled out to pair-review-target).
	sid := resolveTargetSession(rt, opts.DataDir, orDefault(opts.Tag, "default"), orDefault(opts.Agent, "claude"), opts.SessionID)
	if opts.DataDir != "" {
		out := filepath.Join(opts.DataDir, "review-target-"+orDefault(opts.Tag, "default")+".json")
		_ = rt.WriteAtomic(out, targetJSON(gi.abs, "ready", sid))
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

func sessionFromConfig(cfg string) string {
	var c struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal([]byte(cfg), &c) != nil {
		return ""
	}
	return c.SessionID
}
