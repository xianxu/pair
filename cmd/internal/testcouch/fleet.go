package testcouch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

type Options struct {
	Workers       int
	AgentKind     string
	RuntimeBudget time.Duration
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
	Roots       RootsPlan
	Options     Options
	Workspaces  Workspaces
	Binaries    Binaries
	Environment []string
	RunID       string
	t           testing.TB
	mu          sync.Mutex
	closeMu     sync.Mutex
	closing     bool
	starting    bool
	active      int
	idle        chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	model       OwnedRun
	console     *terminalProcess
}

func New(t testing.TB, b Binaries, o Options) *Fleet {
	t.Helper()
	if err := validateOptions(o); err != nil {
		t.Fatal(err)
	}
	if o.AgentKind == "" {
		o.AgentKind = "claude"
	}
	if o.RuntimeBudget == 0 {
		o.RuntimeBudget = 120 * time.Second
	}
	base := os.TempDir()
	if info, err := os.Stat("/Volumes/TestScratch"); err == nil && info.IsDir() {
		base = "/Volumes/TestScratch"
	}
	denied := []string{}
	for _, key := range []string{"HOME", "COUCH_ISOLATED_ROOT", "COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "COUCH_PAIR_DATA_DIR", "PAIR_DATA_DIR", "COUCH_TREE", "ZELLIJ_SOCKET_DIR", "CODEX_HOME", "CLAUDE_CONFIG_DIR"} {
		if value := os.Getenv(key); value != "" {
			denied = append(denied, value)
		}
	}
	p, err := allocateRoots(base, denied)
	if err != nil {
		t.Fatal(err)
	}
	// Worst-case wrapper name must fit before installing or launching anything.
	if len(filepath.Join(p.Root, "m", "wrapper-2147483647-0000000000000000000000000000000000000000.sock")) > 103 {
		_ = os.RemoveAll(p.Root)
		//lint:allow tmp-literal — all broker/wrapper/native terminal sockets need an owner-private short path on macOS.
		p, err = allocateRoots("/tmp", denied)
		if err != nil {
			t.Fatal(err)
		}
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		_ = os.RemoveAll(p.Root)
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	f := &Fleet{Roots: p, Options: o, Binaries: b, RunID: hex.EncodeToString(nonce[:]), t: t, ctx: ctx, cancel: cancel, idle: make(chan struct{})}
	close(f.idle)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := f.Close(cleanup); err != nil {
			t.Errorf("native fleet cleanup failed; root retained at %s: %v", p.Root, err)
		}
	})
	if err = installRuntime(b, p); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(p.Root, agentfixture.OwnershipFile), []byte(f.RunID), 0600); err != nil {
		t.Fatal(err)
	}
	f.Environment, err = runtimeEnvironment(p, p.Mutable["bin"], o.AgentKind, f.RunID, b.Source, filepath.Join(p.Mutable["bin"], "couch"))
	if err != nil {
		t.Fatal(err)
	}
	f.record(RunEvent{Kind: "ready"})
	t.Logf("native fleet isolation: root=%s run=%s agent=%s workers=%d", p.Root, f.RunID, o.AgentKind, o.Workers)
	return f
}

func (f *Fleet) begin(ctx context.Context) (context.Context, func(), error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if f.closing {
		return nil, nil, fmt.Errorf("fleet is closing")
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
	if data, err := json.MarshalIndent(f.model, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(f.Roots.Root, "ownership.json"), data, 0600)
	}
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
	cmd := exec.Command(filepath.Join(f.Roots.Mutable["bin"], "couch"))
	cmd.Dir = f.Workspaces.Primary
	cmd.Env = append([]string(nil), f.Environment...)
	p, err := startTerminal(cmd)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.console = p
	f.mu.Unlock()
	f.record(RunEvent{Kind: "own", Resource: Resource{ID: fmt.Sprint(cmd.Process.Pid), Generation: p.generation, Kind: "couch"}})
	if err = f.WaitAgent(ctx, 0); err != nil {
		return fmt.Errorf("steward startup: %w; screen=%s; output=%s", err, p.Screen(), p.Output())
	}
	return f.WaitActors(ctx, 1)
}

func (f *Fleet) StartWorker(ctx context.Context, n int) error {
	if _, ok := f.Workspaces.Workers[n]; !ok {
		return fmt.Errorf("worker %d was not provisioned", n)
	}
	result, err := f.ActorCLI(ctx, 0, "--reboot", fmt.Sprintf("fixture:%d", n), "--confirm", "--json")
	if err != nil {
		return err
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
	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	if p == nil {
		return fmt.Errorf("no active console")
	}
	return p.Resize(rows, cols)
}

func (f *Fleet) Detach(ctx context.Context) error {
	// Couch's public Alt+d detaches its presentation; no hidden process signal.
	if err := f.Input([]byte{0, 0x1b, 'd'}); err != nil {
		return err
	}
	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *Fleet) RestartSupervisor(ctx context.Context) error {
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

func (f *Fleet) Close(ctx context.Context) error {
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
	f.cancel()
	f.record(RunEvent{Kind: "closing"})
	select {
	case <-idle:
	case <-ctx.Done():
		return ctx.Err()
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
	if err := os.RemoveAll(f.Roots.Root); err != nil {
		return err
	}
	f.mu.Lock()
	f.model = ReduceRun(f.model, RunEvent{Kind: "roots-removed"})
	f.mu.Unlock()
	f.t.Logf("native fleet cleaned: root=%s all owned generations absent; direct child and IO joined", f.Roots.Root)
	return nil
}
