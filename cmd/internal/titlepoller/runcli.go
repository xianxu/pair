package titlepoller

import (
	"io"
	"os/signal"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
)

// RunCLI is the pair-title command body (the launcher spawns bin/pair-title
// directly since #94 M2), shared with any future `pair title` route. It parses
// argv into Options and drives the poller; getenv/stderr are injected so it is
// testable, and it no-ops (exit 0) when required args are missing.
func RunCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	opts, ok := optionsFromCLI(args, getenv)
	if !ok {
		return 0
	}

	// Ignore SIGHUP: bin/pair spawns us with `& disown`, which only removes the
	// job-table entry — the poller still shares a controlling tty with the
	// launching shell, so a terminal teardown would SIGHUP us and freeze the
	// titles. We exit only via the explicit "session gone" branch in Run.
	signal.Ignore(syscall.SIGHUP)

	return Run(opts, NewOSRuntime())
}

// optionsFromCLI is RunCLI's parse without its side effects, so what the poller
// reads from its environment can be observed (pair#183). ok is false when argv
// lacks a tag and an agent: no poller starts.
func optionsFromCLI(args []string, getenv func(string) string) (Options, bool) {
	if len(args) < 2 {
		return Options{}, false
	}
	dataDir := getenv(contextcmd.EnvDataDir)
	if dataDir == "" {
		dataDir = adapt.DataDir()
	}
	opts := Options{
		Tag:             args[0],
		Agent:           args[1],
		DataDir:         dataDir,
		CmuxWorkspaceID: getenv("CMUX_WORKSPACE_ID"),
	}
	if len(args) >= 3 {
		opts.SessionName = args[2]
	}
	return opts, true
}

// SessionEnv is what whoever spawns a title poller must hand it: the ONE
// declaration of its launch contract (pair#183).
//
// The launcher starts a poller from two places, create and attach. Both used to
// satisfy this by exporting variables before the spawn and letting the child
// inherit them, each from its own hand-kept list. Attach's lacked PAIR_SCOPE_KEY,
// and every reattached thread lost its context meter with nothing to say so. It
// lives beside optionsFromCLI because Environ is that parse, inverted.
//
// The environment, not argv as the session watcher's scope key travels
// (sessionwatch.CommandArgs), for two reasons. The poller's readers are
// environment-shaped: contextcmd.EnvFromOS reads the process environment on
// every poll. And an entry appended to the child's environment wins over an
// inherited duplicate, so the contract overrides a stale PAIR_SCOPE_KEY from a
// Pair pane that `pair resume` was run inside.
//
// Its reach is the reads made through contextcmd.EnvFrom and optionsFromCLI --
// every env read in this package and contextcmd today. A direct os.Getenv added
// elsewhere would not be seen by TestSessionEnvSuppliesEveryPairVariableThePollerReads.
// (optionsFromCLI's adapt.DataDir() fallback reads PAIR_DATA_DIR in another
// package, unseen -- harmless: the same variable, reached only when it is empty.)
//
// HOME, XDG_DATA_HOME and CMUX_WORKSPACE_ID are deliberately absent: they belong
// to the operator's terminal, and inheriting them is correct.
type SessionEnv struct {
	dataDir  string
	scopeKey string
}

// NewSessionEnv is POSITIONAL on purpose: a new poller read adds a parameter
// here, and both launcher call sites then fail to compile until they supply it.
// A keyed literal would compile with the new field silently empty -- and Environ
// would hand the child an explicit empty value that overrides a correct inherited
// one, since exec keeps the last duplicate key.
func NewSessionEnv(dataDir, scopeKey string) SessionEnv {
	return SessionEnv{dataDir: dataDir, scopeKey: scopeKey}
}

// Environ renders the contract as KEY=value entries for the child, under the
// names contextcmd reads.
func (e SessionEnv) Environ() []string {
	return []string{
		contextcmd.EnvDataDir + "=" + e.dataDir,
		contextcmd.EnvScopeKey + "=" + e.scopeKey,
	}
}
