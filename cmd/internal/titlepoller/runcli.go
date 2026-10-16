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
	lease, err := storagegc.AcquireSelectedProcess(context.Background(), getenv, "title-poller")
	if err != nil {
		fmt.Fprintf(stderr, "pair-title: retention: %v\n", err)
		return 1
	}
	defer lease.Close()

	owner, err := ParseTitleOwner(getenv(OwnerPIDEnv), getenv(OwnerStartEnv))
	if err != nil || getenv(LeaseFDEnv) != "3" || procutil.Identity(strconv.Itoa(owner.PID)) != owner.Start {
		fmt.Fprintln(stderr, "pair-title: wrapper lease identity unavailable")
		return 1
	}
	reader := os.NewFile(3, "title-wrapper-lease")
	syscall.CloseOnExec(3)
	info, err := reader.Stat()
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		fmt.Fprintln(stderr, "pair-title: wrapper lease is not a pipe")
		return 1
	}
	signals, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer stopSignals()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	leaseDone := make(chan struct{})
	go func() { defer close(leaseDone); var one [1]byte; _, _ = reader.Read(one[:]) }()
	defer func() { _ = reader.Close(); <-leaseDone }()
	state, effects := AdvanceTitleLifetime(TitleLifetime{}, TitleLeaseReady)
	if effects&TitleStartPoll == 0 {
		return 1
	}
	result := make(chan int, 1)
	go func() {
		rt := runtime()
		if osrt, ok := rt.(OSRuntime); ok {
			rt = osrt.WithContext(ctx)
		}
		result <- RunContext(ctx, opts, rt)
	}()
	var code int
	select {
	case code = <-result:
		state, _ = AdvanceTitleLifetime(state, TitlePollStopped)
		return code
	case <-leaseDone:
		state, effects = AdvanceTitleLifetime(state, TitleOwnerEOF)
	case <-signals.Done():
		state, effects = AdvanceTitleLifetime(state, TitleCancelled)
	}
	if effects&TitleCancelPoll != 0 {
		cancel()
	}
	// Cooperative join. The bounded CLI-process fallback remains gated on the
	// TL's explicit approval; do not silently turn this into detached work.
	code = <-result
	_, _ = AdvanceTitleLifetime(state, TitlePollStopped)
	return code
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
