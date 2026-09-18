package launcher

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const useRepoDefaultEnv = "PAIR_USE_REPO_DEFAULT"

func consumeRepoDefaultPolicy(getenv func(string) string, unsetenv func(string) error) bool {
	useRepoDefault := getenv(useRepoDefaultEnv) == "1"
	_ = unsetenv(useRepoDefaultEnv)
	return useRepoDefault
}

func newLaunchOptions(args LaunchArgs, env Env, pairHome, dataDir string, useRepoDefault bool, getenv func(string) string, parkTimeout int) LaunchOptions {
	return LaunchOptions{
		Args:                 args,
		Env:                  env,
		PairHome:             pairHome,
		GlobalDataDir:        dataDir,
		CodexAltScreenOptOut: getenv("PAIR_CODEX_ALT_SCREEN") == "1",
		ParkPromptTimeout:    parkTimeout,
		// #55 compaction env, read from the pane (only consulted when a `continue`
		// launch sets ContinueSlug below).
		PairTag:          getenv("PAIR_TAG"),
		PairAgent:        getenv("PAIR_AGENT"),
		ZellijSession:    getenv("ZELLIJ_SESSION_NAME"),
		PairSession:      getenv("PAIR_SESSION_NAME"),
		ForceInSession:   getenv("PAIR_FORCE_IN_SESSION") == "1",
		FakeInZellij:     getenv("PAIR_FAKE_IN_ZELLIJ") == "1",
		SkipConfigPicker: useRepoDefault,
	}
}

