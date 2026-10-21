package vmcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type hostFakeChild struct{}

func (hostFakeChild) PID() int         { return 123 }
func (hostFakeChild) Identity() string { return "fixture-start" }

type hostFakeDriver struct {
	observe                         func(string)
	exists, running, uncertainStop  bool
	input, results                  string
	clones, launches, deletes       int
	commandExit                     int
	cloneErr                        error
	readyErr, executeErr, exportErr error
	stopHook                        func()
}

func (f *hostFakeDriver) Inspect(context.Context, string) (VMStatus, error) {
	if f.observe != nil {
		f.observe("inspect")
	}
	state := VMStopped
	if f.running {
		state = VMRunning
	}
	return VMStatus{Exists: f.exists, State: state}, nil
}
func (f *hostFakeDriver) Clone(_ context.Context, _, _ string) error {
	if f.observe != nil {
		f.observe("clone")
	}
	if f.exists {
		return fmt.Errorf("clone collision")
	}
	f.exists = true
	f.clones++
	return f.cloneErr
}
func (f *hostFakeDriver) Launch(_ context.Context, _, input string, _ io.Writer) (VMChild, error) {
	if !f.exists || f.running {
		return nil, fmt.Errorf("bad launch")
	}
	f.running = true
	f.input = input
	f.results = filepath.Join(filepath.Dir(input), "guest-results")
	if err := os.Mkdir(f.results, 0700); err != nil {
		return nil, err
	}
	f.launches++
	return hostFakeChild{}, nil
}
func (f *hostFakeDriver) AwaitGuest(context.Context, string) error {
	if f.observe != nil {
		f.observe("ready")
	}
	if f.readyErr != nil {
		return f.readyErr
	}
	if !f.running {
		return fmt.Errorf("not running")
	}
	return nil
}
func (f *hostFakeDriver) Execute(_ context.Context, _ string, _ []string, out, errOut io.Writer) (int, error) {
	if f.observe != nil {
		f.observe("execute")
	}
	fmt.Fprint(out, "command-out")
	fmt.Fprint(errOut, "command-error")
	if f.executeErr != nil {
		return -1, f.executeErr
	}
	if !f.running {
		return -1, fmt.Errorf("not running")
	}
	err := os.WriteFile(filepath.Join(f.results, "status"), []byte(fmt.Sprintf("exit_code=%d\ncollection_error=0\n", f.commandExit)), 0600)
	if err != nil {
		return f.commandExit, err
	}
	raw, _ := json.Marshal(resultManifest{Version: 1, Files: []resultFile{{Path: "status", Size: int64(len(fmt.Sprintf("exit_code=%d\ncollection_error=0\n", f.commandExit)))}}})
	err = os.WriteFile(filepath.Join(f.results, "manifest.json"), raw, 0600)
	return f.commandExit, err
}
func (f *hostFakeDriver) ExportResults(ctx context.Context, _, guestRoot, destination string, remaining int64) error {
	if f.observe != nil {
		f.observe("export")
	}
	if f.exportErr != nil {
		return f.exportErr
	}
	return exportGuestResults(ctx, guestRoot, destination, remaining, func(_ context.Context, path string, _ int64) ([]byte, bool, error) {
		relative := strings.TrimPrefix(path, guestRoot+"/")
		raw, err := os.ReadFile(filepath.Join(f.results, relative))
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return raw, err == nil, err
	})
}
func (f *hostFakeDriver) Stop(context.Context, string, VMChild) error {
	if f.observe != nil {
		f.observe("stop")
	}
	if f.uncertainStop {
		return fmt.Errorf("unconfirmed stop")
	}
	f.running = false
	if f.stopHook != nil {
		f.stopHook()
	}
	return nil
}
func (f *hostFakeDriver) Delete(context.Context, string) error {
	if f.observe != nil {
		f.observe("delete")
	}
	if f.running {
		return fmt.Errorf("deleting live VM")
	}
	f.exists = false
	f.deletes++
	return nil
}
func (f *hostFakeDriver) Recover(_ context.Context, c LaunchClaim) (RecoveryEvidence, error) {
	state := VMStopped
	if f.running {
		state = VMRunning
	}
	return RecoveryEvidence{PID: c.PID, Identity: c.StartIdentity, Process: ProcessGone, Group: ProcessGone, VM: state}, nil
}
func hostFixture(t *testing.T) (*hostRuntime, *hostFakeDriver, RunRequest) {
	t.Helper()
	source, _ := snapshotFixture(t)
	root := filepath.Join(t.TempDir(), "machine")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	raw := []byte(validManifestJSON())
	if err := os.WriteFile(filepath.Join(root, "image.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	driver := &hostFakeDriver{}
	rt := newHostRuntime(root, driver, io.Discard, io.Discard)
	return rt, driver, RunRequest{Checkout: source, Lane: AdhocLane, Timeout: time.Minute, Args: []string{"/usr/bin/true"}}
}
func TestHostRuntimeKeepsInputsUntilConfirmedStop(t *testing.T) {
	rt, driver, req := hostFixture(t)
	driver.uncertainStop = true
	result := Run(context.Background(), req, rt)
	if result.ExitCode() == 0 || !driver.running {
		t.Fatal("uncertain stop accepted", result)
	}
	if _, err := os.Stat(filepath.Join(driver.input, "checkout", "tracked")); err != nil {
		t.Fatal("input removed while VM uncertain", err)
	}
	position, err := AcquirePosition(context.Background(), filepath.Join(rt.root, "positions"), req.Lane, nil)
	if err != nil {
		t.Fatal(err)
	}
	claim, present, err := position.ReadClaim()
	position.Close()
	if err != nil || !present || claim.StartIdentity != "fixture-start" {
		t.Fatal("lost durable startup claim", claim, err)
	}
	again := newHostRuntime(rt.root, driver, io.Discard, io.Discard)
	next := Run(context.Background(), req, again)
	if next.ExitCode() == 0 || driver.launches != 1 {
		t.Fatal("live residue admitted another launch", next, driver.launches)
	}
	driver.running = false
	driver.uncertainStop = false
	retry := newHostRuntime(rt.root, driver, io.Discard, io.Discard)
	if result := Run(context.Background(), req, retry); result.ExitCode() != 0 {
		t.Fatal("confirmed recovery failed", result)
	}
}
func TestHostRuntimeCollectsAndCleansSuccessfulAndFailedCommands(t *testing.T) {
	for _, exit := range []int{0, 7} {
		t.Run(fmt.Sprint(exit), func(t *testing.T) {
			rt, driver, req := hostFixture(t)
			driver.commandExit = exit
			result := Run(context.Background(), req, rt)
			if result.ExitCode() != exit || driver.running || driver.exists {
				t.Fatal("bad terminal ownership", result, driver)
			}
			if _, err := os.Stat(filepath.Join(rt.results, "status")); err != nil {
				t.Fatal("artifacts discarded", err)
			}
			if _, err := os.Stat(driver.input); !os.IsNotExist(err) {
				t.Fatal("scratch retained after confirmed cleanup", err)
			}
			p, err := AcquirePosition(context.Background(), filepath.Join(rt.root, "positions"), req.Lane, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			if _, present, err := p.ReadClaim(); err != nil || present {
				t.Fatal("completed claim retained", err)
			}
		})
	}
}
func TestHostRuntimeRefusesArtifactsAliasBeforeLaunching(t *testing.T) {
	rt, driver, req := hostFixture(t)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(req.Checkout, alias); err != nil {
		t.Fatal(err)
	}
	req.Artifacts = filepath.Join(alias, "results")
	result := Run(context.Background(), req, rt)
	if result.ExitCode() == 0 || driver.clones != 0 {
		t.Fatal("overlapping artifacts admitted", result)
	}
	if _, err := os.Stat(filepath.Join(req.Checkout, "results")); !os.IsNotExist(err) {
		t.Fatal("overlap refusal wrote source", err)
	}
}

func TestHostRuntimeRetainsUnconfirmedCloneOwnership(t *testing.T) {
	rt, driver, req := hostFixture(t)
	driver.cloneErr = fmt.Errorf("clone group: %w", ErrUnconfirmedProcessCleanup)
	state := Run(context.Background(), req, rt)
	if state.ExitCode() == 0 || driver.deletes != 0 {
		t.Fatal("uncertain clone deleted", state, driver.deletes)
	}
	p, err := AcquirePosition(context.Background(), filepath.Join(rt.root, "positions"), req.Lane, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if _, present, err := p.ReadClaim(); err != nil || !present {
		t.Fatal("lost clone uncertainty", err)
	}
}

func TestHostRuntimePersistsSecondaryUncertaintyBeforeReleasing(t *testing.T) {
	for _, effect := range []string{"ready", "execute", "export"} {
		t.Run(effect, func(t *testing.T) {
			rt, driver, req := hostFixture(t)
			switch effect {
			case "ready":
				driver.readyErr = ErrUnconfirmedProcessCleanup
			case "execute":
				driver.executeErr = ErrUnconfirmedProcessCleanup
			case "export":
				driver.exportErr = ErrUnconfirmedProcessCleanup
			}
			state := Run(context.Background(), req, rt)
			if state.ExitCode() == 0 || driver.deletes != 0 {
				t.Fatal("secondary uncertainty discarded", state, driver.deletes)
			}
			if driver.running {
				t.Fatal("uncertain secondary prevented stopping the owned main VM")
			}
			p, err := AcquirePosition(context.Background(), filepath.Join(rt.root, "positions"), req.Lane, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			claim, present, err := p.ReadClaim()
			if err != nil || !present || claim.Stage != ClaimCleanupUnknown || claim.StartIdentity != "fixture-start" {
				t.Fatal("lost exact blocked claim", claim, err)
			}
		})
	}
}
func TestHostRuntimeValidatesEvidenceAfterStop(t *testing.T) {
	rt, driver, req := hostFixture(t)
	driver.stopHook = func() {
		if err := os.WriteFile(filepath.Join(rt.results, "status"), []byte("changed after collect"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	state := Run(context.Background(), req, rt)
	if state.ExitCode() == 0 {
		t.Fatal("last writer changed retained evidence after validation")
	}
}

func TestHostRuntimePreclaimsEffectsAndSeparatesCommandStreams(t *testing.T) {
	rt, driver, req := hostFixture(t)
	seen := map[string]int{}
	driver.observe = func(effect string) {
		seen[effect]++
		claim, present, err := rt.position.ReadClaim()
		if err != nil || !present {
			t.Fatalf("effect %s lacks durable uncertainty: %v", effect, err)
		}
		want := ClaimCleanupUnknown
		if effect == "clone" {
			want = ClaimLaunching
		}
		if claim.Stage != want {
			t.Fatalf("effect %s stage=%s want=%s", effect, claim.Stage, want)
		}
	}
	state := Run(context.Background(), req, rt)
	if state.ExitCode() != 0 {
		t.Fatal(state)
	}
	for _, effect := range []string{"inspect", "clone", "ready", "execute", "export", "stop", "delete"} {
		if seen[effect] == 0 {
			t.Fatal("missing production effect", effect)
		}
	}
	for _, stream := range []struct{ name, want string }{{"stdout.log", "command-out"}, {"stderr.log", "command-error"}} {
		raw, err := os.ReadFile(filepath.Join(rt.artifacts, stream.name))
		if err != nil || string(raw) != stream.want {
			t.Fatal("mixed or missing stream", stream.name, string(raw), err)
		}
	}
}
func TestRunLogsChargeOneSharedByteBudget(t *testing.T) {
	budget := &artifactBudget{remaining: 3}
	var logs []*runLog
	for _, name := range []string{"out", "err"} {
		file, err := os.Create(filepath.Join(t.TempDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		logs = append(logs, &runLog{file: file, budget: budget})
	}
	if n, err := logs[0].Write([]byte("ab")); n != 2 || err != nil {
		t.Fatal(n, err)
	}
	if n, err := logs[1].Write([]byte("c")); n != 1 || err != nil {
		t.Fatal(n, err)
	}
	if n, err := logs[0].Write([]byte("d")); n != 0 || err == nil {
		t.Fatal("combined overflow admitted", n, err)
	}
	if budget.remaining != 0 {
		t.Fatal("wrong remaining capacity", budget.remaining)
	}
}
