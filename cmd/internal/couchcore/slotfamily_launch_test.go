package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func subdirectoryFamilyFixture(t *testing.T) (*testEnv, *ProvisionFixture, string) {
	t.Helper()
	f := newProvisionFixture(t)
	relative := filepath.Join("competition", "arc-agi-3")
	for _, name := range []string{"arc-agi-3", "arc-agi-2"} {
		dir := filepath.Join(f.Primary, "competition", name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README"), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	f.git(f.Primary, "add", "competition")
	f.git(f.Primary, "commit", "-m", "competition directories")
	f.git(f.Primary, "push", "upstream", "main")
	env := newTestEnv(t, f.Primary)
	env.Couch.Git, env.Couch.Path = ExecGit{}, OSPathOps{}
	env.Couch.Slots, env.Couch.Workspaces = NewOSSlotCatalog(f), NewWorkspaceProvisioner(f)
	return env, f, relative
}

func TestRepositoryFamilyLaunchKeepsRelativeDirectoryAcrossSlots(t *testing.T) {
	env, f, relative := subdirectoryFamilyFixture(t)
	start := filepath.Join(f.Primary, relative)
	primary, _, err := env.Couch.Spawn(StartArgs{Cwd: start})
	if err != nil {
		t.Fatal(err)
	}
	if primary.Args.WorkingDir() != start || string(primary.Args.Worktree) != f.Primary {
		t.Fatalf("primary paths %+v", primary.Args)
	}
	prepared, err := env.Couch.PrepareStart(t.Context(), StartArgs{Cwd: start, Action: StartCreate})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(f.host(1), relative)
	if prepared.Resolution.CanonicalPath != want {
		t.Fatalf("added slot cwd %q want %q", prepared.Resolution.CanonicalPath, want)
	}
	actor, handle, err := env.Couch.SpawnPrepared(t.Context(), StartArgs{Cwd: start, Action: StartCreate}, prepared.Resolution.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if actor.Args.WorkingDir() != want || string(actor.Args.Worktree) != f.host(1) || env.Runner.Child(handle.ID()).Dir != want {
		t.Fatalf("actor %+v child %+v", actor.Args, env.Runner.Child(handle.ID()))
	}
	thread, err := env.Couch.Threads.GetThread(actor.Thread)
	if err != nil || thread.StartingPath != want || thread.WorkingPath != want {
		t.Fatalf("thread %+v %v", thread, err)
	}
}

func TestRepositoryFamilyConflictingDirectoryRefusedAfterParkAndRestart(t *testing.T) {
	env, f, relative := subdirectoryFamilyFixture(t)
	actor, _, err := env.Couch.Spawn(StartArgs{Cwd: filepath.Join(f.Primary, relative)})
	if err != nil {
		t.Fatal(err)
	}
	record, err := env.Couch.Threads.GetThread(actor.Thread)
	if err != nil {
		t.Fatal(err)
	}
	park := ParkIdentity{Nonce: "family-park", Address: actor.Thread, PID: actor.PID, ProcessIdentity: actor.Identity}
	begun, err := env.Couch.Threads.BeginPark(actor.Thread, record.Revision, park)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = env.Couch.Threads.FinalizePark(actor.Thread, begun.Revision, park, 1, env.Now); err != nil {
		t.Fatal(err)
	}
	env.Proc.Kill(actor.PID)
	if err = env.Couch.Forget(actor.Args.Worktree, actor.ID); err != nil {
		t.Fatal(err)
	}
	env.Couch.Threads = NewThreadStore(env.Couch.Namespace)
	before := len(env.Runner.Ops)
	conflicting := filepath.Join(f.Primary, "competition", "arc-agi-2")
	for _, path := range []string{conflicting, f.Primary} {
		for _, action := range []StartAction{StartOpen, StartCreate} {
			_, _, err = env.Couch.Spawn(StartArgs{Cwd: path, Action: action})
			if err == nil || !strings.Contains(err.Error(), "family") {
				t.Fatalf("conflicting family action %s: %v", action, err)
			}
		}
	}
	if len(env.Runner.Ops) != before {
		t.Fatal("conflict launched helper")
	}
	if _, err = os.Stat(f.host(1)); !os.IsNotExist(err) {
		t.Fatalf("conflict provisioned slot: %v", err)
	}
}

func TestRepositoryFamilyProvisionedSlotMissingStartingDirectoryRefusesSpawn(t *testing.T) {
	for _, escape := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "symlink escape"}[escape], func(t *testing.T) {
			env, f, relative := subdirectoryFamilyFixture(t)
			start := filepath.Join(f.Primary, relative)
			if _, _, err := env.Couch.Spawn(StartArgs{Cwd: start}); err != nil {
				t.Fatal(err)
			}
			before := len(env.Runner.Ops)
			env.Couch.Workspaces = startReadinessHook{inner: env.Couch.Workspaces, after: func() {
				path := filepath.Join(f.host(1), relative)
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				if escape {
					if err := os.Symlink(t.TempDir(), path); err != nil {
						t.Fatal(err)
					}
				}
			}}
			if _, _, err := env.Couch.Spawn(StartArgs{Cwd: start, Action: StartCreate}); err == nil {
				t.Fatal("unavailable family path launched")
			}
			if len(env.Runner.Ops) != before {
				t.Fatal("unavailable path spawned helper")
			}
			if info, err := os.Stat(f.host(1)); err != nil || !info.IsDir() {
				t.Fatalf("provisioned workspace was removed: %v", err)
			}
		})
	}
}

