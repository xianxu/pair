package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartResolutionPreservesSlotIntent(t *testing.T) {
	input := validStartResolutionInput()
	input.OriginalInput = "pair:2"
	input.Action = StartOpen
	input.Target = ThreadTarget{Kind: ThreadTargetSlot, Slot: allocationCandidate(2).Identity}
	got, err := ResolveStartResolution(input)
	if err != nil {
		t.Fatal(err)
	}
	args := got.CommitArgs()
	if args["path"] != "pair:2" || args["action"] != "open" {
		t.Fatalf("args %+v", args)
	}
	input.Action = StartCreate
	changed, err := ResolveStartResolution(input)
	if err != nil || changed.Fingerprint == got.Fingerprint {
		t.Fatalf("action not bound: %v", err)
	}
}
func TestSlotContextReferencesRejectDependencyAndSelectExactNumber(t *testing.T) {
	f := newProvisionFixture(t)
	primary, err := NewWorkspaceProvisioner(f).identity(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	slot := conventionalSlot(f.Primary, 2)
	catalog := &SlotCatalogFake{Repositories: map[string]SlotRepository{f.Primary: {Identity: primary, Slots: []SlotCandidate{{Identity: slot, Verified: true}}}}, Workspaces: map[string]WorkspaceIdentity{".": primary}}
	c := &Couch{Slots: catalog}
	target, _, err := c.resolveSlotInput(context.Background(), primary.Repo+":2")
	if err != nil || target == nil || target.Slot != slot {
		t.Fatalf("exact %+v %v", target, err)
	}
	primary.Kind = "dependency"
	catalog.Workspaces["."] = primary
	if _, _, err := c.resolveSlotInput(context.Background(), ":2"); err == nil {
		t.Fatal("bare slot accepted in dependency")
	}
}

func managedStartFixture(t *testing.T) (*testEnv, *ProvisionFixture) {
	t.Helper()
	f := newProvisionFixture(t)
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.spawn(t, StartArgs{Cwd: f.Primary})
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	return env, f
}
func TestManagedCreatePreviewDoesNotProvisionAndCommitsExactSlot(t *testing.T) {
	env, f := managedStartFixture(t)
	prepared, err := env.Couch.PrepareStart(context.Background(), StartArgs{Cwd: f.Primary, Action: StartCreate})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Resolution.Target.Slot.Number != 1 || f.WeaveCalls != 0 {
		t.Fatalf("preview %+v compile %d", prepared, f.WeaveCalls)
	}
	if _, err := os.Stat(f.host(1)); !os.IsNotExist(err) {
		t.Fatalf("preview created directory: %v", err)
	}
	_, _, err = env.Couch.SpawnPrepared(context.Background(), StartArgs{Cwd: f.Primary, Action: StartCreate}, prepared.Resolution.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if f.WeaveCalls != 1 {
		t.Fatalf("compile calls %d", f.WeaveCalls)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Slots) != 1 || snapshot.Slots[0].Address == (ThreadAddress{}) {
		t.Fatalf("local thread missing %+v", snapshot.Slots)
	}
}
func TestManagedCreateRefusesNumberDriftBeforeProvision(t *testing.T) {
	env, f := managedStartFixture(t)
	args := StartArgs{Cwd: f.Primary, Action: StartCreate}
	prepared, err := env.Couch.PrepareStart(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	_, _, err = env.Couch.SpawnPrepared(context.Background(), args, prepared.Resolution.Fingerprint)
	if !errors.Is(err, ErrStartResolutionChanged) {
		t.Fatalf("drift = %v", err)
	}
	if f.WeaveCalls != 0 {
		t.Fatal("drift compiled")
	}
	if _, err := os.Stat(f.host(2)); !os.IsNotExist(err) {
		t.Fatalf("escalated slot: %v", err)
	}
}

type startReadinessHook struct {
	inner WorkspaceReadiness
	after func()
}

func (h startReadinessHook) Ensure(ctx context.Context, req ProvisionRequest) (ProvisionResult, error) {
	result, err := h.inner.Ensure(ctx, req)
	if err == nil && h.after != nil {
		h.after()
	}
	return result, err
}
func TestManagedCreateRefusesProfileDriftDuringSetup(t *testing.T) {
	env, f := managedStartFixture(t)
	version := "before"
	env.Couch.RepoAgentDefault = func(string, string) (LaunchProfile, bool, error) {
		return LaunchProfile{Agent: "claude", Argv: []string{"--model", version}}, true, nil
	}
	args := StartArgs{Cwd: f.Primary, Action: StartCreate}
	prepared, err := env.Couch.PrepareStart(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.Workspaces = startReadinessHook{inner: env.Couch.Workspaces, after: func() { version = "after" }}
	_, _, err = env.Couch.SpawnPrepared(context.Background(), args, prepared.Resolution.Fingerprint)
	if !errors.Is(err, ErrStartResolutionChanged) {
		t.Fatalf("changed profile launched: %v", err)
	}
	snapshot, e := env.Couch.Threads.Snapshot()
	if e != nil {
		t.Fatal(e)
	}
	for _, record := range snapshot.Records {
		if record.WorkingPath == f.host(1) && len(record.Incarnations) > 0 {
			t.Fatalf("claim leaked %+v", record)
		}
	}
}

func TestManagedCreateParkAppearingDuringSetupPreservesSlotWithoutLaunch(t *testing.T) {
	env, f := managedStartFixture(t)
	args := StartArgs{Cwd: f.Primary, Action: StartCreate}
	prepared, err := env.Couch.PrepareStart(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	once := false
	env.Couch.Workspaces = startReadinessHook{inner: env.Couch.Workspaces, after: func() {
		if !once {
			once = true
			record := actionableTestThread("parked-during-setup", env.Now)
			scope, _ := launcher.ResolveRepoScope(f.Primary)
			record.Address.RepoScope = scope.Key
			record.StartingPath, record.WorkingPath = f.Primary, f.Primary
			record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
			markActionableParked(&record, env.Now)
			if _, e := env.Couch.Threads.CreateThread(record); e != nil {
				t.Fatal(e)
			}
			env.Artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "native-setup-park")
		}
	}}
	_, _, err = env.Couch.SpawnPrepared(context.Background(), args, prepared.Resolution.Fingerprint)
	if err == nil || !strings.Contains(err.Error(), "parked") {
		t.Fatalf("parked admission: %v", err)
	}
	if _, err := os.Stat(f.host(1)); err != nil {
		t.Fatalf("slot lost: %v", err)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range snapshot.Records {
		if record.WorkingPath == f.host(1) {
			t.Fatalf("launched new slot despite parked work: %+v", record)
		}
	}
}
func TestManagedFirstCreateStaysPrimaryAndKeepsSubdirectory(t *testing.T) {
	f := newProvisionFixture(t)
	sub := filepath.Join(f.Primary, "sub")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	prepared, err := env.Couch.PrepareStart(context.Background(), StartArgs{Cwd: sub, Action: StartCreate})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Resolution.Target.Kind == ThreadTargetSlot || prepared.Resolution.CanonicalPath != sub {
		t.Fatalf("primary changed %+v", prepared.Resolution)
	}
}

func TestQualifiedSlotReferenceUsesEnrolledRootOutsideRepository(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot2", f.host(2), "main")
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newTestThreadStore(t)
	if err := store.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	catalog := &SlotCatalogFake{Repositories: map[string]SlotRepository{f.Primary: repository}, Errors: map[string]error{".": errors.New("not in a repository")}}
	c := &Couch{Slots: catalog, Threads: store}
	target, id, err := c.resolveSlotInput(context.Background(), repository.Identity.Repo+":2")
	if err != nil || target == nil || target.Slot.WorktreeRoot != f.host(2) || id.PrimaryRoot != f.Primary {
		t.Fatalf("enrolled ref %+v %+v %v", target, id, err)
	}
	if _, _, err := c.resolveSlotInput(context.Background(), ":2"); err == nil {
		t.Fatal("bare reference guessed repository")
	}
}
func TestManagedPartialSlotPathPreviewsWithoutCreatingHost(t *testing.T) {
	f := newProvisionFixture(t)
	if err := os.MkdirAll(filepath.Dir(f.host(1)), 0700); err != nil {
		t.Fatal(err)
	}
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	prepared, err := env.Couch.PrepareStart(context.Background(), StartArgs{Cwd: f.host(1)})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Resolution.Target.Slot.WorktreeRoot != f.host(1) || prepared.Resolution.Action != StartOpen {
		t.Fatalf("partial target %+v", prepared.Resolution)
	}
	if _, err := os.Stat(f.host(1)); !os.IsNotExist(err) {
		t.Fatalf("preview created partial host %v", err)
	}
}

func TestStartInteractiveRepairsPartialSlotWithoutCreatingConversation(t *testing.T) {
	f := newProvisionFixture(t)
	p := NewWorkspaceProvisioner(f)
	p.Store = recoveryStorage{ProvisionStorage: ProvisionStore{}, write: func(path string, value any) error {
		if err := (ProvisionStore{}).Write(path, value); err != nil {
			return err
		}
		if intent, ok := value.(CreationIntent); ok && intent.DirInode != 0 {
			return errors.New("interrupted after directory publication")
		}
		return nil
	}}
	if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err == nil {
		t.Fatal("injection failed")
	}
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	_, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: f.host(1)})
	if err == nil {
		t.Fatal("missing conversation silently became fresh")
	}
	if _, e := os.Stat(f.host(1)); e != nil {
		t.Fatalf("ordinary open did not repair host: %v (open: %v)", e, err)
	}
	if f.WeaveCalls != 1 {
		t.Fatalf("readiness calls %d", f.WeaveCalls)
	}
	if _, e := os.Stat(filepath.Join(filepath.Dir(f.host(1)), ".couch", "thread.json")); !os.IsNotExist(e) {
		t.Fatalf("repair manufactured conversation: %v", e)
	}
}

func TestPrimaryStartupEnrollsExistingDirectoriesBeforeInventory(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	_, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: f.Primary})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Slots) != 1 || snapshot.Slots[0].Identity.WorktreeRoot != f.host(1) {
		t.Fatalf("primary hid existing slot: %+v", snapshot.Slots)
	}
}

