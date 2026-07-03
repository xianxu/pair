package launcher

import (
	"io"
	"os"
	"strconv"
	"time"
)

// LaunchNative is the process-level entry the cmd/pair-go PAIR_NATIVE_LAUNCH gate
// calls (#99 M2): it parses launchArgs, resolves the launch Env from the OS, and
// drives RunLaunch against the real OSRuntime. It returns ErrFallbackToShell for
// anything the native create path doesn't own yet (an unsupported verb like
// `continue`/`rename`/`list`, or a decision that resolves to attach/pick) so the
// caller defers to bin/pair-shell. Any other return is handled: the int is the
// exit code and user-facing messages are already on stderr.
func LaunchNative(launchArgs []string, pairHome string, stderr io.Writer) (int, error) {
	args, err := ParseArgs(launchArgs)
	if err != nil {
		return 0, ErrFallbackToShell // unsupported verb — shell owns it (until M5).
	}

	home := os.Getenv("HOME")
	xdg := os.Getenv("XDG_DATA_HOME")
	cwd, err := os.Getwd()
	if err != nil {
		return 0, ErrFallbackToShell
	}
	dataDir := ResolveDataDir(home, xdg)

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
	return RunLaunch(opts, NewOSRuntime(dataDir, pairHome), stderr)
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
