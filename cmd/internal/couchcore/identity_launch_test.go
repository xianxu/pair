package couchcore

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"strings"
	"testing"
)

func TestSpawnUsesStoreConversationAndTerminalCounters(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Identities = couchidentity.IdentityStore{HostDir: t.TempDir(), StoreDir: env.Dir}
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo", Stack: "claude"})
	if string(first.Thread.Tag) != "1-repo-1" {
		t.Fatalf("tag=%s", first.Thread.Tag)
	}
	record, err := env.Couch.Threads.GetThread(first.Thread)
	if err != nil {
		t.Fatal(err)
	}
	if record.SessionBinding == nil || record.SessionBinding.Name != "📁1-1" {
		t.Fatalf("binding=%+v", record.SessionBinding)
	}
}

func TestSpawnRefusesMissingIdentityAllocator(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Identities = nil
	_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo", Stack: "claude"})
	if err == nil || !strings.Contains(err.Error(), "identity allocator") {
		t.Fatalf("err=%v", err)
	}
}

func TestManagedTerminalLifetimesThroughColdAndWarmResume(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Identities = couchidentity.IdentityStore{HostDir: t.TempDir(), StoreDir: env.Dir}
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.AfterAcknowledge = func(id string) error {
		child := env.Runner.Child(id)
		for _, entry := range child.Env {
			if raw, ok := strings.CutPrefix(entry, launcher.CouchSessionIntentEnv+"="); ok {
				var intent launcher.CouchSessionIntent
				if err := json.Unmarshal([]byte(raw), &intent); err != nil {
					return err
				}
				env.Artifacts.SetPairSession(parked.Address, intent.Name, true)
			}
		}
		return nil
	}
	first, h, err := env.Couch.Resume(parked.Address)
	if err != nil {
		t.Fatal(err)
	}
	current, err := env.Couch.Threads.GetThread(parked.Address)
	if err != nil {
		t.Fatal(err)
	}
	if current.SessionBinding.Name != "📁1-1" || current.Address != parked.Address {
		t.Fatalf("cold identity: %+v", current)
	}
	env.Proc.Kill(first.PID)
	env.Runner.SetExited(h.ID(), 0)
	env.Couch.reg = env.Couch.reg.RemoveActor(first.Args.Worktree, first.ID)
	current, err = env.Couch.Threads.RetireIncarnation(current.Address, current.Revision, ProcessIdentity{PID: first.PID, Identity: first.Identity}, env.Now)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetDetachedSession(current.Address, current.SessionBinding.Name)
	second, h2, err := env.Couch.Resume(current.Address)
	if err != nil {
		t.Fatal(err)
	}
	current, err = env.Couch.Threads.GetThread(current.Address)
	if err != nil {
		t.Fatal(err)
	}
	if current.SessionBinding.Name != "📁1-1" || second.Shape != StartWarmReattach {
		t.Fatalf("warm identity: %+v %+v", current.SessionBinding, second)
	}
	env.Proc.Kill(second.PID)
	env.Runner.SetExited(h2.ID(), 0)
	env.Couch.reg = env.Couch.reg.RemoveActor(second.Args.Worktree, second.ID)
	identity := ParkIdentity{Nonce: "park-lifecycle", Address: current.Address, PID: second.PID, ProcessIdentity: second.Identity}
	current, err = env.Couch.Threads.BeginPark(current.Address, current.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	current, err = env.Couch.Threads.FinalizePark(current.Address, current.Revision, identity, 1, env.Now)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetDetachedSession(current.Address, "")
	env.Artifacts.SetPairSession(current.Address, "📁1-1", false)
	third, _, err := env.Couch.Resume(current.Address)
	if err != nil {
		t.Fatal(err)
	}
	current, err = env.Couch.Threads.GetThread(current.Address)
	if err != nil {
		t.Fatal(err)
	}
	if current.SessionBinding.Name != "📁1-2" || third.Thread != parked.Address {
		t.Fatalf("recreated identity: %+v", current)
	}
}
