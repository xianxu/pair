package titlepoller

import (
	"io"
	"os/signal"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// RunCLI is the pair-title command body, shared by the bin/pair-title.sh shim
// (which re-execs the Go binary) and any future `pair title` route. It parses
// argv into Options and drives the poller; getenv/stderr are injected so it is
// testable, and it no-ops (exit 0) when required args are missing.
func RunCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	if len(args) < 2 {
		return 0
	}

	// Ignore SIGHUP: bin/pair spawns us with `& disown`, which only removes the
	// job-table entry — the poller still shares a controlling tty with the
	// launching shell, so a terminal teardown would SIGHUP us and freeze the
	// titles. We exit only via the explicit "session gone" branch in Run.
	signal.Ignore(syscall.SIGHUP)

	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		dataDir = adapt.DataDir()
	}
	opts := Options{
		Tag:             args[0],
		Agent:           args[1],
		DataDir:         dataDir,
		Home:            getenv("HOME"),
		CmuxWorkspaceID: getenv("CMUX_WORKSPACE_ID"),
	}
	return Run(opts, NewOSRuntime())
}
