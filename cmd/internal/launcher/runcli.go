package launcher

import (
	"io"
	"os"
	"strconv"
	"time"
)

// LaunchNative is the process-level entry the cmd/pair-go launch gate calls (#99
// M2; the sole launcher as of M5c — bin/pair-shell is retired). It parses
// launchArgs and dispatches: `--help`/`help` → native usage; `list`/`ls`,
// `rename`, bare `continue` → their subcommand; otherwise resolves the launch Env
// and drives RunLaunch (create / attach / pick / restart loop / compaction). Every
// path is handled here — the returned int is the exit code, user-facing messages
// are already on stdout/stderr, and the error is always nil (no shell to fall back
// to).
func LaunchNative(launchArgs []string, pairHome string, stdout, stderr io.Writer) (int, error) {
	args, err := ParseArgs(launchArgs)
	if err != nil {
		// A genuine usage error (a leading flag that isn't -h/--help, a bad verb
		// arg). The shell no longer exists to defer to (#99 M5c) — print it +
		// exit 2.
		_, _ = io.WriteString(stderr, err.Error()+"\n")
		return 2, nil
	}

	// `pair --help` / `pair help` — native usage to stdout (#99 M5c).
	if args.Command == "help" {
		_, _ = io.WriteString(stdout, UsageText())
		return 0, nil
	}

	home := os.Getenv("HOME")
	xdg := os.Getenv("XDG_DATA_HOME")
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = io.WriteString(stderr, "pair: cannot determine working directory: "+err.Error()+"\n")
		return 1, nil
	}
	dataDir := ResolveDataDir(home, xdg)
	launchDataDir := ScopedLaunchDataDir(dataDir, cwd)
	if explicit := os.Getenv("PAIR_DATA_DIR"); explicit != "" {
		launchDataDir = explicit
	}
	env := Env{
		Home:     home,
		XDGData:  xdg,
		Cwd:      cwd,
		Now:      time.Now(),
		HistoryD: historyDays(),
		DataDir:  launchDataDir,
	}
	rt := NewScopedOSRuntime(dataDir, env.DataDir, pairHome)

	// `list`/`ls` is a read-only listing that prints to stdout and exits — no
	// launch, no zellij handoff (#99 M5a).
	if args.Command == "list" {
		return runList(rt, stdout, stderr), nil
	}

	// `rename <old> <new>` is an offline sidecar move — no launch (#99 M5b).
	if args.Command == "rename" {
		return runRenameScoped(rt, args, env.DataDir, scopeKeyFromDataDir(dataDir, env.DataDir), stdout, stderr), nil
	}

	// Bare `continue` lists the docs + exits; it never launches (#99 M5b).
	if args.Command == "continue" && args.ContinueSlug == "" {
		return runContinueList(rt, stdout, stderr), nil
	}

	// `restart`/`quit` are the nvim-keybind lifecycle writers (#94 M1, ported from
	// bin/pair-{restart,quit}.sh): write markers, exec kill-session. They need the
	// live ZELLIJ_SESSION_NAME the keybind fires under.
	if args.Command == "restart" {
		return runRestart(rt, args, os.Getenv("ZELLIJ_SESSION_NAME"), stderr), nil
	}
	if args.Command == "quit" {
		return runQuit(rt, os.Getenv("ZELLIJ_SESSION_NAME"), stderr), nil
	}
	opts := LaunchOptions{
		Args:                 args,
		Env:                  env,
		PairHome:             pairHome,
		CodexAltScreenOptOut: os.Getenv("PAIR_CODEX_ALT_SCREEN") == "1",
		ParkPromptTimeout:    parkPromptTimeout(),
		// #55 compaction env, read from the pane (only consulted when a `continue`
		// launch sets ContinueSlug below).
		PairTag:        os.Getenv("PAIR_TAG"),
		PairAgent:      os.Getenv("PAIR_AGENT"),
		ZellijSession:  os.Getenv("ZELLIJ_SESSION_NAME"),
		ForceInSession: os.Getenv("PAIR_FORCE_IN_SESSION") == "1",
		FakeInZellij:   os.Getenv("PAIR_FAKE_IN_ZELLIJ") == "1",
	}

	// `continue <slug>`: resolve the doc (seeds the draft on create + drives the
	// compaction marker), pick the agent (explicit port → doc frontmatter → claude).
	if args.Command == "continue" {
		slug, err := NormalizeTag(args.ContinueSlug)
		if err != nil {
			_, _ = io.WriteString(stderr, "pair: invalid slug '"+args.ContinueSlug+"'\n")
			return 1, nil
		}
		docPath, docAgent, ok := rt.ResolveContinuationDoc(slug)
		if !ok {
			_, _ = io.WriteString(stderr, "pair: no continuation matching '"+slug+"' in "+continuationDirPath()+"\n")
			return 1, nil
		}
		opts.ContinueDoc = docPath
		opts.ContinueSlug = slug
		opts.Args.Agent = firstNonEmpty(args.Agent, docAgent, "claude")
	}

	return RunLaunch(opts, rt, stderr)
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
