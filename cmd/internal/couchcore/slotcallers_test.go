package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"path/filepath"
	"testing"
)

// TestDiscoverUnionsNumberSources: slot numbers come from environment
// directories, worktree registrations and resting branches. A number known
// only from leftovers is a candidate marked ErrSlotNeedsReconcile, never
// missing and never a repository-wide failure.
func TestDiscoverUnionsNumberSources(t *testing.T) {
	t.Parallel()

	f := newProvisionFixture(t)
	p := NewWorkspaceProvisioner(f)
	for _, n := range []int{1, 2} {
		if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: n}); err != nil {
			t.Fatal(err)
		}
	}
	// Slot 1: its directory deleted, the registration left behind.
	os.RemoveAll(filepath.Dir(f.host(1)))
	// Slot 3: only a resting branch.
	f.git(f.Primary, "branch", "main-slot3", f.Base)
	repo, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatalf("one broken slot failed the repository's inventory: %v", err)
	}
	got := map[int]SlotCandidate{}
	for _, c := range repo.Slots {
		got[c.Identity.Number] = c
	}
	if c := got[1]; !errors.Is(c.Err, ErrSlotNeedsReconcile) {
		t.Errorf("slot 1 (registration only) = %+v, want needs-reconcile", c)
	}
	if c := got[2]; !c.Verified || c.Err != nil {
		t.Errorf("slot 2 (healthy) = %+v", c)
	}
	if c := got[3]; !errors.Is(c.Err, ErrSlotNeedsReconcile) {
		t.Errorf("slot 3 (branch only) = %+v, want needs-reconcile", c)
	}
}

// TestOpenReconcilesAVerifiedSlotWhoseSetupIsMissing: verifyHost proves only
// identity, so an open of a verified host whose marker is gone must still
// converge setup before the agent starts (red before pair#387 Task 2.5).
func TestOpenReconcilesAVerifiedSlotWhoseSetupIsMissing(t *testing.T) {
	t.Parallel()

	f := newProvisionFixture(t)
	if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
		t.Fatal(err)
	}
	admin := f.git(f.host(1), "rev-parse", "--absolute-git-dir")
	os.Remove(SetupMarkerPath(admin))
	calls := f.WeaveCalls
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	if _, _, err := env.Couch.selectedSlot(context.Background(), f.host(1)); err != nil {
		t.Fatal(err)
	}
	if f.WeaveCalls != calls+1 {
		t.Fatalf("weave calls %d, want the missing setup compiled on open", f.WeaveCalls-calls)
	}
	if _, err := os.Stat(SetupMarkerPath(admin)); err != nil {
		t.Fatalf("marker not restored: %v", err)
	}
}

// TestOpenRefusesWithTheHandoffWhenSetupCannotConverge: the tools:1 shape (a
// dependency with no clone source) refuses with the :0 hand-off and starts no
// agent.
func TestOpenRefusesWithTheHandoffWhenSetupCannotConverge(t *testing.T) {
	t.Parallel()

	f := newProvisionFixture(t)
	if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(f.host(1), "construct", "deps"), "substrate ../ariadne\n")
	os.Remove(SetupMarkerPath(f.git(f.host(1), "rev-parse", "--absolute-git-dir")))
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	_, _, err := env.Couch.selectedSlot(context.Background(), f.host(1))
	var blocked *SlotReconcileError
	if !errors.As(err, &blocked) || blocked.Failure.Class != FailureHandoff || blocked.Failure.Resource != ResourceSetup {
		t.Fatalf("err = %v, want the setup hand-off", err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("an agent started: %v", env.Runner.Ops)
	}
}

