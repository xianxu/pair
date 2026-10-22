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
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestSwitchAgentPreviewAndCommitKeepExistingAddress(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: "codex", Argv: []string{"--model", "spark"}})
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(prepared.Profile.Argv, []string{"--model", "spark"}) {
		t.Fatal(prepared)
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: prepared.Profile.Argv, AcceptedFingerprint: prepared.Fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread != source.Address {
		t.Fatal(result)
	}
	after, err := env.Couch.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if after.LatestLaunchProfile.Agent != "codex" || after.WorkingPath != source.WorkingPath || after.StartingPath != source.StartingPath {
		t.Fatal(after)
	}
	var profile launcher.TrustedLaunchProfile
	for _, entry := range env.Runner.Child(started.Handle.ID()).Env {
		if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
			if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !profile.FreshRequired || profile.ResumeRequired {
		t.Fatal(profile)
	}
}

func TestSwitchAgentStalePreviewParksNothing(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.Threads.updateExistingThread(source.Address, source.Revision, func(r *ThreadRecord) error { r.Description = "changed"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if !errors.Is(err, ErrSwitchResolutionChanged) {
		t.Fatal(err)
	}
	after, _ := env.Couch.Threads.GetThread(source.Address)
	if after.Park != nil || len(after.Incarnations) != 1 || after.Incarnations[0].State != IncarnationLive {
		t.Fatal(after)
	}
}

func TestSwitchAgentInvalidTargetArgumentsParksNothing(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	args := []string{"resume", "previous-native"}
	if _, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args); err == nil {
		t.Fatal("accepted native resume")
	}
	after, _ := env.Couch.Threads.GetThread(source.Address)
	if after.Park != nil || len(env.Runner.Ops) != 0 {
		t.Fatal("refusal mutated the source")
	}
}

func TestFreshExistingCleanupRequiresProvedAbsence(t *testing.T) {
	t.Parallel()

	if !StartFreshExisting.OwnsSession() {
		t.Fatal("fresh switch owns the new session")
	}
	if got := DecideStartCleanup(StartCleanupInput{Shape: StartFreshExisting, HelperDead: true, Presence: PresenceUnobserved, Cleanup: CleanupConfirmed}); got != DurableMarkUnknown {
		t.Fatal(got)
	}
	if got := DecideStartCleanup(StartCleanupInput{Shape: StartFreshExisting, HelperDead: true, Presence: PresenceAbsent, Cleanup: CleanupConfirmed}); got != DurableRollback {
		t.Fatal(got)
	}
}

func TestUncertainSwitchStartDoesNotChangeRepositoryDefaults(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	identity, err := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
	if err != nil {
		t.Fatal(err)
	}
	seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: "claude", Argv: []string{"--verbose"}})
	before, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := env.Proc.Current()
	profile := LaunchProfile{Agent: "codex", Argv: []string{}}
	claimed, err := env.Couch.Threads.CommitStartClaim(source.Address, source.Revision, identity, env.Now, StartEvent{Kind: StartClaimed, Nonce: "switch-uncertain", Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &profile})
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := env.Couch.Threads.AdvanceStart(source.Address, claimed.Revision, StartEvent{Kind: StartHelperRecorded, Nonce: "switch-uncertain", Helper: ProcessIdentity{PID: 1234, Identity: "target"}})
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := env.Couch.Threads.AdvanceStart(source.Address, recorded.Revision, StartEvent{Kind: StartRecoveredUnknown, Nonce: "switch-uncertain"})
	if err != nil {
		t.Fatal(err)
	}
	after, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("uncertain target changed defaults")
	}
	if unknown.LatestLaunchProfile.Agent != "claude" || unknown.VerifiedPark == nil || !hasOccupiedIncarnation(unknown) {
		t.Fatal(unknown)
	}
}

func switchEnvWithLiveThread(t *testing.T) (*relaunchEnv, ThreadRecord) {
	env, source := envWithLiveThread(t)
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	env.Couch.reg = env.Couch.reg.Insert(ActorRecord{ID: "source-actor", Thread: source.Address, Args: StartArgs{Worktree: Worktree(source.StartingPath)}, PID: 42, Identity: "pair-helper"})
	return env, source
}

