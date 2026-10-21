package vmcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type hostFakeChild struct{}

func (hostFakeChild) PID() int         { return 123 }
func (hostFakeChild) Identity() string { return "fixture-start" }

type hostFakeDriver struct {
	exists, running, uncertainStop bool
	input, results                 string
	clones, launches, deletes      int
	commandExit                    int
	cloneErr                       error
}

func (f *hostFakeDriver) Inspect(context.Context, string) (VMStatus, error) {
	state := VMStopped
	if f.running {
		state = VMRunning
	}
	return VMStatus{Exists: f.exists, State: state}, nil
}
func (f *hostFakeDriver) Clone(_ context.Context, _, _ string) error {
	if f.exists {
		return fmt.Errorf("clone collision")
	}
	f.exists = true
	f.clones++
	return f.cloneErr
}
func (f *hostFakeDriver) Launch(_ context.Context, _, input, results string, _ io.Writer) (VMChild, error) {
	if !f.exists || f.running {
		return nil, fmt.Errorf("bad launch")
	}
	f.running = true
	f.input = input
	f.results = results
	f.launches++
	return hostFakeChild{}, nil
}
func (f *hostFakeDriver) AwaitGuest(context.Context, string) error {
	if !f.running {
		return fmt.Errorf("not running")
	}
	return nil
}
func (f *hostFakeDriver) Execute(_ context.Context, _ string, _ []string, _, _ io.Writer) (int, error) {
	if !f.running {
		return -1, fmt.Errorf("not running")
	}
	err := os.WriteFile(filepath.Join(f.results, "status"), []byte(fmt.Sprintf("exit_code=%d\ncollection_error=0\n", f.commandExit)), 0600)
	return f.commandExit, err
}
func (f *hostFakeDriver) Stop(context.Context, string, VMChild) error {
	if f.uncertainStop {
		return fmt.Errorf("unconfirmed stop")
	}
	f.running = false
	return nil
}
func (f *hostFakeDriver) Delete(context.Context, string) error {
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
	return RecoveryEvidence{PID: c.PID, Identity: c.StartIdentity, Process: ProcessGone, VM: state}, nil
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
			if _, err := os.Stat(filepath.Join(driver.results, "status")); err != nil {
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
