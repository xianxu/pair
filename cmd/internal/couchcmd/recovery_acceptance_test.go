package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// Only the original helper uses an OS PID. Replacement terminals remain the
// existing stateful fake, so portable acceptance cannot signal an operator agent.
type recoveryAcceptanceProc struct {
	*couchcore.FakeProcOps
	sourcePID int
}

func (p recoveryAcceptanceProc) Exists(pid int) couchcore.Liveness {
	if pid == p.sourcePID {
		return (couchcore.OSProcOps{}).Exists(pid)
	}
	return p.FakeProcOps.Exists(pid)
}
func (p recoveryAcceptanceProc) Identity(pid int) (string, error) {
	if pid == p.sourcePID {
		return (couchcore.OSProcOps{}).Identity(pid)
	}
	return p.FakeProcOps.Identity(pid)
}

func TestRecoveryDisposableInteractive(t *testing.T) {
	mode := os.Getenv("PAIR_RECOVERY_INTERACTIVE")
	if mode != "warm" && mode != "checkpoint" && mode != "retired-checkpoint" {
		t.Skip("run tests/couch-recovery-smoke.sh for the disposable UI")
	}
	runRecoveryMenuAcceptance(t, mode)
}

func TestRecoveryMenuReachesTerminalAfterActualHelperDeath(t *testing.T) {
	for _, mode := range []string{"warm", "checkpoint", "retired-checkpoint", "archive"} {
		t.Run(mode, func(t *testing.T) { runRecoveryMenuAcceptance(t, mode) })
	}
}