func TestSwitchAgentRefusesForeignOrUnknownSource(t *testing.T) {
	t.Parallel()

	for _, unknown := range []bool{false, true} {
		env, source := envWithLiveThread(t)
		if unknown {
			source, _ = env.Couch.Threads.updateExistingThread(source.Address, source.Revision, func(r *ThreadRecord) error { r.Incarnations[0].State = IncarnationUnknown; return nil })
		}
		args := []string{}
		if _, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args); err == nil {
			t.Fatal("accepted unowned or unknown source")
		}
	}
}

type switchContextFunc func(context.Context, ThreadRecord) (orientation.OrientationContext, error)

func (f switchContextFunc) Resolve(ctx context.Context, r ThreadRecord) (orientation.OrientationContext, error) {
	return f(ctx, r)
}

func TestSwitchAgentSourceChangeDuringGuardDoesNotParkReplacement(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.SwitchGuard = switchGuardFunc(func(_ context.Context, _ ThreadAddress, _ string) error {
		r, _ := env.Couch.Threads.GetThread(source.Address)
		_, err := env.Couch.Threads.updateExistingThread(r.Address, r.Revision, func(next *ThreadRecord) error { next.Description = "concurrent edit"; return nil })
		if err != nil {
			t.Fatal(err)
		}
		return nil
	})
	_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err == nil {
		t.Fatal("accepted changed source")
	}
	after, _ := env.Couch.Threads.GetThread(source.Address)
	if after.Park != nil || env.Proc.Exists(42) != Live {
		t.Fatal("changed source parked")
	}
}

func TestSwitchAgentFailedRegistrationPreservesForeignSessionAndDefaults(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: "claude", Argv: []string{"--verbose"}})
	beforePreferences, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
		return false, errors.New("different launch nonce")
	}
	env.Runner.AfterAcknowledge = func(string) error { env.Artifacts.SetPairSession(source.Address, "foreign-session", true); return nil }
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err == nil || result.Outcome != SwitchStartFailed {
		t.Fatal(result, err)
	}
	binding, _ := env.Artifacts.PairSession(source.Address)
	if !binding.Present {
		t.Fatal("killed foreign session")
	}
	after, _ := env.Couch.Threads.GetThread(source.Address)
	// The foreign indexed terminal is not the proposed target. Exact absence
	// of the proposed terminal and a dead helper allow rollback without touching it.
	if after.LatestLaunchProfile.Agent != "claude" || after.VerifiedPark == nil || hasOccupiedIncarnation(after) {
		t.Fatal(after)
	}
	afterPreferences, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforePreferences, afterPreferences) {
		t.Fatal("failed registration changed repository preferences")
	}
}

func TestSwitchAgentForkFailureKeepsParkedSource(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return false, nil }
	env.Runner.FailNextStart(errors.New("fork failed"))
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err == nil {
		t.Fatal("expected fork error")
	}
	assertVerifiedParkRestored(t, env.Couch.Threads, source)
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingEstablished, "old-native")
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	resumed, _, err := env.Couch.Resume(source.Address)
	if err != nil || resumed.Args.Stack != "claude" {
		t.Fatalf("failed switch lost old resume: %+v %v", resumed, err)
	}
}

