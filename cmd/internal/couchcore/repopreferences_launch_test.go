package couchcore

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryArgumentsExplicitProfileSurvivesUnreadableSettings(t *testing.T) {
	env := newTestEnv(t, "/repo")
	if err := env.Couch.Threads.rememberRepoConfiguration("fake-repo", LaunchProfile{Agent: "codex", Argv: []string{"old"}}, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.Couch.Threads.repoPreferencesPath(), []byte(`{"schema_version":999}`), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--model", "accepted"}
	resolved, err := env.Couch.resolveOrdinaryStartResolution(context.Background(), StartArgs{Cwd: "/repo", Stack: "codex", ExtraArgs: args})
	if err != nil || !reflect.DeepEqual(resolved.Profile.Argv, args) {
		t.Fatalf("explicit profile depends on unreadable settings: %+v %v", resolved, err)
	}
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	switched, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil || !reflect.DeepEqual(switched.Profile.Argv, args) {
		t.Fatalf("explicit switch depends on unreadable settings: %+v %v", switched, err)
	}
	if _, err := env.Couch.PrepareKnownAgentSwitch(context.Background(), source.Address, "codex"); err == nil {
		t.Fatal("unreadable settings silently treated as saved")
	}
}

func TestRepositoryArgumentsNewThreadRejectsNativeSelectorsBeforeClaim(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, explicit := range []bool{false, true} {
			t.Run(agent+map[bool]string{false: "/legacy", true: "/explicit"}[explicit], func(t *testing.T) {
				f := newProvisionFixture(t)
				env := newTestEnv(t, f.Primary)
				env.Couch.Git, env.Couch.Path = ExecGit{}, OSPathOps{}
				argv := []string{"resume", "old-native"}
				if agent == "claude" {
					argv = []string{"--resume", "old-native"}
				}
				args := StartArgs{Worktree: Worktree(f.Primary), Stack: agent}
				if explicit {
					args.ExtraArgs = argv
				} else {
					if err := env.Couch.Threads.RecoverStoreJournal(); err != nil {
						t.Fatal(err)
					}
					id, err := env.Couch.resolveRepoIdentity(context.Background(), f.Primary)
					if err != nil {
						t.Fatal(err)
					}
					legacy := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: id, PhysicalPath: f.Primary, LastAgent: agent, ArgvByAgent: map[string][]string{agent: argv}, Revision: 1}
					if err := writePathLaunchPreferenceForTest(env.Couch.Threads, legacy); err != nil {
						t.Fatal(err)
					}
				}
				if _, _, err := env.Couch.Spawn(args); err == nil {
					t.Fatal("new thread implicitly resumed native context from startup arguments")
				} else if !strings.Contains(err.Error(), "start: invalid startup parameters") {
					t.Fatalf("refused for an unrelated reason: %v", err)
				}
				if len(env.Runner.Ops) != 0 {
					t.Fatalf("invalid arguments launched: %v", env.Runner.Ops)
				}
				snapshot, err := env.Couch.Threads.Snapshot()
				if err != nil || len(snapshot.Records) != 0 {
					t.Fatalf("invalid arguments claimed a thread: %+v %v", snapshot, err)
				}
			})
		}
	}
}

func TestRepositoryArgumentsInvalidCreateDoesNotProvisionSlot(t *testing.T) {
	f := newProvisionFixture(t)
	env := newTestEnv(t, f.Primary)
	env.Couch.Git, env.Couch.Path = ExecGit{}, OSPathOps{}
	env.Couch.Slots, env.Couch.Workspaces = NewOSSlotCatalog(f), NewWorkspaceProvisioner(f)
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: f.Primary, Action: StartCreate, Stack: "codex", ExtraArgs: []string{"resume", "old-native"}}); err == nil {
		t.Fatal("create accepted native context selector")
	}
	if _, err := os.Lstat(f.host(1)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid startup arguments provisioned a slot: %v", err)
	}
}

func TestRepositoryArgumentsResolveOrdinaryStartWithoutPairDefaults(t *testing.T) {
	f := newProvisionFixture(t)
	env := newTestEnv(t, f.Primary)
	env.Couch.Git, env.Couch.Path = ExecGit{}, OSPathOps{}
	id, err := env.Couch.resolveRepoIdentity(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	want := LaunchProfile{Agent: "codex", Argv: []string{"--model", "repo-model"}}
	if err := env.Couch.Threads.rememberRepoConfiguration(id, want, nil); err != nil {
		t.Fatal(err)
	}

	got, err := env.Couch.resolveOrdinaryStartResolution(context.Background(), StartArgs{Cwd: f.Primary, Stack: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Profile, want) || got.ArgvSource != ArgvSourceRepository {
		t.Fatalf("%+v", got)
	}
	explicit := []string{}
	got, err = env.Couch.resolveOrdinaryStartResolution(context.Background(), StartArgs{Cwd: f.Primary, Stack: "codex", ExtraArgs: explicit})
	if err != nil || got.Profile.Argv == nil || len(got.Profile.Argv) != 0 || got.ArgvSource != ArgvSourceExplicit {
		t.Fatalf("explicit empty: %+v %v", got, err)
	}
}

func TestRepositoryArgumentsSwitchOffersBorrowedWithoutImplicitAcceptance(t *testing.T) {
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})

	if err := env.Couch.Threads.rememberRepoConfiguration("/other/.git", LaunchProfile{Agent: "codex", Argv: []string{"--model", "borrowed"}}, nil); err != nil {
		t.Fatal(err)
	}
	preview, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Borrowed == nil || preview.Borrowed.RepoIdentity != "/other/.git" || preview.Configured || !reflect.DeepEqual(preview.Profile.Argv, []string{"--model", "borrowed"}) {
		t.Fatal(preview)
	}
	if _, err := env.Couch.PrepareKnownAgentSwitch(context.Background(), source.Address, "codex"); err == nil {
		t.Fatal("CLI implicitly accepted borrowed settings")
	}
	id, err := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Couch.Threads.rememberRepoConfiguration(id, LaunchProfile{Agent: "codex", Argv: []string{}}, nil); err != nil {
		t.Fatal(err)
	}
	own, err := env.Couch.PrepareKnownAgentSwitch(context.Background(), source.Address, "codex")
	if err != nil || !own.Configured || own.Borrowed != nil || own.Profile.Argv == nil || len(own.Profile.Argv) != 0 {
		t.Fatalf("%+v %v", own, err)
	}
}
