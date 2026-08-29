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

func EnvFromOS() Env {
	return Env{
		Home:         os.Getenv("HOME"),
		XDGDataHome:  os.Getenv("XDG_DATA_HOME"),
		PairDataDir:  os.Getenv("PAIR_DATA_DIR"),
		PairScopeKey: os.Getenv("PAIR_SCOPE_KEY"),
	}
}

func Run(args []string, env Env, stdout io.Writer) int {
	dataDir := resolveDataDir(env)
	runtime := sessioninventory.NewOSRuntime(env.Home, dataDir)
	return RunWithRuntime(args, env, runtime, stdout)
}

// RunWithRuntime applies the production established-binding query through an
// injected inventory runtime.
func RunWithRuntime(args []string, env Env, runtime sessioninventory.Runtime, stdout io.Writer) int {
	if len(args) < 2 {
		return 0
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
