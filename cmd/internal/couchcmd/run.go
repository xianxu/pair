// Package couchcmd is couch's CLI surface.
//
// It follows the house convention: one exported Run per command package with
// the signature Run(args, stdin, stdout, stderr) int, errors printed to the
// injected stderr rather than returned, and no process globals touched inside
// the package (env arrives through Runtime). Compare termcmd/run.go:46-50.
package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/couchsingleton"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchkeys"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchscratch"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/crashreport"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/editorendpoint"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/keyhelp"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/mockserve"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
	"github.com/xianxu/pair/cmd/internal/submission"
	pairterminal "github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/terminalcapture"
	"github.com/xianxu/pair/cmd/internal/threadactivity"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// Runtime is the seam for everything ambient: env lookup and where the store
// lives. Tests supply their own so they never read the developer's real
// ~/.local/share/pair.
type Runtime interface {
	Getenv(string) string
	StoreDir() string
	CurrentRepoScope() (string, error)
	ResolveNamespace() (couchcore.CouchNamespace, error)
	AcquireSupervisor(couchcore.CouchNamespace) (io.Closer, error)
	// NewCouchWith builds the domain against a caller-supplied Runner. The
	// console needs a PtyRunner it has already wired its own sink and size
	// supplier into, which cannot be constructed inside NewCouch.
	NewCouchWith(couchcore.Runner, couchcore.CouchNamespace) (*couchcore.Couch, error)
}

type OSRuntime struct {
	ownership      *runtimeOwnership
	accountHome    func() (string, error)
	env            func(string) string
	selection      *couchsingleton.Selection
	isolatedRoot   string
	isolatedPaths  isolatedChildRoots
	scratchProbeFn func() couchscratch.Probe
	scratchServers func(context.Context) (bool, error)
	scratchTicks   <-chan time.Time
	scratchNow     func() time.Time
}

var _ Runtime = OSRuntime{}

func (r OSRuntime) Getenv(k string) string {
	if r.selection != nil {
		switch k {
		case "COUCH_STORE_DIR":
			return r.selection.Roots.StoreDir
		case "COUCH_PAIR_DATA_DIR":
			return r.selection.Roots.PairDataDir
		case "COUCH_IDENTITY_DIR":
			return r.selection.Roots.IdentityDir
		case "COUCH_ISOLATED_ROOT":
			return r.isolatedRoot
		case "HOME":
			if r.isolatedRoot != "" {
				return r.isolatedPaths.home
			}
		case "XDG_DATA_HOME":
			if r.isolatedRoot != "" {
				return r.isolatedPaths.data
			}
		case "TMPDIR":
			if r.isolatedRoot != "" {
				return r.isolatedPaths.temporary
			}
		}
	}
	if r.env != nil {
		return r.env(k)
	}
	return os.Getenv(k)
}

func (r OSRuntime) StoreDir() string {
	if r.selection != nil {
		return r.selection.Roots.StoreDir
	}
	if dir := r.Getenv("COUCH_STORE_DIR"); dir != "" {
		return dir
	}
	// Unscoped on purpose: the registry spans every worktree, so a per-tree
	// scope directory would mean spawning in /a and listing from /b read
	// different files.
	return filepath.Join(launcher.ResolveDataDir(r.Getenv("HOME"), r.Getenv("XDG_DATA_HOME")), "couch")
}

func (r OSRuntime) CurrentRepoScope() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get startup cwd: %w", err)
	}
	return resolveCurrentRepoScope(cwd, couchcore.ExecGit{}, couchcore.OSPathOps{})
}

func resolveCurrentRepoScope(cwd string, git couchcore.GitRunner, paths couchcore.PathOps) (string, error) {
	tree, err := couchcore.Resolve(cwd, git, paths)
	if err != nil {
		return "", err
	}
	scope, err := launcher.ResolveRepoScope(string(tree))
	if err != nil {
		return "", err
	}
	return scope.Key, nil
}

func (r OSRuntime) ResolveNamespace() (couchcore.CouchNamespace, error) {
	if r.selection != nil {
		return couchcore.ExistingCouchNamespace(r.selection.Roots.StoreDir)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return couchcore.CouchNamespace{}, fmt.Errorf("get startup cwd: %w", err)
	}
	return couchcore.ResolveCouchNamespace(r.StoreDir(), cwd)
}

func (r OSRuntime) AcquireSupervisor(namespace couchcore.CouchNamespace) (io.Closer, error) {
	if r.selection != nil && r.ownership.held() && namespace.Dir() == r.selection.Roots.StoreDir {
		return emptyLease{}, nil
	}
	return nil, errors.New("Couch runtime does not hold singleton supervisor ownership")
}

func (r OSRuntime) NewCouchWith(runner couchcore.Runner, namespace couchcore.CouchNamespace) (*couchcore.Couch, error) {
	if r.selection == nil {
		return nil, errors.New("Couch runtime must resolve singleton selection before composition")
	}
	if !filepath.IsAbs(r.Getenv("HOME")) {
		return nil, errors.New("Couch runtime requires an absolute application HOME")
	}

	dataDir := runtimePairDataDir(r)
	c, err := couchcore.New(
		namespace, r.runtimeRunner(runner), couchcore.OSPathOps{}, couchcore.ExecGit{},
		couchcore.OSProcOps{}, couchcore.NewStore(namespace.Dir()),
		couchcore.SystemClock{}, couchcore.NewRandomIDGen(), rand.Reader,
		couchcore.NewScopedThreadArtifactCollisionChecker(dataDir),
	)
	if err != nil {
		return nil, err
	}
	c.Identities = couchidentity.IdentityStore{HostDir: r.selection.Roots.IdentityDir, StoreDir: namespace.Dir()}
	c.Workspaces = couchcore.NewWorkspaceProvisioner(couchcore.OSProvisionIO{})
	c.Slots = couchcore.NewOSSlotCatalog(couchcore.OSProvisionIO{})
	r.configureScratch(c)
	c.Fleet = couchcore.SDLCFleetSource{IO: couchcore.OSProvisionIO{}, Timeout: couchcore.FleetInventoryTimeout}
	c.SlotIO = couchcore.OSProvisionIO{}
	c.RootAgent = r.Getenv("PAIR_AGENT")
	c.ContinuationSource = (couchcore.OSContinuationSourceReader{DataDir: dataDir}).Read
	recoveryReader := couchcore.OSRecoveryEvidenceReader{DataDir: dataDir, HomeDir: r.Getenv("HOME")}
	c.RecoveryEvidence = recoveryReader.ReadRequest
	c.RecoveryVerifySelection = recoveryReader.VerifySelection
	c.RecoveryPreserve = recoveryReader.Preserve
	c.RecoveryTerminal = recoveryReader.PrepareTerminal
	c.RecoveryRetirement = recoveryReader.PrepareConfirmedTerminal
	c.SlotRetainedCapture = recoveryReader.ReadRetainedTerminal
	renderer, _ := exec.LookPath("pair")
	c.SwitchContext = couchcore.OSSwitchContextResolver{DataDir: dataDir, HomeDir: r.Getenv("HOME"), Renderer: renderer}
	c.SlotTerminal = func(address couchcore.ThreadAddress, agent string, maxLines int) ([]string, error) {
		return scrollbackcmd.RenderOwnedLines(dataDir, address.RepoScope, string(address.Tag), agent, maxLines)
	}
	c.SlotCapture = func(address couchcore.ThreadAddress, agent, token string, maxLines int) (couchcore.TerminalRecording, error) {
		result := couchcore.TerminalRecording{}
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
		if err != nil {
			return result, err
		}
		if token == "" {
			capture, err := paths.ScrollbackArtifacts(agent)
			if err != nil {
				return result, err
			}
			result.Raw, result.Events = capture.Raw, capture.Events
		} else {
			capture, err := paths.ParkedScrollbackArtifacts(token)
			if err != nil {
				return result, err
			}
			result.Raw, result.Events = capture.Raw, capture.Events
		}
		result.Lines, err = scrollbackcmd.RenderOwnedCaptureLines(dataDir, address.RepoScope, string(address.Tag), agent, token, maxLines)
		return result, err
	}
	c.SlotTail = slotTailReader(namespace.Dir())
	c.Slug = couchcore.OSSlugReader{DataDir: dataDir}.Read
	c.SwitchLaunchCheck = func(agent string) error {
		if !launcher.IsSupportedAgent(agent) {
			return fmt.Errorf("switch-agent: unsupported agent %q", agent)
		}
		for _, executable := range []string{"pair", agent} {
			if _, err := exec.LookPath(executable); err != nil {
				return fmt.Errorf("switch-agent: required executable %s is unavailable: %w", executable, err)
			}
		}
		return nil
	}
	if sessions, ok := c.Artifacts.(couchcore.PairSessionIO); ok {
		status := couchcore.OSReadyReader{DataDir: dataDir, Session: sessions.PairSession, Proc: c.Proc}
		if observer, ok := c.Artifacts.(interface {
			PairSessionContext(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error)
		}); ok {
			status.SessionContext = observer.PairSessionContext
		}

		c.FreshRegistration = status.Registered
		c.ContinuationGeneration = status.Generation
	}

	c.RepoAgentDefault = func(repoRoot, agent string) (couchcore.LaunchProfile, bool, error) {
		scopeDir := launcher.ScopedLaunchDataDir(dataDir, repoRoot)
		raw, err := os.ReadFile(launcher.AgentDefaultPath(scopeDir, agent))
		if errors.Is(err, os.ErrNotExist) {
			return couchcore.LaunchProfile{}, false, nil
		}
		if err != nil {
			return couchcore.LaunchProfile{}, false, err
		}
		value, err := launcher.ParseAgentDefault(agent, string(raw))
		if err != nil {
			return couchcore.LaunchProfile{}, false, err
		}
		return couchcore.LaunchProfile{Agent: value.Agent, Argv: value.Args}, true, nil
	}
	return c, nil
}