// TestRebootGetsPastAHoldItsAdviceNames (BR-10): the hold advice for a slot
// whose repair waits on a live agent says "reboot the slot to repair it", so
// reboot's first pass must get past exactly that hold, stop the agent, and let
// its post-stop pass repair. A hand-off still refuses before anything stops.
func TestRebootGetsPastAHoldItsAdviceNames(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		first    ReconcileFailure
		proceeds bool
	}{
		{"live-agent hold", ReconcileFailure{Resource: ResourceHost, Class: FailureHold, Cause: StopReasonAgentLive}, true},
		{"hand-off", ReconcileFailure{Resource: ResourceSetup, Class: FailureHandoff, Cause: "Error: missing substrate"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			old := slotRecordFixture(t, env, local)
			env.Artifacts.SetPairSession(old.Address, "pair-slot-detached", true)
			env.Artifacts.SetDetachedSession(old.Address, "pair-slot-detached")
			calls := 0
			env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
				calls++
				if calls == 1 {
					return ProvisionResult{}, &SlotReconcileError{Address: "repo-name:1", Repo: "repo-name", Failure: tc.first}
				}
				return slotReadyResult(local), nil
			})
			result, err := rebootLegacyForTest(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": old.Address.RepoScope})
			if !tc.proceeds {
				if err == nil || len(env.Artifacts.Quiesces()) != 0 {
					t.Fatalf("a hand-off must refuse with nothing stopped: err=%v quiesced=%v", err, env.Artifacts.Quiesces())
				}
				return
			}
			if err != nil {
				t.Fatalf("reboot refused the hold its own advice names: %v", err)
			}
			if got := env.Artifacts.Quiesces(); len(got) != 1 || got[0] != old.Address {
				t.Fatalf("quiesced %v, want the slot's agent stopped", got)
			}
			if _, ok := result.Started(); !ok || calls < 2 {
				t.Fatalf("result %+v, reconcile calls %d: want the post-stop pass and a fresh start", result, calls)
			}
		})
	}
}

// TestRebootRepairsABrokenHostUnderALiveAgent (BR-10 at the caller, real git):
// the host is unreadable and git worktree repair cannot fix it (its
// registration is gone), and an agent is detached in the slot. Reboot's first
// pass gets past the live-agent hold through the unverified candidate, stops
// the agent, and its post-stop pass sets the host aside, re-adds it and
// starts fresh. The slot record lives in <env>/.couch, so it survives.
func TestRebootRepairsABrokenHostUnderALiveAgent(t *testing.T) {
	t.Parallel()

	env, local := slotRecoveryOperationFixture(t)
	old := slotRecordFixture(t, env, local)
	f := env.Couch.Slots.(*OSSlotCatalog).IO.(*ProvisionFixture)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	host := local.slot.WorktreeRoot
	admin := f.git(host, "rev-parse", "--absolute-git-dir")
	write(t, filepath.Join(host, "kept.txt"), "operator's file")
	if err := os.RemoveAll(admin); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(old.Address, "pair-slot-detached", true)
	env.Artifacts.SetDetachedSession(old.Address, "pair-slot-detached")
	result, err := rebootLegacyForTest(env, map[string]string{"path": host, "repo-scope": old.Address.RepoScope})
	if err != nil {
		t.Fatalf("reboot of a broken host under a live agent: %v", err)
	}
	if got := env.Artifacts.Quiesces(); len(got) != 1 || got[0] != old.Address {
		t.Fatalf("quiesced %v", got)
	}
	if _, ok := result.Started(); !ok {
		t.Fatalf("no fresh agent: %+v", result)
	}
	if f.git(host, "rev-parse", "--is-inside-work-tree") != "true" {
		t.Fatal("host not re-added")
	}
	entries, _ := os.ReadDir(LayoutOf(*local.slot).SavedWork())
	if len(entries) != 1 {
		t.Fatalf("saved work %v, want the broken host set aside", entries)
	}
	if raw, err := os.ReadFile(filepath.Join(LayoutOf(*local.slot).SavedWork(), entries[0].Name(), "tree", "kept.txt")); err != nil || string(raw) != "operator's file" {
		t.Fatalf("the operator's file was not kept: %q %v", raw, err)
	}
}

// Public Fresh uses the shared coordinator, so this fixture supplies an exact
// owned stop set and acknowledges it explicitly. The retained direct-Reboot
// tests above continue to defend their separate archive/repair API contract.
func TestPreparedRebootDispatcherRepairsBrokenHostAfterConfirmedStop(t *testing.T) {
	for _, realGit := range []bool{false, true} {
		name := "fake-git"
		if realGit {
			name = "real-git"
		}
		t.Run(name, func(t *testing.T) { runPreparedRebootBrokenHost(t, realGit) })
	}
}

