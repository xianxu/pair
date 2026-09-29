package reviewcmd

import (
	"encoding/json"
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
		ScopeKey:  getenv("PAIR_SCOPE_KEY"),
		SessionID: getenv("PAIR_SESSION_ID"),
	}, NewOSRuntime(), stdout, stderr)
}

// RunDefinitionCLI is the pair-review-definition command body.
func RunDefinitionCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	term := ""
	var reviewContext json.RawMessage
	seen := map[string]bool{}
	for len(args) > 0 && (args[0] == "--term" || args[0] == "--context") {
		flag := args[0]
		if len(args) < 2 || seen[flag] {
			fmt.Fprintln(stderr, "pair-review-definition: flags need one value and cannot repeat")
			return 2
		}
		seen[flag] = true
		if flag == "--term" {
			term = args[1]
		} else {
			reviewContext = json.RawMessage(args[1])
			if len(reviewContext) == 0 {
				fmt.Fprintln(stderr, "pair-review-definition: empty context")
				return 2
			}
		}
		args = args[2:]
	}

	if len(args) < 2 {
		fmt.Fprintf(stderr, "usage: pair-review-definition [--term TERM] [--context JSON] <request-id> <definition...>\n")
		return 2
	}
	return RunDefinition(DefinitionOptions{
		Context:    reviewContext,
		RequestID:  args[0],
		Term:       term,
		Definition: strings.Join(args[1:], " "),
		Tag:        getenv("PAIR_TAG"),
		Agent:      getenv("PAIR_AGENT"),
		DataDir:    getenv("PAIR_DATA_DIR"),
		ScopeKey:   getenv("PAIR_SCOPE_KEY"),
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
	if len(args) > 0 && args[0] == "--resolve" {
		snapshot := len(args) > 1 && args[len(args)-1] == "--snapshot"
		if snapshot {
			args = args[:len(args)-1]
		}
		if len(args) != 2 && (len(args) != 6 || args[2] != "--selected" || args[4] != "--head") {
			fmt.Fprintln(stderr, "usage: pair review readiness --resolve <directory> [--selected <relative-file> --head <sha>] [--snapshot]")
			return 2
		}
		selected, head := "", ""
		if len(args) == 6 {
			selected, head = args[3], args[5]
		}
		result := resolveIdentityWithSnapshot(NewOSRuntime(), args[1], selected, head, snapshot)
		encoded, _ := json.Marshal(result)
		if len(encoded) > identityLimit {
			result = ReviewIdentity{Status: "invalid", Diagnostic: "encoded review observation exceeds 8 MiB"}
		}
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			return 1
		}
		return 0
	}

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
		ScopeKey:  getenv("PAIR_SCOPE_KEY"),
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
