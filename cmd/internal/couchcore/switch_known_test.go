package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"strings"
	"testing"
)

func TestSwitchAgentRejectsSameAgent(t *testing.T) {
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	args := []string{}
	_, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "claude", &args)
	if err == nil || !strings.Contains(err.Error(), "different agent") {
		t.Fatalf("same agent admitted: %v", err)
	}
}

func TestKnownSwitchConfiguration(t *testing.T) {
	for _, mode := range []string{"absent", "empty", "resume", "path"} {
		t.Run(mode, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			if mode != "absent" {
				env.Couch.RepoAgentDefault = func(_, agent string) (LaunchProfile, bool, error) {
					argv := []string{}
					if mode == "resume" {
						argv = []string{"resume", "old"}
					}
					return LaunchProfile{Agent: agent, Argv: argv}, true, nil
				}
			}
			if mode == "path" {
				identity, _ := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
				p := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: identity, PhysicalPath: source.StartingPath, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {"--model", "saved"}}, Revision: 1}
				if err := writePathLaunchPreferenceForTest(env.Couch.Threads, p); err != nil {
					t.Fatal(err)
				}
				env.Couch.RepoAgentDefault = func(_, _ string) (LaunchProfile, bool, error) {
					t.Fatal("path configuration must win without default lookup")
					return LaunchProfile{}, false, nil
				}
			}
			prepared, err := env.Couch.PrepareKnownAgentSwitch(context.Background(), source.Address, "codex")
			if mode == "absent" || mode == "resume" {
				if err == nil {
					t.Fatal("unsafe configuration accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if prepared.Profile.Argv == nil {
				t.Fatal("configured empty became absent")
			}
			if mode == "path" && len(prepared.Profile.Argv) != 2 {
				t.Fatal(prepared)
			}
		})
	}
}

func TestKnownSwitchDispatcherRevalidatesConfiguration(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted", true: "changed"}[stale], func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			args := []string{}
			env.Couch.RepoAgentDefault = func(_, agent string) (LaunchProfile, bool, error) {
				return LaunchProfile{Agent: agent, Argv: args}, true, nil
			}
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			env.Runner.AfterAcknowledge = func(id string) error {
				env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
				return nil
			}
			p, err := env.Couch.PrepareKnownAgentSwitch(context.Background(), source.Address, "codex")
			if err != nil {
				t.Fatal(err)
			}
			if stale {
				args = []string{"--model", "changed"}
			}
			value, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{Name: "switch-agent", Context: context.Background(), Implicit: true, Args: map[string]string{"repo-scope": source.Address.RepoScope, "tag": string(source.Address.Tag), "agent": "codex", "known-config": "true", "fingerprint": p.Fingerprint}})
			if stale {
				if err == nil || len(env.Runner.Ops) != 0 {
					t.Fatalf("stale config launched: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if value.(SwitchAgentResult).Outcome != SwitchStarted {
				t.Fatal(value)
			}
		})
	}
}

func TestSwitchAgentFormGuardBeforeEffects(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			env, source := switchEnvWithLiveThread(t)
			args := []string{}
			p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			env.Couch.SwitchGuard = switchGuardFunc(func(_ context.Context, a ThreadAddress, token string) error {
				calls++
				if a != source.Address || token != "" {
					t.Fatal("wrong form identity")
				}
				if calls == failAt {
					return context.Canceled
				}
				return nil
			})
			result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
			after, _ := env.Couch.Threads.GetThread(source.Address)
			if err == nil || result.Outcome != SwitchRefused || calls != failAt || after.Revision != source.Revision {
				t.Fatalf("form guard bypass: %+v %v calls=%d", result, err, calls)
			}
		})
	}
}

func TestResumeRefusesUnconfirmedDifferentAgent(t *testing.T) {
	for _, mode := range []string{"present", "error", "helper-live", "helper-unknown", "absent"} {
		t.Run(mode, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			identity, _ := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
			owner, _ := env.Proc.Current()
			profile := LaunchProfile{Agent: "codex", Argv: []string{}}
			claimed, err := env.Couch.Threads.CommitStartClaim(source.Address, source.Revision, identity, env.Now, StartEvent{Kind: StartClaimed, Nonce: "uncertain-target", Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &profile})
			if err != nil {
				t.Fatal(err)
			}
			claimed, err = env.Couch.Threads.AdvanceStart(source.Address, claimed.Revision, StartEvent{Kind: StartSessionBound, Nonce: "uncertain-target", Binding: &couchidentity.SessionBinding{Name: "pair-target", ScopeKey: source.Address.RepoScope, Tag: string(source.Address.Tag), StartNonce: "uncertain-target", Legacy: true}})
			if err != nil {
				t.Fatal(err)
			}
			recorded, err := env.Couch.Threads.AdvanceStart(source.Address, claimed.Revision, StartEvent{Kind: StartHelperRecorded, Nonce: "uncertain-target", Helper: ProcessIdentity{PID: 1234, Identity: "dead-target-helper"}})
			if err != nil {
				t.Fatal(err)
			}
			unknown, err := env.Couch.Threads.AdvanceStart(source.Address, recorded.Revision, StartEvent{Kind: StartRecoveredUnknown, Nonce: "uncertain-target"})
			if err != nil {
				t.Fatal(err)
			}
			if mode == "helper-live" {
				env.Proc.Set(1234, "dead-target-helper")
			}
			if mode == "helper-unknown" {
				env.Proc.SetUnknown(1234)
			}
			env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingEstablished, "old-native")
			env.Artifacts.SetPairSession(source.Address, recordSessionName(unknown), mode == "present")
			if mode == "error" {
				env.Couch.Artifacts = failedSwitchSessionObserver{env.Artifacts}
			}
			env.Runner.AfterAcknowledge = func(id string) error {
				env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
				return nil
			}
			actor, _, err := env.Couch.ResumeContext(context.Background(), source.Address)
			if mode == "absent" {
				if err != nil || actor.Args.Stack != "claude" {
					t.Fatalf("proved absent target cannot resume old: %+v %v", actor, err)
				}
				return
			}
			if ResumeDiagnosticOf(err) != ResumeUnknown {
				t.Fatalf("unconfirmed target not refused: %v", err)
			}
			after, _ := env.Couch.Threads.GetThread(source.Address)
			if after.Revision != unknown.Revision || len(env.Runner.Ops) != 0 {
				t.Fatal("resume discarded uncertain target evidence")
			}
		})
	}
}

type failedSwitchSessionObserver struct {
	*FakeThreadArtifactCollisionChecker
}

func (f failedSwitchSessionObserver) NamedPairSessionContext(context.Context, ThreadAddress, string) (PairSessionBinding, error) {
	return PairSessionBinding{}, context.DeadlineExceeded
}
