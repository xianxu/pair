package couchcore

import (
	"context"
	"path/filepath"
	"testing"
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