// isTerminal reports whether f is a real terminal. Nil-safe: a non-*os.File
// stdio (a pipe, a test buffer) arrives here as nil.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return RunWithRuntime(args, stdin, stdout, stderr, OSRuntime{})
}

// Resolve and Dispatch expose the typed in-process registry to Couch's own
// presentation layers. Public argv reachability is separately constrained by
// Operation.Presentation and ParseCLI.
func Resolve(name string) (couchcore.Operation, bool) {
	op, ok := Dispatch()[name]
	return op, ok
}

func Dispatch() map[string]couchcore.Operation {
	out := map[string]couchcore.Operation{}
	for _, op := range couchcore.Operations() {
		out[op.Name] = op
	}
	return out
}

func RunWithRuntime(args []string, stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	// The broadcast guard (#395) runs a tunnel on Couch's behalf and touches
	// none of Couch's state; it is not an operation and never reaches the CLI.
	if len(args) > 0 && args[0] == broadcast.GuardSubcommand {
		return broadcast.RunGuard(args[1:], stdin, stderr)
	}
	invocation, err := ParseCLI(args, couchcore.Operations())
	if err != nil {
		if len(args) > 0 && args[0] == "--session-sync" {
			for _, arg := range args[1:] {
				if arg == "--json" {
					_ = json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "usage", Error: err.Error()})
					return 2
				}
			}
		}
		fmt.Fprintf(stderr, "couch: %v\n", err)
		return 2
	}
	if invocation.kind == cliMock {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return mockserve.Run(ctx, invocation.args, stdout, stderr)
	}
	if invocation.kind == cliHelp {
		usage(stdout)
		return 0
	}
	if invocation.kind == cliKeysHelp {
		keysUsage(stdout)
		return 0
	}
	if invocation.kind == cliSkill {
		fmt.Fprint(stdout, couchSkill)
		return 0
	}
	if invocation.kind == cliSubmission {
		return runSubmissionCLI(invocation, rt, stdout, stderr, couchmessage.Call)
	}
	if invocation.kind == cliWebQuery {
		// A pure read of a recording directory: no Couch runtime needed,
		// except for link, which asks the running Couch to share it (#477).
		store := func() (string, error) {
			prepared, lease, err := prepareRuntime(rt, false)
			if err != nil {
				return "", err
			}
			lease.Close()
			return prepared.StoreDir(), nil
		}
		return query.Run(invocation.args, replayLinker(store, couchmessage.Call, slotPollClock{now: time.Now, sleep: time.Sleep}, stderr), stdout, stderr)
	}
	if invocation.kind == cliWebReplay {
		// Reads recordings in the store a running Couch records to, and
		// serves them on loopback; it runs nothing of Couch's.
		prepared, lease, err := prepareRuntime(rt, false)
		if err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		defer lease.Close()
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runReplay(ctx, invocation.args, prepared.StoreDir(), couchmessage.Call, slotPollClock{now: time.Now, sleep: time.Sleep}, stdout, stderr)
	}
	if invocation.kind == cliWebFidelity {
		// Reads a recording in the store (or a named directory) and writes
		// its fidelity/ report; it runs nothing of Couch's.
		prepared, lease, err := prepareRuntime(rt, false)
		if err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		defer lease.Close()
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		find := func() (string, error) { return webcast.FindChrome(rt.Getenv, exec.LookPath, fileExists) }
		return runFidelity(ctx, invocation.args, prepared.StoreDir(), find, webcast.CheckFidelity, stdout, stderr)
	}
	if invocation.kind == cliAdopt {
		return runAdoptionCLI(invocation, rt, stdout, stderr)
	}
	if invocation.kind == cliEditorEndpoint {
		prepared, lease, err := prepareRuntime(rt, false)
		if err != nil {
			_ = json.NewEncoder(stdout).Encode(editorendpoint.EndpointReply{Version: 1, Code: "unavailable", Error: "Couch namespace unavailable; start Couch and retry"})
			return 3
		}
		defer lease.Close()
		return runEditorEndpointCLI(invocation, prepared, stdin, stdout, stderr, couchmessage.Call)
	}
	if invocation.kind == cliSessionSync {
		if a := invocation.sessionSync; a.operatorLockCommand() {
			if err := (sessionsync.OperatorLockRequest{ID: a.id, Turn: a.turn}).ValidateTurn(); err != nil {
				return operatorSessionSyncFailure(stdout, err)
			}
		}
		prepared, lease, err := prepareRuntime(rt, false)
		if err != nil {
			if invocation.sessionSync.operatorLockCommand() {
				return sessionSyncUnavailable(stdout, err)
			}
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		defer lease.Close()
		return runSessionSyncCLI(invocation.sessionSync, prepared, stdout, stderr)
	}
	if invocation.kind == cliMessage {
		prepared, lease, err := prepareRuntime(rt, false)
		if err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		defer lease.Close()
		rt = prepared
		return runMessageCLI(invocation, rt, stdout, stderr)
	}
	if invocation.kind == cliLaunch {
		inFile, outFile, ok := terminalFiles(stdin, stdout)
		if !ok {
			fmt.Fprintln(stderr, "couch: interactive launch requires terminal stdin and stdout")
			return 1
		}
		op, _ := Resolve("start")
		return runTypedOperation(op, map[string]string{}, map[string]string{"path": invocation.path}, true, invocation.layout, inFile, outFile, stdin, stdout, stderr, rt, invocation.hubPaths...)
	}

	var op couchcore.Operation
	var argv []string
	switch invocation.kind {
	case cliList:
		op, _ = Resolve("list")
	case cliArchived:
		op, _ = Resolve("archived")
	case cliRecoverPlan:
		op, _ = Resolve("recover-plan")
	case cliShow:
		op, _ = Resolve("show")
		argv = []string{invocation.ref}
	case cliReconcile:
		op, _ = Resolve("reconcile")
		argv = []string{invocation.ref}
	case cliPeek:
		// The running Couch answers live tails in milliseconds (pair#429);
		// the typed peek below is the fallback and the --transcripts form.
		if prepared, lease, err := prepareRuntime(rt, false); err == nil {
			answered := fastPeek(invocation, prepared.StoreDir(), stdout, couchmessage.Call)
			_ = lease.Close()
			if answered {
				return 0
			}
		}
		op, _ = Resolve("peek")
		argv = append([]string{invocation.ref}, invocation.args...)
	case cliInternal:
		op, _ = Resolve(invocation.operation)
		argv = invocation.args
	default:
		fmt.Fprintln(stderr, "couch: invalid invocation")
		return 2
	}
	parsed, err := bindArgs(op, argv)
	if err == nil && op.Name == "request-continuation" {
		err = bindContinuationEnvironment(parsed, rt.Getenv)
	}
	// A spawned child receives the exact composite thread address, so an agent
	// can publish its summary without resolving a mutable path or human label.
	if op.Name == "publish-description" && parsed != nil {
		if parsed["repo-scope"] == "" {
			parsed["repo-scope"] = rt.Getenv("COUCH_THREAD_SCOPE")
		}
		if parsed["tag"] == "" {
			parsed["tag"] = rt.Getenv("COUCH_THREAD_TAG")
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "couch: %v\n", err)
		return 2
	}
	// The read-only forms reject a layout flag in ParseCLI, so they carry none
	// and leave the Couch on its constructed default.
	return runTypedOperation(op, parsed, nil, false, "", nil, nil, stdin, stdout, stderr, rt)
}