func runRecoveryMenuAcceptance(t *testing.T, mode string) {
	t.Helper()
	base := t.TempDir()
	if os.Getenv("PAIR_RECOVERY_INTERACTIVE") == mode {
		short, err := os.MkdirTemp("/tmp", "p250-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(short) })
		base = short
	}
	repo, sibling := filepath.Join(base, "repo"), filepath.Join(base, "checkpoint-worktree")
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", args...)
		command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main", repo)
	git("-C", repo, "-c", "user.name=Recovery Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-qm", "source")
	git("-C", repo, "worktree", "add", "-q", "-b", "checkpoint", sibling)
	rt := newRT(t, repo)
	rt.runner = couchcore.NewFakeRunner()
	source := seedDetachedThread(t, rt, repo)
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	helper := exec.Command("sleep", "60")
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = helper.Process.Kill(); _, _ = helper.Process.Wait() })
	identity, err := (couchcore.OSProcOps{}).Identity(helper.Process.Pid)
	if err != nil || identity == "" {
		t.Fatalf("actual helper identity: %q %v", identity, err)
	}
	c.Proc = recoveryAcceptanceProc{FakeProcOps: rt.proc, sourcePID: helper.Process.Pid}
	if mode != "retired-checkpoint" {
		source, err = c.Threads.UpdateExistingThread(source.Address, source.Revision, func(next *couchcore.ThreadRecord) error {
			next.Incarnations = []couchcore.ThreadIncarnation{{PID: helper.Process.Pid, Identity: identity, State: couchcore.IncarnationLive, LaunchProfile: next.LatestLaunchProfile}}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = helper.Wait()
	if got := c.Proc.Exists(helper.Process.Pid); got != couchcore.Dead {
		t.Fatalf("helper %d %s remains %s", helper.Process.Pid, identity, got)
	}
	t.Logf("fixture source helper pid=%d identity=%s proved dead; mode=%s", helper.Process.Pid, identity, mode)
	session := "pair-recovery-fixture"
	rt.artifacts.SetPairSession(source.Address, session, false)
	if mode == "warm" {
		rt.artifacts.SetDetachedSession(source.Address, session)
		rt.artifacts.SetPairSession(source.Address, session, true)
	}
	data := filepath.Join(base, "pair-data")
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: source.Address.RepoScope, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	ledger := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	if _, err := ledger.Append(paths.Ledger(), sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: source.Address.RepoScope, Tag: string(source.Address.Tag), Agent: "claude"}); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{SessionName: session, ScopeKey: source.Address.RepoScope, Tag: string(source.Address.Tag), RepoRoot: repo, RepoName: "repo"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SessionBindings(), []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	history := []byte("EXISTING-PROMPT-HISTORY-250\n")
	if err := os.WriteFile(paths.Log(), history, 0600); err != nil {
		t.Fatal(err)
	}
	c.PairLifecycle = &couchcore.PairLifecycleController{Threads: c.Threads, DataDir: data, Lifecycle: couchcore.PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}}, Sessions: rt.artifacts, Proc: c.Proc, Clock: c.Clock}
	c.ContinuationSource = (couchcore.OSContinuationSourceReader{DataDir: data}).Read
	checkpointPath := filepath.Join(sibling, "selected checkpoint.md")
	body := "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nRead RECOVERY-EXACT-250 and retain this thread's prompt history.\n"
	if err := os.WriteFile(checkpointPath, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	readyPath, err := paths.AgentReadyChecked("claude")
	if err != nil {
		t.Fatal(err)
	}
	receipt := couchcore.OSOrientationStatusReader{DataDir: data, Session: rt.artifacts.PairSession, Proc: rt.proc}
	c.FreshRegistration, c.OrientationStatus = receipt.Registered, receipt.Read
	c.ContinuationGeneration = receipt.Generation
	rt.runner.AfterAcknowledge = func(id string) error {
		current, err := c.Threads.GetThread(source.Address)
		if err != nil {
			return err
		}
		inc := current.Incarnations[0]
		rt.proc.Set(inc.PID, inc.Identity)
		rt.artifacts.SetDetachedSession(source.Address, "")
		rt.artifacts.SetPairSession(source.Address, session, true)
		if mode == "warm" {
			return nil
		}
		var profile launcher.TrustedLaunchProfile
		for _, entry := range rt.runner.Child(id).Env {
			if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
				if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
					return err
				}
			}
		}
		if !profile.FreshRequired || profile.ResumeRequired || profile.Orientation == nil {
			return fmt.Errorf("checkpoint target was not a fresh oriented launch: %+v", profile)
		}
		if current.Continuation == nil || current.Continuation.Checkpoint.Body != body {
			return fmt.Errorf("replacement lost exact selected checkpoint")
		}
		if err := os.Remove(checkpointPath); err != nil {
			return err
		}
		target, err := ledger.Append(paths.Ledger(), sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: source.Address.RepoScope, Tag: string(source.Address.Tag), Agent: "claude"})
		if err != nil {
			return err
		}
		ready := readiness.ReadyRecord{Tag: string(source.Address.Tag), Agent: "claude", Session: session, Nonce: profile.Orientation.Attempt, PID: inc.PID, LaunchOrdinal: target.Ordinal, Orientation: &orientation.DeliveryState{Phase: orientation.DeliverySubmitted, BodyWritten: true}}
		raw, err := readiness.Encode(ready)
		if err != nil {
			return err
		}
		return os.WriteFile(readyPath, []byte(raw), 0600)
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 110})
	reader, input := io.Pipe()
	interactive := os.Getenv("PAIR_RECOVERY_INTERACTIVE") == mode
	var uiHost hostty.Host = host
	var uiInput io.Reader = reader
	if interactive {
		host := hostty.NewOSHost(os.Stdin, os.Stdout)
		uiHost, uiInput = host, host
	}
	console := couchtty.New(uiHost, uiInput)
	initial := ptychild.NewFakeChild(nil)
	console.Attach("fixture-panel", "fixture", initial)
	rt.runner.AfterBlockedStart = func(id string) {
		rt.runner.Terminal(id).SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return console.Deliver(ctx, id, batch) })
	}
	wireResolver(console, c)
	lease, err := couchcore.AcquireSupervisorLease(rt.namespace, couchcore.OSProcOps{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lease.Close() })
	if competing, err := couchcore.AcquireSupervisorLease(rt.namespace, couchcore.OSProcOps{}); err == nil {
		_ = competing.Close()
		t.Fatal("second recovery owner acquired the same namespace")
	}
	if interactive {
		runInteractiveRecoveryConsole(t, console, c, rt, initial, source, checkpointPath, mode)
		_ = input.Close()
		_ = reader.Close()
		return
	}
	rows, err := console.ActionableProvider()(context.Background(), nil)
	if err != nil || len(rows) != 1 {
		t.Fatalf("recovery inventory: %+v %v", rows, err)
	}
	keys := []byte{}
	// RESTATED for #256. Warm mode used to reach `recover-thread`, because a
	// thread whose helper had actually died classified `stale-incarnation` --
	// debris needing a recovery gesture. It now classifies `detached`: the
	// helper died, the SESSION did not, and reattaching to a running agent is
	// not recovery, it is resume. The rest of this acceptance is unchanged and
	// still asserts the valuable part -- same thread, no session stopped, no
	// checkpoint written, no native binding consulted.
	operation := "resume"
	if mode != "warm" {
		operation = "recover-checkpoint"
		if mode == "archive" {
			operation = "archive"
		}
		state := couchtty.NewMenuState(rows, couchcore.ThreadAddress{})
		state, _ = couchtty.ReduceMenu(state, couchtty.MenuEvent{Kind: couchtty.MenuEventKey, Key: couchtty.PanelKey{Kind: couchtty.KeyTab}})
		keys = append(keys, '\t')
		for n := 0; state.CurrentFrame().SelectedItem != operation && n < 20; n++ {
			state, _ = couchtty.ReduceMenu(state, couchtty.MenuEvent{Kind: couchtty.MenuEventKey, Key: couchtty.PanelKey{Kind: couchtty.KeyDown}})
			keys = append(keys, []byte("\x1b[B")...)
		}
		if state.CurrentFrame().SelectedItem != operation {
			t.Fatalf("recovery path action absent: %+v", state.CurrentFrame())
		}
		keys = append(keys, '\r')
		if mode != "archive" {
			keys = append(keys, []byte(checkpointPath)...)
		} else {
			keys = append(keys, []byte("\x1b[B")...)
			if err := os.Remove(checkpointPath); err != nil {
				t.Fatal(err)
			}
		}
	}
	keys = append(keys, '\r')
	type completed struct {
		call  couchcore.OperationCall
		value any
		err   error
	}
	results := make(chan completed, 16)
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		value, err := actual(call)
		results <- completed{call, value, err}
		return value, err
	})
	done := make(chan int, 1)
	go func() { done <- console.Run() }()
	t.Cleanup(func() {
		console.Stop()
		initial.Exit(0)
		_ = input.Close()
		_ = reader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("recovery Console did not stop")
		}
	})
	if _, err := input.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "recovery row", func() bool {
		return strings.Contains(host.Written(), source.WorkingPath[:min(len(source.WorkingPath), 50)])
	})
	if _, err := input.Write(keys); err != nil {
		t.Fatal(err)
	}
	var start couchcore.StartResult
	for start.Handle == nil {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("%s: %v", result.call.Name, result.err)
			}
			if result.call.Name == operation {
				if mode == "archive" {
					archived, err := c.Threads.ArchivedThreads()
					if err != nil || len(archived) != 1 || archived[0].Address != source.Address {
						t.Fatalf("archive result: %+v %v", archived, err)
					}
					if _, err := c.Threads.GetThread(source.Address); err == nil {
						t.Fatal("archived stale row stayed active")
					}
					if got, err := os.ReadFile(paths.Log()); err != nil || !bytes.Equal(got, history) {
						t.Fatalf("archive changed prompt history: %q %v", got, err)
					}
					return
				}
				child, ok := result.value.(couchcore.StartedChild)
				if !ok {
					t.Fatalf("recovery returned %T", result.value)
				}
				start, ok = child.Started()
				if !ok {
					t.Fatal("recovery returned no terminal")
				}
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("recovery did not start: %s", host.Written())
		}
	}
	t.Cleanup(func() { rt.runner.SetExited(start.Handle.ID(), 0) })
	terminal := start.Handle.(couchcore.TerminalHandle).Terminal()
	terminal.Feed([]byte("RECOVERED-TERMINAL-250"))
	waitWarmAcceptance(t, "recovery output", func() bool { return strings.Contains(host.Written(), "RECOVERED-TERMINAL-250") })
	if _, err := input.Write([]byte("recovery-input-250")); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "recovery input", func() bool { return bytes.Contains(bytes.Join(terminal.Writes(), nil), []byte("recovery-input-250")) })
	if got, err := os.ReadFile(paths.Log()); err != nil || !bytes.Equal(got, history) {
		t.Fatalf("history changed: %q %v", got, err)
	}
	current, err := c.Threads.GetThread(source.Address)
	if err != nil || current.Address != source.Address {
		t.Fatalf("address changed: %+v %v", current, err)
	}
	if len(rt.artifacts.Quiesces()) != 0 {
		t.Fatalf("recovery stopped sessions: %v", rt.artifacts.Quiesces())
	}
	if mode == "warm" {
		if current.Continuation != nil || rt.artifacts.BindingResolutions() != 0 {
			t.Fatal("warm recovery consulted native binding or created a checkpoint")
		}
	} else {
		state, err := c.ReconcileContinuation(context.Background(), source.Address, current.Continuation.ID, current.Continuation.Attempt)
		if err != nil || state.Phase != checkpoint.Complete {
			t.Fatalf("recovery receipt: %+v %v", state, err)
		}
		if current.VerifiedPark != nil || len(current.ParkHistory) != 0 {
			t.Fatal("source absence fabricated a park")
		}
	}
}