func TestRepositoryFamilyAmbiguousLegacySlotOpensRecordedDirectory(t *testing.T) {
	env, f, relative := subdirectoryFamilyFixture(t)
	env.Couch.Slots = nil
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: filepath.Join(f.Primary, relative)}); err != nil {
		t.Fatal(err)
	}
	env.Couch.Slots = NewOSSlotCatalog(f)
	if _, err := env.Couch.Workspaces.Ensure(t.Context(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
		t.Fatal(err)
	}
	repository, err := env.Couch.Slots.Discover(t.Context(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if err = env.Couch.Threads.EnrollSlotRepository(t.Context(), repository); err != nil {
		t.Fatal(err)
	}
	slot := repository.Slots[0].Identity
	record := actionableTestThread("legacy-other-directory", env.Now)
	scope, _ := launcher.ResolveRepoScope(slot.WorktreeRoot)
	record.Address.RepoScope = scope.Key
	record.StartingPath = filepath.Join(slot.WorktreeRoot, "competition", "arc-agi-2")
	record.WorkingPath = record.StartingPath
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	if _, err = env.Couch.Threads.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetDetachedSession(record.Address, "pair-legacy-survivor")
	if _, err = env.Couch.PrepareStart(t.Context(), StartArgs{Cwd: filepath.Join(f.Primary, relative), Action: StartCreate}); err == nil {
		t.Fatal("ambiguous family admitted new slot")
	}
	args := StartArgs{Cwd: slot.WorktreeRoot, Action: StartOpen}
	prepared, err := env.Couch.PrepareStart(t.Context(), args)
	if err != nil {
		t.Fatal(err)
	}
	actor, h, err := env.Couch.SpawnPrepared(t.Context(), args, prepared.Resolution.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if actor.Thread != record.Address || actor.Args.WorkingDir() != record.WorkingPath || string(actor.Args.Worktree) != slot.WorktreeRoot || env.Runner.Child(h.ID()).Dir != record.WorkingPath {
		t.Fatalf("legacy open lost directory: %+v", actor)
	}
}

func TestRepositoryFamilyBareRelativeDirectoryIsExplicit(t *testing.T) {
	env, f, relative := subdirectoryFamilyFixture(t)
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: filepath.Join(f.Primary, relative)}); err != nil {
		t.Fatal(err)
	}
	identity, err := env.Couch.slotWorkspace(t.Context(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := env.Couch.Slots.Discover(t.Context(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{f.Primary: repository}, Workspaces: map[string]WorkspaceIdentity{"competition": identity}}
	env.Couch.Path = NewFakePathOps(map[string]string{NormalizePath("competition"): filepath.Join(f.Primary, "competition")})
	if _, err := env.Couch.PrepareStart(t.Context(), StartArgs{Cwd: "competition", Action: StartCreate}); err == nil {
		t.Fatal("bare relative directory was treated as inherited repo alias")
	}
}

func parkFamilyActor(t *testing.T, env *testEnv, actor ActorRecord) {
	t.Helper()
	current, err := env.Couch.Threads.GetThread(actor.Thread)
	if err != nil {
		t.Fatal(err)
	}
	park := ParkIdentity{Nonce: "family-park-" + string(actor.ID), Address: actor.Thread, PID: actor.PID, ProcessIdentity: actor.Identity}
	begun, err := env.Couch.Threads.BeginPark(actor.Thread, current.Revision, park)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = env.Couch.Threads.FinalizePark(actor.Thread, begun.Revision, park, 1, env.Now); err != nil {
		t.Fatal(err)
	}
	env.Proc.Kill(actor.PID)
	if err = env.Couch.Forget(actor.Args.Worktree, actor.ID); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(actor.Thread, current.SessionBinding.Name, false)
	env.Artifacts.SetNativeBinding(actor.Thread, actor.Args.Stack, sessioninventory.BindingEstablished, "native-family")
}

func TestRepositoryFamilySlotOpenResumeAndFreshKeepStartingDirectory(t *testing.T) {
	env, f, relative := subdirectoryFamilyFixture(t)
	start := filepath.Join(f.Primary, relative)
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: start}); err != nil {
		t.Fatal(err)
	}
	actor, _, err := env.Couch.Spawn(StartArgs{Cwd: start, Action: StartCreate})
	if err != nil {
		t.Fatal(err)
	}
	env.Proc.Set(actor.PID, actor.Identity)
	before := len(env.Runner.Ops)
	opened, err := env.Couch.OpenSlot(t.Context(), f.host(1), "")
	if err != nil || opened.Record.Thread != actor.Thread || len(env.Runner.Ops) != before {
		t.Fatalf("existing open %+v %v", opened, err)
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		child := env.Runner.Child(id)
		address := ThreadAddress{RepoScope: childEnvValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(childEnvValue(child.Env, "COUCH_THREAD_TAG"))}
		env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	parkFamilyActor(t, env, actor)
	resumed, h, err := env.Couch.Resume(actor.Thread)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(f.host(1), relative)
	if resumed.Args.WorkingDir() != want || string(resumed.Args.Worktree) != f.host(1) || env.Runner.Child(h.ID()).Dir != want {
		t.Fatalf("resume paths %+v", resumed.Args)
	}
	parkFamilyActor(t, env, resumed)
	fresh, err := env.Couch.StartFreshSlot(t.Context(), f.host(1), "")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Record.Thread == actor.Thread || fresh.Record.Args.WorkingDir() != want || string(fresh.Record.Args.Worktree) != f.host(1) || env.Runner.Child(fresh.Handle.ID()).Dir != want {
		t.Fatalf("fresh paths %+v", fresh.Record)
	}
}

func TestRepositoryFamilyDependencyCannotStartConflictingDirectory(t *testing.T) {
	env, f, _ := subdirectoryFamilyFixture(t)
	environment := filepath.Dir(f.host(1))
	if err := os.MkdirAll(environment, 0700); err != nil {
		t.Fatal(err)
	}
	dependency := filepath.Join(environment, "dependency")
	f.git(environment, "clone", f.Remote, dependency)
	id := WorkspaceIdentity{SchemaVersion: 2, Repo: "dependency", RepoIdentity: filepath.Join(dependency, ".git"), PrimaryRoot: dependency, WorktreeRoot: dependency, FleetRoot: filepath.Dir(f.Primary), EnvironmentRoot: environment, Kind: "dependency", EnvironmentHost: &EnvironmentHost{Repo: filepath.Base(f.Primary), Slot: 1, RepoIdentity: filepath.Join(f.Primary, ".git"), PrimaryRoot: f.Primary, WorktreeRoot: f.host(1)}}
	start := filepath.Join(dependency, "competition", "arc-agi-3")
	conflict := filepath.Join(dependency, "competition", "arc-agi-2")
	env.Couch.Slots = &SlotCatalogFake{Workspaces: map[string]WorkspaceIdentity{dependency: id, start: id, conflict: id}}
	actor, _, err := env.Couch.Spawn(StartArgs{Cwd: start})
	if err != nil {
		t.Fatal(err)
	}
	if actor.Args.WorkingDir() != start || string(actor.Args.Worktree) != dependency {
		t.Fatalf("dependency ordinary paths %+v", actor.Args)
	}
	before := len(env.Runner.Ops)
	if _, _, err = env.Couch.Spawn(StartArgs{Cwd: conflict}); err == nil {
		t.Fatal("dependency bypassed family restriction")
	}
	if len(env.Runner.Ops) != before || f.WeaveCalls != 0 {
		t.Fatal("dependency conflict launched or provisioned")
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil || len(snapshot.Slots) != 0 {
		t.Fatalf("dependency became numbered family: %+v %v", snapshot.Slots, err)
	}
}
