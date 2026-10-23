package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

type Options struct {
	ExplicitHub   bool
	Workers       int
	AgentKind     string
	RuntimeBudget time.Duration
	// RetainStartupServerLog also preserves successful diagnostic controls.
	RetainStartupServerLog bool
}

func validateOptions(o Options) error {
	if o.Workers < 1 || o.Workers > 8 {
		return fmt.Errorf("native fleet requires 1–8 workers")
	}
	if o.AgentKind != "" && o.AgentKind != "claude" && o.AgentKind != "codex" {
		return fmt.Errorf("fixture personality must be claude or codex")
	}
	if o.RuntimeBudget < 0 || o.RuntimeBudget > 24*time.Hour {
		return fmt.Errorf("invalid fleet runtime budget")
	}
	return nil
}

// Fleet is the single lifetime owner. Public commands enter its creation gate;
// Close revokes that gate before reconciling actual native process generations.
type Fleet struct {
	Roots           RootsPlan
	Options         Options
	Workspaces      Workspaces
	Binaries        Binaries
	Environment     []string
	RunID           string
	t               Reporter
	mu              sync.Mutex
	closeMu         sync.Mutex
	closing         bool
	starting        bool
	active          int
	idle            chan struct{}
	ctx             context.Context
	cancel          context.CancelFunc
	model           OwnedRun
	publicationErr  error
	console         *terminalProcess
	serverLogBefore sync.Once
	serverLogAfter  sync.Once
}