func terminalFiles(stdin io.Reader, stdout io.Writer) (*os.File, *os.File, bool) {
	inFile, inOK := stdin.(*os.File)
	outFile, outOK := stdout.(*os.File)
	return inFile, outFile, inOK && outOK && isTerminal(inFile) && isTerminal(outFile)
}

func runTypedOperation(op couchcore.Operation, parsed, prepareArgs map[string]string, forceConsole bool, layout couchcore.Layout, inFile, outFile *os.File, stdin io.Reader, stdout, stderr io.Writer, rt Runtime, hubPaths ...string) int {
	var hubs *hubRuntime
	var setup []func(*couchcore.Couch) error
	if len(hubPaths) > 0 {
		setup = append(setup, func(c *couchcore.Couch) error {
			var err error
			hubs, err = prepareExplicitHubs(context.Background(), c, hubPaths)
			return err
		})
	}
	finish := func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, out io.Writer) int {
		wireTitles(console, c, rt.Getenv)
		return runConsole(console, c, start, out, hubs)
	}
	return runTypedOperationWithConsole(op, parsed, prepareArgs, forceConsole, layout, inFile, outFile, stdin, stdout, stderr, rt, finish, setup...)
}

type consoleFinisher func(*couchtty.Console, *couchcore.Couch, couchcore.StartResult, io.Writer) int

// singleReference is the reference the router resolves a repository scope
// from. A multi-slot peek (pair#425) names several, each resolved by the
// executor, so the router treats it as no workspace reference and supplies
// the current repository for any that are not slots.
func singleReference(op couchcore.Operation, ref string) string {
	if op.Name == "peek" {
		if refs, err := couchcore.ExpandPeekReferences(ref); err == nil && len(refs) > 1 {
			return ""
		}
	}
	return ref
}

func runTypedOperationWithConsole(op couchcore.Operation, parsed, prepareArgs map[string]string, forceConsole bool, layout couchcore.Layout, inFile, outFile *os.File, stdin io.Reader, stdout, stderr io.Writer, rt Runtime, finishConsole consoleFinisher, setup ...func(*couchcore.Couch) error) (code int) {
	_, workspaceRef, referenceErr := couchcore.ParseWorkspaceReference(singleReference(op, parsed["ref"]))
	if referenceErr != nil {
		renderError(stderr, referenceErr)
		return 1
	}
	prepared, lifetime, err := prepareRuntime(rt, operationOwnsLive(op.Name))
	if err != nil {
		renderError(stderr, err)
		return 1
	}
	defer lifetime.Close()
	rt = prepared
	if operationUsesCurrentRepoScope(op.Name) && !workspaceRef {
		scope, err := rt.CurrentRepoScope()
		if err != nil {
			fmt.Fprintf(stderr, "couch %s: resolve current repository scope: %v\n", op.Name, err)
			return 1
		}
		parsed["repo-scope"] = scope
	}
	namespace, err := rt.ResolveNamespace()
	if err != nil {
		fmt.Fprintf(stderr, "couch: %v\n", err)
		return 1
	}
	// Starting, resuming, recovery and explicit archive acquire the singleton
	// owner. Other owner-required CLI calls route
	// to an already-running owner, which is deliberately unavailable until
	// #147.
	ownsLive := operationOwnsLive(op.Name)
	if ownsLive {
		lease, err := rt.AcquireSupervisor(namespace)
		if err != nil {
			renderError(stderr, err)
			return 1
		}
		defer lease.Close()
	}

	operationContext := context.Background()
	if op.Name == "provision-workspace" {
		var stop context.CancelFunc
		operationContext, stop = signal.NotifyContext(operationContext, os.Interrupt, syscall.SIGTERM)
		defer stop()
	}

	var console *couchtty.Console
	var runner couchcore.Runner
	if forceConsole {
		console, runner, err = consoleRunnerFor(op.Name, stdin, true, inFile, outFile, tracesForRuntime(rt))
	} else {
		console, runner, err = consoleRunner(op.Name, stdin, stdout, tracesForRuntime(rt))
	}

	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	if console != nil {
		defer func() {
			if captureErr := console.CloseCapture(); captureErr != nil {
				fmt.Fprintln(stderr, "couch: capture incomplete:", captureErr)
				code = 1
			}
		}()
	}

	c, err := rt.NewCouchWith(runner, namespace)
	if err != nil {
		fmt.Fprintf(stderr, "couch: %v\n", err)
		return 1
	}
	for _, prepare := range setup {
		if err := prepare(c); err != nil {
			renderError(stderr, err)
			return 1
		}
	}
	if console != nil && ownsLive {
		// No defer: cmd/couch's main ends capture after a normal return.
		installCrashReport(console, namespace.Dir())
	}
	if operationUsesCurrentRepoScope(op.Name) && workspaceRef {
		path, recognized, err := c.WorkspaceReferencePath(context.Background(), parsed["ref"])
		if err != nil {
			renderError(stderr, err)
			return 1
		}
		if recognized {
			scope, err := launcher.ResolveRepoScope(path)
			if err != nil {
				renderError(stderr, err)
				return 1
			}
			parsed["repo-scope"] = scope.Key
		} else {
			scope, err := rt.CurrentRepoScope()
			if err != nil {
				renderError(stderr, err)
				return 1
			}
			parsed["repo-scope"] = scope
		}
	}
	c.WorkspaceProgress = stderr
	if ownsLive && c.ScratchStartup != nil {
		notify := func(text string) { fmt.Fprintln(stderr, "couch: "+text) }
		if console != nil {
			notify = func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) }
		}
		c.ScratchStartup(operationContext, notify)
	}
	if console != nil {
		// Provisioning outlives initial attach and also runs from menu actions.
		// Only the console renderer may write to the terminal it owns. The
		// command runner still captures diagnostics for operation failures.
		c.WorkspaceProgress = io.Discard
	}
	// The one place the CLI's layout choice reaches the domain. Set here rather
	// than through NewCouchWith so the Runtime interface -- and every fake
	// implementing it -- stays unchanged. An empty layout is a non-launch form,
	// which leaves New's process default alone.
	if layout != "" {
		c.Layout = layout
	}

	executors := couchcore.OperationExecutors{DirectStore: couchcore.DirectStoreExecutor(c), Attempt: c.OperationAttemptForCall,
		DiagnosticWarning: func(text string) { fmt.Fprintln(stderr, text) },
	}
	if ownsLive {
		executors.LiveOwner = couchcore.CouchLiveOwnerExecutor(c)
	}
	var result any
	if forceConsole && op.Name == "start" && prepareArgs != nil {
		result, err = dispatchInteractiveStart(c, prepareArgs)
	} else {
		callArgs := parsed
		if prepareArgs != nil {
			preparedValue, prepareErr := couchcore.DispatchOperation(executors, couchcore.OperationCall{
				Name:    "prepare-start",
				Args:    prepareArgs,
				Context: context.Background(),
			})
			if prepareErr != nil {
				renderError(stderr, prepareErr)
				return 1
			}
			prepared, ok := preparedValue.(couchcore.PreparedStart)
			if !ok {
				fmt.Fprintf(stderr, "couch: prepare-start returned %T\n", preparedValue)
				return 1
			}
			callArgs = prepared.Resolution.CommitArgs()
		}
		result, err = couchcore.DispatchOperation(executors, couchcore.OperationCall{
			Name: op.Name, Args: callArgs, Implicit: true, Context: operationContext,
		})
	}
	if err != nil {
		// A slot operation that failed still holds the observation it acted
		// on: show it beside the advice (pair#387).
		var blocked *couchcore.SlotReconcileError
		if errors.As(err, &blocked) && len(blocked.Result.Observation.Resources) > 0 {
			renderSlotReport(stdout, couchcore.SlotReport{Address: blocked.Address, Observation: blocked.Result.Observation, Plan: blocked.Result.Plan})
		}
		renderError(stderr, err)
		return 1
	}
	if console != nil {
		if forceConsole && op.Name == "start" {
			if start, ok := result.(couchcore.StartResult); ok {
				return finishConsole(console, c, start, stdout)
			}
		}
		if child, ok := result.(couchcore.StartedChild); ok {
			if start, hasChild := child.Started(); hasChild {
				return finishConsole(console, c, start, stdout)
			}
		}
	}
	if parsed["json"] == "true" {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			renderError(stderr, err)
			return 1
		}
		return 0
	}
	if threads, ok := result.([]couchcore.ThreadSummary); ok && op.Name == "list" {
		renderThreadRows(stdout, threads, false, listRefusals(rt, couchmessage.Call))
		return 0
	}
	return render(stdout, op, result)
}

