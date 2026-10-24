package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestFreshHubDispatchStartsNeverUsedManagedSlot(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	result, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{Name: "start-hub", Implicit: true, Args: map[string]string{"path": local.slot.WorktreeRoot, "action": string(StartFresh)}, Context: context.Background()})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := result.(StartedChild)
	if !ok {
		t.Fatalf("missing child result: %T", result)
	}
	start, ok := child.Started()
	if !ok || start.Handle == nil {
		t.Fatalf("fresh managed hub not started: %+v", result)
	}
	if start.Record.Args.WorkingDir() != local.slot.WorktreeRoot {
		t.Fatalf("wrong hub: %+v", start.Record)
	}
}

func TestFreshHubCannotCreateOverRetainedArchive(t *testing.T) {
	for _, number := range []int{0, 1} {
		t.Run(string(rune('0'+number)), func(t *testing.T) {
			f := newProvisionFixture(t)
			f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
			env := newTestEnv(t, f.Primary, f.host(1))
			env.Couch.Slots = NewOSSlotCatalog(f)
			env.Couch.Workspaces = NewWorkspaceProvisioner(f)
			env.Git.replies[GitCall{Dir: f.host(1), Args: "rev-parse --git-common-dir"}] = filepath.Join(f.Primary, ".git")
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			path, store := f.Primary, env.Couch.Threads
			if number == 1 {
				path = f.host(1)
				repo, err := env.Couch.Slots.Discover(context.Background(), f.Primary)
				if err != nil {
					t.Fatal(err)
				}
				store = newSlotThreadStore(env.Couch.Namespace, repo.Slots[0].Identity)
			}
			scope, err := launcher.ResolveRepoScope(path)
			if err != nil {
				t.Fatal(err)
			}
			archive := store.archivePath(ThreadAddress{RepoScope: scope.Key, Tag: "couch-0123456789abcdef"})
			if err := os.MkdirAll(filepath.Dir(archive), 0700); err != nil {
				t.Fatal(err)
			}
			retained := validThreadRecord(t)
			retained.Address = ThreadAddress{RepoScope: scope.Key, Tag: "couch-0123456789abcdef"}
			retained.StartingPath, retained.WorkingPath = path, path
			encoded, err := encodeThreadRecord(retained)
			if err != nil {
				t.Fatal(err)
			}
			raw := string(encoded)
			if err := os.WriteFile(archive, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			start, err := env.Couch.StartFreshHub(context.Background(), path)
			if err == nil || start.Handle != nil || len(env.Runner.Ops) != 0 {
				t.Fatalf("automatic startup used occupied target: %+v err=%v effects=%+v", start, err, env.Runner.Ops)
			}
			if got, err := os.ReadFile(archive); err != nil || string(got) != raw {
				t.Fatalf("retained archive changed: %q %v", got, err)
			}
		})
	}
}

func TestFreshHubCannotReplaceStoppedCurrent(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, err := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	old := validThreadRecord(t)
	old.Address.RepoScope = scope.Key
	old.StartingPath, old.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	if _, err := local.CreateThread(old); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(local.recordPath(old.Address))
	if err != nil {
		t.Fatal(err)
	}
	start, err := env.Couch.StartFreshHub(context.Background(), local.slot.WorktreeRoot)
	if err == nil || start.Handle != nil || len(env.Runner.Ops) != 0 {
		t.Fatalf("startup replaced stopped conversation: %+v err=%v effects=%v", start, err, env.Runner.Ops)
	}
	after, err := os.ReadFile(local.recordPath(old.Address))
	if err != nil || string(before) != string(after) {
		t.Fatalf("current history changed: %v", err)
	}
}

func TestFreshHubResolvesRealPrimaryAndNumberedCatalogTargets(t *testing.T) {
	for _, number := range []int{0, 1} {
		t.Run(string(rune('0'+number)), func(t *testing.T) {
			f := newProvisionFixture(t)
			f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
			env := newTestEnv(t, f.Primary, f.host(1))
			env.Couch.Slots = NewOSSlotCatalog(f)
			env.Couch.Workspaces = NewWorkspaceProvisioner(f)
			env.Git.replies[GitCall{Dir: f.host(1), Args: "rev-parse --git-common-dir"}] = filepath.Join(f.Primary, ".git")
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			path := f.Primary
			if number == 1 {
				path = f.host(1)
			}
			start, err := env.Couch.StartFreshHub(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := start.Started(); !ok || start.Record.Args.WorkingDir() != path {
				t.Fatalf("wrong started target: %+v", start)
			}
		})
	}
}