func TestSwitchAgentUsesRepositoryPreferencesAndPreservesOtherAgents(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"claude", "codex", "agy", "muse"} {
		t.Run(target, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			sourceAgent := "claude"
			if target == sourceAgent {
				sourceAgent = "codex"
			}
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: sourceAgent, Argv: []string{}})
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			env.Runner.AfterAcknowledge = func(id string) error {
				env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
				return nil
			}
			identity, err := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
			if err != nil {
				t.Fatal(err)
			}
			previous := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: identity, PhysicalPath: source.StartingPath, LastAgent: "muse", ArgvByAgent: map[string][]string{"claude": {"--model", "sonnet"}, "codex": {"--model", "spark"}, "agy": {"--model", "gemini"}, "muse": {"--model", "muse-default"}}}
			previous.Revision = 1
			if err := writePathLaunchPreferenceForTest(env.Couch.Threads, previous); err != nil {
				t.Fatal(err)
			}
			legacyPath := env.Couch.Threads.pathLaunchPreferencePath(identity, source.StartingPath)
			legacyBefore, err := os.ReadFile(legacyPath)
			if err != nil {
				t.Fatal(err)
			}
			for agent, args := range previous.ArgvByAgent {
				seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: agent, Argv: args})
			}
			before, err := env.Couch.Threads.previewRepoPreferences()
			if err != nil {
				t.Fatal(err)
			}
			p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, target, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(p.Profile.Argv, previous.ArgvByAgent[target]) {
				t.Fatal(p.Profile)
			}
			// Explicit empty selection wins over remembered nonempty parameters.
			empty := []string{}
			p, err = env.Couch.PrepareAgentSwitch(context.Background(), source.Address, target, &empty)
			if err != nil {
				t.Fatal(err)
			}
			result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: target, Argv: empty, AcceptedFingerprint: p.Fingerprint})
			if err != nil {
				t.Fatal(err)
			}
			notAdopted, err := env.Couch.Threads.previewRepoPreferences()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, notAdopted) {
				t.Fatal("registered child changed repository preferences before adoption")
			}
			started, ok := result.Started()
			if !ok {
				t.Fatal("successful switch omitted adopted child")
			}
			if started.Configuration == nil {
				t.Fatal("successful explicit switch lost accepted configuration before adoption")
			}
			env.Couch.PublishAdoptedConfiguration(started)
			if warning := started.Warning(); warning != "" {
				t.Fatal("adoption did not remember accepted parameters", warning)
			}
			next, err := env.Couch.Threads.previewRepoPreferences()
			if err != nil {
				t.Fatal(err)
			}
			for agent, args := range previous.ArgvByAgent {
				want := args
				if agent == target {
					want = []string{}
				}
				resolved, err := ResolveRepoArguments(repoRecordPointer(next, identity), agent, nil)
				if err != nil || !resolved.Found || !reflect.DeepEqual(resolved.Argv, want) {
					t.Fatalf("agent%s args%+v want%v err%v", agent, resolved, want, err)
				}
			}
			legacyAfter, err := os.ReadFile(legacyPath)
			if err != nil || !bytes.Equal(legacyBefore, legacyAfter) {
				t.Fatal("legacy preference changed", err)
			}
			_, workingFound, err := env.Couch.Threads.GetPathLaunchPreference(identity, source.WorkingPath)
			if err != nil || workingFound {
				t.Fatal("created a separate working-path preference", err)
			}
			var profile launcher.TrustedLaunchProfile
			for _, entry := range env.Runner.Child(result.Handle.ID()).Env {
				if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
					if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
						t.Fatal(err)
					}
				}
			}
			if !profile.FreshRequired || profile.ResumeRequired || profile.Prompt != "" || profile.FreshNonce == "" {
				t.Fatal(profile)
			}
		})
	}
}

func TestSwitchAgentSilentSourceDeathNeedsVerifiedCleanup(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	env.Couch.SwitchGuard = switchGuardFunc(func(_ context.Context, _ ThreadAddress, _ string) error {
		calls++
		if calls == 1 {
			return nil
		}
		env.Proc.Kill(42)
		env.Artifacts.SetPairSession(source.Address, "pair-exact", false)
		return nil
	})
	env.Couch.PairLifecycle.CompletionTimeout = 0
	result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err == nil || result.Outcome != SwitchParkIncomplete || len(env.Runner.Ops) != 0 {
		t.Fatal(result, err)
	}
	after, _ := env.Couch.Threads.GetThread(source.Address)
	if after.VerifiedPark != nil || after.Park == nil {
		t.Fatal("death was treated as completed cleanup", after)
	}
}

func TestSwitchAgentPrefillAllowsEditingRememberedResumeParameters(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: "codex", Argv: []string{"resume", "old-session"}})
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", nil)
	if err != nil {
		t.Fatal("remembered parameters must be editable", err)
	}
	if !reflect.DeepEqual(p.Profile.Argv, []string{"resume", "old-session"}) {
		t.Fatal(p)
	}
	if _, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &p.Profile.Argv); err == nil {
		t.Fatal("accepted resume parameters for fresh launch")
	}
	empty := []string{}
	if _, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &empty); err != nil {
		t.Fatal(err)
	}
}

type switchGuardFunc func(context.Context, ThreadAddress, string) error

func (f switchGuardFunc) ConfirmAgentSwitch(ctx context.Context, a ThreadAddress, source string, _ SwitchSourceEvidence) error {
	return f(ctx, a, source)
}

func seedSwitchRepoArguments(t *testing.T, c *Couch, path string, profile LaunchProfile) string {
	t.Helper()
	identity, err := c.resolveRepoIdentity(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Threads.rememberRepoConfiguration(identity, profile, nil); err != nil {
		t.Fatal(err)
	}
	return identity
}
