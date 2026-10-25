package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestParseBenchProfile(t *testing.T) {
	for _, c := range []struct {
		name, agent string
		argv        []string
		refusal     string
	}{
		{"explicit argv", "codex", []string{"--model", "x"}, ""},
		{"explicit empty argv", "claude", []string{}, ""},
		{"absent argv", "claude", nil, "explicit"},
		{"no agent", "", []string{}, "agent"},
		{"unknown agent", "nope", []string{}, "agent"},
		{"NUL", "claude", []string{"a\x00b"}, "NUL"},
		{"oversize", "claude", []string{strings.Repeat("x", MaxRepoArgumentBytes+1)}, "exceed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			profile, err := ParseBenchProfile(c.agent, c.argv)
			if c.refusal == "" {
				if err != nil || profile.Agent != c.agent || !reflect.DeepEqual(profile.Argv, c.argv) {
					t.Fatalf("profile %+v err %v", profile, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.refusal) {
				t.Fatalf("want refusal %q, got %v", c.refusal, err)
			}
		})
	}
}

// launchedProfile records the trusted launch profile each child started with.
func launchedProfile(t *testing.T, env *testEnv) *launcher.TrustedLaunchProfile {
	t.Helper()
	var launched launcher.TrustedLaunchProfile
	after := env.Runner.AfterAcknowledge
	env.Runner.AfterAcknowledge = func(id string) error {
		if err := json.Unmarshal([]byte(environmentValue(env.Runner.Child(id).Env, launcher.CouchLaunchProfileEnv)), &launched); err != nil {
			return err
		}
		return after(id)
	}
	return &launched
}

func repoPreferenceBytes(t *testing.T, c *Couch) []byte {
	t.Helper()
	global, err := c.Threads.repoPreferencesBackend()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(global.repoPreferencesPath())
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return raw
}

// pair#847: an explicit bench profile launches exactly that agent and argv on
// Reboot's Fresh path, and is never written back to the per-repository saved
// parameters that production slots of the same repository share.
func TestRebootExplicitProfileLaunchesExactlyAndIsNotRemembered(t *testing.T) {
	env, old := primaryRebootFixture(t)
	launched := launchedProfile(t, env)
	before := repoPreferenceBytes(t, env.Couch)
	argv := []string{"--bench", "one"}
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: old.Address, Agent: "codex", Argv: &argv}); err != nil {
		t.Fatal(err)
	}
	if launched.Agent != "codex" || !reflect.DeepEqual(launched.Argv, argv) || launched.ArgvSource != string(ArgvSourceExplicit) {
		t.Fatalf("launched %+v", launched)
	}
	if after := repoPreferenceBytes(t, env.Couch); !bytes.Equal(before, after) {
		t.Fatalf("saved parameters changed:\n%s\n%s", before, after)
	}
}

func TestRebootExplicitArgvRequiresAgentBeforeAnyStop(t *testing.T) {
	env, old := primaryRebootFixture(t)
	argv := []string{"--bench"}
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: old.Address, Argv: &argv}); err == nil {
		t.Fatal("explicit argv without an agent admitted")
	}
	if len(env.Runner.Ops) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatal("refused profile stopped the source")
	}
}

// A vacant slot fresh-starts through the same recovery with the explicit profile.
func TestVacantSlotFreshStartTakesExplicitProfile(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	registered := map[string]bool{}
	var launched launcher.TrustedLaunchProfile
	after := env.Runner.AfterAcknowledge
	env.Runner.AfterAcknowledge = func(id string) error {
		if err := after(id); err != nil {
			return err
		}
		child := env.Runner.Child(id)
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &launched); err != nil {
			return err
		}
		address := ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(environmentValue(child.Env, "COUCH_THREAD_TAG"))}
		current, err := local.GetThread(address)
		if err != nil {
			return err
		}
		env.Proc.Set(current.Incarnations[0].PID, current.Incarnations[0].Identity)
		var intent launcher.CouchSessionIntent
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchSessionIntentEnv)), &intent); err != nil {
			return err
		}
		env.Artifacts.SetPairSession(address, intent.Name, true)
		env.Artifacts.SetNativeBinding(address, launched.Agent, sessioninventory.BindingEstablished, "bench-native")
		registered[launched.Agent+"/"+launched.FreshNonce] = true
		return nil
	}
	env.Couch.FreshRegistration = func(_ context.Context, address ThreadAddress, agent, nonce string) (bool, error) {
		binding, err := env.Artifacts.PairSession(address)
		return registered[agent+"/"+nonce] && binding.Present, err
	}
	before := repoPreferenceBytes(t, env.Couch)
	argv := []string{"--bench", "vacant"}
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Path: local.slot.WorktreeRoot, Agent: "codex", Argv: &argv}); err != nil {
		t.Fatal(err)
	}
	if launched.Agent != "codex" || !reflect.DeepEqual(launched.Argv, argv) || launched.ArgvSource != string(ArgvSourceExplicit) {
		t.Fatalf("launched %+v", launched)
	}
	if after := repoPreferenceBytes(t, env.Couch); !bytes.Equal(before, after) {
		t.Fatal("saved parameters changed")
	}
}

// BenchStart's provenance is read back from the record the launch committed.
func TestBenchStartCarriesProvenance(t *testing.T) {
	env, old := primaryRebootFixture(t)
	argv := []string{"--bench", "proof"}
	result, err := env.Couch.BenchStart(t.Context(), RebootTarget{Address: old.Address, Agent: "codex", Argv: &argv})
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	p := result.Provenance
	if !ok || p == nil || p.Tag != string(started.Record.Thread.Tag) || p.Root == "" || p.Agent != "codex" || !reflect.DeepEqual(p.Argv, argv) || p.ArgvSource != string(ArgvSourceExplicit) {
		t.Fatalf("provenance %+v started %+v", p, started.Record)
	}
	if result.Archived != old.Address {
		t.Fatalf("embedded reboot result lost: %+v", result.RebootResult)
	}
}

// BR-2: a reboot that only archives (the checkout is gone) starts nothing, so
// bench-start refuses instead of reporting success without a launch.
func TestBenchStartRefusesAnArchiveOnlyReboot(t *testing.T) {
	env, old := primaryRebootFixture(t)
	env.Couch.Path.(*FakePathOps).Fail(old.StartingPath)
	argv := []string{"--bench"}
	result, err := env.Couch.BenchStart(t.Context(), RebootTarget{Address: old.Address, Agent: "codex", Argv: &argv})
	var refusal *SlotOperationError
	if !errors.As(err, &refusal) || refusal.Code != SlotOpBenchNotStarted || result.Provenance != nil {
		t.Fatalf("archive-only bench-start: %+v %v", result, err)
	}
}
