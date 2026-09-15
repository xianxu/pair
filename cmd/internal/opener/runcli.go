package opener

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"io"
)

// RunScrollbackCLI is the pair-scrollback-open command body (Alt+/). It reads
// the session from pair's env and parses the optional `--jump prev|next`.
func RunScrollbackCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	opts := Options{
		Tag:      getenv("PAIR_TAG"),
		Agent:    getenv("PAIR_AGENT"),
		DataDir:  getenv("PAIR_DATA_DIR"),
		PairHome: getenv("PAIR_HOME"),
		ScopeKey: getenv("PAIR_SCOPE_KEY"),
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "--jump" && i+1 < len(args) {
			opts.Jump = args[i+1]
			i++
		}
	}
	lease, err := storagegc.AcquireSelectedProcess(context.Background(), getenv, "scrollback-opener")
	if err != nil {
		fmt.Fprintf(stderr, "pair-scrollback-open: retention: %v\n", err)
		return 1
	}
	defer lease.Close()
	return RunScrollback(opts, NewOSRuntime(), stderr)
}

// RunChangelogCLI is the pair-changelog-open command body (Alt+l).
func RunChangelogCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	opts := Options{
		Tag:       getenv("PAIR_TAG"),
		Agent:     getenv("PAIR_AGENT"),
		DataDir:   getenv("PAIR_DATA_DIR"),
		PairHome:  getenv("PAIR_HOME"),
		ScopeKey:  getenv("PAIR_SCOPE_KEY"),
		SessionID: getenv("PAIR_SESSION_ID"),
	}
	lease, err := storagegc.AcquireSelectedProcess(context.Background(), getenv, "changelog-opener")
	if err != nil {
		fmt.Fprintf(stderr, "pair-changelog-open: retention: %v\n", err)
		return 1
	}
	defer lease.Close()
	return RunChangelog(opts, NewOSRuntime(), stderr)
}
