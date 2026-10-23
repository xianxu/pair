package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestSwitchAgentEditedRestartResumesExactNativeOrDisclosesFresh(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, missing := range []bool{false, true} {
			t.Run(agent+map[bool]string{false: "/found", true: "/missing"}[missing], func(t *testing.T) {
				env := newTestEnv(t, "/repo")
				env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
				source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{}})
				a := bindID(3)
				env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, a)
				argv := []string{"--model", "edited"}
				p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, agent, &argv)
				if err != nil {
					t.Fatal(err)
				}
				accepted := p.AcceptAgentSwitch(false)
				if missing {
					env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingUnbound, "")
				}
				env.Runner.AfterAcknowledge = func(id string) error {
					env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
					env.Artifacts.SetPaneSidecar(source.Address, agent)
					return nil
				}
				got, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Restart: accepted.restart, Address: source.Address, Agent: agent, Argv: argv, AcceptedFingerprint: p.Fingerprint})
				if err != nil || got.Outcome != SwitchStarted {
					t.Fatalf("edited restart: %+v %v", got, err)
				}
				want := StartColdResume
				if missing {
					want = StartFreshExisting
				}
				if got.Record.Shape != want {
					t.Fatalf("shape=%q want%q", got.Record.Shape, want)
				}
				if strings.Contains(got.Warning(), EditedRestartFallbackNotice) != missing {
					t.Fatalf("fallback notice %q missing=%t", got.Warning(), missing)
				}
				if starts := countStarts(env.Runner); starts != 1 {
					t.Fatalf("starts=%d", starts)
				}
				var profile launcher.TrustedLaunchProfile
				for _, entry := range env.Runner.Child(got.Handle.ID()).Env {
					if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
						if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
							t.Fatal(err)
						}
					}
				}
				if profile.Agent != agent || !reflect.DeepEqual(profile.Argv, argv) || profile.FreshRequired != missing || profile.ResumeRequired == missing || (!missing && profile.RequiredSessionID != a) {
					t.Fatal("lost accepted profile/native anchor", profile)
				}
			})
		}
	}
}

func TestSwitchAgentEditedRestartFailedAttemptRequiresPositiveAbsenceAndCleanup(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, scenario := range []string{"missing", "still-found", "replacement", "unknown", "cancelled", "cleanup-unknown"} {
			t.Run(agent+"/"+scenario, func(t *testing.T) {
				env := newTestEnv(t, "/repo")
				env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
				source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{}})
				env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, bindID(3))
				argv := []string{"--model", "edited"}
				p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, agent, &argv)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cause := errors.New("native resume failed")
				attempts := 0
				first := ""
				env.Runner.BeforeAcknowledge = func(id string) error {
					attempts++
					if attempts > 1 {
						if env.Runner.Child(first).alive {
							t.Fatal("fresh child preceded old helper cleanup")
						}
						return nil
					}
					first = id
					switch scenario {
					case "missing", "cancelled", "cleanup-unknown":
						env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingUnbound, "")
					case "replacement":
						env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, bindID(5))
					case "unknown":
						env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingProvisional, "")
					}
					if scenario == "cancelled" {
						cancel()
					}
					if scenario == "cleanup-unknown" {
						env.Artifacts.QuiesceNamedErr = errors.New("session ownership unknown")
					}
					return cause
				}
				env.Runner.AfterAcknowledge = func(id string) error {
					env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
					return nil
				}
				got, err := env.Couch.SwitchAgent(ctx, SwitchAgentRequest{Restart: p.AcceptAgentSwitch(false).restart, Address: source.Address, Agent: agent, Argv: argv, AcceptedFingerprint: p.Fingerprint})
				if scenario == "missing" {
					if err != nil || got.Outcome != SwitchStarted || got.Record.Shape != StartFreshExisting || !strings.Contains(got.Warning(), EditedRestartFallbackNotice) || countStarts(env.Runner) != 2 {
						t.Fatalf("missing did not yield one clean fallback: %+v starts=%d err=%v", got, countStarts(env.Runner), err)
					}
				} else if err == nil || got.Outcome == SwitchStarted || countStarts(env.Runner) != 1 || strings.Contains(got.Warning(), EditedRestartFallbackNotice) {
					t.Fatalf("uncertainty yielded fallback: %+v starts=%d err=%v", got, countStarts(env.Runner), err)
				}
			})
		}
	}
}

type editedRestartFallbackClaimArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	failed        bool
	fallbackNonce string
	firstNonce    string
}

func (f *editedRestartFallbackClaimArtifacts) BindTranscript(_ context.Context, thread ThreadRecord) (TranscriptBind, error) {
	for _, incarnation := range thread.Incarnations {
		if incarnation.Start != nil {
			if f.firstNonce == "" {
				f.firstNonce = incarnation.Start.Nonce
			}
			if incarnation.Start.Nonce == f.firstNonce {
				continue
			}
			f.fallbackNonce = incarnation.Start.Nonce
			return TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: bindID(5)}}, nil
		}
	}
	if f.failed {
		return TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingUnbound}}, refuseBinding(ResumeBindingUnbound)
	}
	return TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: bindID(3)}}, nil
}

func TestSwitchAgentEditedRestartFailureNamesActiveFallbackAttempt(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{}})
			env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, bindID(3))
			artifacts := &editedRestartFallbackClaimArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts}
			env.Couch.Artifacts = artifacts
			argv := []string{}
			p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, agent, &argv)
			if err != nil {
				t.Fatal(err)
			}
			env.Runner.BeforeAcknowledge = func(string) error {
				artifacts.failed = true
				return errors.New("first resume failed")
			}
			_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Restart: p.AcceptAgentSwitch(false).restart, Address: source.Address, Agent: agent, Argv: argv, AcceptedFingerprint: p.Fingerprint})
			var failure *launchFailureError
			if artifacts.fallbackNonce == "" || !errors.As(err, &failure) {
				t.Fatalf("missing fallback refusal: %v", err)
			}
			if failure.failure.Nonce != artifacts.fallbackNonce {
				t.Fatalf("failure names %q, active fallback is %q", failure.failure.Nonce, artifacts.fallbackNonce)
			}
			if countStarts(env.Runner) != 1 {
				t.Fatal("refused fresh claim launched child")
			}
		})
	}
}
