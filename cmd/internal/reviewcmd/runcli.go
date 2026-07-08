package reviewcmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// RunTargetCLI is the pair-review-target command body.
func RunTargetCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintf(stderr, "usage: pair-review-target <file> <proposed|ready>\n")
		return 2
	}
	return RunTarget(TargetOptions{
		File:      args[0],
		Status:    args[1],
		Tag:       getenv("PAIR_TAG"),
		Agent:     getenv("PAIR_AGENT"),
		DataDir:   getenv("PAIR_DATA_DIR"),
		SessionID: getenv("PAIR_SESSION_ID"),
	}, NewOSRuntime(), stdout, stderr)
}

// RunDefinitionCLI is the pair-review-definition command body.
func RunDefinitionCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	term := ""
	if len(args) >= 2 && args[0] == "--term" {
		term = args[1]
		args = args[2:]
	}
	if len(args) < 2 {
		fmt.Fprintf(stderr, "usage: pair-review-definition [--term TERM] <request-id> <definition...>\n")
		return 2
	}
	return RunDefinition(DefinitionOptions{
		RequestID:  args[0],
		Term:       term,
		Definition: strings.Join(args[1:], " "),
		Tag:        getenv("PAIR_TAG"),
		Agent:      getenv("PAIR_AGENT"),
		DataDir:    getenv("PAIR_DATA_DIR"),
		SessionID:  getenv("PAIR_SESSION_ID"),
	}, NewOSRuntime(), stdout, stderr)
}

// RunOpenCLI is the pair-review-open command body.
func RunOpenCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	file := ""
	if len(args) > 0 {
		file = args[0]
	}
	return RunOpen(OpenOptions{
		File:     file,
		Tag:      getenv("PAIR_TAG"),
		DataDir:  getenv("PAIR_DATA_DIR"),
		PairHome: getenv("PAIR_HOME"),
	}, NewOSRuntime(), stderr)
}

// RunReadinessCLI is the pair-review-readiness command body.
func RunReadinessCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	prepare := false
	if len(args) > 0 && args[0] == "--prepare" {
		prepare = true
		args = args[1:]
	}
	file := ""
	if len(args) > 0 {
		file = args[0]
	}
	home := getenv("PAIR_HOME")
	if home == "" {
		home = repoRootFromExe()
	}
	return RunReadiness(ReadinessOptions{
		File:      file,
		Prepare:   prepare,
		PairHome:  home,
		Tag:       getenv("PAIR_TAG"),
		Agent:     getenv("PAIR_AGENT"),
		DataDir:   getenv("PAIR_DATA_DIR"),
		SessionID: getenv("PAIR_SESSION_ID"),
	}, NewOSRuntime(), stdout, stderr)
}

// repoRootFromExe mirrors the shell's `PAIR_HOME:-$(cd $(dirname $0)/.. && pwd)`
// fallback: the binary lives at <root>/bin/pair-review-readiness.
func repoRootFromExe() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(filepath.Dir(exe))
}
