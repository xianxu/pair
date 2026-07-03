package launcher

import (
	"io"
	"os"
	"strconv"
	"time"
)

// LaunchNative is the process-level entry the cmd/pair-go launch gate calls
// (#99 M2; the DEFAULT entry as of M4): it parses launchArgs, resolves the launch
// Env from the OS, and drives RunLaunch (or the read-only `list`/`ls`, the
// `rename`/`continue` subcommands, and in-session compaction — all native as of
// M5b) against the real OSRuntime. It returns ErrFallbackToShell only for what the
// shell still owns until M5c: a leading flag (`--help`/`-h`) and the shell-only
// test/debug seams (`shellOnlySeamActive`). Any other return is handled: the int
// is the exit code and user-facing messages are already on stdout/stderr.
func LaunchNative(launchArgs []string, pairHome string, stdout, stderr io.Writer) (int, error) {
	args, err := ParseArgs(launchArgs)
	if err != nil {
		return 0, ErrFallbackToShell // still-shell verb / leading flag — shell owns it (until M5c).
	}

	// Shell-only test/debug seams have no native equivalent — bin/pair-shell
	// short-circuits them early (before any zellij/fzf), so decline here too. Under
	// M4 they reached the shell only when the launch happened to decline (the pick
	// fallback); making pick+compaction native removed those paths, so route them
	// explicitly (until the shell retires at M5c). PAIR_TEST_CALL dispatches a
	// shell helper; PAIR_DEBUG_ARGS/PAIR_DEBUG_HISTORY resolve argv/history + exit.
	// Without this, a `pair continue …` probe would enter the native path and block
	// on the create prompt (the M5a fzf-/dev/tty class of hang).
	if shellOnlySeamActive() {
		return 0, ErrFallbackToShell
	}

	home := os.Getenv("HOME")
	xdg := os.Getenv("XDG_DATA_HOME")
	cwd, err := os.Getwd()
	if err != nil {
		return 0, ErrFallbackToShell
	}
	dataDir := ResolveDataDir(home, xdg)
	rt := NewOSRuntime(dataDir, pairHome)

	// `list`/`ls` is a read-only listing that prints to stdout and exits — no
	// launch, no zellij handoff (#99 M5a).
	if args.Command == "list" {
		return runList(rt, stdout, stderr), nil
	}

	// `rename <old> <new>` is an offline sidecar move — no launch (#99 M5b).
	if args.Command == "rename" {
		return runRename(rt, args, dataDir, stdout, stderr), nil
	}

	// Bare `continue` lists the docs + exits; it never launches (#99 M5b).
	if args.Command == "continue" && args.ContinueSlug == "" {
		return runContinueList(rt, stdout, stderr), nil
	}

	env := Env{
		Home:     home,
		XDGData:  xdg,
		Cwd:      cwd,
		Now:      time.Now(),
		HistoryD: historyDays(),
		DataDir:  dataDir,
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

// shellOnlySeamActive reports whether a shell-only test/debug seam env var is set
// — the native launcher declines these to bin/pair-shell (until M5c).
func shellOnlySeamActive() bool {
	return os.Getenv("PAIR_TEST_CALL") != "" ||
		os.Getenv("PAIR_DEBUG_ARGS") != "" ||
		os.Getenv("PAIR_DEBUG_HISTORY") != ""
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
