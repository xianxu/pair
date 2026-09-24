package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
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
	r, err = local.FinalizePark(r.Address, r.Revision, id, 1, env.Now)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSlotPreferencePrimaryDefaultsAcrossFreshAndSwitch(t *testing.T) {
	for _, action := range []string{"fresh", "switch"} {
		t.Run(action, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			r := slotPreferenceRecord(t, env, local)
			env.Couch.RootAgent = "codex"
			env.Couch.RepoAgentDefault = func(root, agent string) (LaunchProfile, bool, error) {
				argv := []string{"--wrong-slot-default"}
				if root == local.slot.PrimaryRoot {
					argv = []string{"--sandbox", "workspace-write"}
				}
				return LaunchProfile{Agent: agent, Argv: argv}, true, nil
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
	env, local := slotRecoveryOperationFixture(t)
	r := slotPreferenceRecord(t, env, local)
	pref := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: local.slot.RepoIdentity, PhysicalPath: r.StartingPath, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {"resume", "old-native"}}, Revision: 1}
	raw, err := json.Marshal(pref)
	if err != nil {
		t.Fatal(err)
	}
	prefPath := filepath.Join(local.root, "preferences.json")
	if err = os.WriteFile(prefPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(local.root, "thread.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.StartFreshSlot(context.Background(), r.StartingPath, "")
	if err == nil || !strings.Contains(err.Error(), "resume") {
		t.Fatalf("expected resume-argument refusal, got %v", err)
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
}

func TestSlotPreferenceFirstUseKeepsRootAgentAndPrimaryDefaults(t *testing.T) {
	env, f := managedStartFixture(t)
	env.Couch.RootAgent = "codex"
	env.Couch.RepoAgentDefault = func(root, agent string) (LaunchProfile, bool, error) {
		if root != f.Primary {
			t.Fatalf("first-use default root %q, want %q", root, f.Primary)
		}
		return LaunchProfile{Agent: agent, Argv: []string{"--sandbox", "workspace-write"}}, true, nil
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
