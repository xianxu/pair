package couchcore

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryNamesAtConversationAllocationBoundaries(t *testing.T) {
	for _, name := range []string{"项目", "!!!", strings.Repeat("a", 255)} {
		for _, fresh := range []bool{false, true} {
			if fresh && len(name) == 255 {
				continue
			}
			entry := "new"
			if fresh {
				entry = "fresh"
			}
			t.Run(entry+"/"+name, func(t *testing.T) {
				f := newProvisionFixture(t, name)
				path := f.Primary
				if fresh {
					f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
					path = f.host(1)
				}
				env := newTestEnv(t, path)
				env.Couch.Git = ExecGit{}
				env.Couch.Path = OSPathOps{}
				env.Couch.Identities = couchidentity.IdentityStore{HostDir: t.TempDir(), StoreDir: env.Dir}
				env.Runner.AfterAcknowledge = func(id string) error {
					child := env.Runner.Child(id)
					var intent launcher.CouchSessionIntent
					if err := json.Unmarshal([]byte(childEnvValue(child.Env, launcher.CouchSessionIntentEnv)), &intent); err != nil {
						return err
					}
					env.Artifacts.SetPairSession(ThreadAddress{RepoScope: intent.Scope, Tag: ThreadTag(intent.Tag)}, intent.Name, true)
					return nil
				}
				var actor ActorRecord
				var err error
				if !fresh {
					actor, _, err = env.Couch.Spawn(StartArgs{Worktree: Worktree(path), Cwd: path, Stack: "claude"})
				} else {
					env.Couch.Slots = NewOSSlotCatalog(f)
					repository, e := env.Couch.Slots.Discover(context.Background(), f.Primary)
					if e != nil {
						t.Fatal(e)
					}
					if e = env.Couch.Threads.EnrollSlotRepository(context.Background(), repository); e != nil {
						t.Fatal(e)
					}
					local := newSlotThreadStore(env.Couch.Namespace, repository.Slots[0].Identity)
					env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) { return slotReadyResult(local), nil })
					env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
					result, e := env.Couch.StartFreshSlot(context.Background(), path, "claude")
					actor, err = result.Record, e
				}
				if err != nil {
					t.Fatal(err)
				}
				token := "repo"
				if len(name) == 255 {
					token = strings.Repeat("a", 64)
				}
				if string(actor.Thread.Tag) != "1-"+token+"-1" {
					t.Fatalf("tag=%q", actor.Thread.Tag)
				}
				record, e := env.Couch.Threads.GetThread(actor.Thread)
				if e != nil {
					t.Fatal(e)
				}
				if record.SessionBinding == nil || record.SessionBinding.Name != "📁1-1" {
					t.Fatalf("binding=%+v", record.SessionBinding)
				}
				if filepath.Base(record.StartingPath) != name {
					t.Fatalf("starting path=%q", record.StartingPath)
				}
			})
		}
	}
}
