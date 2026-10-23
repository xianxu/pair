package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type cooperativeSwitchTestGuard struct {
	switchGuardFunc
	interrupt func(context.Context, ThreadAddress, string, SwitchSourceEvidence) error
}

func (g cooperativeSwitchTestGuard) InterruptBeforeAgentSwitch(ctx context.Context, address ThreadAddress, source string, evidence SwitchSourceEvidence) (string, error) {
	return source, g.interrupt(ctx, address, source, evidence)
}

func TestSwitchAgentCooperativePhaseRunsOnceUnderLifecycleHold(t *testing.T) {
	for _, scenario := range []string{"refused", "changed", "fresh-idle-guard", "held", "missing-guard", "dispatch"} {
		t.Run(scenario, func(t *testing.T) {
			env, source := switchEnvWithLiveThread(t)
			argv := []string{}
			prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &argv)
			if err != nil {
				t.Fatal(err)
			}
			phases, checks := 0, 0
			env.Couch.SwitchGuard = cooperativeSwitchTestGuard{
				switchGuardFunc: func(context.Context, ThreadAddress, string) error { checks++; return context.Canceled },
				interrupt: func(ctx context.Context, address ThreadAddress, token string, evidence SwitchSourceEvidence) error {
					phases++
					if address != source.Address || token != "" || evidence.Vacant() || !evidence.MatchesRecord(source) {
						t.Fatal("cooperative phase lost exact live source")
					}
					_, release, err := env.Couch.hold(context.Background(), address, "competing-operation")
					if err == nil {
						release()
						t.Fatal("interrupt ran outside the lifecycle hold")
					}
					if !IsThreadBusy(err) {
						t.Fatalf("unexpected hold proof: %v", err)
					}
					if scenario == "changed" {
						name := "changed"
						_, err := env.Couch.Threads.ApplyThreadMetadata(address, source.Revision, ThreadMetadataPatch{Name: &name})
						return err
					}
					if scenario == "refused" {
						return context.Canceled
					}
					return nil
				},
			}
			if scenario == "held" {
				_, release, err := env.Couch.hold(context.Background(), source.Address, "other")
				if err != nil {
					t.Fatal(err)
				}
				defer release()
			}
			if scenario == "missing-guard" {
				env.Couch.SwitchGuard = nil
			}
			if scenario == "dispatch" {
				_, err = DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{Name: "switch-agent", Implicit: true, Context: context.Background(), Args: map[string]string{"repo-scope": source.Address.RepoScope, "tag": string(source.Address.Tag), "agent": "codex", "argv": "[]", "fingerprint": prepared.Fingerprint, "interrupt-first": "true"}})
			} else {
				_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: argv, AcceptedFingerprint: prepared.Fingerprint, InterruptFirst: true})
			}
			if err == nil {
				t.Fatal("cooperative refusal/stale source started a target")
			}
			wantPhases, wantChecks := 1, 0
			if scenario == "held" || scenario == "missing-guard" {
				wantPhases = 0
			}
			if scenario == "fresh-idle-guard" || scenario == "dispatch" {
				wantChecks = 1
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("lost fresh idle guard: %v", err)
				}
			}
			if phases != wantPhases || checks != wantChecks {
				t.Fatalf("phases=%d checks=%d want=%d/%d: %v", phases, checks, wantPhases, wantChecks, err)
			}
			after, err := env.Couch.Threads.GetThread(source.Address)
			if err != nil {
				t.Fatal(err)
			}
			if after.Park != nil || after.LatestLaunchProfile.Agent != "claude" || env.Proc.Exists(42) != Live {
				t.Fatal("refused cooperative operation changed its source")
			}
		})
	}
}

// reconnectSwitchGuard models a wrapper reconnect after the first live proof,
// without changing the durable thread. A supplied token must keep naming the
// original wrapper through the final guard.
type reconnectSwitchGuard struct {
	current string
	checks  int
}

func (g *reconnectSwitchGuard) ConfirmAgentSwitch(_ context.Context, _ ThreadAddress, source string, _ SwitchSourceEvidence) (string, error) {
	pinned := g.current
	g.checks++
	if source != "" && source != g.current {
		return "", errors.New("wrapper reconnected")
	}
	if g.checks == 1 {
		g.current = "replacement-wrapper"
	}
	return pinned, nil
}

func (g *reconnectSwitchGuard) InterruptBeforeAgentSwitch(_ context.Context, _ ThreadAddress, source string, _ SwitchSourceEvidence) (string, error) {
	pinned := g.current
	g.current = "replacement-wrapper"
	return pinned, nil
}

func TestSwitchAgentPinsLiveSourceAcrossFinalGuard(t *testing.T) {
	for _, cooperative := range []bool{false, true} {
		for _, remote := range []bool{false, true} {
			t.Run(fmt.Sprintf("cooperative=%t/remote=%t", cooperative, remote), func(t *testing.T) {
				env, source := switchEnvWithLiveThread(t)
				argv := []string{}
				prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &argv)
				if err != nil {
					t.Fatal(err)
				}
				guard := &reconnectSwitchGuard{current: "original-wrapper"}
				env.Couch.SwitchGuard = guard
				token := ""
				if remote {
					token = "original-wrapper"
				}
				result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: argv, AcceptedFingerprint: prepared.Fingerprint, Source: token, InterruptFirst: cooperative})
				if err == nil || !strings.Contains(err.Error(), "wrapper reconnected") || result.Outcome != SwitchRefused {
					t.Fatalf("reconnect reached park/start: %+v, %v", result, err)
				}
				after, err := env.Couch.Threads.GetThread(source.Address)
				if err != nil {
					t.Fatal(err)
				}
				if after.Revision != source.Revision || after.Park != nil || len(env.Runner.Ops) != 0 || env.Proc.Exists(42) != Live {
					t.Fatal("reconnected source caused effects")
				}
			})
		}
	}
}
