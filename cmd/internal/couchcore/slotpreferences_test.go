package couchcore

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func slotPreferenceRecord(t *testing.T, env *testEnv, local *ThreadStore) ThreadRecord {
	t.Helper()
	r := validThreadRecord(t)
	scope, err := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	r.Address.RepoScope = scope.Key
	r.StartingPath, r.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	r.LatestLaunchProfile = &profile
	r.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-helper", State: IncarnationLive, RepoIdentity: local.slot.RepoIdentity, LaunchProfile: &profile}}
	r, err = local.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	id := ParkIdentity{Nonce: "prefs-park", Address: r.Address, PID: 42, ProcessIdentity: "pair-helper"}
	r, err = local.BeginPark(r.Address, r.Revision, id)
	if err != nil {
		t.Fatal(err)
	}
	r, err = local.FinalizePark(context.Background(), r.Address, r.Revision, id, 1, env.Now)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSlotPreferenceRepositoryArgumentsAcrossFreshAndSwitch(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"fresh", "switch"} {
		t.Run(action, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			r := slotPreferenceRecord(t, env, local)
			env.Couch.RootAgent = "codex"
			if err := local.rememberRepoConfiguration(local.slot.RepoIdentity, LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}, nil); err != nil {
				t.Fatal(err)
			}
			var got LaunchProfile
			if action == "fresh" {
				result, err := env.Couch.StartFreshSlot(context.Background(), r.StartingPath, "")
				if err != nil {
					t.Fatal(err)
				}
				got = LaunchProfile{Agent: result.Record.Args.Stack, Argv: result.Record.Args.ExtraArgs}
			} else {
				result, err := env.Couch.PrepareAgentSwitch(context.Background(), r.Address, "codex", nil)
				if err != nil {
					t.Fatal(err)
				}
				got = result.Profile
			}
			want := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("profile = %+v, want %+v", got, want)
			}
		})
	}
}

func TestSlotFreshInvalidPreferencePreservesCurrent(t *testing.T) {
	t.Parallel()
	for _, poison := range []string{"resume-selector", "unknown-schema"} {
		t.Run(poison, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			r := slotPreferenceRecord(t, env, local)
			env.Couch.RootAgent = "codex"
			if err := local.rememberRepoConfiguration(local.slot.RepoIdentity, LaunchProfile{Agent: "codex", Argv: []string{"resume", "old-native"}}, nil); err != nil {
				t.Fatal(err)
			}
			prefPath := local.repoPreferencesPath()
			if poison == "unknown-schema" {
				if err := os.WriteFile(prefPath, []byte(`{"schema_version":999,"repositories":{}}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			raw, err := os.ReadFile(prefPath)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(filepath.Join(local.root, "thread.json"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = env.Couch.StartFreshSlot(context.Background(), r.StartingPath, "")
			if err == nil || (poison == "resume-selector" && !strings.Contains(err.Error(), "resume")) {
				t.Fatalf("expected invalid canonical preference refusal (%s), got %v", poison, err)
			}
			after, readErr := os.ReadFile(filepath.Join(local.root, "thread.json"))
			if readErr != nil || !bytes.Equal(before, after) {
				t.Fatalf("invalid preferences replaced current record: %v", readErr)
			}
			got, err := os.ReadFile(prefPath)
			if err != nil || !bytes.Equal(raw, got) {
				t.Fatal("preferences changed", err)
			}
			if _, err = os.Stat(local.archivePath(r.Address)); !os.IsNotExist(err) {
				t.Fatalf("invalid settings archived current: %v", err)
			}
			if len(env.Runner.Ops) != 0 {
				t.Fatal("invalid settings launched a child")
			}
		})
	}
}

func TestSlotPreferenceFirstUseKeepsRootAgentAndRepositoryArguments(t *testing.T) {
	t.Parallel()

	env, f := managedStartFixture(t)
	env.Couch.RootAgent = "codex"
	identity, err := env.Couch.resolveRepoIdentity(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Couch.Threads.rememberRepoConfiguration(identity, LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}, nil); err != nil {
		t.Fatal(err)
	}
	args := StartArgs{Cwd: f.Primary, Action: StartCreate}
	prepared, err := env.Couch.PrepareStart(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Resolution.Profile.Agent != "codex" || prepared.Resolution.ArgvSource != ArgvSourceRepoDefault {
		t.Fatalf("first use: %+v", prepared.Resolution)
	}
	if _, err = os.Stat(f.host(1)); !os.IsNotExist(err) {
		t.Fatalf("preview created slot: %v", err)
	}
	record, _, err := env.Couch.SpawnPrepared(context.Background(), args, prepared.Resolution.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if record.Args.Stack != "codex" || !reflect.DeepEqual(record.Args.ExtraArgs, []string{"--sandbox", "workspace-write"}) {
		t.Fatalf("created %+v", record.Args)
	}
}