func New(t testing.TB, b Binaries, o Options) *Fleet {
	t.Helper()
	f, err := NewOwned(context.Background(), b, o, OwnerConfig{
		ScratchParent: legacyScratchParent(), Reporter: t, AllowScratchFallback: true,
	})
	if f != nil {
		t.Cleanup(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := f.Close(cleanup); err != nil {
				t.Errorf("native fleet cleanup or ownership evidence failed; root=%s: %v", f.Roots.Root, err)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *Fleet) begin(ctx context.Context) (context.Context, func(), error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if f.ctx != nil && f.ctx.Err() != nil {
		return nil, nil, f.ctx.Err()
	}
	if f.publicationErr != nil {
		return nil, nil, f.publicationErr
	}
	if f.model.Error != "" || f.model.Phase == "stuck" {
		return nil, nil, fmt.Errorf("%w: %s", agentfixture.ErrCleanupUncertain, f.model.Error)
	}
	if f.closing {
		return nil, nil, fmt.Errorf("fleet is closing")
	}
	if f.console != nil {
		if err := f.console.failure(); err != nil {
			return nil, nil, fmt.Errorf("supervisor terminal: %w", err)
		}
	}
	if f.active == 0 {
		f.idle = make(chan struct{})
	}
	f.active++
	child, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(f.ctx, cancel)
	return child, func() {
		stop()
		cancel()
		f.mu.Lock()
		f.active--
		if f.active == 0 {
			close(f.idle)
		}
		f.mu.Unlock()
	}, nil
}

func (f *Fleet) record(e RunEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.model = ReduceRun(f.model, e)
	data, err := json.MarshalIndent(f.model, "", "  ")
	if err == nil {
		err = durablefile.WriteAtomic(filepath.Join(f.Roots.Root, "ownership.json"), data, ".ownership-*")
	}
	if err != nil && f.publicationErr == nil {
		f.publicationErr = fmt.Errorf("publish native ownership evidence: %w", err)
	}
}

func (f *Fleet) ownershipError() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.publicationErr
}

func (f *Fleet) Prepare(ctx context.Context) error {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	f.Workspaces, err = PrepareWorkspaces(ctx, f.Binaries, f.Roots, f.Environment, f.Options.Workers)
	return err
}

func (f *Fleet) Start(ctx context.Context) error {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	if f.Workspaces.Primary == "" {
		return fmt.Errorf("prepare genuine workspaces first")
	}
	f.mu.Lock()
	already := f.console != nil || f.starting
	if !already {
		f.starting = true
	}
	f.mu.Unlock()
	if already {
		return fmt.Errorf("supervisor already started")
	}
	defer func() { f.mu.Lock(); f.starting = false; f.mu.Unlock() }()
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := f.consoleCommand()
	p, err := startTerminal(cmd)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.console = p
	f.mu.Unlock()
	f.record(RunEvent{Kind: "own", Resource: Resource{ID: fmt.Sprint(cmd.Process.Pid), Generation: p.generation, Kind: "couch"}})
	if err := f.ownershipError(); err != nil {
		return err
	}
	if err = f.WaitAgent(ctx, 0); err != nil {
		return fmt.Errorf("steward startup: %w; screen=%s; output=%s", err, p.Screen(), p.Output())
	}
	if err := f.WaitActors(ctx, 1); err != nil {
		return fmt.Errorf("steward registration: %w; screen=%s", err, p.Screen())
	}
	return nil
}

func (f *Fleet) StartWorker(ctx context.Context, n int) error {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	if _, ok := f.Workspaces.Workers[n]; !ok {
		return fmt.Errorf("worker %d was not provisioned", n)
	}
	result, err := f.ActorCLI(ctx, 0, "--reboot", fmt.Sprintf("fixture:%d", n), "--confirm", "--json")
	if err != nil {
		return fmt.Errorf("worker %d CLI: %w; stdout=%s stderr=%s", n, err, result.Stdout, result.Stderr)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("worker launch exit%d: %s %s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if err = f.WaitAgent(ctx, n); err != nil {
		return fmt.Errorf("worker %d: %w; %s", n, err, f.Screen())
	}
	return f.WaitActors(ctx, n+1)
}

func (f *Fleet) Screen() string {
	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	if p == nil {
		return ""
	}
	return p.Screen()
}
func (f *Fleet) Input(data []byte) error {
	_, done, err := f.begin(context.Background())
	if err != nil {
		return err
	}
	defer done()
	f.mu.Lock()
	p := f.console
	closing := f.closing
	f.mu.Unlock()
	if p == nil || closing {
		return fmt.Errorf("no active console")
	}
	return p.Input(data)
}
func (f *Fleet) Resize(rows, cols uint16) error {
	_, done, err := f.begin(context.Background())
	if err != nil {
		return err
	}
	defer done()
	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	if p == nil {
		return fmt.Errorf("no active console")
	}
	return p.Resize(rows, cols)
}

func (f *Fleet) Detach(ctx context.Context) error {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	if p == nil {
		return fmt.Errorf("supervisor is not started")
	}
	// Encode physical keys from the actual host's negotiated keyboard mode.
	// ESC d is ordinary input, not Couch's disambiguated Alt+d chord.
	if err := p.press("\x00", 32, 5); err != nil {
		return err
	}
	if err := waitUntil(ctx, func() (bool, error) {
		return strings.Contains(p.Screen(), "Ctrl+Y copy last notice / selected error"), p.failure()
	}); err != nil {
		return fmt.Errorf("open switcher: %w; screen=%s", err, p.Screen())
	}
	if err := p.press("\x1bd", 100, 3); err != nil {
		return err
	}
	return waitUntil(ctx, func() (bool, error) {
		select {
		case <-p.done:
			return true, nil
		default:
		}
		if strings.Contains(p.Screen(), "error: Couch: leave couch:") {
			// The public copy action exposes the complete notice beyond the
			// screen's clipped row. The isolated host records its OSC52 effect.
			before := p.copiedVersion()
			if err := p.press("\x19", 121, 5); err != nil {
				return false, err
			}
			if err := waitUntil(ctx, func() (bool, error) { return p.copiedVersion() > before && p.copiedText() != "", p.failure() }); err != nil {
				return false, fmt.Errorf("detach refused: %s; copy notice: %w", p.Screen(), err)
			}
			return false, fmt.Errorf("detach refused: %s", p.copiedText())
		}
		if err := ctx.Err(); err != nil {
			return false, fmt.Errorf("detach supervisor: %w; screen=%s", err, p.Screen())
		}
		return false, p.failure()
	})
}

func (f *Fleet) RestartSupervisor(ctx context.Context) error {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	if p != nil {
		select {
		case <-p.done:
		default:
			if err := p.Close(ctx); err != nil {
				return err
			}
		}
	}
	f.mu.Lock()
	f.console = nil
	f.mu.Unlock()
	return f.Start(ctx)
}

func (f *Fleet) Close(ctx context.Context) (result error) {
	// Evidence publication failure never revokes in-memory cleanup authority.
	// Finish actual cleanup, then surface the bounded first failure to the caller.
	defer func() { result = errors.Join(result, f.ownershipError()) }()
	f.closeMu.Lock()
	defer f.closeMu.Unlock()
	f.mu.Lock()
	if f.model.Phase == "cleaned" {
		f.mu.Unlock()
		return nil
	}
	f.closing = true
	idle := f.idle
	f.mu.Unlock()
	if f.t.Failed() || f.Options.RetainStartupServerLog {
		f.t.Logf("native server evidence before teardown: %s", f.captureStartupServerLogOnce(&f.serverLogBefore))
	}
	f.cancel()
	f.record(RunEvent{Kind: "closing"})
	select {
	case <-idle:
	case <-ctx.Done():
		return ctx.Err()
	}
	if f.t.Failed() {
		// Native pane and readiness evidence can disappear during console or
		// agent teardown. Freeze a bounded observation after admitted calls join.
		f.t.Logf("native startup evidence before teardown: %s", f.captureStartupEvidence(ctx, nil))
	}
	if err := f.captureOwned(ctx); err != nil {
		return err
	}
	// End the real supervisor's native operation producers before a final
	// census; a canceled CLI alone cannot join an admitted recovery operation.
	f.mu.Lock()
	console := f.console
	f.mu.Unlock()
	if console != nil {
		stop, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := console.Close(stop)
		cancel()
		if err != nil {
			if e := (nativeProcesses{}).Signal(console.cmd.Process.Pid, console.generation, syscall.SIGKILL); e != nil {
				return e
			}
			select {
			case <-console.done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	if err := f.stopAgents(ctx); err != nil {
		f.record(RunEvent{Kind: "unknown", Error: err.Error()})
		return err
	}
	if err := f.captureOwned(ctx); err != nil {
		return err
	}
	if err := f.reconcileProcesses(ctx); err != nil {
		return err
	}

	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	if p != nil {
		select {
		case <-p.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if f.t.Failed() {
		evidence, err := nativeFailureEvidence(f.Roots.Mutable["store"])
		f.t.Logf("native failure evidence before root removal: %s; read_error=%v", evidence, err)
		// The buffered wrapper writers are now joined. Capture their final
		// startup records without attempting another native query after teardown.
		f.t.Logf("native wrapper evidence after join: %s", f.captureStartupLog())
	} else if f.Options.RetainStartupServerLog {
		f.t.Logf("native server evidence after join: %s", f.captureStartupServerLogOnce(&f.serverLogAfter))
	}
	if err := os.RemoveAll(f.Roots.Root); err != nil {
		return err
	}
	f.mu.Lock()
	f.model = ReduceRun(f.model, RunEvent{Kind: "roots-removed"})
	f.mu.Unlock()
	f.t.Logf("native fleet cleaned: root=%s all owned generations absent; direct child and IO joined", f.Roots.Root)
	return nil
}

// consoleCommand preserves bare startup by default; explicit qualification uses
// the same public directory argument as an authenticated configured hub.
func (f *Fleet) consoleCommand() *exec.Cmd {
	args := []string{}
	if f.Options.ExplicitHub {
		args = append(args, f.Workspaces.Primary)
	}
	cmd := exec.Command(filepath.Join(f.Roots.Mutable["bin"], "couch"), args...)
	cmd.Dir = f.Workspaces.Primary
	cmd.Env = append([]string(nil), f.Environment...)
	return cmd
}
