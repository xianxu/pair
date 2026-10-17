package titlepoller

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// RunCLI is the wrapper-owned title subprocess. The wrapper passes only a
// pipe reader as fd3 and retains the sole CLOEXEC writer for its lifetime.
// Missing CLI arguments retain the historical no-op; a real title invocation
// requires both selected storage and the exact wrapper/pipe launch contract.
func RunCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	return runCLI(args, getenv, stderr, func() Runtime { return NewOSRuntime() })
}

func runCLI(args []string, getenv func(string) string, stderr io.Writer, runtime func() Runtime) int {
	opts, ok := optionsFromCLI(args, getenv)
	if !ok {
		return 0
	}

	if selectedTag := getenv("PAIR_TAG"); selectedTag != "" && selectedTag != opts.Tag {
		fmt.Fprintln(stderr, "pair-title: retention: PAIR_TAG differs from command tag")
		return 1
	}

	owner, err := ParseTitleOwner(getenv(OwnerPIDEnv), getenv(OwnerStartEnv))
	if err != nil || getenv(LeaseFDEnv) != "3" || procutil.Identity(strconv.Itoa(owner.PID)) != owner.Start {
		fmt.Fprintln(stderr, "pair-title: wrapper lease identity unavailable")
		return 1
	}
	// The wrapper creates the title with Setsid. Refuse before arming any
	// group-wide signal unless this process owns that entire group.
	if syscall.Getpgrp() != os.Getpid() {
		fmt.Fprintln(stderr, "pair-title: wrapper-owned process group unavailable")
		return 1
	}
	reader := os.NewFile(3, "title-wrapper-lease")
	info, err := reader.Stat()
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		fmt.Fprintln(stderr, "pair-title: wrapper lease is not a pipe")
		return 1
	}
	syscall.CloseOnExec(3)
	defer reader.Close()
	state, effects := AdvanceTitleLifetime(TitleLifetime{}, TitleLeaseReady)
	if effects&TitleStartPoll == 0 {
		return 1
	}
	// No Start/register gap exists: commands inherit our already-owned group.
	// One kernel group signal ends the title and every such descendant even
	// during an indivisible proof decode, blocking IPC, or concurrent Start.
	go func() {
		_, _ = io.Copy(io.Discard, reader)
		_, effects := AdvanceTitleLifetime(state, TitleOwnerEOF)
		if effects&TitleExit != 0 {
			exitTitleGroup()
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer signal.Stop(signals)
	go func() {
		<-signals
		_, effects := AdvanceTitleLifetime(state, TitleSignalled)
		if effects&TitleExit != 0 {
			exitTitleGroup()
		}
	}()
	lease, err := storagegc.AcquireSelectedProcess(context.Background(), getenv, "title-poller")
	if err != nil {
		fmt.Fprintf(stderr, "pair-title: retention: %v\n", err)
		return 1
	}
	defer lease.Close()

	code := Run(opts, runtime())
	_, effects = AdvanceTitleLifetime(state, TitlePollStopped)
	if effects&TitleExit != 0 {
		exitTitleGroup()
	}
	return code
}

func exitTitleGroup() {
	_ = syscall.Kill(-os.Getpid(), syscall.SIGKILL)
	// Only a failed group signal can reach this fallback. No cleanup/join or
	// in-process work may delay termination after lease EOF.
	os.Exit(1)
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

// SessionEnv names the exact selected storage for a title. The wrapper obtains
// these fields from its own Pair launch; attach updates the launcher's exports
// but never spawns a second title beside an existing wrapper. TitleOwner owns
// the independent incarnation/lease portion of the contract. Later env entries
// override inherited duplicates. HOME and CMUX_WORKSPACE_ID belong to the
// operator's terminal and remain inherited.
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