func runPreparedRebootBrokenHost(t *testing.T, realGit bool) {
	t.Helper()
	env, local := slotRecoveryOperationFixture(t)
	old := slotRecordFixture(t, env, local)
	if realGit {
		env.Couch.Git = ExecGit{}
	}
	f := env.Couch.Slots.(*OSSlotCatalog).IO.(*ProvisionFixture)
	host := local.slot.WorktreeRoot
	admin := f.git(host, "rev-parse", "--absolute-git-dir")
	write(t, filepath.Join(host, "kept.txt"), "operator's file")
	if err := os.RemoveAll(admin); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(old.Address, "pair-slot-detached", true)
	env.Artifacts.SetDetachedSession(old.Address, "pair-slot-detached")
	env.Proc.Set(700, "owned-server")
	env.Proc.DiesOn[700] = os.Kill
	retired := false
	env.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 700, Identity: "owned-server", Role: "terminal server"}}, Digest: "owned-terminal-generation", Check: func(context.Context) error {
			identity, err := env.Proc.Identity(700)
			if err != nil {
				return err
			}
			if identity != "owned-server" {
				return errors.New("terminal generation changed")
			}
			return nil
		}, Retire: func(context.Context) error {
			if _, err := os.Stat(admin); !errors.Is(err, os.ErrNotExist) {
				return errors.New("host repaired before confirmed stop")
			}
			if err := env.Proc.Signal(700, os.Kill); err != nil {
				return err
			}
			if err := env.Artifacts.Quiesce(old.Address); err != nil {
				return err
			}
			retired = true
			return nil
		}}, nil
	}
	provisioner := NewWorkspaceProvisioner(f)
	repairs := 0
	env.Couch.Workspaces = slotReadinessFunc(func(ctx context.Context, request ProvisionRequest) (ProvisionResult, error) {
		if !retired {
			return ProvisionResult{}, errors.New("workspace repair ran before confirmed retirement")
		}
		repairs++
		return provisioner.Ensure(ctx, request)
	})
	readyNonce := ""
	var readyAddress ThreadAddress
	env.Couch.FreshRegistration = func(_ context.Context, address ThreadAddress, _ string, nonce string) (bool, error) {
		return nonce != "" && nonce == readyNonce && address == readyAddress, nil
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		child := env.Runner.Child(id)
		var profile launcher.TrustedLaunchProfile
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
			return err
		}
		if !profile.FreshRequired || profile.ResumeRequired || profile.FreshNonce == "" || profile.Prompt == "" {
			return errors.New("Fresh launch lacks strict recovery identity")
		}
		readyAddress = ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(profile.Tag)}
		current, err := local.GetThread(readyAddress)
		if err != nil {
			return err
		}
		if current.Incarnations[0].Start == nil || current.Incarnations[0].Start.Nonce != profile.FreshNonce {
			return errors.New("fresh nonce differs from committed claim")
		}
		readyNonce = profile.FreshNonce
		env.Artifacts.SetPairSession(readyAddress, continuationChildSession(t, env.Runner, id), true)
		env.Artifacts.SetSessionPresence(readyAddress, SessionObservation{State: SessionPresent})
		return nil
	}
	executor := OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}
	_, err := DispatchOperation(executor, OperationCall{Name: OpReboot, Args: map[string]string{"path": host, "repo-scope": old.Address.RepoScope}, Implicit: true, Context: t.Context()})
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) || retired || repairs != 0 || len(env.Proc.Signals[700]) != 0 {
		t.Fatalf("pre-confirm effects: err=%v retired=%v repairs=%d signals=%v", err, retired, repairs, env.Proc.Signals)
	}
	value, err := DispatchOperation(executor, OperationCall{Name: "recovery-confirmation", Args: map[string]string{"token": required.Offer.Token}, Implicit: true, Context: t.Context()})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := value.(StartedChild)
	if !ok {
		t.Fatalf("Fresh result %T", value)
	}
	start, ok := child.Started()
	if !ok || start.Record.Thread == old.Address || !retired || repairs == 0 {
		t.Fatalf("Fresh did not repair after retirement: %+v retired=%v repairs=%d", start, retired, repairs)
	}
	if f.git(host, "rev-parse", "--is-inside-work-tree") != "true" {
		t.Fatal("host not repaired")
	}
	entries, _ := os.ReadDir(LayoutOf(*local.slot).SavedWork())
	if len(entries) != 1 {
		t.Fatalf("saved work %v", entries)
	}
	raw, err := os.ReadFile(filepath.Join(LayoutOf(*local.slot).SavedWork(), entries[0].Name(), "tree", "kept.txt"))
	if err != nil || string(raw) != "operator's file" {
		t.Fatalf("operator work lost: %q %v", raw, err)
	}
}
