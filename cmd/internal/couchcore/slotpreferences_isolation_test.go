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

// Exercise repository preference publication through real starts and managed
// slots, then reload Couch. Process/session evidence is fake; Git/storage are real.
func TestSlotPreferencesIndependentAcrossRestartResumeAndFresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	f := newProvisionFixture(t)
	env := newTestEnv(t, f.Primary)
	env.Couch.Git, env.Couch.Path = ExecGit{}, OSPathOps{}
	profiles := []LaunchProfile{
		{Agent: "claude", Argv: []string{"--verbose"}},
		{Agent: "codex", Argv: []string{"--no-alt-screen"}},
		{Agent: "muse", Argv: []string{}},
	}
	readPreferences := func() []byte {
		t.Helper()
		raw, err := os.ReadFile(env.Couch.Threads.repoPreferencesPath())
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	assertUnchanged := func(before []byte) {
		t.Helper()
		if !bytes.Equal(before, readPreferences()) {
			t.Fatal("repository preferences changed without adopted configuration")
		}
	}
	actors := make([]ActorRecord, 3)
	paths := []string{f.Primary, f.host(1), f.host(2)}
	for i, profile := range profiles {
		args := StartArgs{Cwd: f.Primary, Stack: profile.Agent, ExtraArgs: cloneArgv(profile.Argv)}
		if i > 0 {
			args.Action = StartCreate
		}
		beforeAdoption := readPreferences()
		actor, handle, err := env.Couch.Spawn(args)
		if err != nil {
			t.Fatalf("start :%d: %v", i, err)
		}
		assertUnchanged(beforeAdoption)
		started := StartResult{Record: actor, Handle: handle}
		env.Couch.PublishAdoptedConfiguration(started)
		if started.Warning() != "" {
			t.Fatal(started.Warning())
		}
		actors[i] = actor
		env.Proc.Set(actor.PID, actor.Identity)
		if i == 0 {
			env.Couch.Slots = NewOSSlotCatalog(f)
			env.Couch.Workspaces = NewWorkspaceProvisioner(f)
		}
	}
	identity := filepath.Join(f.Primary, ".git")
	remembered := map[string][]string{}
	for _, profile := range profiles {
		remembered[profile.Agent] = cloneArgv(profile.Argv)
	}
	assertPreferences := func() {
		t.Helper()
		doc, err := env.Couch.Threads.previewRepoPreferences()
		if err != nil || len(doc.Repositories) != 1 {
			t.Fatalf("repository preferences = %+v, %v", doc, err)
		}
		preference, exists := doc.Repositories[identity]
		if !exists || len(preference.Agents) != len(remembered) {
			t.Fatalf("repository agent inventory = %+v", preference)
		}
		for agent, argv := range remembered {
			got := preference.Agents[agent]
			if !got.Confirmed || got.Conflict || !reflect.DeepEqual(got.Argv, argv) {
				t.Fatalf("%s preference = %+v, want %v", agent, got, argv)
			}
		}
	}
	assertNoFolderPreferences := func() {
		t.Helper()
		entries, err := os.ReadDir(filepath.Join(env.Couch.Threads.root, "path-preferences"))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("new global per-folder preferences: %v", entries)
		}
		for _, path := range paths[1:] {
			backend, err := env.Couch.Threads.storeForPath(path, "", identity)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(backend.pathLaunchPreferencePath(identity, path)); !os.IsNotExist(err) {
				t.Fatalf("new local per-folder preferences: %v", err)
			}
		}
	}
	assertPreferences()
	reopened, err := New(env.Couch.Namespace, env.Runner, OSPathOps{}, ExecGit{}, env.Proc, NewStore(env.Dir), FixedClock{T: env.Now}, NewFixedIDGen("c3d4", "e5f6"), newIncrementingEntropy(), env.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	reopened.Identities = env.Couch.Identities
	reopened.ProcessTable = env.Couch.ProcessTable
	env.Couch = reopened
	env.Couch.Slots, env.Couch.Workspaces = NewOSSlotCatalog(f), NewWorkspaceProvisioner(f)
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	assertPreferences()

	// Persist the normal park transaction, with the fake supplying child death
	// and an established native conversation; resume uses the public launch API.
	resumeSequence := 0
	resumeHistorical := func(i int) {
		t.Helper()
		actor := actors[i]
		resumeSequence++
		beforeResume := readPreferences()
		record, err := env.Couch.Threads.GetThread(actor.Thread)
		if err != nil {
			t.Fatal(err)
		}
		park := ParkIdentity{Nonce: fmt.Sprintf("preferences-park-%d-%d", i, resumeSequence), Address: actor.Thread, PID: actor.PID, ProcessIdentity: actor.Identity}
		begun, err := env.Couch.Threads.BeginPark(actor.Thread, record.Revision, park)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := env.Couch.Threads.FinalizePark(context.Background(), actor.Thread, begun.Revision, park, 1, env.Now); err != nil {
			t.Fatal(err)
		}
		env.Proc.Kill(actor.PID)
		if err := env.Couch.Forget(actor.Args.Worktree, actor.ID); err != nil {
			t.Fatal(err)
		}
		env.Artifacts.SetPairSession(actor.Thread, record.SessionBinding.Name, false)
		env.Artifacts.SetNativeBinding(actor.Thread, profiles[i].Agent, sessioninventory.BindingEstablished, fmt.Sprintf("native-preferences-%d", i))
		env.Runner.AfterAcknowledge = func(id string) error {
			env.Artifacts.SetPairSession(actor.Thread, continuationChildSession(t, env.Runner, id), true)
			return nil
		}
		resumed, handle, err := env.Couch.Resume(actor.Thread)
		if err != nil {
			t.Fatalf("resume :%d: %v", i, err)
		}
		if resumed.Args.Stack != profiles[i].Agent || !reflect.DeepEqual(resumed.Args.ExtraArgs, profiles[i].Argv) {
			t.Fatalf(":%d resumed %+v", i, resumed.Args)
		}
		started, _ := (StartResult{Record: resumed, Handle: handle}).Started()
		if started.Configuration != nil {
			t.Fatal("automatic resume invented configuration acceptance")
		}
		env.Couch.PublishAdoptedConfiguration(started)
		assertUnchanged(beforeResume)
		actors[i] = resumed
	}
	for i := range actors {
		resumeHistorical(i)
	}
	assertPreferences()
	env.Runner.AfterAcknowledge = nil
	// Switching :1 changes the shared Claude configuration only after adoption.
	beforeSwitch := readPreferences()
	source := actors[1]
	env.Proc.Kill(source.PID)
	if err := env.Couch.Forget(source.Args.Worktree, source.ID); err != nil {
		t.Fatal(err)
	}
	current, err := env.Couch.Threads.GetThread(source.Thread)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(source.Thread, current.SessionBinding.Name, false)
	env.Artifacts.SetSessionPresence(source.Thread, SessionObservation{State: SessionAbsent})
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(source.Thread, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	edited := []string{"--verbose", "--debug"}
	preview, err := env.Couch.PrepareAgentSwitch(ctx, source.Thread, "claude", &edited)
	if err != nil {
		t.Fatal(err)
	}
	switched, err := env.Couch.SwitchAgent(ctx, SwitchAgentRequest{Address: source.Thread, Agent: "claude", Argv: edited, AcceptedFingerprint: preview.Fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	assertUnchanged(beforeSwitch)
	startedSwitch, ok := switched.Started()
	if !ok {
		t.Fatal("successful switch did not expose its adopted child")
	}
	env.Couch.PublishAdoptedConfiguration(startedSwitch)
	if switched.Warning() != "" {
		t.Fatal(switched.Warning())
	}
	remembered["claude"] = cloneArgv(edited)
	assertPreferences()
	otherSlot, err := env.Couch.Threads.storeForPath(paths[2], "", identity)
	if err != nil || otherSlot.slot == nil {
		t.Fatalf("other slot backend = %v, %v", otherSlot, err)
	}
	shared, err := env.Couch.slotLaunchProfile(ctx, otherSlot, *otherSlot.slot, paths[2], "claude")
	if err != nil || !reflect.DeepEqual(shared.Profile.Argv, edited) {
		t.Fatalf("other slot missed shared Claude edit: %+v, %v", shared, err)
	}
	// The primary's original Claude conversation keeps its exact historical
	// flags even though another slot has changed that agent's repository value.
	resumeHistorical(0)
	assertPreferences()
	env.Runner.AfterAcknowledge = nil
	env.Proc.Kill(switched.Record.PID)
	current, err = env.Couch.Threads.GetThread(source.Thread)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(source.Thread, current.SessionBinding.Name, false)
	env.Artifacts.SetSessionPresence(source.Thread, SessionObservation{State: SessionAbsent})
	env.Runner.AfterAcknowledge = nil
	beforeFresh := readPreferences()
	fresh, err := env.Couch.StartFreshSlot(ctx, paths[1], "claude")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Record.Thread == source.Thread || fresh.Record.Args.Stack != "claude" || !reflect.DeepEqual(fresh.Record.Args.ExtraArgs, edited) {
		t.Fatalf("fresh %+v", fresh.Record)
	}
	assertUnchanged(beforeFresh)
	env.Couch.PublishAdoptedConfiguration(fresh)
	if fresh.Warning() != "" {
		t.Fatal(fresh.Warning())
	}
	assertPreferences()
	assertNoFolderPreferences()

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
	doc, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if _, found := doc.Repositories[depIdentity]; found {
		t.Fatal("dependency discovery created a preference authority")
	}
	assertNoFolderPreferences()
	assertPreferences()
}
