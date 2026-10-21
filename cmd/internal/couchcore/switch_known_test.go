package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"reflect"
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
	for _, mode := range []string{"absent", "empty", "resume", "repository"} {
		t.Run(mode, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			if mode != "absent" {
				args := []string{}
				if mode == "resume" {
					args = []string{"resume", "old"}
				}
				if mode == "repository" {
					args = []string{"--model", "saved"}
				}
				seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: "codex", Argv: args})
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
			if mode == "repository" && len(prepared.Profile.Argv) != 2 {
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
			seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: "codex", Argv: args})
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
				seedSwitchRepoArguments(t, env.Couch, source.StartingPath, LaunchProfile{Agent: "codex", Argv: args})
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
			actor, handle, err := env.Couch.ResumeContext(context.Background(), source.Address)
			if mode == "present" {
				if err != nil || handle == nil || actor.Shape != StartWarmReattach {
					t.Fatalf("listed target could not attach: %v", err)
				}
				t.Cleanup(func() { env.Runner.SetExited(handle.ID(), 0) })
				after, _ := env.Couch.Threads.GetThread(source.Address)
				if !reflect.DeepEqual(after, unknown) {
					t.Fatal("attachment modified unresolved replacement")
				}
				return
			}
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

// A replacement profile moves from the pending claim to the incarnation at
// unknown promotion, then becomes the latest successful profile at registration.
// Recovery must preserve whichever representation owns the target at the crash.
func TestInterruptedSwitchResumePreservesTargetAuthority(t *testing.T) {
	for _, phase := range []string{"unbound", "bound", "pending", "reconciled-pending", "unknown", "registered"} {
		for _, presence := range []string{"present", "unknown", "appears", "absent"} {
			if phase == "registered" && presence == "appears" {
				continue
			}
			t.Run(phase+"/"+presence, func(t *testing.T) {
				env := newTestEnv(t, "/repo")
				source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{"--verbose"}})
				source, err := env.Couch.Threads.updateExistingThread(source.Address, source.Revision, func(r *ThreadRecord) error {
					r.SessionBinding = &couchidentity.SessionBinding{Name: "pair-source", ScopeKey: r.Address.RepoScope, Tag: string(r.Address.Tag), StartNonce: "old-start", Legacy: true}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				identity, err := env.Couch.resolveRepoIdentity(context.Background(), source.StartingPath)
				if err != nil {
					t.Fatal(err)
				}
				target := LaunchProfile{Agent: "codex", Argv: []string{"--model", "target-model"}}
				current, err := env.Couch.Threads.CommitStartClaim(source.Address, source.Revision, identity, env.Now, StartEvent{Kind: StartClaimed, Nonce: "interrupted-switch", Owner: SupervisorOwner{PID: 4444, Identity: "dead-supervisor"}, Profile: &target})
				if err != nil {
					t.Fatal(err)
				}
				for _, event := range []StartEvent{
					{Kind: StartSessionBound, Nonce: "interrupted-switch", Binding: &couchidentity.SessionBinding{Name: "pair-target", ScopeKey: source.Address.RepoScope, Tag: string(source.Address.Tag), StartNonce: "interrupted-switch", Legacy: true}},
					{Kind: StartHelperRecorded, Nonce: "interrupted-switch", Helper: ProcessIdentity{PID: 1234, Identity: "dead-helper"}},
				} {
					if (phase == "unbound" && event.Kind == StartSessionBound) || (phase == "bound" && event.Kind == StartHelperRecorded) {
						continue
					}
					current, err = env.Couch.Threads.AdvanceStart(source.Address, current.Revision, event)
					if err != nil {
						t.Fatal(err)
					}
				}
				if phase == "unknown" || phase == "registered" {
					kind := StartRecoveredUnknown
					if phase == "registered" {
						kind = StartRegistered
					}
					current, err = env.Couch.Threads.AdvanceStart(source.Address, current.Revision, StartEvent{Kind: kind, Nonce: "interrupted-switch"})
					if err != nil {
						t.Fatal(err)
					}
				}
				env.Artifacts.SetRegistration(source.Address, RegistrationAbsent, nil)
				env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingEstablished, "old-native")
				env.Artifacts.SetNativeBinding(source.Address, "codex", sessioninventory.BindingEstablished, "target-native")
				env.Artifacts.SetPairSession(source.Address, "pair-target", presence == "present")
				if presence == "present" {
					env.Artifacts.SetDetachedSession(source.Address, "pair-target")
				}
				if presence == "unknown" {
					env.Couch.Artifacts = failedSwitchSessionObserver{env.Artifacts}
				}
				if presence == "appears" && phase == "reconciled-pending" {
					env.Artifacts.SetPairSession(source.Address, "pair-target", true)
				}
				if presence == "appears" {
					env.Couch.Artifacts = &appearingSwitchSessionObserver{FakeThreadArtifactCollisionChecker: env.Artifacts}
				}
				if phase == "reconciled-pending" {
					if err := env.Couch.reconcileInterruptedStarts(); err != nil {
						t.Fatal(err)
					}
					current, err = env.Couch.Threads.GetThread(source.Address)
					if err != nil {
						t.Fatal(err)
					}
					if presence != "absent" && (len(current.Incarnations) != 1 || current.Incarnations[0].Start == nil || recordSessionName(current) != "pair-target") {
						t.Fatal("startup reconciliation discarded pending target", current)
					}
				}
				env.Runner.AfterAcknowledge = func(id string) error {
					env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
					return nil
				}
				if phase != "registered" && (presence != "absent" || phase == "unbound") {
					env.Runner.FailNextStart(errors.New("unexpected replacement spawn"))
				}
				// This matrix protects lifecycle adoption and cold replacement, not
				// the public nonmutating attachment tested above.
				actor, handle, resumeErr := env.Couch.resumeContextWith(context.Background(), source.Address, ResumeOptions{}, true)
				after, err := env.Couch.Threads.GetThread(source.Address)
				if err != nil {
					t.Fatal(err)
				}
				if (presence != "absent" || phase == "unbound") && phase != "registered" {
					if resumeErr == nil || !reflect.DeepEqual(after, current) || len(env.Runner.Ops) != 0 {
						t.Fatalf("resume erased or duplicated target: err=%v revision=%d→%d incarnations=%d→%d session=%q→%q ops=%+v", resumeErr, current.Revision, after.Revision, len(current.Incarnations), len(after.Incarnations), recordSessionName(current), recordSessionName(after), env.Runner.Ops)
					}
					return
				}
				if phase == "registered" && presence == "unknown" {
					if resumeErr == nil || after.LatestLaunchProfile.Agent != "codex" || recordSessionName(after) != "pair-target" || len(env.Runner.Ops) != 0 {
						t.Fatal("registered target authority lost during unknown recovery", resumeErr, after)
					}
					return
				}
				wantAgent, wantNative := "claude", "old-native"
				if phase == "registered" {
					wantAgent, wantNative = "codex", "target-native"
				}
				if resumeErr != nil || actor.Args.Stack != wantAgent {
					t.Fatalf("wrong restored authority: %+v %v", actor, resumeErr)
				}
				if presence == "present" {
					if actor.Shape != StartWarmReattach {
						t.Fatalf("surviving registered target duplicated: %+v", actor)
					}
				} else {
					var launch launcher.TrustedLaunchProfile
					if err := json.Unmarshal([]byte(environmentValue(env.Runner.Child(handle.ID()).Env, launcher.CouchLaunchProfileEnv)), &launch); err != nil {
						t.Fatal(err)
					}
					if launch.Agent != wantAgent || launch.RequiredSessionID != wantNative {
						t.Fatalf("wrong native resume: %+v", launch)
					}
				}
			})
		}
	}
}

// Appear between the two cold replacement gates, after the first exact absent
// observation. The second gate must preserve the newly present target.
type appearingSwitchSessionObserver struct {
	*FakeThreadArtifactCollisionChecker
	calls int
}

func (f *appearingSwitchSessionObserver) NamedPairSessionContext(ctx context.Context, address ThreadAddress, name string) (PairSessionBinding, error) {
	f.calls++
	if f.calls >= 2 {
		f.SetPairSession(address, "pair-target", true)
	}
	return f.FakeThreadArtifactCollisionChecker.NamedPairSessionContext(ctx, address, name)
}
