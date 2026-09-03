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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
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

type OSRuntime struct{}

var _ Runtime = OSRuntime{}

func (OSRuntime) Getenv(k string) string { return os.Getenv(k) }

func (r OSRuntime) StoreDir() string {
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
	cwd, err := os.Getwd()
	if err != nil {
		return couchcore.CouchNamespace{}, fmt.Errorf("get startup cwd: %w", err)
	}
	return couchcore.ResolveCouchNamespace(r.StoreDir(), cwd)
}

func (r OSRuntime) AcquireSupervisor(namespace couchcore.CouchNamespace) (io.Closer, error) {
	return couchcore.AcquireSupervisorLease(namespace, couchcore.OSProcOps{})
}

func (r OSRuntime) NewCouchWith(runner couchcore.Runner, namespace couchcore.CouchNamespace) (*couchcore.Couch, error) {
	dataDir := launcher.ResolveDataDir(r.Getenv("HOME"), r.Getenv("XDG_DATA_HOME"))
	c, err := couchcore.New(
		namespace, runner, couchcore.OSPathOps{}, couchcore.ExecGit{},
		couchcore.OSProcOps{}, couchcore.NewStore(namespace.Dir()),
		couchcore.SystemClock{}, couchcore.NewRandomIDGen(), rand.Reader,
		couchcore.NewScopedThreadArtifactCollisionChecker(dataDir),
	)
	if err != nil {
		return nil, err
	}
	c.RootAgent = r.Getenv("PAIR_AGENT")
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
	invocation, err := ParseCLI(args, couchcore.Operations())
	if err != nil {
		fmt.Fprintf(stderr, "couch: %v\n", err)
		return 2
	}
	if invocation.kind == cliHelp {
		usage(stdout)
		return 0
	}
	if invocation.kind == cliLaunch {
		inFile, outFile, ok := terminalFiles(stdin, stdout)
		if !ok {
			fmt.Fprintln(stderr, "couch: interactive launch requires terminal stdin and stdout")
			return 1
		}
		op, _ := Resolve("start")
		return runTypedOperation(op, map[string]string{}, map[string]string{"path": invocation.path}, true, inFile, outFile, stdin, stdout, stderr, rt)
	}

	var op couchcore.Operation
	var argv []string
	switch invocation.kind {
	case cliList:
		op, _ = Resolve("list")
	case cliShow:
		op, _ = Resolve("show")
		argv = []string{invocation.ref}
	case cliInternal:
		op, _ = Resolve(invocation.operation)
		argv = invocation.args
	default:
		fmt.Fprintln(stderr, "couch: invalid invocation")
		return 2
	}
	parsed, err := bindArgs(op, argv)
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
	return runTypedOperation(op, parsed, nil, false, nil, nil, stdin, stdout, stderr, rt)
}

func terminalFiles(stdin io.Reader, stdout io.Writer) (*os.File, *os.File, bool) {
	inFile, inOK := stdin.(*os.File)
	outFile, outOK := stdout.(*os.File)
	return inFile, outFile, inOK && outOK && isTerminal(inFile) && isTerminal(outFile)
}

func runTypedOperation(op couchcore.Operation, parsed, prepareArgs map[string]string, forceConsole bool, inFile, outFile *os.File, stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	return runTypedOperationWithConsole(op, parsed, prepareArgs, forceConsole, inFile, outFile, stdin, stdout, stderr, rt, runConsole)
}

type consoleFinisher func(*couchtty.Console, *couchcore.Couch, couchcore.StartResult, io.Writer) int

func runTypedOperationWithConsole(op couchcore.Operation, parsed, prepareArgs map[string]string, forceConsole bool, inFile, outFile *os.File, stdin io.Reader, stdout, stderr io.Writer, rt Runtime, finishConsole consoleFinisher) int {
	if operationUsesCurrentRepoScope(op.Name) {
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
	// Starting a new root and resuming a parked root are the two entrypoints
	// that bootstrap the singleton owner. Other owner-required CLI calls route
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

	var console *couchtty.Console
	var runner couchcore.Runner
	if forceConsole {
		console, runner = consoleRunnerFor(op.Name, stdin, true, inFile, outFile)
	} else {
		console, runner = consoleRunner(op.Name, stdin, stdout)
	}

	c, err := rt.NewCouchWith(runner, namespace)
	if err != nil {
		fmt.Fprintf(stderr, "couch: %v\n", err)
		return 1
	}

	executors := couchcore.OperationExecutors{DirectStore: couchcore.DirectStoreExecutor(c)}
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
			Name: op.Name, Args: callArgs, Implicit: true, Context: context.Background(),
		})
	}
	if err != nil {
		renderError(stderr, err)
		return 1
	}
	if ownsLive && c.PairLifecycle != nil {
		recoveryContext, stopRecovery := context.WithCancel(context.Background())
		defer stopRecovery()
		go func() {
			// Every failure is already durable on the occupied park transaction.
			// The worker must not paint concurrently with the terminal owner.
			_ = c.RecoverActiveParks(recoveryContext)
		}()
	}
	if console != nil {
		if start, ok := result.(couchcore.StartResult); ok {
			return finishConsole(console, c, start, stdout)
		}
	}
	return render(stdout, op, result)
}

