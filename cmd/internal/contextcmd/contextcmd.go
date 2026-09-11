// Package contextcmd implements the pair-context command body.
package contextcmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/ctxmeter"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type Env struct {
	Home         string
	XDGDataHome  string
	PairDataDir  string
	PairScopeKey string
}

// The Pair-session variables this package reads. Named once, because the title
// poller's spawn contract (titlepoller.SessionEnv) has to render exactly these
// (pair#183).
const (
	EnvDataDir  = "PAIR_DATA_DIR"
	EnvScopeKey = "PAIR_SCOPE_KEY"
)

// ExitNoScopeKey is Run's status when no repo scope key was supplied.
//
// Kept apart from "no established binding" -- status 0, nothing printed -- which
// is an ordinary state for a session that has not bound yet. An empty key is a
// caller that forgot to pass scope: sessionledger rejects every record without
// one, so no session can ever match it. pair#183 was exactly that caller, and it
// read as a missing feature because both states printed the same nothing.
const ExitNoScopeKey = 2

func EnvFromOS() Env { return EnvFrom(os.Getenv) }

// EnvFrom reads Env through getenv -- injected, so a spawner's contract can be
// checked against what this package actually reads rather than against a list.
func EnvFrom(getenv func(string) string) Env {
	return Env{
		Home:         getenv("HOME"),
		XDGDataHome:  getenv("XDG_DATA_HOME"),
		PairDataDir:  getenv(EnvDataDir),
		PairScopeKey: getenv(EnvScopeKey),
	}
}

func Run(args []string, env Env, stdout, stderr io.Writer) int {
	dataDir := resolveDataDir(env)
	runtime := sessioninventory.NewOSRuntime(env.Home, dataDir)
	return RunWithRuntime(args, env, runtime, stdout, stderr)
}

// RunWithRuntime applies the production established-binding query through an
// injected inventory runtime.
func RunWithRuntime(args []string, env Env, runtime sessioninventory.Runtime, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		return 0
	}
	if env.PairScopeKey == "" {
		fmt.Fprintf(stderr, "pair context: %s is not set, so no session can match; whoever started this did not pass the repo scope\n", EnvScopeKey)
		return ExitNoScopeKey
	}
	tag, agent := args[0], args[1]
	query, err := sessioninventory.QuerySession(runtime, env.PairScopeKey, tag, sessioninventory.Agent(agent))
	if err != nil || query.Status != sessioninventory.BindingEstablished || query.Root == nil {
		return 0
	}
	if usage, ok, usageErr := sessioninventory.TokenUsageForRoot(runtime, *query.Root); usageErr == nil && ok {
		fmt.Fprintln(stdout, ctxmeter.Humanize(usage.InputTokens))
	}
	return 0
}

func resolveDataDir(env Env) string {
	if env.PairDataDir != "" {
		return env.PairDataDir
	}
	base := env.XDGDataHome
	if base == "" {
		base = filepath.Join(env.Home, ".local", "share")
	}
	return filepath.Join(base, "pair")
}
