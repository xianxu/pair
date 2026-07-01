package opener

import "io"

// RunScrollbackCLI is the pair-scrollback-open command body (Alt+/). It reads
// the session from pair's env and parses the optional `--jump prev|next`.
func RunScrollbackCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	opts := Options{
		Tag:      getenv("PAIR_TAG"),
		Agent:    getenv("PAIR_AGENT"),
		DataDir:  getenv("PAIR_DATA_DIR"),
		PairHome: getenv("PAIR_HOME"),
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "--jump" && i+1 < len(args) {
			opts.Jump = args[i+1]
			i++
		}
	}
	return RunScrollback(opts, NewOSRuntime(), stderr)
}

// RunChangelogCLI is the pair-changelog-open command body (Alt+l).
func RunChangelogCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	opts := Options{
		Tag:       getenv("PAIR_TAG"),
		Agent:     getenv("PAIR_AGENT"),
		DataDir:   getenv("PAIR_DATA_DIR"),
		PairHome:  getenv("PAIR_HOME"),
		SessionID: getenv("PAIR_SESSION_ID"),
	}
	return RunChangelog(opts, NewOSRuntime(), stderr)
}