func dispatchInteractiveStart(c *couchcore.Couch, args map[string]string) (couchcore.StartResult, error) {
	return c.StartInteractive(context.Background(), couchcore.StartArgs{Cwd: args["path"], Stack: args["agent"], Action: couchcore.StartAction(args["action"])})
}

func operationUsesCurrentRepoScope(name string) bool {
	switch name {
	case "show", "peek", "park", "resume", "reboot", "reap", "recover":
		return true
	default:
		return false
	}
}

// operationOwnsLive is the pure entrypoint policy. Both ways into Couch must
// acquire the same singleton before they can create a child or take a terminal.
func operationOwnsLive(name string) bool {
	return name == "start" || name == "resume" || name == "reboot" || name == "reap" || name == "recover"
}

// WantsConsole is the console DECISION, separated from building one.
//
// Pure, and that is the point: the previous pins for this needed a real pty and
// so skipped in the sandbox this issue documents as its environment -- meaning
// the mutation "disable the console entirely" stayed green, which is the
// gated-only-pin lesson for the third time. The decision is the thing worth
// pinning; constructing a Console is plumbing.
//
// hasTerminal must be true for BOTH directions. couch measures the input fd and
// draws on the output fd, so a redirected stdout with a tty stdin would
// otherwise build a console that paints into a file.
func WantsConsole(name string, hasTerminal bool) bool {
	return operationOwnsLive(name) && hasTerminal
}

// consoleRunner decides which Runner this invocation gets, and builds the
// Console when it is the pty one.
//
// Returning (nil, ExecRunner) is the injected fallback for non-console typed
// operations.
func consoleRunner(name string, stdin io.Reader, stdout io.Writer, settings ...consoleTraceConfig) (*couchtty.Console, couchcore.Runner, error) {
	inFile, _ := stdin.(*os.File)
	outFile, _ := stdout.(*os.File)

	// Typed non-console operations use ExecRunner. Public launch is separately
	// terminal-gated before it can reach this seam.
	return consoleRunnerFor(name, stdin, isTerminal(inFile) && isTerminal(outFile), inFile, outFile, settings...)
}

// consoleRunnerFor is consoleRunner with the terminal question already answered,
// so the WIRING can be pinned without a pty.
//
// Splitting it is not decoration: pinning only WantsConsole left "does
// consoleRunner actually use it" uncovered, and forcing consoleRunner to return
// (nil, ExecRunner) kept the whole suite green (M2 BR-24, twice).
func consoleRunnerFor(name string, stdin io.Reader, hasTerminal bool, inFile, outFile *os.File, settings ...consoleTraceConfig) (*couchtty.Console, couchcore.Runner, error) {
	getenv := os.Getenv
	if len(settings) > 0 {
		getenv = settings[0].getenv
	}
	if !WantsConsole(name, hasTerminal) {
		return nil, couchcore.ExecRunner{}, nil
	}

	path, captureConfig, err := captureSettings(getenv)
	if err != nil {
		return nil, nil, err
	}
	// Parsed before anything is opened, so a bad value leaks nothing.
	store := ""
	if len(settings) > 0 {
		store = settings[0].store
		captureConfig.CloseTimeout = settings[0].captureCloseTimeout
	}
	broadcastConfig, err := broadcastSettings(getenv, store, guardArgv())
	if err != nil {
		return nil, nil, err
	}
	// One public endpoint for every cast (#454): a named tunnel can be open
	// only once, so the screen broadcast and the web cast share it.
	castHost := broadcast.NewHost(broadcast.HostConfig{Tunnel: broadcastConfig.Tunnel, ProbeTimeout: broadcastConfig.ProbeTimeout, Resolve: broadcastConfig.Resolve})
	broadcastConfig.Host = castHost
	webcastConfig, err := webcastSettings(getenv, store, castHost, exec.LookPath, fileExists)
	if err != nil {
		return nil, nil, err
	}
	// A tunnel whose Couch and guard both died is cleared now, not at the
	// next broadcast.
	if c, ok := broadcastConfig.Tunnel.(broadcast.Cloudflared); ok && c.Records != "" {
		broadcast.ReapOrphans(c.Records, c.RunDir)
	}
	var recorder *terminalcapture.Recorder
	if path != "" {
		recorder, err = terminalcapture.Open(path, captureConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("COUCH_CAPTURE_DIR: %w", err)
		}
	}
	osHost := hostty.NewOSHost(inFile, outFile)
	var host hostty.Host = osHost
	if recorder != nil {
		host = &captureHost{OSHost: osHost, recorder: recorder}
	}
	if inFile != nil {
		stdin = osHost
	}
	console := couchtty.New(host, stdin)
	console.SetCapture(recorder)
	console.SetBroadcast(broadcastConfig)
	console.SetWebcast(webcastConfig)
	if store != "" {
		// Recordings shared for replay on the same Host (#477), until Couch exits.
		console.SetReplayShares(webcast.NewReplayShares(castHost, recordingsDir(store)))
		submissions, err := submission.NewStore(filepath.Join(store, "webcast", "submissions"))
		if err != nil {
			return nil, nil, err
		}
		console.SetSubmissionStore(submissions)
	}
	options := diagnosticlog.Options{Proof: diagnosticlog.DefaultProof}
	if len(settings) > 0 {
		config := settings[0]
		getenv = config.getenv
		// A configured standalone trace belongs to Couch's resolved Pair
		// root, even when no Pair launcher exported PAIR_DATA_DIR.
		if getenv("COUCH_INPUT_TRACE") != "" || getenv("COUCH_TRACE") != "" || getenv("COUCH_MOUSE_TRACE") != "" {
			if e := os.MkdirAll(config.root, 0700); e != nil {
				options.Registry = func(context.Context, diagnosticlog.RegistryEntry) error { return e }
			} else {
				options = diagnosticlog.EnvironmentOptions(func(key string) string {
					if key == "PAIR_DATA_DIR" {
						return config.root
					}
					return getenv(key)
				})
			}
		}
	}
	_ = console.SetInputTrace(getenv("COUCH_INPUT_TRACE"), options)
	_ = console.SetEventTrace(getenv("COUCH_TRACE"), processStartedAt, options)
	_ = console.SetMouseTrace(getenv("COUCH_MOUSE_TRACE"), options)
	wireIdleFading(console, getenv)

	profile := sync.OnceValues(func() ([]string, error) {
		root := workbenchshortcut.DataDirFromEnv()
		if len(settings) > 0 && settings[0].root != "" {
			root = settings[0].root
		}
		return runtimebundle.TerminalEnvironment(root)
	})
	return console, &couchcore.PtyRunner{
		Environment: profile,
		Size:        console.ChildSize,
		Sink:        console.Deliver,
		Observer:    captureObserver(recorder),
	}, nil
}

type consoleTraceConfig struct {
	getenv func(string) string
	root   string
	store  string
	// captureCloseTimeout overrides the capture's drain bound; zero keeps
	// terminalcapture's default. Only tests set it.
	captureCloseTimeout time.Duration
}

func tracesForRuntime(rt Runtime) consoleTraceConfig {
	return consoleTraceConfig{getenv: rt.Getenv, root: runtimePairDataDir(rt), store: rt.StoreDir()}
}

// installCrashReport keeps this console's fatal panics on disk (#397): its
// stderr is the terminal it redraws over. Only the lease holder installs it --
// the lease is what proves every older file in the crash dir is a dead
// incarnation's. A failure is a notice; crash capture never stops couch.
//
// It returns nothing to close on purpose: ending capture from a defer here
// would run while a panic unwinds and delete the file before the runtime
// writes it (#397 BR-1). crashreport.Finish runs in main after a normal return.
func installCrashReport(console *couchtty.Console, store string) {
	_, reports, err := crashreport.Install(crashreport.Dir(store), time.Now(), os.Getpid(), crashreport.ProcessAlive)
	if summary := crashreport.Summary(reports); summary != "" {
		console.Notify(couchtty.Notice{Kind: "crash", Control: true, Body: summary})
	}
	if err != nil {
		console.Notify(couchtty.Notice{Kind: "crash-capture", Control: true, Body: "crash capture unavailable: " + err.Error()})
	}
}

