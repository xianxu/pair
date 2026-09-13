package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

func TestSwitchAgentPreviewAndCommitKeepExistingAddress(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Couch.RepoAgentDefault = func(_, agent string) (LaunchProfile, bool, error) {
		return LaunchProfile{Agent: agent, Argv: []string{"--model", "spark"}}, true, nil
	}
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(prepared.Profile.Argv, []string{"--model", "spark"}) {
		t.Fatal(prepared)
	}
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(source.Address, "pair-switched", true)
		return nil
	}
	result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{
		Address: source.Address, Agent: "codex", Argv: prepared.Profile.Argv, AcceptedFingerprint: prepared.Fingerprint,
	})
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
	if result.Orientation == nil || !strings.Contains(result.Orientation.Body, "claude") {
		t.Fatal(result)
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
	env, source := switchEnvWithLiveThread(t)
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.Threads.UpdateExistingThread(source.Address, source.Revision, func(r *ThreadRecord) error { r.Description = "changed"; return nil })
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
	if !StartFreshExisting.OwnsSession() {
		t.Fatal("fresh switch owns the new session")
	}
	if got := DecideStartCleanup(StartCleanupInput{Shape: StartFreshExisting, HelperDead: true, Presence: PresenceUnobserved}); got != DurableMarkUnknown {
		t.Fatal(got)
	}
	if got := DecideStartCleanup(StartCleanupInput{Shape: StartFreshExisting, HelperDead: true, Presence: PresenceAbsent}); got != DurableRollback {
		t.Fatal(got)
	}
}

func TestUncertainSwitchStartDoesNotChangePathDefaults(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	identity, err := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
	if err != nil {
		t.Fatal(err)
	}
	before, found, err := env.Couch.Threads.GetPathLaunchPreference(identity, source.StartingPath)
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
	after, stillFound, err := env.Couch.Threads.GetPathLaunchPreference(identity, source.StartingPath)
	if err != nil {
		t.Fatal(err)
	}
	if found != stillFound || !reflect.DeepEqual(before, after) {
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
	for _, unknown := range []bool{false, true} {
		env, source := envWithLiveThread(t)
		if unknown {
			source, _ = env.Couch.Threads.UpdateExistingThread(source.Address, source.Revision, func(r *ThreadRecord) error { r.Incarnations[0].State = IncarnationUnknown; return nil })
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

func TestSwitchAgentSourceChangeDuringContextLookupDoesNotParkReplacement(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.SwitchContext = switchContextFunc(func(_ context.Context, r ThreadRecord) (orientation.OrientationContext, error) {
		_, err := env.Couch.Threads.UpdateExistingThread(r.Address, r.Revision, func(next *ThreadRecord) error { next.Description = "concurrent edit"; return nil })
		if err != nil {
			t.Fatal(err)
		}
		return orientation.OrientationContext{Tag: string(r.Address.Tag), WorkingPath: r.WorkingPath, SourceAgent: "claude"}, nil
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
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
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
	if after.LatestLaunchProfile.Agent != "claude" || after.VerifiedPark == nil || !hasOccupiedIncarnation(after) {
		t.Fatal(after)
	}
}

func TestSwitchAgentForkFailureKeepsParkedSource(t *testing.T) {
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
}

func TestSwitchAgentUsesSharedStartingPathPreferenceAndPreservesOtherAgents(t *testing.T) {
	for _, target := range []string{"claude", "codex", "agy", "muse"} {
		t.Run(target, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			identity, err := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
			if err != nil {
				t.Fatal(err)
			}
			previous := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: identity, PhysicalPath: source.StartingPath, LastAgent: "muse", ArgvByAgent: map[string][]string{"claude": {"--model", "sonnet"}, "codex": {"--model", "spark"}, "agy": {"--model", "gemini"}, "muse": {"--model", "muse-default"}}}
			previous.Revision = 1
			if err := writePathLaunchPreferenceForTest(env.Couch.Threads, previous); err != nil {
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
			if result.Orientation.Agent != target {
				t.Fatal(result)
			}
			next, found, err := env.Couch.Threads.GetPathLaunchPreference(identity, source.StartingPath)
			if err != nil || !found || next.LastAgent != target || len(next.ArgvByAgent[target]) != 0 {
				t.Fatal(next, err)
			}
			for other, args := range previous.ArgvByAgent {
				if other != target && !reflect.DeepEqual(next.ArgvByAgent[other], args) {
					t.Fatal("other preference changed", next)
				}
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
			if !profile.FreshRequired || profile.ResumeRequired || profile.Orientation == nil || profile.Orientation.Attempt != result.Orientation.Attempt {
				t.Fatal(profile)
			}
		})
	}
}

func TestSwitchAgentSilentSourceDeathNeedsVerifiedCleanup(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.SwitchContext = switchContextFunc(func(_ context.Context, r ThreadRecord) (orientation.OrientationContext, error) {
		env.Proc.Kill(42)
		env.Artifacts.SetPairSession(r.Address, "pair-exact", false)
		return orientation.OrientationContext{Tag: string(r.Address.Tag), WorkingPath: r.WorkingPath, SourceAgent: "claude"}, nil
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
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Couch.RepoAgentDefault = func(_, agent string) (LaunchProfile, bool, error) {
		return LaunchProfile{Agent: agent, Argv: []string{"resume", "old-session"}}, true, nil
	}
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