func dispatchInteractiveStart(c *couchcore.Couch, args map[string]string) (couchcore.StartResult, error) {
	return c.StartInteractive(context.Background(), couchcore.StartArgs{Cwd: args["path"], Stack: args["agent"]})
}

func operationUsesCurrentRepoScope(name string) bool {
	switch name {
	case "show", "name", "describe", "park", "resume":
		return true
	default:
		return false
	}
}

// operationOwnsLive is the pure entrypoint policy. Both ways into Couch must
// acquire the same singleton before they can create a child or take a terminal.
func operationOwnsLive(name string) bool {
	return name == "start" || name == "resume"
}

// consoleRunner decides which Runner this invocation gets, and builds the
// Console when it is the pty one.
//
// Returning (nil, ExecRunner{}) is the injected fallback for non-console typed
// operations.
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

func consoleRunner(name string, stdin io.Reader, stdout io.Writer) (*couchtty.Console, couchcore.Runner) {
	inFile, _ := stdin.(*os.File)
	outFile, _ := stdout.(*os.File)

	// Typed non-console operations use ExecRunner. Public launch is separately
	// terminal-gated before it can reach this seam.
	return consoleRunnerFor(name, stdin, isTerminal(inFile) && isTerminal(outFile), inFile, outFile)
}

// consoleRunnerFor is consoleRunner with the terminal question already answered,
// so the WIRING can be pinned without a pty.
//
// Splitting it is not decoration: pinning only WantsConsole left "does
// consoleRunner actually use it" uncovered, and forcing consoleRunner to return
// (nil, ExecRunner) kept the whole suite green (M2 BR-24, twice).
func consoleRunnerFor(name string, stdin io.Reader, hasTerminal bool, inFile, outFile *os.File) (*couchtty.Console, couchcore.Runner) {
	if !WantsConsole(name, hasTerminal) {
		return nil, couchcore.ExecRunner{}
	}

	host := hostty.NewOSHost(inFile, outFile)
	console := couchtty.New(host, stdin)
	return console, &couchcore.PtyRunner{
		Size: console.ChildSize,
		Sink: console.Deliver,
	}
}

// runConsole attaches the spawned child and hands the terminal over. This
// displaces render's StartResult branch, which printed a line and then blocked
// on Handle.Wait for the child's lifetime.
func runConsole(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, stdout io.Writer) int {
	// Wire the switcher's actionable projection HERE, on the path that actually
	// runs a console. Typeahead stays pure over the resulting in-memory rows.
	wireResolver(console, c)

	_, ok := start.Handle.(couchcore.TerminalHandle)
	if !ok {
		// A runner that cannot offer a terminal: fall back rather than crash.
		fmt.Fprintf(stdout, "couch: no terminal available; running without a console\n")
		if start.Handle != nil {
			return start.Handle.Wait()
		}
		return 1
	}
	if err := dispatchInitialAttach(console, start); err != nil {
		renderError(stdout, err)
		return 1
	}
	return console.Run()
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
func wireResolver(console *couchtty.Console, c *couchcore.Couch) {
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
	console.SetForget(c.Forget)

	// The switcher's actions run through the SAME declared table the CLI
	// dispatches: the console names an operation and couchcore performs it, so
	// there is no operator action the advisor cannot also perform (#148's
	// design test) and no way for the switcher to grow a private verb.
	couchLive := couchcore.CouchLiveOwnerExecutor(c)
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		return couchcore.DispatchOperation(couchcore.OperationExecutors{
			DirectStore: couchcore.DirectStoreExecutor(c),
			LiveOwner: func(call couchcore.OperationCall) (any, error) {
				if call.Operation.Effect == couchcore.EffectConsole {
					result, err := console.ExecuteConsoleOperation(call)
					if err != nil && call.Operation.Name == "attach" {
						if start, ok := call.TypedPayload.(couchcore.StartResult); ok {
							return nil, c.AbortStarted(start, err)
						}
					}
					return result, err
				}
				return couchLive(call)
			},
		}, call)
	})
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
	switch v := result.(type) {
	case couchcore.StartResult:
		// Reached only through injected/internal non-console orchestration. Public
		// launch is terminal-gated before actor creation.
		fmt.Fprintf(w, "couch: no console — inheriting stdio, no pty, no reserved row\n")
		fmt.Fprintf(w, "started %s on %s (pid %d)\n", v.Record.ID, v.Record.Args.Worktree, v.Record.PID)
		if v.Handle != nil {
			// Couch launch waits for the child's lifetime: this path has
			// no pty, so the child owns the terminal until it exits (#146).
			return v.Handle.Wait()
		}
		return 0
	case []couchcore.TreeSummary:
		renderTrees(w, v)
	case []couchcore.ThreadSummary:
		if op.Name == "show" {
			renderThreadDetails(w, v)
		} else {
			renderThreads(w, v)
		}
	case couchcore.Worktree:
		fmt.Fprintf(w, "%s\n", v)
	case couchcore.StopResult:
		if v.Signalled {
			fmt.Fprintf(w, "signalled %s on %s (pid %d)\n", v.Record.ID, v.Record.Args.Worktree, v.Record.PID)
		} else {
			fmt.Fprintf(w, "forgot %s on %s -- it was not running\n", v.Record.ID, v.Record.Args.Worktree)
		}
	case string:
		if v == "" {
			fmt.Fprintln(w, "(no description)")
		} else {
			fmt.Fprintln(w, v)
		}
	default:
		fmt.Fprintf(w, "%v\n", v)
	}
	return 0
}

