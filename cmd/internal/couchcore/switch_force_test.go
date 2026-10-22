package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type forceSwitchTestGuard struct {
	switchGuardFunc
	confirm func(context.Context, ThreadAddress, string, SwitchSourceEvidence) error
}

func (g forceSwitchTestGuard) PrepareForcedAgentSwitch(context.Context, ThreadAddress, SwitchSourceEvidence) (string, *int, error) {
	n := 3
	return "exact-wrapper", &n, nil
}
func (g forceSwitchTestGuard) ConfirmForcedAgentSwitch(ctx context.Context, a ThreadAddress, s string, e SwitchSourceEvidence) error {
	return g.confirm(ctx, a, s, e)
}

func TestForceSwitchConsentIsLocalBoundAndConsumedOnce(t *testing.T) {
	for _, scenario := range []string{"valid", "serialized", "replay", "changed-argv", "changed-record", "other-owner"} {
		t.Run(scenario, func(t *testing.T) {
			env, source := switchEnvWithLiveThread(t)
			checks, ordinary := 0, 0
			env.Couch.SwitchGuard = forceSwitchTestGuard{switchGuardFunc: func(context.Context, ThreadAddress, string) error {
				ordinary++
				return errors.New("ordinary idle refusal")
			}, confirm: func(_ context.Context, a ThreadAddress, s string, e SwitchSourceEvidence) error {
				checks++
				if a != source.Address || s != "exact-wrapper" || !e.MatchesRecord(source) {
					t.Fatal("force proof changed")
				}
				return context.Canceled
			}}
			argv := []string{}
			p, err := env.Couch.PrepareForcedAgentSwitch(context.Background(), source.Address, "codex", &argv)
			if err != nil {
				t.Fatal(err)
			}
			if p.Force == nil || p.Force.ChangedFiles() == nil || *p.Force.ChangedFiles() != 3 {
				t.Fatal("missing dirty count evidence")
			}
			consent := p.Force.Confirm()
			request := SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: argv, AcceptedFingerprint: p.Fingerprint, Force: consent}
			switch scenario {
			case "serialized":
				raw, err := json.Marshal(consent)
				if err != nil {
					t.Fatal(err)
				}
				var decoded AgentSwitchForceConsent
				if err := json.Unmarshal(raw, &decoded); err != nil {
					t.Fatal(err)
				}
				request.Force = &decoded
			case "changed-argv":
				request.Argv = []string{"--model", "other"}
			case "changed-record":
				name := "changed"
				if _, err := env.Couch.Threads.ApplyThreadMetadata(source.Address, source.Revision, ThreadMetadataPatch{Name: &name}); err != nil {
					t.Fatal(err)
				}
			case "other-owner":
				other, _ := switchEnvWithLiveThread(t)
				env.Couch = other.Couch
			}
			_, err = env.Couch.SwitchAgent(context.Background(), request)
			if err == nil {
				t.Fatal("refused force started target")
			}
			if ordinary != 0 {
				t.Fatal("force ran ordinary idle gate")
			}
			if scenario == "valid" || scenario == "replay" {
				if !errors.Is(err, context.Canceled) || checks != 1 {
					t.Fatalf("force did not reach exact guard once: %v checks%d", err, checks)
				}
			} else if checks != 0 {
				t.Fatal("invalid capability reached effects")
			}
			if scenario == "replay" {
				_, err = env.Couch.SwitchAgent(context.Background(), request)
				if err == nil || !strings.Contains(err.Error(), "consent") {
					t.Fatalf("force replay accepted: %v", err)
				}
				if checks != 1 {
					t.Fatal("replay reached guard")
				}
			}
		})
	}
}

func TestForceSwitchReusesParkCleanupAndPreservesWorktreeFiles(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, cleanup := range []string{"confirmed", "missing-completion", "helper-survives"} {
			t.Run(agent+"/"+cleanup, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, "unsaved.txt")
				contents := []byte("operator changes\x00\n")
				if err := os.WriteFile(path, contents, 0600); err != nil {
					t.Fatal(err)
				}
				base := newTestEnv(t, root)
				r := validThreadRecord(t)
				r.StartingPath = root
				r.WorkingPath = root
				r.Reservation = false
				profile := LaunchProfile{Agent: agent, Argv: []string{}}
				r.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-helper", State: IncarnationLive, LaunchProfile: &profile}}
				r.LatestLaunchProfile = &profile
				source, err := base.Couch.Threads.CreateThread(r)
				if err != nil {
					t.Fatal(err)
				}
				env := withParkLifecycle(t, base, source, 42, "pair-helper")
				env.Couch.reg = env.Couch.reg.Insert(ActorRecord{ID: "source", Thread: source.Address, Args: StartArgs{Worktree: Worktree(root)}, PID: 42, Identity: "pair-helper"})
				env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
				checks := 0
				env.Couch.SwitchGuard = forceSwitchTestGuard{switchGuardFunc: func(context.Context, ThreadAddress, string) error {
					t.Fatal("force used ordinary dirty/idle gate")
					return context.Canceled
				}, confirm: func(context.Context, ThreadAddress, string, SwitchSourceEvidence) error { checks++; return nil }}
				if cleanup != "confirmed" {
					env.Artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
						if cleanup == "helper-survives" {
							completion := successCompletion(env.Lifecycle.lastRequest, env.Now)
							env.Lifecycle.completion = &completion
						}
						return nil
					}
					env.Couch.PairLifecycle.Wait = func(context.Context, time.Duration) error { return context.DeadlineExceeded }
				}
				target := "claude"
				if agent == "claude" {
					target = "codex"
				}
				argv := []string{}
				p, err := env.Couch.PrepareForcedAgentSwitch(context.Background(), source.Address, target, &argv)
				if err != nil {
					t.Fatal(err)
				}
				result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: target, Argv: argv, AcceptedFingerprint: p.Fingerprint, Force: p.Force.Confirm()})
				if cleanup == "confirmed" {
					if err != nil || result.Outcome != SwitchStarted || env.Proc.Exists(42) != Dead {
						t.Fatalf("force failed: %+v %v", result, err)
					}
				} else {
					if err == nil || result.Outcome != SwitchParkIncomplete || len(env.Runner.Ops) != 0 {
						t.Fatalf("unconfirmed cleanup launched target: %+v %v ops%+v", result, err, env.Runner.Ops)
					}
				}
				if checks != 1 {
					t.Fatalf("force guard checks=%d", checks)
				}
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(after, contents) {
					t.Fatalf("on-disk files changed: %q %v", after, err)
				}
			})
		}
	}
}
