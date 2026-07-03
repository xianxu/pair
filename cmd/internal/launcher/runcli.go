package launcher

import (
	"io"
	"os"
	"strconv"
	"time"
)

// LaunchNative is the process-level entry the cmd/pair-go launch gate calls
// (#99 M2; the DEFAULT entry as of M4): it parses launchArgs, resolves the launch
// Env from the OS, and drives RunLaunch (or the read-only `list` subcommand)
// against the real OSRuntime. It returns ErrFallbackToShell for anything the
// native launcher doesn't own yet — a still-shell verb (`continue`/`rename`), a
// leading flag (`--help`/`-h`, shell-owned), an in-pane launch (compaction), or a
// rename/continue restart re-entry — so the caller defers to bin/pair-shell. Any
// other return is handled: the int is the exit code and user-facing messages are
// already on stdout/stderr.
func LaunchNative(launchArgs []string, pairHome string, stdout, stderr io.Writer) (int, error) {
	args, err := ParseArgs(launchArgs)
	if err != nil {
		return 0, ErrFallbackToShell // still-shell verb / leading flag — shell owns it (until M5c).
	}

	// PAIR_TEST_CALL dispatches a shell-internal helper for headless unit-testing
	// (bin/pair-shell short-circuits it early, before any zellij/fzf). The native
	// launcher has no equivalent, so decline before touching zellij/fzf — else a
	// bare `pair` here would enter the M5a native pick and block on fzf's /dev/tty.
	// Under M4 these invocations reached the shell only via the pick's fallback;
	// making the pick native removed that path, so route them explicitly (until
	// the shell retires at M5c). Same rationale as the PAIR_LEGACY_LAUNCH gate.
	if os.Getenv("PAIR_TEST_CALL") != "" {
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
		return runList(rt, stdout), nil
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
