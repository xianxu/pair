package sessionwatch

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// RunCLI is the `pair session-watch` command body. It parses argv into Options
// and drives the watcher; getenv/stderr are injected so it is testable, and it
// no-ops (exit 0) when required args are missing.
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

// CommandArgs serializes the internal session-watch process contract. Watcher
// metadata precedes -- so agent arguments that resemble watcher flags remain
// untouched.
func CommandArgs(exe, agent, tag, scopeKey, cwd, repoRoot, repoName string, launchOrdinal uint64, pidNotBefore time.Time, agentArgs []string) []string {
	args := []string{exe, "session-watch", agent, tag, cwd}
	args = append(args, "--scope-key", scopeKey, "--launch-ordinal", strconv.FormatUint(launchOrdinal, 10))
	if !pidNotBefore.IsZero() {
		args = append(args, "--pid-not-before", pidNotBefore.Format(time.RFC3339Nano))
	}
	if repoRoot != "" {
		args = append(args, "--repo-root", repoRoot)
	}
	if repoName != "" {
		args = append(args, "--repo-name", repoName)
	}
	args = append(args, "--")
	return append(args, agentArgs...)
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
	repoRoot := ""
	repoName := ""
	scopeKey := ""
	launchOrdinal := uint64(0)
	pidNotBefore := time.Time{}
	agentArgs := append([]string(nil), args[3:]...)
	for len(agentArgs) > 0 {
		if agentArgs[0] == "--" {
			agentArgs = append([]string(nil), agentArgs[1:]...)
			break
		}
		if len(agentArgs) >= 2 && agentArgs[0] == "--repo-root" {
			repoRoot = agentArgs[1]
			agentArgs = agentArgs[2:]
			continue
		}
		if len(agentArgs) >= 2 && agentArgs[0] == "--repo-name" {
			repoName = agentArgs[1]
			agentArgs = agentArgs[2:]
			continue
		}
		if len(agentArgs) >= 2 && agentArgs[0] == "--scope-key" {
			scopeKey = agentArgs[1]
			agentArgs = agentArgs[2:]
			continue
		}
		if len(agentArgs) >= 2 && agentArgs[0] == "--launch-ordinal" {
			parsed, err := strconv.ParseUint(agentArgs[1], 10, 64)
			if err != nil || parsed == 0 {
				return Options{}, false
			}
			launchOrdinal = parsed
			agentArgs = agentArgs[2:]
			continue
		}
		if agentArgs[0] == "--pid-not-before" {
			if len(agentArgs) < 2 {
				return Options{}, false
			}
			parsed, err := time.Parse(time.RFC3339Nano, agentArgs[1])
			if err != nil {
				return Options{}, false
			}
			pidNotBefore = parsed
			agentArgs = agentArgs[2:]
			continue
		}
		break
	}
	return Options{
		Agent:         args[0],
		Tag:           args[1],
		ScopeKey:      scopeKey,
		LaunchOrdinal: launchOrdinal,
		Cwd:           args[2],
		RepoRoot:      repoRoot,
		RepoName:      repoName,
		PIDNotBefore:  pidNotBefore,
		Args:          agentArgs,
		Home:          home,
		DataDir:       dataDir,
		PIDWait:       ParseDurationSeconds(getenv("PAIR_SESSION_WATCH_PID_WAIT_SECONDS"), 2*time.Second),
		Timeout:       60 * time.Second,
		Poll:          100 * time.Millisecond,
	}, true
}
