package couchcore

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Exercise publication through real starts, then reload both Couch and its
// stores. The fake owns process/session evidence; Git and preferences are real.
func TestSlotPreferencesIndependentAcrossRestartResumeAndFresh(t *testing.T) {
	ctx := context.Background()
	f := newProvisionFixture(t)
	env := newTestEnv(t, f.Primary)
	env.Couch.Git, env.Couch.Path = ExecGit{}, OSPathOps{}
	profiles := []LaunchProfile{
		{Agent: "claude", Argv: []string{"--verbose"}},
		{Agent: "codex", Argv: []string{"--no-alt-screen"}},
		{Agent: "muse", Argv: []string{}},
	}
	env.Couch.RepoAgentDefault = func(_ string, agent string) (LaunchProfile, bool, error) {
		for _, profile := range profiles {
			if profile.Agent == agent {
				return profile, true, nil
			}
		}
		return LaunchProfile{}, false, nil
	}
	actors := make([]ActorRecord, 3)
	paths := []string{f.Primary, f.host(1), f.host(2)}
	for i, profile := range profiles {
		args := StartArgs{Cwd: f.Primary, Stack: profile.Agent}
		if i > 0 {
			args.Action = StartCreate
		}
		actor, _, err := env.Couch.Spawn(args)
		if err != nil {
			t.Fatalf("start :%d: %v", i, err)
		}
		actors[i] = actor
		env.Proc.Set(actor.PID, actor.Identity)
		if i == 0 {
			env.Couch.Slots = NewOSSlotCatalog(f)
			env.Couch.Workspaces = NewWorkspaceProvisioner(f)
		}
	}
	identity := filepath.Join(f.Primary, ".git")
	preferenceFiles := make([]string, 3)
	before := make([][]byte, 3)
	for i, path := range paths {
		store, err := env.Couch.Threads.storeForPath(path)
		if err != nil {
			t.Fatal(err)
		}
		preferenceFiles[i] = store.pathLaunchPreferencePath(identity, path)
		before[i], err = os.ReadFile(preferenceFiles[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	assertPreferences := func() {
		t.Helper()
		for i, path := range paths {
			preference, found, err := env.Couch.Threads.GetPathLaunchPreference(identity, path)
			if err != nil || !found || preference.LastAgent != profiles[i].Agent || !reflect.DeepEqual(preference.ArgvByAgent[profiles[i].Agent], profiles[i].Argv) {
				t.Fatalf(":%d preference = %+v, found %v, err %v", i, preference, found, err)
			}
		}
	}
	assertPreferences()
	reopened, err := New(env.Couch.Namespace, env.Runner, OSPathOps{}, ExecGit{}, env.Proc, NewStore(env.Dir), FixedClock{T: env.Now}, NewFixedIDGen("c3d4", "e5f6"), newIncrementingEntropy(), env.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch = reopened
	env.Couch.Slots, env.Couch.Workspaces = NewOSSlotCatalog(f), NewWorkspaceProvisioner(f)
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	assertPreferences()

	// Persist the normal park transaction, with the fake supplying child death
	// and an established native conversation; resume uses the public launch API.
	for i, actor := range actors {
		record, err := env.Couch.Threads.GetThread(actor.Thread)
		if err != nil {
			t.Fatal(err)
		}
		park := ParkIdentity{Nonce: fmt.Sprintf("preferences-park-%d", i), Address: actor.Thread, PID: actor.PID, ProcessIdentity: actor.Identity}
		begun, err := env.Couch.Threads.BeginPark(actor.Thread, record.Revision, park)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := env.Couch.Threads.FinalizePark(actor.Thread, begun.Revision, park, 1, env.Now); err != nil {
			t.Fatal(err)
		}
		env.Proc.Kill(actor.PID)
		if err := env.Couch.Forget(actor.Args.Worktree, actor.ID); err != nil {
			t.Fatal(err)
		}
		env.Artifacts.SetPairSession(actor.Thread, "", false)
		env.Artifacts.SetNativeBinding(actor.Thread, profiles[i].Agent, sessioninventory.BindingEstablished, fmt.Sprintf("native-preferences-%d", i))
		env.Runner.AfterAcknowledge = func(string) error {
			env.Artifacts.SetPairSession(actor.Thread, "pair-"+string(actor.Thread.Tag), true)
			return nil
		}
		resumed, _, err := env.Couch.Resume(actor.Thread)
		if err != nil {
			t.Fatalf("resume :%d: %v", i, err)
		}
		if resumed.Args.Stack != profiles[i].Agent || !reflect.DeepEqual(resumed.Args.ExtraArgs, profiles[i].Argv) {
			t.Fatalf(":%d resumed %+v", i, resumed.Args)
		}
		actors[i] = resumed
	}
	assertPreferences()
	env.Runner.AfterAcknowledge = nil
	for i, path := range preferenceFiles {
		before[i], err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	// Switching :1 must publish there alone, retaining its previous agent argv.
	source := actors[1]
	env.Proc.Kill(source.PID)
	if err := env.Couch.Forget(source.Args.Worktree, source.ID); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(source.Thread, "", false)
	edited := []string{"--verbose", "--debug"}
	preview, err := env.Couch.PrepareAgentSwitch(ctx, source.Thread, "claude", &edited)
	if err != nil {
		t.Fatal(err)
	}
	switched, err := env.Couch.SwitchAgent(ctx, SwitchAgentRequest{Address: source.Thread, Agent: "claude", Argv: edited, AcceptedFingerprint: preview.Fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	profiles[1] = LaunchProfile{Agent: "claude", Argv: edited}
	assertPreferences()
	preference, _, err := env.Couch.Threads.GetPathLaunchPreference(identity, paths[1])
	if err != nil || !reflect.DeepEqual(preference.ArgvByAgent["codex"], []string{"--no-alt-screen"}) {
		t.Fatalf("lost previous agent: %+v %v", preference, err)
	}
	env.Proc.Kill(switched.Record.PID)
	env.Artifacts.SetPairSession(source.Thread, "", false)
	fresh, err := env.Couch.StartFreshSlot(ctx, paths[1], "")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Record.Thread == source.Thread || fresh.Record.Args.Stack != "claude" || !reflect.DeepEqual(fresh.Record.Args.ExtraArgs, edited) {
		t.Fatalf("fresh %+v", fresh.Record)
	}
	assertPreferences()
	for _, i := range []int{0, 2} {
		after, err := os.ReadFile(preferenceFiles[i])
		if err != nil || !bytes.Equal(before[i], after) {
			t.Fatalf(":%d preference bytes changed: %v", i, err)
		}
	}

	// Ordinary dependency clones and nested main-repo directories do not create
	// another preference authority merely through discovery/host resolution.
	dependency := filepath.Join(filepath.Dir(paths[1]), "dependency")
	f.git(filepath.Dir(paths[1]), "clone", f.Remote, dependency)
	sub := filepath.Join(paths[1], "sub")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	target, _, err := env.Couch.resolveSlotInput(ctx, sub)
	if err != nil || target == nil || target.Slot.WorktreeRoot != paths[1] {
		t.Fatalf("nested main host %+v %v", target, err)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil || len(snapshot.Records) != 3 {
		t.Fatalf("dependency added records: %+v %v", snapshot, err)
	}
	depIdentity, err := env.Couch.resolveRepoIdentity(ctx, dependency)
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := env.Couch.Threads.GetPathLaunchPreference(depIdentity, dependency); err != nil || found {
		t.Fatalf("dependency preferences found=%v: %v", found, err)
	}
	assertPreferences()
}
