package sessionwatch

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// RunCLI is the pair-session-watch command body, shared by the
// bin/pair-session-watch shim and the `pair session-watch` dispatcher route. It
// parses argv into Options and drives the watcher; getenv/stderr are injected so
// it is testable, and it no-ops (exit 0) when required args are missing.
func RunCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	opts, ok := buildOptions(args, getenv)
	if !ok {
		return 0
	}
	cleanupPairTag := ensurePairTag(opts.Tag)
	defer cleanupPairTag()
	logger := adapt.Open("session-watch", opts.Agent)
	defer logger.Close()
	if err := Run(opts, NewOSRuntime(logger)); err != nil {
		fmt.Fprintf(stderr, "pair-session-watch: %v\n", err)
		return 1
	}
	return 0
}

func ensurePairTag(tag string) func() {
	if os.Getenv("PAIR_TAG") != "" || tag == "" {
		return func() {}
	}
	_ = os.Setenv("PAIR_TAG", tag)
	return func() { _ = os.Unsetenv("PAIR_TAG") }
}

func buildOptions(args []string, getenv func(string) string) (Options, bool) {
	if len(args) < 3 {
		return Options{}, false
	}
	home := getenv("HOME")
	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		dataDir = adapt.DataDir()
	}
	return Options{
		Agent:   args[0],
		Tag:     args[1],
		Cwd:     args[2],
		Args:    append([]string(nil), args[3:]...),
		Home:    home,
		DataDir: dataDir,
		PIDWait: ParseDurationSeconds(getenv("PAIR_SESSION_WATCH_PID_WAIT_SECONDS"), 2*time.Second),
		Timeout: 60 * time.Second,
		Poll:    100 * time.Millisecond,
	}, true
}