// The interactive front end deliberately uses the same fake terminal seam as
// portable acceptance; the script runs actual Zellij conformance first. This is
// a disposable UI exercise, not a paid-agent conversation or native harness test.
func runInteractiveRecoveryConsole(t *testing.T, console *couchtty.Console, c *couchcore.Couch, rt testRT, initial *ptychild.Child, source couchcore.ThreadRecord, path, mode string) {
	t.Helper()
	defer console.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var echoes sync.WaitGroup
	var targets []string
	var targetMu sync.Mutex
	echo := func(child *ptychild.Child, banner string) {
		echoes.Add(1)
		go func() {
			defer echoes.Done()
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			written := 0
			shown := false
			for {
				select {
				case <-ctx.Done():
					console.Stop()
					return
				case <-ticker.C:
					if !shown {
						child.Feed([]byte(banner))
						shown = true
					}
					chunks := child.Writes()
					for _, chunk := range chunks[written:] {
						if bytes.ContainsAny(chunk, "\x03\x04") {
							console.Stop()
							return
						}
						child.Feed(append([]byte("\r\nfixture echo: "), chunk...))
					}
					written = len(chunks)
				}
			}
		}()
	}
	initial.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error {
		return console.Deliver(ctx, "fixture-panel", batch)
	})
	instruction := "\r\nDISPOSABLE RECOVERY FIXTURE (no paid agent)\r\nReal source helper was killed and reaped. Ctrl+Space opens Couch.\r\n"
	if mode == "warm" {
		instruction += "Select the detached row and press Enter to reattach.\r\n"
	} else {
		instruction += "Select the stale row, Tab, Recover from checkpoint, then enter:\r\n" + path + "\r\n"
	}
	instruction += "After recovery, type text to check echo. Ctrl+D exits. Five-minute limit.\r\n"
	echo(initial, instruction)
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		value, err := actual(call)
		if err == nil && (call.Name == "resume" || call.Name == "recover-thread" || call.Name == "recover-checkpoint") {
			if child, ok := value.(couchcore.StartedChild); ok {
				if start, ok := child.Started(); ok {
					targetMu.Lock()
					targets = append(targets, start.Handle.ID())
					targetMu.Unlock()
					terminal := start.Handle.(couchcore.TerminalHandle).Terminal()
					banner := "\r\nRECOVERED SAME THREAD: " + string(source.Address.Tag) + "\r\n"
					if mode != "warm" {
						banner += "Checkpoint NEXT ACTION: RECOVERY-EXACT-250\r\n"
					}
					banner += "Deterministic fixture echo is ready. Type text; Ctrl+D exits.\r\n"
					echo(terminal, banner)
				}
			}
		}
		return value, err
	})
	console.Run()
	cancel()
	echoes.Wait()
	targetMu.Lock()
	for _, id := range targets {
		rt.runner.SetExited(id, 0)
	}
	targetMu.Unlock()
	initial.Exit(0)
	record, err := c.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.Address != source.Address {
		t.Fatal("interactive recovery changed thread address")
	}
	fmt.Fprintln(os.Stderr, "Disposable recovery UI closed; fixture resources will be removed.")
}