// processStartedAt is when this couch process began: package initialisation,
// before Run. It stamps COUCH_TRACE's startup event (pair#206), so the trace
// measures the whole startup, not only the part after the console exists.
var processStartedAt = time.Now()

// runConsole opens the switcher when startup has no child. Explicit launch
// results attach their child before handing the terminal over.
func runConsole(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, stdout io.Writer, configuredHubs ...*hubRuntime) int {
	// Wire the switcher's actionable projection HERE, on the path that actually
	// runs a console. Typeahead stays pure over the resulting in-memory rows.
	wireResolver(console, c, configuredHubs...)

	_, ok := start.Handle.(couchcore.TerminalHandle)
	if start.Handle != nil && !ok {
		// A runner that cannot offer a terminal: fall back rather than crash.
		fmt.Fprintf(stdout, "couch: no terminal available; running without a console\n")
		if start.Handle != nil {
			return start.Handle.Wait()
		}
		return 1
	}
	if err := beginConsole(console, start); err != nil {
		renderError(stdout, err)
		return 1
	}
	messages, messageErr := startMessageService(console, c, configuredHubs...)
	if messageErr != nil {
		renderError(stdout, messageErr)
		return 1
	}
	if messages != nil {
		defer messages.Close()
	}
	if len(configuredHubs) > 0 && configuredHubs[0] != nil {
		startExplicitHubs(console, configuredHubs[0], c)
	}
	code := console.Run()
	recordConsoleExit(console.TerminalFailure())
	return code
}

// recordConsoleExit leaves the reason a console ended on a failed write to
// the terminal in this run's crash file, so the next start reports it once
// (pair#409). The exit itself is deliberate; only its silence was wrong.
func recordConsoleExit(failure error) {
	if reason, ok := pairterminal.ExitReason(failure); ok {
		_ = crashreport.RecordExit(reason)
	}
}

// beginConsole attaches the explicitly started hub child, then arms the existing
// bounded warm-only pass before Run admits its first inventory. No background
// attempt may launch or recover a conversation whose terminal is absent.
func beginConsole(console *couchtty.Console, start couchcore.StartResult) error {
	if start.Handle == nil {
		return nil
	}
	if err := dispatchInitialAttach(console, start); err != nil {
		return err
	}
	console.ArmReattachPass(start.Record.Thread)
	if warning := start.Warning(); warning != "" {
		// Startup adoption bypasses finishOperation; queue a standing notice
		// for the first paint so slow startup cannot hide borrowed ownership.
		console.Notify(couchtty.Notice{Kind: "status", Control: true, Body: warning})
	}
	return nil
}

func dispatchInitialAttach(console *couchtty.Console, start couchcore.StartResult) error {
	dispatch := console.Ops()
	if dispatch == nil {
		return fmt.Errorf("console operation dispatcher is unavailable")
	}
	_, err := dispatch(couchcore.OperationCall{
		Name: "attach",
		Args: map[string]string{
			"repo-scope": start.Record.Thread.RepoScope,
			"tag":        string(start.Record.Thread.Tag),
		},
		Implicit: true, TypedPayload: start,
	})
	return err
}

// wireResolver supplies the proof-bearing actionable projection. Reference
// matching for keystrokes is intentionally in-memory inside the pure menu.
func wireResolver(console *couchtty.Console, c *couchcore.Couch, configuredHubs ...*hubRuntime) {
	console.SetSessionOwnerPass(c.WithSessionOwnerPass)
	artifacts, _ := c.Artifacts.(shortcutFocusArtifacts)
	console.SetShortcutFocusProbe(shortcutFocusProbe(artifacts, queryShortcutClients))
	console.SetContinuationProvider(c.ContinuationRequests)
	console.SetActionableProvider(func(ctx context.Context, observations []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		rows, err := c.ActionableThreadInventoryContext(ctx, observations)
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return rows, nil
		}
	})
	if c.Git != nil {
		console.SetSlotGitProbe(func(ctx context.Context, dir string) (couchcore.SlotGitStatus, error) {
			return couchcore.ProbeSlotGit(ctx, c.Git, dir)
		})
	}
	console.SetForget(c.Forget)

	// The switcher's actions run through the SAME declared table the CLI
	// dispatches: the console names an operation and couchcore performs it, so
	// there is no operator action the advisor cannot also perform (#148's
	// design test) and no way for the switcher to grow a private verb.
	couchLive := couchcore.CouchLiveOwnerExecutor(c)
	console.SetOperationAttemptFactory(c.OperationAttemptForCall)
	dispatch := func(call couchcore.OperationCall) (any, error) {
		return couchcore.DispatchOperation(couchcore.OperationExecutors{
			Attempt:           c.OperationAttemptForCall,
			DiagnosticWarning: func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) },
			DirectStore:       couchcore.DirectStoreExecutor(c),
			LiveOwner: func(call couchcore.OperationCall) (any, error) {
				if couchmessage.IsDriveOp(call.Name) {
					return executeAgentDrive(call)
				}
				if call.Name == "move-slot" {
					return executeHubMove(call)
				}
				if call.Operation.Effect == couchcore.EffectConsole {
					result, err := console.ExecuteConsoleOperation(call)
					if err != nil && call.Operation.Name == "attach" {
						if start, ok := call.TypedPayload.(couchcore.StartResult); ok {
							// Attach runs on the console goroutine, and the abort may
							// wait on the thread's gate: run it off that goroutine so
							// the console keeps rendering and draining results
							// (pair#205). The attach error is the operation's answer;
							// a cleanup failure beyond it follows as a notice.
							attachErr := err
							console.GoTracked(func() {
								abortErr := c.AbortStarted(call.Context, start, attachErr)
								if abortErr != nil && abortErr.Error() != attachErr.Error() {
									console.Notify(couchtty.Notice{Kind: "status", Body: fmt.Sprintf("cleanup after a failed attach of %s: %v", start.Record.Thread.Tag, abortErr)})
								}
							})
							return nil, err
						}
					}
					return result, err
				}
				return couchLive(call)
			},
		}, call)
	}
	var hubs *hubRuntime
	if len(configuredHubs) > 0 {
		hubs = configuredHubs[0]
	}
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) { return dispatchHubOperation(hubs, call, dispatch) })
}

// bindArgs maps positional argv onto the operation's declared ArgSpecs, plus
// --flag=value form for the optional ones.
func bindArgs(op couchcore.Operation, argv []string) (map[string]string, error) {
	out := map[string]string{}
	known := make(map[string]couchcore.ArgSpec, len(op.Args))
	for _, spec := range op.Args {
		if !spec.Implicit {
			known[spec.Name] = spec
		}
	}
	var positional []string
	for _, a := range argv {
		if strings.HasPrefix(a, "--") {
			name, value, found := strings.Cut(strings.TrimPrefix(a, "--"), "=")
			spec, exists := known[name]
			if !exists {
				return nil, fmt.Errorf("unknown flag --%s", name)
			}
			if spec.ValueRequired && (!found || value == "") {
				return nil, fmt.Errorf("--%s requires a non-empty value in --%s=<value> form", name, name)
			}
			if !found {
				value = "true"
			}
			out[name] = value
			continue
		}
		positional = append(positional, a)
	}
	i := 0
	for _, spec := range op.Args {
		if spec.Implicit {
			continue
		}
		if _, already := out[spec.Name]; already {
			continue
		}
		// FlagOnly arguments never bind positionally -- they gate something, so
		// a stray positional word must not be able to set them.
		if spec.FlagOnly {
			continue
		}
		if i < len(positional) {
			out[spec.Name] = positional[i]
			i++
			continue
		}
		if spec.Required {
			return nil, fmt.Errorf("missing required argument %q", spec.Name)
		}
	}
	if i < len(positional) {
		return nil, fmt.Errorf("unexpected argument %q", positional[i])
	}
	return out, nil
}