func TestCanonicalThreadReferenceNeverFallsBackToTag(t *testing.T) {
	env, f := managedStartFixture(t)
	primary, err := NewWorkspaceProvisioner(f).identity(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{f.Primary: {Identity: primary}}, Workspaces: map[string]WorkspaceIdentity{".": primary}}
	matches, err := env.Couch.ResolveThreadReference("wrong-scope", ":0")
	if err != nil || len(matches) != 1 {
		t.Fatalf("primary reference: %+v %v", matches, err)
	}
	if _, err := env.Couch.ResolveThreadReference(matches[0].Address.RepoScope, ":2"); err == nil {
		t.Fatal("missing slot fell back to a primary thread")
	}
	path, recognized, err := env.Couch.WorkspaceReferencePath(context.Background(), primary.Repo+":0")
	if err != nil || !recognized || path != f.Primary {
		t.Fatalf("reference path %q %v %v", path, recognized, err)
	}
}

func TestManagedLaunchThenParkBlocksNextCreateButAllowsExistingOpen(t *testing.T) {
	env, f := managedStartFixture(t)
	args := StartArgs{Cwd: f.Primary, Action: StartCreate}
	created, _, err := env.Couch.Spawn(args)
	if err != nil {
		t.Fatal(err)
	}
	record := actionableTestThread("parked-after-launch", env.Now)
	scope, _ := launcher.ResolveRepoScope(f.Primary)
	record.Address.RepoScope = scope.Key
	record.StartingPath, record.WorkingPath = f.Primary, f.Primary
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, env.Now)
	if _, err := env.Couch.Threads.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "native-after-launch-park")
	if _, _, err := env.Couch.Spawn(args); err == nil || !strings.Contains(err.Error(), "parked") {
		t.Fatalf("next create admission: %v", err)
	}
	if _, err := os.Stat(f.host(2)); !os.IsNotExist(err) {
		t.Fatalf("blocked create provisioned slot: %v", err)
	}
	repository, err := env.Couch.Slots.Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{f.Primary: repository}, Workspaces: map[string]WorkspaceIdentity{".": repository.Identity, f.host(1): repository.Identity}}
	matches, err := env.Couch.ResolveThreadReference(scope.Key, ":1")
	if err != nil || len(matches) != 1 || matches[0].Address != created.Thread {
		t.Fatalf("slot ref crossed scope: %+v %v", matches, err)
	}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Artifacts.SetNativeBinding(created.Thread, "claude", sessioninventory.BindingEstablished, "native-existing-slot")
	env.Artifacts.SetSessionPresence(created.Thread, SessionObservation{State: SessionPresent})
	env.Artifacts.SetPairSession(created.Thread, "existing-slot", true)
	env.Artifacts.SetDetachedSession(created.Thread, "existing-slot")
	opened, err := env.Couch.OpenSlot(context.Background(), f.host(1), "")
	if err != nil || opened.Record.Thread != created.Thread {
		t.Fatalf("existing open blocked: %+v %v", opened, err)
	}
}

// #331: numbered slots existing, or a live thread in one of them, do not make
// :0 occupied. Once the primary's own thread is archived, a create on the
// primary path starts there again instead of allocating another slot.
func TestManagedCreateReturnsToPrimaryOnceItsThreadIsArchived(t *testing.T) {
	f := newProvisionFixture(t)
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	primary, _ := env.spawn(t, StartArgs{Cwd: f.Primary})
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	args := StartArgs{Cwd: f.Primary, Action: StartCreate}
	slot, _ := env.spawn(t, args)
	if slot.Thread == primary.Thread {
		t.Fatal("second create reused the occupied primary")
	}
	if err := env.Couch.Threads.ArchiveThread(primary.Thread); err != nil {
		t.Fatal(err)
	}
	prepared, err := env.Couch.PrepareStart(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Resolution.Target.Kind == ThreadTargetSlot || prepared.Resolution.CanonicalPath != f.Primary {
		t.Fatalf("free primary resolved to %+v, want :0", prepared.Resolution.Target)
	}
	if _, err := os.Stat(f.host(2)); !os.IsNotExist(err) {
		t.Fatalf("preview allocated another slot: %v", err)
	}
}