// LaunchNative is the process-level entry the cmd/pair-go launch gate calls (#99
// M2; the sole launcher as of M5c — bin/pair-shell is retired). It parses
// launchArgs and dispatches: `--help`/`help` → native usage; `list`/`ls`,
// `rename`, bare `continue` → their subcommand; otherwise resolves the launch Env
// and drives RunLaunch (create / attach / pick / restart loop / compaction). Every
// path is handled here — the returned int is the exit code, user-facing messages
// are already on stdout/stderr, and the error is always nil (no shell to fall back
// to).
func LaunchNative(launchArgs []string, pairHome string, stdout, stderr io.Writer) (int, error) {
	expectedDigest := os.Getenv(checkpoint.DigestEnv)
	_ = os.Unsetenv(checkpoint.DigestEnv)
	useRepoDefault := consumeRepoDefaultPolicy(os.Getenv, os.Unsetenv)
	couchProfile := os.Getenv(CouchLaunchProfileEnv)
	_ = os.Unsetenv(CouchLaunchProfileEnv)
	args, err := ParseArgs(launchArgs)
	if err != nil {
		// A genuine usage error (a leading flag that isn't -h/--help, a bad verb
		// arg). The shell no longer exists to defer to (#99 M5c) — print it +
		// exit 2.
		_, _ = io.WriteString(stderr, err.Error()+"\n")
		return 2, nil
	}
	if couchProfile != "" {
		args, err = applyCouchLaunchEnvironment(args, couchProfile, useRepoDefault)
		if err != nil {
			_, _ = io.WriteString(stderr, "pair: "+err.Error()+"\n")
			return 2, nil
		}
	}

	// `pair --help` / `pair help` — native usage to stdout (#99 M5c).
	if args.Command == "help" {
		_, _ = io.WriteString(stdout, UsageText())
		return 0, nil
	}
	if args.Command == "version" {
		_, _ = io.WriteString(stdout, VersionText())
		return 0, nil
	}

	home := os.Getenv("HOME")
	xdg := os.Getenv("XDG_DATA_HOME")
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = io.WriteString(stderr, "pair: cannot determine working directory: "+err.Error()+"\n")
		return 1, nil
	}
	repoRoot := gitRootOrCwd(cwd)
	dataDir := ResolveDataDir(home, xdg)
	launchDataDir := ScopedLaunchDataDir(dataDir, repoRoot)
	if explicit := os.Getenv("PAIR_DATA_DIR"); explicit != "" {
		launchDataDir = explicit
	}
	env := Env{
		Home:             home,
		XDGData:          xdg,
		Cwd:              cwd,
		RepoRoot:         repoRoot,
		Now:              time.Now(),
		HistoryD:         historyDays(),
		DataDir:          launchDataDir,
		CouchThreadScope: os.Getenv("COUCH_THREAD_SCOPE"),
		CouchThreadTag:   os.Getenv("COUCH_THREAD_TAG"),
	}
	rt := NewScopedOSRuntime(dataDir, env.DataDir, pairHome)

	// `list`/`ls` is a read-only listing that prints to stdout and exits — no
	// launch, no zellij handoff (#99 M5a).
	if args.Command == "list" {
		return runList(rt, stdout, stderr), nil
	}

	// `rename <old> <new>` is an offline sidecar move — no launch (#99 M5b).
	if args.Command == "rename" {
		if env.CouchThreadScope != "" || env.CouchThreadTag != "" {
			fmt.Fprintln(stderr, "pair: hosted tag rename is unsupported; use Couch name to change the thread label")
			return 1, nil
		}
		return runRenameScoped(rt, args, env.DataDir, scopeKeyFromDataDir(dataDir, env.DataDir), stdout, stderr), nil
	}

	// Bare `continue` lists the docs + exits; it never launches (#99 M5b).
	if args.Command == "continue" && args.ContinueSlug == "" && args.ContinueCheckpoint == "" && args.ContinueRetry == "" {
		return runContinueList(rt, stdout, stderr), nil
	}

	// `restart`/`quit` are the nvim-keybind lifecycle writers (#94 M1, ported from
	// bin/pair-{restart,quit}.sh): write markers, exec kill-session. They need the
	// live ZELLIJ_SESSION_NAME the keybind fires under.
	if args.Command == "restart" {
		if env.CouchThreadScope != "" || env.CouchThreadTag != "" {
			fmt.Fprintln(stderr, "pair: hosted inner restart is unsupported; use Couch relaunch (Alt+n)")
			return 1, nil
		}
		return runRestart(rt, args, os.Getenv("ZELLIJ_SESSION_NAME"), os.Getenv("PAIR_TAG"), stderr), nil
	}
	if args.Command == "quit" {
		return runQuit(rt, os.Getenv("ZELLIJ_SESSION_NAME"), stderr), nil
	}
	opts := newLaunchOptions(args, env, pairHome, dataDir, useRepoDefault, os.Getenv, parkPromptTimeout())

	if args.Command == "continue" {
		if args.ContinueRetry != "" {
			if env.CouchThreadScope != "" || env.CouchThreadTag != "" {
				fmt.Fprintln(stderr, "pair: a hosted thread's continuation belongs to Couch: "+checkpoint.Exits(checkpoint.Failed, env.CouchThreadTag))
				return 1, nil
			}
			if err := prepareContinuationRetry(&opts, rt, args.ContinueRetry); err != nil {
				fmt.Fprintf(stderr, "pair: %v\n", err)
				return 1, nil
			}
		} else {
			path := args.ContinueCheckpoint
			if path == "" {
				slug, err := NormalizeTag(args.ContinueSlug)
				if err != nil {
					fmt.Fprintf(stderr, "pair: invalid continuation slug: %v\n", err)
					return 1, nil
				}
				var ok bool
				path, _, ok = rt.ResolveContinuationDoc(slug)
				if !ok {
					fmt.Fprintf(stderr, "pair: no readable continuation matching %q in %s\n", slug, continuationDirPath())
					return 1, nil
				}
				opts.ContinueSlug = slug
			}
			c, err := rt.ReadCheckpoint(path)
			if err != nil {
				fmt.Fprintf(stderr, "pair: %v\n", err)
				return 1, nil
			}
			if expectedDigest != "" && expectedDigest != c.Digest {
				fmt.Fprintln(stderr, "pair: checkpoint changed since the writer committed it; source kept running")
				return 1, nil
			}
			opts.ContinueCheckpoint = c
			opts.ContinueDoc = c.SourcePath
			opts.Args.Agent = firstNonEmpty(args.Agent, c.Agent(), "claude")
		}
	}

	return RunLaunch(opts, rt, stderr)
}

func applyCouchLaunchEnvironment(args LaunchArgs, raw string, useRepoDefault bool) (LaunchArgs, error) {
	resolved, source, err := ApplyCouchLaunchProfile(args, raw)
	if err != nil {
		return LaunchArgs{}, err
	}
	if (source == "repo-default") != useRepoDefault {
		return LaunchArgs{}, fmt.Errorf("couch launch profile argv provenance disagrees with PAIR_USE_REPO_DEFAULT")
	}
	return resolved, nil
}

func gitRootOrCwd(cwd string) string {
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return canonicalPath(cwd)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return canonicalPath(cwd)
	}
	return canonicalPath(root)
}

func canonicalPath(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return path
}

func historyDays() int {
	if v := os.Getenv("PAIR_HISTORY_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 14
}

// parkPromptTimeout reads PAIR_PARK_PROMPT_TIMEOUT (default 5, invalid → 5); a
// valid 0 is a legitimate "don't wait" (shell 1562-1563).
func parkPromptTimeout() int {
	if v := os.Getenv("PAIR_PARK_PROMPT_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 5
}