func render(w io.Writer, op couchcore.Operation, result any) int {
	if child, ok := result.(couchcore.StartedChild); ok {
		if started, hasChild := child.Started(); hasChild {
			fmt.Fprintln(w, "couch: no console — inheriting stdio, no pty, no reserved row")
			fmt.Fprintf(w, "started %s on %s (pid %d)\n", started.Record.ID, started.Record.Args.Worktree, started.Record.PID)
			if started.Handle != nil {
				return started.Handle.Wait()
			}
			return 0
		}
	}
	// An agent runs publish-description from its shell after claiming
	// dispatched work (#419), so it reads one line, not a record dump.
	if record, ok := result.(couchcore.ThreadRecord); ok && op.Name == "publish-description" {
		if record.PublishedSummary == "" {
			fmt.Fprintf(w, "cleared published summary for %s\n", record.Address.Tag)
		} else {
			fmt.Fprintf(w, "published summary for %s: %s\n", record.Address.Tag, record.PublishedSummary)
		}
		return 0
	}
	switch v := result.(type) {
	case couchcore.PeekResult:
		renderPeek(w, v)
	case couchcore.PeekSnapshot:
		for i, slot := range v.Slots {
			if i > 0 {
				fmt.Fprintln(w)
			}
			renderPeek(w, slot)
		}
	case couchcore.ProvisionResult:
		if op.Name == "reconcile" {
			renderReconcile(w, v)
			break
		}
		if err := json.NewEncoder(w).Encode(v); err != nil {
			return 1
		}
	case couchcore.RecoverPlan:
		// One JSON document and nothing else: the report is read by an agent.
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(v); err != nil {
			return 1
		}
	case couchcore.ContinuationResult:
		return render(w, op, v.Status)
	case couchcore.ContinuationStatus:
		fmt.Fprintf(w, "continuation %s/%s: %s\n", v.Address.RepoScope, v.Address.Tag, v.Phase)
		if v.Failure != "" {
			fmt.Fprintln(w, v.Failure)
		}
	case []couchcore.ThreadSummary:
		renderThreads(w, v)
	case couchcore.ShowResult:
		for _, status := range v.Scratch {
			fmt.Fprintln(w, status+" [current launch selection]")
		}
		if len(v.Threads) > 0 {
			renderThreadDetails(w, v.Threads)
		}
		if v.Slot != nil {
			renderSlotReport(w, *v.Slot)
		}
		renderLaunchFailures(w, v.LaunchFailures, v.LaunchFailureError)
	case couchcore.Worktree:
		fmt.Fprintf(w, "%s\n", v)
	case couchcore.ArchiveResult:
		fmt.Fprintf(w, "archived %s\n", v.Record.Address.Tag)
		if warning := v.Warning(); warning != "" {
			fmt.Fprintf(w, "%s\n", warning)
		}
	case couchcore.RebootResult:
		// A reboot that started an agent rendered above, as a start. What
		// reaches here retired a record and started nothing; say why.
		if v.Archived != (couchcore.ThreadAddress{}) {
			fmt.Fprintf(w, "archived %s\n", v.Archived.Tag)
		}
		if v.Reason != "" {
			fmt.Fprintf(w, "no fresh agent started: %s\n", v.Reason)
		}
		if warning := v.Warning(); warning != "" {
			fmt.Fprintf(w, "%s\n", warning)
		}
	case couchcore.ReapResult:
		fmt.Fprintf(w, "reaped orphaned server PID %d (%s) for %s; resume it next\n", v.Server.PID, v.Server.Session, v.Address.Tag)
	case couchcore.StopResult:
		if v.Signalled {
			fmt.Fprintf(w, "signalled %s on %s (pid %d)\n", v.Record.ID, v.Record.Args.Worktree, v.Record.PID)
		} else {
			fmt.Fprintf(w, "forgot %s on %s -- it was not running\n", v.Record.ID, v.Record.Args.Worktree)
		}
	default:
		fmt.Fprintf(w, "%v\n", v)
	}
	return 0
}

// renderPeek prints a peek: a header naming the thread and where its lines
// came from, the slot's recent terminal lines (live ones carry ‹dim›, ‹rev›
// and ‹cursor› markup), the cursor, then where its logs live and what could
// not be read. A multi-slot peek prints one such section per slot.
func renderPeek(w io.Writer, r couchcore.PeekResult) {
	agent := r.Agent
	if agent == "" {
		agent = "unknown"
	}
	source := r.Source
	if source == "" {
		source = "none"
	}
	fmt.Fprintf(w, "peek %s  agent %s  tag %s  source %s\n", r.Ref, agent, r.Tag, source)
	if r.Complete {
		fmt.Fprintln(w, "--- complete recording projection ---")
	} else {
		fmt.Fprintln(w, "--- recent terminal ---")
	}
	for _, line := range r.Lines {
		fmt.Fprintln(w, line)
	}
	fmt.Fprintln(w, "---")
	if r.Cursor != "" {
		fmt.Fprintf(w, "cursor: %s\n", r.Cursor)
	}
	if r.Truncated > 0 {
		fmt.Fprintf(w, "truncated: %d older lines\n", r.Truncated)
	}
	if r.Session != nil {
		fmt.Fprintf(w, "session: %s (nonce %s)\n", r.Session.Name, r.Session.StartNonce)
	}
	if r.NativeSession != "" {
		fmt.Fprintf(w, "native session: %s\n", r.NativeSession)
	}
	if r.Raw != "" {
		fmt.Fprintf(w, "tty raw: %s\n", r.Raw)
	}
	if r.Events != "" {
		fmt.Fprintf(w, "tty events: %s\n", r.Events)
	}
	if r.SentPrompts != "" {
		fmt.Fprintf(w, "sent prompts: %s\n", r.SentPrompts)
	}
	for _, path := range r.Transcripts {
		fmt.Fprintf(w, "transcript: %s\n", path)
	}
	for _, reason := range r.Unavailable {
		fmt.Fprintf(w, "unavailable: %s\n", reason)
	}
}

// renderReconcile prints what couch --reconcile did: the slot's final
// resources and plan, then any degraded warnings.
func renderReconcile(w io.Writer, r couchcore.ProvisionResult) {
	if r.Report != nil {
		renderSlotReport(w, *r.Report)
	}
	fmt.Fprintf(w, "%s %s at %s\n", r.Address, r.Disposition, r.Path)
	if r.Warning != "" {
		fmt.Fprintln(w, "warning: "+strings.ReplaceAll(r.Warning, "\n", "\nwarning: "))
	}
}

// renderSlotReport prints a slot's resources in converge order and the plan
// reconcile would run (pair#387): the plan half of plan/apply.
func renderSlotReport(w io.Writer, r couchcore.SlotReport) {
	if r.Address == "" {
		fmt.Fprintln(w, "slot: "+r.PlanError)
		return
	}
	fmt.Fprintf(w, "slot %s\n", r.Address)
	for _, res := range r.Observation.Resources {
		state := res.State.String()
		if res.State == couchcore.StatePending {
			state = "pending"
		}
		if res.Sub != "" {
			state += " (" + res.Sub + ")"
		}
		if res.Reason != "" {
			state += ": " + res.Reason
		}
		fmt.Fprintf(w, "  %-14s %s\n", res.ID, state)
	}
	fmt.Fprintln(w, "plan: "+couchcore.SlotPlanSummary(r.Plan, r.PlanError))
	if f := r.LastFailure; f != nil {
		at := string(f.Resource)
		if f.Step != "" {
			at = string(f.Step)
		}
		fmt.Fprintf(w, "last failure %s at %s: %s\n", f.At.Format(time.RFC3339), at, f.Cause)
	}
	if r.LastFailureError != "" {
		fmt.Fprintln(w, "last failure: unreadable: "+r.LastFailureError)
	}
}

// renderThreads consumes the same one-row-per-composite-thread inventory as
// the panel and advisor. Human names lead named rows; only unnamed rows expose
// the opaque tag as their fallback label.
func renderThreads(w io.Writer, threads []couchcore.ThreadSummary) {
	renderThreadRows(w, threads, false, nil)
}

// renderThreadDetails keeps the immutable composite address available for
// diagnostics and exact follow-up operations. List intentionally stays
// name-first and compact; show is the detail view.
func renderThreadDetails(w io.Writer, threads []couchcore.ThreadSummary) {
	renderThreadRows(w, threads, true, nil)
}