// renderThreads consumes the same one-row-per-composite-thread inventory as
// the panel and advisor. Human names lead named rows; only unnamed rows expose
// the opaque tag as their fallback label.
func renderThreads(w io.Writer, threads []couchcore.ThreadSummary) {
	renderThreadRows(w, threads, false)
}

// renderThreadDetails keeps the immutable composite address available for
// diagnostics and exact follow-up operations. List intentionally stays
// name-first and compact; show is the detail view.
func renderThreadDetails(w io.Writer, threads []couchcore.ThreadSummary) {
	renderThreadRows(w, threads, true)
}

func renderThreadRows(w io.Writer, threads []couchcore.ThreadSummary, includeAddress bool) {
	if len(threads) == 0 {
		fmt.Fprintln(w, "no threads")
		return
	}
	dim, reset := dimCodes(w)
	for _, thread := range threads {
		open, close := dim, reset
		if thread.Live() {
			open, close = "", ""
		}
		fmt.Fprintf(w, "%s%-22s %s%s\n", open, thread.Label(), thread.WorkingPath, close)
		if includeAddress {
			fmt.Fprintf(w, "%s  address: %s/%s%s\n", open, thread.Address.RepoScope, thread.Address.Tag, close)
		}
		if summary := thread.DisplaySummary(); summary != "" {
			fmt.Fprintf(w, "%s  %s%s\n", open, summary, close)
		}
		for _, incarnation := range thread.Incarnations {
			if incarnation.PID > 0 {
				fmt.Fprintf(w, "%s  %-14s pid %d%s\n", open, incarnation.State, incarnation.PID, close)
			} else {
				fmt.Fprintf(w, "%s  %s%s\n", open, incarnation.State, close)
			}
		}
		if len(thread.Incarnations) == 0 {
			// A thread with no incarnation is not necessarily idle: a DETACHED
			// thread's agent is still running behind its zellij session, and
			// only the client is gone. Saying "no agent running" there would
			// contradict the switcher, which offers that row for reattach.
			state := "(no client attached; agent may still be running)"
			if thread.Parked {
				state = "(parked; no agent running)"
			}
			fmt.Fprintf(w, "%s  %s%s\n", open, state, close)
		}
	}
}

// renderTrees prints one block per worktree. A tree with no live actor is
// dimmed rather than omitted: a named tree nobody is running is exactly the
// parked thread this project exists to stop losing, so it must stay visible.
func renderTrees(w io.Writer, trees []couchcore.TreeSummary) {
	if len(trees) == 0 {
		fmt.Fprintln(w, "no trees")
		return
	}
	dim, reset := dimCodes(w)
	for _, t := range trees {
		label := t.Name
		if label == "" {
			label = t.Tree.Repo()
		}
		open, close := dim, reset
		if t.Live() {
			open, close = "", ""
		}
		fmt.Fprintf(w, "%s%-22s %s%s\n", open, label, t.Tree, close)
		if t.Desc != "" {
			fmt.Fprintf(w, "%s  %s%s\n", open, t.Desc, close)
		}
		for _, a := range t.Actors {
			// "unknown" is rendered distinctly: it means the probe could not
			// answer, not that the agent is gone.
			state := a.State.String()
			fmt.Fprintf(w, "%s  %-14s %s  pid %d%s\n", open, a.Record.ID, state, a.Record.PID, close)
		}
		if len(t.Actors) == 0 {
			fmt.Fprintf(w, "%s  (no agent running)%s\n", open, close)
		}
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
	fmt.Fprintf(w, "couch: %v\n", err)
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "couch - supervise agent actors, one per working tree")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "usage: couch [path]")
	fmt.Fprintln(w, "       couch --list")
	fmt.Fprintln(w, "       couch --show <thread>")
	fmt.Fprintln(w, "       couch --help")
}