func renderThreadRows(w io.Writer, threads []couchcore.ThreadSummary, includeAddress bool, refusals map[string]couchmessage.RefusalInfo) {
	if len(threads) == 0 {
		fmt.Fprintln(w, "no threads")
		return
	}
	dim, reset := dimCodes(w)
	labels := couchcore.LabelsFor(threads,
		func(t couchcore.ThreadSummary) couchcore.ThreadAddress { return t.Address },
		func(t couchcore.ThreadSummary) string { return t.Label() })
	for _, thread := range threads {
		// Dim by the CLASSIFIED state. Reading liveness from the incarnations
		// here while the label came from the classifier printed a stale row
		// undimmed above the words "stale — helper ownership unresolved".
		open, close := dim, reset
		if thread.State == couchcore.ThreadLive {
			open, close = "", ""
		}
		fmt.Fprintf(w, "%s%-22s %s%s\n", open, labels[thread.Address], thread.WorkingPath, close)
		if includeAddress {
			fmt.Fprintf(w, "%s  address: %s/%s%s\n", open, thread.Address.RepoScope, thread.Address.Tag, close)
		}
		if summary := thread.DisplaySummary(); summary != "" {
			fmt.Fprintf(w, "%s  %s%s\n", open, summary, close)
		}
		// The RECORDED lifecycle, labelled as such. Beside the classified state
		// below it, the pair is the diagnostic: "recorded live / stale" is a
		// couch that exited without detaching, and reading the two lines
		// together is how the operator sees that at a glance.
		for _, incarnation := range thread.Incarnations {
			if incarnation.PID > 0 {
				fmt.Fprintf(w, "%s  recorded: %-8s pid %d%s\n", open, incarnation.State, incarnation.PID, close)
			} else {
				fmt.Fprintf(w, "%s  recorded: %s%s\n", open, incarnation.State, close)
			}
		}
		// The classified state, from the same rule the switcher uses. This used
		// to be a local two-case guess over a `Parked` bool, which is how one
		// store produced two different stories about the same thread.
		fmt.Fprintf(w, "%s  %s%s\n", open, threadStateText(thread), close)
		// Keyed by the slot's own reference, not the row label, which gains a
		// tag tail when labels collide.
		if thread.Target.Kind == couchcore.ThreadTargetSlot {
			slot := couchcore.WorkspaceReference{Repo: thread.Target.Slot.Repo, Number: thread.Target.Slot.Number}.String()
			if reason, refused := refusals[slot]; refused {
				fmt.Fprintf(w, "  %s\n", receiveStateText(slot, reason))
			}
		}
	}
}

// threadStateText shares the switcher's continuity choice, with room for
// diagnostic detail. Archived records retain their separate restore wording.
func threadStateText(thread couchcore.ThreadSummary) string {
	if thread.State == couchcore.ThreadArchived {
		return "archived (restore by moving it back and re-adding the address)"
	}
	switch couchcore.ContinuityOutcome(couchcore.ActionableThreadSummary{State: thread.State, Reason: thread.Reason}) {
	case couchcore.Continue:
		if thread.State == couchcore.ThreadDetached {
			return "Continue (no client attached; the agent is still running)"
		}
		return "Continue"
	case couchcore.ResumeDisk:
		return "Resume from disk (no agent running)"
	case couchcore.ContinuityStarting:
		return "Starting (couch claimed a start and has not finished it)"
	default:
		label := "Start fresh"
		if !couchcore.RebootAdmitted(thread.State, thread.Reason, false) {
			label = "Relaunch"
		}
		if thread.Reason != "" {
			return label + ": " + thread.Reason.Label()
		}
		return label
	}
}

// dimCodes returns ANSI dim/reset only when the destination is a real
// terminal, so piped or captured output stays plain text.
func dimCodes(w io.Writer) (string, string) {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return "", ""
	}
	return "\x1b[2m", "\x1b[0m"
}

// renderError prints one error. Its capacity-refusal shape went with admission
// (pair#170 M4): couch-lite has no capacity to exceed, so there is no refusal to
// give a next action to.
func renderError(w io.Writer, err error) {
	fmt.Fprintf(w, "couch: %s\n", couchcore.OperatorError(err))
}

func usage(w io.Writer) { usageWith(w, couchkeys.Bindings()) }

// usageWith renders the public help over a given chord table, so a test can
// prove `couch --help` renders whatever the table declares (#282).
func usageWith(w io.Writer, bindings []couchkeys.Binding) {
	fmt.Fprintln(w, "couch - supervise agent actors, one per working tree")
	fmt.Fprintln(w)
	verbLine(w, launchVerb, "usage: couch [path ...] [--layout2|--layout3]")
	verbLine(w, "mock", "       couch mock serve DIR [--port N] [--real URL] | mock use DIR mock-N | mock check DIR")
	verbLine(w, "--list", "       couch --list")
	verbLine(w, "--show", "       couch --show <thread>")
	verbLine(w, "--reconcile", "       couch --reconcile repo:N")
	verbLine(w, "--peek", "       couch --peek repo:N[:M...][,repo:N...] [--lines N | --all] [--previous] [--json] [--transcripts]")
	fmt.Fprintln(w, "             Read-only: a slot's recent terminal; --transcripts adds where its transcripts\n"+
		"             live, and costs seconds.\n"+
		"             --previous: exact previous tty only [seconds], incompatible with --transcripts; --all: complete recording [heavy].")
	verbLine(w, "--archived", "       couch --archived")
	verbLine(w, "--recover-plan-from-sdlc", "       couch --recover-plan-from-sdlc")
	fmt.Fprintln(w, "             Recovery report: one JSON row per slot joining sdlc's claims and")
	fmt.Fprintln(w, "             slot verdicts with Couch's threads, with a suggested next step.")
	verbLine(w, "--switch-agent", "       couch --switch-agent repo:N --agent NAME --confirm [--json]")
	verbLine(w, "--keys", "       couch --keys repo:N STEP... [--json]")
	fmt.Fprintf(w, "             Hub only. STEP: %s\n", agentdrive.StepSyntax())
	fmt.Fprintln(w, "             Steps, rules and the slash-command panel round trip: couch --keys --help.")
	verbLine(w, "--interrupt", "       couch --interrupt repo:N --confirm [--json]")
	fmt.Fprintln(w, "             Hub only: one Ctrl+C into the slot's working turn; refused with no turn. Never retried.")
	verbLine(w, "--set-model", "       couch --set-model repo:N --model NAME --effort LEVEL [--scope session|default] --confirm [--json]")
	fmt.Fprintln(w, "             Hub only, another idle clean slot: use NAME's last successful path config,")
	fmt.Fprintln(w, "             then repo default. Missing config: ask the operator to set it in the switch form.")
	fmt.Fprintln(w, "             No self, startup parameter edits or context delivery. Read the receipt before retrying.")
	verbLine(w, "--operation-status", "       couch --operation-status ID [--json]")
	verbLine(w, "--resume", "       couch --resume repo:N [--candidate KEY] [--json]")
	verbLine(w, "--resume-candidates", "       couch --resume-candidates repo:N [--json]")
	verbLine(w, "--recovery-confirmation", "       couch --recovery-confirmation TOKEN [--json]")
	verbLine(w, "--reboot", "       couch --reboot repo:N --confirm [--json]")
	fmt.Fprintln(w, "             Archive the slot's conversation and replace it with a fresh agent on the current")
	fmt.Fprintln(w, "             binary. Any attributed processes that must stop require the exact printed")
	fmt.Fprintln(w, "             recovery-confirmation command. Canceling that confirmation changes nothing.")
	verbLine(w, "--relaunch", "       couch --relaunch repo:N --confirm [--same-binary] [--force-unknown] [--json]")
	verbLine(w, "--reload-context", "       couch --reload-context repo:N --confirm [--force-unknown] [--json]")
	verbLine(w, "--prepare", "       couch --prepare repo:N --confirm [--force-unknown] [--json]")
	fmt.Fprintln(w, "             Bring an idle, clean slot on its resting branch up for new work:")
	fmt.Fprintln(w, "             weave refresh in it, then a fresh agent conversation.")
	verbLine(w, "--reap", "       couch --reap repo:N --confirm [--json]")
	verbLine(w, "--recover-park", "       couch --recover-park repo:N --confirm [--json]")
	fmt.Fprintln(w, "             Resolve an interrupted park, retaining its conversation. Never starts an agent;")
	fmt.Fprintln(w, "             follow the receipt’s next action. Read --operation-status ID if the reply is lost.")
	verbLine(w, "--recover", "       couch --recover repo:N [--json]")
	fmt.Fprintln(w, "             From a live Couch slot only: run the report's step on one slot")
	fmt.Fprintln(w, "             through the running Couch, in the background. Verify by reading")
	fmt.Fprintln(w, "             the report again; an uncertain outcome means read it before resending.")
	verbLine(w, "--actors", "       couch --actors [--json]")
	verbLine(w, "--resources", "       couch --resources [--json]")
	verbLine(w, "--reap-orphans", "       couch --reap-orphans [--confirm] [--json]")
	fmt.Fprintln(w, "             The unattributed embedded nvim --resources named. Without --confirm it")
	fmt.Fprintln(w, "             re-classifies and lists them, ending nothing. With --confirm it ends each")
	fmt.Fprintln(w, "             one and its descendants, every signal fenced on the identity the sweep")
	fmt.Fprintln(w, "             recorded: a pid that moved on or was recycled is skipped, never signalled.")
	verbLine(w, "--broadcast-list", "       couch --broadcast-list [--json]")
	fmt.Fprintln(w, "             The running broadcast and its viewer count, and the recordings shared for")
	fmt.Fprintln(w, "             replay (never a link).")
	verbLine(w, "--session-sync", "       couch --session-sync register FILE --owner SLOT [--json]")
	fmt.Fprintln(w, "             lookup FILE | list | forget FILE | rename OLD NEW [--json]")
	fmt.Fprintln(w, "             read|diff|status|write ... FILE: registered markdown and legacy session files.")
	verbLine(w, "--session-sync", "       couch --session-sync read --frontmatter FILE [--json]")
	verbLine(w, "--session-sync", "       couch --session-sync write --frontmatter BODYFILE FILE --if-hash H")
	fmt.Fprintln(w, "             Read or replace only the leading --- block; body bytes and unread body edits stay intact.")
	fmt.Fprintln(w, "             CAS writes refuse under the operator lock; after 30 minutes, ping its holder.")
	verbLine(w, "--session-sync", "       couch --session-sync apply EDITS.json FILE --if-hash H [--fold ID] [--json]")
	fmt.Fprintln(w, "             Apply {old, occurrence, new, explain} records to the whole file, all or none")
	fmt.Fprintln(w, "             (edits-invalid lists each bad record). Every agent write keeps a round.")
	verbLine(w, "--session-sync", "       couch --session-sync round FILE --json")
	fmt.Fprintln(w, "             The last agent round's spans, and whether a submit has cleared it.")
	fmt.Fprintln(w, "             Registration lasts until forget; rename moves metadata only. See couch --skill.")
	verbLine(w, "--editor-endpoint", "       couch --editor-endpoint register --json")
	verbLine(w, "--editor-endpoint", "       couch --editor-endpoint unregister --json")
	fmt.Fprintln(w, "             Bounded editor attachment JSON on stdin; unregister compares a batch of exact generations.")
	verbLine(w, "--editor-endpoint", "       couch --editor-endpoint lookup --namespace PATH --document-id ID --json")
	fmt.Fprintln(w, "             Private live-editor endpoint metadata; no document acquisition or remote editing.")
	verbLine(w, "--adopt-store", "       couch --adopt-store <absolute-path> [--pair-data <path>] [--identity-dir <path>]")
	fmt.Fprintln(w, "             [--legacy-store <path>]... [--exclude-store <path>]... [--apply <digest>]")
	fmt.Fprintln(w, "             Preview legacy stores as JSON; apply the current digest to select in place.")
	fmt.Fprintln(w, "             Stop and upgrade old Couch supervisors first. Exclusions preserve data.")
	fmt.Fprintln(w, "             One supervisor per OS account on local storage; a second launch refuses.")
	fmt.Fprintln(w, "             COUCH_ISOLATED_ROOT explicitly separates diagnostic/test runtimes.")
	verbLine(w, "--move-slot", "       couch --move-slot repo:N[,repo:N...] --to-hub HUB [--persist]")
	fmt.Fprintln(w, "             Hub TLs move their own slots or main’s; --persist edits started hubs’ .hub files.")
	verbLine(w, "--send-to", "       couch --send-to repo[:N]|hub [--agent NAME] --message TEXT|--message-file PATH")
	fmt.Fprintln(w, "             --message-file sends the file verbatim (less one trailing newline): write a")
	fmt.Fprintln(w, "             message containing code or shell syntax with an editor, never quote it.")
	verbLine(w, "--message-status", "       couch --message-status ID [--json]")
	fmt.Fprintln(w, "             Full ID or unique prefix of a receipt you sent or received; ambiguous prefixes list full IDs.")
	verbLine(w, "--notify", "       couch --notify [--title T] MESSAGE")
	fmt.Fprintln(w, "             Only a live hub notifies (default title: Couch); other slots send to their hub.")
	fmt.Fprintln(w, "             Slots' OSC 9/777/99 notifications are swallowed, including the hub's; BEL is unchanged.")
	fmt.Fprintln(w, "             Title: 256 bytes; message: 4096 bytes. Uncertain write: do not retry automatically.")
	verbLine(w, query.Flag, "       couch "+query.Flag+" DIR dom|events|notes|link ...")
	fmt.Fprintln(w, "             Read-only: ask a broadcast-web recording for the page at t, what happened")
	fmt.Fprintln(w, "             in a window, and every note with its element and the actions before it;")
	fmt.Fprintln(w, "             link prints its replay link at a moment, shared by the running Couch.")
	fmt.Fprintln(w, "             `couch "+query.Flag+"` alone prints its usage.")
	verbLine(w, submissionFlag, "       couch "+submissionFlag+" prepare|status|begin|ack ID")
	fmt.Fprintln(w, "             Submit the cast's open notes from this draft; prints a JSON status for retries.")
	verbLine(w, replayFlag, "       couch "+replayFlag+" [ID|DIR] [--at MS] [--local]")
	fmt.Fprintln(w, "             Replay a broadcast-web recording in the viewer: prints its link at MS (#t=MS),")
	fmt.Fprintln(w, "             shared through the running Couch's broadcast tunnel for other devices. With")
	fmt.Fprintln(w, "             no Couch running, no ID (the list), or --local: served on this machine only")
	fmt.Fprintln(w, "             until Ctrl-C.")
	verbLine(w, fidelityFlag, "       couch "+fidelityFlag+" [ID|DIR]")
	fmt.Fprintln(w, "             Check a broadcast-web recording (default: the newest) against the tab's")
	fmt.Fprintln(w, "             screenshots in a headless Chrome; writes DIR/fidelity/report.md. Exit 3 when")
	fmt.Fprintln(w, "             a frame is over the tolerance. Recorded casts are checked when they end.")
	verbLine(w, "--skill", "       couch --skill")
	verbLine(w, "--help", "       couch --help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, costGuidance)
	fmt.Fprintln(w)
	// Deliberately free of the words the public-surface test forbids: they are
	// internal operation names, and the remedy for a refusal belongs in the
	// refusal itself, where it can name the actual threads.
	fmt.Fprintln(w, "  --layout3  give every thread pair's right-hand terminal (default: --layout3).")
	fmt.Fprintln(w, "  --layout2  use the two-pane workbench without the right-hand terminal.")
	fmt.Fprintln(w, "             One layout per couch: it refuses to run alongside a thread")
	fmt.Fprintln(w, "             already holding a session in the other layout.")
	fmt.Fprintln(w)
	// Couch's chords, laid out by the function Pair's Alt+h page uses (#282).
	fmt.Fprint(w, keyhelp.Render(couchkeys.HelpSections(bindings)))
	fmt.Fprintln(w, "\nThe agent also reserves these Pair shortcuts:")
	for _, binding := range workbenchshortcut.GlobalBindings() {
		if binding.Scope == workbenchshortcut.ScopeGlobal && binding.AgentReserved {
			fmt.Fprintf(w, "  %s  %s\n", workbenchshortcut.ChordName(binding.Chord), binding.Help)
		}
	}
	fmt.Fprintln(w, "Other workbench keys reach the focused agent. Click another pane to leave it.")
}

// wireIdleFading hands the console what idle fading needs from the environment
// (pair#247): the colour modes, and a probe that reads each live thread's last
// activity through the one shared definition. Each thread's sources live under
// its own repository's scope directory, not the global data root.
func wireIdleFading(console *couchtty.Console, getenv func(string) string) {
	colorTerm := strings.ToLower(getenv("COLORTERM"))
	console.SetColorModes(colorTerm == "truecolor" || colorTerm == "24bit", getenv("NO_COLOR") != "")
	home := getenv("HOME")
	dataDir := getenv("COUCH_PAIR_DATA_DIR")
	if dataDir == "" {
		dataDir = launcher.ResolveDataDir(home, getenv("XDG_DATA_HOME"))
	}
	console.SetActivityProbe(couchtty.ThreadActivityProbe(dataDir, threadactivity.NewOSRuntime(home)))
}
